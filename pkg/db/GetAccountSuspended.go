package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// GetAccountSuspended returns whether account is suspended
func GetAccountSuspended(r *http.Request, aUUID uuid.UUID) (bool, error) {
	var suspended bool
	sqlStmt := `
	SELECT suspended
	FROM accounts
	WHERE uuid = $1
	`
	if err := db.QueryRow(sqlStmt, aUUID).Scan(&suspended); err != nil {
		message := "error querying for account suspended"
		pqErr, ok := err.(*pq.Error)
		if ok {
			log.Sugar.Errorw(message,
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"aID", aUUID,
				"pq_sev", pqErr.Severity,
				"pq_code", pqErr.Code,
				"pq_name", pqErr.Name,
				"pq_detail", pqErr.Detail,
			)
		} else {
			log.Sugar.Errorw(message,
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"aID", aUUID,
			)
		}
		return false, err
	}
	return suspended, nil
}
