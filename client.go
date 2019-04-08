package main

import (
	"log"
	"time"
	"encoding/json"
	"github.com/fatih/color"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 2048
)

var (
	newline    = []byte{'\n'}
	space      = []byte{' '}
	msgCounter = int32(0)
)

func newMsgNum() int32 {
	msgCounter++
	return msgCounter
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	playerID     string
	gameId *string
	wasConnected bool
	version      int // implementirano u verziji 54
	deviceUUID   *string
	hub          *Hub
	conn         *websocket.Conn
	send         chan []byte
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) readPump() {
	defer func() {
		c.hub.chUnregisterClient <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("websocket.IsUnexpectedCloseError: %v", err)
			}
			break
		}

		c.processMessage(message)
	}
}

func (c *Client) processMessage(message []byte) {
	var dic map[string]interface{}
	if err := json.Unmarshal(message, &dic); err != nil {
		panic(err)
	}
	log.Println(dic)

	if dic["msg_num"] == nil {
		dic["msg_num"] = float64(newMsgNum())
	} else {
		log.Println("msg_num is not nil")
	}

	color.Magenta("%v", dic)

	js, err := json.Marshal(dic)
	if err != nil {
		panic(err)
	}

	action := &Action{
		name:         dic["msg_func"].(string),
		message:      js,
		fromPlayerID: c.playerID,
		msgNum:       int32(dic["msg_num"].(float64)),
	}
	
	c.hub.chAction <- action
}


// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
		log.Println("Client closed")
	}()
	for {
		select {
		case message, ok := <-c.send:
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

			// Add queued chat messages to the current websocket message.
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-c.send)
			}

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
