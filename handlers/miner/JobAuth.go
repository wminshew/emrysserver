package miner

import (
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/dgrijalva/jwt-go/request"
	"github.com/gorilla/mux"
	"log"
	"net/http"
)

// Extract bearer token from Job-Authorization header
var jobAuthorizationHeaderExtractor = &request.HeaderExtractor{"Job-Authorization"}

// JobAuth authenticates miner job tokens
func JobAuth(h http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := &jwt.StandardClaims{}
		token, err := request.ParseFromRequest(r, jobAuthorizationHeaderExtractor,
			func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(secret), nil
			}, request.WithClaims(claims))
		if err != nil {
			log.Printf("Unable to parse miner job JWT.\n")
			http.Error(w, "Unable to parse miner job JWT.", http.StatusInternalServerError)
			return
		}

		if token.Valid {
			log.Printf("Valid miner job JWT: %v\n", claims.Subject)
		} else {
			log.Printf("Invalid or unauthorized job JWT.\n")
			http.Error(w, "Invalid or unauthorized job JWT.", http.StatusUnauthorized)
			return
		}

		vars := mux.Vars(r)
		jID := vars["jID"]
		if jID != claims.Subject {
			log.Printf("URL path job ID doesn't match miner request header Job-Authorization claim.\n")
			http.Error(w, "URL path job ID doesn't match miner request header Job-Authorization claim.", http.StatusUnauthorized)
			return
		}

		h(w, r)
	})
}
