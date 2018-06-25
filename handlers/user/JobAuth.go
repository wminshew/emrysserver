package user

import (
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/dgrijalva/jwt-go/request"
	"github.com/gorilla/mux"
	"github.com/wminshew/emrysserver/pkg/app"
	"net/http"
)

// Extract bearer token from Job-Authorization header
var jobAuthorizationHeaderExtractor = &request.HeaderExtractor{"Job-Authorization"}

// JobAuth authenticates miner job tokens
func JobAuth(h app.Handler) app.Handler {
	return func(w http.ResponseWriter, r *http.Request) *app.Error {
		claims := &jwt.StandardClaims{}
		token, err := request.ParseFromRequest(r, jobAuthorizationHeaderExtractor,
			func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(secret), nil
			}, request.WithClaims(claims))
		if err != nil {
			app.Sugar.Errorw("failed to parse user job JWT",
				"url", r.URL,
				"err", err.Error(),
			)
			return &app.Error{Code: http.StatusBadRequest, Message: "error parsing user job token"}
		}

		if token.Valid {
			app.Sugar.Infow("valid user job jWT",
				"url", r.URL,
				"sub", claims.Subject,
			)
		} else {
			app.Sugar.Infow("unauthorized user job JWT",
				"url", r.URL,
				"sub", claims.Subject,
			)
			return &app.Error{Code: http.StatusUnauthorized, Message: "unauthorized user job JWT"}
		}

		vars := mux.Vars(r)
		jID := vars["jID"]
		if jID != claims.Subject {
			app.Sugar.Infow("URL path job ID doesn't match user request header Job-Authorization claim",
				"url", r.URL,
				"sub", claims.Subject,
				"jID", jID,
			)
			return &app.Error{Code: http.StatusUnauthorized, Message: "URL path user job ID doesn't match user request header Job-Authorization claim"}
		}

		return h(w, r)
	}
}
