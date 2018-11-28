package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
)

// SetAccountBalance updates the account aUUID's balance to newBalance
func SetAccountBalance(aUUID uuid.UUID, newBalance float64) error {
	sqlStmt := `
	UPDATE accounts
	SET balance = $2
	WHERE a.uuid = $1
	`
	_, err := db.Exec(sqlStmt, aUUID, newBalance)
	if err != nil {
		message := "error updating accounts balance"
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
	}
	return err
}
