package auth

import (
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/dgrijalva/jwt-go/request"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// Jwt returns middleware for authenticating jwts
func Jwt(secret string) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return jwtAuth(h, secret)
	}
}

const (
	challenge = "WWW-Authenticate"
	realm     = "Bearer realm=\"emrys.io\""
)

// jwtAuth authenticates jwts, given a secret
func jwtAuth(h http.Handler, secret string) app.Handler {
	return func(w http.ResponseWriter, r *http.Request) *app.Error {
		claims := &jwt.StandardClaims{}
		token, err := request.ParseFromRequest(r, request.AuthorizationHeaderExtractor,
			func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(secret), nil
			}, request.WithClaims(claims))
		if err != nil {
			log.Sugar.Infow("error parsing jwt",
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
			)
			w.Header().Set(challenge, realm)
			return &app.Error{Code: http.StatusUnauthorized, Message: "unauthorized jwt, please login again"}
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
			return &app.Error{Code: http.StatusUnauthorized, Message: "unauthorized jwt, please login again"}
		}

		r.Header.Set("X-Jwt-Claims-Subject", claims.Subject)
		r.Header.Del("Authorization")
		h.ServeHTTP(w, r)
		return nil
	}
}
