// Package user ...
package user

import (
	"context"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/dgrijalva/jwt-go/request"
	"github.com/satori/go.uuid"
	"log"
	"net/http"
)

type userClaims struct {
	Email string `json:"email"`
	jwt.StandardClaims
}

// JWTAuth authenticates user JWT
func JWTAuth(h http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := &userClaims{}
		token, err := request.ParseFromRequest(r, request.AuthorizationHeaderExtractor,
			func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(secret), nil
			}, request.WithClaims(claims))

		if token.Valid {
			log.Printf("Valid user login: %v\n", claims.Email)
		} else {
			log.Printf("Invalid or unauthorized user JWT\n")
			http.Error(w, "Invalid or unauthorized JWT.", http.StatusUnauthorized)
			return
		}

		u, err := uuid.FromString(claims.Subject)
		if err != nil {
			log.Printf("Unable to retrieve valid uuid from jwt\n")
			http.Error(w, "Unable to retrieve valid uuid from jwt", http.StatusInternalServerError)
			return
		}
		ctx := r.Context()
		ctxKey := contextKey("user_uuid")
		ctx = context.WithValue(ctx, ctxKey, u)
		r = r.WithContext(ctx)
		h(w, r)
	})
}
