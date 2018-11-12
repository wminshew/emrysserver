package db

import (
	"database/sql"
	"github.com/lib/pq"
	"github.com/wminshew/emrysserver/pkg/log"
)

// GetOutstandingMinerPayments returns the jobs for which miners haven't been paid
func GetOutstandingMinerPayments() (*sql.Rows, error) {
	sqlStmt := `
	SELECT p.job_uuid, b.miner_uuid, j.rate, s.auction_completed, j.completed_at, j.canceled_at, j.failed_at
	FROM payments p
	INNER JOIN jobs j ON (p.job_uuid = j.uuid)
	INNER JOIN statuses s ON (p.job_uuid = s.job_uuid)
	INNER JOIN bids b ON (j.win_bid_uuid = b.uuid)
	WHERE p.miner_paid IS NULL
		AND j.active = false
	`
	rows, err := db.Query(sqlStmt)
	if err != nil {
		message := "error querying for outstanding miner payments"
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
