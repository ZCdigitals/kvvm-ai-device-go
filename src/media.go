package src

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
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

type MediaSocketOnData func(header *MediaFrameHeader, frame []byte)

type MediaSocket struct {
	path       string
	listener   net.Listener
	connection net.Conn
	onData     MediaSocketOnData

	cmd     *exec.Cmd
	running bool
}

func NewMediaSocket(path string) MediaSocket {
	return MediaSocket{
		path:    path,
		running: false,
	}
}

func (m *MediaSocket) Init() error {
	var err error

	// delete exists
	os.Remove(m.path)

	// start listen
	m.listener, err = net.Listen("unix", m.path)
	if err != nil {
		return err
	}

	// chmod
	// err = os.Chmod(m.path, 0666)
	// if err != nil {
	// 	return err
	// }

	m.cmd = exec.Command(VIDEO_CMD,
		"-w", "1920",
		"-h", "1080",
		"-i", "/dev/video0",
		"-o", m.path,
	)

	err = m.cmd.Start()
	if err != nil {
		m.listener.Close()
		return fmt.Errorf("media socket cmd start error %s", err)
	}

	go m.accept()

	return nil
}

func (m *MediaSocket) accept() {
	c, err := m.listener.Accept()
	if err != nil {
		log.Printf("media socket accept error %s\n", err)
		return
	}
	m.connection = c
	defer m.Close()

	m.handle()
}

func (m *MediaSocket) handle() {
	if m.connection == nil {
		return
	}

	// data is h264 encoded, 1MB should be enough
	buffer := make([]byte, 1024*1024)

	var header MediaFrameHeader
	for m.running {
		err := m.read(buffer[:32])
		if err != nil {
			log.Printf("media read header error %s\n", err)
			return
		}

		// parse header
		header = ParseMediaFrameHeader(buffer[:32])

		// check size
		if header.size == 0 {
			log.Printf("media frame header %d size is 0\n", header.id)
			continue
		} else if header.size > uint32(len(buffer)) {
			log.Printf("media frame header %d size is too larger %d\n", header.id, header.size)
			continue
		}

		frame := make([]byte, header.size)
		err = m.read(frame)
		if err != nil {
			log.Printf("media read data error %s\n", err)
			return
		}

		if m.onData != nil {
			m.onData(&header, frame)
		}
	}
}

func (m *MediaSocket) read(buffer []byte) error {
	if m.connection == nil {
		return nil
	}

	total := 0
	for total < len(buffer) {
		n, err := m.connection.Read(buffer[total:])
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		total += n
	}

	if total != len(buffer) {
		return fmt.Errorf("incomplete read: expected %d, got %d", total, len(buffer))
	}

	return nil
}

func (m *MediaSocket) Close() {
	m.running = false

	if m.cmd != nil {
		err := m.cmd.Process.Signal(os.Interrupt)
		if err != nil {
			log.Printf("media cmd stop error %s\n", err)
		}
	}
	if m.connection != nil {
		err := m.connection.Close()
		if err != nil {
			log.Printf("media connection close error %s\n", err)
		}
	}
	if m.listener != nil {
		err := m.listener.Close()
		if err != nil {
			log.Printf("media listener close error %s\n", err)
		}
	}
	os.Remove(m.path)
}

type MediaRtp struct {
	device string
	ip     string
	port   int

	listener *net.UDPConn

	cmd *exec.Cmd
}

func (rtp *MediaRtp) Init() error {
	if rtp.listener != nil {
		err := rtp.listener.Close()

		if err != nil {
			log.Printf("close rtp listener error %s", err)
		}
	}

	if rtp.cmd != nil {
		err := rtp.cmd.Cancel()

		if err != nil {
			log.Printf("cancel rtp cmd error %s", err)
		}
	}

	listener, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.ParseIP(rtp.ip),
		Port: rtp.port,
	})
	if err != nil {
		return err
	}

	// Increase the UDP receive buffer size
	// Default UDP buffer sizes vary on different operating systems
	bufferSize := 300000 // 300KB
	err = listener.SetReadBuffer(bufferSize)
	if err != nil {
		return err
	}

	log.Println("rtp listen start")

	// run gstreamer
	// device := "/dev/video0"
	rtp.cmd = exec.Command("gst-launch-1.0", "-q",
		"v4l2src", "device="+rtp.device, "io-mode=mmap", "!",
		"video/x-raw,format=NV12,width=1920,height=1080", "!",
		"mpph264enc", "gop=2", "!",
		"rtph264pay", "config-interval=-1", "aggregate-mode=zero-latency", "!",
		"udpsink", "host="+rtp.ip, "port="+fmt.Sprint(rtp.port),
	)

	err = rtp.cmd.Start()
	if err != nil {
		return err
	}

	return nil
}

func (rtp *MediaRtp) Close() {
	if rtp.cmd != nil {
		err := rtp.cmd.Cancel()

		if err != nil {
			log.Printf("rtp cmd cancel error %s", err)
		}
	}

	if rtp.listener != nil {
		err := rtp.listener.Close()
		if err != nil {
			log.Printf("rtp listener close error %s", err)
		}
	}
}

func (rtp *MediaRtp) Read(b []byte) (int, error) {
	n, _, err := rtp.listener.ReadFrom(b)
	return n, err
}
