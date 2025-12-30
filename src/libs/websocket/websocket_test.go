package websocket

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

type testServer struct {
	*httptest.Server
	upgrader websocket.Upgrader

	messages    []string
	connections []*websocket.Conn
	mu          sync.RWMutex
}

func (ts *testServer) handleWebSocket(w http.ResponseWriter, req *http.Request) {
	// use auth
	token := req.Header.Get("Authorization")
	if token != "Bearer test-token" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// upgrade
	conn, err := ts.upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Println("test server upgrade error", err)
		return
	}
	defer conn.Close()

	// store connection
	ts.mu.Lock()
	ts.connections = append(ts.connections, conn)
	ts.mu.Unlock()

	for {
		// read
		t, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("test server read error", err)
			break
		}

		// store message
		ts.mu.Lock()
		ts.messages = append(ts.messages, string(msg))
		ts.mu.Unlock()

		// send back
		err = conn.WriteMessage(t, msg)
		if err != nil {
			log.Println("test server send error", err)
		}
	}
}

func newTestServer(t *testing.T) *testServer {
	ts := testServer{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", ts.handleWebSocket)

	ts.Server = httptest.NewServer(mux)

	return &ts
}

func TestWebSocket(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	ws := NewWebSocket(
		fmt.Sprintf("ws://%s/ws", ts.Listener.Addr().String()),
		"test-token",
	)
	defer ws.Close()

	if len(ts.connections) > 0 {
		t.Errorf("test server has connections %d", len(ts.connections))
		return
	}

	t.Run("should open right", func(t *testing.T) {
		err := ws.Open()
		if err != nil {
			t.Errorf("open error %v", err)
		}

		// wait connect
		time.Sleep(100 * time.Millisecond)

		if len(ts.connections) == 0 {
			t.Errorf("test server has no connection")
		}
	})

	if len(ts.messages) > 0 {
		t.Errorf("test server has messages %d", len(ts.messages))
		return
	}

	t.Run("should send right", func(t *testing.T) {
		err := ws.Send([]byte("hello websockt"))
		if err != nil {
			t.Errorf("send error %v", err)
		}

		// wait send
		time.Sleep(100 * time.Millisecond)

		if len(ts.messages) == 0 {
			t.Errorf("test server has no message")
		}
	})
}
