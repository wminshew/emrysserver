package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// SetStatusImageBuilt sets job jUUID status in database to image_built=true
func SetStatusImageBuilt(r *http.Request, jUUID uuid.UUID) *app.Error {
	sqlStmt := `
	UPDATE statuses
	SET (image_built) = ($1)
	WHERE job_uuid = $2
	`
	if _, err := Db.Exec(sqlStmt, true, jUUID); err != nil {
		pqErr := err.(*pq.Error)
		log.Sugar.Errorw("failed to update job status",
			"url", r.URL,
			"err", err.Error(),
			"jID", jUUID,
			"pq_sev", pqErr.Severity,
			"pq_code", pqErr.Code,
			"pq_detail", pqErr.Detail,
		)
		_ = SetJobInactive(r, jUUID)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	return nil
}
