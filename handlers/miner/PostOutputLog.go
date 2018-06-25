package miner

import (
	"github.com/gorilla/mux"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/check"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
)

// PostOutputLog receives job output from miner
func PostOutputLog(w http.ResponseWriter, r *http.Request) *app.Error {
	vars := mux.Vars(r)
	jID := vars["jID"]
	p := path.Join("job", jID, "log")
	u := url.URL{
		Scheme: "http",
		Host:   "localhost:8081",
		Path:   p,
	}
	m := "POST"
	req, err := http.NewRequest(m, u.String(), r.Body)
	if err != nil {
		app.Sugar.Errorw("failed to create request",
			"url", r.URL,
			"method", m,
			"path", u.String(),
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}
	req = req.WithContext(r.Context())
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		app.Sugar.Errorw("failed to execute request",
			"url", r.URL,
			"method", m,
			"path", u.String(),
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}
	defer check.Err(r, resp.Body.Close)

	if resp.StatusCode != http.StatusOK {
		b, _ := ioutil.ReadAll(resp.Body)
		return &app.Error{Code: resp.StatusCode, Message: string(b)}
	}

	return nil
}
