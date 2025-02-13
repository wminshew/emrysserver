package main

import (
	"github.com/satori/go.uuid"
	stripe "github.com/stripe/stripe-go"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// getStripeDashboard returns the user's stripe dashboard url
var getStripeDashboard app.Handler = func(w http.ResponseWriter, r *http.Request) *app.Error {
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

	stripeAccountID, err := db.GetAccountStripeAccountID(aUUID)
	if err != nil {
		log.Sugar.Errorw("error getting stripe account ID",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	} else if stripeAccountID == "" {
		log.Sugar.Errorw("account has no stripe account ID but trying to access dashboard",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "No stripe account ID exists for this acount"}
	}

	params := &stripe.LoginLinkParams{
		Account: stripe.String(stripeAccountID),
	}
	link, err := stripeLoginLinkC.New(params)
	if err != nil {
		log.Sugar.Errorw("error getting new stripe dashboard link",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	if _, err := w.Write([]byte(link.URL)); err != nil {
		log.Sugar.Errorw("error writing account stripe dashboard url",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	return nil
}
