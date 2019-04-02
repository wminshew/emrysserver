package payments

import (
	"fmt"
	"github.com/satori/go.uuid"
	stripe "github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/transfer"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// PayMiner pays the miner for job jUUID
func PayMiner(r *http.Request, jUUID uuid.UUID) {
	aUUID, err := db.GetJobWinner(jUUID)
	if err != nil {
		log.Sugar.Errorw("error getting job winner",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"jID", jUUID,
		)
		return
	}

	stripeAccountID, err := db.GetAccountStripeAccountID(aUUID)
	if err != nil {
		log.Sugar.Errorw("error getting stripe account ID",
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

	params := &stripe.TransferParams{
		Destination:   stripe.String(stripeAccountID),
		Amount:        stripe.Int64(jobAmount),
		Currency:      stripe.String(string(stripe.CurrencyUSD)),
		TransferGroup: stripe.String(fmt.Sprintf("Payout for job %s", jUUID.String())),
	}
	t, err := transfer.New(params)
	if err != nil {
		log.Sugar.Errorw("error creating miner transfer",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"jID", jUUID,
		)
		return
	}

	if err := db.SetPaymentsMinerPaid(jUUID, t.ID, t.Amount); err != nil {
		log.Sugar.Errorw("error setting payments miner paid",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"jID", jUUID,
		)
		return
	}
}
