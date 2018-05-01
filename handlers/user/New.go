package user

import (
	"encoding/json"
	"github.com/wminshew/emrysserver/db"
	"github.com/wminshew/emrysserver/handlers"
	"golang.org/x/crypto/bcrypt"
	"log"
	"net/http"
)

const cost = 14

// New creates a new users entry in database if successful
func New(w http.ResponseWriter, r *http.Request) {
	creds := &handlers.Credentials{}
	err := json.NewDecoder(r.Body).Decode(creds)
	if err != nil {
		log.Printf("Error decoding json: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(creds.Password), cost)

	// TODO: Need to deliver clear error messages to user if possible (i.e. if email already exists, or if its invalid)
	if _, err = db.Db.Query("INSERT INTO users VALUES ($1, $2)", creds.Email, string(hashedPassword)); err != nil {
		log.Printf("Error querying db: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	log.Printf("User %s successfully added!", creds.Email)
}
