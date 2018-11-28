package db

import (
	"database/sql"
	"github.com/lib/pq"
	"github.com/wminshew/emrysserver/pkg/log"
)

// GetAccountBalances returns rows holding account balances above the payout
// threshold
func GetAccountBalances(payoutThreshold float64) (*sql.Rows, error) {
	sqlStmt := `
	SELECT a.uuid, a.balance
	FROM accounts a
	WHERE a.confirmed = true AND
		a.suspended = false AND
		(a.balance < 0 OR
			a.balance > $1)
	`
	rows, err := db.Query(sqlStmt, payoutThreshold)
	if err != nil {
		message := "error querying for account balances"
		pqErr, ok := err.(*pq.Error)
		if ok {
			log.Sugar.Errorw(message,
				"err", err.Error(),
				"pq_sev", pqErr.Severity,
				"pq_code", pqErr.Code,
				"pq_detail", pqErr.Detail,
			)
		} else {
			log.Sugar.Errorw(message,
				"err", err.Error(),
			)
		}
	}
	return rows, err
}
