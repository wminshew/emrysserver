package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// SetUserConfirmed sets user uUUID as confirmed
func SetUserConfirmed(r *http.Request, uUUID uuid.UUID) error {
	sqlStmt := `
	UPDATE users
	SET confirmed = true
	WHERE user_uuid = $1
	`
	if _, err := db.Exec(sqlStmt, uUUID); err != nil {
		message := "error updating user confirmed"
		pqErr, ok := err.(*pq.Error)
		if ok {
			log.Sugar.Errorw(message,
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"uID", uUUID,
				"pq_sev", pqErr.Severity,
				"pq_code", pqErr.Code,
				"pq_detail", pqErr.Detail,
			)
		} else {
			log.Sugar.Errorw(message,
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"uID", uUUID,
			)
		}
		return err
	}

	return nil
}
