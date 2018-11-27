package db

import (
	"database/sql"
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
	"time"
)

// SetJobFinishedAndStatusOutputDataPostedAndDebitUser sets job completed
// (and inactive) and status for job jUUID and debits user accordingly
func SetJobFinishedAndStatusOutputDataPostedAndDebitUser(r *http.Request,
	jUUID uuid.UUID) *app.Error {
	ctx := r.Context()
	tx, txerr := db.BeginTx(ctx, nil)
	if message, err := func() (string, error) {
		if txerr != nil {
			return errBeginTx, txerr
		}

		uUUID := uuid.UUID{}
		createdAt := time.Time{}
		completedAt := pq.NullTime{}
		rate := sql.NullFloat64{}

		sqlStmt := `
		UPDATE jobs j
		SET j.completed_at = NOW(),
		j.active = false
		FROM accounts a, projects proj
		WHERE j.uuid = $1 AND
			j.completed_at IS NULL AND
			proj.uuid = j.project_uuid AND
			a.uuid = proj.user_uuid
		RETURNING a.uuid, j.created_at, j.completed_at, j.rate
		`
		if err := tx.QueryRow(sqlStmt, jUUID).Scan(&uUUID, &createdAt, &completedAt, &rate); err != nil {
			return "error updating jobs completed_at, active", err
		}

		log.Sugar.Infof("Completed at: %v\n", completedAt.Time) // TODO: rm
		if completedAt.Valid {
			sqlStmt = `
			UPDATE statuses s
			SET s.output_data_posted = $2
			WHERE s.job_uuid = $1 AND
				s.output_data_posted IS NULL
			`
			if _, err := db.Exec(sqlStmt, jUUID, completedAt.Time); err != nil {
				return "error updating statuses output_data_posted", err
			}

			sqlStmt = `
			UPDATE payments p
			SET p.user_paid = $2
			WHERE p.job_uuid = $1 AND
				p.user_paid IS NULL
			`
			if _, err := db.Exec(sqlStmt, jUUID, completedAt.Time); err != nil {
				return "error updating payments user_paid", err
			}

			log.Sugar.Infof("Rate: %v\n", rate) //TODO: rm
			if rate.Valid {
				amt := rate.Float64 * completedAt.Time.Sub(createdAt).Hours()
				sqlStmt = `
				UPDATE accounts a
				SET a.balance = balance - $2
				WHERE a.uuid = $1
				`
				if _, err := db.Exec(sqlStmt, uUUID, amt); err != nil {
					return "error updating accounts balance", err
				}

				// TODO: add miner credit
			}
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
				log.Sugar.Errorf("Error rolling tx back: %v", err)
			}
		}
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}
	return nil
}
