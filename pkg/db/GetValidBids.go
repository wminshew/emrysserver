package db

import (
	"database/sql"
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// GetValidBids returns the valid bids for job jUUID
func GetValidBids(r *http.Request, jUUID uuid.UUID) (*sql.Rows, error) {
	sqlStmt := `
	SELECT b1.bid_uuid, b1.bid_rate
	FROM bids b1
	WHERE b1.job_uuid = $1
		AND b1.late = false
		AND NOT EXISTS(SELECT 1
			FROM bids b2
			INNER JOIN jobs j ON (b2.bid_uuid = j.win_bid_uuid)
			WHERE j.active = true 
				AND b2.device_uuid = b1.device_uuid
				AND b2.miner_uuid = b2.miner_uuid
		)
	ORDER BY
		b1.bid_rate ASC
		b1.created_at ASC
	LIMIT 2
	`
	rows, err := db.Query(sqlStmt, jUUID)
	if err != nil {
		message := "error querying for valid bids"
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
	}
	return rows, err
}
