package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
)

// SetJobFailed sets job failed_at and active=falsefor job jUUID
func SetJobFailed(jUUID uuid.UUID) error {
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
