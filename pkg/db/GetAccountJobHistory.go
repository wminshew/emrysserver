package db

import (
	"database/sql"
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
)

// GetAccountJobHistory returns rows holding jobs related to account aUUID
func GetAccountJobHistory(aUUID uuid.UUID) (*sql.Rows, error) {
	sqlStmt := `
	SELECT j.uuid
	FROM jobs j, projects proj, users u
	WHERE u.uuid = $1 AND
		proj.user_uuid = u.uuid AND
		j.project_uuid = proj.uuid
	`
	rows, err := db.Query(sqlStmt, aUUID)
	if err != nil {
		message := "error querying for account job history"
		pqErr, ok := err.(*pq.Error)
		if ok {
			log.Sugar.Errorw(message,
				"err", err.Error(),
				"pq_sev", pqErr.Severity,
				"pq_code", pqErr.Code,
				"pq_detail", pqErr.Detail,
			)
		} else {
			log.Sugar.Errorw(message,
				"err", err.Error(),
			)
		}
	}
	return rows, err
}
