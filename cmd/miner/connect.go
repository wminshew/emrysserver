package main

import (
	"fmt"
	"github.com/cenkalti/backoff"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
	"time"
)

// connect handles miner requests to /miner/connect, establishing
// a pubsub pattern for new jobs to be distributed for bidding
var connect app.Handler = func(w http.ResponseWriter, r *http.Request) *app.Error {
	mID := r.Header.Get("X-Jwt-Claims-Subject")
	mUUID, err := uuid.FromString(mID)
	if err != nil {
		log.Sugar.Errorw("error parsing miner ID",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "error parsing miner ID"}
	}

	acctID, err := db.GetAccountStripeAccountID(mUUID)
	if err != nil {
		log.Sugar.Errorw("error getting stripe account ID",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	} else if acctID == "" {
		log.Sugar.Infow("no stripe payout account",
			"method", r.Method,
			"url", r.URL,
			"mID", mUUID,
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "no stripe payout account. " +
			"Please verify your payout information at https://www.emrys.io/account and reach out to support if problems continue."}
	}

	// TODO: replace with stripe backend configuration with retries built in?
	ctx := r.Context()
	operation := func() error {
		_, err = stripeAccountC.GetByID(acctID, nil)
		return err
	}
	if err := backoff.RetryNotify(operation,
		backoff.WithContext(backoff.WithMaxRetries(backoff.NewExponentialBackOff(), maxRetries), ctx),
		func(err error, t time.Duration) {
			log.Sugar.Errorw("miner's stripe account not recognized or inactive, retrying",
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
			)
		}); err != nil {
		log.Sugar.Errorw("miner's stripe account not recognized or inactive--aborting",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "your stripe account is inactive or non-existent. " +
			"Please verify your payout information on https://www.emrys.io/account and reach out to support if problems continue."}
	}

	q := r.URL.Query()
	q.Set("category", "jobs")
	q.Set("timeout", fmt.Sprintf("%d", maxTimeout))
	r.URL.RawQuery = q.Encode()
	minerManager.SubscriptionHandler(w, r)
	return nil
}
