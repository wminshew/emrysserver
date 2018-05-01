package miner

import (
	"database/sql"
	"encoding/json"
	"github.com/dgrijalva/jwt-go"
	"github.com/wminshew/emrysserver/db"
	"github.com/wminshew/emrysserver/handlers"
	"golang.org/x/crypto/bcrypt"
	"log"
	"net/http"
	"os"
	"time"
)

var secret = os.Getenv("SECRET")

type signInResponse struct {
	Token string `json:"token"`
}

// SignIn takes credentials from the request and, if valid, returns a token
func SignIn(w http.ResponseWriter, r *http.Request) {
	creds := &handlers.Credentials{}
	err := json.NewDecoder(r.Body).Decode(creds)
	if err != nil {
		log.Printf("Error decoding json: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	storedCreds := &handlers.Credentials{}
	// errors from QueryRow are defered until Scan
	result := db.Db.QueryRow("SELECT email, password FROM miners WHERE email=$1", creds.Email)
	err = result.Scan(&storedCreds.Email, &storedCreds.Password)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("Unauthorized miner: %s\n", creds.Email)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		log.Printf("Database error during sign in: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err = bcrypt.CompareHashAndPassword([]byte(storedCreds.Password), []byte(creds.Password)); err != nil {
		log.Printf("Unauthorized miner: %s\n", creds.Email)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"exp":   time.Now().Add(time.Hour * 24).Unix(),
		"iss":   "auth.service",
		"iat":   time.Now().Unix(),
		"email": storedCreds.Email,
		"sub":   storedCreds.Email,
	})

	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		log.Printf("Internal error: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := signInResponse{
		Token: tokenString,
	}
	if err = json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding JSON response: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
