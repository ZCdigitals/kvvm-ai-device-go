package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type WebSocketOnMessage func(messageType int, message []byte)

type WebSocket struct {
	url         string
	accessToken string

	cancel context.CancelFunc
	wg     sync.WaitGroup

	connection   *websocket.Conn
	connectionMu sync.RWMutex

	OnMessage WebSocketOnMessage
	OnClose   func()
}

func NewWebSocket(url string, accessToken string) WebSocket {
	return WebSocket{url: url, accessToken: accessToken}
}

func (ws *WebSocket) buildHeader() http.Header {
	h := http.Header{}

	h.Add("Authorization", fmt.Sprintf("Bearer %s", ws.accessToken))

	return h
}

func (ws *WebSocket) openConnection() error {
	ws.connectionMu.Lock()
	defer ws.connectionMu.Unlock()

	if ws.connection != nil {
		return fmt.Errorf("websocket connection exists")
	}

	c, _, err := websocket.DefaultDialer.Dial(ws.url, ws.buildHeader())
	if err != nil {
		return err
	}

	c.SetCloseHandler(func(code int, text string) error {
		ws.Close()
		return nil
	})

	ws.connection = c

	return nil
}

func (ws *WebSocket) closeConnection() error {
	ws.connectionMu.Lock()
	defer ws.connectionMu.Unlock()

	if ws.connection == nil {
		return fmt.Errorf("websocket null connection")
	}

	err := ws.connection.Close()
	ws.connection = nil

	return err
}

func (ws *WebSocket) handle(ctx context.Context) {
	defer func() {
		ws.closeConnection()
		ws.wg.Done()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			{
				err := ws.read()
				if err != nil {
					log.Println("websocket read error", err)
					continue
				}
			}
		}
	}
}

func (ws *WebSocket) read() error {
	ws.connectionMu.RLock()
	defer ws.connectionMu.RUnlock()

	ws.connection.SetReadDeadline(time.Now().Add(10 * time.Second))
	t, msg, err := ws.connection.ReadMessage()

	if err != nil {
		return err
	}

	if ws.OnMessage != nil {
		ws.OnMessage(t, msg)
	}

	return nil
}

func (ws *WebSocket) Open() error {
	err := ws.openConnection()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	ws.cancel = cancel
	ws.wg.Add(1)

	go ws.handle(ctx)

	return nil
}

func (ws *WebSocket) Close() {
	if ws.cancel != nil {
		ws.cancel()
		ws.cancel = nil
	}

	ws.wg.Wait()

	if ws.OnClose != nil {
		ws.OnClose()
	}
}

func (ws *WebSocket) Send(message any) error {
	ws.connectionMu.RLock()
	defer ws.connectionMu.RUnlock()

	if ws.connection == nil {
		return fmt.Errorf("websocket connection is null")
	}

	j, err := json.Marshal(message)
	if err != nil {
		log.Println("websocket json marshal error", err)
		return err
	}

	return ws.connection.WriteMessage(websocket.TextMessage, j)
}
