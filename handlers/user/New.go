package user

import (
	"encoding/json"
	"github.com/wminshew/emrys/pkg/creds"
	"github.com/wminshew/emrysserver/db"
	"golang.org/x/crypto/bcrypt"
	"log"
	"net/http"
)

const cost = 14

// New creates a new users entry in database if successful
func New(w http.ResponseWriter, r *http.Request) {
	c := &creds.User{}
	err := json.NewDecoder(r.Body).Decode(c)
	if err != nil {
		log.Printf("Error decoding json: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(c.Password), cost)

	// TODO: Need to deliver clear error messages to user if possible (i.e. if email already exists, or if its invalid)
	if _, err = db.Db.Query("INSERT INTO users VALUES ($1, $2)", c.Email, string(hashedPassword)); err != nil {
		log.Printf("Error querying db: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	log.Printf("User %s successfully added!", c.Email)
}
