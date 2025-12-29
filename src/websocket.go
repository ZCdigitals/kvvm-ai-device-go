package src

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"
)

type WebSocketOnMessage func(message []byte)

type WebSocket struct {
	url         string
	accessToken string

	OnMessage WebSocketOnMessage
	OnClose   func()

	running uint32

	connection *websocket.Conn

	mux sync.RWMutex
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

func (ws *WebSocket) buildHeader() http.Header {
	h := http.Header{}

	h.Add("Authorization", fmt.Sprintf("Bearer %s", ws.accessToken))

	return h
}

func (ws *WebSocket) openConnection() error {
	if ws.connection != nil {
		return fmt.Errorf("webscoket connection exists")
	}

	connection, _, err := websocket.DefaultDialer.Dial(ws.url, ws.buildHeader())
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
		return fmt.Errorf("webscoket null connection")
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
		} else if ws.OnMessage != nil {
			ws.OnMessage(msg)
		}
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
	if ws.OnClose != nil {
		ws.OnClose()
	}

	ws.setRunning(false)
}

func (ws *WebSocket) Send(message any) error {
	if ws.connection == nil {
		return fmt.Errorf("websocket connection is null")
	}

	j, err := json.Marshal(message)
	if err != nil {
		log.Println("websocket json marshal error", err)
		return err
	}

	ws.mux.Lock()
	defer ws.mux.Unlock()
	return ws.connection.WriteMessage(websocket.TextMessage, j)
}
