package main

import (
	"context"
	"github.com/wminshew/emrysserver/pkg/log"
	"github.com/wminshew/emrysserver/pkg/storage"
	"io"
	"os"
	"path"
)

var (
	dockerfilePath = os.Getenv("DOCKER_PATH")
)

// downloadDockerfile downloads the main dockerfile
func downloadDockerfile(ctx context.Context) error {
	log.Sugar.Infof("Downloading dockerfile...")

	var err error
	p := dockerfilePath
	if _, err = os.Stat(p); os.IsNotExist(err) {
		if err := os.MkdirAll(path.Dir(p), 0755); err != nil {
			return err
		}
		f, err := os.Create(p)
		if err != nil {
			return nil
		}
		or, err := storage.NewReader(ctx, p)
		if err != nil {
			return err
		}
		if _, err := io.Copy(f, or); err != nil {
			return err
		}
		if err := or.Close(); err != nil {
			return err
		}
		return f.Close()
	}
	return err
}
