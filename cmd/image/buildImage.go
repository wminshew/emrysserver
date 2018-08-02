package main

import (
	"context"
	"docker.io/go-docker/api/types"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/mholt/archiver"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrys/pkg/job"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// buildImage handles building images for jobs posted by users
func buildImage() app.Handler {
	return func(w http.ResponseWriter, r *http.Request) *app.Error {
		vars := mux.Vars(r)
		jID := vars["jID"]
		jUUID, err := uuid.FromString(jID)
		if err != nil {
			log.Sugar.Errorw("failed to parse job ID",
				"url", r.URL,
				"err", err.Error(),
			)
			return &app.Error{Code: http.StatusBadRequest, Message: "error parsing job ID"}
		}

		inputDir := filepath.Join("job", jID, "input")
		if err := os.MkdirAll(inputDir, 0755); err != nil {
			log.Sugar.Errorw("failed to create job input directory",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			if err := db.SetJobInactive(r, jUUID); err != nil {
				log.Sugar.Errorf("Error setting job %v inactive: %v\n", jUUID, err)
			}
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}

		log.Sugar.Infof("Storing input files on disk...")
		if err := archiver.TarGz.Read(r.Body, inputDir); err != nil {
			log.Sugar.Errorw("failed to un-targz request body to input dir",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			if err := db.SetJobInactive(r, jUUID); err != nil {
				log.Sugar.Errorf("Error setting job %v inactive: %v\n", jUUID, err)
			}
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}

		linkedDocker := filepath.Join(inputDir, "Dockerfile")
		if err := os.Link(dockerfilePath, linkedDocker); err != nil {
			log.Sugar.Errorw("failed link dockerfile into user dir",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			if err := db.SetJobInactive(r, jUUID); err != nil {
				log.Sugar.Errorf("Error setting job %v inactive: %v\n", jUUID, err)
			}
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}

		main := r.Header.Get("X-Main")
		reqs := r.Header.Get("X-Reqs")
		ctxFiles := []string{
			filepath.Join(inputDir, main),
			filepath.Join(inputDir, reqs),
			filepath.Join(inputDir, "Dockerfile"),
		}
		pr, pw := io.Pipe()
		go func() {
			defer app.CheckErr(r, pw.Close)
			if err := archiver.TarGz.Write(pw, ctxFiles); err != nil {
				log.Sugar.Errorw("failed to tar-gzip docker context",
					"url", r.URL,
					"err", err.Error(),
					"jID", jID,
				)
				return
			}
		}()

		log.Sugar.Infof("Sending ctxFiles to docker daemon...")
		// TODO: why does this throw errors with storage read and docker build?
		// ctx := r.Context()
		ctx := context.Background()
		strRef := fmt.Sprintf("%s/%s", registryHost, jID)
		log.Sugar.Infof("Cache-from %s %s\n", dockerBaseCudaRef, localBaseJobRef)
		buildResp, err := dClient.ImageBuild(ctx, pr, types.ImageBuildOptions{
			BuildArgs: map[string]*string{
				"MAIN": &main,
				"REQS": &reqs,
			},
			CacheFrom:   []string{dockerBaseCudaRef, localBaseJobRef},
			ForceRemove: true,
			Tags:        []string{strRef},
		})
		if err != nil {
			log.Sugar.Errorw("failed to build image",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			if err := db.SetJobInactive(r, jUUID); err != nil {
				log.Sugar.Errorf("Error setting job %v inactive: %v\n", jUUID, err)
			}
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}
		defer app.CheckErr(r, buildResp.Body.Close)

		log.Sugar.Infof("Logging image build response...")
		if err := job.ReadJSON(buildResp.Body); err != nil {
			log.Sugar.Errorw("failed to build image",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			if err := db.SetJobInactive(r, jUUID); err != nil {
				log.Sugar.Errorf("Error setting job %v inactive: %v\n", jUUID, err)
			}
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}

		// TODO: tag and push image with mID
		// pushAddr := fmt.Sprintf("%s/%s/%s", registryHost, mID, jID)
		pushAddr := strRef
		pushResp, err := dClient.ImagePush(ctx, pushAddr, types.ImagePushOptions{
			RegistryAuth: "none",
		})
		if err != nil {
			log.Sugar.Errorw("failed to push image",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
				"pushAddr", pushAddr,
			)
			if err := db.SetJobInactive(r, jUUID); err != nil {
				log.Sugar.Errorf("Error setting job %v inactive: %v\n", jUUID, err)
			}
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}
		defer app.CheckErr(r, pushResp.Close)

		log.Sugar.Infof("Logging image push response...")
		if err := job.ReadJSON(pushResp); err != nil {
			log.Sugar.Errorw("failed to push image",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			if err := db.SetJobInactive(r, jUUID); err != nil {
				log.Sugar.Errorf("Error setting job %v inactive: %v\n", jUUID, err)
			}
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}

		return db.SetStatusImageBuilt(r, jUUID)
	}
}
