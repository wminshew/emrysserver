package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
	"time"
)

// GetStatusOutputData gets status output_data_posted for job jUUID
func GetStatusOutputData(r *http.Request, jUUID uuid.UUID) (time.Time, error) {
	tOutputDataPosted := pq.NullTime{}
	sqlStmt := `
	SELECT output_data_posted
	FROM statuses
	WHERE job_uuid = $1
	`
	if err := db.QueryRow(sqlStmt, jUUID).Scan(&tOutputDataPosted); err != nil {
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
				"pq_msg", pqErr.Message,
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

	tOutputReturn := time.Time{}
	if tOutputDataPosted.Valid {
		tOutputReturn = tOutputDataPosted.Time
	}
	return tOutputReturn, nil
}
