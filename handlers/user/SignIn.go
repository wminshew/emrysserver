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
		w.WriteHeader(http.StatusBadRequest)
		log.Printf("Error decoding json:\n", err)
		return
	}

	storedCreds := &Credentials{}
	// errors from QueryRow are defered until Scan
	result := db.Db.QueryRow("SELECT email, password FROM users WHERE email=$1", creds.Email)
	err = result.Scan(&storedCreds.Email, &storedCreds.Password)
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusUnauthorized)
			log.Printf("Unauthorized user: %s\n", creds.Email)
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("Internal error:\n", err)
		return
	}

	if err = bcrypt.CompareHashAndPassword([]byte(storedCreds.Password), []byte(creds.Password)); err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		log.Printf("Unauthorized user: %s\n", creds.Email)
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
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("Internal error:\n", err)
	}

	response := SignInResponse{
		Token: tokenString,
	}
	json.NewEncoder(w).Encode(response)
}
