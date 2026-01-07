package mdns

import (
	"fmt"
	"net"
	"sync"

	MDNS "github.com/pion/mdns/v2"
	"golang.org/x/net/ipv4"
)

type Mdns struct {
	connection   *MDNS.Conn
	connectionMu sync.Mutex

	names []string
}

func NewMdns(names []string) Mdns {
	return Mdns{
		names: names,
	}
}

func (m *Mdns) openConnection() error {
	m.connectionMu.Lock()
	defer m.connectionMu.Unlock()

	if m.connection != nil {
		return fmt.Errorf("mdns connection exists")
	}

	addr, err := net.ResolveUDPAddr("udp4", MDNS.DefaultAddressIPv4)
	if err != nil {
		return err
	}
	listener, err := net.ListenUDP("udp4", addr)
	if err != nil {
		return err
	}
	packet := ipv4.NewPacketConn(listener)

	config := MDNS.Config{
		LocalNames: m.names,
	}

	c, err := MDNS.Server(packet, nil, &config)
	if err != nil {
		listener.Close()
		return err
	}

	m.connection = c

	return nil
}

func (m *Mdns) closeConnection() error {
	m.connectionMu.Lock()
	defer m.connectionMu.Unlock()

	if m.connection == nil {
		return fmt.Errorf("mdns null connection")
	}

	m.connection.Close()
	m.connection = nil

	return nil
}

func (m *Mdns) Open() error {
	return m.openConnection()
}

func (m *Mdns) Close() error {
	return m.closeConnection()
}
