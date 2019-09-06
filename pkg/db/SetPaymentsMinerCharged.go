package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
)

// SetPaymentsMinerCharged sets payments miner charged
func SetPaymentsMinerCharged(jUUID uuid.UUID, chargeID string, jobAmount int64) error {
	sqlStmt := `
		UPDATE payments
		SET miner_charged_at = NOW(),
		miner_charged_id = $2,
		miner_charged_amt = $3
		WHERE job_uuid = $1
		`
	if _, err := db.Exec(sqlStmt, jUUID, chargeID, jobAmount); err != nil {
		message := "error updating payments miner charged"
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
