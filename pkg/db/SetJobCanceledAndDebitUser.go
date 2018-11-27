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

// SetJobCanceledAndDebitUser sets job canceled_at and active=false for job jUUID
// and debits user balance accordingly
func SetJobCanceledAndDebitUser(r *http.Request, jUUID uuid.UUID) *app.Error {
	ctx := r.Context()
	tx, txerr := db.BeginTx(ctx, nil)
	if message, err := func() (string, error) {
		if txerr != nil {
			return errBeginTx, txerr
		}
		var uUUID uuid.UUID
		var createdAt, canceledAt time.Time
		rate := sql.NullFloat64{}
		sqlStmt := `
		UPDATE jobs j
		SET j.canceled_at = NOW(),
		j.active = false
		FROM accounts a, projects proj
		WHERE j.uuid = $1 AND
			j.canceled_at IS NULL AND
			proj.uuid = j.project_uuid AND
			a.uuid = proj.user_uuid
		RETURNING a.uuid, j.created_at, j.canceled_at, j.rate
		`
		if err := db.QueryRow(sqlStmt, jUUID).Scan(&uUUID, &createdAt, &canceledAt, &rate); err != nil {
			return "error updating job canceled_at, active", err
		}

		log.Sugar.Infof("Rate: %v\n", rate)
		if rate.Valid {
			amt := rate.Float64 * canceledAt.Sub(createdAt).Hours()
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

	// TODO: add responsewriter & return user balancer w/ w.Write ?
	return nil
}
