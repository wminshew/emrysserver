package miner

import (
	"database/sql"
	"encoding/json"
	"github.com/dgrijalva/jwt-go"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrys/pkg/creds"
	"github.com/wminshew/emrysserver/db"
	"golang.org/x/crypto/bcrypt"
	"log"
	"net/http"
	"os"
	"time"
)

var secret = os.Getenv("SECRET")

type loginResponse struct {
	Token string `json:"token"`
}

// Login takes miner credentials from the request and, if valid, returns a token
func Login(w http.ResponseWriter, r *http.Request) {
	c := &creds.Miner{}
	err := json.NewDecoder(r.Body).Decode(c)
	if err != nil {
		log.Printf("Error decoding json: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	storedC := &creds.Miner{}
	u := uuid.UUID{}
	// errors from QueryRow are defered until Scan
	result := db.Db.QueryRow("SELECT miner_email, password, miner_uuid FROM miners WHERE miner_email=$1", c.Email)
	err = result.Scan(&storedC.Email, &storedC.Password, &u)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("Unauthorized miner: %s\n", c.Email)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		log.Printf("Database error during login: %v\n", err)
		http.Error(w, "Error during login.", http.StatusInternalServerError)
		return
	}

	if err = bcrypt.CompareHashAndPassword([]byte(storedC.Password), []byte(c.Password)); err != nil {
		log.Printf("Unauthorized miner: %s\n", c.Email)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"exp":   time.Now().Add(time.Hour * 24).Unix(),
		"iss":   "auth.service",
		"iat":   time.Now().Unix(),
		"email": storedC.Email,
		"sub":   u,
	})

	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		log.Printf("Internal error: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := loginResponse{
		Token: tokenString,
	}
	if err = json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding JSON response: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
