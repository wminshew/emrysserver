package main

import (
	"github.com/jcuga/golongpoll"
	"github.com/wminshew/emrysserver/pkg/log"
)

var (
	jobsManager *golongpoll.LongpollManager
	maxTimeout  = 60 * 2
)

// initJobsManager initializes the longpoll manager that handles user and miner
// connections while distributing output logs
func initJobsManager() {
	log.Sugar.Infof("Initializing longpoll manager...")

	var err error
	if jobsManager, err = golongpoll.StartLongpoll(golongpoll.Options{
		LoggingEnabled:                 true,
		MaxLongpollTimeoutSeconds:      maxTimeout,
		MaxEventBufferSize:             100,
		EventTimeToLiveSeconds:         60 * 2,
		DeleteEventAfterFirstRetrieval: true,
	}); err != nil {
		log.Sugar.Errorf("Longpoll manager failed to initialize! Panic!")
		panic(err)
	}
}
