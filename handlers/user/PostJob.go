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

// PostJob handles new jobs posted by users
func PostJob(w http.ResponseWriter, r *http.Request) {
	maxMemory := int64(1) << 31
	err := r.ParseMultipartForm(maxMemory)
	if err != nil {
		log.Printf("Error parsing request: %v\n", err)
		http.Error(w, "Internal error! Please try again, and if the problem continues contact support.", http.StatusInternalServerError)
		return
	}

	fw := flushwriter.New(w)
	_, err = fw.Write([]byte("Unpacking request...\n"))
	if err != nil {
		log.Printf("Error writing to flushwriter: %v\n", err)
	}

	ctxKey := contextKey("user_uuid")
	uUUID, ok := r.Context().Value(ctxKey).(uuid.UUID)
	if !ok {
		log.Printf("user_uuid in request context corrupted.\n")
		_, err = fw.Write([]byte("Unable to retrieve valid uuid from jwt. Please login again.\n"))
		if err != nil {
			log.Printf("Error writing to flushwriter: %v\n", err)
		}
		return
	}
	jobID := uuid.NewV4()
	j := &job.Job{
		ID:     jobID,
		UserID: uUUID,
	}

	// TODO: re-factor job processing; take out file saving, add relevant paths to r.context
	// TODO: add extra directory layer for project/job number (git vcs?); return job number to client
	// TODO: use s3 or something else?
	// jobDir := filepath.Join("job-upload", uUUID.String(), j.ID.String())
	jobDir := filepath.Join("job-upload", j.ID.String())
	if err = os.MkdirAll(jobDir, 0755); err != nil {
		log.Printf("Error creating {user}/{job} directory %s: %v\n", jobDir, err)
		_, err = fw.Write([]byte("Internal error! Please try again, and if the problem continues contact support.\n"))
		if err != nil {
			log.Printf("Error writing to flushwriter: %v\n", err)
		}
		return
	}

	requirementsTempFile, requirementsHeader, err := r.FormFile("requirements")
	if err != nil {
		log.Printf("Error reading requirements form file: %v\n", err)
		_, err = fw.Write([]byte("Internal error! Please try again, and if the problem continues contact support.\n"))
		if err != nil {
			log.Printf("Error writing to flushwriter: %v\n", err)
		}
		return
	}
	defer check.Err(requirementsTempFile.Close)
	requirementsPath := filepath.Join(jobDir, filepath.Base(requirementsHeader.Filename))
	requirementsFile, err := os.OpenFile(requirementsPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Printf("Error opening requirements file: %v\n", err)
		_, err = fw.Write([]byte("Internal error! Please try again, and if the problem continues contact support.\n"))
		if err != nil {
			log.Printf("Error writing to flushwriter: %v\n", err)
		}
		return
	}
	defer check.Err(requirementsFile.Close)
	_, err = io.Copy(requirementsFile, requirementsTempFile)
	if err != nil {
		log.Printf("Error copying requirements file to disk: %v\n", err)
		_, err = fw.Write([]byte("Internal error! Please try again, and if the problem continues contact support.\n"))
		if err != nil {
			log.Printf("Error writing to flushwriter: %v\n", err)
		}
		return
	}

	trainTempFile, trainHeader, err := r.FormFile("train")
	if err != nil {
		log.Printf("Error reading train form file: %v\n", err)
		_, err = fw.Write([]byte("Internal error! Please try again, and if the problem continues contact support.\n"))
		if err != nil {
			log.Printf("Error writing to flushwriter: %v\n", err)
		}
		return
	}
	defer check.Err(trainTempFile.Close)
	trainPath := filepath.Join(jobDir, filepath.Base(trainHeader.Filename))
	trainFile, err := os.OpenFile(trainPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		log.Printf("Error opening train file: %v\n", err)
		_, err = fw.Write([]byte("Internal error! Please try again, and if the problem continues contact support.\n"))
		if err != nil {
			log.Printf("Error writing to flushwriter: %v\n", err)
		}
		return
	}
	defer check.Err(trainFile.Close)
	_, err = io.Copy(trainFile, trainTempFile)
	if err != nil {
		log.Printf("Error copying train file to disk: %v\n", err)
		_, err = fw.Write([]byte("Internal error! Please try again, and if the problem continues contact support.\n"))
		if err != nil {
			log.Printf("Error writing to flushwriter: %v\n", err)
		}
		return
	}

	dataTempFile, dataHeader, err := r.FormFile("data")
	if err != nil {
		log.Printf("Error reading data form file: %v\n", err)
		_, err = fw.Write([]byte("Internal error! Please try again, and if the problem continues contact support.\n"))
		if err != nil {
			log.Printf("Error writing to flushwriter: %v\n", err)
		}
		return
	}
	defer check.Err(dataTempFile.Close)
	// TODO: need to ... save filepathing somehow? could save with own stuff, and include Filename in response header
	// or token claims or something? Figure it out when I move off disk...
	dataPath := filepath.Join(jobDir, filepath.Base(dataHeader.Filename))
	dataFile, err := os.OpenFile(dataPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Printf("Error opening data file: %v\n", err)
		_, err = fw.Write([]byte("Internal error! Please try again, and if the problem continues contact support.\n"))
		if err != nil {
			log.Printf("Error writing to flushwriter: %v\n", err)
		}
		return
	}
	defer check.Err(dataFile.Close)
	_, err = io.Copy(dataFile, dataTempFile)
	if err != nil {
		log.Printf("Error copying data file to disk: %v\n", err)
		_, err = fw.Write([]byte("Internal error! Please try again, and if the problem continues contact support.\n"))
		if err != nil {
			log.Printf("Error writing to flushwriter: %v\n", err)
		}
		return
	}

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
	buildCtxPath := jobDir + ".tar.gz"
	ctxFiles, err := filepath.Glob(filepath.Join(jobDir, "/*"))
	if err != nil {
		log.Printf("Error collecting docker context files: %v\n", err)
		_, err = fw.Write([]byte("Internal error! Please try again, and if the problem continues contact support.\n"))
		if err != nil {
			log.Printf("Error writing to flushwriter: %v\n", err)
		}
		return
	}
	err = archiver.TarGz.Make(buildCtxPath, ctxFiles)
	if err != nil {
		log.Printf("Error archiving docker context files: %v\n", err)
		_, err = fw.Write([]byte("Internal error! Please try again, and if the problem continues contact support.\n"))
		if err != nil {
			log.Printf("Error writing to flushwriter: %v\n", err)
		}
		return
	}
	defer check.Err(func() error { return os.Remove(buildCtxPath) })
	buildCtx, err := os.Open(buildCtxPath)
	if err != nil {
		log.Printf("Error opening archived docker context files: %v\n", err)
		_, err = fw.Write([]byte("Internal error! Please try again, and if the problem continues contact support.\n"))
		if err != nil {
			log.Printf("Error writing to flushwriter: %v\n", err)
		}
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
		_, err = fw.Write([]byte("Internal error! Please try again, and if the problem continues contact support.\n"))
		if err != nil {
			log.Printf("Error writing to flushwriter: %v\n", err)
		}
		return
	}
	defer check.Err(buildResp.Body.Close)

	printJSONStream(buildResp.Body)

	_, err = fw.Write([]byte("Image built!\n"))
	if err != nil {
		log.Printf("Error writing to flushwriter: %v\n", err)
	}

	_, err = fw.Write([]byte("Beginning miner auction for job...\n"))
	if err != nil {
		log.Printf("Error writing to flushwriter: %v\n", err)
	}
	log.Printf("Auctioning job: %v\n", j.ID)
	if _, err = db.Db.Query("INSERT INTO jobs (job_uuid, user_uuid) VALUES ($1, $2)",
		j.ID, j.UserID); err != nil {
		log.Printf("Error inserting job into db: %v\n", err)
		_, err = fw.Write([]byte("Internal error! Please try again, and if the problem continues contact support.\n"))
		if err != nil {
			log.Printf("Error writing to flushwriter: %v\n", err)
		}
		return
	}
	go miner.Pool.AuctionJob(&job.Job{
		ID: j.ID,
	})
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
