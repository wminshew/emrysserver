package main

import (
	"context"
	"docker.io/go-docker/api/types"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/mholt/archiver"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrys/pkg/jsonmessage"
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
		project := vars["project"]
		jID := vars["jID"]
		jUUID, err := uuid.FromString(jID)
		if err != nil {
			log.Sugar.Errorw("failed to parse job ID",
				"url", r.URL,
				"err", err.Error(),
			)
			return &app.Error{Code: http.StatusBadRequest, Message: "error parsing job ID"}
		}
		uID := vars["uID"]
		uUUID, err := uuid.FromString(uID)
		if err != nil {
			log.Sugar.Errorw("failed to parse job ID",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
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

		// ctx := r.Context() // TODO: why does this throw errors with storage read and docker build?
		ctx := context.Background()
		cacheSlice := []string{dockerBaseCudaRef, localBaseJobRef}
		latestProjectBuild := fmt.Sprintf("%s/%s/%s:%s", registryHost, uUUID, project, "latest")
		if pullResp, err := dClient.ImagePull(ctx, latestProjectBuild, types.ImagePullOptions{}); err != nil {
			log.Sugar.Infof("failed to find %s: %v", latestProjectBuild, err)
		} else {
			if err := jsonmessage.DisplayJSONMessagesStream(pullResp, os.Stdout, os.Stdout.Fd(), nil); err != nil {
				log.Sugar.Errorf("failed to pull %s: %v", latestProjectBuild, err)
			} else {
				cacheSlice = append(cacheSlice, latestProjectBuild)
			}
			if err := pullResp.Close(); err != nil {
				log.Sugar.Errorf("failed to close cache pull response %s: %v\n", latestProjectBuild, err)
			}
		}

		log.Sugar.Infof("Sending ctxFiles to docker daemon...")
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

		strRef := fmt.Sprintf("%s/%s/%s:%s", registryHost, uUUID, project, jID)
		strRefLatest := fmt.Sprintf("%s/%s/%s:%s", registryHost, uUUID, project, "latest")
		strRefMiner := fmt.Sprintf("%s/%s/%s:%s", registryHost, "miner", jID, "latest")
		strRefs := []string{strRef, strRefLatest, strRefMiner}
		log.Sugar.Infof("Caching from: %v", cacheSlice)
		log.Sugar.Infof("Tagging as: %v", strRefs)
		buildResp, err := dClient.ImageBuild(ctx, pr, types.ImageBuildOptions{
			BuildArgs: map[string]*string{
				"DEVPI_HOST":         &devpiHost,
				"DEVPI_TRUSTED_HOST": &devpiTrustedHost,
				"MAIN":               &main,
				"REQS":               &reqs,
			},
			CacheFrom:      cacheSlice,
			ForceRemove:    true,
			SuppressOutput: true,
			Tags:           strRefs,
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
		if err := jsonmessage.DisplayJSONMessagesStream(buildResp.Body, os.Stdout, os.Stdout.Fd(), nil); err != nil {
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

		for _, ref := range strRefs {
			pushAddr := ref
			log.Sugar.Infof("Pushing %s...", pushAddr)
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

			if err := jsonmessage.DisplayJSONMessagesStream(pushResp, os.Stdout, os.Stdout.Fd(), nil); err != nil {
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
		}

		return db.SetStatusImageBuilt(r, jUUID)
	}
}
