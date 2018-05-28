package user

import (
	"bufio"
	"context"
	"docker.io/go-docker"
	"docker.io/go-docker/api/types"
	// "encoding/json"
	// "fmt"
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
	// defer check.Err(func() error { return os.RemoveAll(jobDir) })

	rqmtsTempFile, rqmtsHeader, err := r.FormFile("requirements")
	if err != nil {
		log.Printf("Error reading requirements form file: %v\n", err)
		_, err = fw.Write([]byte("Internal error! Please try again, and if the problem continues contact support.\n"))
		if err != nil {
			log.Printf("Error writing to flushwriter: %v\n", err)
		}
		return
	}
	defer check.Err(rqmtsTempFile.Close)
	rqmtsPath := filepath.Join(jobDir, rqmtsHeader.Filename)
	rqmtsFile, err := os.OpenFile(rqmtsPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Printf("Error opening requirements file: %v\n", err)
		_, err = fw.Write([]byte("Internal error! Please try again, and if the problem continues contact support.\n"))
		if err != nil {
			log.Printf("Error writing to flushwriter: %v\n", err)
		}
		return
	}
	defer check.Err(rqmtsFile.Close)
	_, err = io.Copy(rqmtsFile, rqmtsTempFile)
	if err != nil {
		log.Printf("Error copying requirements file to disk: %v\n", err)
		_, err = fw.Write([]byte("Internal error! Please try again, and if the problem continues contact support.\n"))
		if err != nil {
			log.Printf("Error writing to flushwriter: %v\n", err)
		}
		return
	}

	mainTempFile, mainHeader, err := r.FormFile("main")
	if err != nil {
		log.Printf("Error reading main form file: %v\n", err)
		_, err = fw.Write([]byte("Internal error! Please try again, and if the problem continues contact support.\n"))
		if err != nil {
			log.Printf("Error writing to flushwriter: %v\n", err)
		}
		return
	}
	defer check.Err(mainTempFile.Close)
	mainPath := filepath.Join(jobDir, mainHeader.Filename)
	mainFile, err := os.OpenFile(mainPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		log.Printf("Error opening main file: %v\n", err)
		_, err = fw.Write([]byte("Internal error! Please try again, and if the problem continues contact support.\n"))
		if err != nil {
			log.Printf("Error writing to flushwriter: %v\n", err)
		}
		return
	}
	defer check.Err(mainFile.Close)
	_, err = io.Copy(mainFile, mainTempFile)
	if err != nil {
		log.Printf("Error copying main file to disk: %v\n", err)
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
	dataPath := filepath.Join(jobDir, dataHeader.Filename)
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
	requirements := rqmtsHeader.Filename
	main := mainHeader.Filename
	buildResp, err := cli.ImageBuild(ctx, buildCtx, types.ImageBuildOptions{
		BuildArgs: map[string]*string{
			"HOME":         &userHome,
			"REQUIREMENTS": &requirements,
			"MAIN":         &main,
		},
		ForceRemove: true,
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
	// type stream struct {
	// 	Stream string
	// }
	// dec := json.NewDecoder(r)
	// for dec.More() {
	// 	var s stream
	// 	err := dec.Decode(&s)
	// 	if err != nil {
	// 		log.Printf("Error decoding json build stream: %v\n", err)
	// 	}
	//
	// 	fmt.Printf("%v", s.Stream)
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		log.Printf(scanner.Text())
	}
}
