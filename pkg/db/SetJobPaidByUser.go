package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// SetJobPaidByUser sets user_paid for job jUUID
func SetJobPaidByUser(r *http.Request, jUUID uuid.UUID) error {
	sqlStmt := `
	UPDATE payments
	SET user_paid = NOW()
	WHERE job_uuid = $1 AND
		user_paid IS NULL
	`
	if _, err := db.Exec(sqlStmt, jUUID); err != nil {
		message := "error updating payments"
		pqErr, ok := err.(*pq.Error)
		if ok {
			log.Sugar.Errorw(message,
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"jID", jUUID,
				"pq_sev", pqErr.Severity,
				"pq_code", pqErr.Code,
				"pq_detail", pqErr.Detail,
			)
		} else {
			log.Sugar.Errorw(message,
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"jID", jUUID,
			)
		}
		return err
	}

	return nil
}
