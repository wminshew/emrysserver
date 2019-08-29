package db

import (
	"database/sql"
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
)

// GetJobDiskQuota returns the winning bid's disk quota
func GetJobDiskQuota(jUUID uuid.UUID) (int64, error) {
	var disk sql.NullInt64 // TODO: uint64
	sqlStmt := `
	SELECT b.disk
	FROM bids b
	INNER JOIN jobs j ON (j.win_bid_uuid = b.uuid)
	WHERE j.uuid = $1
	`
	if err := db.QueryRow(sqlStmt, jUUID).Scan(&disk); err != nil {
		message := "error querying for jobs.active"
		pqErr, ok := err.(*pq.Error)
		if ok {
			log.Sugar.Errorw(message,
				"err", err.Error(),
				"jID", jUUID,
				"pq_sev", pqErr.Severity,
				"pq_code", pqErr.Code,
				"pq_name", pqErr.Name,
				"pq_detail", pqErr.Detail,
			)
		} else {
			log.Sugar.Errorw(message,
				"err", err.Error(),
				"jID", jUUID,
			)
		}
		return 0, err
	}

	if disk.Valid {
		return disk.Int64, nil
	}
	return 0, nil
}
