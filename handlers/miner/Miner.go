package miner

import (
	"github.com/gorilla/websocket"
	"log"
	"time"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 1024
)

var (
	newline = []byte{'\n'}
)

type miner struct {
	// miner Pool
	pool *Pool

	// websocket connection
	conn *websocket.Conn

	// buffered channel for outbound messages
	send chan []byte
}

func (m *miner) readPump() {
	defer func() {
		m.pool.unregister <- m
		if err := m.conn.Close(); err != nil {
			log.Printf("Error defer-closing websocket: %v\n", err)
		}
	}()

	m.conn.SetReadLimit(maxMessageSize)
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
		case message, ok := <-m.send:
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

			w, err := m.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				log.Printf("Error making next websocket writer: %v\n", err)
				return
			}
			_, err = w.Write(message)
			if err != nil {
				log.Printf("Error writing message to websocket: %v\n", err)
			}

			// send any other queued messages
			n := len(m.send)
			for i := 0; i < n; i++ {
				_, err = w.Write(newline)
				if err != nil {
					log.Printf("Error writing message to websocket: %v\n", err)
				}
				_, err = w.Write(<-m.send)
				if err != nil {
					log.Printf("Error writing message to websocket: %v\n", err)
				}
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
