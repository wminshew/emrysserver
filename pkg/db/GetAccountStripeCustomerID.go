package db

import (
	"database/sql"
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// GetAccountStripeCustomerID gets the account aUUID's stripe customer id
func GetAccountStripeCustomerID(r *http.Request, aUUID uuid.UUID) (string, error) {
	var stripeCustomerID sql.NullString
	sqlStmt := `
	SELECT a.stripe_customer_id
	FROM accounts a
	WHERE a.uuid = $1
	`
	if err := db.QueryRow(sqlStmt, aUUID).Scan(&stripeCustomerID); err != nil {
		message := "error querying stripe customer ID"
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
	if stripeCustomerID.Valid {
		return stripeCustomerID.String, nil
	}
	return "", nil
}
