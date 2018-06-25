// Package check handles errors in defer statements
package check

import (
	"github.com/wminshew/emrysserver/pkg/app"
)

// Err checks if deferredFunc throws an error and sets err if it hasn't already been set
func Err(r *http.Request, deferredFunc func() error) {
	if err := deferredFunc(); err != nil {
		app.Sugar.Errorw("error in deferred function",
			"url", r.URL,
			"err", err.Error(),
		)
	}
}
