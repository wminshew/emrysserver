package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
)

const errCheckViolation = "23514"

// SetJobFailed sets job failed_at and active=false for job jUUID
func SetJobFailed(jUUID uuid.UUID) error {
	// TODO: add miner balance debit penalty
	sqlStmt := `
	UPDATE jobs
	SET (failed_at, active)= (NOW(), false)
	WHERE uuid = $1 AND
		failed_at IS NULL
	`
	if _, err := db.Exec(sqlStmt, jUUID); err != nil {
		message := "error updating job failed_at"
		pqErr, ok := err.(*pq.Error)
		if ok {
			if pqErr.Code == errCheckViolation {
				log.Sugar.Infow("Job already ended, not updating failed_at",
					"jID", jUUID,
				)
				return nil
			}
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
