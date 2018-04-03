package handlers

import (
	"encoding/json"
	"github.com/wminshew/emrysserver/db"
	"golang.org/x/crypto/bcrypt"
	"log"
	"net/http"
)

const Cost = 14

func SignUpUser(w http.ResponseWriter, r *http.Request) {
	creds := &Credentials{}
	err := json.NewDecoder(r.Body).Decode(creds)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Printf("Error decoding json:\n", err)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(creds.Password), Cost)

	// TODO: Need to deliver clear error messages to user if possible (i.e. if username already exists)
	if _, err = db.Db.Query("INSERT INTO users VALUES ($1, $2)", creds.Username, string(hashedPassword)); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("Error querying db:\n", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	log.Printf("User %s successfully added!", creds.Username)
}
