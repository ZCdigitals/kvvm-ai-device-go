package websocket

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	TextMessage   = websocket.TextMessage
	BinaryMessage = websocket.BinaryMessage
	CloseMessage  = websocket.CloseMessage
	PingMessage   = websocket.PingMessage
	PongMessage   = websocket.PongMessage
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
	// set a short timeout, this will stop read
	ws.connection.SetReadDeadline(time.Now().Add(10 * time.Millisecond))

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
	ws.wg.Add(1)
	defer func() {
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
					nerr, ok := err.(net.Error)
					if ok && nerr.Timeout() {
						// time out
						return
					}

					log.Println("websocket read error", ws.url, err)
					return
				}
			}
		}
	}
}

func (ws *WebSocket) read() error {
	ws.connectionMu.RLock()
	defer ws.connectionMu.RUnlock()

	t, msg, err := ws.connection.ReadMessage()
	if err != nil {

		return err
	}

	switch t {
	case TextMessage, BinaryMessage:
		if ws.OnMessage != nil {
			ws.OnMessage(t, msg)
		}
		break
	case PingMessage:
		ws.connection.WriteMessage(PongMessage, msg)
		break
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

	go ws.handle(ctx)

	return nil
}

func (ws *WebSocket) Close() {
	ws.closeConnection()

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

	log.Println("ws send", message)

	if ws.connection == nil {
		return fmt.Errorf("websocket connection is null")
	}

	return ws.connection.WriteJSON(message)
}

func (ws *WebSocket) SendBinary(b []byte) error {
	ws.connectionMu.RLock()
	defer ws.connectionMu.RUnlock()

	log.Println("ws send binary", b)

	if ws.connection == nil {
		return fmt.Errorf("websocket connection is null")
	}

	return ws.connection.WriteMessage(websocket.BinaryMessage, b)
}
