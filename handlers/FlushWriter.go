package handlers

import (
	"io"
	"net/http"
)

// FlushWriter streams responses to io.Writer instead of
// buffering until the request is fully processed.
//
// source: https://github.com/bmorton/flushwriter/blob/master/flush_writer.go
type FlushWriter struct {
	flusher http.Flusher
	writer  io.Writer
}

// Write satisifies the io.Writer interface so that flushWriter can wrap the
// supplied io.Writer with a Flusher.
func (fw *FlushWriter) Write(p []byte) (n int, err error) {
	n, err = fw.writer.Write(p)
	if fw.flusher != nil {
		fw.flusher.Flush()
	}
	return n, err
}

// NewFlushWriter creates a FlushWriter using the io.Writer provided as the writer and flusher.
func NewFlushWriter(w io.Writer) *FlushWriter {
	fw := &FlushWriter{writer: w}
	if f, ok := w.(http.Flusher); ok {
		fw.flusher = f
	}

	return fw
}
