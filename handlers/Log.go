// Package handlers handles requests from user and miner clients
package handlers

import (
	"github.com/wminshew/emrysserver/pkg/app"
	"net/http"
)

// Log logs request method, URL, & address
func Log(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.Sugar.Infow("request",
			"method", r.Method,
			"url", r.URL,
			"source", r.RemoteAddr,
		)
		handler.ServeHTTP(w, r)
	})
}
