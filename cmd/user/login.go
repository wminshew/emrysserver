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
	"github.com/wminshew/emrysserver/pkg/log"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"strconv"
	"time"
)

const (
	stdDuration = 7
)

// login takes user credentials from the request and, if valid, returns a token
func login(w http.ResponseWriter, r *http.Request) *app.Error {
	c := &creds.User{}
	if err := json.NewDecoder(r.Body).Decode(c); err != nil {
		log.Sugar.Errorw("failed to decode json request body",
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "Error parsing json request body"}
	}

	storedC := &creds.User{}
	u := uuid.UUID{}
	sqlStmt := `
	SELECT user_email, password, user_uuid
	FROM users
	WHERE user_email=$1
	`
	if err := db.Db.QueryRow(sqlStmt, c.Email).Scan(&storedC.Email, &storedC.Password, &u); err != nil {
		if err == sql.ErrNoRows {
			log.Sugar.Infow("unauthorized user",
				"url", r.URL,
				"email", c.Email,
			)
			return &app.Error{Code: http.StatusUnauthorized, Message: "unauthorized user"}
		}

		pqErr := err.(*pq.Error)
		if pqErr.Fatal() {
			log.Sugar.Fatalw("failed to query database",
				"url", r.URL,
				"err", err.Error(),
				"email", c.Email,
				"pq_sev", pqErr.Severity,
				"pq_code", pqErr.Code,
				"pq_detail", pqErr.Detail,
			)
		} else {
			log.Sugar.Errorw("failed to query database",
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

	if err := bcrypt.CompareHashAndPassword([]byte(storedC.Password), []byte(c.Password)); err != nil {
		log.Sugar.Infow("unauthorized user",
			"url", r.URL,
			"email", c.Email,
		)
		return &app.Error{Code: http.StatusUnauthorized, Message: "unauthorized user"}
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
		"sub":   u,
	})

	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		log.Sugar.Errorw("failed to sign token",
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
		log.Sugar.Errorw("failed to encode json response",
			"url", r.URL,
			"err", err.Error(),
			"email", c.Email,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	log.Sugar.Infow("user login",
		"url", r.URL,
		"sub", u,
		"email", c.Email,
	)
	return nil
}
