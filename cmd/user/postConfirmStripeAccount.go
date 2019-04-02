package main

import (
	"encoding/json"
	"fmt"
	"github.com/cenkalti/backoff"
	"github.com/satori/go.uuid"
	stripe "github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/account"
	"github.com/wminshew/emrys/pkg/check"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"time"
)

type stripeOAuth struct {
	AccessToken  string `json:"access_token,omitempty"`
	Livemode     bool   `json:"livemode,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	StripePubKey string `json:"stripe_publishable_key,omitempty"`
	StripeUserID string `json:"stripe_user_id,omitempty"`
	Scope        string `json:"scope,omitempty"`
	Err          string `json:"error,omitempty"`
	ErrDetail    string `json:"error_description,omitempty"`
}

// postConfirmStripeAccount confirms a new stripe account
var postConfirmStripeAccount app.Handler = func(w http.ResponseWriter, r *http.Request) *app.Error {
	code := r.URL.Query().Get("code")
	if code == "" {
		return &app.Error{Code: http.StatusBadRequest, Message: "no stripe authorization code"}
	}

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

	ctx := r.Context()
	client := &http.Client{}
	u := url.URL{
		Scheme: "https",
		Host:   "connect.stripe.com",
		Path:   path.Join("oauth", "token"),
	}
	data := url.Values{}
	data.Set("client_secret", stripeSecretKey)
	data.Set("code", code)
	data.Set("grant_type", "authorization_code")
	var stripeResp stripeOAuth
	operation := func() error {
		resp, err := client.PostForm(u.String(), data)
		if err != nil {
			return err
		}
		defer check.Err(resp.Body.Close)

		if resp.StatusCode == http.StatusBadGateway {
			return fmt.Errorf("server: temporary error")
		} else if resp.StatusCode >= 300 {
			b, _ := ioutil.ReadAll(resp.Body)
			return backoff.Permanent(fmt.Errorf("server: %v", string(b)))
		}

		if err := json.NewDecoder(resp.Body).Decode(&stripeResp); err != nil {
			return backoff.Permanent(fmt.Errorf("decoding response: %v", err))
		}

		return nil
	}
	if err := backoff.RetryNotify(operation,
		backoff.WithContext(backoff.WithMaxRetries(backoff.NewExponentialBackOff(), maxRetries), ctx),
		func(err error, t time.Duration) {
			log.Sugar.Errorw("error posting stripe code for new account, retrying",
				"method", r.Method,
				"url", r.URL,
				"err", err.Error(),
				"aID", aUUID,
			)
		}); err != nil {
		log.Sugar.Errorw("error posting stripe code for new account--aborting",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"aID", aUUID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "error posting stripe authorization code"}
	}

	if stripeResp.Err != "" {
		log.Sugar.Errorw("error posting stripe code for new account",
			"method", r.Method,
			"url", r.URL,
			"err", stripeResp.Err,
			"errDetail", stripeResp.ErrDetail,
			"aID", aUUID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "error posting stripe authorization code"}
	}

	if err := db.SetAccountStripeAccountID(aUUID, stripeResp.StripeUserID); err != nil {
		log.Sugar.Errorw("error setting stripe account ID",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"aID", aUUID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	// set account payout schedule to monthly with anchor on the 1st
	params := &stripe.AccountParams{
		Settings: &stripe.AccountSettingsParams{
			Payouts: &stripe.AccountSettingsPayoutsParams{
				Schedule: &stripe.PayoutScheduleParams{
					DelayDays:     stripe.Int64(7),
					Interval:      stripe.String("monthly"),
					MonthlyAnchor: stripe.Int64(1),
				},
			},
		},
	}
	if _, err := account.Update(stripeResp.StripeUserID, params); err != nil {
		log.Sugar.Errorw("error updating stripe account",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"aID", aUUID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	if _, err := w.Write([]byte(stripeResp.StripeUserID)); err != nil {
		log.Sugar.Errorw("error writing stripe account ID",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"aID", aUUID,
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}
	return nil
}
