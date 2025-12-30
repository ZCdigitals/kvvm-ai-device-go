package socket

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

type SocketOnData func(header SocketHeader, body []byte)

type Socket struct {
	path string

	messageId uint32

	cancel context.CancelFunc
	wg     sync.WaitGroup

	listener     net.Listener
	connection   net.Conn
	connectionMu sync.RWMutex

	OnData SocketOnData
}

func NewSocket(path string) Socket {
	return Socket{path: path}
}

func (s *Socket) openListener() error {
	if s.listener != nil {
		return fmt.Errorf("socket listener exists")
	}

	// delete exists
	os.Remove(s.path)

	// start
	l, err := net.Listen("unix", s.path)
	if err != nil {
		return err
	}
	s.listener = l

	return nil
}

func (s *Socket) closeListener() error {
	// avoid null listener
	if s.listener == nil {
		return fmt.Errorf("socket null listener")
	}

	// close
	err := s.listener.Close()
	s.listener = nil
	os.Remove(s.path)

	return err
}

func (s *Socket) openConnection(ctx context.Context) error {
	s.connectionMu.Lock()
	defer s.connectionMu.Unlock()

	// avoid null listener
	if s.listener == nil {
		return fmt.Errorf("socket null listener")
	} else if s.connection != nil {
		return fmt.Errorf("socket connection exists")
	}

	// accept
	c, err := s.listener.Accept()
	if err != nil {
		s.closeListener()
		return err
	}
	s.connection = c

	s.wg.Add(1)
	go s.handle(ctx)

	return nil
}

func (s *Socket) closeConnection() error {
	s.connectionMu.Lock()
	defer s.connectionMu.Unlock()

	// avoid null connection
	if s.connection == nil {
		return fmt.Errorf("socket null connection")
	}

	// close
	err := s.connection.Close()
	s.connection = nil

	return err
}

func (s *Socket) handle(ctx context.Context) {
	defer func() {
		s.closeConnection()
		s.closeListener()
		s.wg.Done()
	}()

	// header buffer
	hb := make([]byte, SocketHeaderLength)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			{
				// read header
				err := s.read(ctx, hb)
				if err != nil {
					log.Println("socket read header error", err)
					return
				}

				header := ParseSocketHeader(hb)

				if header.Size == 0 {
					// no body
					if s.OnData != nil {
						s.OnData(header, nil)
					}
					continue
				}

				// body
				body := make([]byte, header.Size)
				// read body
				err = s.read(ctx, body)
				if err != nil {
					log.Println("socket read body error", err)
					return
				}

				if s.OnData != nil {
					s.OnData(header, body)
				}
			}
		}

	}
}

func (s *Socket) read(ctx context.Context, buffer []byte) error {
	s.connectionMu.RLock()
	defer s.connectionMu.RUnlock()

	conn := s.connection

	// avoid null connection
	if conn == nil {
		return fmt.Errorf("socket null connection")
	}

	total := 0
	for total < len(buffer) {
		select {
		case <-ctx.Done():
			return fmt.Errorf("socket is closing")
		default:
			{
				n, err := conn.Read(buffer[total:])
				if err == io.EOF {
					break
				} else if err != nil {
					return err
				}

				total += n
			}
		}
	}

	if total != len(buffer) {
		// not enough length
		return fmt.Errorf("socket incomplete read: expected %d, got %d", len(buffer), total)
	}

	return nil
}

func (s *Socket) useMessageId() uint32 {
	id := atomic.LoadUint32(&s.messageId)

	atomic.AddUint32(&s.messageId, 1)

	return id
}

func (s *Socket) send(header [8]uint32, body []byte) error {
	size := uint32(0)
	if body != nil {
		size = uint32(len(body))
	}

	// send header
	h := SocketHeader{
		ID:        s.useMessageId(),
		Size:      size,
		Timestamp: uint64(time.Now().UnixMicro()),
		Reserved:  header,
	}
	err := s.write(h.ToBytes())
	if err != nil {
		return err
	}

	// no body
	if size == 0 {
		return nil
	}

	// send body
	return s.write(body)
}

func (s *Socket) write(b []byte) error {
	s.connectionMu.RLock()
	defer s.connectionMu.RUnlock()

	// avoid null connection
	if s.connection == nil {
		return fmt.Errorf("socket null connection")
	}

	_, err := s.connection.Write(b)

	return err
}

func (s *Socket) Open() error {
	err := s.openListener()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel

	go s.openConnection(ctx)

	return nil
}

func (s *Socket) Close() {
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}

	s.wg.Wait()
}

func (s *Socket) Send(header [8]uint32, body []byte) error {
	return s.send(header, body)
}

func (s *Socket) SendHeader(header [8]uint32) error {
	return s.send(header, nil)
}

func (s *Socket) SendBody(body []byte) error {
	return s.send([8]uint32{}, body)
}
