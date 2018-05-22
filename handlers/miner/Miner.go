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
	// miner ID
	ID uuid.UUID

	// miner pool
	pool *pool

	// websocket connection
	conn *websocket.Conn

	// buffered channel for outbound jobs
	sendMsg chan []byte
}

// func (m *miner) readPump() {
// 	defer func() {
// 		m.pool.unregister <- m
// 		if err := m.conn.Close(); err != nil {
// 			log.Printf("Error defer-closing websocket: %v\n", err)
// 		}
// 	}()
//
// 	err := m.conn.SetReadDeadline(time.Now().Add(pongWait))
// 	if err != nil {
// 		log.Printf("Error setting websocket read deadline: %v\n", err)
// 	}
// 	m.conn.SetPongHandler(func(string) error {
// 		err = m.conn.SetReadDeadline(time.Now().Add(pongWait))
// 		if err != nil {
// 			log.Printf("Error setting websocket read deadline: %v\n", err)
// 		}
// 		return nil
// 	})
//
// 	for {
// 		msgType, r, err := m.conn.NextReader()
// 		if err != nil {
// 			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
// 				log.Printf("Error unexpected websocket close: %v\n", err)
// 			}
// 			break
// 		}
// 		log.Printf("Message received!\n")
// 		switch msgType {
// 		case websocket.BinaryMessage:
// 			zr, err := zlib.NewReader(r)
// 			if err != nil {
// 				log.Printf("Error decompressing message: %v\n", err)
// 				break
// 			}
// 			b := &job.Bid{}
// 			err = gob.NewDecoder(zr).Decode(b)
// 			if err != nil {
// 				log.Printf("Error decoding message: %v\n", err)
// 				break
// 			}
// 			err = zr.Close()
// 			if err != nil {
// 				log.Printf("Error closing zlib reader: %v\n", err)
// 				break
// 			}
// 			b.ID = uuid.NewV4()
// 			b.MinerID = m.ID
// 			log.Printf("Received bid: %+v\n", b)
// 			select {
// 			case m.pool.Bids[b.JobID] <- b:
// 			default:
// 				// TODO: should this bid be saved in db still?
// 				log.Printf("Late bid: %+v\n", b)
// 				m.sendText <- []byte("Your bid was received after the auction closed. Please try again.")
// 			}
// 		case websocket.TextMessage:
// 			log.Printf("Miner: %v TextMessage: \n", m)
// 			_, err = io.Copy(os.Stdout, r)
// 			if err != nil {
// 				log.Printf("Error copying websocket.TextMessage to os.Stdout: %v\n", err)
// 			}
// 		default:
// 			log.Printf("Non-text or -binary websocket message received. Closing.\n")
// 			break
// 		}
// 	}
// }

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
		// case rp, ok := <-m.sendImg:
		// 	r := *rp
		// 	err := m.conn.SetWriteDeadline(time.Now().Add(longWriteWait))
		// 	if err != nil {
		// 		log.Printf("Error setting websocket write deadline: %v\n", err)
		// 	}
		// 	if !ok {
		// 		err := m.conn.WriteMessage(websocket.CloseMessage, []byte{})
		// 		if err != nil {
		// 			log.Printf("Error writing close message to websocket: %v\n", err)
		// 		}
		// 		return
		// 	}
		// 	w, err := m.conn.NextWriter(websocket.BinaryMessage)
		// 	if err != nil {
		// 		log.Printf("Error getting next socket writer: %v\n", err)
		// 		return
		// 	}
		// 	zw := zlib.NewWriter(w)
		// 	n, err := io.Copy(zw, r)
		// 	if err != nil {
		// 		log.Printf("Error copying image reader to zlib writer: %v\n", err)
		// 		log.Printf("n: %v\n", n)
		// 		break
		// 	}
		// 	err = r.Close()
		// 	if err != nil {
		// 		log.Printf("Error closing image reader: %v\n", err)
		// 		break
		// 	}
		// 	err = zw.Close()
		// 	if err != nil {
		// 		log.Printf("Error closing zlib writer: %v\n", err)
		// 		break
		// 	}
		// 	err = w.Close()
		// 	if err != nil {
		// 		log.Printf("Error closing websocket writer: %v\n", err)
		// 		break
		// 	}
		// case msg, ok := <-m.sendText:
		// 	err := m.conn.SetWriteDeadline(time.Now().Add(writeWait))
		// 	if err != nil {
		// 		log.Printf("Error setting websocket write deadline: %v\n", err)
		// 	}
		// 	if !ok {
		// 		err = m.conn.WriteMessage(websocket.CloseMessage, []byte{})
		// 		if err != nil {
		// 			log.Printf("Error writing close message to websocket: %v\n", err)
		// 		}
		// 		return
		// 	}
		// 	err = m.conn.WriteMessage(websocket.TextMessage, msg)
		// 	if err != nil {
		// 		log.Printf("Error writing msg to socket: %v\n", err)
		// 		return
		// 	}
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
