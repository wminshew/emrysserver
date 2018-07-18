package user

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/lib/pq"
	"github.com/mholt/archiver"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/db"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/check"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

type operation struct {
	Metadata metadata `json:"metadata,omitempty"`
	Done     bool     `json:"done,omitempty"`
}

type metadata struct {
	Build build `json:"build,omitempty"`
}

type build struct {
	ID     string `json:"id,omitempty"`
	Status string `json:"status,omitempty"`
}

// BuildImage handles building images for jobs posted by users
func BuildImage(w http.ResponseWriter, r *http.Request) *app.Error {
	vars := mux.Vars(r)
	jID := vars["jID"]
	jUUID, err := uuid.FromString(jID)
	if err != nil {
		app.Sugar.Errorw("failed to parse job ID",
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "error parsing job ID"}
	}

	ctx := r.Context()
	// cli, err := docker.NewEnvClient()
	// if err != nil {
	// 	app.Sugar.Errorw("failed to create docker client",
	// 		"url", r.URL,
	// 		"jID", jID,
	// 		"err", err.Error(),
	// 	)
	// 	_ = db.SetJobInactive(r, jUUID)
	// 	return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	// }

	inputDir := filepath.Join("job", jID, "input")
	if err = os.MkdirAll(inputDir, 0755); err != nil {
		app.Sugar.Errorw("failed to create job directory",
			"url", r.URL,
			"err", err.Error(),
			"jID", jID,
		)
		_ = db.SetJobInactive(r, jUUID)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	vals := []string{"requirements", "main"}
	reqs := vals[0]
	main := vals[1]
	for i := range vals {
		p := path.Join(inputDir, vals[i])
		if _, err = os.Stat(p); os.IsNotExist(err) {
			if err = func() error {
				f, err := os.Create(p)
				if err != nil {
					return nil
				}
				or, err := bkt.Object(p).NewReader(ctx)
				if err != nil {
					return err
				}
				if _, err = io.Copy(f, or); err != nil {
					return err
				}
				if err = f.Close(); err != nil {
					return err
				}
				return nil
			}(); err != nil {
				app.Sugar.Errorw("failed to download input file",
					"url", r.URL,
					"err", err.Error(),
					"jID", jID,
				)
				_ = db.SetJobInactive(r, jUUID)
				return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
			}
		}
	}

	// userDockerfile := filepath.Join("Dockerfiles", "Dockerfile.user")
	userDockerfile := filepath.Join("Dockerfiles", "Dockerfile")
	if _, err = os.Stat(userDockerfile); os.IsNotExist(err) {
		if err = func() error {
			if err = os.Mkdir("Dockerfiles", 0755); err != nil {
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
			if err = f.Close(); err != nil {
				return err
			}
			return nil
		}(); err != nil {
			app.Sugar.Errorw("failed to download dockerfile.user",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			_ = db.SetJobInactive(r, jUUID)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}
	}

	linkedDocker := filepath.Join(inputDir, "Dockerfile")
	if err = os.Link(userDockerfile, linkedDocker); err != nil {
		app.Sugar.Errorw("failed link dockerfile into user dir",
			"url", r.URL,
			"err", err.Error(),
			"jID", jID,
		)
		_ = db.SetJobInactive(r, jUUID)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	ctxFiles := []string{
		filepath.Join(inputDir, reqs),
		filepath.Join(inputDir, main),
		filepath.Join(inputDir, "Dockerfile"),
	}
	sourcePath := path.Join("job", jID, "input", "source.tar.gz")
	ow := bkt.Object(sourcePath).NewWriter(ctx)
	if err = archiver.TarGz.Write(ow, ctxFiles); err != nil {
		app.Sugar.Errorw("failed to tar-gzip docker context",
			"url", r.URL,
			"err", err.Error(),
			"jID", jID,
		)
		_ = db.SetJobInactive(r, jUUID)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}
	if err = ow.Close(); err != nil {
		app.Sugar.Errorw("failed to close cloud storage writer",
			"url", r.URL,
			"err", err.Error(),
			"jID", jID,
		)
		_ = db.SetJobInactive(r, jUUID)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}
	// pr, pw := io.Pipe()
	// go func() {
	// 	defer check.Err(r, pw.Close)
	// 	if err = archiver.TarGz.Write(pw, ctxFiles); err != nil {
	// 		app.Sugar.Errorw("failed to tar-gzip docker context",
	// 			"url", r.URL,
	// 			"err", err.Error(),
	// 			"jID", jID,
	// 		)
	// 		return
	// 	}
	// }()

	// userHome := "/home/user"
	// buildResp, err := cli.ImageBuild(ctx, pr, types.ImageBuildOptions{
	// 	BuildArgs: map[string]*string{
	// 		"HOME":         &userHome,
	// 		"REQUIREMENTS": &reqs,
	// 		"MAIN":         &main,
	// 	},
	// 	ForceRemove: true,
	// 	Tags:        []string{jID},
	// })
	// if err != nil {
	// 	app.Sugar.Errorw("failed to build image",
	// 		"url", r.URL,
	// 		"err", err.Error(),
	// 		"jID", jID,
	// 	)
	// 	_ = db.SetJobInactive(r, jUUID)
	// 	return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	// }
	// defer check.Err(r, buildResp.Body.Close)
	//
	// if err = job.ReadJSON(buildResp.Body); err != nil {
	// 	app.Sugar.Errorw("failed to build image",
	// 		"url", r.URL,
	// 		"err", err.Error(),
	// 		"jID", jID,
	// 	)
	// 	_ = db.SetJobInactive(r, jUUID)
	// 	return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	// }

	m := "POST"
	project := "emrys-12"
	p := path.Join("v1", "projects", project, "builds")
	u := url.URL{
		Scheme: "https",
		Host:   "cloudbuild.googleapis.com",
		Path:   p,
	}
	userHome := "/home/user"
	b := fmt.Sprintf(`
	{
		"source": {
			"storageSource": {
				"bucket": "emrys-dev",
				"object": "%s"
			}
		}
		"steps": [
			{
				"name": "gcr.io/cloud-builders/docker",
				"args": [
					"build",
					"--build-arg",
					"HOME=%s"
					"--build-arg",
					"REQUIREMENTS=%s"
					"--build-arg",
					"MAIN=%s"
					"-t",
					"gcr.io/$PROJECT_ID/$_IMAGE:$_BUILD",
					"-t",
					"gcr.io/$PROJECT_ID/$_IMAGE:latest",
					"."
				]
			}
		],
		"images": [
			"gcr.io/$PROJECT_ID/$_IMAGE:$_BUILD",
			"gcr.io/$PROJECT_ID/$_IMAGE:latest"
		],
		"tags": [
			"$_IMAGE",
			"$_BUILD"
		],
		"substitutions": {
			"_IMAGE": "%s",
			"_BUILD": "%s"
		}
	}
	`, sourcePath, userHome, reqs, main, jID, string(time.Now().Unix()))
	body := strings.NewReader(b)
	req, err := http.NewRequest(m, u.String(), body)
	if err != nil {
		app.Sugar.Errorw("failed to create request",
			"url", r.URL,
			"err", err.Error(),
			"jID", jID,
			"method", m,
			"path", u.String(),
		)
		_ = db.SetJobInactive(r, jUUID)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}
	req = req.WithContext(ctx)

	resp, err := oauthClient.Do(req)
	if err != nil {
		app.Sugar.Errorw("failed to execute request",
			"url", r.URL,
			"err", err.Error(),
			"jID", jID,
			"method", m,
			"path", u.String(),
		)
		_ = db.SetJobInactive(r, jUUID)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	opResp := &operation{}
	if err := json.NewDecoder(resp.Body).Decode(opResp); err != nil {
		app.Sugar.Errorw("failed to decode json",
			"url", r.URL,
			"err", err.Error(),
			"jID", jID,
			"method", m,
			"path", u.String(),
		)
		_ = db.SetJobInactive(r, jUUID)
		check.Err(r, resp.Body.Close)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}
	check.Err(r, resp.Body.Close)
	buildID := opResp.Metadata.Build.ID
	status := opResp.Metadata.Build.Status

	m = "GET"
	p = path.Join(p, buildID)
	u = url.URL{
		Scheme: "https",
		Host:   "cloudbuild.googleapis.com",
		Path:   p,
	}
	req, err = http.NewRequest(m, u.String(), nil)
	if err != nil {
		app.Sugar.Errorw("failed to create request",
			"url", r.URL,
			"err", err.Error(),
			"jID", jID,
			"method", m,
			"path", u.String(),
		)
		_ = db.SetJobInactive(r, jUUID)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}
	req = req.WithContext(ctx)

	// responsibly ping google until build is successful, throw error otherwise
	for status != "SUCCESS" {
		time.Sleep(5 * time.Second)
		resp, err = oauthClient.Do(req)
		if err != nil {
			app.Sugar.Errorw("failed to execute request",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
				"method", m,
				"path", u.String(),
			)
			_ = db.SetJobInactive(r, jUUID)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}

		buildResp := &build{}
		if err = json.NewDecoder(resp.Body).Decode(buildResp); err != nil {
			app.Sugar.Errorw("failed to decode json",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
				"method", m,
				"path", u.String(),
			)
			_ = db.SetJobInactive(r, jUUID)
			check.Err(r, resp.Body.Close)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}
		status = buildResp.Status
		check.Err(r, resp.Body.Close)

		// Possible statuses: https://cloud.google.com/container-builder/docs/api/reference/rest/v1/projects.builds#Build.Status
		switch status {
		case "FAILURE", "INTERNAL_ERROR", "TIMEOUT", "CANCELLED":
			app.Sugar.Errorw("failed to build image",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
				"method", m,
				"path", u.String(),
				"buildStatus", status,
			)
			_ = db.SetJobInactive(r, jUUID)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		case "SUCCESS":
			app.Sugar.Infow("image built",
				"url", r.URL,
				"jID", jID,
				"buildStatus", status,
			)
		default:
			app.Sugar.Infow("image building...",
				"url", r.URL,
				"jID", jID,
				"buildStatus", status,
			)
		}
	}

	sqlStmt := `
	UPDATE statuses
	SET (image_built) = ($1)
	WHERE job_uuid = $2
	`
	if _, err = db.Db.Exec(sqlStmt, true, jUUID); err != nil {
		pqErr := err.(*pq.Error)
		if pqErr.Fatal() {
			app.Sugar.Fatalw("failed to update job status",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
				"pq_sev", pqErr.Severity,
				"pq_code", pqErr.Code,
				"pq_detail", pqErr.Detail,
			)
		} else {
			app.Sugar.Errorw("failed to update job status",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
				"pq_sev", pqErr.Severity,
				"pq_code", pqErr.Code,
				"pq_detail", pqErr.Detail,
			)
		}
		_ = db.SetJobInactive(r, jUUID)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	return nil
}
