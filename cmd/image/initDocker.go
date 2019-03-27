package main

import (
	"context"
	"docker.io/go-docker"
	"docker.io/go-docker/api/types"
	"github.com/wminshew/emrysserver/pkg/log"
	"time"
)

var (
	dClient        *docker.Client
	imageBuildTime = make(map[string]time.Time)
)

const (
	prunePeriod = 3 * 24 * time.Hour
)

// initDocker initializes the docker client
func initDocker(ctx context.Context) {
	log.Sugar.Infof("Initializing docker client...")

	var err error
	if dClient, err = docker.NewEnvClient(); err != nil {
		log.Sugar.Errorf("error initializing docker client: %v", err)
		panic(err)
	}

	// TODO: move to initContainer
	go seedDockerdCache(ctx)

	go pruneDocker(ctx, dClient)
}

func pruneDocker(ctx context.Context, dClient *docker.Client) {
	for {
		select {
		case <-ctx.Done():
			return
			// TODO: add trigger if disk gets close to capacity, evict by LRU
		case <-time.After(prunePeriod):
			for imgRefStr, t := range imageBuildTime {
				if time.Since(t) > prunePeriod {
					if _, err := dClient.ImageRemove(ctx, imgRefStr, types.ImageRemoveOptions{
						Force: true,
					}); err != nil {
						log.Sugar.Errorf("Docker prune: error removing job image %v: %v", imgRefStr, err)
						continue
					}
					log.Sugar.Infof("Removed image %v", imgRefStr)
					delete(imageBuildTime, imgRefStr)
				}
			}
			log.Sugar.Infof("Pruning build cache")
			if _, err := dClient.BuildCachePrune(ctx); err != nil {
				log.Sugar.Errorf("Docker prune: error pruning build cache: %v", err)
			}
		}
	}
}
