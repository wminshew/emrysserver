package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// SetAccountStripeCustomerLast4 sets the account aUUID's stripe's customer's card last 4 digits
func SetAccountStripeCustomerLast4(r *http.Request, aUUID uuid.UUID, stripeCardLast4 string) error {
	sqlStmt := `
	UPDATE accounts
	SET stripe_customer_last4 = $2
	WHERE uuid = $1
	`
	if _, err := db.Exec(sqlStmt, aUUID, stripeCardLast4); err != nil {
		message := "error updating account stripe customer card last4"
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
