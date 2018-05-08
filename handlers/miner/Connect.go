// Package miner contains handlers related to how the server connects to miners
package miner

import (
	"github.com/gorilla/websocket"
	"github.com/wminshew/emrys/pkg/job"
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

	m := &miner{
		pool:    Pool,
		conn:    conn,
		sendJob: make(chan *job.Job),
	}
	m.pool.register <- m

	go m.writePump()
	go m.readPump()
}
