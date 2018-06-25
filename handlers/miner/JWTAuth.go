package miner

import (
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/dgrijalva/jwt-go/request"
	"github.com/gorilla/mux"
	"github.com/wminshew/emrysserver/pkg/app"
	"net/http"
)

type minerClaims struct {
	Email string `json:"email"`
	jwt.StandardClaims
}

// JWTAuth authenticates miner tokens
func JWTAuth(h app.Handler) app.Handler {
	return func(w http.ResponseWriter, r *http.Request) *app.Error {
		claims := &minerClaims{}
		token, err := request.ParseFromRequest(r, request.AuthorizationHeaderExtractor,
			func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(secret), nil
			}, request.WithClaims(claims))
		if err != nil {
			app.Sugar.Errorw("failed to parse miner JWT",
				"url", r.URL,
				"err", err.Error(),
			)
			return &app.Error{Code: http.StatusBadRequest, Message: "error parsing miner token. Please login again"}
		}

		if token.Valid {
			app.Sugar.Infow("valid miner login",
				"url", r.URL,
				"sub", claims.Subject,
			)
		} else {
			app.Sugar.Infow("unauthorized miner JWT",
				"url", r.URL,
				"sub", claims.Subject,
			)
			return &app.Error{Code: http.StatusUnauthorized, Message: "unauthorized miner JWT. Please login again"}
		}

		vars := mux.Vars(r)
		mID := vars["mID"]
		if mID != claims.Subject {
			app.Sugar.Infow("URL path miner ID doesn't match miner request header Authorization claim",
				"url", r.URL,
				"sub", claims.Subject,
				"mID", mID,
			)
			return &app.Error{Code: http.StatusUnauthorized, Message: "URL path miner ID doesn't match miner request header Authorization claim"}
		}

		return h(w, r)
	}
}
