package src

import (
	"flag"
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

	VideoMonitorPath       string
	VideoMonitorBinPath    string
	VideoMonitorSocketPath string

	HidPath string

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
	var videoMonitorPath string
	var videoMonitorBinPath string
	var videoMonitorSocketPath string

	var hidPath string

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
	flag.StringVar(&videoMonitorPath, "video-monitor-path", "/dev/v4l-subdev2", "Video sub device path")
	flag.StringVar(&videoMonitorBinPath, "video-monitor-bin-path", "/root/video-monitor", "Video monitor bin path")
	flag.StringVar(&videoMonitorSocketPath, "video-monitor-socket-path", "/var/run/monitor.sock", "Video monitor socket path")

	flag.StringVar(&hidPath, "hid-path", "/dev/hidg0", "HID path")

	flag.StringVar(&frontBinPath, "front-bin-path", "/root/font", "Front bin path")
	flag.StringVar(&frontSocketPath, "front-socket-path", "/var/run/front.sock", "Front socket path")

	flag.BoolVar(&version, "version", false, "Print version")
	flag.BoolVar(&help, "help", false, "Print help")

	flag.Usage = func() {
		log.Println("Usage of", os.Args[0])
		flag.PrintDefaults()
		help = true
	}

	// parse
	flag.Parse()

	// valid args
	if id == "" {
		log.Fatalln("ID is required")
	} else if mqttUrl == "" {
		log.Fatalln("Mqtt url is required")
	} else if wsUrl == "" {
		log.Fatalln("Ws url is required")
	}

	return Args{
		Id: id,

		MqttUrl: mqttUrl,
		WsUrl:   wsUrl,

		MediaSource:            mediaSource,
		VideoPath:              videoPath,
		VideoBinPath:           videoBinPath,
		VideoSocketPath:        videoSocketPath,
		VideoMonitorPath:       videoMonitorPath,
		VideoMonitorBinPath:    videoMonitorBinPath,
		VideoMonitorSocketPath: videoMonitorSocketPath,

		HidPath: hidPath,

		FrontBinPath:    frontBinPath,
		FrontSocketPath: frontSocketPath,

		Version: version,
		Help:    help,
	}
}
