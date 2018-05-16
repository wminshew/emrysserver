// Package miner contains handlers related to how the server connects to miners
package miner

import (
	"github.com/gorilla/websocket"
	"github.com/satori/go.uuid"
	"io"
	"log"
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
func Connect(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error upgrading connection to websocket: %v\n", err)
		return
	}

	ctxKey := contextKey("miner_uuid")
	u, ok := r.Context().Value(ctxKey).(uuid.UUID)
	if !ok {
		log.Printf("miner_uuid in request context corrupted\n")
		http.Error(w, "Unable to retrieve valid uuid from jwt. Please login again.", http.StatusInternalServerError)
		return
	}
	m := &miner{
		ID:       u,
		pool:     Pool,
		conn:     conn,
		sendJob:  make(chan []byte),
		sendText: make(chan []byte),
		sendImg:  make(chan *io.ReadCloser),
	}
	m.pool.register <- m

	go m.writePump()
	go m.readPump()
}
