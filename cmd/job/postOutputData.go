package main

import (
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
					return fmt.Errorf("opening output data.tar.gz: %v", err)
				}
				defer app.CheckErr(r, f.Close)
				ctx := context.Background()
				ow := storage.NewWriter(ctx, p)
				if _, err = io.Copy(ow, f); err != nil {
					return fmt.Errorf("copying tee reader to cloud storage object writer: %v", err)
				}
				if err = ow.Close(); err != nil {
					return fmt.Errorf("closing cloud storage object writer: %v", err)
				}
				return nil
			}
			if err := backoff.RetryNotify(operation,
				backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 10),
				func(err error, t time.Duration) {
					log.Sugar.Errorw("failed to upload output data.tar.gz to gcs--retrying",
						"url", r.URL,
						"err", err.Error(),
						"jID", jID,
					)
				}); err != nil {
				log.Sugar.Errorw("failed to upload output data.tar.gz to gcs--abort",
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
