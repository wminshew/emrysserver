package main

import (
	"database/sql"
	"encoding/json"
	"github.com/dgrijalva/jwt-go"
	"github.com/lib/pq"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrys/pkg/creds"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/db"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"strconv"
	"time"
)

const (
	stdDuration = 7
)

// login takes miner credentials from the request and, if valid, returns a token
func login() app.Handler {
	return func(w http.ResponseWriter, r *http.Request) *app.Error {
		c := &creds.Miner{}
		err := json.NewDecoder(r.Body).Decode(c)
		if err != nil {
			app.Sugar.Errorw("failed to decode json request body",
				"url", r.URL,
				"err", err.Error(),
			)
			return &app.Error{Code: http.StatusBadRequest, Message: "error parsing json request body"}
		}

		storedC := &creds.Miner{}
		mUUID := uuid.UUID{}
		sqlStmt := `
	SELECT miner_email, password, miner_uuid
	FROM miners
	WHERE miner_email=$1
	`
		if err = db.Db.QueryRow(sqlStmt, c.Email).Scan(&storedC.Email, &storedC.Password, &mUUID); err != nil {
			if err == sql.ErrNoRows {
				app.Sugar.Infow("unauthorized miner",
					"url", r.URL,
					"email", c.Email,
				)
				return &app.Error{Code: http.StatusUnauthorized, Message: "unauthorized miner"}
			}

			pqErr := err.(*pq.Error)
			if pqErr.Fatal() {
				app.Sugar.Fatalw("failed to query database",
					"url", r.URL,
					"err", err.Error(),
					"email", c.Email,
					"pq_sev", pqErr.Severity,
					"pq_code", pqErr.Code,
					"pq_detail", pqErr.Detail,
				)
			} else {
				app.Sugar.Errorw("failed to query database",
					"url", r.URL,
					"err", err.Error(),
					"email", c.Email,
					"pq_sev", pqErr.Severity,
					"pq_code", pqErr.Code,
					"pq_detail", pqErr.Detail,
				)
			}
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}

		if err = bcrypt.CompareHashAndPassword([]byte(storedC.Password), []byte(c.Password)); err != nil {
			app.Sugar.Infow("unauthorized miner",
				"url", r.URL,
				"email", c.Email,
			)
			return &app.Error{Code: http.StatusUnauthorized, Message: "unauthorized miner"}
		}

		days := stdDuration
		if d, err := strconv.Atoi(c.Duration); err == nil {
			days = d
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"exp":   time.Now().Add(time.Hour * 24 * time.Duration(days)).Unix(),
			"iss":   "auth.service",
			"iat":   time.Now().Unix(),
			"email": storedC.Email,
			"sub":   mUUID,
		})

		tokenString, err := token.SignedString([]byte(minerSecret))
		if err != nil {
			app.Sugar.Errorw("failed to sign token",
				"url", r.URL,
				"err", err.Error(),
				"email", c.Email,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}

		resp := creds.LoginResp{
			Token: tokenString,
		}
		if err = json.NewEncoder(w).Encode(resp); err != nil {
			app.Sugar.Errorw("failed to encode json response",
				"url", r.URL,
				"err", err.Error(),
				"email", c.Email,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}

		app.Sugar.Infow("miner login",
			"url", r.URL,
			"sub", mUUID,
			"email", c.Email,
		)
		return nil
	}
}
