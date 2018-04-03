package handlers

import (
	"database/sql"
	"encoding/json"
	"github.com/wminshew/emrysserver/db"
	"golang.org/x/crypto/bcrypt"
	"log"
	"net/http"
)

func SignInUser(w http.ResponseWriter, r *http.Request) {
	creds := &Credentials{}
	err := json.NewDecoder(r.Body).Decode(creds)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Printf("Error decoding json:\n", err)
		return
	}

	storedCreds := &Credentials{}
	// errors from QueryRow are defered until Scan is called
	result := db.Db.QueryRow("SELECT password FROM users WHERE username=$1", creds.Username)
	err = result.Scan(&storedCreds.Password)
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusUnauthorized)
			log.Printf("Unauthorized user: %s\n", creds.Username)
			return
		}

		// TODO: do we really want to give users different errors based on whether we have that username or not?
		// seems like it might give too much away (aka hacker knows whether a username exists)
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("Internal error:\n", err)
		return
	}

	if err = bcrypt.CompareHashAndPassword([]byte(storedCreds.Password), []byte(creds.Password)); err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		log.Printf("Unauthorized user: %s\n", creds.Username)
		return
	}

	w.WriteHeader(http.StatusOK)
	return
}
