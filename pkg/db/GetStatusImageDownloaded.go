package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
	"time"
)

// GetStatusImageDownloaded gets status image_downloaded for job jUUID
func GetStatusImageDownloaded(r *http.Request, jUUID uuid.UUID) (time.Time, error) {
	t := pq.NullTime{}
	sqlStmt := `
	SELECT image_downloaded
	FROM statuses
	WHERE job_uuid = $1
	`
	if err := db.QueryRow(sqlStmt, jUUID).Scan(&t); err != nil {
		message := "error querying image_downloaded"
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

	if t.Valid {
		return t.Time, nil
	}
	return time.Time{}, nil
}
