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

	// save log, if exists
	mUUID, err := db.GetJobWinner(jUUID)
	if err != nil {
		log.Sugar.Errorw("error getting job winner",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"jID", jUUID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "error canceling notebook job"}
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"aud":   "emrys.io",
		"exp":   time.Now().Add(time.Minute * 5).Unix(),
		"iss":   "emrys.io",
		"iat":   time.Now().Unix(),
		"sub":   mUUID,
		"scope": []string{"miner"},
	})

	authToken, err := token.SignedString([]byte(authSecret))
	if err != nil {
		log.Sugar.Errorw("error signing token",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"jID", jUUID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "error canceling notebook job"}
	}

	u := url.URL{
		Scheme: "http",
		Host:   "job-svc:8080",
		Path:   fmt.Sprintf("job/%s/log", jUUID),
	}
	operation := func() error {
		req, err := http.NewRequest(post, u.String(), strings.NewReader("JOB CANCELED."))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", authToken))

		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer check.Err(resp.Body.Close)

		if resp.StatusCode == http.StatusBadGateway {
			return fmt.Errorf("server: temporary error")
		} else if resp.StatusCode >= 300 {
			b, _ := ioutil.ReadAll(resp.Body)
			return fmt.Errorf("server: %v", string(b))
		}

		return nil
	}
	if err := backoff.RetryNotify(operation,
		backoff.WithMaxRetries(backoff.NewExponentialBackOff(), maxRetries),
		func(err error, t time.Duration) {
			log.Sugar.Errorw("error posting cancellation to job output log--retrying",
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"jID", jUUID,
			)
		}); err != nil {
		log.Sugar.Errorw("error posting cancellation to job output log--abort",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"jID", jUUID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "error canceling notebook job"}
	}

	operation = func() error {
		// POST with empty body signifies log upload complete
		req, err := http.NewRequest(post, u.String(), nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", authToken))

		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer check.Err(resp.Body.Close)

		if resp.StatusCode == http.StatusBadGateway {
			return fmt.Errorf("server: temporary error")
		} else if resp.StatusCode >= 300 {
			b, _ := ioutil.ReadAll(resp.Body)
			return fmt.Errorf("server: %v", string(b))
		}

		return nil
	}
	if err := backoff.RetryNotify(operation,
		backoff.WithMaxRetries(backoff.NewExponentialBackOff(), maxRetries),
		func(err error, t time.Duration) {
			log.Sugar.Errorw("error posting error to job output log--retrying",
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"jID", jUUID,
			)
		}); err != nil {
		log.Sugar.Errorw("error posting error to job output log--abort",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"jID", jUUID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "error canceling notebook job"}
	}

	return db.SetJobCanceledAndDebitUser(r, jUUID)
}
