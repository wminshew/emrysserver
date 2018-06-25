package user

import (
	"database/sql"
	"encoding/json"
	"github.com/dgrijalva/jwt-go"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrys/pkg/creds"
	"github.com/wminshew/emrysserver/db"
	"github.com/wminshew/emrysserver/pkg/app"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"os"
	"strconv"
	"time"
)

var secret = os.Getenv("SECRET")

const (
	stdDuration = 7
)

// Login takes user credentials from the request and, if valid, returns a token
func Login(w http.ResponseWriter, r *http.Request) *app.Error {
	c := &creds.User{}
	err := json.NewDecoder(r.Body).Decode(c)
	if err != nil {
		app.Sugar.Errorw("failed to decode json request body",
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
	if err = db.Db.QueryRow(sqlStmt, c.Email).Scan(&storedC.Email, &storedC.Password, &u); err != nil {
		if err == sql.ErrNoRows {
			app.Sugar.Infow("unauthorized user",
				"url", r.URL,
				"email", c.Email,
			)
			return &app.Error{Code: http.StatusUnauthorized, Message: "unauthorized user"}
		}

		app.Sugar.Errorw("failed to query database",
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	if err = bcrypt.CompareHashAndPassword([]byte(storedC.Password), []byte(c.Password)); err != nil {
		app.Sugar.Infow("unauthorized user",
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
		app.Sugar.Errorw("failed to sign token",
			"url", r.URL,
			"err", err.Error(),
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
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	app.Sugar.Infow("user login",
		"url", r.URL,
		"sub", u,
	)
	return nil
}
