// Package user ...
package user

import (
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/dgrijalva/jwt-go/request"
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
		token, err := request.ParseFromRequest(r, request.AuthorizationHeaderExtractor,
			func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(secret), nil
			}, request.WithClaims(&userClaims{}))

		if claims, ok := token.Claims.(*userClaims); ok && token.Valid {
			log.Printf("Valid user login: %v", claims.Email)
		} else {
			log.Printf("Unauthorized user JWT")
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		h(w, r)
	})
}
