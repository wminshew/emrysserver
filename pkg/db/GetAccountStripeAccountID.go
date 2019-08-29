package db

import (
	"database/sql"
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
)

// GetAccountStripeAccountID gets the account aUUID's stripe account id
func GetAccountStripeAccountID(aUUID uuid.UUID) (string, error) {
	var stripeAccountID sql.NullString
	sqlStmt := `
	SELECT a.stripe_account_id
	FROM accounts a
	WHERE a.uuid = $1
	`
	if err := db.QueryRow(sqlStmt, aUUID).Scan(&stripeAccountID); err != nil {
		message := "error querying stripe account ID"
		pqErr, ok := err.(*pq.Error)
		if ok {
			log.Sugar.Errorw(message,
				"err", err.Error(),
				"pq_sev", pqErr.Severity,
				"pq_code", pqErr.Code,
				"pq_name", pqErr.Name,
				"pq_detail", pqErr.Detail,
			)
		} else {
			log.Sugar.Errorw(message,
				"err", err.Error(),
			)
		}
		return "", err
	}
	if stripeAccountID.Valid {
		return stripeAccountID.String, nil
	}
	return "", nil
}
