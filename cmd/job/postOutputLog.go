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

// postOutputLog receives the miner's container execution for the user
func postOutputLog() app.Handler {
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

		outputDir := path.Join("output", jID)
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			log.Sugar.Errorw("error making output dir",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}
		outputLog := path.Join(outputDir, "log")
		f, err := os.OpenFile(outputLog, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Sugar.Errorw("error creating or open append only file",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}
		defer app.CheckErr(r, f.Close)

		if r.ContentLength == 0 {
			if err := jobsManager.Publish(jID, struct{}{}); err != nil {
				log.Sugar.Errorw("error publishing bytes",
					"url", r.URL,
					"err", err.Error(),
					"jID", jID,
				)
				return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
			}

			go func() {
				operation := func() error {
					ctx := context.Background()
					uploadLog := path.Join("output", jID, "log")
					ow := storage.NewWriter(ctx, uploadLog)
					if _, err = io.Copy(ow, f); err != nil {
						return fmt.Errorf("copying log file to cloud storage object writer: %v", err)
					}
					if err = ow.Close(); err != nil {
						return fmt.Errorf("closing cloud storage object writer: %v", err)
					}
					return nil
				}
				if err := backoff.RetryNotify(operation,
					backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 10),
					func(err error, t time.Duration) {
						log.Sugar.Errorw("error uploading output log to gcs--retrying",
							"url", r.URL,
							"err", err.Error(),
							"jID", jID,
						)
					}); err != nil {
					log.Sugar.Errorw("error uploading output log to gcs--abort",
						"url", r.URL,
						"err", err.Error(),
						"jID", jID,
					)
					return
				}
				go func() {
					defer app.CheckErr(r, func() error { return os.Remove(outputLog) }) // no need to cache locally
					time.Sleep(15 * time.Minute)
				}()
			}()

			return db.SetStatusOutputLogPosted(r, jUUID)
		}

		var buf bytes.Buffer
		if _, err := io.Copy(&buf, r.Body); err != nil {
			log.Sugar.Errorw("error copying request body to buffer",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}
		b := buf.Bytes()
		if _, err := io.Copy(f, bytes.NewReader(b)); err != nil {
			log.Sugar.Errorw("error copying buffer to file",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}
		if err := jobsManager.Publish(jID, b); err != nil {
			log.Sugar.Errorw("error publishing bytes",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}

		return nil
	}
}
