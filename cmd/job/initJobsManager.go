package main

import (
	"github.com/jcuga/golongpoll"
	"github.com/wminshew/emrysserver/pkg/log"
	"os"
)

var (
	jobsManager   *golongpoll.LongpollManager
	maxTimeout    = 60 * 2
	debugLongpoll = (os.Getenv("DEBUG_LONGPOLL") == "true")
)

// initJobsManager initializes the longpoll manager that handles user and miner
// connections while distributing output logs
func initJobsManager() {
	log.Sugar.Infof("Initializing longpoll manager...")

	var err error
	if jobsManager, err = golongpoll.StartLongpoll(golongpoll.Options{
		LoggingEnabled:                 debugLongpoll,
		MaxLongpollTimeoutSeconds:      maxTimeout,
		MaxEventBufferSize:             100,
		EventTimeToLiveSeconds:         60 * 2,
		DeleteEventAfterFirstRetrieval: true,
	}); err != nil {
		log.Sugar.Errorf("error initializing longpoll manager: %v", err)
		panic(err)
	}
}
