package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// SetAccountStripeCustomerID sets the account aUUID's stripe's customer ID
func SetAccountStripeCustomerID(r *http.Request, aUUID uuid.UUID, stripeCustomerID string) error {
	sqlStmt := `
	UPDATE accounts
	SET stripe_customer_id = $2
	WHERE uuid = $1
	`
	if _, err := db.Exec(sqlStmt, aUUID, stripeCustomerID); err != nil {
		message := "error updating account stripe customer ID"
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
		return err
	}

	return nil
}
