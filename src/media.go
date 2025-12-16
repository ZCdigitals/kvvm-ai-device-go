package src

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"strconv"
	"sync/atomic"
)

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
	frameHeaderLength = 32
	// frame is h264 encoded, 1MB should be enough
	frameLengthMax = 1 * 1024 * 1024
)

type MediaVideoOnData func(header *MediaFrameHeader, frame []byte)

type MediaVideo struct {
	path       string
	binPath    string
	socketPath string

	width   uint
	height  uint
	bitRate uint
	gop     uint

	running uint32

	listener   *net.Listener
	connection *net.Conn

	cmd *exec.Cmd

	onData MediaVideoOnData
}

func NewMediaVideo(width uint, height uint, path string, binPath string, socketPath string, bitRate uint, gop uint) MediaVideo {
	return MediaVideo{
		width:      width,
		height:     height,
		path:       path,
		binPath:    binPath,
		socketPath: socketPath,
		bitRate:    bitRate,
		gop:        gop,
		running:    0,
	}
}

func (m *MediaVideo) openListener() error {
	// delete exists
	os.Remove(m.socketPath)

	// start listen
	l, err := net.Listen("unix", m.socketPath)
	if err != nil {
		log.Println("media socket listener open error", err)
		return err
	}
	m.listener = &l

	return nil
}

func (m *MediaVideo) closeListener() error {
	if m.listener != nil {
		err := (*m.listener).Close()
		if err != nil {
			log.Println("media socket listener close error", err)
		}
		m.listener = nil
		os.Remove(m.socketPath)

		return err
	}

	return nil
}

func (m *MediaVideo) openConnection() error {
	// avoid null listener
	if m.listener == nil {
		return fmt.Errorf("media socket null listener")
	}

	c, err := (*m.listener).Accept()
	if err != nil {
		log.Println("media socket connection open error", err)
		return err
	}
	m.connection = &c

	return nil
}

func (m *MediaVideo) closeConnection() error {
	if m.connection != nil {
		err := (*m.connection).Close()
		if err != nil {
			log.Println("media socket connection close error", err)
		}
		m.connection = nil

		return err
	}

	return nil
}

func (m *MediaVideo) startCmd() error {
	// avoid null listener
	if m.listener == nil {
		return fmt.Errorf("media socket null listener")
	}

	m.cmd = exec.Command(m.binPath,
		"-w", strconv.FormatUint(uint64(m.width), 10),
		"-h", strconv.FormatUint(uint64(m.height), 10),
		"-i", m.path,
		"-o", m.socketPath,
		"-b", strconv.FormatUint(uint64(m.bitRate), 10),
		"-g", strconv.FormatUint(uint64(m.gop), 10),
	)

	// chmod
	// err = os.Chmod(m.path, 0666)
	// if err != nil {
	// 	return err
	// }

	err := m.cmd.Start()
	if err != nil {
		log.Println("media socket cmd start error", err)
		return err
	}

	return nil
}

func (m *MediaVideo) stopCmd() error {
	if m.cmd != nil {
		err := m.cmd.Process.Signal(os.Interrupt)
		if err != nil {
			log.Println("media socket cmd stop error", err)
		}
		m.cmd = nil

		return err
	}

	return nil
}

func (m *MediaVideo) close() {
	m.stopCmd()
	m.closeConnection()
	m.closeListener()
}

func (m *MediaVideo) handle() {
	defer m.close()

	headerBuffer := make([]byte, frameHeaderLength)

	for m.isRunning() {
		err := m.read(headerBuffer)
		if err != nil {
			log.Println("media socket read header error", err)
			return
		}

		// parse header
		header := ParseMediaFrameHeader(headerBuffer)

		// check size
		if header.size == 0 {
			log.Println("media socket frame", header.id, "size is 0")
			continue
		} else if header.size > frameLengthMax {
			log.Println("media socket frame", header.id, "size is too larger", header.size)
			continue
		}

		frameBuffer := make([]byte, header.size)
		err = m.read(frameBuffer)
		if err != nil {
			log.Println("media socket read data error", err)
			return
		}

		if m.onData != nil {
			m.onData(&header, frameBuffer)
		}
	}
}

func (m *MediaVideo) read(buffer []byte) error {
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

func (m *MediaVideo) isRunning() bool {
	return atomic.LoadUint32(&m.running) == 1
}

func (m *MediaVideo) setRunning(running bool) {
	if running {
		atomic.StoreUint32(&m.running, 1)
	} else {
		atomic.StoreUint32(&m.running, 0)
	}
}

func (m *MediaVideo) Open() error {
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

func (m *MediaVideo) Close() {
	m.setRunning(false)
}

type MediaGstOnData func(frame []byte)

type MediaGst struct {
	width      uint
	height     uint
	inputPath  string
	outputIp   string
	outputPort int
	bitRate    uint
	gop        uint

	running uint32

	connection *net.UDPConn

	cmd *exec.Cmd

	onData MediaGstOnData
}

func NewMediaGst(width uint, height uint, inputPath string, outputIp string, outputPort int, bitRate uint, gop uint) MediaGst {
	return MediaGst{
		width:      width,
		height:     height,
		inputPath:  inputPath,
		outputIp:   outputIp,
		outputPort: outputPort,
		bitRate:    bitRate,
		gop:        gop,
		running:    0,
	}
}

func (m *MediaGst) openConnection() error {
	c, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.ParseIP(m.outputIp),
		Port: m.outputPort,
	})
	if err != nil {
		log.Println("media rtp connection open error", err)
		return err
	}

	// Increase the UDP receive buffer size
	// Default UDP buffer sizes vary on different operating systems
	bufferSize := 300000 // 300KB
	err = c.SetReadBuffer(bufferSize)
	if err != nil {
		log.Println("media rtp connection set buffer error", err)
		return err
	}

	m.connection = c

	return nil
}

func (m *MediaGst) closeConnection() error {
	if m.connection != nil {
		err := m.connection.Close()
		if err != nil {
			log.Println("rtp listener close error", err)
		}

		return err
	}

	return nil
}

func (m *MediaGst) startCmd() error {
	// avoid null connection
	if m.cmd == nil {
		return fmt.Errorf("media rtp null connection")
	}

	m.cmd = exec.Command("gst-launch-1.0", "-q",
		// here we use `mmap` mode
		// `drm` mode will get `core dump`, i do not know why
		"v4l2src", "device="+m.inputPath, "io-mode=mmap", "!",
		fmt.Sprintf("video/x-raw,format=NV12,width=%d,height=%d", m.width, m.height), "!",
		"mpph264enc", fmt.Sprintf("gop=%d", m.gop), "!",
		"rtph264pay", "config-interval=-1", "aggregate-mode=zero-latency", "!",
		"udpsink", "host="+m.outputIp, fmt.Sprintf("port=%d", m.outputPort),
	)

	err := m.cmd.Start()
	if err != nil {
		log.Println("media rtp cmd start error", err)
		return err
	}

	return nil
}

func (m *MediaGst) stopCmd() error {
	if m.cmd != nil {
		err := m.cmd.Process.Signal(os.Interrupt)
		if err != nil {
			log.Println("media socket cmd stop error", err)
		}
		m.cmd = nil

		return err
	}

	return nil
}

func (m *MediaGst) close() {
	m.stopCmd()
	m.closeConnection()
}

func (m *MediaGst) handle() {
	defer m.close()

	for m.isRunning() {
		frameBuffer := make([]byte, 1600) // UDP MTU

		// avoid null connection
		if m.connection == nil {
			return
		}

		n, _, err := m.connection.ReadFrom(frameBuffer)
		if err == io.EOF {
			return
		} else if err != nil {
			log.Println("media rtp read data error", err)
			return
		}

		if m.onData != nil {
			m.onData(frameBuffer[:n])
		}
	}
}

func (m *MediaGst) isRunning() bool {
	return atomic.LoadUint32(&m.running) == 1
}

func (m *MediaGst) setRunning(running bool) {
	if running {
		atomic.StoreUint32(&m.running, 1)
	} else {
		atomic.StoreUint32(&m.running, 0)
	}
}

func (m *MediaGst) Open() error {
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

func (m *MediaGst) Close() {
	m.setRunning(false)
}
