package main

import (
	"context"
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
)

// postOutputData receives the miner's container execution for the user
func postOutputData() app.Handler {
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

		outputDir := path.Join("output", jID)
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			log.Sugar.Errorw("failed to make output dir",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}
		p := path.Join(outputDir, "data.tar.gz")
		f, err := os.Create(p)
		if err != nil {
			log.Sugar.Errorw("failed to create output data.tar.gz",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}
		if _, err = io.Copy(f, r.Body); err != nil {
			log.Sugar.Errorw("failed to copy data.tar.gz to file",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			app.CheckErr(r, f.Close)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}
		app.CheckErr(r, f.Close)

		go func() {
			operation := func() error {
				f, err := os.Open(p)
				if err != nil {
					log.Sugar.Errorw("failed to open output data.tar.gz",
						"url", r.URL,
						"err", err.Error(),
						"jID", jID,
					)
					return err
				}
				defer app.CheckErr(r, f.Close)
				ctx := context.Background()
				ow := storage.NewWriter(ctx, p)
				if _, err = io.Copy(ow, f); err != nil {
					log.Sugar.Errorw("failed to copy tee reader to cloud storage object writer",
						"url", r.URL,
						"err", err.Error(),
						"jID", jID,
					)
					return err
				}
				if err = ow.Close(); err != nil {
					log.Sugar.Errorw("failed to close cloud storage object writer",
						"url", r.URL,
						"err", err.Error(),
						"jID", jID,
					)
					return err
				}
				return nil
			}
			if err := backoff.Retry(operation, backoff.NewExponentialBackOff()); err != nil {
				log.Sugar.Errorw("failed to upload output data.tar.gz to gcs",
					"url", r.URL,
					"err", err.Error(),
					"jID", jID,
				)
				return
			}
		}()

		return db.SetJobFinishedAndStatusOutputDataPosted(r, jUUID)
	}
}
