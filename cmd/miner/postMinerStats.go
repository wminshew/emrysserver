package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cenkalti/backoff"
	"github.com/dgrijalva/jwt-go"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrys/pkg/check"
	"github.com/wminshew/emrys/pkg/job"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/db"
	"github.com/wminshew/emrysserver/pkg/log"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"time"
)

type activeMiner struct {
	ActiveWorkers map[uuid.UUID]*activeWorker `json:"activeWorkers"`
}

type activeWorker struct {
	lastPost time.Time
	JobID    uuid.UUID `json:"JobID"`
}

var (
	activeMiners map[uuid.UUID]*activeMiner
)

// postMinerStats receives a snapshot of the miner's system and resets active workers' timeouts
var postMinerStats app.Handler = func(w http.ResponseWriter, r *http.Request) *app.Error {
	mID := r.Header.Get("X-Jwt-Claims-Subject")
	mUUID, err := uuid.FromString(mID)
	if err != nil {
		log.Sugar.Errorw("error parsing miner ID",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "error parsing miner ID"}
	}

	minerStats := &job.MinerStats{}
	if err = json.NewDecoder(r.Body).Decode(minerStats); err != nil {
		log.Sugar.Errorw("error decoding miner stats",
			"method", r.Method,
			"url", r.URL,
			"err", err.Error(),
			"mID", mID,
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "error decoding miner stats request body"}
	}

	if activeMiners[mUUID] == nil {
		activeMiners[mUUID] = &activeMiner{
			ActiveWorkers: map[uuid.UUID]*activeWorker{},
		}
	}
	aMiner := activeMiners[mUUID]

	for _, wStats := range minerStats.WorkerStats {
		if !uuid.Equal(wStats.JobID, uuid.Nil) {
			if ch, ok := activeWorkers[wStats.JobID]; ok {
				ch <- struct{}{}
			} else {
				// should only happen if the pod is restarted while a job is running
				notebook, err := db.GetJobNotebook(wStats.JobID)
				if err != nil {
					log.Sugar.Errorw("error getting job notebook",
						"method", r.Method,
						"url", r.URL,
						"err", err.Error(),
						"jID", wStats.JobID,
					)
					return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
				}
				go monitorJob(wStats.JobID, notebook)
			}
		}
		if aMiner.ActiveWorkers[wStats.GPUStats.ID] == nil {
			aMiner.ActiveWorkers[wStats.GPUStats.ID] = &activeWorker{}
		}
		aWorker := aMiner.ActiveWorkers[wStats.GPUStats.ID]
		aWorker.JobID = wStats.JobID
		aWorker.lastPost = time.Now()
	}

	go func() {
		// TODO: store snapshots in gcs or DB instead of logger? kafka -> db?
		log.Sugar.Infow("miner stats",
			"mID", mID,
			"stats", minerStats,
		)

		client := &http.Client{}
		ctx := context.Background()
		// check if user has exceeded disk quota & cancel if so
		for _, wStats := range minerStats.WorkerStats {
			if !uuid.Equal(wStats.JobID, uuid.Nil) {
				diskQuota, err := db.GetJobDiskQuota(wStats.JobID)
				if err != nil {
					log.Sugar.Errorw("error getting job disk requirements",
						"method", r.Method,
						"url", r.URL,
						"err", err.Error(),
						"jID", wStats.JobID,
					)
					return
				}

				dockerDisk := wStats.DockerDisk
				if dockerDisk != nil && diskQuota != 0 {
					// TODO: should probably be uint64
					if diskQuota < (dockerDisk.SizeRw + dockerDisk.SizeRootFs +
						dockerDisk.SizeDataDir + dockerDisk.SizeOutputDir) {
						log.Sugar.Infow("user disk quota exceeded, canceling job",
							"method", r.Method,
							"url", r.URL,
							"jID", wStats.JobID,
						)

						uUUID, project, err := db.GetJobOwnerAndProject(r, wStats.JobID)
						if err != nil {
							log.Sugar.Errorw("error getting job owner and project",
								"method", r.Method,
								"url", r.URL,
								"err", err.Error(),
								"jID", wStats.JobID,
							)
							return
						}
						token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
							"aud":   "emrys.io",
							"exp":   time.Now().Add(time.Minute * 5).Unix(),
							"iss":   "emrys.io",
							"iat":   time.Now().Unix(),
							"sub":   uUUID,
							"scope": []string{"user"},
						})
						authToken, err := token.SignedString([]byte(authSecret))
						if err != nil {
							log.Sugar.Errorw("error signing token",
								"method", r.Method,
								"url", r.URL,
								"err", err.Error(),
								"jID", wStats.JobID,
							)
							return
						}

						p := path.Join("user", "project", project, "job", wStats.JobID.String(), "cancel")
						u := url.URL{
							Scheme: "http",
							Host:   "user-svc:8080",
							Path:   p,
						}
						operation := func() error {
							req, err := http.NewRequest(http.MethodPost, u.String(), nil)
							if err != nil {
								return err
							}
							req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", authToken))
							req = req.WithContext(ctx)

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
								log.Sugar.Errorw("error posting user cancellation to user-svc, retrying",
									"method", r.Method,
									"url", r.URL,
									"err", err.Error(),
									"jID", wStats.JobID,
								)
							}); err != nil {
							log.Sugar.Errorw("error posting user cancellation to user-svc--aborting",
								"method", r.Method,
								"url", r.URL,
								"err", err.Error(),
								"jID", wStats.JobID,
							)
							return
						}
					}
				}
			}
		}
	}()

	return nil
}
