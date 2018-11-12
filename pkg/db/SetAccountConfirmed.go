package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// SetAccountConfirmed sets user aUUID as confirmed
func SetAccountConfirmed(r *http.Request, aUUID uuid.UUID) error {
	sqlStmt := `
	UPDATE accounts
	SET confirmed = true
	WHERE uuid = $1
	`
	if _, err := db.Exec(sqlStmt, aUUID); err != nil {
		message := "error updating user confirmed"
		pqErr, ok := err.(*pq.Error)
		if ok {
			log.Sugar.Errorw(message,
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"aID", aUUID,
				"pq_sev", pqErr.Severity,
				"pq_code", pqErr.Code,
				"pq_detail", pqErr.Detail,
			)
		} else {
			log.Sugar.Errorw(message,
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"aID", aUUID,
			)
		}
		return err
	}

	return nil
}
