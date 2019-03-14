package main

import (
	"fmt"
	"github.com/cenkalti/backoff"
	"github.com/dgrijalva/jwt-go"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrys/pkg/check"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	activeWorker = make(map[uuid.UUID]chan struct{})
)

const (
	baseMinerPenalty = 0.5
	maxRetries       = 5
	post             = "POST"
)

func monitorJob(jUUID uuid.UUID) {
	activeWorker[jUUID] = make(chan struct{})
	defer delete(activeWorker, jUUID)
	for {
		select {
		case <-time.After(time.Second * time.Duration(minerTimeout)):
			mUUID, err := db.GetJobWinner(jUUID)
			if err != nil {
				log.Sugar.Errorw("error getting job winner",
					"err", err.Error(),
					"jID", jUUID,
				)
			}
			log.Sugar.Infow("miner failed job",
				"jID", jUUID,
			)
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
					"err", err.Error(),
					"jID", jUUID,
				)
			}

			client := http.Client{}
			u := url.URL{
				Scheme: "http",
				Host:   "job-svc:8080",
				Path:   fmt.Sprintf("job/%s/log", jUUID),
			}
			operation := func() error {
				req, err := http.NewRequest(post, u.String(), strings.NewReader("ERROR: supplier "+
					"has crashed. Please re-submit this job, you will not be charged."))
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
						"err", err.Error(),
						"jID", jUUID,
					)
				}); err != nil {
				log.Sugar.Errorw("error posting error to job output log--abort",
					"err", err.Error(),
					"jID", jUUID,
				)
				return
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
						"err", err.Error(),
						"jID", jUUID,
					)
				}); err != nil {
				log.Sugar.Errorw("error posting error to job output log--abort",
					"err", err.Error(),
					"jID", jUUID,
				)
				return
			}

			if err := db.SetJobFailed(jUUID, baseMinerPenalty); err != nil {
				log.Sugar.Errorw("error setting job failed",
					"err", err.Error(),
					"jID", jUUID,
				)
				return
			}
			return
		case <-activeWorker[jUUID]:
		}
	}
}
