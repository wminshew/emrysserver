package main

import (
	"github.com/jcuga/golongpoll"
	"github.com/wminshew/emrysserver/pkg/log"
)

var jobsManager *golongpoll.LongpollManager

// initJobsManager initializes the longpoll manager that handles miner
// connections while waiting for job auctions
func initJobsManager() {
	log.Sugar.Infof("Initializing longpoll manager...")

	var err error
	if jobsManager, err = golongpoll.StartLongpoll(golongpoll.Options{
		LoggingEnabled:            true,
		MaxLongpollTimeoutSeconds: 60 * 10,
		MaxEventBufferSize:        100,
		EventTimeToLiveSeconds:    10, // auctions only last 5 seconds, anyway
	}); err != nil {
		log.Sugar.Errorf("Longpoll manager failed to initialize! Panic!")
		panic(err)
	}
}
