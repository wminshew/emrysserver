package main

import (
	"context"
	"fmt"
	"github.com/cenkalti/backoff"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrys/pkg/check"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"github.com/wminshew/emrysserver/pkg/storage"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"time"
)

const maxRetries = 10

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
			backoff.WithMaxRetries(backoff.NewExponentialBackOff(), maxRetries),
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
		var err error
		defer func() {
			if err == nil {
				// post to {jID}-output-data-posted
				mID := r.Header.Get("X-Jwt-Claims-Subject")
				mUUID, err := uuid.FromString(mID)
				if err != nil {
					log.Sugar.Errorw("error parsing miner ID",
						"method", r.Method,
						"url", r.URL,
						"err", err.Error(),
						"jID", jUUID,
					)
					return
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
					return
				}

				client := http.Client{}
				p := path.Join("job", jUUID.String(), "cancel")
				u := url.URL{
					Scheme: "http",
					Host:   "job-svc:8080",
					Path:   p,
				}
				ctx := r.Context()
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
					backoff.WithContext(backoff.WithMaxRetries(backoff.NewExponentialBackOff(), maxRetries), ctx),
					func(err error, t time.Duration) {
						log.Sugar.Errorw("error posting output-data-posted to longpoll, retrying",
							"method", r.Method,
							"url", r.URL,
							"err", err.Error(),
							"jID", jUUID,
						)
					}); err != nil {
					log.Sugar.Errorw("error posting output-data-posted to longpoll--aborting",
						"method", r.Method,
						"url", r.URL,
						"err", err.Error(),
						"jID", jUUID,
					)
					return
				}
			}
		}()

		if err = db.SetStatusOutputDataPosted(jUUID); err != nil {
			log.Sugar.Errorw("error setting output data posted status",
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}

		return nil
	}
	return db.SetJobFinishedAndStatusOutputDataPostedAndDebitUser(r, jUUID)
}
