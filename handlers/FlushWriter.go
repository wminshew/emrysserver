package handlers

import (
	"io"
	"net/http"
)

// flushWriter streams responses to io.Writer instead of
// buffering until the request is fully processed.
//
// source: https://github.com/bmorton/flushwriter/blob/master/flush_writer.go
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
