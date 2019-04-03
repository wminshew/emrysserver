package main

import (
	"context"
	"fmt"
	"github.com/cenkalti/backoff"
	"github.com/dgrijalva/jwt-go"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrys/pkg/check"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"github.com/wminshew/emrysserver/pkg/payments"
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
	maxRetries = 10
)

func monitorJob(jUUID uuid.UUID, notebook bool) {
	activeWorker[jUUID] = make(chan struct{})
	defer delete(activeWorker, jUUID)
	for {
		select {
		case <-time.After(time.Second * time.Duration(minerTimeout)):
			// check if job has completed or been canceled [i.e. is active]
			if active, err := db.GetJobActive(jUUID); err != nil {
				log.Sugar.Errorw("error checking if job is active",
					"jID", jUUID,
				)
				return
			} else if !active {
				log.Sugar.Infow("removing job monitoring from inactive job",
					"jID", jUUID,
				)
				return
			}

			ctx := context.Background()
			client := &http.Client{}
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
					backoff.WithContext(backoff.WithMaxRetries(backoff.NewExponentialBackOff(), maxRetries), ctx),
					func(err error, t time.Duration) {
						log.Sugar.Errorw("error deleting notebook user, retrying",
							"err", err.Error(),
							"jID", jUUID,
						)
					}); err != nil {
					log.Sugar.Errorw("error deleting notebook user--aborting",
						"err", err.Error(),
						"jID", jUUID,
					)
					return
				}
			}

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

			u := url.URL{
				Scheme: "http",
				Host:   "job-svc:8080",
				Path:   fmt.Sprintf("job/%s/log", jUUID),
			}
			operation := func() error {
				req, err := http.NewRequest(http.MethodPost, u.String(), strings.NewReader("ERROR: supplier "+
					"has crashed. Please re-submit this job, you will not be charged.\n"))
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
				backoff.WithContext(backoff.WithMaxRetries(backoff.NewExponentialBackOff(), maxRetries), ctx),
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
				req, err := http.NewRequest(http.MethodPost, u.String(), nil)
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
				backoff.WithContext(backoff.WithMaxRetries(backoff.NewExponentialBackOff(), maxRetries), ctx),
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

			if err := db.SetJobFailed(jUUID); err != nil {
				return // already logged
			}
			go payments.ChargeMiner(jUUID)

			return
		case <-activeWorker[jUUID]:
		}
	}
}
