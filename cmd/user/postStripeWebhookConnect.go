package main

import (
	"github.com/stripe/stripe-go/webhook"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/email"
	"github.com/wminshew/emrysserver/pkg/log"
	"io/ioutil"
	"net/http"
)

// postStripeWebhookConnect handles webhooks from stripe for emrys connected accounts
var postStripeWebhookConnect app.Handler = func(w http.ResponseWriter, r *http.Request) *app.Error {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Sugar.Errorw("error reading stripe webhook request body",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	event, err := webhook.ConstructEvent(body, r.Header.Get("Stripe-Signature"), stripeWebhookSecretConnect)
	if err != nil {
		log.Sugar.Errorw("error verifying stripe webhook signature",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "error parsing job ID"}
	}

	go func() {
		log.Sugar.Infof("%+v", event)

		switch event.Type {
		case "payout.paid":
			log.Sugar.Infow("payout.paid",
				"amt", event.GetObjectValue("amount"),
				"tx", event.GetObjectValue("balance_transaction"),
				"dest", event.GetObjectValue("destination"),
				"status", event.GetObjectValue("status"),
			)
		case "payout.failed":
			amt := event.GetObjectValue("amount")
			dest := event.GetObjectValue("destination")
			log.Sugar.Errorw("payout.failed",
				"amt", amt,
				"tx", event.GetObjectValue("balance_transaction"),
				"dest", dest,
				"status", event.GetObjectValue("status"),
			)
			if err := email.SendPayoutFailed(dest, amt); err != nil {
				log.Sugar.Errorw("error sending payout-failed email to support",
					"method", r.Method,
					"url", r.URL,
					"err", err.Error(),
				)
				return
			}
		default:
		}
	}()

	return nil
}
