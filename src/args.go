package src

import (
	"flag"
	"fmt"
	"log"
	"os"
)

type Args struct {
	Id string

	MqttUrl string
	WsUrl   string

	MediaSource     uint
	VideoPath       string
	VideoBinPath    string
	VideoSocketPath string

	HidPath    string
	HidUdcPath string

	FrontBinPath    string
	FrontSocketPath string

	Version bool
	Help    bool
}

func ParseArgs() Args {
	var id string

	var mqttUrl string
	var wsUrl string

	var mediaSource uint
	var videoPath string
	var videoBinPath string
	var videoSocketPath string

	var hidPath string
	var hidUdcPath string

	var frontBinPath string
	var frontSocketPath string

	var version bool
	var help bool

	flag.StringVar(&id, "id", "", "device serial no")

	flag.StringVar(&mqttUrl, "mqtt-url", "", "Mqtt broker url")
	flag.StringVar(&wsUrl, "ws-url", "", "Websoket url")

	flag.UintVar(&mediaSource, "media-source", 1, "Media source, 1 video, 2 gstreamer")
	flag.StringVar(&videoPath, "video-path", "/dev/video0", "Video path")
	flag.StringVar(&videoBinPath, "video-bin-path", "/root/video", "Video bin path")
	flag.StringVar(&videoSocketPath, "video-socket-path", "/var/run/capture.sock", "Video socket path")

	flag.StringVar(&hidPath, "hid-path", "/dev/hidg0", "HID path")
	flag.StringVar(&hidUdcPath, "hid-udc-path", "/sys/class/udc", "HID UDC path")

	flag.StringVar(&frontBinPath, "front-bin-path", "/root/font", "Front bin path")
	flag.StringVar(&frontSocketPath, "front-socket-path", "/var/run/front.sock", "Front socket path")

	flag.BoolVar(&version, "version", false, "Print version")
	flag.BoolVar(&help, "help", false, "Print help")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		help = true
	}

	if mqttUrl == "" {
		log.Fatalln("Mqtt url is required")
	} else if wsUrl == "" {
		log.Fatalln("Ws url is required")
	}

	return Args{
		Id: id,

		MqttUrl: mqttUrl,
		WsUrl:   wsUrl,

		MediaSource:     mediaSource,
		VideoPath:       videoPath,
		VideoBinPath:    videoBinPath,
		VideoSocketPath: videoSocketPath,

		HidPath:    hidPath,
		HidUdcPath: hidUdcPath,

		FrontBinPath:    frontBinPath,
		FrontSocketPath: frontSocketPath,

		Version: version,
		Help:    help,
	}
}
