// Package user ...
package user

import (
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/dgrijalva/jwt-go/request"
	"github.com/gorilla/mux"
	"github.com/wminshew/emrysserver/pkg/app"
	"net/http"
)

type userClaims struct {
	Email string `json:"email"`
	jwt.StandardClaims
}

// JWTAuth authenticates user JWT
func JWTAuth(h app.Handler) app.Handler {
	return func(w http.ResponseWriter, r *http.Request) *app.Error {
		claims := &userClaims{}
		token, err := request.ParseFromRequest(r, request.AuthorizationHeaderExtractor,
			func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(userSecret), nil
			}, request.WithClaims(claims))
		if err != nil {
			app.Sugar.Errorw("failed to parse user JWT",
				"url", r.URL,
				"err", err.Error(),
			)
			return &app.Error{Code: http.StatusBadRequest, Message: "error parsing user token. Please login again"}
		}

		if token.Valid {
			app.Sugar.Infow("valid user login",
				"url", r.URL,
				"sub", claims.Subject,
			)
		} else {
			app.Sugar.Infow("unauthorized user JWT",
				"url", r.URL,
				"sub", claims.Subject,
			)
			return &app.Error{Code: http.StatusUnauthorized, Message: "unauthorized user JWT. Please login again"}
		}

		vars := mux.Vars(r)
		uID := vars["uID"]
		if uID != claims.Subject {
			app.Sugar.Infow("URL path user ID doesn't match user request header Authorization claim",
				"url", r.URL,
				"sub", claims.Subject,
				"uID", uID,
			)
			return &app.Error{Code: http.StatusUnauthorized, Message: "URL path user ID doesn't match user request header Authorization claim"}
		}

		return h(w, r)
	}
}
