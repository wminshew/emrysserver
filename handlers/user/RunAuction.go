package user

import (
	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/db"
	"github.com/wminshew/emrysserver/handlers/miner"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/check"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
)

// RunAuction handles running auctions for jobs posted by users
func RunAuction(w http.ResponseWriter, r *http.Request) *app.Error {
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

	// Start auction
	s := "http"
	h := "localhost:8081"
	p := path.Join("job", jID, "auction")
	u := url.URL{
		Scheme: s,
		Host:   h,
		Path:   p,
	}
	m := "POST"
	req, err := http.NewRequest(m, u.String(), nil)
	if err != nil {
		app.Sugar.Errorw("failed to create request",
			"url", r.URL,
			"err", err.Error(),
			"method", m,
			"path", u.String(),
		)
		_ = db.SetJobInactive(r, jUUID)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}
	req = req.WithContext(r.Context())
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		app.Sugar.Errorw("failed to execute request",
			"url", r.URL,
			"err", err.Error(),
			"method", m,
			"path", u.String(),
		)
		_ = db.SetJobInactive(r, jUUID)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	if resp.StatusCode != http.StatusOK {
		check.Err(r, resp.Body.Close)
		_ = db.SetJobInactive(r, jUUID)
		return &app.Error{Code: resp.StatusCode, Message: resp.Status}
	}
	check.Err(r, resp.Body.Close)

	// Message miners
	// TODO: fix this janky mess
	if apperr := miner.PostAuction(w, r); apperr != nil {
		return apperr
	}

	// TODO: switch to http once miner running on own server behind proxy which handles https
	// h = "localhost"
	// p = path.Join("miner", "job", jID, "auction")
	// u = url.URL{
	// 	Scheme: s,
	// 	Host:   h,
	// 	Path:   p,
	// }
	// m = "POST"
	// req, err = http.NewRequest(m, u.String(), nil)
	// if err != nil {
	// 	app.Sugar.Errorw("failed to create request",
	// 		"url", r.URL,
	// 		"err", err.Error(),
	// 		"method", m,
	// 		"path", u.String(),
	// 	)
	// 	_ = db.SetJobInactive(r, jUUID)
	// 	return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	// }
	// req = req.WithContext(r.Context())
	// resp, err = client.Do(req)
	// if err != nil {
	// 	app.Sugar.Errorw("failed to execute request",
	// 		"url", r.URL,
	// 		"err", err.Error(),
	// 		"method", m,
	// 		"path", u.String(),
	// 	)
	// 	_ = db.SetJobInactive(r, jUUID)
	// 	return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	// }
	//
	// if resp.StatusCode != http.StatusOK {
	// 	check.Err(r, resp.Body.Close)
	// 	_ = db.SetJobInactive(r, jUUID)
	// 	return &app.Error{Code: resp.StatusCode, Message: resp.Status}
	// }
	// check.Err(r, resp.Body.Close)

	// Query auction success
	h = "localhost:8081"
	p = path.Join("job", jID, "auction", "success")
	u = url.URL{
		Scheme: s,
		Host:   h,
		Path:   p,
	}
	m = "GET"
	req, err = http.NewRequest(m, u.String(), nil)
	if err != nil {
		app.Sugar.Errorw("failed to create request",
			"url", r.URL,
			"err", err.Error(),
			"method", m,
			"path", u.String(),
		)
		_ = db.SetJobInactive(r, jUUID)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}
	req = req.WithContext(r.Context())
	resp, err = client.Do(req)
	if err != nil {
		app.Sugar.Errorw("failed to execute request",
			"url", r.URL,
			"err", err.Error(),
			"method", m,
			"path", u.String(),
		)
		_ = db.SetJobInactive(r, jUUID)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	if resp.StatusCode != http.StatusOK {
		defer check.Err(r, resp.Body.Close)
		_ = db.SetJobInactive(r, jUUID)
		b, _ := ioutil.ReadAll(resp.Body)
		return &app.Error{Code: resp.StatusCode, Message: string(b)}
	}
	check.Err(r, resp.Body.Close)

	return nil
}
