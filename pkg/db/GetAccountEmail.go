package db

import (
	"database/sql"
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// GetAccountEmail returns account email given the account uuid
func GetAccountEmail(r *http.Request, aUUID uuid.UUID) (string, error) {
	var email sql.NullString
	sqlStmt := `
	SELECT email
	FROM accounts
	WHERE uuid = $1
	`
	if err := db.QueryRow(sqlStmt, aUUID).Scan(&email); err != nil {
		message := "error querying for account email"
		pqErr, ok := err.(*pq.Error)
		if ok {
			log.Sugar.Errorw(message,
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
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
			)
		}
		return "", err
	}
	if email.Valid {
		return email.String, nil
	}
	return "", nil
}
