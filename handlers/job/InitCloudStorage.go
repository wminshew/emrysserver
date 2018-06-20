package job

import (
	"cloud.google.com/go/storage"
	"context"
	"log"
)

const (
	bktName = "emrys-dev"
)

var outputBkt *storage.BucketHandle

// InitCloudStorage initializes the google cloud storage client
func InitCloudStorage() {
	log.Printf("Initializing cloud storage...\n")

	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		panic(err)
	}

	outputBkt = client.Bucket(bktName)
}
