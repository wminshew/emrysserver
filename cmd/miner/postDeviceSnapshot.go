package main

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrys/pkg/job"
	"github.com/wminshew/emrysserver/pkg/app"
	// "github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// postDeviceSnapshot receives a worker's GPU snapshot and resets the worker's timeout
func postDeviceSnapshot() app.Handler {
	return func(w http.ResponseWriter, r *http.Request) *app.Error {
		vars := mux.Vars(r)
		jID := vars["jID"]
		// jUUID, err := uuid.FromString(jID)
		_, err := uuid.FromString(jID)
		if err != nil {
			log.Sugar.Errorw("error parsing job ID",
				"url", r.URL,
				"err", err.Error(),
			)
			return &app.Error{Code: http.StatusBadRequest, Message: "error parsing job ID"}
		}

		mID := r.Header.Get("X-Jwt-Claims-Subject")
		// mUUID, err := uuid.FromString(mID)
		_, err = uuid.FromString(mID)
		if err != nil {
			log.Sugar.Errorw("error parsing miner ID",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return &app.Error{Code: http.StatusBadRequest, Message: "error parsing miner ID"}
		}

		d := &job.DeviceSnapshot{}
		if err = json.NewDecoder(r.Body).Decode(d); err != nil {
			log.Sugar.Errorw("error decoding gpu snapshot",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return &app.Error{Code: http.StatusBadRequest, Message: "error parsing json gpu snapshot request body"}
		}

		// if it doesn't exist, worker isn't active TODO: is this right?
		// TODO: include snapshot in bids (or at least some info), and create activeworker upon winning
		// if ch, ok := activeWorker[mUUID][d.ID]; ok {
		// 	ch <- struct{}{}
		// }

		// TODO: save device snapshot to database

		return nil
	}
}
