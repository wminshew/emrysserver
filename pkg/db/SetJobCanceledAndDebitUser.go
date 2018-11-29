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
		uUUID := uuid.UUID{}
		mUUID := uuid.UUID{}
		createdAt := time.Time{}
		canceledAt := pq.NullTime{}
		rate := sql.NullFloat64{}
		sqlStmt := `
		UPDATE jobs j
		SET canceled_at = NOW(),
		active = false
		FROM users u, projects proj, miners m, bids b
		WHERE j.uuid = $1 AND
			j.canceled_at IS NULL AND
			proj.uuid = j.project_uuid AND
			u.uuid = proj.user_uuid AND (
			j.win_bid_uuid IS NULL OR (
			b.uuid = j.win_bid_uuid AND
			m.uuid = b.miner_uuid
			))
		RETURNING u.uuid, m.uuid, j.created_at, j.canceled_at, j.rate
		`
		if err := db.QueryRow(sqlStmt, jUUID).Scan(&uUUID, &mUUID, &createdAt, &canceledAt, &rate); err != nil {
			return "error updating jobs canceled_at, active", err
		}

		if canceledAt.Valid {
			if rate.Valid {
				amt := rate.Float64 * canceledAt.Time.Sub(createdAt).Hours()
				sqlStmt = `
				UPDATE accounts a
				SET balance = balance - $2
				WHERE a.uuid = $1
				`
				if _, err := db.Exec(sqlStmt, uUUID, amt); err != nil {
					return "error updating user accounts balance", err
				}

				sqlStmt = `
				UPDATE accounts a
				SET balance = balance + $2
				WHERE a.uuid = $1
				`
				if _, err := db.Exec(sqlStmt, mUUID, amt); err != nil {
					return "error updating miner accounts balance", err
				}

				sqlStmt = `
				UPDATE payments p
				SET processed = $2,
				amount = $3
				WHERE p.job_uuid = $1 AND
					p.processed IS NULL
				`
				if _, err := db.Exec(sqlStmt, jUUID, canceledAt.Time, amt); err != nil {
					return "error updating payments processed", err
				}
			} else {
				sqlStmt = `
				UPDATE payments p
				SET processed = $2
				WHERE p.job_uuid = $1 AND
					p.processed IS NULL
				`
				if _, err := db.Exec(sqlStmt, jUUID, canceledAt.Time); err != nil {
					return "error updating payments processed", err
				}
			}
		} else {
			// shouldn't happen
			log.Sugar.Errorf("null canceledAt")
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
