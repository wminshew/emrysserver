package payments

import (
	"fmt"
	"github.com/satori/go.uuid"
	stripe "github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/charge"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
)

const baseMinerPenalty = 50

// ChargeMiner pays the miner for job jUUID
func ChargeMiner(jUUID uuid.UUID) {
	aUUID, err := db.GetJobWinner(jUUID)
	if err != nil {
		log.Sugar.Errorw("error getting job winner",
			"err", err.Error(),
			"jID", jUUID,
		)
		return
	}

	stripeAccountID, err := db.GetAccountStripeAccountID(aUUID)
	if err != nil {
		log.Sugar.Errorw("error getting stripe account ID",
			"err", err.Error(),
			"jID", jUUID,
		)
		return
	}

	jobAmount, err := getJobAmount(jUUID)
	if err != nil {
		log.Sugar.Errorw("error getting job amount",
			"err", err.Error(),
			"jID", jUUID,
		)
		return
	}
	jobAmount += baseMinerPenalty

	params := &stripe.ChargeParams{
		Amount:      stripe.Int64(jobAmount),
		Currency:    stripe.String(string(stripe.CurrencyUSD)),
		Description: stripe.String(fmt.Sprintf("Failure penalty for job %s", jUUID.String())),
	}
	if err := params.SetSource(stripeAccountID); err != nil {
		log.Sugar.Errorw("error setting stripe account ID as charge source",
			"err", err.Error(),
			"jID", jUUID,
		)
		return
	}
	ch, err := charge.New(params)
	if err != nil {
		log.Sugar.Errorw("error creating miner charge",
			"err", err.Error(),
			"jID", jUUID,
		)
		return
	}

	if err := db.SetPaymentsMinerCharged(jUUID, ch.ID, ch.Amount); err != nil {
		log.Sugar.Errorw("error setting payments miner charged",
			"err", err.Error(),
			"jID", jUUID,
		)
		return
	}
}
