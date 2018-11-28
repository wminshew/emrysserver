package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// GetAccountBalance updates the account aUUID's balance to newBalance
func GetAccountBalance(r *http.Request, aUUID uuid.UUID) (float64, error) {
	var balance float64
	sqlStmt := `
	SELECT a.balance
	FROM accounts a
	WHERE a.uuid = $1
	`
	if err := db.QueryRow(sqlStmt, aUUID).Scan(&balance); err != nil {
		message := "error querying account balance"
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
		return 0, err
	}
	return balance, nil
}
