package main

import (
	"context"
	"docker.io/go-docker"
	"github.com/wminshew/emrysserver/pkg/log"
)

var (
	dClient *docker.Client
)

// initDocker initializes the docker client
func initDocker() {
	log.Sugar.Infof("Initializing docker client...")

	var err error
	if dClient, err = docker.NewEnvClient(); err != nil {
		log.Sugar.Errorf("error initializing docker client: %v", err)
		panic(err)
	}

	ctx := context.Background()
	if err = downloadDockerfile(ctx); err != nil {
		log.Sugar.Errorf("error downloading dockerfile: %v", err)
		panic(err)
	}

	seedDockerdCache(ctx)
}
