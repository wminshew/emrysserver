package main

import (
	// "fmt"
	// "github.com/satori/go.uuid"
	// stripe "github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/webhook"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/log"
	"io/ioutil"
	"net/http"
)

// postStripeWebhookAccount handles webhooks from stripe for emrys account
var postStripeWebhookAccount app.Handler = func(w http.ResponseWriter, r *http.Request) *app.Error {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Sugar.Errorw("error reading stripe webhook request body",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	event, err := webhook.ConstructEvent(body, r.Header.Get("Stripe-Signature"), stripeWebhookSecretAccount)
	if err != nil {
		log.Sugar.Errorw("error verrifying stripe webhook signature",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "error parsing job ID"}
	}

	go func() {
		log.Sugar.Infof("%+v", event)
	}()

	return nil
}
