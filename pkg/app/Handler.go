package app

import (
	"net/http"
)

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
