package main

import (
	"bytes"
	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	// "github.com/wminshew/emrysserver/pkg/storage"
	"io"
	"net/http"
	"os"
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

		// TODO: handle errors / set job inactive?

		if r.ContentLength == 0 {
			if err := jobsManager.Publish(jID, struct{}{}); err != nil {
				log.Sugar.Errorw("failed to publish bytes",
					"url", r.URL,
					"err", err.Error(),
					"jID", jID,
				)
				return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
			}
			// TODO: upload final log file to gcs?
			return db.SetStatusOutputLogPosted(r, jUUID)
		}

		var buf bytes.Buffer
		if _, err := io.Copy(&buf, r.Body); err != nil {
			log.Sugar.Errorw("failed to copy request body to buffer",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}
		b := buf.Bytes()

		outputDir := path.Join("output", jID)
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			log.Sugar.Errorw("failed to make output dir",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}
		p := path.Join(outputDir, "log")
		f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Sugar.Errorw("failed to create or open append only file",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}
		defer app.CheckErr(r, f.Close)
		if _, err := io.Copy(f, bytes.NewReader(b)); err != nil {
			log.Sugar.Errorw("failed to copy buffer to file",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}

		if err := jobsManager.Publish(jID, b); err != nil {
			log.Sugar.Errorw("failed to publish bytes",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}

		// ctx := r.Context()
		// p := path.Join("job", jID, "output", "log")
		// ow := storage.NewWriter(ctx, p)
		// if _, err = io.Copy(ow, tee); err != nil {
		// 	log.Sugar.Errorw("failed to copy tee reader to cloud storage object writer",
		// 		"url", r.URL,
		// 		"err", err.Error(),
		// 		"jID", jID,
		// 	)
		// 	if err := db.SetJobInactive(r, jUUID); err != nil {
		// 		log.Sugar.Errorf("Error setting job %v inactive: %v\n", jUUID, err)
		// 	}
		// 	return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		// }
		//
		// if err = ow.Close(); err != nil {
		// 	log.Sugar.Errorw("failed to close cloud storage object writer",
		// 		"url", r.URL,
		// 		"err", err.Error(),
		// 		"jID", jID,
		// 	)
		// 	if err := db.SetJobInactive(r, jUUID); err != nil {
		// 		log.Sugar.Errorf("Error setting job %v inactive: %v\n", jUUID, err)
		// 	}
		// 	return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		// }

		return nil
	}
}
