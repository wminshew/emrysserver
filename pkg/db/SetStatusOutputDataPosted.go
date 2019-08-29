package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
)

// SetStatusOutputDataPosted sets job jUUID status in database to output_data_posted=NOW()
func SetStatusOutputDataPosted(jUUID uuid.UUID) error {
	sqlStmt := `
	UPDATE statuses
	SET output_data_posted = NOW()
	WHERE job_uuid = $1 AND
		output_data_posted IS NULL
	`
	if _, err := db.Exec(sqlStmt, jUUID); err != nil {
		message := "error updating job status to output data posted"
		pqErr, ok := err.(*pq.Error)
		if ok {
			log.Sugar.Errorw(message,
				"err", err.Error(),
				"jID", jUUID,
				"pq_sev", pqErr.Severity,
				"pq_code", pqErr.Code,
				"pq_name", pqErr.Name,
				"pq_detail", pqErr.Detail,
			)
		} else {
			log.Sugar.Errorw(message,
				"err", err.Error(),
				"jID", jUUID,
			)
		}
		return err
	}

	return nil
}
