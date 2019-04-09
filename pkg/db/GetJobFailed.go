package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
)

// GetJobFailed returns whether job failed
func GetJobFailed(jUUID uuid.UUID) (bool, error) {
	failedAt := pq.NullTime{}
	sqlStmt := `
	SELECT failed_at
	FROM jobs
	WHERE uuid = $1
	`
	if err := db.QueryRow(sqlStmt, jUUID).Scan(&failedAt); err != nil {
		message := "error querying for jobs.failed_at"
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
		return false, err
	}

	if failedAt.Valid {
		return true, nil
	}
	return false, nil
}
