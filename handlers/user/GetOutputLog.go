package user

import (
	"github.com/gorilla/mux"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/check"
	"github.com/wminshew/emrysserver/pkg/flushwriter"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
)

// GetOutputLog streams job output to user
func GetOutputLog(w http.ResponseWriter, r *http.Request) *app.Error {
	vars := mux.Vars(r)
	jID := vars["jID"]
	p := path.Join("job", jID, "log")
	u := url.URL{
		Scheme: "http",
		Host:   "localhost:8081",
		Path:   p,
	}
	m := "GET"
	req, err := http.NewRequest(m, u.String(), nil)
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

	fw := flushwriter.New(w)
	if _, err = io.Copy(fw, resp.Body); err != nil {
		app.Sugar.Errorw("failed to copy pipe reader to flushwriter",
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	return nil
}
