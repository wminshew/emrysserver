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

	promoUsed := false
	var credit, promoUses int
	promo := r.URL.Query().Get("promo")
	if isUser {
		if promo != "" {
			promoCredit, expiration, uses, maxUses, err := db.GetPromoInfo(promo)
			if err != nil {
				log.Sugar.Errorw("error getting promo info",
					"method", r.Method,
					"url", r.URL,
					"err", err.Error(),
					"email", c.Email,
				)
				return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
			}

			if promoCredit == 0 {
				// should never happen
				log.Sugar.Errorw("promo has no credit set",
					"method", r.Method,
					"url", r.URL,
					"promo", promo,
				)
				credit = newUserCredit
			} else if expiration.IsZero() {
				// should never happen
				log.Sugar.Errorw("promo has no expiration date set",
					"method", r.Method,
					"url", r.URL,
					"promo", promo,
				)
				credit = newUserCredit
			} else if expiration.Before(time.Now()) {
				log.Sugar.Infof("promo %s expired as of %s", promo, expiration.Format("2009-01-01"))
				credit = newUserCredit
			} else if uses >= maxUses {
				log.Sugar.Infof("promo %s used max number of times: %d / %d", promo, uses, maxUses)
				credit = newUserCredit
			} else {
				credit = promoCredit
				promoUsed = true
				promoUses = uses
			}
		} else {
			credit = newUserCredit
		}
	}

	aUUID := uuid.NewV4()
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

	if promoUsed {
		// TODO: race condition: two users registering at the same time could use the same promo but only register one use
		// counter: would still be picked up by log
		// solution: could add to log which "use" slot the entry takes & make a unique index across promo, use slot
		if err := db.SetPromoUses(promo, promoUses); err != nil {
			log.Sugar.Errorw("error updating promo uses",
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"aID", aUUID,
				"promo", promo,
				"uses", promoUses,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}
		if err := db.InsertPromoUse(aUUID, promo); err != nil {
			log.Sugar.Errorw("error inserting promo use to log",
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"aID", aUUID,
				"promo", promo,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}
	}

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
