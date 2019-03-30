package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// SetAccountStripeID sets the account aUUID's stripe id
func SetAccountStripeID(r *http.Request, aUUID uuid.UUID, stripeID string) error {
	sqlStmt := `
	UPDATE accounts
	SET stripe_id = $2
	WHERE uuid = $1
	`
	if _, err := db.Exec(sqlStmt, aUUID, stripeID); err != nil {
		message := "error updating account stripe ID"
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
