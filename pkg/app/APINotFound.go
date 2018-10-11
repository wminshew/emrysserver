package app

import (
	"net/http"
)

// APINotFound allows kubernetes to health check the pod
func APINotFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte("API endpoint not found"))
}
