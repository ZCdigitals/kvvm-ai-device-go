package video

import (
	"device-go/src/libs/exec"
	"device-go/src/libs/socket"
)

type VideoMonitor struct {
	ex     exec.Exec
	socket socket.Socket

	IsConnected bool
	width       uint32
	height      uint32
}

func NewVideoMonitor(
	path string,
	binPath string,
	socketPath string,
) VideoMonitor {
	return VideoMonitor{
		ex: exec.NewExec(
			binPath,
			"-d", path,
			"-s", socketPath,
		),
		socket: socket.NewSocket(socketPath),
	}
}

func (vm *VideoMonitor) Open() error {
	vm.socket.OnData = func(header socket.SocketHeader, body []byte) {
		// connect status
		vm.IsConnected = header.Reserved[0] == 2
		// width
		vm.width = header.Reserved[1]
		// height
		vm.height = header.Reserved[2]
	}

	err := vm.socket.Open()
	if err != nil {
		return err
	}

	err = vm.ex.Start()
	if err != nil {
		vm.socket.Close()
		return err
	}

	return nil
}

func (vm *VideoMonitor) Close() {
	vm.ex.Stop()
	vm.socket.Close()
}
