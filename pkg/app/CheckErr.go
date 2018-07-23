package app

import (
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
)

// CheckErr checks if deferredFunc throws an error and sets err if it hasn't already been set
func CheckErr(r *http.Request, deferredFunc func() error) {
	if err := deferredFunc(); err != nil {
		log.Sugar.Errorw("error in deferred function",
			"url", r.URL,
			"err", err.Error(),
		)
	}
}
