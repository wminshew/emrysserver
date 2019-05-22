package main

import (
	"encoding/json"
	"github.com/dgrijalva/jwt-go"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrys/pkg/creds"
	"github.com/wminshew/emrys/pkg/validate"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/email"
	"github.com/wminshew/emrysserver/pkg/log"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"time"
)

// newAccount creates a new accounts entry in database if successful
var newAccount app.Handler = func(w http.ResponseWriter, r *http.Request) *app.Error {
	c := &creds.Account{}
	if err := json.NewDecoder(r.Body).Decode(c); err != nil {
		log.Sugar.Errorw("error decoding json request body",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "error parsing json request body"}
	}

	agreedToTOSAndPrivacy := r.URL.Query().Get("terms") != ""
	if !agreedToTOSAndPrivacy {
		log.Sugar.Infow("must agree to the Terms of Service and Privacy Policy",
			"method", r.Method,
			"url", r.URL,
			"email", c.Email,
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "must agree to the Terms of Service and Privacy Policy"}
	}

	if c.FirstName == "" {
		log.Sugar.Infow("no first name included",
			"method", r.Method,
			"url", r.URL,
			"email", c.Email,
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "must include first name"}
	}
	if c.LastName == "" {
		log.Sugar.Infow("no last name included",
			"method", r.Method,
			"url", r.URL,
			"email", c.Email,
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "must include last name"}
	}

	isUser := r.URL.Query().Get("user") != ""
	isMiner := r.URL.Query().Get("miner") != ""
	if !isUser && !isMiner {
		log.Sugar.Infow("must sign up as a user and/or miner",
			"method", r.Method,
			"url", r.URL,
			"email", c.Email,
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "must sign up as a user or miner"}
	}

	if c.Email == "" {
		log.Sugar.Infow("no email address included",
			"method", r.Method,
			"url", r.URL,
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "must sign up with email address"}
	} else if !validate.EmailRegexp().MatchString(c.Email) {
		log.Sugar.Infow("invalid email",
			"method", r.Method,
			"url", r.URL,
			"email", c.Email,
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "email invalid"}
	}

	if c.Password == "" {
		log.Sugar.Infow("no password included",
			"method", r.Method,
			"url", r.URL,
			"email", c.Email,
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "no password included"}
	} else if !validate.Password(c.Password) {
		log.Sugar.Infow("invalid password",
			"method", r.Method,
			"url", r.URL,
			"email", c.Email,
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "invalid password"}
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(c.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Sugar.Errorw("error hashing password",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"email", c.Email,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	aUUID := uuid.NewV4()
	credit := newUserCredit
	if !isUser {
		credit = 0
	}
	if err := db.InsertAccount(r, c.Email, string(hashedPassword), aUUID, c.FirstName, c.LastName, isUser, isMiner, credit); err != nil {
		// error already logged
		if err == db.ErrEmailExists {
			return &app.Error{Code: http.StatusBadRequest, Message: err.Error()}
		} else if err == db.ErrNullViolation {
			return &app.Error{Code: http.StatusBadRequest, Message: err.Error()}
		}
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}
	log.Sugar.Infof("Account %s (%s) successfully added!", c.Email, aUUID.String())

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"aud": "emrys.io",
		"exp": time.Now().Add(time.Hour * 24).Unix(),
		"iss": "emrys.io",
		"iat": time.Now().Unix(),
		"sub": aUUID,
	})
	tokenString, err := token.SignedString([]byte(authSecret))
	if err != nil {
		log.Sugar.Errorw("error signing token",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"aID", aUUID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	// TODO: put in go func?
	if err := email.SendEmailConfirmation(c.Email, tokenString); err != nil {
		log.Sugar.Errorw("error sending account confirmation email",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"aID", aUUID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	return nil
}
