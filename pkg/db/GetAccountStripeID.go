package db

import (
	"database/sql"
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// GetAccountStripeID gets the account aUUID's stripe id
func GetAccountStripeID(r *http.Request, aUUID uuid.UUID) (string, error) {
	var stripeID sql.NullString
	sqlStmt := `
	SELECT a.stripe_id
	FROM accounts a
	WHERE a.uuid = $1
	`
	if err := db.QueryRow(sqlStmt, aUUID).Scan(&stripeID); err != nil {
		message := "error querying account stripe ID"
		pqErr, ok := err.(*pq.Error)
		if ok {
			log.Sugar.Errorw(message,
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"pq_sev", pqErr.Severity,
				"pq_code", pqErr.Code,
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
	if stripeID.Valid {
		return stripeID.String, nil
	}
	return "", nil
}
