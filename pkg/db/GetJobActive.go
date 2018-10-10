package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// GetJobActive returns whether job is active
func GetJobActive(r *http.Request, jUUID uuid.UUID) (bool, error) {
	var active bool
	sqlStmt := `
	SELECT j.active
	FROM jobs j
	WHERE j.job_uuid = $1
	`
	if err := db.QueryRow(sqlStmt, jUUID).Scan(&active); err != nil {
		message := "error querying for job owner"
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
		return false, err
	}
	return active, nil
}
