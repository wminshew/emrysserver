package main

import (
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// confirmMiner confirms a new user
var confirmMiner app.Handler = func(w http.ResponseWriter, r *http.Request) *app.Error {
	tokenStr := r.URL.Query().Get("token")

	claims := &jwt.StandardClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims,
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(minerSecret), nil
		})
	if err != nil {
		log.Sugar.Infow("error parsing jwt",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusUnauthorized, Message: "unauthorized jwt, please contact support"}
	}

	if token.Valid {
		log.Sugar.Infow("valid jwt",
			"method", r.Method,
			"url", r.URL,
			"sub", claims.Subject,
		)
	} else {
		log.Sugar.Infow("unauthorized jwt",
			"method", r.Method,
			"url", r.URL,
			"sub", claims.Subject,
		)
		// TODO: handle expired tokens specially
		return &app.Error{Code: http.StatusUnauthorized, Message: "unauthorized jwt, please contact support"}
	}

	mUUID, err := uuid.FromString(claims.Subject)
	if err != nil {
		log.Sugar.Errorw("error parsing user claims.Subject",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "error parsing token, please contact support"}
	}

	if err := db.SetMinerConfirmed(r, mUUID); err != nil {
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"} // already logged
	}
	log.Sugar.Infof("Miner %s successfully confirmed!", mUUID.String())

	_, _ = w.Write([]byte("email confirmed"))

	return nil
}
