package main

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrys/pkg/job"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
)

// syncUser receives the map of the user's data set metadata and determines which files needed to be re-uploaded
func syncUser() app.Handler {
	return func(w http.ResponseWriter, r *http.Request) *app.Error {
		vars := mux.Vars(r)
		jID := vars["jID"]
		jUUID, err := uuid.FromString(jID)
		if err != nil {
			log.Sugar.Errorw("error parsing job ID",
				"url", r.URL,
				"err", err.Error(),
			)
			return &app.Error{Code: http.StatusBadRequest, Message: "error parsing job ID"}
		}

		uID := vars["uID"]
		_, err = uuid.FromString(uID)
		if err != nil {
			log.Sugar.Errorw("error parsing user ID",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return &app.Error{Code: http.StatusBadRequest, Message: "error parsing user ID"}
		}

		project := vars["project"]
		projectDir := filepath.Join("data", uID, project)
		if _, err = os.Stat(projectDir); os.IsNotExist(err) {
			if err := os.MkdirAll(projectDir, 0755); err != nil {
				log.Sugar.Errorw("error retrieving server project metadata",
					"url", r.URL,
					"err", err.Error(),
					"jID", jID,
				)
				return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
			}
			if err := downloadProject(projectDir); err != nil {
				log.Sugar.Errorw("error downloading project from gcs",
					"url", r.URL,
					"err", err.Error(),
					"jID", jID,
				)
				return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
			}
			go func() {
				if err := checkAndEvictProjects(); err != nil {
					log.Sugar.Errorf("Error managing disk utilization: %v\n", err)
					return
				}
			}()
		} else if err != nil {
			log.Sugar.Errorw("error retrieving project directory",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}
		go func() {
			touchProjectMd(projectDir)
		}()

		serverMetadata := make(map[string]*job.FileMetadata)
		if err := getProjectMetadata(r, uID, project, &serverMetadata); err != nil {
			log.Sugar.Errorw("error retrieving server project metadata",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}
		defer func() {
			if err := storeProjectMetadata(r, uID, project, &serverMetadata); err != nil {
				log.Sugar.Errorw("error storing project metatdata",
					"url", r.URL,
					"err", err.Error(),
					"jID", jID,
				)
				return
			}
		}()

		userMetadata := make(map[string]*job.FileMetadata)
		if err := json.NewDecoder(r.Body).Decode(&userMetadata); err != nil && err != io.EOF {
			log.Sugar.Errorw("error decoding user project metadata from request body",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return &app.Error{Code: http.StatusBadRequest, Message: "error parsing user metadata"}
		}

		uIDProject := path.Join(uID, project)
		mdSync[uIDProject] = make(map[string]*job.FileMetadata)
		uploadList := []string{}
		keepList := make(map[string]bool)
		dataDir := filepath.Join("data", uID, project, "data")
		for relPath, userFileMd := range userMetadata {
			serverFileMd, ok := serverMetadata[relPath]
			if !ok {
				uploadList = append(uploadList, relPath)
				mdSync[uIDProject][relPath] = userFileMd
				continue
			}
			if serverFileMd.ModTime == userFileMd.ModTime {
				keepList[relPath] = true
			} else if serverFileMd.Hash == userFileMd.Hash {
				keepList[relPath] = true
				serverFileMd.ModTime = userFileMd.ModTime
			} else {
				uploadList = append(uploadList, relPath)
				mdSync[uIDProject][relPath] = userFileMd
				p := filepath.Join(dataDir, relPath)
				if err := os.Remove(p); err != nil {
					log.Sugar.Errorw("error removing data set file",
						"url", r.URL,
						"err", err.Error(),
						"jID", jID,
						"project", project,
						"path", relPath,
					)
					return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
				}
				delete(serverMetadata, relPath)
			}
		}

		for relPath := range serverMetadata {
			if _, ok := keepList[relPath]; !ok {
				p := filepath.Join(dataDir, relPath)
				if err := os.Remove(p); err != nil {
					log.Sugar.Errorw("error removing data set file",
						"url", r.URL,
						"err", err.Error(),
						"jID", jID,
						"project", project,
						"path", relPath,
					)
					return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
				}
				delete(serverMetadata, relPath)
			}
		}

		if len(uploadList) == 0 {
			return db.SetStatusDataSynced(r, jUUID)
		}

		if err := json.NewEncoder(w).Encode(uploadList); err != nil {
			log.Sugar.Errorw("error encoding upload list as json",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}

		return nil
	}
}
