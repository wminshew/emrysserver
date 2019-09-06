package db

import (
	"errors"
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

var (
	// ErrEmailExists lets server send a proper response to client
	ErrEmailExists = errors.New("an account with this email already exists")
	// ErrNullViolation lets server send a proper response to client
	ErrNullViolation = errors.New("your new account request is missing required information")
)

const (
	errEmailExistsCode   = "23505"
	errNullViolationCode = "23502"
	errBeginTx           = "error beginning tx"
	errCommitTx          = "error committing tx"
)

// InsertAccount inserts a new account into the db
func InsertAccount(r *http.Request, email, hashedPassword string, aUUID uuid.UUID, firstName, lastName string, isUser, isMiner bool, newUserCredit int) error {
	ctx := r.Context()
	tx, txerr := db.BeginTx(ctx, nil)
	if message, err := func() (string, error) {
		if txerr != nil {
			return errBeginTx, txerr
		}

		sqlStmt := `
		INSERT INTO accounts (uuid, email, password, first_name, last_name, credit)
		VALUES ($1, $2, $3, $4, $5, $6)
		`
		if _, err := tx.Exec(sqlStmt, aUUID, email, hashedPassword, firstName, lastName, newUserCredit); err != nil {
			return "error inserting account", err
		}

		if isUser {
			sqlStmt = `
			INSERT INTO users (uuid)
			VALUES ($1)
			`
			if _, err := tx.Exec(sqlStmt, aUUID); err != nil {
				return "error inserting user", err
			}
		}

		if isMiner {
			sqlStmt = `
			INSERT INTO miners (uuid)
			VALUES ($1)
			`
			if _, err := tx.Exec(sqlStmt, aUUID); err != nil {
				return "error inserting miner", err
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
				"aID", aUUID,
				"email", email,
				"pq_sev", pqErr.Severity,
				"pq_code", pqErr.Code,
				"pq_msg", pqErr.Message,
				"pq_detail", pqErr.Detail,
			)
			if pqErr.Code == errEmailExistsCode {
				return ErrEmailExists
			} else if pqErr.Code == errNullViolationCode {
				return ErrNullViolation
			}
		} else {
			log.Sugar.Errorw(message,
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"aID", aUUID,
				"email", email,
			)
		}
		return err
	}

	return nil
}
