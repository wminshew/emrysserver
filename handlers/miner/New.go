package miner

import (
	"encoding/json"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrys/pkg/creds"
	"github.com/wminshew/emrysserver/db"
	"golang.org/x/crypto/bcrypt"
	"log"
	"net/http"
)

const cost = 14

// New creates a new miners entry in database if successful
func New(w http.ResponseWriter, r *http.Request) {
	c := &creds.Miner{}
	err := json.NewDecoder(r.Body).Decode(c)
	if err != nil {
		log.Printf("Error decoding json: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(c.Password), cost)

	u := uuid.NewV4()
	if _, err = db.Db.Query("INSERT INTO miners (miner_email, password, miner_uuid) VALUES ($1, $2, $3)",
		c.Email, string(hashedPassword), u); err != nil {
		log.Printf("Error querying db: %v\n", err)
		// TODO: Deliver clearer error messages when possible
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	log.Printf("Miner %s successfully added!", c.Email)
}
