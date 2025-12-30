package socket

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"
)

type SocketOnData func(header SocketHeader, body []byte)

type Socket struct {
	path string

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
