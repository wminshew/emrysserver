package miner

import (
	"github.com/gorilla/websocket"
	"github.com/satori/go.uuid"
	"log"
	"time"
)

const (
	writeWait     = 10 * time.Second
	longWriteWait = 5 * 60 * time.Second
	pongWait      = 5 * 60 * time.Second
	pingPeriod    = (pongWait * 9) / 10
)

var (
	newline = []byte{'\n'}
)

type miner struct {
	ID      uuid.UUID
	pool    *pool
	conn    *websocket.Conn
	sendMsg chan []byte
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
		case msg, ok := <-m.sendMsg:
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
			err = m.conn.WriteMessage(websocket.BinaryMessage, msg)
			if err != nil {
				log.Printf("Error writing message to socket: %v\n", err)
				return
			}

			// send queued messages
			n := len(m.sendMsg)
			for i := 0; i < n; i++ {
				err = m.conn.WriteMessage(websocket.BinaryMessage, <-m.sendMsg)
				if err != nil {
					log.Printf("Error writing newline to websocket: %v\n", err)
				}
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
