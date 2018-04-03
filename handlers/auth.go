// server handlers
package handlers

import (
	"golang.org/x/crypto/bcrypt"
	"log"
)

func auth(user, pass string) bool {
	// username := "admin"
	password := "123456"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), Cost)
	if err != nil {
		log.Println(err)
	}
	err = bcrypt.CompareHashAndPassword(hashedPassword, []byte(pass))
	if err != nil {
		log.Println(err)
	} else {
		return true
	}
	return false
}
