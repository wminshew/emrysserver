package main

import (
	"fmt"
	"github.com/cenkalti/backoff"
	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrys/pkg/check"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

const (
	maxRetries = 10
)

// postJob handles new jobs posted by users
var postJob app.Handler = func(w http.ResponseWriter, r *http.Request) *app.Error {
	vars := mux.Vars(r)
	project := vars["project"]
	uID := r.Header.Get("X-Jwt-Claims-Subject")
	uUUID, err := uuid.FromString(uID)
	if err != nil {
		log.Sugar.Errorw("error parsing user ID",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "error parsing user ID"}
	}

	subID, err := db.GetAccountStripeSubscriptionID(r, uUUID)
	if err != nil {
		log.Sugar.Errorw("error getting stripe subscription ID",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	} else if subID == "" {
		log.Sugar.Errorw("user posted job with no stripe subscription",
			"method", r.Method,
			"url", r.URL,
			"uID", uUUID,
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "no payment information on file. " +
			"Please verify your payment information on https://www.emrys.io/account and reach out to support if problems continue."}
	}

	sub, err := stripeSubC.Get(subID, nil)
	if err != nil {
		log.Sugar.Errorw("error getting stripe subscription",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	if sub.Status != "active" {
		log.Sugar.Errorw("user posted job with inactive stripe subscription",
			"method", r.Method,
			"url", r.URL,
			"uID", uUUID,
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "your stripe subscription is currently inactive. " +
			"Please verify your payment information on https://www.emrys.io/account and reach out to support if problems continue."}
	}

	jobID := uuid.NewV4()
	w.Header().Set("X-Job-ID", jobID.String())

	nbQuery := r.URL.Query().Get("notebook")
	notebook := (nbQuery == "1")

	if err := db.InsertJob(r, uUUID, project, jobID, notebook); err != nil {
		log.Sugar.Errorw("error inserting job",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "error inserting job"}
	}

	if notebook {
		ctx := r.Context()
		client := &http.Client{}
		u := url.URL{
			Scheme: "http",
			Host:   "notebook-svc:8080",
			Path:   "user",
		}
		q := u.Query()
		q.Set("jID", jobID.String())
		u.RawQuery = q.Encode()
		var sshKeyBytes []byte

		operation := func() error {
			req, err := http.NewRequest(http.MethodPost, u.String(), nil)
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

			sshKeyBytes, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				return backoff.Permanent(fmt.Errorf("reading response: %v", err))
			}
			return nil
		}
		if err := backoff.RetryNotify(operation,
			backoff.WithContext(backoff.WithMaxRetries(backoff.NewExponentialBackOff(), maxRetries), ctx),
			func(err error, t time.Duration) {
				log.Sugar.Errorw("error adding notebook user, retrying",
					"method", r.Method,
					"url", r.URL,
					"err", err.Error(),
				)
			}); err != nil {
			log.Sugar.Errorw("error adding notebook user, aborting",
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "error posting notebook job"}
		}
		// TODO: add json wrapper?
		if _, err := w.Write(sshKeyBytes); err != nil {
			log.Sugar.Errorw("error returning ssh key",
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "error returning ssh key"}
		}
	}

	return nil
}
