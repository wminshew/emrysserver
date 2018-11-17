package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// GetAccountEmail returns account email given the account uuid
func GetAccountEmail(r *http.Request, aUUID uuid.UUID) (string, error) {
	var email *string
	sqlStmt := `
	SELECT a.email
	FROM accounts a
	WHERE a.uuid=$1
	`
	if err := db.QueryRow(sqlStmt, aUUID).Scan(email); err != nil {
		message := "error querying for email"
		pqErr, ok := err.(*pq.Error)
		if ok {
			log.Sugar.Errorw(message,
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"email", email,
				"pq_sev", pqErr.Severity,
				"pq_code", pqErr.Code,
				"pq_detail", pqErr.Detail,
			)
		} else {
			log.Sugar.Errorw(message,
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"email", email,
			)
		}
		return "", err
	}
	return *email, nil
}
