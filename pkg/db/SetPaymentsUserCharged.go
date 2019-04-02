package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
)

// SetPaymentsUserCharged sets payments user charged
func SetPaymentsUserCharged(jUUID uuid.UUID, invoiceID string, jobAmount int64) error {
	sqlStmt := `
		UPDATE payments
		SET user_charged_at = NOW(),
		user_charged_id = $2,
		user_charged_amt = $3
		WHERE job_uuid = $1
		`
	if _, err := db.Exec(sqlStmt, jUUID, invoiceID, jobAmount); err != nil {
		message := "error updating payments user charged"
		pqErr, ok := err.(*pq.Error)
		if ok {
			log.Sugar.Errorw(message,
				"err", err.Error(),
				"jID", jUUID,
				"pq_sev", pqErr.Severity,
				"pq_code", pqErr.Code,
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
