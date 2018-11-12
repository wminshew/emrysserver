package db

import (
	"database/sql"
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// InsertJob inserts a new job, status, and payment into the db
func InsertJob(r *http.Request, uUUID uuid.UUID, project string, jUUID uuid.UUID) *app.Error {
	ctx := r.Context()
	tx, txerr := db.BeginTx(ctx, nil)
	if message, err := func() (string, error) {
		if txerr != nil {
			return errBeginTx, txerr
		}
		pUUID := uuid.UUID{}
		sqlStmt := `
	SELECT uuid
	FROM projects
	WHERE (name, user_uuid) = ($1, $2)
	`
		if err := tx.QueryRow(sqlStmt, project, uUUID).Scan(&pUUID); err != nil {
			if err == sql.ErrNoRows {
				pUUID = uuid.NewV4()
				sqlStmt = `
	INSERT INTO projects (uuid, name, user_uuid)
	VALUES ($1, $2, $3)
	`
				if _, err := tx.Exec(sqlStmt, pUUID, project, uUUID); err != nil {
					return "error inserting project", err
				}

			} else {
				return "error finding project", err
			}
		}

		sqlStmt = `
	INSERT INTO jobs (uuid, project_uuid, active)
	VALUES ($1, $2, true)
	`
		if _, err := tx.Exec(sqlStmt, jUUID, pUUID); err != nil {
			return "error inserting job", err
		}

		sqlStmt = `
	INSERT INTO payments (job_uuid)
	VALUES ($1)
	`
		if _, err := tx.Exec(sqlStmt, jUUID); err != nil {
			return "error inserting payment", err
		}
		sqlStmt = `
	INSERT INTO statuses (job_uuid)
	VALUES ($1)
	`
		if _, err := tx.Exec(sqlStmt, jUUID); err != nil {
			return "error inserting status", err
		}

		if err := tx.Commit(); err != nil {
			return errCommitTx, err
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
