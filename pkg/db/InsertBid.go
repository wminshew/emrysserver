package db

import (
	"github.com/lib/pq"
	"github.com/wminshew/emrys/pkg/job"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// InsertBid inserts a new bid into the db
func InsertBid(r *http.Request, b *job.Bid, meetsReqs bool) error {
	sqlStmt := `
	INSERT INTO bids (uuid, job_uuid, miner_uuid, device_uuid, late, meets_requirements, rate, gpu, ram, disk, pcie)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	if _, err := db.Exec(sqlStmt, b.ID, b.JobID, b.MinerID, b.DeviceID, b.Late, meetsReqs,
		b.Specs.Rate, b.Specs.GPU, b.Specs.RAM, b.Specs.Disk, b.Specs.Pcie); err != nil {
		message := "error inserting bid"
		pqErr, ok := err.(*pq.Error)
		if ok {
			log.Sugar.Errorw(message,
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"jID", b.JobID,
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
				"jID", b.JobID,
			)
		}
		return err
	}
	return nil
}
