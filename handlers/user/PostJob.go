package user

import (
	"context"
	"docker.io/go-docker"
	"docker.io/go-docker/api/types"
	"encoding/json"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/mholt/archiver"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrys/pkg/check"
	"github.com/wminshew/emrys/pkg/job"
	"github.com/wminshew/emrysserver/db"
	"github.com/wminshew/emrysserver/handlers/miner"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
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

	vars := mux.Vars(r)
	uID := vars["uID"]
	uUUID, err := uuid.FromString(uID)
	if err != nil {
		log.Printf("Error parsing user ID: %v\n", err)
		http.Error(w, "Error parsing user ID in path", http.StatusBadRequest)
		return
	}

	jobID := uuid.NewV4()
	j := &job.Job{
		ID:     jobID,
		UserID: uUUID,
	}

	sqlStmt := `
	INSERT INTO jobs (job_uuid, user_uuid, active)
	VALUES ($1, $2, $3)
	`
	if _, err = db.Db.Exec(sqlStmt, j.ID, j.UserID, true); err != nil {
		log.Printf("Error inserting job: %v\n", err)
		http.Error(w, "Internal error! Please try again, and if the problem continues contact support.", http.StatusInternalServerError)
		return
	}
	sqlStmt = `
	INSERT INTO payments (job_uuid, user_paid, miner_paid)
	VALUES ($1, $2, $3)
	`
	if _, err = db.Db.Exec(sqlStmt, j.ID, false, false); err != nil {
		log.Printf("Error inserting payment: %v\n", err)
		setJobInactive(j.ID)
		http.Error(w, "Internal error! Please try again, and if the problem continues contact support.", http.StatusInternalServerError)
		return
	}
	sqlStmt = `
	INSERT INTO statuses (job_uuid, user_data_stored,
	image_built, auction_completed,
	image_downloaded, data_downloaded,
	output_log_posted, output_dir_posted)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	if _, err = db.Db.Exec(sqlStmt, j.ID, false, false, false, false, false, false, false); err != nil {
		log.Printf("Error inserting status: %v\n", err)
		setJobInactive(j.ID)
		http.Error(w, "Internal error! Please try again, and if the problem continues contact support.", http.StatusInternalServerError)
		return
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
		setJobInactive(j.ID)
		http.Error(w, "Internal error.", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Set-Job-Authorization", tString)

	e := json.NewEncoder(w)
	err = e.Encode(map[string]string{"stream": "Unpacking request...\n"})
	if err != nil {
		log.Printf("Error writing to http response json encoder: %v\n", err)
		setJobInactive(j.ID)
		return
	}
	f, ok := w.(http.Flusher)
	if !ok {
		log.Printf("Error flushing response writer\n")
		setJobInactive(j.ID)
		err = e.Encode(map[string]string{"error": "Internal error! Please try again, and if the problem continues contact support.\n"})
		if err != nil {
			log.Printf("Error writing to http response json encoder: %v\n", err)
			return
		}
		return
	}
	f.Flush()

	jobDir := filepath.Join("job-upload", j.ID.String())
	if err = os.MkdirAll(jobDir, 0755); err != nil {
		log.Printf("Error creating {job} directory %s: %v\n", jobDir, err)
		setJobInactive(j.ID)
		err = e.Encode(map[string]string{"error": "Internal error! Please try again, and if the problem continues contact support.\n"})
		if err != nil {
			log.Printf("Error writing to http response json encoder: %v\n", err)
			return
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
			setJobInactive(j.ID)
			err = e.Encode(map[string]string{"error": "Internal error! Please try again, and if the problem continues contact support.\n"})
			if err != nil {
				log.Printf("Error writing to http response json encoder: %v\n", err)
				return
			}
			return
		}
	}
	reqFilename := headers[vals[0]].Filename
	mainFilename := headers[vals[1]].Filename

	sqlStmt = `
	UPDATE statuses
	SET (user_data_stored) = ($1)
	WHERE job_uuid = $2
	`
	if _, err = db.Db.Exec(sqlStmt, true, j.ID); err != nil {
		log.Printf("Error updating status (user_data_stored) in db: %v\n", err)
		setJobInactive(j.ID)
		err = e.Encode(map[string]string{"error": "Internal error! Please try again, and if the problem continues contact support.\n"})
		if err != nil {
			log.Printf("Error writing to http response json encoder: %v\n", err)
			return
		}
		return
	}

	err = e.Encode(map[string]string{"stream": "Building image...\n"})
	if err != nil {
		log.Printf("Error writing to http response json encoder: %v\n", err)
		setJobInactive(j.ID)
		return
	}
	f.Flush()

	ctx := context.Background()
	cli, err := docker.NewEnvClient()
	if err != nil {
		log.Printf("Error creating new docker client: %v\n", err)
		setJobInactive(j.ID)
		err = e.Encode(map[string]string{"error": "Internal error! Please try again, and if the problem continues contact support.\n"})
		if err != nil {
			log.Printf("Error writing to http response json encoder: %v\n", err)
			return
		}
		return
	}

	linkedDocker := filepath.Join(jobDir, "Dockerfile")
	userDockerfile := filepath.Join("Dockerfiles", "Dockerfile.user")
	err = os.Link(userDockerfile, linkedDocker)
	if err != nil {
		log.Printf("Error linking dockerfile into user directory: %v\n", err)
		setJobInactive(j.ID)
		err = e.Encode(map[string]string{"error": "Internal error! Please try again, and if the problem continues contact support.\n"})
		if err != nil {
			log.Printf("Error writing to http response json encoder: %v\n", err)
			return
		}
		return
	}
	defer check.Err(func() error { return os.Remove(linkedDocker) })

	ctxFiles, err := filepath.Glob(filepath.Join(jobDir, "/*"))
	if err != nil {
		log.Printf("Error collecting docker context files: %v\n", err)
		setJobInactive(j.ID)
		err = e.Encode(map[string]string{"error": "Internal error! Please try again, and if the problem continues contact support.\n"})
		if err != nil {
			log.Printf("Error writing to http response json encoder: %v\n", err)
			return
		}
		return
	}

	pr, pw := io.Pipe()
	go func() {
		defer check.Err(pw.Close)
		if err = archiver.TarGz.Write(pw, ctxFiles); err != nil {
			log.Printf("Error tar-gzipping docker context files: %v\n", err)
			err = e.Encode(map[string]string{"error": "Internal error! Please try again, and if the problem continues contact support.\n"})
			if err != nil {
				log.Printf("Error writing to http response json encoder: %v\n", err)
				return
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
		// PullParent:  true,
		Tags: []string{j.ID.String()},
	})
	if err != nil {
		log.Printf("Error building image: %v\n", err)
		setJobInactive(j.ID)
		err = e.Encode(map[string]string{"error": "Internal error! Please try again, and if the problem continues contact support.\n"})
		if err != nil {
			log.Printf("Error writing to http response json encoder: %v\n", err)
			return
		}
		return
	}
	defer check.Err(buildResp.Body.Close)

	err = job.ReadJSON(buildResp.Body)
	if err != nil {
		log.Printf("Error building image: %v\n", err)
		setJobInactive(j.ID)
		err = e.Encode(map[string]string{"error": "Internal error! Please try again, and if the problem continues contact support.\n"})
		if err != nil {
			log.Printf("Error writing to http response json encoder: %v\n", err)
			return
		}
		return
	}

	err = e.Encode(map[string]string{"stream": "Image built!\n"})
	if err != nil {
		log.Printf("Error writing to http response json encoder: %v\n", err)
		setJobInactive(j.ID)
		return
	}
	f.Flush()

	sqlStmt = `
	UPDATE statuses
	SET (image_built) = ($1)
	WHERE job_uuid = $2
	`
	if _, err = db.Db.Exec(sqlStmt, true, j.ID); err != nil {
		log.Printf("Error updating status (image_built) in db: %v\n", err)
		setJobInactive(j.ID)
		err = e.Encode(map[string]string{"error": "Internal error! Please try again, and if the problem continues contact support.\n"})
		if err != nil {
			log.Printf("Error writing to http response json encoder: %v\n", err)
			return
		}
		return
	}

	err = e.Encode(map[string]string{"stream": "Beginning miner auction for job...\n"})
	if err != nil {
		log.Printf("Error writing to http response json encoder: %v\n", err)
		setJobInactive(j.ID)
		return
	}
	f.Flush()
	log.Printf("Auctioning job: %v\n", j.ID)
	go miner.Pool.AuctionJob(&job.Job{
		ID: j.ID,
	})

	p := path.Join("job", j.ID.String(), "auction", "success")
	u := url.URL{
		Scheme: "http",
		Host:   "localhost:8081",
		Path:   p,
	}
	resp, err := http.Get(u.String())
	if err != nil {
		log.Printf("Error GET %v: %v\n", u.String(), err)
		setJobInactive(j.ID)
		err = e.Encode(map[string]string{"error": "Internal error! Please try again, and if the problem continues contact support.\n"})
		if err != nil {
			log.Printf("Error writing to http response json encoder: %v\n", err)
			return
		}
		return
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("Internal error: Response header error: %v\n", resp.Status)
		check.Err(resp.Body.Close)
		setJobInactive(j.ID)
		err = e.Encode(map[string]string{"error": "Internal error! Please try again, and if the problem continues contact support.\n"})
		if err != nil {
			log.Printf("Error writing to http response json encoder: %v\n", err)
			return
		}
		return
	}

	var dec map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&dec)
	if err != nil {
		log.Printf("Error decoding response json %v: %v\n", u.String(), err)
		setJobInactive(j.ID)
		err = e.Encode(map[string]string{"error": "Internal error! Please try again, and if the problem continues contact support.\n"})
		if err != nil {
			log.Printf("Error writing to http response json encoder: %v\n", err)
			return
		}
		return
	}

	success, ok := dec["success"]
	if !ok {
		setJobInactive(j.ID)
		err = e.Encode(map[string]string{"error": "Internal error! Please try again, and if the problem continues contact support.\n"})
		if err != nil {
			log.Printf("Error writing to http response json encoder: %v\n", err)
			return
		}
	}

	successBool, ok := success.(bool)
	if !ok {
		setJobInactive(j.ID)
		err = e.Encode(map[string]string{"error": "Internal error! Please try again, and if the problem continues contact support.\n"})
		if err != nil {
			log.Printf("Error writing to http response json encoder: %v\n", err)
			return
		}
		return
	}

	if successBool {
		err = e.Encode(map[string]string{"stream": "Auction success!\n"})
		if err != nil {
			log.Printf("Error writing to http response json encoder: %v\n", err)
			setJobInactive(j.ID)
			return
		}
		f.Flush()
	} else {
		errDetail, ok := dec["error"]
		if !ok {
			setJobInactive(j.ID)
			err = e.Encode(map[string]string{"error": "Internal error! Please try again, and if the problem continues contact support.\n"})
			if err != nil {
				log.Printf("Error writing to http response json encoder: %v\n", err)
				return
			}
		}

		errDetailString, ok := errDetail.(string)
		if !ok {
			setJobInactive(j.ID)
			err = e.Encode(map[string]string{"error": "Internal error! Please try again, and if the problem continues contact support.\n"})
			if err != nil {
				log.Printf("Error writing to http response json encoder: %v\n", err)
				return
			}
		}

		setJobInactive(j.ID)
		err = e.Encode(map[string]string{"error": errDetailString})
		if err != nil {
			log.Printf("Error writing to http response json encoder: %v\n", err)
			return
		}
	}
}

func saveFormFile(r *http.Request, value, dir string, perm os.FileMode) (*multipart.FileHeader, error) {
	f, fh, err := r.FormFile(value)
	if err != nil {
		log.Printf("Error reading %v form file from request: %v\n", value, err)
		return nil, err
	}
	defer check.Err(f.Close)
	path := filepath.Join(dir, fh.Filename)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		log.Printf("Error opening %v file: %v\n", path, err)
		return nil, err
	}
	defer check.Err(file.Close)
	_, err = io.Copy(file, f)
	if err != nil {
		log.Printf("Error copying %v form file to disk: %v\n", value, err)
		return nil, err
	}

	return fh, nil
}

func setJobInactive(jUUID uuid.UUID) {
	sqlStmt := `
	UPDATE jobs
	SET (active) = ($1)
	WHERE job_uuid = $2
	`
	if _, err := db.Db.Exec(sqlStmt, false, jUUID); err != nil {
		log.Printf("Error updating jobs (active) in db: %v\n", err)
		return
	}
}
