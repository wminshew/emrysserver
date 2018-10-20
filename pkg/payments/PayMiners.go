package payments

import (
	"fmt"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"time"
)

const (
	fixedPenalty = 0.5
)

// PayMiners pays all miners their outstanding balances
func PayMiners() {
	// TODO: add retry logic?
	minerCredit := make(map[uuid.UUID]float64)

	rows, err := db.GetOutstandingMinerPayments()
	if err != nil {
		log.Sugar.Errorw("error getting outstanding miner payments",
			"err", err.Error(),
		)
		return
	}
	defer func() { _ = rows.Close() }()

	// organize into balance per miner
	for rows.Next() {
		var jUUID, mUUID uuid.UUID
		var payRate float64
		var auctionCompleted, jobCompletedAt, jobCanceledAt, jobFailedAt time.Time
		if err = rows.Scan(&jUUID, &mUUID, &payRate, &auctionCompleted, &jobCompletedAt, &jobCanceledAt, &jobFailedAt); err != nil {
			log.Sugar.Errorw("error scanning outstanding miner payments",
				"err", err.Error(),
			)
			return
		}
		if !jobCompletedAt.IsZero() {
			minerCredit[mUUID] += payRate * jobCompletedAt.Sub(auctionCompleted).Hours()
		} else if !jobCanceledAt.IsZero() {
			minerCredit[mUUID] += payRate * jobCanceledAt.Sub(auctionCompleted).Hours()
		} else if !jobFailedAt.IsZero() {
			minerCredit[mUUID] -= fixedPenalty + payRate*jobFailedAt.Sub(auctionCompleted).Hours()
		} else {
			// should never reach here
			log.Sugar.Errorw("PANIC: error scanning outstanding miner payments: job is inactive but all finish states are null",
				"jID", jUUID,
			)
		}
	}
	if err = rows.Err(); err != nil {
		log.Sugar.Errorw("error scanning outstanding miner payments",
			"err", err.Error(),
		)
		return
	}

	for mUUID, credit := range minerCredit {
		if credit > 0 {
			// TODO: add retry logic
			// TODO: stripe or btcpay

			log.Sugar.Infow(fmt.Sprintf("miner paid %.2f", credit),
				"mID", mUUID,
			)

			// TODO: add retry logic
			if err := db.SetMinerPaid(mUUID); err != nil {
				log.Sugar.Errorw("error updating miner payments",
					"err", err.Error(),
					"mID", mUUID,
				)
			}
		}
	}

}
