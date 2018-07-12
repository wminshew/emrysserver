package app

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

// InitLogger initializes Logger and Sugar
func InitLogger() {
	var err error
	if appEnv == "dev" {
		// if Logger, err = zap.NewDevelopment(); err != nil {
		if Logger, err = zapdriver.NewDevelopment(); err != nil {
			panic(err)
		}
	} else {
		// if Logger, err = zap.NewProduction(); err != nil {
		if Logger, err = zapdriver.NewProduction(); err != nil {
			panic(err)
		}
	}
	Sugar = Logger.Sugar()
	Sugar.Infow("Initialized Logger!")
}

// source: https://blog.golang.org/error-handling-and-go
// source: https://mwholt.blogspot.com/2015/05/handling-errors-in-http-handlers-in-go.html

// Handler in pkg app replaces http.Handler to allow for better error handling
type Handler func(http.ResponseWriter, *http.Request) *Error

// Error in pkg app replaces os.Error to allow for better error handling
type Error struct {
	Code    int
	Message string
}

// ServeHTTP on Handler allows app.Handler to be converted to http.Handler
func (fn Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if e := fn(w, r); e != nil { // e is *appError, not os.Error
		http.Error(w, e.Message, e.Code)
	}
}
