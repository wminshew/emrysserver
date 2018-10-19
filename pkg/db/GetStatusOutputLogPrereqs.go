package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
	"time"
)

// GetStatusOutputLogPrereqs gets status auction_completed for job jUUID
func GetStatusOutputLogPrereqs(r *http.Request, jUUID uuid.UUID) (time.Time, time.Time, error) {
	tDataDownloaded := time.Time{}
	tImageDownloaded := time.Time{}
	sqlStmt := `
	SELECT (data_downloaded, image_downloaded)
	FROM statuses
	WHERE job_uuid = $1
	`
	if err := db.QueryRow(sqlStmt, jUUID).Scan(&tDataDownloaded, &tImageDownloaded); err != nil {
		message := "error querying data_downloaded and image_downloaded"
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
		return time.Time{}, time.Time{}, err
	}

	return tDataDownloaded, tImageDownloaded, nil
}
