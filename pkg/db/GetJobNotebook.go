package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
)

// GetJobNotebook returns whether job is a notebook
func GetJobNotebook(jUUID uuid.UUID) (bool, error) {
	var notebook bool
	sqlStmt := `
	SELECT notebook
	FROM jobs
	WHERE uuid = $1
	`
	if err := db.QueryRow(sqlStmt, jUUID).Scan(&notebook); err != nil {
		message := "error querying for jobs.notebook"
		pqErr, ok := err.(*pq.Error)
		if ok {
			log.Sugar.Errorw(message,
				"err", err.Error(),
				"jID", jUUID,
				"pq_sev", pqErr.Severity,
				"pq_code", pqErr.Code,
				"pq_msg", pqErr.Message,
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
	return notebook, nil
}
