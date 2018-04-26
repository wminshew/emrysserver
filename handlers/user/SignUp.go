package user

import (
	"encoding/json"
	"github.com/wminshew/emrysserver/db"
	"golang.org/x/crypto/bcrypt"
	"log"
	"net/http"
)

const Cost = 14

// creates new users entry in database if successful
func SignUp(w http.ResponseWriter, r *http.Request) {
	creds := &Credentials{}
	err := json.NewDecoder(r.Body).Decode(creds)
	if err != nil {
		log.Printf("Error decoding json:\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(creds.Password), Cost)

	// TODO: Need to deliver clear error messages to user if possible (i.e. if email already exists, or if its invalid)
	if _, err = db.Db.Query("INSERT INTO users VALUES ($1, $2)", creds.Email, string(hashedPassword)); err != nil {
		log.Printf("Error querying db:\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	log.Printf("User %s successfully added!", creds.Email)
}
