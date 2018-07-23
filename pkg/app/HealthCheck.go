package app

import (
	"net/http"
)

// HealthCheck allows kubernetes to health check the pod
func HealthCheck(w http.ResponseWriter, r *http.Request) {}
