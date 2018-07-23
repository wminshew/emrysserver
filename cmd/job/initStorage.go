package main

import (
	"cloud.google.com/go/storage"
	"context"
	"github.com/wminshew/emrysserver/pkg/app"
)

const (
	bktName = "emrys-dev"
)

var bkt *storage.BucketHandle

// initStorage initializes the google cloud storage client
func initStorage() {
	app.Sugar.Infof("Initializing job storage...")

	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		app.Sugar.Errorf("Cloud storage failed to initialize! Panic!")
		panic(err)
	}
	bkt = client.Bucket(bktName)
}
