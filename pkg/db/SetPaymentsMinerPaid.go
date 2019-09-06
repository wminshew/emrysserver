package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
)

// SetPaymentsMinerPaid sets payments miner paid
func SetPaymentsMinerPaid(jUUID uuid.UUID, transferID string, jobAmount int64) error {
	sqlStmt := `
		UPDATE payments
		SET miner_paid_at = NOW(),
		miner_paid_id = $2,
		miner_paid_amt = $3
		WHERE job_uuid = $1
		`
	if _, err := db.Exec(sqlStmt, jUUID, transferID, jobAmount); err != nil {
		message := "error updating payments miner paid"
		pqErr, ok := err.(*pq.Error)
		if ok {
			log.Sugar.Errorw(message,
				"err", err.Error(),
				"jID", jUUID,
				"pq_sev", pqErr.Severity,
				"pq_code", pqErr.Code,
				"pq_msg", pqErr.Message,
				"pq_detail", pqErr.Detail,
			)
		} else {
			log.Sugar.Errorw(message,
				"err", err.Error(),
				"jID", jUUID,
			)
		}
		return err
	}

	return nil
}
