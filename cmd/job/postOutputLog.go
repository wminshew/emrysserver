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

const (
	maxBackoffElapsedTime = 72 * time.Hour
)

// postOutputLog receives the miner's container execution for the user
var postOutputLog app.Handler = func(w http.ResponseWriter, r *http.Request) *app.Error {
	vars := mux.Vars(r)
	jID := vars["jID"]
	jUUID, err := uuid.FromString(jID)
	if err != nil {
		log.Sugar.Errorw("error parsing job ID",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "error parsing job ID"}
	}

	outputDir := path.Join("output", jID)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Sugar.Errorw("error making output dir",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"jID", jID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}
	outputLog := path.Join(outputDir, "log")
	f, err := os.OpenFile(outputLog, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Sugar.Errorw("error creating or opening append only file",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"jID", jID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}
	defer app.CheckErr(r, f.Close)

	if r.ContentLength == 0 {
		if err := jobsManager.Publish(jID, struct{}{}); err != nil {
			log.Sugar.Errorw("error publishing empty struct for log posted",
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}

		go func() {
			f, err := os.OpenFile(outputLog, os.O_RDONLY, 0644)
			if err != nil {
				log.Sugar.Errorw("error opening read only file",
					"method", r.Method,
					"url", r.URL,
					"err", err.Error(),
					"jID", jID,
				)
				return
			}
			defer app.CheckErr(r, f.Close)

			ctx := context.Background()
			operation := func() error {
				uploadLog := path.Join("output", jID, "log")
				ow := storage.NewWriter(ctx, uploadLog)
				defer app.CheckErr(r, ow.Close)
				if _, err = io.Copy(ow, f); err != nil {
					return fmt.Errorf("copying log file to cloud storage object writer: %v", err)
				}
				return nil
			}
			expBackOff := backoff.NewExponentialBackOff()
			expBackOff.MaxElapsedTime = maxBackoffElapsedTime
			if err := backoff.RetryNotify(operation,
				backoff.WithContext(expBackOff, ctx),
				func(err error, t time.Duration) {
					log.Sugar.Errorw("error uploading output log to gcs--retrying",
						"method", r.Method,
						"url", r.URL,
						"err", err.Error(),
						"jID", jID,
					)
				}); err != nil {
				log.Sugar.Errorw("error uploading output log to gcs--abort",
					"method", r.Method,
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
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"jID", jID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}
	b := buf.Bytes()
	if _, err := io.Copy(f, bytes.NewReader(b)); err != nil {
		log.Sugar.Errorw("error copying buffer to file",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"jID", jID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}
	if err := jobsManager.Publish(jID, b); err != nil {
		log.Sugar.Errorw("error publishing bytes",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"jID", jID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	return nil
}
