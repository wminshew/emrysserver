package main

import (
	"encoding/json"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/dgrijalva/jwt-go/request"
	"github.com/wminshew/emrys/pkg/creds"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
	"time"
)

const (
	refreshDuration = 1
)

// refreshToken takes the token from the request and, if valid, returns a new token with short expiry
var refreshToken app.Handler = func(w http.ResponseWriter, r *http.Request) *app.Error {
	claims := &creds.JwtClaims{}
	token, err := request.ParseFromRequest(r, request.AuthorizationHeaderExtractor,
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(authSecret), nil
		}, request.WithClaims(claims))
	if err != nil {
		log.Sugar.Infow("error parsing jwt",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusUnauthorized, Message: "unauthorized jwt"}
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
		return &app.Error{Code: http.StatusUnauthorized, Message: "unauthorized jwt"}
	}

	newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"aud":   "emrys.io",
		"exp":   time.Now().Add(time.Hour * refreshDuration).Unix(),
		"iss":   "emrys.io",
		"iat":   time.Now().Unix(),
		"sub":   claims.Subject,
		"scope": claims.Scope,
	})

	tokenString, err := newToken.SignedString([]byte(authSecret))
	if err != nil {
		log.Sugar.Errorw("error signing token",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"aID", claims.Subject,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	resp := creds.LoginResp{
		Token: tokenString,
	}
	if err = json.NewEncoder(w).Encode(resp); err != nil {
		log.Sugar.Errorw("error encoding json response",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"aID", claims.Subject,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	log.Sugar.Infow("account token refresh",
		"method", r.Method,
		"url", r.URL,
		"sub", claims.Subject,
	)
	return nil
}
