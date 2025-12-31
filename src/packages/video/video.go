package video

import (
	"device-go/src/libs/exec"
	"device-go/src/libs/socket"
	"strconv"
)

type VideoOnData func(id uint32, timestamp uint64, frame []byte)

type Video struct {
	ex     exec.Exec
	socket socket.Socket

	OnData VideoOnData
}

func NewVideo(
	path string,
	binPath string,
	socketPath string,
	width uint,
	height uint,
	bitRate uint,
	gop uint,
) Video {
	return Video{
		ex: exec.NewExec(
			binPath,
			"-w", strconv.FormatUint(uint64(width), 10),
			"-h", strconv.FormatUint(uint64(height), 10),
			"-i", path,
			"-o", socketPath,
			"-b", strconv.FormatUint(uint64(bitRate), 10),
			"-g", strconv.FormatUint(uint64(gop), 10),
		),
		socket: socket.NewSocket(socketPath),
	}
}

func (v *Video) Open() error {
	v.socket.OnData = func(header socket.SocketHeader, body []byte) {
		if v.OnData == nil {
			return
		}
		v.OnData(header.ID, header.Timestamp, body)
	}

	err := v.socket.Open()
	if err != nil {
		return err
	}

	err = v.ex.Start()
	if err != nil {
		v.socket.Close()
		return err
	}

	return nil
}

func (v *Video) Close() {
	v.ex.Stop()
	v.socket.Close()
}
