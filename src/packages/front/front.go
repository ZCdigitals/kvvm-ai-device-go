package front

import (
	"device-go/src/libs/exec"
	"device-go/src/libs/socket"
)

const (
	FrontMessageTypeStatus uint32 = 0x00000001

	FrontMessageTypeTranscriptStart  uint32 = 0x10000001
	FrontMessageTypeTranscriptStop   uint32 = 0x10000002
	FrontMessageTypeTranscriptCancel uint32 = 0x10000003
	FrontMessageTypeTranscriptEnd    uint32 = 0x10000004
	FrontMessageTypeTranscriptData   uint32 = 0x10000010

	FrontMessageTypeLog uint32 = 0x20000000

	FrontMessageTypeAgentList uint32 = 0x30000000

	FrontMessageTypeApprovalStart  uint32 = 0x40000000
	FrontMessageTypeApprovalAccept uint32 = 0x40000001
	FrontMessageTypeApprovalDeny   uint32 = 0x40000002
	FrontMessageTypeApprovalCancel uint32 = 0x40000003
	FrontMessageTypeApprovalEnd    uint32 = 0x40000004

	FrontMessageTypeWorkflowRecordStart uint32 = 0x50000000
	FrontMessageTypeWorkflowRecordSave  uint32 = 0x50000001
	FrontMessageTypeWorkflowRecordPause uint32 = 0x50000002

	FrontMessageTypeError uint32 = 0xffffffff
)

type Front struct {
	ex     exec.Exec
	socket socket.Socket

	OnTranscriptStart     func()
	OnTranscriptStop      func()
	OnTranscriptCancel    func()
	OnApprovalAccept      func()
	OnApprovalDeny        func()
	OnApprovalCancel      func()
	OnWorkflowRecortStart func()
	OnWorkflowRecortSave  func()
	OnWorkflowRecortPause func()
}

func NewFront(binPath string, socketPath string, version string) Front {
	return Front{
		ex: exec.NewExec(
			binPath,
			"--main-version", version,
			"--message-path", socketPath,
		),
		socket: socket.NewSocket(socketPath),
	}
}

func (f *Front) Open() error {
	err := f.socket.Open()
	if err != nil {
		return err
	}

	err = f.ex.Start()
	if err != nil {
		f.socket.Close()
		return err
	}

	f.socket.OnData = func(header socket.SocketHeader, body []byte) {
		// todo
	}

	return nil
}

func (f *Front) Close() {
	f.ex.Stop()
	f.socket.Close()
}
