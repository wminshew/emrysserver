package log

import (
	"github.com/blendle/zapdriver"
	"go.uber.org/zap"
	"net/http"
	"os"
)

var (
	appEnv = os.Getenv("APP_ENV")
	// Logger provides highly performant, strongly typed structured logging
	Logger *zap.Logger
	// Sugar provides performant weakly typed, structured logging
	Sugar *zap.SugaredLogger
)

// Init initializes Logger and Sugar
func Init() {
	var err error
	if appEnv == "dev" {
		if Logger, err = zapdriver.NewDevelopment(); err != nil {
			panic(err)
		}
	} else {
		if Logger, err = zapdriver.NewProduction(); err != nil {
			panic(err)
		}
	}
	Sugar = Logger.Sugar()
	Sugar.Infow("Initialized Logger!")
}

// Log logs request method, URL, & address
func Log(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Sugar.Infof("%s %s from %s", r.Method, r.URL, r.RemoteAddr)
		// dump, _ := httputil.DumpRequest(r, false)
		// fmt.Printf("%s", dump)
		h.ServeHTTP(w, r)
	})
}
