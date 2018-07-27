package main

import (
	"context"
	"docker.io/go-docker/api/types"
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
		if err = os.MkdirAll(inputDir, 0755); err != nil {
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

		// TODO: why does this throw errors with storage read and docker build?
		// ctx := r.Context()
		ctx := context.Background()
		if err = archiver.TarGz.Read(r.Body, inputDir); err != nil {
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

		dockerDir := filepath.Join("Dockerfiles")
		userDockerfile := filepath.Join(dockerDir, "Dockerfile")
		if _, err = os.Stat(userDockerfile); os.IsNotExist(err) {
			if err = func() error {
				if err = os.MkdirAll(dockerDir, 0755); err != nil {
					return err
				}
				f, err := os.Create(userDockerfile)
				if err != nil {
					return nil
				}
				or, err := bkt.Object(userDockerfile).NewReader(ctx)
				if err != nil {
					return err
				}
				if _, err = io.Copy(f, or); err != nil {
					return err
				}
				if err = or.Close(); err != nil {
					return err
				}
				if err = f.Close(); err != nil {
					return err
				}
				return nil
			}(); err != nil {
				log.Sugar.Errorw("failed to download dockerfile",
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

		linkedDocker := filepath.Join(inputDir, "Dockerfile")
		if err = os.Link(userDockerfile, linkedDocker); err != nil {
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

		userHome := "/home/user"
		buildResp, err := dClient.ImageBuild(ctx, pr, types.ImageBuildOptions{
			BuildArgs: map[string]*string{
				"HOME": &userHome,
				"MAIN": &main,
				"REQS": &reqs,
			},
			ForceRemove: true,
			Tags:        []string{jID},
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

		return db.SetStatusImageBuilt(r, jUUID)
	}
}
