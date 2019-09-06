package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
)

// GetAccountCredit gets the account aUUID's credit
func GetAccountCredit(aUUID uuid.UUID) (int64, error) {
	var credit int64
	sqlStmt := `
	SELECT credit
	FROM accounts
	WHERE uuid = $1
	`
	if err := db.QueryRow(sqlStmt, aUUID).Scan(&credit); err != nil {
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
		return 0, err
	}
	return credit, nil
}
