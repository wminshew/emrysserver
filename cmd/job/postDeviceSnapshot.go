package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/cenkalti/backoff"
	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"github.com/wminshew/emrysserver/pkg/storage"
	"io"
	"net/http"
	"os"
	"path"
	"time"
)

// postDeviceSnapshot receives a worker's GPU snapshot and resets the worker's timeout
func postDeviceSnapshot() app.Handler {
	return func(w http.ResponseWriter, r *http.Request) *app.Error {
		vars := mux.Vars(r)
		jID := vars["jID"]
		jUUID, err := uuid.FromString(jID)
		if err != nil {
			log.Sugar.Errorw("error parsing job ID",
				"url", r.URL,
				"err", err.Error(),
			)
			return &app.Error{Code: http.StatusBadRequest, Message: "error parsing job ID"}
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

		dUUID, err := uuid.FromString(d.ID)
		if err != nil {
			log.Sugar.Errorw("error parsing device ID",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return &app.Error{Code: http.StatusBadRequest, Message: "error parsing device ID"}
		}

		// if it doesn't exist, worker isn't active TODO: is this right?
		if ch, ok := activeWorker[mUUID][dUUID]; ok {
			ch <- struct{}{}
		}

		// TODO: save device snapshot to database

		return nil
	}
}
