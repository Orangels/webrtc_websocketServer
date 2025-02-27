// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/exp/slog"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	//maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1048576,
	WriteBufferSize: 1048576,
}

type Channel struct {
	// websocket 连接.
	conn *websocket.Conn

	//
	observer ChannelObserver

	// Buffered channel of outbound messages.
	output chan []byte
}

func NewChannel(conn *websocket.Conn, observer ChannelObserver) *Channel {
	return &Channel{
		conn:     conn,
		observer: observer,
		output:   make(chan []byte, 1024),
	}
}

func (c *Channel) Startup() {
	go c.readPump()
	go c.writePump()
}

func (c *Channel) Write(data []byte) {
	c.output <- data
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Channel) readPump() {
	defer func() {
		c.observer.OnChannelClose(c)
		c.conn.Close()
	}()
	//c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Error("Read websocket message failed!", err)
			}
			break
		}

		msg := NewMessage(c, message)
		GetManager().PostMessage(msg)
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Channel) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		c.observer.OnChannelClose(c)
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.output:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
