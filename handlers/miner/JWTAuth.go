package miner

import (
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/dgrijalva/jwt-go/request"
	"github.com/gorilla/mux"
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
		claims := &minerClaims{}
		token, err := request.ParseFromRequest(r, request.AuthorizationHeaderExtractor,
			func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(secret), nil
			}, request.WithClaims(claims))
		if err != nil {
			log.Printf("Unable to parse miner JWT: %v\n", err)
			http.Error(w, "Unable to parse miner JWT", http.StatusInternalServerError)
			return
		}

		if token.Valid {
			log.Printf("Valid miner login: %v\n", claims.Email)
		} else {
			log.Printf("Invalid or unauthorized miner JWT\n")
			http.Error(w, "Invalid or unauthorized JWT.", http.StatusUnauthorized)
			return
		}

		vars := mux.Vars(r)
		mID := vars["mID"]
		if mID != claims.Subject {
			log.Printf("URL path miner ID doesn't match miner request header Authorization claim.\n")
			http.Error(w, "URL path miner ID doesn't match miner request header Authorization claim.", http.StatusUnauthorized)
			return
		}

		h(w, r)
	})
}
