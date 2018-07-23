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
	SELECT b1.bid_uuid, b1.min_rate
	FROM bids b1
	WHERE b1.job_uuid = $1
		AND b1.late = false
		AND NOT EXISTS(SELECT 1
			FROM bids b2
			INNER JOIN jobs j ON (b2.bid_uuid = j.win_bid_uuid)
			WHERE b2.miner_uuid = b1.miner_uuid
				AND j.active = true
		)
	ORDER BY b1.min_rate
	LIMIT 2
	`
	rows, err := Db.Query(sqlStmt, jUUID)
	if err != nil {
		pqErr := err.(*pq.Error)
		log.Sugar.Errorw("failed to query for valid bids",
			"url", r.URL,
			"err", err.Error(),
			"jID", jUUID.String(),
			"pq_sev", pqErr.Severity,
			"pq_code", pqErr.Code,
			"pq_detail", pqErr.Detail,
		)
	}
	return rows, err
}
