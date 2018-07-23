package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// GetJobOwner returns user uuid of job jUUID owner
func GetJobOwner(r *http.Request, jUUID uuid.UUID) (uuid.UUID, error) {
	uUUID := uuid.UUID{}
	sqlStmt := `
	SELECT j.user_uuid
	FROM jobs
	WHERE j.job_uuid = $1
	`
	if err := Db.QueryRow(sqlStmt, jUUID).Scan(&uUUID); err != nil {
		pqErr := err.(*pq.Error)
		log.Sugar.Errorw("failed to query db",
			"url", r.URL,
			"err", err.Error(),
			"jID", jUUID.String(),
			"pq_sev", pqErr.Severity,
			"pq_code", pqErr.Code,
			"pq_detail", pqErr.Detail,
		)
		return uuid.UUID{}, err
	}
	return uUUID, nil
}
