package miner

import (
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/dgrijalva/jwt-go/request"
	"log"
	"net/http"
)

type minerClaims struct {
	Email string `json:"email"`
	jwt.StandardClaims
}

// JWTAuth authenticates miner tokens
func JWTAuth(h http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := request.ParseFromRequest(r, request.AuthorizationHeaderExtractor,
			func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(secret), nil
			}, request.WithClaims(&minerClaims{}))

		if claims, ok := token.Claims.(*minerClaims); ok && token.Valid {
			log.Printf("Valid miner login: %v", claims.Email)
		} else {
			log.Printf("Unauthorized miner JWT")
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		h(w, r)
	})
}
