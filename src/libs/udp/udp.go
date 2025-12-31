package udp

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
)

type UDPOnData func(b []byte)

type UDP struct {
	ip   string
	port int

	cancel context.CancelFunc
	wg     sync.WaitGroup

	connection   *net.UDPConn
	connectionMu sync.RWMutex

	OnData UDPOnData
}

func NewUDP(ip string, port int) UDP {
	return UDP{
		ip:   ip,
		port: port,
	}
}

const udpBufferSize = 3000

func (u *UDP) openConnection(ctx context.Context) error {
	u.connectionMu.Lock()
	defer u.connectionMu.Unlock()

	if u.connection != nil {
		return fmt.Errorf("udp connection exists")
	}

	addr := net.UDPAddr{
		IP:   net.ParseIP(u.ip),
		Port: u.port,
	}
	c, err := net.ListenUDP(
		"udp",
		&addr,
	)
	if err != nil {
		return err
	}
	u.connection = c

	err = c.SetReadBuffer(udpBufferSize)
	if err != nil {
		u.closeConnection()
		return err
	}

	go u.handle(ctx)

	return nil
}

func (u *UDP) closeConnection() error {
	u.connectionMu.Lock()
	defer u.connectionMu.Unlock()

	if u.connection == nil {
		return fmt.Errorf("udp null connection")
	}

	err := u.connection.Close()
	u.connection = nil

	return err
}

const udpFrameBufferSize = 1600 // udp mtu

func (u *UDP) handle(ctx context.Context) {
	u.wg.Add(1)
	defer func() {
		u.closeConnection()
		u.wg.Done()
	}()

	buffer := make([]byte, udpFrameBufferSize)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			{
				// read frame
				n, err := u.read(buffer)
				if err == io.EOF {
					return
				} else if err != nil {
					log.Println("udp read error", u.ip, err)
				}

				if u.OnData != nil {
					u.OnData(buffer[:n])
				}
			}
		}
	}
}

func (u *UDP) read(b []byte) (int, error) {
	u.connectionMu.RLock()
	defer u.connectionMu.RUnlock()

	conn := u.connection

	// avoid null connection
	if conn == nil {
		return 0, fmt.Errorf("udp null connection")
	}

	n, _, err := conn.ReadFrom(b)

	return n, err
}

func (u *UDP) Open() error {
	err := u.Open()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	u.cancel = cancel

	go u.openConnection(ctx)

	return nil
}

func (u *UDP) Close() {
	if u.cancel != nil {
		u.cancel()
		u.cancel = nil
	}

	u.wg.Wait()
}
