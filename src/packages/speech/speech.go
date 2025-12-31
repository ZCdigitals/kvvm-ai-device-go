package speech

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"device-go/src/libs/exec"
	"device-go/src/libs/socket"
	"device-go/src/libs/websocket"
)

type SpeechResult struct {
	Time int    `json:"time"`
	Text string `json:"text"`
}

func UnmarshalSpeechResult(b []byte) (SpeechResult, error) {
	sr := SpeechResult{}

	err := json.Unmarshal(b, &sr)

	return sr, err
}

type SpeechOnText func(text string)

type Speech struct {
	ex     exec.Exec
	socket socket.Socket
	WS     *websocket.WebSocket

	OnText SpeechOnText
}

func NewSpeech(
	hardware string,
	binPath string,
	socketPath string,
	sampleRate uint,
	channel uint,
) Speech {
	return Speech{
		ex: exec.NewExec(
			binPath,
			"-d", hardware,
			"-s", socketPath,
			// S16_LE
			"-f", "2",
			"-r", strconv.FormatUint(uint64(sampleRate), 10),
			"-c", strconv.FormatUint(uint64(channel), 10),
		),
		socket: socket.NewSocket(socketPath),
	}
}

func (s *Speech) openWs() error {
	if s.WS == nil {
		return fmt.Errorf("speech null ws")
	}

	s.WS.OnClose = func() {
		s.Close()
	}
	s.WS.OnMessage = func(messageType int, message []byte) {
		sr, err := UnmarshalSpeechResult(message)
		if err != nil {
			log.Println("speech unmarshal result error", err)
			return
		} else if s.OnText != nil {
			s.OnText(sr.Text)
		}
	}

	s.WS.Open()

	return nil
}

func (s *Speech) Open() error {
	err := s.openWs()
	if err != nil {
		return err
	}

	s.socket.OnData = func(header socket.SocketHeader, body []byte) {
		if s.WS == nil {
			return
		}
		s.WS.SendBinary(body)
	}

	s.socket.Open()
	s.ex.Start()

	return nil
}

func (s *Speech) Close() {
	if s.WS != nil {
		s.WS.OnClose = nil
		s.WS.Close()
		s.WS = nil
	}

	s.ex.Stop()
	s.socket.Close()
}
