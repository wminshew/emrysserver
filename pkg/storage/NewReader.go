package storage

import (
	"cloud.google.com/go/storage"
	"context"
)

// ErrObjectNotExist returned if object doesn't exist
var ErrObjectNotExist = storage.ErrObjectNotExist

// NewReader returns a reader for downloading objects from the emrys-dev bucket
func NewReader(ctx context.Context, p string) (*storage.Reader, error) {
	return bkt.Object(p).NewReader(ctx)
}
