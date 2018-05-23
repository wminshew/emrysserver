package miner

import (
	"context"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/dgrijalva/jwt-go/request"
	"github.com/satori/go.uuid"
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
		if err != nil {
			log.Printf("Unable to parse miner JWT\n")
			http.Error(w, "Unable to parse miner JWT", http.StatusInternalServerError)
			return
		}

		claims, ok := token.Claims.(*minerClaims)
		if ok && token.Valid {
			log.Printf("Valid miner login: %v\n", claims.Email)
		} else {
			log.Printf("Invalid or unauthorized miner JWT\n")
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
		ctxKey := contextKey("miner_uuid")
		ctx = context.WithValue(ctx, ctxKey, u)
		r = r.WithContext(ctx)
		h(w, r)
	})
}
