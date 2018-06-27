package user

import (
	"docker.io/go-docker"
	"docker.io/go-docker/api/types"
	"github.com/gorilla/mux"
	"github.com/lib/pq"
	"github.com/mholt/archiver"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrys/pkg/job"
	"github.com/wminshew/emrysserver/db"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/check"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
)

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
	cli, err := docker.NewEnvClient()
	if err != nil {
		app.Sugar.Errorw("failed to create docker client",
			"url", r.URL,
			"jID", jID,
			"err", err.Error(),
		)
		_ = db.SetJobInactive(r, jUUID)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

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

	userDockerfile := filepath.Join("Dockerfiles", "Dockerfile.user")
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
	pr, pw := io.Pipe()
	go func() {
		defer check.Err(r, pw.Close)
		if err = archiver.TarGz.Write(pw, ctxFiles); err != nil {
			app.Sugar.Errorw("failed to tar-gzip docker context",
				"url", r.URL,
				"err", err.Error(),
				"jID", jID,
			)
			return
		}
	}()

	userHome := "/home/user"
	buildResp, err := cli.ImageBuild(ctx, pr, types.ImageBuildOptions{
		BuildArgs: map[string]*string{
			"HOME":         &userHome,
			"REQUIREMENTS": &reqs,
			"MAIN":         &main,
		},
		ForceRemove: true,
		Tags:        []string{jID},
	})
	if err != nil {
		app.Sugar.Errorw("failed to build image",
			"url", r.URL,
			"err", err.Error(),
			"jID", jID,
		)
		_ = db.SetJobInactive(r, jUUID)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}
	defer check.Err(r, buildResp.Body.Close)

	if err = job.ReadJSON(buildResp.Body); err != nil {
		app.Sugar.Errorw("failed to build image",
			"url", r.URL,
			"err", err.Error(),
			"jID", jID,
		)
		_ = db.SetJobInactive(r, jUUID)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
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
