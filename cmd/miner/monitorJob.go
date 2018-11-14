package main

import (
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"time"
)

var (
	activeWorker = make(map[uuid.UUID]chan struct{})
)

func monitorJob(jUUID uuid.UUID) {
	activeWorker[jUUID] = make(chan struct{})
	defer delete(activeWorker, jUUID)
	for {
		select {
		case <-time.After(time.Second * time.Duration(minerTimeout)):
			log.Sugar.Infow("miner failed job",
				"jID", jUUID,
			)
			if err := db.SetJobFailed(jUUID); err != nil {
				log.Sugar.Errorw("error setting job failed",
					"err", err.Error(),
					"jID", jUUID,
				)
				return
			}
			return
		case <-activeWorker[jUUID]:
		}
	}
}
