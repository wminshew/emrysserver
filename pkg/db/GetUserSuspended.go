package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// GetUserSuspended returns whether user is suspended
func GetUserSuspended(r *http.Request, uUUID uuid.UUID) (bool, error) {
	var suspended bool
	sqlStmt := `
	SELECT suspended
	FROM users
	WHERE user_uuid = $1
	`
	if err := db.QueryRow(sqlStmt, uUUID).Scan(&suspended); err != nil {
		message := "error querying for user suspended"
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
		return false, err
	}
	return suspended, nil
}
