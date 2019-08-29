package db

import (
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// SetJobFinishedAndStatusOutputDataPosted sets job completed
// (and inactive) and status for job jUUID
func SetJobFinishedAndStatusOutputDataPosted(r *http.Request,
	jUUID uuid.UUID) error {
	ctx := r.Context()
	tx, txerr := db.BeginTx(ctx, nil)
	if message, err := func() (string, error) {
		if txerr != nil {
			return errBeginTx, txerr
		}

		completedAt := pq.NullTime{}
		sqlStmt := `
		UPDATE jobs j
		SET completed_at = NOW(),
		active = false
		FROM users u, projects proj, miners m, bids b
		WHERE j.uuid = $1 AND
			j.completed_at IS NULL AND
			proj.uuid = j.project_uuid AND
			u.uuid = proj.user_uuid AND
			b.uuid = j.win_bid_uuid AND
			m.uuid = b.miner_uuid
		RETURNING j.completed_at
		`
		if err := tx.QueryRow(sqlStmt, jUUID).Scan(&completedAt); err != nil {
			return "error updating jobs completed_at, active", err
		}

		if completedAt.Valid {
			sqlStmt = `
			UPDATE statuses s
			SET output_data_posted = $2
			WHERE s.job_uuid = $1 AND
				s.output_data_posted IS NULL
			`
			if _, err := db.Exec(sqlStmt, jUUID, completedAt.Time); err != nil {
				return "error updating statuses output_data_posted", err
			}
		} else {
			// shouldn't happen
			log.Sugar.Errorf("null completedAt")
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
				"pq_name", pqErr.Name,
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
		return err
	}
	return nil
}
