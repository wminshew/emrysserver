package main

import (
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// getStripeCustomerLast4 returns the account's stripe's card last4
var getStripeCustomerLast4 app.Handler = func(w http.ResponseWriter, r *http.Request) *app.Error {
	aID := r.Header.Get("X-Jwt-Claims-Subject")
	aUUID, err := uuid.FromString(aID)
	if err != nil {
		log.Sugar.Errorw("error parsing account ID",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "error parsing job ID"}
	}

	stripeCardLast4, err := db.GetAccountStripeCustomerLast4(r, aUUID)
	if err != nil {
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"} // already logged
	}

	if _, err := w.Write([]byte(stripeCardLast4)); err != nil {
		log.Sugar.Errorw("error writing account stripe card last4",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	return nil
}
