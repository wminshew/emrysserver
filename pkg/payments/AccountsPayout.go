package payments

import (
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
)

const payoutThreshold = 5

// AccountsPayout charges all users & pays all miners their outstanding balances, subject to a threshold
func AccountsPayout() {
	log.Sugar.Infof("Accounts payout: beginning")

	accountBalances := make(map[uuid.UUID]float64)

	rows, err := db.GetAccountBalances(payoutThreshold)
	if err != nil {
		return // already logged
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Sugar.Errorf("Error closing rows")
		}
	}()

	for rows.Next() {
		var aUUID uuid.UUID
		var b float64
		if err = rows.Scan(&aUUID, &b); err != nil {
			log.Sugar.Errorw("error scanning account balances",
				"err", err.Error(),
			)
			return
		}
		accountBalances[aUUID] = b
	}
	if err = rows.Err(); err != nil {
		log.Sugar.Errorw("error scanning account balances",
			"err", err.Error(),
		)
		return
	}

	// for aUUID, b := range accountBalances {
	// TODO: stripe payout w/ retry

	// if success {
	// 	log.Sugar.Infow("account balance processed",
	// 		"aID", aUUID,
	// 		"balance", b,
	// 	)
	//
	// 	// err logged inside function; want to continue regardless to remaining accounts
	// 	_ = db.SetAccountBalance(aUUID, 0)
	// } else {
	// 	log.Sugar.Errorw("error paying out account",
	// 		"aID", aUUID,
	// 		"balance", b,
	// 	)
	// }
	// }

	log.Sugar.Infof("Accounts payout: complete")
}
