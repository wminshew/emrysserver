package main

import (
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// postStripeToken adds a stripe payment token to the user's account
var postStripeToken app.Handler = func(w http.ResponseWriter, r *http.Request) *app.Error {
	if err := r.ParseForm(); err != nil {
		log.Sugar.Errorw("error parsing request form",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "error parsing request form"}
	}

	stripeToken := r.Form.Get("stripeToken")
	if stripeToken == "" {
		log.Sugar.Errorw("error retrieving stripe token from form",
			"method", r.Method,
			"url", r.URL,
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "no stripe token included in request"}
	}

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

	if err := db.SetAccountStripeToken(r, aUUID, stripeToken); err != nil {
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"} // already logged
	}

	return nil
}
