package gstreamer

import (
	"device-go/src/libs/exec"
	"device-go/src/libs/udp"
	"fmt"
)

type GstreamerOnData func(frame []byte)

type Gstreamer struct {
	ex  exec.Exec
	udp udp.UDP

	OnData GstreamerOnData
}

func NewGstreamer(
	path string,
	ip string,
	port int,
	width uint,
	height uint,
	bitRate uint,
	gop uint,
) Gstreamer {
	return Gstreamer{
		ex: exec.NewExec(
			"gst-launch-1.0", "-q",
			// here we use `mmap` mode
			// `drm` mode will get `core dump`, i do not know why
			"v4l2src", "device="+path, "io-mode=mmap", "!",
			fmt.Sprintf("video/x-raw,format=NV12,width=%d,height=%d", width, height), "!",
			"mpph264enc", fmt.Sprintf("gop=%d", gop), "!",
			"rtph264pay", "config-interval=-1", "aggregate-mode=zero-latency", "!",
			"udpsink", "host="+ip, fmt.Sprintf("port=%d", port),
		),
		udp: udp.NewUDP(ip, port),
	}
}

func (g *Gstreamer) Open() error {
	g.udp.OnData = func(b []byte) {
		if g.OnData == nil {
			return
		}
		g.OnData(b)
	}

	err := g.udp.Open()
	if err != nil {
		return err
	}

	err = g.ex.Start()
	if err != nil {
		g.udp.Close()
		return err
	}
	return nil
}

func (g *Gstreamer) Close() {
	g.ex.Stop()
	g.udp.Close()
}
