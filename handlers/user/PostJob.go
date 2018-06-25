package user

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrys/pkg/check"
	"github.com/wminshew/emrys/pkg/job"
	"github.com/wminshew/emrysserver/db"
	"github.com/wminshew/emrysserver/pkg/app"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// PostJob handles new jobs posted by users
func PostJob(w http.ResponseWriter, r *http.Request) *app.Error {
	maxMemory := int64(1) << 31
	err := r.ParseMultipartForm(maxMemory)
	if err != nil {
		app.Sugar.Errorw("failed to parse multipart form request",
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "error parsing multipart form request"}
	}

	vars := mux.Vars(r)
	uID := vars["uID"]
	uUUID, err := uuid.FromString(uID)
	if err != nil {
		app.Sugar.Errorw("failed to parse user ID",
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "error parsing user ID"}
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
		app.Sugar.Errorw("failed to sign job token",
			"url", r.URL,
			"jID", j.ID,
			"err", err.Error(),
		)
		_ = setJobInactive(r, j.ID)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}
	w.Header().Set("Set-Job-Authorization", tString)

	sqlStmt := `
	INSERT INTO jobs (job_uuid, user_uuid, active)
	VALUES ($1, $2, $3)
	`
	if _, err = db.Db.Exec(sqlStmt, j.ID, j.UserID, true); err != nil {
		app.Sugar.Errorw("failed to insert job",
			"url", r.URL,
			"jID", j.ID,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}
	sqlStmt = `
	INSERT INTO payments (job_uuid, user_paid, miner_paid)
	VALUES ($1, $2, $3)
	`
	if _, err = db.Db.Exec(sqlStmt, j.ID, false, false); err != nil {
		app.Sugar.Errorw("failed to insert payment",
			"url", r.URL,
			"jID", j.ID,
			"err", err.Error(),
		)
		_ = setJobInactive(r, j.ID)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}
	sqlStmt = `
	INSERT INTO statuses (job_uuid, user_data_stored,
	image_built, auction_completed,
	image_downloaded, data_downloaded,
	output_log_posted, output_dir_posted)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	if _, err = db.Db.Exec(sqlStmt, j.ID, false, false, false, false, false, false, false); err != nil {
		app.Sugar.Errorw("failed to insert status",
			"url", r.URL,
			"jID", j.ID,
			"err", err.Error(),
		)
		_ = setJobInactive(r, j.ID)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	inputDir := filepath.Join("job", j.ID.String(), "input")
	if err = os.MkdirAll(inputDir, 0755); err != nil {
		app.Sugar.Errorw("failed to create job directory",
			"url", r.URL,
			"jID", j.ID,
			"err", err.Error(),
		)
		_ = setJobInactive(r, j.ID)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	vals := []string{"requirements", "main", "data"}
	for i := range vals {
		err = uploadAndCacheFormFile(r, inputDir, vals[i])
		if err != nil {
			app.Sugar.Errorw("failed to upload form file",
				"url", r.URL,
				"jID", j.ID,
				"err", err.Error(),
			)
			_ = setJobInactive(r, j.ID)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}
	}

	sqlStmt = `
	UPDATE statuses
	SET (user_data_stored) = ($1)
	WHERE job_uuid = $2
	`
	if _, err = db.Db.Exec(sqlStmt, true, j.ID); err != nil {
		app.Sugar.Errorw("failed to update status",
			"url", r.URL,
			"jID", j.ID,
			"err", err.Error(),
		)
		_ = setJobInactive(r, j.ID)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	return nil
}

func uploadAndCacheFormFile(r *http.Request, dir, val string) error {
	f, _, err := r.FormFile(val)
	if err != nil {
		return err
	}
	defer check.Err(f.Close)

	p := filepath.Join(dir, val)
	file, err := os.Create(p)
	if err != nil {
		return err
	}
	defer check.Err(file.Close)
	tee := io.TeeReader(f, file)

	ctx := r.Context()
	ow := bkt.Object(p).NewWriter(ctx)
	_, err = io.Copy(ow, tee)
	if err != nil {
		return err
	}
	if err = ow.Close(); err != nil {
		return err
	}

	return nil
}

func setJobInactive(r *http.Request, jUUID uuid.UUID) error {
	sqlStmt := `
	UPDATE jobs
	SET (active) = ($1)
	WHERE job_uuid = $2
	`
	if _, err := db.Db.Exec(sqlStmt, false, jUUID); err != nil {
		app.Sugar.Errorw("failed to update job",
			"url", r.URL,
			"jID", jUUID,
			"err", err.Error(),
		)
		return err
	}
	return nil
}
