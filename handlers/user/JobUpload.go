package user

import (
	"context"
	"docker.io/go-docker"
	"docker.io/go-docker/api/types"
	"encoding/json"
	"fmt"
	"github.com/mholt/archiver"
	"github.com/satori/go.uuid"
	"github.com/wminshew/check"
	"github.com/wminshew/emrys/pkg/job"
	"github.com/wminshew/emrysserver/db"
	"github.com/wminshew/emrysserver/handlers/miner"
	"github.com/wminshew/emrysserver/pkg/flushwriter"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

// JobUpload handles job posted by user
func JobUpload(w http.ResponseWriter, r *http.Request) {
	maxMemory := int64(1) << 31
	err := r.ParseMultipartForm(maxMemory)
	if err != nil {
		log.Printf("Error parsing request: %v\n", err)
		http.Error(w, "Internal error.", http.StatusInternalServerError)
		return
	}
	fw := flushwriter.New(w)
	_, err = fw.Write([]byte("Unpacking request...\n"))
	if err != nil {
		log.Printf("Error writing to flushWriter: %v\n", err)
	}

	ctxKey := contextKey("user_uuid")
	uUUID, ok := r.Context().Value(ctxKey).(uuid.UUID)
	if !ok {
		log.Printf("user_uuid in request context corrupted\n")
		http.Error(w, "Unable to retrieve valid uuid from jwt. Please login again.", http.StatusInternalServerError)
		return
	}
	uname := uUUID.String()

	// TODO: re-factor job processing; take out file saving, add relevant paths to r.context
	// TODO: add extra directory layer for project/job number (git vcs?); return job number to client
	// TODO: use s3 or something else?
	userDir := filepath.Join("user-upload", uname)
	if err = os.MkdirAll(userDir, 0755); err != nil {
		log.Printf("Error creating user directory %s: %v\n", userDir, err)
		http.Error(w, "Internal error.", http.StatusInternalServerError)
		return
	}

	requirementsTempFile, requirementsHeader, err := r.FormFile("requirements")
	if err != nil {
		log.Printf("Error reading requirements form file: %v\n", err)
		http.Error(w, "Internal error.", http.StatusInternalServerError)
		return
	}
	defer check.Err(requirementsTempFile.Close)
	requirementsPath := filepath.Join(userDir, filepath.Base(requirementsHeader.Filename))
	requirementsFile, err := os.OpenFile(requirementsPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Printf("Error opening requirements file: %v\n", err)
		http.Error(w, "Internal error.", http.StatusInternalServerError)
		return
	}
	defer check.Err(requirementsFile.Close)
	_, err = io.Copy(requirementsFile, requirementsTempFile)
	if err != nil {
		log.Printf("Error copying requirements file to disk: %v\n", err)
		http.Error(w, "Internal error.", http.StatusInternalServerError)
		return
	}

	trainTempFile, trainHeader, err := r.FormFile("train")
	if err != nil {
		log.Printf("Error reading train form file: %v\n", err)
		http.Error(w, "Internal error.", http.StatusInternalServerError)
		return
	}
	defer check.Err(trainTempFile.Close)
	trainPath := filepath.Join(userDir, filepath.Base(trainHeader.Filename))
	trainFile, err := os.OpenFile(trainPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		log.Printf("Error opening train file: %v\n", err)
		http.Error(w, "Internal error.", http.StatusInternalServerError)
		return
	}
	defer check.Err(trainFile.Close)
	_, err = io.Copy(trainFile, trainTempFile)
	if err != nil {
		log.Printf("Error copying train file to disk: %v\n", err)
		http.Error(w, "Internal error.", http.StatusInternalServerError)
		return
	}

	dataTempFile, dataHeader, err := r.FormFile("data")
	if err != nil {
		log.Printf("Error reading data form file: %v\n", err)
		http.Error(w, "Internal error.", http.StatusInternalServerError)
		return
	}
	defer check.Err(dataTempFile.Close)
	dataPath := filepath.Join(userDir, filepath.Base(dataHeader.Filename))
	dataFile, err := os.OpenFile(dataPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Printf("Error opening data file: %v\n", err)
		http.Error(w, "Internal error.", http.StatusInternalServerError)
		return
	}
	defer check.Err(dataFile.Close)
	defer check.Err(func() error { return os.Remove(dataPath) })
	_, err = io.Copy(dataFile, dataTempFile)
	if err != nil {
		log.Printf("Error copying data file to disk: %v\n", err)
		http.Error(w, "Internal error.", http.StatusInternalServerError)
		return
	}
	// TODO: consider whether to save down data at all; maybe just proxy pipe to miner
	// TODO: need to remove old data dir contents / properly manage data update from
	// last job using git lfs or rsync or something
	err = archiver.TarGz.Open(dataPath, userDir)
	if err != nil {
		log.Printf("Error unzipping data dir: %v\n", err)
		http.Error(w, "Internal error.", http.StatusInternalServerError)
		return
	}

	_, err = fw.Write([]byte("Beginning miner auction for job...\n"))
	if err != nil {
		log.Printf("Error writing to flushWriter: %v\n", err)
	}
	jobID := uuid.NewV4()
	j := &job.Job{
		ID:     jobID,
		UserID: uUUID,
	}
	log.Printf("Auctioning job: %v\n", j.ID)
	if _, err = db.Db.Query("INSERT INTO jobs (job_uuid, user_uuid) VALUES ($1, $2)",
		j.ID, j.UserID); err != nil {
		log.Printf("Error inserting job into db: %v\n", err)
		http.Error(w, "Internal error.", http.StatusInternalServerError)
		return
	}
	go miner.Pool.AuctionJob(&job.Job{
		ID: j.ID,
	})

	_, err = fw.Write([]byte("Building image...\n"))
	if err != nil {
		log.Printf("Error writing to flushWriter: %v\n", err)
	}

	ctx := context.Background()
	cli, err := docker.NewEnvClient()
	if err != nil {
		log.Printf("Error creating new docker client: %v\n", err)
		http.Error(w, "Internal error.", http.StatusInternalServerError)
		return
	}

	linkedDocker := filepath.Join(userDir, "Dockerfile")
	userDockerfile := filepath.Join("Dockerfiles", "Dockerfile.user")
	err = os.Link(userDockerfile, linkedDocker)
	if err != nil {
		log.Printf("Error linking dockerfile into user directory: %v\n", err)
		http.Error(w, "Internal error.", http.StatusInternalServerError)
		return
	}
	defer check.Err(func() error { return os.Remove(linkedDocker) })
	buildCtxPath := userDir + ".tar.gz"
	ctxFiles, err := filepath.Glob(filepath.Join(userDir, "/*"))
	if err != nil {
		log.Printf("Error collecting docker context files: %v\n", err)
		http.Error(w, "Internal error.", http.StatusInternalServerError)
		return
	}
	err = archiver.TarGz.Make(buildCtxPath, ctxFiles)
	if err != nil {
		log.Printf("Error archiving docker context files: %v\n", err)
		http.Error(w, "Internal error.", http.StatusInternalServerError)
		return
	}
	defer check.Err(func() error { return os.Remove(buildCtxPath) })
	buildCtx, err := os.Open(buildCtxPath)
	if err != nil {
		log.Printf("Error opening archived docker context files: %v\n", err)
		http.Error(w, "Internal error.", http.StatusInternalServerError)
		return
	}
	userHome := "/home/user"
	// TODO: ImageBuild isn't throwing an error if it can't find its FROM img?
	buildResp, err := cli.ImageBuild(ctx, buildCtx, types.ImageBuildOptions{
		BuildArgs: map[string]*string{
			"HOME": &userHome,
		},
		ForceRemove: true,
		// TODO: tags/labels for emrys/project/job/user?
		Tags: []string{j.ID.String()},
		// Labels: map[string]string{}
	})
	if err != nil {
		log.Printf("Error building image: %v\n", err)
		http.Error(w, "Internal error.", http.StatusInternalServerError)
		return
	}
	defer check.Err(buildResp.Body.Close)

	printJSONStream(buildResp.Body)

	_, err = fw.Write([]byte("Image built!\n"))
	if err != nil {
		log.Printf("Error writing to flushWriter: %v\n", err)
	}
	// img, err := cli.ImageSave(ctx, []string{j.ID.String()})

	// sync image build with job auction
	// TODO: wg?

	// TODO: insert job into DB?
	// j.UserID = uUUID

	_, err = fw.Write([]byte("Sending image and data to winning bidder...\n"))
	if err != nil {
		log.Printf("Error writing to flushWriter: %v\n", err)
	}

	// TODO: Send data to miner
}

func printJSONStream(r io.Reader) {
	type stream struct {
		stream string
	}
	dec := json.NewDecoder(r)
	for dec.More() {
		var s stream
		err := dec.Decode(&s)
		if err != nil {
			log.Printf("Error decoding json build stream: %v\n", err)
		}

		fmt.Printf("%v", s.stream)
	}
}
