package main

import (
	"context"
	"docker.io/go-docker/api/types"
	"fmt"
	"github.com/mholt/archiver"
	"github.com/wminshew/emrys/pkg/check"
	"github.com/wminshew/emrys/pkg/jsonmessage"
	"github.com/wminshew/emrysserver/pkg/log"
	"io"
	"os"
	"path/filepath"
	"time"
)

var (
	localBaseJobRef      string
	remoteBaseCudaRef    string
	dockerfilePath       = os.Getenv("DOCKER_PATH")
	dockerEntrypointPath = os.Getenv("DOCKER_ENTRYPOINT_PATH")
)

// TODO: move to initContainers
// TODO: this function should probably be able to err out properly
// seedDockerdCache downloads and possibly builds early-stage docker images
func seedDockerdCache(ctx context.Context) {
	// TODO: make ENV/ARGS
	time.Sleep(15 * time.Second) // wait for dockerd to boot
	log.Sugar.Infof("Seeding dockerd cache...")

	var pullResp io.ReadCloser
	var err error
	// TODO: make ENV/ARGS
	registry := registryHost
	repo := "nvidia"
	img := "cuda"
	tag := "10.1-base-ubuntu18.04"
	localBaseCudaRef := fmt.Sprintf("%s/%s/%s:%s", registry, repo, img, tag)
	log.Sugar.Infof("Pulling %s...", localBaseCudaRef)
	if pullResp, err = dClient.ImagePull(ctx, localBaseCudaRef, types.ImagePullOptions{}); err != nil {
		log.Sugar.Infof("error finding %s: %v", localBaseCudaRef, err)
	} else if err = jsonmessage.DisplayJSONMessagesStream(pullResp, os.Stdout, os.Stdout.Fd(), nil); err != nil {
		log.Sugar.Errorf("error pulling %s: %v", localBaseCudaRef, err)
		if err = pullResp.Close(); err != nil {
			log.Sugar.Errorf("error closing dockerd cache pull response: %v\n", err)
		}
	}

	remoteBaseCudaRef = fmt.Sprintf("%s/%s:%s", repo, img, tag)
	log.Sugar.Infof("Pulling %s...", remoteBaseCudaRef)
	if pullResp, err = dClient.ImagePull(ctx, remoteBaseCudaRef, types.ImagePullOptions{}); err != nil {
		log.Sugar.Infof("error finding %s: %v", remoteBaseCudaRef, err)
	} else if err = jsonmessage.DisplayJSONMessagesStream(pullResp, os.Stdout, os.Stdout.Fd(), nil); err != nil {
		log.Sugar.Errorf("error pulling %s: %v", remoteBaseCudaRef, err)
		if err := pullResp.Close(); err != nil {
			log.Sugar.Errorf("error closing dockerd cache pull response: %v\n", err)
		}
	} else {
		log.Sugar.Infof("Pushing %s to %s...", remoteBaseCudaRef, localBaseCudaRef)
		if err = dClient.ImageTag(ctx, remoteBaseCudaRef, localBaseCudaRef); err != nil {
			log.Sugar.Errorf("error tagging %s as %s: %v", remoteBaseCudaRef, localBaseCudaRef, err)
		} else if pushResp, err := dClient.ImagePush(ctx, localBaseCudaRef, types.ImagePushOptions{
			RegistryAuth: "none",
		}); err != nil {
			log.Sugar.Errorf("error pushing %s: %v", localBaseCudaRef, err)
		} else if err = jsonmessage.DisplayJSONMessagesStream(pushResp, os.Stdout, os.Stdout.Fd(), nil); err != nil {
			log.Sugar.Errorf("error pushing %s: %v", localBaseCudaRef, err)
			if err := pushResp.Close(); err != nil {
				log.Sugar.Errorf("error closing dockerd cache push response: %v\n", err)
			}
		}
	}

	repo = "emrys"
	img = "base"
	tag = "18.04-10.1"
	localBaseJobRef = fmt.Sprintf("%s/%s/%s:%s", registry, repo, img, tag)
	log.Sugar.Infof("Pulling %s...", localBaseJobRef)
	if pullResp, err = dClient.ImagePull(ctx, localBaseJobRef, types.ImagePullOptions{}); err != nil {
		log.Sugar.Infof("error finding %s: %v", localBaseJobRef, err)
	} else if err = jsonmessage.DisplayJSONMessagesStream(pullResp, os.Stdout, os.Stdout.Fd(), nil); err != nil {
		log.Sugar.Errorf("error pulling %s: %v", localBaseJobRef, err)
		if err := pullResp.Close(); err != nil {
			log.Sugar.Errorf("error closing dockerd cache pull response: %v\n", err)
		}
	}

	// copy base-dockerfile [can't build off soft link, which is what configmap evidently is]
	// TODO: come up with better naming / organizational structure here...
	seedBuildCtx := filepath.Join("docker-temp", "seed-build-ctx")
	if err := os.MkdirAll(seedBuildCtx, 0755); err != nil {
		log.Sugar.Errorf("error creating dir %s: %s", seedBuildCtx, err)
		return
	}
	defer check.Err(func() error { return os.RemoveAll(seedBuildCtx) })

	inputDockerfilePath := filepath.Join(seedBuildCtx, "Dockerfile")
	if err := func() error {
		dockerfile, err := os.Open(dockerfilePath)
		if err != nil {
			return err
		}
		defer check.Err(dockerfile.Close)

		inputDockerfile, err := os.Create(inputDockerfilePath)
		if err != nil {
			return err
		}
		defer check.Err(inputDockerfile.Close)

		_, err = io.Copy(inputDockerfile, dockerfile)
		return err
	}(); err != nil {
		log.Sugar.Errorf("error copying dockerfile into %s: %s", seedBuildCtx, err)
		return
	}

	inputDockerEntrypointPath := filepath.Join(seedBuildCtx, "entrypoint.sh")
	if err := func() error {
		dockerEntrypoint, err := os.Open(dockerEntrypointPath)
		if err != nil {
			return err
		}
		defer check.Err(dockerEntrypoint.Close)

		inputDockerEntrypoint, err := os.Create(inputDockerEntrypointPath)
		if err != nil {
			return err
		}
		defer check.Err(inputDockerEntrypoint.Close)

		_, err = io.Copy(inputDockerEntrypoint, dockerEntrypoint)
		return err
	}(); err != nil {
		log.Sugar.Errorf("error copying docker entrypoint into %s: %s", seedBuildCtx, err)
		return
	}

	// build from dockerfile, then push it to local registry
	ctxFiles := []string{inputDockerfilePath, inputDockerEntrypointPath}
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
		CacheFrom: []string{remoteBaseCudaRef, localBaseJobRef},
		Tags:      []string{localBaseJobRef},
		Target:    "base",
	}); err != nil {
		log.Sugar.Infof("error building %s: %v", localBaseJobRef, err)
	} else if err = jsonmessage.DisplayJSONMessagesStream(buildResp.Body, os.Stdout, os.Stdout.Fd(), nil); err != nil {
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
		} else if err = jsonmessage.DisplayJSONMessagesStream(pushResp, os.Stdout, os.Stdout.Fd(), nil); err != nil {
			log.Sugar.Errorf("error pushing %s: %v", localBaseJobRef, err)
			if err := pushResp.Close(); err != nil {
				log.Sugar.Errorf("error closing dockerd cache push response: %v\n", err)
			}
		}
	}
}
