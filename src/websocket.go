package src

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"
)

type WebSocketOnMessage func(message []byte)

type WebSocket struct {
	id  string
	url string
	key string

	onMessage WebSocketOnMessage
	onClose   func()

	running uint32

	connection *websocket.Conn

	mux sync.RWMutex
}

func (ws *WebSocket) openConnection() error {
	header := http.Header{}
	header.Add("x-device-key", ws.key)

	u, err := url.Parse(ws.url)
	if err != nil {
		log.Fatalf("websocket url parse error %v\n", err)
	}
	u.Path = fmt.Sprintf("/ws/device/%s/response", ws.id)

	connection, _, err := websocket.DefaultDialer.Dial(u.String(), header)
	if err != nil {
		log.Println("webscoket open error", err)
		return err
	}

	ws.connection = connection
	ws.connection.SetCloseHandler(func(code int, text string) error {
		ws.Close()
		return nil
	})
	log.Println("websocket open")

	return nil
}

func (ws *WebSocket) closeConnection() error {
	if ws.connection == nil {
		return nil
	}

	err := ws.connection.Close()
	ws.connection = nil
	log.Println("websocket close ")

	return err
}

func (ws *WebSocket) handle() {
	defer ws.closeConnection()

	for ws.isRunning() {
		_, msg, err := ws.connection.ReadMessage()
		if err != nil {
			log.Println("websocket read message error", err)
			continue
		} else if ws.onMessage != nil {
			ws.onMessage(msg)
		}
	}
}

func (ws *WebSocket) isRunning() bool {
	return atomic.LoadUint32(&ws.running) == 1
}

func (ws *WebSocket) setRunning(running bool) {
	if running {
		atomic.StoreUint32(&ws.running, 1)
	} else {
		atomic.StoreUint32(&ws.running, 0)
	}
}

func (ws *WebSocket) Open() error {
	ws.setRunning(true)

	err := ws.openConnection()
	if err != nil {
		return err
	}

	go ws.handle()

	return nil
}

func (ws *WebSocket) Close() {
	if ws.onClose != nil {
		ws.onClose()
	}

	ws.setRunning(false)
}

func (ws *WebSocket) Send(message any) error {
	if ws.connection == nil {
		return fmt.Errorf("websocket connection is null")
	} else if !ws.isRunning() {
		return fmt.Errorf("websocket is not running")
	}

	j, err := json.Marshal(message)
	if err != nil {
		log.Println("websocket json marshal error", err)
		return err
	}

	ws.mux.Lock()
	defer ws.mux.Unlock()
	err = ws.connection.WriteMessage(websocket.BinaryMessage, j)

	return err
}
