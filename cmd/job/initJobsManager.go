package main

import (
	"github.com/jcuga/golongpoll"
	"github.com/wminshew/emrysserver/pkg/log"
)

var (
	jobsManager *golongpoll.LongpollManager
	maxTimeout  = 60 * 2
	// TODO: activeJobs map[uuid.UUID]bool
	// TODO: activeWorkers map[uuid.UUID][uuid.UUID]bool
	// TODO: activeWorkers map[uuid.UUID][uuid.UUID]chan struct{}
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
		log.Sugar.Errorf("error initializing longpoll manager: %v", err)
		panic(err)
	}
}

// TODO:
// var workerTimeout = 30 * time.Second
//
// init job? maybe add http endpoint and sync.Once?
// func activateWorker(mID, jID) {
// for {
// 	select {
//  case <- jobFinishedMap:
//		?minerSuccess() // not sure i need to do anything in case of success
//		return
// 	case <- workerActiveMap[mID][wID]:
// 	case <- time.After(workerTimeout):
//		minerFailed()
//		return
// 	}
// }
// }
