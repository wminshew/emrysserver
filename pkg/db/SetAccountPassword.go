package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// SetAccountPassword sets account aUUID's password
func SetAccountPassword(r *http.Request, aUUID uuid.UUID, hashedPassword string) error {
	sqlStmt := `
	UPDATE accounts
	SET password = $1
	WHERE uuid = $2
	`
	if _, err := db.Exec(sqlStmt, hashedPassword, aUUID); err != nil {
		message := "error updating account password"
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
