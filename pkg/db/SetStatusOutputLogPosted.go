package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// SetStatusOutputLogPosted sets job jUUID status in database to image_built=true
func SetStatusOutputLogPosted(r *http.Request, jUUID uuid.UUID) *app.Error {
	sqlStmt := `
	UPDATE statuses
	SET output_log_posted = $1
	WHERE job_uuid = $2
	`
	if _, err := db.Exec(sqlStmt, true, jUUID); err != nil {
		message := "failed to update job status to output log posted"
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
		if err := SetJobInactive(r, jUUID); err != nil {
			log.Sugar.Errorf("Error setting job %v inactive: %v\n", jUUID, err)
		}
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	return nil
}
