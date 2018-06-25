// Package miner contains handlers related to how the server connects to miners
package miner

import (
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/app"
	"net/http"
	"time"
)

var upgrader = websocket.Upgrader{
	HandshakeTimeout: 10 * time.Second,
	ReadBufferSize:   1024,
	WriteBufferSize:  1024,
}

// Connect handles miner client requests to /miner/connect,
// establishing a websocket and moving connection into an
// available worker Pool
func Connect(w http.ResponseWriter, r *http.Request) *app.Error {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		app.Sugar.Errorw("failed to upgrade connection",
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusInternalServerError, Message: "internal error"}
	}

	vars := mux.Vars(r)
	mID := vars["mID"]
	mUUID, err := uuid.FromString(mID)
	if err != nil {
		app.Sugar.Errorw("failed to parse miner ID",
			"url", r.URL,
			"err", err.Error(),
		)
		return &app.Error{Code: http.StatusBadRequest, Message: "failed to parse miner ID in path"}
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
