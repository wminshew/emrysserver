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
			return "failed to begin tx", txerr
		}
		pUUID := uuid.UUID{}
		sqlStmt := `
	SELECT project_uuid
	FROM projects
	WHERE (project_name, user_uuid) = ($1, $2)
	`
		if err := tx.QueryRow(sqlStmt, project, uUUID).Scan(&pUUID); err != nil {
			if err == sql.ErrNoRows {
				pUUID = uuid.NewV4()
				sqlStmt = `
	INSERT INTO projects (project_uuid, project_name, user_uuid)
	VALUES ($1, $2, $3)
	`
				if _, err := tx.Exec(sqlStmt, pUUID, project, uUUID); err != nil {
					return "failed to insert project", err
				}

			} else {
				return "failed to find project", err
			}
		}

		sqlStmt = `
	INSERT INTO jobs (job_uuid, project_uuid, active)
	VALUES ($1, $2, $3)
	`
		if _, err := tx.Exec(sqlStmt, jUUID, pUUID, true); err != nil {
			return "failed to insert job", err
		}

		sqlStmt = `
	INSERT INTO payments (job_uuid, user_paid, miner_paid)
	VALUES ($1, $2, $3)
	`
		if _, err := tx.Exec(sqlStmt, jUUID, false, false); err != nil {
			return "failed to insert payment", err
		}
		sqlStmt = `
	INSERT INTO statuses (job_uuid, user_data_stored,
	image_built, auction_completed,
	image_downloaded, data_downloaded,
	output_log_posted, output_dir_posted)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
		if _, err := tx.Exec(sqlStmt, jUUID, false, false, false, false, false, false, false); err != nil {
			return "failed to insert status", err
		}

		if err := tx.Commit(); err != nil {
			return "failed to commit tx", err
		}

		return "", nil
	}(); err != nil {
		pqErr, ok := err.(*pq.Error)
		if ok {
			log.Sugar.Errorw(message,
				"url", r.URL,
				"err", err.Error(),
				"jID", jUUID,
				"pq_sev", pqErr.Severity,
				"pq_code", pqErr.Code,
				"pq_detail", pqErr.Detail,
			)
		} else {
			log.Sugar.Errorw(message,
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
