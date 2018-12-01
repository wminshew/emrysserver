package main

import (
	"github.com/jcuga/golongpoll"
	"github.com/wminshew/emrysserver/pkg/log"
	"os"
)

var (
	minerManager  *golongpoll.LongpollManager
	maxTimeout    = 60 * 10
	debugLongpoll = (os.Getenv("DEBUG_LONGPOLL") == "true")
)

// initMinerManager initializes the longpoll manager that handles miner
// connections while waiting for job auctions
func initMinerManager() {
	log.Sugar.Infof("Initializing longpoll manager...")

	var err error
	if minerManager, err = golongpoll.StartLongpoll(golongpoll.Options{
		LoggingEnabled:            debugLongpoll,
		MaxLongpollTimeoutSeconds: maxTimeout,
		MaxEventBufferSize:        100,
		EventTimeToLiveSeconds:    10,
	}); err != nil {
		log.Sugar.Errorf("error initializing longpoll manager: %v", err)
		panic(err)
	}
}
