package src

import (
	"log"
	"net"
)

type Rtp struct {
	addr net.UDPAddr

	listener *net.UDPConn
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
	// device := "/dev/video0"
	// cmd := exec.Command("gst-launch-1.0", "-q",
	// 	"v4l2src", "device="+device, "io-mode=mmap", "!",
	// 	"videoconvert", "!",
	// 	"x264enc", "!",
	// 	"rtph264pay", "!",
	// 	"udpsink", "host="+string(rtp.addr.IP), "port="+fmt.Sprint(rtp.addr.Port),
	// )

	// err = cmd.Run()
	// if err != nil {
	// 	log.Fatal("run gstreamer error ", err)
	// }
}

func (rtp *Rtp) Read(b []byte) (int, net.Addr, error) {
	return rtp.listener.ReadFrom(b)
}

func (rtp *Rtp) Init() {
	rtp.listen()
}

func (rtp *Rtp) Close() error {
	return rtp.listener.Close()
}
