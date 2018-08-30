package storage

import (
	"cloud.google.com/go/storage"
	"context"
	"github.com/wminshew/emrysserver/pkg/log"
)

const (
	bktName = "emrys-dev"
)

var bkt *storage.BucketHandle

// Init initializes the google cloud storage client with access to the emrys-dev bucket
func Init() {
	log.Sugar.Infof("Initializing cloud storage...")

	if client, err := storage.NewClient(context.Background()); err != nil {
		log.Sugar.Errorf("error initializing gcs: %v", err)
		panic(err)
	} else {
		bkt = client.Bucket(bktName)
	}
}
