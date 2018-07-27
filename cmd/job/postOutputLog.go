package main

import (
	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"github.com/wminshew/emrysserver/pkg/storage"
	"io"
	"io/ioutil"
	"net/http"
	"path"
)

// postOutputLog receives the miner's container execution for the user
func postOutputLog() app.Handler {
	return func(w http.ResponseWriter, r *http.Request) *app.Error {
		vars := mux.Vars(r)
		jID := vars["jID"]
		jUUID, err := uuid.FromString(jID)
		if err != nil {
			log.Sugar.Errorw("failed to parse job ID",
				"url", r.URL,
				"err", err.Error(),
			)
			return &app.Error{Code: http.StatusBadRequest, Message: "error parsing job ID"}
		}

		var tee io.Reader
		if pipe, ok := logPipes[jUUID]; ok {
			tee = io.TeeReader(r.Body, pipe.w)
			defer app.CheckErr(r, pipe.w.Close)
		} else {
			tee = io.TeeReader(r.Body, ioutil.Discard)
		}

		ctx := r.Context()
		p := path.Join("job", jID, "output", "log")
		ow := storage.NewWriter(ctx, p)
		if _, err = io.Copy(ow, tee); err != nil {
			log.Sugar.Errorw("failed to copy tee reader to cloud storage object writer",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			if err := db.SetJobInactive(r, jUUID); err != nil {
				log.Sugar.Errorf("Error setting job %v inactive: %v\n", jUUID, err)
			}
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}

		if err = ow.Close(); err != nil {
			log.Sugar.Errorw("failed to close cloud storage object writer",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			if err := db.SetJobInactive(r, jUUID); err != nil {
				log.Sugar.Errorf("Error setting job %v inactive: %v\n", jUUID, err)
			}
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}

		return db.SetStatusOutputLogPosted(r, jUUID)
	}
}
