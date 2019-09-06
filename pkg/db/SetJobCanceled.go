package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// SetJobCanceled sets job canceled_at and active=false for job jUUID
func SetJobCanceled(r *http.Request, jUUID uuid.UUID) error {
	sqlStmt := `
		UPDATE jobs j
		SET canceled_at = NOW(),
		active = false
		FROM users u, projects proj, miners m, bids b
		WHERE j.uuid = $1 AND
			j.canceled_at IS NULL AND
			proj.uuid = j.project_uuid AND
			u.uuid = proj.user_uuid AND (
			j.win_bid_uuid IS NULL OR (
			b.uuid = j.win_bid_uuid AND
			m.uuid = b.miner_uuid
			))
		`
	if _, err := db.Exec(sqlStmt, jUUID); err != nil {
		message := "error updating jobs canceled_at, active"
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
		return err
	}

	return nil
}
