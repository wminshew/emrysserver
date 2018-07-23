package main

import (
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
}
