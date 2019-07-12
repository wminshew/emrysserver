package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/cenkalti/backoff"
	"github.com/pkg/errors"
	"github.com/wminshew/emrys/pkg/check"
	"github.com/wminshew/emrysserver/pkg/log"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

// PostToJobs posts msg to the jobs channel
func PostToJobs(msg string) error {
	ctx := context.Background()
	client := &http.Client{}
	u := url.URL{
		Scheme: "https",
		Host:   "hooks.slack.com",
		Path:   "services/TJ42GDSA0/BLBTB84JG/19807dDjC06aY1EwVP5NOnfR",
	}

	buf := &bytes.Buffer{}
	slackMsg := slackMessage{
		Text: msg,
	}
	if err := json.NewEncoder(buf).Encode(&slackMsg); err != nil {
		return errors.Wrap(err, "encoding slack message")
	}

	operation := func() error {
		req, err := http.NewRequest(http.MethodPost, u.String(), buf)
		if err != nil {
			return err
		}
		req.Header.Set("Content-type", "application/json")

		resp, err := client.Do(req)
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

		return nil
	}
	if err := backoff.RetryNotify(operation,
		backoff.WithContext(backoff.WithMaxRetries(backoff.NewExponentialBackOff(), maxRetries), ctx),
		func(err error, t time.Duration) {
			log.Sugar.Errorw("error posting to jobs in slack, retrying",
				"err", err.Error(),
			)
		}); err != nil {
		return errors.Wrap(err, "posting to jobs in slack")
	}

	return nil
}
