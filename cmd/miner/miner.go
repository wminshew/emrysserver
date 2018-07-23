package main

import (
	"github.com/gorilla/websocket"
	"github.com/satori/go.uuid"
	"github.com/wminshew/emrysserver/pkg/app"
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
			app.Sugar.Errorw("failed to close websocket",
				"miner", m.ID,
				"err", err.Error(),
			)
		}
	}()

	for {
		select {
		case msg, ok := <-m.sendMsg:
			err := m.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err != nil {
				app.Sugar.Errorw("failed to set websocket write deadline",
					"miner", m.ID,
					"err", err.Error(),
				)
			}
			if !ok {
				err = m.conn.WriteMessage(websocket.CloseMessage, []byte{})
				if err != nil {
					app.Sugar.Errorw("failed to close websocket",
						"miner", m.ID,
						"err", err.Error(),
					)
				}
				return
			}
			err = m.conn.WriteMessage(websocket.BinaryMessage, msg)
			if err != nil {
				app.Sugar.Errorw("failed to write to websocket",
					"miner", m.ID,
					"err", err.Error(),
				)
				return
			}

			// send queued messages
			n := len(m.sendMsg)
			for i := 0; i < n; i++ {
				err = m.conn.WriteMessage(websocket.BinaryMessage, <-m.sendMsg)
				if err != nil {
					app.Sugar.Errorw("failed to write to websocket",
						"miner", m.ID,
						"err", err.Error(),
					)
				}
			}
		case <-ticker.C:
			err := m.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err != nil {
				app.Sugar.Errorw("failed to set websocket write deadline",
					"miner", m.ID,
					"err", err.Error(),
				)
			}
			if err = m.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				app.Sugar.Errorw("failed to ping websocket",
					"miner", m.ID,
					"err", err.Error(),
				)
				return
			}
		}

	}
}
