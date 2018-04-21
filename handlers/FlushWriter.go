package handlers

import (
	"io"
	"net/http"
)

// source: https://github.com/bmorton/flushwriter/blob/master/flush_writer.go
//
// flushWriter is used to stream responses to the provided io.Writer instead of
// buffering and sending in blocks once the request is fully processed.
//
// The implementation comes from this StackOverflow post:
// http://stackoverflow.com/questions/19292113/not-buffered-http-responsewritter-in-golang
type flushWriter struct {
	flusher http.Flusher
	writer  io.Writer
}

// Write satisifies the io.Writer interface so that flushWriter can wrap the
// supplied io.Writer with a Flusher.
func (fw flushWriter) Write(p []byte) (n int, err error) {
	n, err = fw.writer.Write(p)
	if fw.flusher != nil {
		fw.flusher.Flush()
	}
	return n, err
}

// newflushWriter creates a flushWriter using the io.Writer provided as the
// writer and flusher.
func newFlushWriter(w io.Writer) flushWriter {
	fw := flushWriter{writer: w}
	if f, ok := w.(http.Flusher); ok {
		fw.flusher = f
	}

	return fw
}
