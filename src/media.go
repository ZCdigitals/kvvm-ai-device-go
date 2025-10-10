package src

import (
	"fmt"
	"log"
	"net"
	"os/exec"
)

type Rtp struct {
	device string
	ip     string
	port   int

	listener *net.UDPConn

	cmd *exec.Cmd
}

func (rtp *Rtp) Init() {
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
		log.Fatalf("rtp listen error %s", err)
	}

	// Increase the UDP receive buffer size
	// Default UDP buffer sizes vary on different operating systems
	bufferSize := 300000 // 300KB
	err = listener.SetReadBuffer(bufferSize)
	if err != nil {
		log.Fatalf("rtp set buffer size error %s", err)
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
		log.Fatalf("run gstreamer error %s", err)
	}
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

func (rtp *Rtp) Read(b []byte) (int, error) {
	n, _, err := rtp.listener.ReadFrom(b)
	return n, err
}
