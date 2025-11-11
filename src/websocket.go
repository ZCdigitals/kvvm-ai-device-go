package src

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"

	"github.com/gorilla/websocket"
)

type WebSocketOnMessage func(message []byte)

type WebSocket struct {
	id        string
	url       string
	key       string
	onMessage WebSocketOnMessage

	running bool

	connection *websocket.Conn
	mux        sync.RWMutex
}

func (ws *WebSocket) Init() {
	ws.running = true

	header := http.Header{}
	header.Add("x-device-key", ws.key)

	u, err := url.Parse(ws.url)
	if err != nil {
		log.Fatalf("websocket url parse error %s", err)
	}
	u.Path = fmt.Sprintf("/ws/device/%s/response", ws.id)

	connection, _, err := websocket.DefaultDialer.Dial(u.String(), header)
	if err != nil {
		log.Printf("webscoket init error %s", err)
		return
	}

	ws.connection = connection

	// recevice message
	go func() {
		defer connection.Close()

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

	ws.mux.Lock()
	defer ws.mux.Unlock()
	ws.connection.WriteMessage(websocket.BinaryMessage, j)
}
