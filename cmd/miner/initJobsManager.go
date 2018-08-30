package main

import (
	"github.com/jcuga/golongpoll"
	"github.com/wminshew/emrysserver/pkg/log"
)

var (
	jobsManager *golongpoll.LongpollManager
	maxTimeout  = 60 * 10
)

// initJobsManager initializes the longpoll manager that handles miner
// connections while waiting for job auctions
func initJobsManager() {
	log.Sugar.Infof("Initializing longpoll manager...")

	var err error
	if jobsManager, err = golongpoll.StartLongpoll(golongpoll.Options{
		LoggingEnabled:            true,
		MaxLongpollTimeoutSeconds: maxTimeout,
		MaxEventBufferSize:        100,
		EventTimeToLiveSeconds:    10,
	}); err != nil {
		log.Sugar.Errorf("error initializing longpoll manager: %v", err)
		panic(err)
	}
}
