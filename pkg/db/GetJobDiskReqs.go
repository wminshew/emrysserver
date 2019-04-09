package db

import (
	"database/sql"
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
)

// GetJobDiskReqs returns the job disk requirements
func GetJobDiskReqs(jUUID uuid.UUID) (int64, error) {
	var disk sql.NullInt64 // TODO: uint64
	sqlStmt := `
	SELECT disk
	FROM requirements
	WHERE job_uuid = $1
	`
	if err := db.QueryRow(sqlStmt, jUUID).Scan(&disk); err != nil {
		message := "error querying for jobs.active"
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
		return 0, err
	}

	if disk.Valid {
		return disk.Int64, nil
	}
	return 0, nil
}
