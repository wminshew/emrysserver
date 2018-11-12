package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
)

// SetMinerPaid updates db to set miner_paid on jobs miner has now been paid for
func SetMinerPaid(mUUID uuid.UUID) error {
	sqlStmt := `
	UPDATE payments p
	INNER JOIN jobs j ON (p.job_uuid = j.uuid)
	INNER JOIN bids b ON (j.win_bid_uuid = b.uuid)
	SET p.miner_paid = NOW()
	WHERE b.miner_uuid = $1 AND
		p.miner_paid IS NULL AND
		j.active = false
	`
	if _, err := db.Exec(sqlStmt, mUUID); err != nil {
		message := "error updating miner payments"
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
		return err
	}
	return nil
}
