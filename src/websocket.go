package src

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

type WebSocketOnMessage func(message []byte)

type WebSocket struct {
	url       string
	key       string
	onMessage WebSocketOnMessage

	running bool

	connection *websocket.Conn
}

func (ws *WebSocket) Init() {
	ws.running = true

	header := http.Header{}
	header.Add("x-device-key", ws.key)
	connection, _, err := websocket.DefaultDialer.Dial(ws.url, header)
	if err != nil {
		log.Printf("webscoket init error %s", err)
		return
	}
	defer connection.Close()

	ws.connection = connection

	// recevice message
	go func() {
		for ws.running {
			_, msg, err := connection.ReadMessage()
			if err != nil {
				log.Printf("websocket read message error %s", err)
				return
			} else if ws.onMessage != nil {
				ws.onMessage(msg)
			}
		}
	}()
}

func (ws *WebSocket) Close() {
	ws.running = false
}

func (ws *WebSocket) Send(message any) {
	if ws.connection == nil {
		return
	}

	j, err := json.Marshal(message)
	if err != nil {
		log.Fatalf("json string error %s", err)
	}

	ws.connection.WriteMessage(websocket.BinaryMessage, j)
}
