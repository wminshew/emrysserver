package main

import (
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"time"
)

var activeWorker = make(map[uuid.UUID]chan struct{})

const (
	timeout = 30 * time.Second
)

func monitorJob(jUUID uuid.UUID) {
	activeWorker[jUUID] = make(chan struct{})
	defer delete(activeWorker, jUUID)
	for {
		select {
		case <-time.After(timeout):
			if err := db.SetJobFailed(jUUID); err != nil {
				log.Sugar.Errorw("error setting job failed",
					"err", err.Error(),
					"jID", jUUID,
				)
				return
				// return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
			}

			return
		case <-activeWorker[jUUID]:
		}
	}
}
