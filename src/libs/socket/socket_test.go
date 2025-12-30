package socket

import (
	"bytes"
	"encoding/binary"
	"net"
	"path/filepath"
	"testing"
	"time"
)

func useTestHeader(size uint32) []byte {
	hb := []byte{
		0x12, 0x34, 0x56, 0x78, // id
		0x12, 0x34, 0x56, 0x78, // size
		0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78, // timestamp
		0x12, 0x34, 0x56, 0x7a, // reserved 0
		0x12, 0x34, 0x56, 0x7a, // reserved 1
		0x12, 0x34, 0x56, 0x7a, // reserved 2
		0x12, 0x34, 0x56, 0x7a, // reserved 3
		0x12, 0x34, 0x56, 0x7a, // reserved 4
		0x12, 0x34, 0x56, 0x7a, // reserved 5
		0x12, 0x34, 0x56, 0x7a, // reserved 6
		0x12, 0x34, 0x56, 0x7a, // reserved 7
	}

	binary.LittleEndian.PutUint32(hb[4:], size)

	return hb
}

func useTestBody() []byte {
	return []byte("Hello, Socket!")
}

func TestSocket(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.sock")

	s := NewSocket(path)

	headers := make(chan SocketHeader, 10)
	bodys := make(chan []byte, 10)

	s.OnData = func(header SocketHeader, body []byte) {
		headers <- header

		if body == nil {
			return
		}

		bodys <- body
	}

	// start
	err := s.Open()
	if err != nil {
		t.Fatalf("socket open error %v", err)
	}
	defer s.Close()

	// wait for start
	time.Sleep(100 * time.Millisecond)

	// client
	client, err := net.Dial("unix", path)
	if err != nil {
		t.Fatalf("connect error %v", err)
	}
	defer client.Close()

	// send no body data
	t.Run("should send no body data right", func(t *testing.T) {
		header := useTestHeader(0x00)

		_, err := client.Write(header)
		if err != nil {
			t.Errorf("send header error %v", err)
		}

		select {
		case h := <-headers:
			{
				if h.Size != 0 {
					t.Errorf("header size not match %d 0", h.Size)
				}
			}
		case <-time.After(time.Second):
			{
				t.Error("timeout")
			}
		}
	})

	t.Run("should send data with body right", func(t *testing.T) {
		body := useTestBody()
		size := uint32(len(body))
		header := useTestHeader(size)

		_, err := client.Write(header)
		if err != nil {
			t.Errorf("send header error %v", err)
		}

		_, err = client.Write(body)
		if err != nil {
			t.Errorf("send body error %v", err)
		}

		select {
		case h := <-headers:
			{
				if h.Size != size {
					t.Errorf("header size not match %d %d", h.Size, size)
				}

				b := <-bodys
				{
					if !bytes.Equal(b, body) {
						t.Errorf("body not same %s %s", b, body)
					}
				}
			}
		case <-time.After(time.Second):
			{
				t.Error("timeout")
			}
		}
	})
}
