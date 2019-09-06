package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
)

// SetAccountCredit sets the account aUUID's credit
func SetAccountCredit(aUUID uuid.UUID, newCredit int64) error {
	sqlStmt := `
	UPDATE accounts
	SET credit = $2
	WHERE uuid = $1
	`
	if _, err := db.Exec(sqlStmt, aUUID, newCredit); err != nil {
		message := "error querying account credit"
		pqErr, ok := err.(*pq.Error)
		if ok {
			log.Sugar.Errorw(message,
				"err", err.Error(),
				"pq_sev", pqErr.Severity,
				"pq_code", pqErr.Code,
				"pq_msg", pqErr.Message,
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
