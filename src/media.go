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

const VIDEO_CMD string = "/root/video"

type MediaFrameHeader struct {
	id     uint32
	width  uint32
	height uint32
	// pixel color format, only support nv12
	//
	// 0 nv12
	format    uint32
	timestamp uint64
	size      uint32
	reserved  uint32
}

func ParseMediaFrameHeader(b []byte) MediaFrameHeader {
	return MediaFrameHeader{
		id:        binary.LittleEndian.Uint32(b[0:4]),
		width:     binary.LittleEndian.Uint32(b[4:8]),
		height:    binary.LittleEndian.Uint32(b[8:12]),
		format:    binary.LittleEndian.Uint32(b[12:16]),
		timestamp: binary.LittleEndian.Uint64(b[16:24]),
		size:      binary.LittleEndian.Uint32(b[24:28]),
		reserved:  binary.LittleEndian.Uint32(b[28:32]),
	}
}

const (
	headerLength = 32
	// frame is h264 encoded, 1MB should be enough
	maxFrameSize = 1 * 1024 * 1024
)

type MediaSocketOnData func(header *MediaFrameHeader, frame []byte)

type MediaSocket struct {
	path string

	running uint32

	listener   *net.Listener
	connection *net.Conn

	cmd *exec.Cmd

	onData MediaSocketOnData
}

func NewMediaSocket(path string) MediaSocket {
	return MediaSocket{
		path:    path,
		running: 0,
	}
}

func (m *MediaSocket) openListener() error {
	// delete exists
	os.Remove(m.path)

	// start listen
	l, err := net.Listen("unix", m.path)
	if err != nil {
		log.Printf("media socket listener open error %v\n", err)
		return err
	}
	m.listener = &l

	return nil
}

func (m *MediaSocket) closeListener() error {
	if m.listener != nil {
		err := (*m.listener).Close()
		if err != nil {
			log.Printf("media socket listener close error %v\n", err)
		}
		m.listener = nil
		os.Remove(m.path)

		return err
	}

	return nil
}

func (m *MediaSocket) openConnection() error {
	// avoid null listener
	if m.listener == nil {
		return fmt.Errorf("media socket null listener")
	}

	c, err := (*m.listener).Accept()
	if err != nil {
		log.Printf("media socket connection open error %s\n", err)
		return err
	}
	m.connection = &c

	return nil
}

func (m *MediaSocket) closeConnection() error {
	if m.connection != nil {
		err := (*m.connection).Close()
		if err != nil {
			log.Printf("media socket connection close error %s\n", err)
		}
		m.connection = nil

		return err
	}

	return nil
}

func (m *MediaSocket) startCmd() error {
	// avoid null listener
	if m.listener == nil {
		return fmt.Errorf("media socket null listener")
	}

	m.cmd = exec.Command(VIDEO_CMD,
		"-w", "1920",
		"-h", "1080",
		"-i", "/dev/video0",
		"-o", m.path,
	)

	// chmod
	// err = os.Chmod(m.path, 0666)
	// if err != nil {
	// 	return err
	// }

	err := m.cmd.Start()
	if err != nil {
		log.Printf("media socket cmd start error %v\n", err)
		return err
	}

	return nil
}

func (m *MediaSocket) stopCmd() error {
	if m.cmd != nil {
		err := m.cmd.Process.Signal(os.Interrupt)
		if err != nil {
			log.Printf("media socket cmd stop error %s\n", err)
		}
		m.cmd = nil

		return err
	}

	return nil
}

func (m *MediaSocket) close() {
	m.stopCmd()
	m.closeConnection()
	m.closeListener()
}

func (m *MediaSocket) handle() {
	defer m.close()

	headerBuffer := make([]byte, headerLength)

	for m.isRunning() {
		err := m.read(headerBuffer)
		if err != nil {
			log.Printf("media socket read header error %s\n", err)
			return
		}

		// parse header
		header := ParseMediaFrameHeader(headerBuffer)

		// check size
		if header.size == 0 {
			log.Printf("media socket frame %d size is 0\n", header.id)
			continue
		} else if header.size > maxFrameSize {
			log.Printf("media socket frame %d size is too larger %d\n", header.id, header.size)
			continue
		}

		frameBuffer := make([]byte, header.size)
		err = m.read(frameBuffer)
		if err != nil {
			log.Printf("media socket read data error %v\n", err)
			return
		}

		if m.onData != nil {
			m.onData(&header, frameBuffer)
		}
	}
}

func (m *MediaSocket) read(buffer []byte) error {
	// avoid null connection
	if m.connection == nil {
		return fmt.Errorf("media socket null connection")
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
		return fmt.Errorf("media socket is closing")
	}

	if total != len(buffer) {
		return fmt.Errorf("media socket incomplete read: expected %d, got %d", len(buffer), total)
	}

	return nil
}

func (m *MediaSocket) isRunning() bool {
	return atomic.LoadUint32(&m.running) == 1
}

func (m *MediaSocket) setRunning(running bool) {
	if running {
		atomic.StoreUint32(&m.running, 1)
	} else {
		atomic.StoreUint32(&m.running, 0)
	}
}

func (m *MediaSocket) Open() error {
	m.setRunning(true)

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

	go m.handle()

	return nil
}

func (m *MediaSocket) Close() {
	m.setRunning(false)
}

type MediaRtpOnData func(frame []byte)

type MediaRtp struct {
	device string
	ip     string
	port   int

	width  uint
	height uint

	running uint32

	connection *net.UDPConn

	cmd *exec.Cmd

	onData MediaRtpOnData
}

func (m *MediaRtp) openConnection() error {
	c, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.ParseIP(m.ip),
		Port: m.port,
	})
	if err != nil {
		log.Printf("media rtp connection open error %v\n", err)
		return err
	}

	// Increase the UDP receive buffer size
	// Default UDP buffer sizes vary on different operating systems
	bufferSize := 300000 // 300KB
	err = c.SetReadBuffer(bufferSize)
	if err != nil {
		log.Printf("media rtp connection set buffer error %v\n", err)
		return err
	}

	m.connection = c

	return nil
}

func (m *MediaRtp) closeConnection() error {
	if m.connection != nil {
		err := m.connection.Close()
		if err != nil {
			log.Printf("rtp listener close error %v", err)
		}

		return err
	}

	return nil
}

func (m *MediaRtp) startCmd() error {
	// avoid null connection
	if m.cmd == nil {
		return fmt.Errorf("media rtp null connection")
	}

	m.cmd = exec.Command("gst-launch-1.0", "-q",
		"v4l2src", "device="+m.device, "io-mode=mmap", "!",
		fmt.Sprintf("video/x-raw,format=NV12,width=%d,height=%d", m.width, m.height), "!",
		"mpph264enc", "gop=60", "!",
		"rtph264pay", "config-interval=-1", "aggregate-mode=zero-latency", "!",
		"rtpsink", "host="+m.ip, "port="+fmt.Sprint(m.port),
	)

	err := m.cmd.Start()
	if err != nil {
		log.Printf("media rtp cmd start error %v\n", err)
		return err
	}

	return nil
}

func (m *MediaRtp) stopCmd() error {
	if m.cmd != nil {
		err := m.cmd.Process.Signal(os.Interrupt)
		if err != nil {
			log.Printf("media socket cmd stop error %s\n", err)
		}
		m.cmd = nil

		return err
	}

	return nil
}

func (m *MediaRtp) close() {
	m.stopCmd()
	m.closeConnection()
}

func (m *MediaRtp) handle() {
	defer m.close()

	for m.isRunning() {
		frameBuffer := make([]byte, 1600) // UDP MTU
		n, _, err := m.connection.ReadFrom(frameBuffer)
		if err == io.EOF {
			return
		} else if err != nil {
			log.Printf("media rtp read data error %v\n", err)
			return
		}

		if m.onData != nil {
			m.onData(frameBuffer[:n])
		}
	}
}

func (m *MediaRtp) read(buffer []byte) (int, error) {
	// avoid null connection
	if m.connection == nil {
		return 0, fmt.Errorf("media rtp null connection")
	}

	n, _, err := m.connection.ReadFrom(buffer)
	if err == io.EOF {
		return 0, nil
	}

	return n, nil
}

func (m *MediaRtp) isRunning() bool {
	return atomic.LoadUint32(&m.running) == 1
}

func (m *MediaRtp) setRunning(running bool) {
	if running {
		atomic.StoreUint32(&m.running, 1)
	} else {
		atomic.StoreUint32(&m.running, 0)
	}
}

func (m *MediaRtp) Open() error {
	m.setRunning(true)

	err := m.openConnection()
	if err != nil {
		return err
	}

	err = m.startCmd()
	if err != nil {
		m.closeConnection()
		return err
	}

	go m.handle()

	return nil
}

func (m *MediaRtp) Close() {
	m.setRunning(false)
}
