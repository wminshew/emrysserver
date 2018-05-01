// Package miner contains handlers related to how the server connects to miners
package miner

import (
	"github.com/gorilla/websocket"
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
func Connect(p *Pool) func(w http.ResponseWriter, r *http.Request) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("Error upgrading connection to websocket: %v\n", err)
			return
		}

		miner := &miner{
			pool: p,
			conn: conn,
			send: make(chan []byte, 256),
		}
		miner.pool.register <- miner

		go miner.writePump()
		go miner.readPump()
	})
}
