package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
	"time"
)

// GetStatusDataDownloaded gets status data_downloaded for job jUUID
func GetStatusDataDownloaded(r *http.Request, jUUID uuid.UUID) (time.Time, *app.Error) {
	t := time.Time{}
	sqlStmt := `
	SELECT data_downloaded
	FROM statuses
	WHERE job_uuid = $1
	`
	if _, err := db.QueryRow(sqlStmt, jUUID).Scan(&t); err != nil {
		message := "error querying data_downloaded"
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
		return time.Time{}, err
	}

	return t, nil
}
