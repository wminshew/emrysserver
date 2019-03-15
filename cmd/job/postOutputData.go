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
var postOutputData app.Handler = func(w http.ResponseWriter, r *http.Request) *app.Error {
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

	if tDataDownloaded, tImageDownloaded, tOutputLogPosted, err := db.GetStatusOutputDataPrereqs(r, jUUID); err != nil {
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"} // err already logged
	} else if tDataDownloaded.IsZero() || tImageDownloaded.IsZero() || tOutputLogPosted.IsZero() {
		log.Sugar.Infow("miner tried to post output data without completing prereqs",
			"method", r.Method,
			"url", r.URL,
			"jID", jID,
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "must successfully download data, image and post output log before posting output data"}
	}

	jcQuery := r.URL.Query().Get("jobcanceled")
	jobCanceled := (jcQuery == "1")

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
	p := path.Join(outputDir, "data.tar.gz")
	f, err := os.Create(p)
	if err != nil {
		log.Sugar.Errorw("error creating output data.tar.gz",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"jID", jID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}
	if _, err = io.Copy(f, r.Body); err != nil {
		log.Sugar.Errorw("error copying data.tar.gz to file",
			"method", r.Method,
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
			ctx := context.Background()
			f, err := os.Open(p)
			if err != nil {
				return fmt.Errorf("opening output data.tar.gz: %v", err)
			}
			defer app.CheckErr(r, f.Close)
			ow := storage.NewWriter(ctx, p)
			defer app.CheckErr(r, ow.Close)
			if _, err = io.Copy(ow, f); err != nil {
				return fmt.Errorf("copying tee reader to cloud storage object writer: %v", err)
			}
			return nil
		}
		if err := backoff.RetryNotify(operation,
			backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 10),
			func(err error, t time.Duration) {
				log.Sugar.Errorw("error uploading output data.tar.gz to gcs--retrying",
					"method", r.Method,
					"url", r.URL,
					"err", err.Error(),
					"jID", jID,
				)
			}); err != nil {
			log.Sugar.Errorw("error uploading output data.tar.gz to gcs--abort",
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return
		}
		go func() {
			defer app.CheckErr(r, func() error { return os.Remove(p) }) // no need to cache locally
			time.Sleep(15 * time.Minute)
		}()
	}()

	if jobCanceled {
		return db.SetStatusOutputDataPosted(r, jUUID)
	}
	return db.SetJobFinishedAndStatusOutputDataPostedAndDebitUser(r, jUUID)
}
