package src

import (
	"fmt"
	"log"
	"net"
	"os/exec"
)

type Rtp struct {
	addr net.UDPAddr

	listener *net.UDPConn

	cmd *exec.Cmd
}

func NewRtp(ip string, port int) Rtp {
	return Rtp{
		addr: net.UDPAddr{
			IP:   net.ParseIP(ip),
			Port: port,
		},
	}
}

func (rtp *Rtp) listen() {
	listener, err := net.ListenUDP("udp", &rtp.addr)
	if err != nil {
		log.Fatal("rtp listen error ", err)
	}

	rtp.listener = listener

	// Increase the UDP receive buffer size
	// Default UDP buffer sizes vary on different operating systems
	bufferSize := 300000 // 300KB
	err = listener.SetReadBuffer(bufferSize)
	if err != nil {
		log.Fatal("rtp set buffer size error ", err)
	}

	log.Println("rtp listen start")

	// run gstreamer
	device := "/dev/video0"
	rtp.cmd = exec.Command("gst-launch-1.0", "-q",
		"v4l2src", "device="+device, "io-mode=mmap", "!",
		"video/x-raw,format=NV12,width=1920,height=1080", "!",
		"mpph264enc", "gop=2", "!",
		"rtph264pay", "config-interval=-1", "aggregate-mode=zero-latency", "!",
		"udpsink", "host="+string(rtp.addr.IP), "port="+fmt.Sprint(rtp.addr.Port),
	)

	err = rtp.cmd.Start()
	if err != nil {
		log.Fatal("run gstreamer error ", err)
	}
}

func (rtp *Rtp) Read(b []byte) (int, error) {
	n, _, err := rtp.listener.ReadFrom(b)
	return n, err
}

func (rtp *Rtp) Init() {
	rtp.listen()
}

func (rtp *Rtp) Close() error {
	if rtp.cmd != nil {
		err := rtp.cmd.Cancel()

		if err != nil {
			return err
		}
	}

	if rtp.listener != nil {
		err := rtp.listener.Close()

		if err != nil {
			return err
		}
	}

	return nil
}
