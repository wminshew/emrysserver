package db

import (
	"database/sql"
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// GetAccountStripeCustomerLast4 gets the account aUUID's stripe customer card's last 4 digits
func GetAccountStripeCustomerLast4(r *http.Request, aUUID uuid.UUID) (string, error) {
	var stripeCustomerLast4 sql.NullString
	sqlStmt := `
	SELECT a.stripe_customer_last4
	FROM accounts a
	WHERE a.uuid = $1
	`
	if err := db.QueryRow(sqlStmt, aUUID).Scan(&stripeCustomerLast4); err != nil {
		message := "error querying stripe customer card last 4 digits"
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
	if stripeCustomerLast4.Valid {
		return stripeCustomerLast4.String, nil
	}
	return "", nil
}
