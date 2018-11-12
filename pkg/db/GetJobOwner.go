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
	SELECT p.user_uuid
	FROM projects p
	INNER JOIN jobs j ON (j.project_uuid = p.uuid)
	WHERE j.uuid = $1
	`
	if err := db.QueryRow(sqlStmt, jUUID).Scan(&uUUID); err != nil {
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
		return uuid.UUID{}, err
	}
	return uUUID, nil
}
