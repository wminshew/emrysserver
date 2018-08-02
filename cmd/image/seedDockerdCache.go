package main

import (
	"context"
	"docker.io/go-docker/api/types"
	"fmt"
	"github.com/mholt/archiver"
	"github.com/wminshew/emrys/pkg/job"
	"github.com/wminshew/emrysserver/pkg/log"
	"io"
	"strings"
)

var (
	localBaseJobRef   string
	dockerBaseCudaRef string
)

// seedDockerdCache downloads and possibly builds early-stage docker images
func seedDockerdCache(ctx context.Context) {
	log.Sugar.Infof("Seeding dockerd cache...")

	var imgPullResp io.ReadCloser
	var err error
	// TODO: make these ENV/ARGS?
	img := "nvidia/cuda"
	tag := "9.0-base-ubuntu16.04"
	digestAlgo := "sha256"
	digest := "ba1c9865dcafe8af90e60869f94acda6ca6b74981d59b63cff842be284ab2aed"
	local := true
	localBaseCudaRef := fmt.Sprintf("%s/%s:%s@%s:%s", registryHost, img, tag, digestAlgo, digest)
	if imgPullResp, err = dClient.ImagePull(ctx, localBaseCudaRef, types.ImagePullOptions{}); err != nil {
		log.Sugar.Infof("failed to find %s: %v", localBaseCudaRef, err)
		local = false
	} else {
		if err = job.ReadJSON(imgPullResp); err != nil {
			log.Sugar.Errorf("failed to pull %s: %v", localBaseCudaRef, err)
			if err := imgPullResp.Close(); err != nil {
				log.Sugar.Errorf("failed to close dockerd cache pull response: %v\n", err)
			}
		}
	}

	dockerBaseCudaRef = fmt.Sprintf("%s:%s@%s:%s", img, tag, digestAlgo, digest)
	if imgPullResp, err = dClient.ImagePull(ctx, dockerBaseCudaRef, types.ImagePullOptions{}); err != nil {
		log.Sugar.Infof("failed to find %s: %v", dockerBaseCudaRef, err)
	} else {
		if err = job.ReadJSON(imgPullResp); err != nil {
			log.Sugar.Errorf("failed to pull %s: %v", dockerBaseCudaRef, err)
			if err := imgPullResp.Close(); err != nil {
				log.Sugar.Errorf("failed to close dockerd cache pull response: %v\n", err)
			}
		} else {
			localBaseRefNoDigest := strings.Split(localBaseCudaRef, "@")[0]
			if err = dClient.ImageTag(ctx, dockerBaseCudaRef, localBaseRefNoDigest); err != nil {
				log.Sugar.Errorf("failed to tag %s as %s: %v", dockerBaseCudaRef, localBaseCudaRef, err)
			} else {
				if imgPushResp, err := dClient.ImagePush(ctx, localBaseCudaRef, types.ImagePushOptions{
					RegistryAuth: "none",
				}); err != nil {
					log.Sugar.Errorf("failed to push %s: %v", localBaseCudaRef, err)
				} else {
					if err = job.ReadJSON(imgPushResp); err != nil {
						log.Sugar.Errorf("failed to push %s: %v", localBaseCudaRef, err)
						if err := imgPushResp.Close(); err != nil {
							log.Sugar.Errorf("failed to close dockerd cache push response: %v\n", err)
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
	if imgPullResp, err = dClient.ImagePull(ctx, localBaseJobRef, types.ImagePullOptions{}); err != nil {
		log.Sugar.Infof("failed to find %s: %v", localBaseJobRef, err)
		local = false
	} else {
		if err = job.ReadJSON(imgPullResp); err != nil {
			log.Sugar.Errorf("failed to pull %s: %v", localBaseJobRef, err)
			if err := imgPullResp.Close(); err != nil {
				log.Sugar.Errorf("failed to close dockerd cache pull response: %v\n", err)
			}
		}
	}

	if !local {
		// build from dockerfile, then push it to local registry
		ctxFiles := []string{dockerfilePath}
		pr, pw := io.Pipe()
		go func() {
			if err := archiver.TarGz.Write(pw, ctxFiles); err != nil {
				log.Sugar.Errorw("failed to tar-gzip docker context: %v", err)
			}
			if err := pw.Close(); err != nil {
				log.Sugar.Errorw("failed to close tar-gzip pw: %v", err)
			}
		}()

		if imgBuildResp, err := dClient.ImageBuild(ctx, pr, types.ImageBuildOptions{
			CacheFrom: []string{dockerBaseCudaRef},
			Tags:      []string{localBaseJobRef},
			Target:    "base",
		}); err != nil {
			log.Sugar.Infof("failed to build %s: %v", localBaseJobRef, err)
		} else {
			if err = job.ReadJSON(imgBuildResp.Body); err != nil {
				log.Sugar.Errorf("failed to build %s: %v", localBaseJobRef, err)
				if err := imgBuildResp.Body.Close(); err != nil {
					log.Sugar.Errorf("failed to close dockerd cache build response: %v\n", err)
				}
			} else {
				if imgPushResp, err := dClient.ImagePush(ctx, localBaseJobRef, types.ImagePushOptions{
					RegistryAuth: "none",
				}); err != nil {
					log.Sugar.Errorf("failed to push %s: %v", localBaseJobRef, err)
				} else {
					if err = job.ReadJSON(imgPushResp); err != nil {
						log.Sugar.Errorf("failed to push %s: %v", localBaseJobRef, err)
						if err := imgPushResp.Close(); err != nil {
							log.Sugar.Errorf("failed to close dockerd cache push response: %v\n", err)
						}
					}
				}
			}
		}
	}
}
