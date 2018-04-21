// package user
package user

import (
	"database/sql"
	"encoding/json"
	"github.com/dgrijalva/jwt-go"
	"github.com/wminshew/emrysserver/db"
	"golang.org/x/crypto/bcrypt"
	"log"
	"net/http"
	"os"
	"time"
)

var secret = os.Getenv("SECRET")

type SignInResponse struct {
	Token string `json:"token"`
}

func SignIn(w http.ResponseWriter, r *http.Request) {
	creds := &Credentials{}
	err := json.NewDecoder(r.Body).Decode(creds)
	if err != nil {
		log.Printf("Error decoding json:\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	storedCreds := &Credentials{}
	// errors from QueryRow are defered until Scan
	result := db.Db.QueryRow("SELECT email, password FROM users WHERE email=$1", creds.Email)
	err = result.Scan(&storedCreds.Email, &storedCreds.Password)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("Unauthorized user: %s\n", creds.Email)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		log.Printf("Database error during sign in:\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err = bcrypt.CompareHashAndPassword([]byte(storedCreds.Password), []byte(creds.Password)); err != nil {
		log.Printf("Unauthorized user: %s\n", creds.Email)
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
		log.Printf("Internal error:\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := SignInResponse{
		Token: tokenString,
	}
	json.NewEncoder(w).Encode(response)
}
