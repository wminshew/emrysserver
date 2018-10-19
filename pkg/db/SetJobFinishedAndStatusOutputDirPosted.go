package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// SetJobFinishedAndStatusOutputDataPosted sets job completed (and inactive) and status for job jUUID
func SetJobFinishedAndStatusOutputDataPosted(r *http.Request, jUUID uuid.UUID) *app.Error {
	ctx := r.Context()
	tx, txerr := db.BeginTx(ctx, nil)
	if message, err := func() (string, error) {
		if txerr != nil {
			return "error beginning tx", txerr
		}

		sqlStmt := `
	UPDATE jobs
	SET (completed_at, active) = (NOW(), false)
	WHERE job_uuid = $1 AND
		completed_at IS NULL AND
		canceled_at IS NULL
	`
		if _, err := tx.Exec(sqlStmt, jUUID); err != nil {
			return "error updating job winner", err
		}

		sqlStmt = `
	UPDATE statuses
	SET output_data_posted = NOW()
	WHERE job_uuid = $1 AND
		output_data_posted IS NULL
	`
		if _, err := tx.Exec(sqlStmt, jUUID); err != nil {
			return "error updating job status", err
		}

		if err := tx.Commit(); err != nil {
			return "error committing tx", err
		}

		return "", nil
	}(); err != nil {
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
		if txerr == nil {
			if err := tx.Rollback(); err != nil {
				log.Sugar.Errorf("Error rolling tx back job %v: %v\n", jUUID, err)
			}
		}
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}
	return nil
}
