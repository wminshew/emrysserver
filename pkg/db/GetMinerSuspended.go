package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// GetMinerSuspended returns whether miner is suspended
func GetMinerSuspended(r *http.Request, mUUID uuid.UUID) (bool, error) {
	var suspended bool
	sqlStmt := `
	SELECT suspended
	FROM miners
	WHERE miner_uuid = $1
	`
	if err := db.QueryRow(sqlStmt, mUUID).Scan(&suspended); err != nil {
		message := "error querying for miner suspended"
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
		return false, err
	}
	return suspended, nil
}
