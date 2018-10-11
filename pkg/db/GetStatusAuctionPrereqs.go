package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
	"time"
)

// GetStatusAuctionPrereqs gets status auction_completed for job jUUID
func GetStatusAuctionPrereqs(r *http.Request, jUUID uuid.UUID) (time.Time, time.Time, *app.Error) {
	tDataSynced := time.Time{}
	tImageBuilt := time.Time{}
	sqlStmt := `
	SELECT (data_synced, image_built)
	FROM statuses
	WHERE job_uuid = $1
	`
	if _, err := db.QueryRow(sqlStmt, jUUID).Scan(&tDataSynced, &tImageBuilt); err != nil {
		message := "error querying data_synced and image_built"
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

	return tDataSynced, tImageBuilt, nil
}
