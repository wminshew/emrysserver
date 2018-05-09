package miner

import (
	"compress/zlib"
	"encoding/gob"
	"github.com/gorilla/websocket"
	"github.com/wminshew/emrys/pkg/job"
	"log"
	"time"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
	// maxMessageSize = 1024
)

var (
	newline = []byte{'\n'}
)

type miner struct {
	// miner pool
	pool *pool

	// websocket connection
	conn *websocket.Conn

	// buffered channel for outbound jobs
	sendJob chan *job.Job
}

func (m *miner) readPump() {
	defer func() {
		m.pool.unregister <- m
		if err := m.conn.Close(); err != nil {
			log.Printf("Error defer-closing websocket: %v\n", err)
		}
	}()

	// m.conn.SetReadLimit(maxMessageSize)
	err := m.conn.SetReadDeadline(time.Now().Add(pongWait))
	if err != nil {
		log.Printf("Error setting websocket read deadline: %v\n", err)
	}
	m.conn.SetPongHandler(func(string) error {
		err = m.conn.SetReadDeadline(time.Now().Add(pongWait))
		if err != nil {
			log.Printf("Error setting websocket read deadline: %v\n", err)
		}
		return nil
	})
	for {
		_, message, err := m.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Error unexpected websocket close: %v\n", err)
			}
			break
		}
		log.Printf("Miner: %v Message: %s\n", m, string(message))
	}
}

func (m *miner) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		if err := m.conn.Close(); err != nil {
			log.Printf("Error defer-closing websocket: %v\n", err)
		}
	}()

	for {
		select {
		case j, ok := <-m.sendJob:
			err := m.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err != nil {
				log.Printf("Error setting websocket write deadline: %v\n", err)
			}
			if !ok {
				err = m.conn.WriteMessage(websocket.CloseMessage, []byte{})
				if err != nil {
					log.Printf("Error writing close message to websocket: %v\n", err)
				}
				return
			}

			w, err := m.conn.NextWriter(websocket.BinaryMessage)
			if err != nil {
				log.Printf("Error making next websocket writer: %v\n", err)
				return
			}
			// TODO: seems inefficient to do this for every miner individually? Need to refactor
			// maybe put in BroadcastJob; write to buffer, then buffer.Bytes to send
			zw := zlib.NewWriter(w)
			enc := gob.NewEncoder(zw)
			err = enc.Encode(j)
			if err != nil {
				log.Printf("Error encoding, compressing, and sending job: %v\n", err)
			}

			// send any other queued jobs
			n := len(m.sendJob)
			for i := 0; i < n; i++ {
				_, err = w.Write(newline)
				if err != nil {
					log.Printf("Error writing newline to websocket: %v\n", err)
				}
				err = enc.Encode(j)
				if err != nil {
					log.Printf("Error encoding, compressing, and sending job: %v\n", err)
				}
			}

			if err := zw.Close(); err != nil {
				log.Printf("Error closing websocket writer: %v\n", err)
				return
			}
			if err := w.Close(); err != nil {
				log.Printf("Error closing websocket writer: %v\n", err)
				return
			}
		case <-ticker.C:
			err := m.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err != nil {
				log.Printf("Error setting write deadline: %v\n", err)
			}
			if err = m.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Error pinging websocket: %v\n", err)
				return
			}
		}

	}
}
