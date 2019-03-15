package main

import (
	"fmt"
	"github.com/cenkalti/backoff"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrys/pkg/check"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	del        = "DELETE"
	maxRetries = 5
)

// postCancelJob handles user job cancellations
var postCancelJob app.Handler = func(w http.ResponseWriter, r *http.Request) *app.Error {
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
	log.Sugar.Infow("user canceled job",
		"method", r.Method,
		"url", r.URL,
		"jID", jUUID,
	)

	nbQuery := r.URL.Query().Get("notebook")
	notebook := (nbQuery == "1")

	ctx := r.Context()
	client := http.Client{}
	if notebook {
		u := url.URL{
			Scheme: "http",
			Host:   "notebook-svc:8080",
			Path:   "user",
		}
		q := u.Query()
		q.Set("jID", jUUID.String())
		u.RawQuery = q.Encode()

		operation := func() error {
			req, err := http.NewRequest(del, u.String(), nil)
			if err != nil {
				return err
			}

			resp, err := client.Do(req)
			if err != nil {
				return err
			}
			defer check.Err(resp.Body.Close)

			if resp.StatusCode == http.StatusBadGateway {
				return fmt.Errorf("server: temporary error")
			} else if resp.StatusCode >= 300 {
				b, _ := ioutil.ReadAll(resp.Body)
				return backoff.Permanent(fmt.Errorf("server: %v", string(b)))
			}

			return nil
		}
		if err := backoff.RetryNotify(operation,
			backoff.WithContext(backoff.WithMaxRetries(backoff.NewExponentialBackOff(), maxBackoffRetries), ctx),
			func(err error, t time.Duration) {
				log.Sugar.Errorw("error deleting notebook user, retrying",
					"method", r.Method,
					"url", r.URL,
					"err", err.Error(),
				)
			}); err != nil {
			log.Sugar.Errorw("error deleting notebook user, aborting",
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "error canceling notebook job"}
		}
	}

	// wait for output data to be posted
	log.Sugar.Infof("wait for output data")
	for {
		if tOutputDataPosted, err := db.GetStatusOutputData(r, jUUID); err != nil {
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"} // err already logged
		} else if !tOutputDataPosted.IsZero() {
			break
		}

		// check if job is still active [miner might fail here]
		if active, err := db.GetJobActive(r, jUUID); err != nil {
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"} // err already logged
		} else if !active {
			return &app.Error{Code: http.StatusOK, Message: "the miner failed to upload your output. You will not be charged for this job accordingly"} // err already logged
		}
		log.Sugar.Infof("sleeping...")
		time.Sleep(10 * time.Second)
	}

	log.Sugar.Infof("canceling job and debiting user...")
	return db.SetJobCanceledAndDebitUser(r, jUUID)
}
