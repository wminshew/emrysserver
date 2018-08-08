package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// SetJobInactive sets job jUUID in database to active=false
func SetJobInactive(r *http.Request, jUUID uuid.UUID) error {
	sqlStmt := `
	UPDATE jobs
	SET active = $1
	WHERE job_uuid = $2
	`
	if _, err := db.Exec(sqlStmt, false, jUUID); err != nil {
		message := "failed to update job"
		pqErr, ok := err.(*pq.Error)
		if ok {
			log.Sugar.Errorw(message,
				"url", r.URL,
				"err", err.Error(),
				"jID", jUUID,
				"pq_sev", pqErr.Severity,
				"pq_code", pqErr.Code,
				"pq_detail", pqErr.Detail,
			)
		} else {
			log.Sugar.Errorw(message,
				"url", r.URL,
				"err", err.Error(),
				"jID", jUUID,
			)
		}
		return err
	}
	return nil
}
