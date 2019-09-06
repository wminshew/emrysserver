package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// GetAccountScope returns account scope given the account uuid
func GetAccountScope(r *http.Request, aUUID uuid.UUID) (bool, bool, error) {
	var isUser, isMiner bool
	sqlStmt := `
	SELECT exists(
		SELECT 1
		FROM users
		WHERE uuid = $1
	)
	`
	if err := db.QueryRow(sqlStmt, aUUID).Scan(&isUser); err != nil {
		message := "error querying for account as user"
		pqErr, ok := err.(*pq.Error)
		if ok {
			log.Sugar.Errorw(message,
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
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
			)
		}
		return false, false, err
	}

	sqlStmt = `
	SELECT exists(
		SELECT 1
		FROM miners
		WHERE uuid = $1
	)
	`
	if err := db.QueryRow(sqlStmt, aUUID).Scan(&isMiner); err != nil {
		message := "error querying for account as miner"
		pqErr, ok := err.(*pq.Error)
		if ok {
			log.Sugar.Errorw(message,
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
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
			)
		}
		return false, false, err
	}

	return isUser, isMiner, nil
}
