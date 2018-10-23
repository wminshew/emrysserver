package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// SetMinerConfirmed sets miner mUUID as confirmed
func SetMinerConfirmed(r *http.Request, mUUID uuid.UUID) error {
	sqlStmt := `
	UPDATE miners
	SET confirmed = true
	WHERE miner_uuid = $1
	`
	if _, err := db.Exec(sqlStmt, mUUID); err != nil {
		message := "error updating miner confirmed"
		pqErr, ok := err.(*pq.Error)
		if ok {
			log.Sugar.Errorw(message,
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"mID", mUUID,
				"pq_sev", pqErr.Severity,
				"pq_code", pqErr.Code,
				"pq_detail", pqErr.Detail,
			)
		} else {
			log.Sugar.Errorw(message,
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"mID", mUUID,
			)
		}
		return err
	}

	return nil
}
