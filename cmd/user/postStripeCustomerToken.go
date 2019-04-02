package main

import (
	"github.com/satori/go.uuid"
	stripe "github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/customer"
	"github.com/stripe/stripe-go/sub"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// postStripeCustomerToken creates or updates the account's stripe customer with payment info,
// and if appropriate subscribes them to emrys-user-access
var postStripeCustomerToken app.Handler = func(w http.ResponseWriter, r *http.Request) *app.Error {
	stripeToken := r.URL.Query().Get("stripeToken")
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

	stripeCustomerID, err := db.GetAccountStripeCustomerID(r, aUUID)
	if err != nil {
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"} // already logged
	}

	if stripeCustomerID != "" {
		customerParams := &stripe.CustomerParams{}
		if err := customerParams.SetSource(stripeToken); err != nil {
			log.Sugar.Errorw("error setting customer source with stripe token",
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}
		cus, err := customer.Update(stripeCustomerID, customerParams)
		if err != nil {
			log.Sugar.Errorw("error updating customer source with stripe token",
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}

		if err := db.SetAccountStripeCustomerLast4(r, aUUID, cus.Sources.Data[0].Card.Last4); err != nil {
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"} // already logged
		}
	} else {
		userEmail, err := db.GetAccountEmail(r, aUUID)
		if err != nil {
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"} // already logged
		}

		customerParams := &stripe.CustomerParams{
			Email: stripe.String(userEmail),
		}
		if err := customerParams.SetSource(stripeToken); err != nil {
			log.Sugar.Errorw("error setting customer source with stripe token",
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}
		cus, err := customer.New(customerParams)
		if err != nil {
			log.Sugar.Errorw("error creating new stripe customer",
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}
		stripeCustomerID = cus.ID

		if err := db.SetAccountStripeCustomerID(r, aUUID, stripeCustomerID); err != nil {
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"} // already logged
		}

		if err := db.SetAccountStripeCustomerLast4(r, aUUID, cus.Sources.Data[0].Card.Last4); err != nil {
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"} // already logged
		}
	}

	stripeSubID, err := db.GetAccountStripeSubscriptionID(r, aUUID)
	if err != nil {
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"} // already logged
	} else if stripeSubID == "" {

		subItems := []*stripe.SubscriptionItemsParams{
			{
				Plan: stripe.String(stripePlanID),
			},
		}
		subParams := &stripe.SubscriptionParams{
			Customer: stripe.String(stripeCustomerID),
			Items:    subItems,
		}
		subscription, err := sub.New(subParams)
		if err != nil {
			log.Sugar.Errorw("error creating new stripe subscription",
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}

		if err := db.SetAccountStripeSubscriptionID(r, aUUID, subscription.ID); err != nil {
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"} // already logged
		}
	}

	return nil
}
