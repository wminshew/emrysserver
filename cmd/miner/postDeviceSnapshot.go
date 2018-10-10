package main

import (
	"encoding/json"
	// "github.com/gorilla/mux"
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
		mID := r.Header.Get("X-Jwt-Claims-Subject")
		// mUUID, err := uuid.FromString(mID)
		_, err := uuid.FromString(mID)
		if err != nil {
			log.Sugar.Errorw("error parsing miner ID",
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
			)
			return &app.Error{Code: http.StatusBadRequest, Message: "error parsing miner ID"}
		}

		d := &job.DeviceSnapshot{}
		if err = json.NewDecoder(r.Body).Decode(d); err != nil {
			log.Sugar.Errorw("error decoding gpu snapshot",
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"mID", mID,
			)
			return &app.Error{Code: http.StatusBadRequest, Message: "error parsing json gpu snapshot request body"}
		}

		// TODO: store snapshots in files or DB instead of logger?
		log.Sugar.Infow("device snapshot",
			"mID", mID,
			"dID", d.ID,
			"snapshot", d,
		)

		jID := r.URL.Query().Get("jID")
		if jID == "" {
			return nil
		}

		jUUID, err := uuid.FromString(jID)
		if err != nil {
			log.Sugar.Errorw("error parsing job ID",
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"mID", mID,
			)
			return &app.Error{Code: http.StatusBadRequest, Message: "error parsing job ID"}
		}

		if ch, ok := activeWorker[jUUID]; ok {
			ch <- struct{}{}
		}

		return nil
	}
}
