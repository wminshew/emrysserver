// package user
package user

import (
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/dgrijalva/jwt-go/request"
	"log"
	"net/http"
)

type UserClaims struct {
	Email string `json:"email"`
	jwt.StandardClaims
}

// authenticates user JWT
func JWTAuth(h http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := request.ParseFromRequest(r, request.AuthorizationHeaderExtractor,
			func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(secret), nil
			}, request.WithClaims(&UserClaims{}))

		if claims, ok := token.Claims.(*UserClaims); ok && token.Valid {
			log.Printf("Valid login: %v", claims.Email)
		} else {
			log.Printf("Unauthorized login: %v", claims.Email)
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		// TODO: add Context to pass JWT claims & possibly validity?
		// I mean technically everything should be authed and shouldn't get into an API if invalid..
		// but might be good to have. Not sure
		h(w, r)
	})
}
