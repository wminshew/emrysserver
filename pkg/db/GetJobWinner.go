package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// GetJobWinner returns the miner uuid of the winning bid for job jUUID
func GetJobWinner(r *http.Request, jUUID uuid.UUID) (uuid.UUID, error) {
	mUUID := uuid.UUID{}
	sqlStmt := `
	SELECT b.miner_uuid
	FROM bids b
	INNER JOIN jobs j ON (j.win_bid_uuid = b.bid_uuid)
	WHERE j.job_uuid = $1
	`
	if err := Db.QueryRow(sqlStmt, jUUID).Scan(&mUUID); err != nil {
		pqErr := err.(*pq.Error)
		log.Sugar.Errorw("failed to query db",
			"url", r.URL,
			"err", err.Error(),
			"jID", jUUID.String(),
			"pq_sev", pqErr.Severity,
			"pq_code", pqErr.Code,
			"pq_detail", pqErr.Detail,
		)
		return uuid.UUID{}, err
	}
	return mUUID, nil
}
