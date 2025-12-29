package src

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"sync/atomic"
)

type VideoMonitorReport struct {
	isConnected bool
	width       uint32
	height      uint32
}

func ParseVideoMonitorReport(b []byte) VideoMonitorReport {
	return VideoMonitorReport{
		isConnected: binary.LittleEndian.Uint32(b[0:4]) == 2,
		width:       binary.LittleEndian.Uint32(b[4:8]),
		height:      binary.LittleEndian.Uint32(b[8:12]),
	}
}

const videoMonitorReportLength = 12

type VideoMonitor struct {
	path       string
	binPath    string
	socketPath string

	isConnected bool
	width       uint32
	height      uint32

	running uint32

	listener   *net.Listener
	connection *net.Conn

	cmd *exec.Cmd
}

func (m *VideoMonitor) isRunning() bool {
	return atomic.LoadUint32(&m.running) == 1
}

func (m *VideoMonitor) setRunning(running bool) {
	if running {
		atomic.StoreUint32(&m.running, 1)
	} else {
		atomic.StoreUint32(&m.running, 0)
	}
}

func (m *VideoMonitor) openListener() error {
	if m.listener != nil {
		return fmt.Errorf("video monitor listener exists")
	}

	// delete exists
	os.Remove(m.socketPath)

	// start listen
	l, err := net.Listen("unix", m.socketPath)
	if err != nil {
		log.Println("video monitor listener open error", err)
		return err
	}
	m.listener = &l

	return nil
}

func (m *VideoMonitor) closeListener() error {
	if m.listener == nil {
		return fmt.Errorf("video monitor null listener")
	}

	err := (*m.listener).Close()
	if err != nil {
		log.Println("video monitor listener close error", err)
	}
	m.listener = nil
	os.Remove(m.socketPath)

	return err
}

func (m *VideoMonitor) openConnection() error {
	// avoid null listener
	if m.listener == nil {
		return fmt.Errorf("video monitor null listener")
	} else if m.connection != nil {
		return fmt.Errorf("video monitor connection exists")
	}

	c, err := (*m.listener).Accept()
	if err != nil {
		log.Println("video monitor connection open error", err)
		return err
	}
	m.connection = &c

	return nil
}

func (m *VideoMonitor) closeConnection() error {
	if m.connection == nil {
		return fmt.Errorf("video monitor null connection")
	}

	err := (*m.connection).Close()
	if err != nil {
		log.Println("video monitor connection close error", err)
	}
	m.connection = nil

	return err
}

func (m *VideoMonitor) startCmd() error {
	// avoid null listener
	if m.listener == nil {
		return fmt.Errorf("video monitor null listener")
	} else if m.cmd != nil {
		return fmt.Errorf("video monitor cmd exists")
	}

	m.cmd = exec.Command(m.binPath,
		"-d", m.path,
		"-s", m.socketPath,
	)

	// chmod
	// err = os.Chmod(m.path, 0666)
	// if err != nil {
	// 	return err
	// }

	err := m.cmd.Start()
	if err != nil {
		log.Println("video monitor cmd start error", err)
		return err
	}

	return nil
}

func (m *VideoMonitor) stopCmd() error {
	if m.cmd == nil {
		return fmt.Errorf("video monitor null cmd")
	}

	err := m.cmd.Process.Signal(os.Interrupt)
	if err != nil {
		log.Println("video monitor cmd stop error", err)
	}
	m.cmd = nil

	return err
}

func (m *VideoMonitor) close() {
	m.stopCmd()
	m.closeConnection()
	m.closeListener()
}

func (m *VideoMonitor) read(buffer []byte) error {
	// avoid null connection
	if m.connection == nil {
		return fmt.Errorf("video monitor null connection")
	}

	total := 0
	for total < len(buffer) && m.isRunning() {
		n, err := (*m.connection).Read(buffer[total:])
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		total += n
	}

	if !m.isRunning() {
		return fmt.Errorf("video monitor is closing")
	}

	if total != len(buffer) {
		return fmt.Errorf("video monitor incomplete read: expected %d, got %d", len(buffer), total)
	}

	return nil
}

func (m *VideoMonitor) handle() {
	defer m.close()

	buffer := make([]byte, videoMonitorReportLength)

	for m.isRunning() {
		err := m.read(buffer)
		if err != nil {
			log.Println("video monitor read error", err)
			return
		}

		report := ParseVideoMonitorReport(buffer)

		m.isConnected = report.isConnected
		m.width = report.width
		m.height = report.height
	}
}

func (m *VideoMonitor) Open() error {
	err := m.openListener()
	if err != nil {
		return err
	}

	err = m.startCmd()

	if err != nil {
		m.close()
		return err
	}

	err = m.openConnection()
	if err != nil {
		m.close()
		return err
	}

	m.setRunning(true)
	go m.handle()

	return nil
}

func (m *VideoMonitor) Close() {
	m.setRunning(false)
}
