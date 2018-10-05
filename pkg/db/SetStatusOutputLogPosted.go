package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// SetStatusOutputLogPosted sets job jUUID status in database to output_log_posted=NOW()
func SetStatusOutputLogPosted(r *http.Request, jUUID uuid.UUID) *app.Error {
	sqlStmt := `
	UPDATE statuses
	SET output_log_posted = NOW()
	WHERE job_uuid = $1
	`
	if _, err := db.Exec(sqlStmt, jUUID); err != nil {
		message := "error updating job status to output log posted"
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
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	return nil
}
