package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// GetJobOwnerAndProject returns user uuid and project of job jUUID
func GetJobOwnerAndProject(r *http.Request, jUUID uuid.UUID) (uuid.UUID, string, error) {
	uUUID := uuid.UUID{}
	var project string
	sqlStmt := `
	SELECT (p.user_uuid, p.project_name)
	FROM projects p
	INNER JOIN jobs j ON (j.project_uuid = p.project_uuid)
	WHERE j.job_uuid = $1
	`
	if err := db.QueryRow(sqlStmt, jUUID).Scan(&uUUID, &project); err != nil {
		message := "failed to query for job owner and project"
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
		return uuid.UUID{}, "", err
	}
	return uUUID, project, nil
}
