package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// SetJobCanceled sets job canceled_at and active=falsefor job jUUID
func SetJobCanceled(r *http.Request, jUUID uuid.UUID) *app.Error {
	sqlStmt := `
	UPDATE jobs
	SET (canceled_at, active)= (NOW(), false)
	WHERE job_uuid = $1 AND
		canceled_at IS NULL AND
		completed_at IS NULL
	`
	if _, err := db.Exec(sqlStmt, jUUID); err != nil {
		message := "error canceling job"
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
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}
	return nil
}
