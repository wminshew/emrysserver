package main

import (
	"encoding/json"
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
	"path"
	"time"
)

const (
	maxRetries = 10
	buffer     = 10
	maxTimeout = 600
)

type pollResponse struct {
	Events    []pollEvent `json:"events"`
	Timestamp int64       `json:"timestamp"`
}

// source: https://github.com/jcuga/golongpoll/blob/master/go-client/glpclient/client.go
type pollEvent struct {
	// Timestamp is milliseconds since epoch to match javascripts Date.getTime()
	Timestamp int64  `json:"timestamp"`
	Category  string `json:"category"`
	// Data can be anything that is able to passed to json.Marshal()
	Data json.RawMessage `json:"data"`
}

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
			req, err := http.NewRequest(http.MethodDelete, u.String(), nil)
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
					"jID", jUUID,
				)
			}); err != nil {
			log.Sugar.Errorw("error deleting notebook user--aborting",
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"jID", jUUID,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "error canceling notebook job"}
		}
	}

	if auctionCompleted, err := db.GetStatusAuctionCompleted(r, jUUID); err != nil {
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"} // err already logged
	} else if !auctionCompleted.IsZero() {
		uID := r.Header.Get("X-Jwt-Claims-Subject")
		uUUID, err := uuid.FromString(uID)
		if err != nil {
			log.Sugar.Errorw("error parsing user ID",
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"jID", jUUID,
			)
			return &app.Error{Code: http.StatusBadRequest, Message: "error parsing jwt"}
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"aud":   "emrys.io",
			"exp":   time.Now().Add(time.Minute * 5).Unix(),
			"iss":   "emrys.io",
			"iat":   time.Now().Unix(),
			"sub":   uUUID,
			"scope": []string{"user"},
		})
		authToken, err := token.SignedString([]byte(authSecret))
		if err != nil {
			log.Sugar.Errorw("error signing token",
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"jID", jUUID,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}

		p := path.Join("job", jUUID.String(), "cancel")
		u := url.URL{
			Scheme: "http",
			Host:   "job-svc:8080",
			Path:   p,
		}
		operation := func() error {
			req, err := http.NewRequest(http.MethodPost, u.String(), nil)
			if err != nil {
				return err
			}
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", authToken))
			req = req.WithContext(ctx)

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
				log.Sugar.Errorw("error posting user cancellation to job-svc, retrying",
					"method", r.Method,
					"url", r.URL,
					"err", err.Error(),
					"jID", jUUID,
				)
			}); err != nil {
			log.Sugar.Errorw("error posting user cancellation to job-svc--aborting",
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"jID", jUUID,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "error canceling notebook job"}
		}

		// wait for output data to be posted
		p = path.Join("job", jUUID.String(), "data", "posted")
		u.Path = p
		q := u.Query()
		q.Set("timeout", fmt.Sprintf("%d", maxTimeout))
		sinceTime := (time.Now().Unix() - buffer) * 1000
		q.Set("since_time", fmt.Sprintf("%d", sinceTime))
		u.RawQuery = q.Encode()
		for {
			pr := pollResponse{}
			operation := func() error {
				req, err := http.NewRequest(http.MethodGet, u.String(), nil)
				if err != nil {
					return err
				}
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", authToken))
				req = req.WithContext(ctx)

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

				if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
					return backoff.Permanent(fmt.Errorf("decoding response: %v", err))
				}

				return nil
			}
			if err := backoff.RetryNotify(operation,
				backoff.WithContext(backoff.WithMaxRetries(backoff.NewExponentialBackOff(), maxBackoffRetries), ctx),
				func(err error, t time.Duration) {
					log.Sugar.Errorw("error polling for output-data-posted, retrying",
						"method", r.Method,
						"url", r.URL,
						"err", err.Error(),
						"jID", jUUID,
					)
				}); err != nil {
				log.Sugar.Errorw("error polling for output-data-posted--aborting",
					"method", r.Method,
					"url", r.URL,
					"err", err.Error(),
					"jID", jUUID,
				)
				return &app.Error{Code: http.StatusInternalServerError, Message: "error canceling notebook job"}
			}

			if len(pr.Events) > 0 {
				break
			}

			// check if job is still active [miner might fail here]
			if active, err := db.GetJobActive(jUUID); err != nil {
				log.Sugar.Errorw("error checking if job is active",
					"method", r.Method,
					"url", r.URL,
					"err", err.Error(),
					"jID", jUUID,
				)
				return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
			} else if !active {
				log.Sugar.Errorw("miner failed to complete job during user cancellation",
					"method", r.Method,
					"url", r.URL,
					"jID", jUUID,
				)
				return &app.Error{Code: http.StatusGone, Message: "the miner failed to upload your output. You will not be charged for this job accordingly"}
			}

			if pr.Timestamp > sinceTime {
				sinceTime = pr.Timestamp
			}

			q = u.Query()
			q.Set("since_time", fmt.Sprintf("%d", sinceTime))
			u.RawQuery = q.Encode()
		}
	}

	return db.SetJobCanceledAndDebitUser(r, jUUID)
}
