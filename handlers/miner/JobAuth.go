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
// Uses PostExtractionFilter to strip "Bearer " prefix from header
var jobAuthorizationHeaderExtractor = &request.HeaderExtractor{"Job-Authorization"}

// JobAuth authenticates miner job tokens
func JobAuth(h http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := request.ParseFromRequest(r, jobAuthorizationHeaderExtractor,
			func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(secret), nil
			}, request.WithClaims(&jwt.StandardClaims{}))
		if err != nil {
			log.Printf("Unable to parse job JWT\n")
			http.Error(w, "Unable to parse job JWT", http.StatusInternalServerError)
			return
		}

		claims, ok := token.Claims.(*jwt.StandardClaims)
		if ok && token.Valid {
			vars := mux.Vars(r)
			jID := vars["jID"]
			log.Printf("jID: %v\n", jID)
			log.Printf("token: %v\n", claims.Subject)
			if jID != claims.Subject {
				log.Printf("URL path job ID doesn't match header job JWT\n")
				http.Error(w, "URL path job ID doesn't match header job JWT.", http.StatusUnauthorized)
				return
			}
			log.Printf("Valid job token: %v\n", claims.Subject)
		} else {
			log.Printf("Invalid or unauthorized job JWT\n")
			http.Error(w, "Invalid or unauthorized job JWT.", http.StatusUnauthorized)
			return
		}

		h(w, r)
	})
}
