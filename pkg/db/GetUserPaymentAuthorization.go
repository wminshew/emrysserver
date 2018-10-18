package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// GetUserPaymentAuthorization returns whether user is authorized for payments
func GetUserPaymentAuthorization(r *http.Request, uUUID uuid.UUID) (bool, error) {
	var authorized bool
	// TODO: name the db column better..
	sqlStmt := `
	SELECT authorized_payments
	FROM users
	WHERE user_uuid = $1
	`
	if err := db.QueryRow(sqlStmt, uUUID).Scan(&authorized); err != nil {
		message := "error querying for user payment authorization"
		pqErr, ok := err.(*pq.Error)
		if ok {
			log.Sugar.Errorw(message,
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"jID", jUUID,
				"pq_sev", pqErr.Severity,
				"pq_code", pqErr.Code,
				"pq_detail", pqErr.Detail,
			)
		} else {
			log.Sugar.Errorw(message,
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"jID", jUUID,
			)
		}
		return false, err
	}
	return authorization, nil
}
