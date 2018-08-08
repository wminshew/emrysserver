package db

import (
	"github.com/lib/pq"
	"github.com/wminshew/emrys/pkg/job"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// InsertBid inserts a new bid into the db
func InsertBid(r *http.Request, b *job.Bid) error {
	sqlStmt := `
	INSERT INTO bids (bid_uuid, job_uuid, miner_uuid, min_rate, late)
	VALUES ($1, $2, $3, $4, $5)
	`
	if _, err := db.Exec(sqlStmt, b.ID, b.JobID, b.MinerID, b.MinRate, b.Late); err != nil {
		message := "failed to insert bid"
		pqErr, ok := err.(*pq.Error)
		if ok {
			log.Sugar.Errorw(message,
				"url", r.URL,
				"err", err.Error(),
				"jID", b.JobID,
				"pq_sev", pqErr.Severity,
				"pq_code", pqErr.Code,
				"pq_detail", pqErr.Detail,
			)
		} else {
			log.Sugar.Errorw(message,
				"url", r.URL,
				"err", err.Error(),
				"jID", b.JobID,
			)
		}
		return err
	}
	return nil
}
