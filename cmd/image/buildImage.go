package main

import (
	"context"
	"docker.io/go-docker/api/types"
	"fmt"
	"github.com/cenkalti/backoff"
	"github.com/gorilla/mux"
	"github.com/mholt/archiver"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrys/pkg/check"
	"github.com/wminshew/emrys/pkg/jsonmessage"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"github.com/wminshew/emrysserver/pkg/storage"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

const maxRetries = 10

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
	uID := r.Header.Get("X-Jwt-Claims-Subject")
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
	nbQuery := r.URL.Query().Get("notebook")
	notebook := (nbQuery == "1")
	notebookStr := strconv.FormatBool(notebook)

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
	defer app.CheckErr(r, func() error { return os.RemoveAll(inputDir) })

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
		log.Sugar.Errorw("dockerfile missing",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"jID", jID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	inputDockerfilePath := filepath.Join(inputDir, "Dockerfile")
	if err := func() error {
		dockerfile, err := os.Open(dockerfilePath)
		if err != nil {
			return err
		}
		defer check.Err(dockerfile.Close)

		inputDockerfile, err := os.Create(inputDockerfilePath)
		if err != nil {
			return err
		}
		defer check.Err(inputDockerfile.Close)

		_, err = io.Copy(inputDockerfile, dockerfile)
		return err
	}(); err != nil {
		log.Sugar.Errorw("error copying dockerfile into job input dir",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"jID", jID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	if _, err := os.Stat(dockerEntrypointPath); os.IsNotExist(err) {
		log.Sugar.Errorw("entrypoint.sh missing",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"jID", jID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	inputDockerEntrypointPath := filepath.Join(inputDir, "entrypoint.sh")
	if err := func() error {
		dockerEntrypoint, err := os.Open(dockerEntrypointPath)
		if err != nil {
			return err
		}
		defer check.Err(dockerEntrypoint.Close)

		inputDockerEntrypoint, err := os.Create(inputDockerEntrypointPath)
		if err != nil {
			return err
		}
		defer check.Err(inputDockerEntrypoint.Close)

		_, err = io.Copy(inputDockerEntrypoint, dockerEntrypoint)
		return err
	}(); err != nil {
		log.Sugar.Errorw("error copying entrypoint.sh into job input dir",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"jID", jID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	main := r.Header.Get("X-Main")
	condaEnv := r.Header.Get("X-Conda-Env")
	pipReqs := r.Header.Get("X-Pip-Reqs")

	ctxFiles := []string{
		inputDockerfilePath,
		inputDockerEntrypointPath,
	}
	if main != "" {
		ctxFiles = append(ctxFiles, filepath.Join(inputDir, main))
	} else {
		main = "does-not-exist"
	}
	if condaEnv != "" {
		ctxFiles = append(ctxFiles, filepath.Join(inputDir, condaEnv))
	} else {
		condaEnv = "does-not-exist"
	}
	if pipReqs != "" {
		ctxFiles = append(ctxFiles, filepath.Join(inputDir, pipReqs))
	} else {
		pipReqs = "does-not-exist"
	}

	defer func() {
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
			defer app.CheckErr(r, ow.Close)

			if _, err = io.Copy(ow, pr); err != nil {
				return fmt.Errorf("copying pipe reader to cloud storage object writer: %v", err)
			}
			return nil
		}
		if err := backoff.RetryNotify(operation,
			backoff.WithContext(backoff.WithMaxRetries(backoff.NewExponentialBackOff(), maxRetries), ctx),
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

	// cacheSlice := []string{remoteBaseCudaRef, localBaseJobRef}
	// latestProjectBuild := fmt.Sprintf("%s/%s/%s:%s", registryHost, uUUID, project, "latest")
	// imageBuildTime[latestProjectBuild] = time.Now()
	// if pullResp, err := dClient.ImagePull(ctx, latestProjectBuild, types.ImagePullOptions{}); err != nil {
	// 	log.Sugar.Infof("error finding %s: %v", latestProjectBuild, err)
	// } else {
	// 	if err := jsonmessage.DisplayJSONMessagesStream(pullResp, os.Stdout, os.Stdout.Fd(), nil); err != nil {
	// 		log.Sugar.Errorf("error pulling %s: %v", latestProjectBuild, err)
	// 	} else {
	// 		cacheSlice = append(cacheSlice, latestProjectBuild)
	// 	}
	// 	if err := pullResp.Close(); err != nil {
	// 		log.Sugar.Errorf("error closing cache pull response %s: %v\n", latestProjectBuild, err)
	// 	}
	// }

	strRef := fmt.Sprintf("%s/%s/%s:%s", registryHost, uUUID, project, jID)
	strRefLatest := fmt.Sprintf("%s/%s/%s:%s", registryHost, uUUID, project, "latest")
	strRefMiner := fmt.Sprintf("%s/%s/%s:%s", registryHost, "miner", jID, "latest") // TODO: remove latest tag?
	strRefs := []string{strRef, strRefLatest, strRefMiner}
	operation := func() error {
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

		// for _, ref := range strRefs {
		// 	imageBuildTime[ref] = time.Now()
		// }
		cacheSlice := []string{remoteBaseCudaRef, localBaseJobRef}
		log.Sugar.Infof("Caching from: %v", cacheSlice)
		log.Sugar.Infof("Tagging as: %v", strRefs)
		buildResp, err := dClient.ImageBuild(ctx, pr, types.ImageBuildOptions{
			BuildArgs: map[string]*string{
				// "DEVPI_HOST":         &devpiHost,
				// "DEVPI_TRUSTED_HOST": &devpiTrustedHost,
				"MAIN":      &main,
				"CONDA_ENV": &condaEnv,
				"PIP_REQS":  &pipReqs,
				"NOTEBOOK":  &notebookStr,
			},
			CacheFrom:   cacheSlice,
			ForceRemove: true,
			Tags:        strRefs,
		})
		if err != nil {
			return err
		}
		defer app.CheckErr(r, buildResp.Body.Close)

		log.Sugar.Infof("Logging image build response...")
		return jsonmessage.DisplayJSONMessagesStream(buildResp.Body, os.Stdout, os.Stdout.Fd(), nil)
	}
	if err := backoff.RetryNotify(operation,
		backoff.WithContext(backoff.WithMaxRetries(backoff.NewExponentialBackOff(), maxRetries), ctx),
		func(err error, t time.Duration) {
			log.Sugar.Errorw("error building image--retrying",
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
		}); err != nil {
		log.Sugar.Errorw("error building image--abort",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"jID", jID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	defer func() {
		// wait to remove image, in case job fails & is promptly re-run
		time.Sleep(5 * time.Minute) // TODO: make wait period ENV

		log.Sugar.Infof("Removing images %v", strRefs)
		for _, ref := range strRefs {
			if _, err := dClient.ImageRemove(ctx, ref, types.ImageRemoveOptions{
				Force: true,
			}); err != nil {
				log.Sugar.Errorw("error removing image",
					"method", r.Method,
					"url", r.URL,
					"err", err.Error(),
					"jID", jID,
					"ref", ref,
				)
				return
			}
		}

		log.Sugar.Infof("Pruning build cache")
		if _, err := dClient.BuildCachePrune(ctx); err != nil {
			log.Sugar.Errorw("error pruning build cache: %v",
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return
		}
	}()

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
