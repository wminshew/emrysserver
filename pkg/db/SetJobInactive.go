package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
	// "net/http"
)

// SetJobInactive sets job jUUID in database to active=false
func SetJobInactive(jUUID uuid.UUID) error {
	sqlStmt := `
	UPDATE jobs
	SET active = false
	WHERE uuid = $1
	`
	if _, err := db.Exec(sqlStmt, jUUID); err != nil {
		message := "error updating job"
		pqErr, ok := err.(*pq.Error)
		if ok {
			log.Sugar.Errorw(message,
				// "method", r.Method,
				// "url", r.URL,
				"err", err.Error(),
				"jID", jUUID,
				"pq_sev", pqErr.Severity,
				"pq_code", pqErr.Code,
				"pq_detail", pqErr.Detail,
			)
		} else {
			log.Sugar.Errorw(message,
				// "method", r.Method,
				// "url", r.URL,
				"err", err.Error(),
				"jID", jUUID,
			)
		}
		return err
	}
	return nil
}
