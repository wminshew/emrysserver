package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// SetAccountStripeSubscriptionID sets the account aUUID's stripe's subscription ID
func SetAccountStripeSubscriptionID(r *http.Request, aUUID uuid.UUID, stripeSubscriptionID string) error {
	sqlStmt := `
	UPDATE accounts
	SET stripe_subscription_id = $2
	WHERE uuid = $1
	`
	if _, err := db.Exec(sqlStmt, aUUID, stripeSubscriptionID); err != nil {
		message := "error updating account stripe subscription ID"
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
		return err
	}

	return nil
}
