package main

import (
	"fmt"
	"github.com/satori/go.uuid"
	"github.com/stripe/stripe-go/account"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
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
	}

	_, err = account.GetByID(acctID, nil)
	if err != nil {
		log.Sugar.Errorw("miner connected w/o stripe account",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "your stripe account is inactive or non-existent. " +
			"Please verify your payout information on https://www.emrys.io and reach out to support if problems continue."}
	}

	q := r.URL.Query()
	q.Set("category", "jobs")
	q.Set("timeout", fmt.Sprintf("%d", maxTimeout))
	r.URL.RawQuery = q.Encode()
	minerManager.SubscriptionHandler(w, r)
	return nil
}
