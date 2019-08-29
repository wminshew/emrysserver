package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
	"time"
)

// GetStatusOutputDataPrereqs gets status auction_completed for job jUUID
func GetStatusOutputDataPrereqs(r *http.Request, jUUID uuid.UUID) (time.Time, time.Time, time.Time, error) {
	tDataDownloaded := pq.NullTime{}
	tImageDownloaded := pq.NullTime{}
	tOutputLogPosted := pq.NullTime{}
	sqlStmt := `
	SELECT data_downloaded, image_downloaded, output_log_posted
	FROM statuses
	WHERE job_uuid = $1
	`
	if err := db.QueryRow(sqlStmt, jUUID).Scan(&tDataDownloaded, &tImageDownloaded, &tOutputLogPosted); err != nil {
		message := "error querying output data prereqs"
		pqErr, ok := err.(*pq.Error)
		if ok {
			log.Sugar.Errorw(message,
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"jID", jUUID,
				"pq_sev", pqErr.Severity,
				"pq_code", pqErr.Code,
				"pq_name", pqErr.Name,
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
		return time.Time{}, time.Time{}, time.Time{}, err
	}

	tDataReturn := time.Time{}
	if tDataDownloaded.Valid {
		tDataReturn = tDataDownloaded.Time
	}
	tImageReturn := time.Time{}
	if tImageDownloaded.Valid {
		tImageReturn = tImageDownloaded.Time
	}
	tOutputReturn := time.Time{}
	if tOutputLogPosted.Valid {
		tOutputReturn = tOutputLogPosted.Time
	}
	return tDataReturn, tImageReturn, tOutputReturn, nil
}
