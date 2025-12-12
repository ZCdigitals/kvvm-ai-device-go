package src

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"sync/atomic"
	"time"
)

const (
	FrontMessageHeaderTypeStatus = 0x00000001

	FrontMessageHeaderTypeTranscriptStart  = 0x10000001
	FrontMessageHeaderTypeTranscriptStop   = 0x10000002
	FrontMessageHeaderTypeTranscriptCancel = 0x10000003
	FrontMessageHeaderTypeTranscriptEnd    = 0x10000004
	FrontMessageHeaderTypeTranscriptData   = 0x10000010

	FrontMessageHeaderTypeLog = 0x20000000

	FrontMessageHeaderTypeAgentList = 0x30000000

	FrontMessageHeaderTypeApprovalStart  = 0x40000000
	FrontMessageHeaderTypeApprovalAccept = 0x40000001
	FrontMessageHeaderTypeApprovalDeny   = 0x40000002
	FrontMessageHeaderTypeApprovalCancel = 0x40000003
	FrontMessageHeaderTypeApprovalEnd    = 0x40000004

	FrontMessageHeaderTypeWorkflowRecordStart = 0x50000000
	FrontMessageHeaderTypeWorkflowRecordSave  = 0x50000001
	FrontMessageHeaderTypeWorkflowRecordPause = 0x50000002

	FrontMessageHeaderTypeError = 0xffffffff
)

const frontMessageHeaderSize = 4 + 8 + 4 + 4

type FrontMessageHeader struct {
	id        uint32
	timestamp uint64
	msgType   uint32
	bodySize  uint32
}

func (ms *FrontMessageHeader) ToBytes() []byte {
	b := make([]byte, frontMessageHeaderSize)

	binary.LittleEndian.PutUint32(b[0:4], ms.id)
	binary.LittleEndian.PutUint64(b[4:12], ms.timestamp)
	binary.LittleEndian.PutUint32(b[12:16], ms.msgType)
	binary.LittleEndian.PutUint32(b[16:20], ms.bodySize)

	return b
}

// parse front message header from bytes
func ParseFrontMessageHeader(b []byte) FrontMessageHeader {
	return FrontMessageHeader{
		id:        binary.LittleEndian.Uint32(b[0:4]),
		timestamp: binary.LittleEndian.Uint64(b[4:12]),
		msgType:   binary.LittleEndian.Uint32(b[12:16]),
		bodySize:  binary.LittleEndian.Uint32(b[16:20]),
	}
}

const (
	frontMessageStatusSystemUnknown = 0x0
	frontMessageStatusSystemOffline = 0x1
	frontMessageStatusSystemOnline  = 0x2

	frontMessageStatusHdmiUnknown   = 0x0
	frontMessageStatusHdmiNoSignal  = 0x1
	frontMessageStatusHdmiConnected = 0x2

	frontMessageStatusUsbUnknown      = 0x0
	frontMessageStatusUsbDisconnected = 0x1
	frontMessageStatusUsbConnected    = 0x2

	frontMessageStatusWifiUnknown    = 0x0
	frontMessageStatusWifiDisable    = 0x1
	frontMessageStatusWifiConnecting = 0x2
	frontMessageStatusWifiConnected  = 0x3
)

const frontMessageStatusSize = 4 + 4 + 4 + 4

type FrontMessageStatus struct {
	system uint32
	hdmi   uint32
	usb    uint32
	wifi   uint32
}

func (ms *FrontMessageStatus) ToBytes() []byte {
	b := make([]byte, frontMessageStatusSize)

	binary.LittleEndian.PutUint32(b[0:4], ms.system)
	binary.LittleEndian.PutUint32(b[4:8], ms.hdmi)
	binary.LittleEndian.PutUint32(b[8:12], ms.usb)
	binary.LittleEndian.PutUint32(b[12:16], ms.wifi)

	return b
}

type FrontMessageApproval struct {
	id    uint32
	app   string
	title string
	desc  string
}

func (ma *FrontMessageApproval) ToBytes() []byte {
	appB := []byte(ma.app)
	titleB := []byte(ma.title)
	descB := []byte(ma.desc)

	// id (4) + len(app) (4) + app + len(title) (4) + title + len(desc) (4) + desc
	totalSize := 4 + 4 + len(appB) + 4 + len(titleB) + 4 + len(descB)

	b := make([]byte, totalSize)
	offset := 0

	binary.LittleEndian.PutUint32(b[offset:offset+4], ma.id)
	offset += 4

	// app
	binary.LittleEndian.PutUint32(b[offset:offset+4], uint32(len(appB)))
	offset += 4
	copy(b[offset:offset+len(appB)], appB)
	offset += len(appB)

	// title
	binary.LittleEndian.PutUint32(b[offset:offset+4], uint32(len(titleB)))
	offset += 4
	copy(b[offset:offset+len(titleB)], titleB)
	offset += len(titleB)

	// desc
	binary.LittleEndian.PutUint32(b[offset:offset+4], uint32(len(descB)))
	offset += 4
	copy(b[offset:offset+len(descB)], descB)
	offset += len(descB)

	return b
}

const frontCmd string = "/root/front"

type Front struct {
	messagePath string
	messageId   uint32

	running uint32

	listener   *net.Listener
	connection *net.Conn

	cmd *exec.Cmd

	onTranscriptStart     FrontVoidCallback
	onTranscriptStop      FrontVoidCallback
	onTranscriptCancel    FrontVoidCallback
	onApprovalAccept      FrontVoidCallback
	onApprovalDeny        FrontVoidCallback
	onApprovalCancel      FrontVoidCallback
	onWorkflowRecortStart FrontVoidCallback
	onWorkflowRecortSave  FrontVoidCallback
	onWorkflowRecortPause FrontVoidCallback
}

type FrontVoidCallback func()

func NewFront(messagePath string) Front {
	return Front{
		messagePath: messagePath,
		messageId:   0,
	}
}

func (f *Front) isRunning() bool {
	return atomic.LoadUint32(&f.running) == 1
}

func (f *Front) setRunning(running bool) {
	if running {
		atomic.StoreUint32(&f.running, 1)
	} else {
		atomic.StoreUint32(&f.running, 0)
	}
}

func (f *Front) useMessageId() uint32 {
	id := atomic.LoadUint32(&f.messageId)

	atomic.AddUint32(&f.messageId, 1)

	return id
}

func (f *Front) openListener() error {
	// delete exists
	os.Remove(f.messagePath)

	// start listen
	l, err := net.Listen("unix", f.messagePath)
	if err != nil {
		log.Printf("front listener open error %v\n", err)
		return err
	}
	f.listener = &l

	return nil
}

func (f *Front) closeListener() error {
	if f.listener != nil {
		err := (*f.listener).Close()
		if err != nil {
			log.Printf("front listener close error %v\n", err)
		}
		f.listener = nil
		os.Remove(f.messagePath)

		return err
	}

	return nil
}

func (f *Front) openConnection() error {
	// avoid null listener
	if f.listener == nil {
		return fmt.Errorf("front null listener")
	}

	c, err := (*f.listener).Accept()
	if err != nil {
		log.Printf("front connection open error %s\n", err)
		return err
	}
	f.connection = &c

	return nil
}

func (f *Front) closeConnection() error {
	if f.connection != nil {
		err := (*f.connection).Close()
		if err != nil {
			log.Printf("front connection close error %s\n", err)
		}
		f.connection = nil

		return err
	}

	return nil
}

func (f *Front) startCmd() error {
	// avoid null listener
	if f.listener == nil {
		return fmt.Errorf("front null listener")
	}

	f.cmd = exec.Command(frontCmd,
		"-mv", Version,
		"-mp", f.messagePath,
	)

	// chmod
	// err = os.Chmod(m.path, 0666)
	// if err != nil {
	// 	return err
	// }

	err := f.cmd.Start()
	if err != nil {
		log.Printf("front cmd start error %v\n", err)
		return err
	}

	return nil
}

func (f *Front) stopCmd() error {
	if f.cmd != nil {
		err := f.cmd.Process.Signal(os.Interrupt)
		if err != nil {
			log.Printf("front cmd stop error %s\n", err)
		}
		f.cmd = nil

		return err
	}

	return nil
}

func (f *Front) close() {
	f.stopCmd()
	f.closeConnection()
	f.closeListener()
}

func (f *Front) handle() {
	defer f.close()

	headerBuffer := make([]byte, frontMessageHeaderSize)

	for f.isRunning() {
		err := f.read(headerBuffer)
		if err != nil {
			log.Printf("front read header error %s\n", err)
			return
		}

		// parse header
		header := ParseFrontMessageHeader(headerBuffer)

		// todo, process header
		switch header.msgType {
		case FrontMessageHeaderTypeTranscriptStart:
			if f.onTranscriptStart != nil {
				f.onTranscriptStart()
			}
		case FrontMessageHeaderTypeTranscriptStop:
			if f.onTranscriptStop != nil {
				f.onTranscriptStop()
			}
		case FrontMessageHeaderTypeTranscriptCancel:
			if f.onTranscriptCancel != nil {
				f.onTranscriptCancel()
			}
		case FrontMessageHeaderTypeApprovalAccept:
			if f.onApprovalAccept != nil {
				f.onApprovalAccept()
			}
		case FrontMessageHeaderTypeApprovalDeny:
			if f.onApprovalDeny != nil {
				f.onApprovalDeny()
			}
		case FrontMessageHeaderTypeApprovalCancel:
			if f.onApprovalCancel != nil {
				f.onApprovalCancel()
			}
		case FrontMessageHeaderTypeWorkflowRecordStart:
			if f.onWorkflowRecortStart != nil {
				f.onWorkflowRecortStart()
			}
		case FrontMessageHeaderTypeWorkflowRecordSave:
			if f.onWorkflowRecortSave != nil {
				f.onWorkflowRecortSave()
			}
		case FrontMessageHeaderTypeWorkflowRecordPause:
			if f.onWorkflowRecortPause != nil {
				f.onWorkflowRecortPause()
			}
		default:
			log.Printf("front unknown message type %d\n", header.msgType)
		}

		if header.bodySize == 0 {
			continue
		}

		bodyBuffer := make([]byte, header.bodySize)

		err = f.read(bodyBuffer)
		if err != nil {
			log.Printf("front read data error %v\n", err)
			return
		}

		// todo, process body
	}
}

func (f *Front) read(buffer []byte) error {
	// avoid null connection
	if f.connection == nil {
		return fmt.Errorf("front null connection")
	}

	total := 0
	for total < len(buffer) && f.isRunning() {
		n, err := (*f.connection).Read(buffer[total:])
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		total += n
	}

	if !f.isRunning() {
		return fmt.Errorf("front is closing")
	}

	if total != len(buffer) {
		return fmt.Errorf("front incomplete read: expected %d, got %d", len(buffer), total)
	}

	return nil
}

func (f *Front) write(buffer []byte) error {
	// avoid null connection
	if f.connection == nil {
		return fmt.Errorf("front null connection")
	} else if !f.isRunning() {
		return fmt.Errorf("front is closing")
	}

	_, err := (*f.connection).Write(buffer)

	return err
}

func (f *Front) send(msgType uint32, body []byte) {
	bs := 0

	if body != nil {
		bs = len(body)
	}

	// send header
	mh := FrontMessageHeader{
		id:        f.useMessageId(),
		timestamp: uint64(time.Now().UnixMicro()),
		msgType:   msgType,
		bodySize:  uint32(bs),
	}
	err := f.write(mh.ToBytes())
	if err != nil {
		log.Printf("front send header error %v\n", err)
		return
	}

	// null body
	if bs == 0 {
		return
	}

	// send body
	err = f.write(body)
	if err != nil {
		log.Printf("front send body error %v\n", err)
		return
	}
}

func (f *Front) Open() error {
	err := f.openListener()
	if err != nil {
		return err
	}

	err = f.startCmd()

	if err != nil {
		f.close()
		return err
	}

	err = f.openConnection()
	if err != nil {
		f.close()
		return err
	}

	f.setRunning(true)
	go f.handle()

	return nil
}

func (f *Front) Close() {
	f.setRunning(false)
}

func (f *Front) SendStatus(
	systemOnline bool,
	hdmiConnected bool,
	usbConnected bool,
	wifiConnecting bool,
	wifiConnected bool,
) error {
	ms := FrontMessageStatus{
		system: frontMessageStatusSystemUnknown,
		hdmi:   frontMessageStatusHdmiUnknown,
		usb:    frontMessageStatusUsbUnknown,
		wifi:   frontMessageStatusWifiUnknown,
	}

	if systemOnline {
		ms.system = frontMessageStatusSystemOnline
	} else {
		ms.system = frontMessageStatusSystemOffline
	}

	if hdmiConnected {
		ms.hdmi = frontMessageStatusHdmiConnected
	} else {
		ms.hdmi = frontMessageStatusHdmiNoSignal
	}

	if usbConnected {
		ms.usb = frontMessageStatusUsbConnected
	} else {
		ms.usb = frontMessageStatusUsbDisconnected
	}

	if wifiConnected {
		ms.wifi = frontMessageStatusWifiConnected
	} else if wifiConnecting {
		ms.wifi = frontMessageStatusWifiConnecting
	} else {
		ms.wifi = frontMessageStatusWifiDisable
	}

	return f.write(ms.ToBytes())
}

func (f *Front) SendTranscriptData(data string) {
	f.send(FrontMessageHeaderTypeTranscriptData, []byte(data))
}

func (f *Front) SendLog(data string) {
	f.send(FrontMessageHeaderTypeLog, []byte(data))
}

func (f *Front) SendApprovalEnd() {
	f.send(FrontMessageHeaderTypeApprovalEnd, nil)
}
