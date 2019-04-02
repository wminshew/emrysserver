package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
)

// SetAccountStripeAccountID sets the account aUUID's stripe id
func SetAccountStripeAccountID(aUUID uuid.UUID, stripeAccountID string) error {
	sqlStmt := `
	UPDATE accounts
	SET stripe_account_id = $2
	WHERE uuid = $1
	`
	if _, err := db.Exec(sqlStmt, aUUID, stripeAccountID); err != nil {
		message := "error updating account stripe account ID"
		pqErr, ok := err.(*pq.Error)
		if ok {
			log.Sugar.Errorw(message,
				"err", err.Error(),
				"pq_sev", pqErr.Severity,
				"pq_code", pqErr.Code,
				"pq_detail", pqErr.Detail,
			)
		} else {
			log.Sugar.Errorw(message,
				"err", err.Error(),
			)
		}
		return err
	}

	return nil
}
