package main

import (
	"fmt"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"github.com/wminshew/emrysserver/pkg/slack"
	sheets "google.golang.org/api/sheets/v4"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

const (
	feedbackSpreadsheetID = "1cV-GOFw6AWyszRprM9YLd_sgt7wmfG1ucb9q7OmYeNk"
)

// postFeedback posts feedback from an account to the feedback gsheets
var postFeedback app.Handler = func(w http.ResponseWriter, r *http.Request) *app.Error {
	aID := r.Header.Get("X-Jwt-Claims-Subject")
	aUUID, err := uuid.FromString(aID)
	if err != nil {
		log.Sugar.Errorw("error parsing account ID",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "error parsing job ID"}
	}

	email, err := db.GetAccountEmail(r, aUUID)
	if err != nil {
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"} // already logged
	}

	isUser, isMiner, err := db.GetAccountScope(r, aUUID)
	if err != nil {
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"} // already logged
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Sugar.Errorw("error reading feedback request body",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"aID", aUUID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	go func() {
		if err := slack.PostToFeedback(
			fmt.Sprintf("%s: %s", email, body),
		); err != nil {
			log.Sugar.Errorw("error posting to slack",
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"aID", aUUID,
			)
			return
		}
	}()

	a1Range := "Sheet1!A2:E2"
	valueInputOption := "RAW"
	insertDataOption := "INSERT_ROWS"
	ts := time.Now()
	tsUnix := strconv.FormatInt(ts.Unix(), 10)
	newRowStr := []string{
		ts.String(), tsUnix, email,
	}
	if isUser {
		newRowStr = append(newRowStr, "1")
	} else {
		newRowStr = append(newRowStr, "0")
	}
	if isMiner {
		newRowStr = append(newRowStr, "1")
	} else {
		newRowStr = append(newRowStr, "0")
	}
	newRowStr = append(newRowStr, string(body))
	newRow := make([]interface{}, len(newRowStr))
	for i, v := range newRowStr {
		newRow[i] = v
	}
	values := make([][]interface{}, 1)
	values[0] = newRow
	rb := &sheets.ValueRange{
		MajorDimension: "ROWS",
		Range:          a1Range,
		Values:         values,
	}

	ctx := r.Context()
	_, err = sheetsService.Spreadsheets.Values.Append(feedbackSpreadsheetID, a1Range, rb).
		ValueInputOption(valueInputOption).InsertDataOption(insertDataOption).Context(ctx).Do()
	if err != nil {
		log.Sugar.Errorw("error appending user feedback",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"aID", aUUID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	return nil
}
