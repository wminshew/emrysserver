package main

import (
	"cloud.google.com/go/storage"
	"context"
	"github.com/wminshew/emrysserver/pkg/log"
)

const (
	bktName = "emrys-dev"
)

var bkt *storage.BucketHandle

// initStorage initializes the google cloud storage client for user nodes
func initStorage() {
	log.Sugar.Infof("Initializing user storage...")

	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Sugar.Errorf("Cloud storage failed to initialize! Panic!")
		panic(err)
	}
	bkt = client.Bucket(bktName)
}
