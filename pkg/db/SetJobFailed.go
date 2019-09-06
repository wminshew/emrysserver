package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
)

// https://github.com/lib/pq/blob/master/error.go#L78
const errCheckViolation = "23514" // "check_violation"

// SetJobFailed sets job failed_at and active=false for job jUUID
func SetJobFailed(jUUID uuid.UUID) error {
	sqlStmt := `
		UPDATE jobs j
		SET failed_at = NOW(),
		active = false
		FROM miners m, bids b
		WHERE j.uuid = $1 AND
			j.failed_at IS NULL AND
			b.uuid = j.win_bid_uuid AND
			m.uuid = b.miner_uuid
		`
	if _, err := db.Exec(sqlStmt, jUUID); err != nil {
		message := "error updating jobs failed_at, active"
		pqErr, ok := err.(*pq.Error)
		if ok {
			if pqErr.Code == errCheckViolation {
				log.Sugar.Infow("Job already ended, not updating failed_at",
					"jID", jUUID,
				)
			} else {
				log.Sugar.Errorw(message,
					"err", err.Error(),
					"jID", jUUID,
					"pq_sev", pqErr.Severity,
					"pq_code", pqErr.Code,
				"pq_msg", pqErr.Message,
					"pq_detail", pqErr.Detail,
				)
			}
		} else {
			log.Sugar.Errorw(message,
				"err", err.Error(),
				"jID", jUUID,
			)
		}
		return err
	}

	return nil
}
