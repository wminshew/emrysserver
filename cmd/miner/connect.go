package main

import (
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/app"
	"github.com/wminshew/emrysserver/pkg/log"
	"net/http"
	"time"
)

var upgrader = websocket.Upgrader{
	HandshakeTimeout: 10 * time.Second,
	ReadBufferSize:   1024,
	WriteBufferSize:  1024,
}

// connect handles miner client requests to /miner/connect,
// establishing a websocket and moving connection into an
// available worker Pool
func connect() app.Handler {
	return func(w http.ResponseWriter, r *http.Request) *app.Error {
		vars := mux.Vars(r)
		mID := vars["mID"]
		mUUID, err := uuid.FromString(mID)
		if err != nil {
			log.Sugar.Errorw("failed to parse miner ID",
				"url", r.URL,
				"err", err.Error(),
				"mID", mID,
			)
			return &app.Error{Code: http.StatusBadRequest, Message: "failed to parse miner ID in path"}
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Sugar.Errorw("failed to upgrade connection",
				"url", r.URL,
				"err", err.Error(),
				"mID", mID,
			)
			return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
		}

		m := &miner{
			ID:      mUUID,
			pool:    p,
			conn:    conn,
			sendMsg: make(chan []byte),
		}
		m.pool.register <- m

		go m.writePump()

		return nil
	}
}
