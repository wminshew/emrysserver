package user

import (
	"context"
	"docker.io/go-docker"
	"docker.io/go-docker/api/types"
	"encoding/json"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/mholt/archiver"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrys/pkg/check"
	"github.com/wminshew/emrys/pkg/job"
	"github.com/wminshew/emrysserver/db"
	"github.com/wminshew/emrysserver/handlers/miner"
	"github.com/wminshew/emrysserver/pkg/flushwriter"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// PostJob handles new jobs posted by users
func PostJob(w http.ResponseWriter, r *http.Request) {
	maxMemory := int64(1) << 31
	err := r.ParseMultipartForm(maxMemory)
	if err != nil {
		log.Printf("Error parsing request: %v\n", err)
		http.Error(w, "Internal error! Please try again, and if the problem continues contact support.", http.StatusInternalServerError)
		return
	}

	ctxKey := contextKey("user_uuid")
	uUUID, ok := r.Context().Value(ctxKey).(uuid.UUID)
	if !ok {
		log.Printf("user_uuid in request context corrupted.\n")
		http.Error(w, "Unable to retrive valid uuid from jwt. Please login again.", http.StatusBadRequest)
		return
	}

	jobID := uuid.NewV4()
	j := &job.Job{
		ID:     jobID,
		UserID: uUUID,
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"exp": time.Now().Add(time.Hour * 24).Unix(),
		"iss": "job.service",
		"iat": time.Now().Unix(),
		"sub": j.ID,
	})
	tString, err := t.SignedString([]byte(secret))
	if err != nil {
		log.Printf("Error signing token string: %v\n", err)
		http.Error(w, "Internal error.", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Set-Job-Authorization", tString)

	fw := flushwriter.New(w)
	_, err = fw.Write([]byte("Unpacking request...\n"))
	if err != nil {
		log.Printf("Error writing to flushwriter: %v\n", err)
	}

	jobDir := filepath.Join("job-upload", j.ID.String())
	if err = os.MkdirAll(jobDir, 0755); err != nil {
		log.Printf("Error creating {job} directory %s: %v\n", jobDir, err)
		_, err = fw.Write([]byte("Internal error! Please try again, and if the problem continues contact support.\n"))
		if err != nil {
			log.Printf("Error writing to flushwriter: %v\n", err)
		}
		return
	}

	vals := []string{"requirements", "main", "data"}
	perms := []os.FileMode{0644, 0755, 0644}
	headers := make(map[string]*multipart.FileHeader, len(vals))

	for i := range vals {
		val := vals[i]
		perm := perms[i]
		headers[val], err = saveFormFile(r, val, jobDir, perm)
		if err != nil {
			log.Printf("Error saving %v form file: %v\n", val, err)
			_, err = fw.Write([]byte("Internal error! Please try again, and if the problem continues contact support.\n"))
			if err != nil {
				log.Printf("Error writing to flushwriter: %v\n", err)
			}
			return
		}
	}
	reqFilename := headers[vals[0]].Filename
	mainFilename := headers[vals[1]].Filename

	_, err = fw.Write([]byte("Building image...\n"))
	if err != nil {
		log.Printf("Error writing to flushwriter: %v\n", err)
	}

	ctx := context.Background()
	cli, err := docker.NewEnvClient()
	if err != nil {
		log.Printf("Error creating new docker client: %v\n", err)
		_, err = fw.Write([]byte("Internal error! Please try again, and if the problem continues contact support.\n"))
		if err != nil {
			log.Printf("Error writing to flushwriter: %v\n", err)
		}
		return
	}

	linkedDocker := filepath.Join(jobDir, "Dockerfile")
	userDockerfile := filepath.Join("Dockerfiles", "Dockerfile.user")
	err = os.Link(userDockerfile, linkedDocker)
	if err != nil {
		log.Printf("Error linking dockerfile into user directory: %v\n", err)
		_, err = fw.Write([]byte("Internal error! Please try again, and if the problem continues contact support.\n"))
		if err != nil {
			log.Printf("Error writing to flushwriter: %v\n", err)
		}
		return
	}
	defer check.Err(func() error { return os.Remove(linkedDocker) })

	ctxFiles, err := filepath.Glob(filepath.Join(jobDir, "/*"))
	if err != nil {
		log.Printf("Error collecting docker context files: %v\n", err)
		_, err = fw.Write([]byte("Internal error! Please try again, and if the problem continues contact support.\n"))
		if err != nil {
			log.Printf("Error writing to flushwriter: %v\n", err)
		}
		return
	}

	pr, pw := io.Pipe()
	go func() {
		defer check.Err(pw.Close)
		if err = archiver.TarGz.Write(pw, ctxFiles); err != nil {
			log.Printf("Error tar-gzipping docker context files: %v\n", err)
			_, err = fw.Write([]byte("Internal error! Please try again, and if the problem continues contact support.\n"))
			if err != nil {
				log.Printf("Error writing to flushwriter: %v\n", err)
			}
		}
	}()

	userHome := "/home/user"
	buildResp, err := cli.ImageBuild(ctx, pr, types.ImageBuildOptions{
		BuildArgs: map[string]*string{
			"HOME":         &userHome,
			"REQUIREMENTS": &reqFilename,
			"MAIN":         &mainFilename,
		},
		ForceRemove: true,
		PullParent:  true,
		Tags:        []string{j.ID.String()},
	})
	if err != nil {
		log.Printf("Error building image: %v\n", err)
		_, err = fw.Write([]byte("Internal error! Please try again, and if the problem continues contact support.\n"))
		if err != nil {
			log.Printf("Error writing to flushwriter: %v\n", err)
		}
		return
	}
	defer check.Err(buildResp.Body.Close)

	err = printJSONStream(buildResp.Body)
	if err != nil {
		log.Printf("Error building image: %v\n", err)
		_, err = fw.Write([]byte("Error building docker image! Please try again, and if the problem continues contact support.\n"))
		if err != nil {
			log.Printf("Error writing to flushwriter: %v\n", err)
		}
		return
	}

	_, err = fw.Write([]byte("Image built!\n"))
	if err != nil {
		log.Printf("Error writing to flushwriter: %v\n", err)
	}

	_, err = fw.Write([]byte("Beginning miner auction for job...\n"))
	if err != nil {
		log.Printf("Error writing to flushwriter: %v\n", err)
	}
	log.Printf("Auctioning job: %v\n", j.ID)
	sqlStmt := `
	INSERT INTO jobs (job_uuid, user_uuid, active)
	VALUES ($1, $2, $3)
	`
	if _, err = db.Db.Exec(sqlStmt, j.ID, j.UserID, true); err != nil {
		log.Printf("Error inserting job into db: %v\n", err)
		_, err = fw.Write([]byte("Internal error. Please try again, and if the problem continues contact support.\n"))
		if err != nil {
			log.Printf("Error writing to flushwriter: %v\n", err)
		}
		return
	}
	go miner.Pool.AuctionJob(&job.Job{
		ID: j.ID,
	})
}

func printJSONStream(r io.Reader) error {
	var stream map[string]interface{}

	dec := json.NewDecoder(r)
	for dec.More() {
		if err := dec.Decode(&stream); err != nil {
			return fmt.Errorf("error decoding json stream: %v", err)
		}
		for k, v := range stream {
			if k == "stream" {
				log.Printf("%v", v)
			} else {
				log.Printf("%v: %v\n", k, v)
			}
		}
		if err, ok := stream["error"]; ok {
			return fmt.Errorf("%v", err)
		}
	}
	return nil
}

func saveFormFile(r *http.Request, value, dir string, perm os.FileMode) (*multipart.FileHeader, error) {
	tempFile, header, err := r.FormFile(value)
	if err != nil {
		log.Printf("Error reading %v form file from request: %v\n", value, err)
		return nil, err
	}
	defer check.Err(tempFile.Close)
	path := filepath.Join(dir, header.Filename)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		log.Printf("Error opening %v file: %v\n", path, err)
		return nil, err
	}
	defer check.Err(file.Close)
	_, err = io.Copy(file, tempFile)
	if err != nil {
		log.Printf("Error copying %v form file to disk: %v\n", value, err)
		return nil, err
	}

	return header, nil
}
