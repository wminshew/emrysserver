package user

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
		token, err := request.ParseFromRequest(r, jobAuthorizationHeaderExtractor,
			func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(secret), nil
			}, request.WithClaims(&jwt.StandardClaims{}))
		if err != nil {
			log.Printf("Unable to parse user job JWT.\n")
			http.Error(w, "Unable to parse user job JWT. ", http.StatusInternalServerError)
			return
		}

		claims, ok := token.Claims.(*jwt.StandardClaims)
		if ok && token.Valid {
			vars := mux.Vars(r)
			jID := vars["jID"]
			log.Printf("jID: %v\n", jID)
			log.Printf("token: %v\n", claims.Subject)
			if jID != claims.Subject {
				log.Printf("URL path job ID doesn't match user request header Job-Authorization claim.\n")
				http.Error(w, "URL path job ID doesn't match user request header Job-Authorization claim.", http.StatusUnauthorized)
				return
			}
			log.Printf("Valid user job JWT: %v\n", claims.Subject)
		} else {
			log.Printf("Invalid or unauthorized user job JWT.\n")
			http.Error(w, "Invalid or unauthorized user job JWT.", http.StatusUnauthorized)
			return
		}

		h(w, r)
	})
}
