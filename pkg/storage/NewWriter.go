package storage

import (
	"cloud.google.com/go/storage"
	"context"
)

// NewWriter returns a writer for uploading objects to the emrys-dev bucket
func NewWriter(ctx context.Context, p string) *storage.Writer {
	return bkt.Object(p).NewWriter(ctx)
}
