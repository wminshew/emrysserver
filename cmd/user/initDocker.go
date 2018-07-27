package main

import (
	"context"
	"docker.io/go-docker"
	"github.com/wminshew/emrysserver/pkg/log"
)

var dClient *docker.Client

// initDocker initializes the docker client
func initDocker() {
	log.Sugar.Infof("Initializing docker client...")

	var err error
	if dClient, err = docker.NewEnvClient(); err != nil {
		log.Sugar.Errorf("Docker client failed to initialize! Panic!")
		panic(err)
	}
	ctx := context.Background()
	if info, err := dClient.Info(ctx); err != nil {
		log.Sugar.Errorf("Unable to ping docker client!")
		panic(err)
	} else {
		log.Sugar.Infof("Docker info: %s\n", info)
	}
}
