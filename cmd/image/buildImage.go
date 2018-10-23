package main

import (
	"context"
	"docker.io/go-docker/api/types"
	"fmt"
	"github.com/cenkalti/backoff"
	"github.com/gorilla/mux"
	"github.com/mholt/archiver"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrys/pkg/jsonmessage"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"github.com/wminshew/emrysserver/pkg/storage"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// buildImage handles building images for jobs posted by users
var buildImage app.Handler = func(w http.ResponseWriter, r *http.Request) *app.Error {
	vars := mux.Vars(r)
	project := vars["project"]
	jID := vars["jID"]
	jUUID, err := uuid.FromString(jID)
	if err != nil {
		log.Sugar.Errorw("error parsing job ID",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "error parsing job ID"}
	}
	if t, err := db.GetStatusImageBuilt(r, jUUID); err != nil {
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"} // already logged err
	} else if !t.IsZero() {
		log.Sugar.Infow("user tried to re-build image",
			"method", r.Method,
			"url", r.URL,
			"jID", jID,
		)
		return nil
	}
	uID := vars["uID"]
	uUUID, err := uuid.FromString(uID)
	if err != nil {
		log.Sugar.Errorw("error parsing job ID",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"jID", jID,
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "error parsing job ID"}
	}

	inputDir := filepath.Join("job", jID, "input")
	if err := os.MkdirAll(inputDir, 0755); err != nil {
		log.Sugar.Errorw("error creating job input directory",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"jID", jID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	log.Sugar.Infof("Storing input files on disk...")
	if err := archiver.TarGz.Read(r.Body, inputDir); err != nil {
		log.Sugar.Errorw("error un-targzpping request body to input dir",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"jID", jID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	ctx := r.Context()
	if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
		if err := downloadDockerfile(ctx); err != nil {
			log.Sugar.Errorw("error downloading dockerfile",
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}
	}

	linkedDocker := filepath.Join(inputDir, "Dockerfile")
	if _, err := os.Stat(linkedDocker); os.IsNotExist(err) {
		if err := os.Link(dockerfilePath, linkedDocker); err != nil {
			log.Sugar.Errorw("error linking dockerfile into user dir",
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}
	}

	main := r.Header.Get("X-Main")
	reqs := r.Header.Get("X-Reqs")
	ctxFiles := []string{
		filepath.Join(inputDir, main),
		filepath.Join(inputDir, reqs),
		filepath.Join(inputDir, "Dockerfile"),
	}

	defer func() {
		defer app.CheckErr(r, func() error { return os.RemoveAll(inputDir) })
		ctx := context.Background()
		operation := func() error {
			pr, pw := io.Pipe()
			go func() {
				defer app.CheckErr(r, pw.Close)
				if err := archiver.TarGz.Write(pw, ctxFiles); err != nil {
					log.Sugar.Errorw("error tar-gzipping docker context for cloud storage",
						"method", r.Method,
						"url", r.URL,
						"err", err.Error(),
						"jID", jID,
					)
					return
				}
			}()

			p := filepath.Join("image", jID, "dockerContext.tar.gz")
			ow := storage.NewWriter(ctx, p)
			if _, err = io.Copy(ow, pr); err != nil {
				return fmt.Errorf("copying pipe reader to cloud storage object writer: %v", err)
			}
			if err = ow.Close(); err != nil {
				return fmt.Errorf("closing cloud storage object writer: %v", err)
			}
			return nil
		}
		if err := backoff.RetryNotify(operation,
			backoff.WithContext(backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 10), ctx),
			func(err error, t time.Duration) {
				log.Sugar.Errorw("error uploading input dockerContext.tar.gz to gcs--retrying",
					"method", r.Method,
					"url", r.URL,
					"err", err.Error(),
					"jID", jID,
				)
			}); err != nil {
			log.Sugar.Errorw("error uploading input dockerContext.tar.gz to gcs--abort",
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return
		}
	}()

	cacheSlice := []string{dockerBaseCudaRef, localBaseJobRef}
	latestProjectBuild := fmt.Sprintf("%s/%s/%s:%s", registryHost, uUUID, project, "latest")
	imageBuildTime[latestProjectBuild] = time.Now()
	if pullResp, err := dClient.ImagePull(ctx, latestProjectBuild, types.ImagePullOptions{}); err != nil {
		log.Sugar.Infof("error finding %s: %v", latestProjectBuild, err)
	} else {
		if err := jsonmessage.DisplayJSONMessagesStream(pullResp, os.Stdout, os.Stdout.Fd(), nil); err != nil {
			log.Sugar.Errorf("error pulling %s: %v", latestProjectBuild, err)
		} else {
			cacheSlice = append(cacheSlice, latestProjectBuild)
		}
		if err := pullResp.Close(); err != nil {
			log.Sugar.Errorf("error closing cache pull response %s: %v\n", latestProjectBuild, err)
		}
	}

	log.Sugar.Infof("Sending ctxFiles to docker daemon...")
	pr, pw := io.Pipe()
	go func() {
		defer app.CheckErr(r, pw.Close)
		if err := archiver.TarGz.Write(pw, ctxFiles); err != nil {
			log.Sugar.Errorw("error tar-gzipping docker context",
				"method", r.Method,
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
	for _, ref := range strRefs {
		imageBuildTime[ref] = time.Now()
	}
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
		log.Sugar.Errorw("error building image",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"jID", jID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}
	defer app.CheckErr(r, buildResp.Body.Close)

	log.Sugar.Infof("Logging image build response...")
	if err := jsonmessage.DisplayJSONMessagesStream(buildResp.Body, os.Stdout, os.Stdout.Fd(), nil); err != nil {
		log.Sugar.Errorw("error building image",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"jID", jID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	for _, ref := range strRefs {
		pushAddr := ref
		log.Sugar.Infof("Pushing %s...", pushAddr)
		pushResp, err := dClient.ImagePush(ctx, pushAddr, types.ImagePushOptions{
			RegistryAuth: "none",
		})
		if err != nil {
			log.Sugar.Errorw("error pushing image",
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
				"pushAddr", pushAddr,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}
		defer app.CheckErr(r, pushResp.Close)

		if err := jsonmessage.DisplayJSONMessagesStream(pushResp, os.Stdout, os.Stdout.Fd(), nil); err != nil {
			log.Sugar.Errorw("error pushing image",
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}
	}

	return db.SetStatusImageBuilt(r, jUUID)
}
