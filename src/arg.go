package src

import (
	"flag"
	"fmt"
	"log"
	"os"
)

type Args struct {
	Id string

	MediaSource uint
	MediaPath   string
	HidPath     string

	Version bool
	Help    bool
}

func ParseArgs() Args {
	var id string

	var mediaSource uint
	var mediaPath string
	var hidPath string

	var version bool
	var help bool

	flag.StringVar(&id, "device-id", "", "Device ID")

	flag.UintVar(&mediaSource, "media-source", 1, "Media source, 1 video, 2 gstreamer")
	flag.StringVar(&mediaPath, "media-path", "", "Media path")
	flag.StringVar(&hidPath, "hid-path", "", "HID path")

	flag.BoolVar(&version, "version", false, "Print version")
	flag.BoolVar(&help, "help", false, "Print help")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		help = true
	}

	// check id
	if id == "" {
		log.Fatalln("Must input device id")
	}

	return Args{
		Id: id,

		MediaSource: mediaSource,
		MediaPath:   mediaPath,
		HidPath:     hidPath,

		Version: version,
		Help:    help,
	}
}
