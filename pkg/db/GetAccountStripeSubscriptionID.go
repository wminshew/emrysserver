package db

import (
	"database/sql"
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// GetAccountStripeSubscriptionID gets the account aUUID's stripe subscription id
func GetAccountStripeSubscriptionID(r *http.Request, aUUID uuid.UUID) (string, error) {
	var stripeSubscriptionID sql.NullString
	sqlStmt := `
	SELECT a.stripe_subscription_id
	FROM accounts a
	WHERE a.uuid = $1
	`
	if err := db.QueryRow(sqlStmt, aUUID).Scan(&stripeSubscriptionID); err != nil {
		message := "error querying stripe subscription ID"
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
	if stripeSubscriptionID.Valid {
		return stripeSubscriptionID.String, nil
	}
	return "", nil
}
