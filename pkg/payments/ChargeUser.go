package payments

import (
	"context"
	"fmt"
	"github.com/cenkalti/backoff"
	"github.com/satori/go.uuid"
	stripe "github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/invoiceitem"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
	"time"
)

const (
	maxBackoffElapsedTime = 72 * time.Hour
)

// ChargeUser charges the user for job jUUID
func ChargeUser(r *http.Request, stripeInvoiceItemC *invoiceitem.Client, jUUID uuid.UUID) {
	aUUID, err := db.GetJobOwner(r, jUUID)
	if err != nil {
		log.Sugar.Errorw("error getting job owner",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"jID", jUUID,
		)
		return
	}

	stripeCustomerID, err := db.GetAccountStripeCustomerID(r, aUUID)
	if err != nil {
		log.Sugar.Errorw("error getting stripe customer ID",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"jID", jUUID,
		)
		return
	}

	stripeSubscriptionID, err := db.GetAccountStripeSubscriptionID(r, aUUID)
	if err != nil {
		log.Sugar.Errorw("error getting stripe subscription ID",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"jID", jUUID,
		)
		return
	}

	jobAmount, err := getJobAmount(jUUID)
	if err != nil {
		log.Sugar.Errorw("error getting job amount",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"jID", jUUID,
		)
		return
	}

	credit, err := db.GetAccountCredit(aUUID)
	if err != nil {
		log.Sugar.Errorw("error getting account credit",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"jID", jUUID,
		)
		return
	}

	if credit >= jobAmount {
		credit -= jobAmount
		if err := db.SetAccountCredit(aUUID, credit); err != nil {
			log.Sugar.Errorw("error setting account credit",
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"jID", jUUID,
			)
			return
		}

		if err := db.SetPaymentsUserCharged(jUUID, "", 0, jobAmount); err != nil {
			log.Sugar.Errorw("error setting payments user charged",
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"jID", jUUID,
			)
			return
		}

		return
	} else if credit > 0 {
		jobAmount -= credit
		if err := db.SetAccountCredit(aUUID, 0); err != nil {
			log.Sugar.Errorw("error setting account credit",
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"jID", jUUID,
			)
			return
		}
	}

	params := &stripe.InvoiceItemParams{
		Customer:     stripe.String(stripeCustomerID),
		Subscription: stripe.String(stripeSubscriptionID),
		Amount:       stripe.Int64(jobAmount),
		Currency:     stripe.String(string(stripe.CurrencyUSD)),
		Description:  stripe.String(fmt.Sprintf("Payment for job %s", jUUID.String())),
	}
	params.SetIdempotencyKey(uuid.NewV4().String())

	ctx := context.Background()
	ii := &stripe.InvoiceItem{}
	operation := func() error {
		var err error
		ii, err = stripeInvoiceItemC.New(params)
		return err
	}
	expBackOff := backoff.NewExponentialBackOff()
	expBackOff.MaxElapsedTime = maxBackoffElapsedTime
	if err := backoff.RetryNotify(operation,
		backoff.WithContext(expBackOff, ctx),
		func(err error, t time.Duration) {
			log.Sugar.Errorw("error creating user invoice, retrying",
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"jID", jUUID,
			)
		}); err != nil {
		log.Sugar.Errorw("error creating user invoice--aborting",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"jID", jUUID,
		)
		return
	}

	if err := db.SetPaymentsUserCharged(jUUID, ii.ID, ii.Amount, credit); err != nil {
		log.Sugar.Errorw("error setting payments user charged",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"jID", jUUID,
		)
		return
	}
}
