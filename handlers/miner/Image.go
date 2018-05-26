package miner

import (
	"compress/zlib"
	"context"
	"docker.io/go-docker"
	"github.com/gorilla/mux"
	"github.com/wminshew/emrys/pkg/check"
	"io"
	"log"
	"net/http"
)

// Image sends the {jID} job docker image to the miner for execution
func Image(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jID := vars["jID"]

	ctx := context.Background()
	cli, err := docker.NewEnvClient()
	if err != nil {
		log.Printf("Error creating new docker client: %v\n", err)
		http.Error(w, "Internal error.", http.StatusInternalServerError)
		return
	}
	img, err := cli.ImageSave(ctx, []string{jID})
	defer check.Err(img.Close)

	zw := zlib.NewWriter(w)
	defer check.Err(zw.Close)

	_, err = io.Copy(zw, img)
	if err != nil {
		log.Printf("Error copying img to zlib response writer: %v\n", err)
		return
	}
}
