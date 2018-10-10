package main

import (
	"context"
	"docker.io/go-docker/api/types"
	"fmt"
	"github.com/mholt/archiver"
	"github.com/wminshew/emrys/pkg/jsonmessage"
	"github.com/wminshew/emrysserver/pkg/log"
	"io"
	"os"
	"strings"
	"time"
)

var (
	localBaseJobRef   string
	dockerBaseCudaRef string
)

// seedDockerdCache downloads and possibly builds early-stage docker images
func seedDockerdCache(ctx context.Context) {
	time.Sleep(5 * time.Second) // wait for dockerd to boot
	log.Sugar.Infof("Seeding dockerd cache...")

	var pullResp io.ReadCloser
	var err error
	// TODO: make ENV/ARGS?
	img := "nvidia/cuda"
	tag := "9.0-base-ubuntu16.04"
	digestAlgo := "sha256"
	digest := "ba1c9865dcafe8af90e60869f94acda6ca6b74981d59b63cff842be284ab2aed"
	local := true
	localBaseCudaRef := fmt.Sprintf("%s/%s:%s@%s:%s", registryHost, img, tag, digestAlgo, digest)
	log.Sugar.Infof("Pulling %s...", localBaseCudaRef)
	if pullResp, err = dClient.ImagePull(ctx, localBaseCudaRef, types.ImagePullOptions{}); err != nil {
		log.Sugar.Infof("error finding %s: %v", localBaseCudaRef, err)
		local = false
	} else {
		if err = jsonmessage.DisplayJSONMessagesStream(pullResp, os.Stdout, os.Stdout.Fd(), nil); err != nil {
			log.Sugar.Errorf("error pulling %s: %v", localBaseCudaRef, err)
			if err := pullResp.Close(); err != nil {
				log.Sugar.Errorf("error closing dockerd cache pull response: %v\n", err)
			}
		}
	}

	dockerBaseCudaRef = fmt.Sprintf("%s:%s@%s:%s", img, tag, digestAlgo, digest)
	log.Sugar.Infof("Pulling %s...", dockerBaseCudaRef)
	if pullResp, err = dClient.ImagePull(ctx, dockerBaseCudaRef, types.ImagePullOptions{}); err != nil {
		log.Sugar.Infof("error finding %s: %v", dockerBaseCudaRef, err)
	} else {
		if err = jsonmessage.DisplayJSONMessagesStream(pullResp, os.Stdout, os.Stdout.Fd(), nil); err != nil {
			log.Sugar.Errorf("error pulling %s: %v", dockerBaseCudaRef, err)
			if err := pullResp.Close(); err != nil {
				log.Sugar.Errorf("error closing dockerd cache pull response: %v\n", err)
			}
		} else if !local {
			localBaseRefNoDigest := strings.Split(localBaseCudaRef, "@")[0]
			if err = dClient.ImageTag(ctx, dockerBaseCudaRef, localBaseRefNoDigest); err != nil {
				log.Sugar.Errorf("error tagging %s as %s: %v", dockerBaseCudaRef, localBaseRefNoDigest, err)
			} else {
				log.Sugar.Infof("Pushing %s...", localBaseRefNoDigest)
				if pushResp, err := dClient.ImagePush(ctx, localBaseRefNoDigest, types.ImagePushOptions{
					RegistryAuth: "none",
				}); err != nil {
					log.Sugar.Errorf("error pushing %s: %v", localBaseRefNoDigest, err)
				} else {
					if err = jsonmessage.DisplayJSONMessagesStream(pushResp, os.Stdout, os.Stdout.Fd(), nil); err != nil {
						log.Sugar.Errorf("error pushing %s: %v", localBaseRefNoDigest, err)
						if err := pushResp.Close(); err != nil {
							log.Sugar.Errorf("error closing dockerd cache push response: %v\n", err)
						}
					}
				}
			}
		}
	}

	img = "emrys/base"
	tag = "1604-90"
	local = true
	localBaseJobRef = fmt.Sprintf("%s/%s:%s", registryHost, img, tag)
	log.Sugar.Infof("Pulling %s...", localBaseJobRef)
	if pullResp, err = dClient.ImagePull(ctx, localBaseJobRef, types.ImagePullOptions{}); err != nil {
		log.Sugar.Infof("error finding %s: %v", localBaseJobRef, err)
		local = false
	} else {
		if err = jsonmessage.DisplayJSONMessagesStream(pullResp, os.Stdout, os.Stdout.Fd(), nil); err != nil {
			log.Sugar.Errorf("error pulling %s: %v", localBaseJobRef, err)
			if err := pullResp.Close(); err != nil {
				log.Sugar.Errorf("error closing dockerd cache pull response: %v\n", err)
			}
		}
	}

	if !local {
		// build from dockerfile, then push it to local registry
		ctxFiles := []string{dockerfilePath}
		pr, pw := io.Pipe()
		go func() {
			if err := archiver.TarGz.Write(pw, ctxFiles); err != nil {
				log.Sugar.Errorf("error tar-gzipping docker context: %v", err)
			}
			if err := pw.Close(); err != nil {
				log.Sugar.Errorf("error closing tar-gzip pw: %v", err)
			}
		}()

		log.Sugar.Infof("Building %s...", localBaseJobRef)
		if buildResp, err := dClient.ImageBuild(ctx, pr, types.ImageBuildOptions{
			CacheFrom: []string{dockerBaseCudaRef},
			Tags:      []string{localBaseJobRef},
			Target:    "base",
		}); err != nil {
			log.Sugar.Infof("error building %s: %v", localBaseJobRef, err)
		} else {
			if err = jsonmessage.DisplayJSONMessagesStream(buildResp.Body, os.Stdout, os.Stdout.Fd(), nil); err != nil {
				log.Sugar.Errorf("error building %s: %v", localBaseJobRef, err)
				if err := buildResp.Body.Close(); err != nil {
					log.Sugar.Errorf("error closing dockerd cache build response: %v\n", err)
				}
			} else {
				log.Sugar.Infof("Pushing %s...", localBaseJobRef)
				if pushResp, err := dClient.ImagePush(ctx, localBaseJobRef, types.ImagePushOptions{
					RegistryAuth: "none",
				}); err != nil {
					log.Sugar.Errorf("error pushing %s: %v", localBaseJobRef, err)
				} else {
					if err = jsonmessage.DisplayJSONMessagesStream(pushResp, os.Stdout, os.Stdout.Fd(), nil); err != nil {
						log.Sugar.Errorf("error pushing %s: %v", localBaseJobRef, err)
						if err := pushResp.Close(); err != nil {
							log.Sugar.Errorf("error closing dockerd cache push response: %v\n", err)
						}
					}
				}
			}
		}
	}
}
