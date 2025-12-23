package src

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"
)

type SpeechResult struct {
	Time int    `json:"time"`
	Text string `json:"text"`
}

func UnmarshalSpeechResult(data []byte) (SpeechResult, error) {
	sr := SpeechResult{}

	err := json.Unmarshal(data, &sr)

	return sr, err
}

type SpeechRecordHeader struct {
	id   uint32
	rate uint32
	// audio format
	//
	// 2 S16_LE
	format    uint32
	timestamp uint64
	size      uint32
	reserved  uint32
}

func ParseSpeechRecordHeader(b []byte) SpeechRecordHeader {
	return SpeechRecordHeader{
		id:        binary.LittleEndian.Uint32(b[0:4]),
		rate:      binary.LittleEndian.Uint32(b[4:8]),
		format:    binary.LittleEndian.Uint32(b[8:12]),
		timestamp: binary.LittleEndian.Uint64(b[12:20]),
		size:      binary.LittleEndian.Uint32(b[20:24]),
		reserved:  binary.LittleEndian.Uint32(b[24:28]),
	}
}

const recordHeaderLength = 28

type SpeechOnText func(text string)

type Speech struct {
	id  string
	url string
	key string

	hardware   string
	binPath    string
	socketPath string

	sampleRate uint
	channel    uint

	running uint32
	results []SpeechResult

	connection *websocket.Conn

	mux sync.RWMutex

	socketListener   *net.Listener
	socketConnection *net.Conn

	cmd *exec.Cmd

	onText SpeechOnText
}

func (s *Speech) isRunning() bool {
	return atomic.LoadUint32(&s.running) == 1
}

func (s *Speech) setRunning(running bool) {
	if running {
		atomic.StoreUint32(&s.running, 1)
	} else {
		atomic.StoreUint32(&s.running, 0)
	}
}

func (s *Speech) openConnection() error {
	if s.connection != nil {
		return fmt.Errorf("speech connection exists")
	}

	header := http.Header{}
	header.Add("x-device-key", s.key)

	u, err := url.Parse(s.url)
	if err != nil {
		log.Fatalln("speech url parse error", err)
	}
	u.Path = fmt.Sprintf("/ws/device/%s/stt", s.id)
	u.Query().Add("rate", strconv.FormatUint(uint64(s.sampleRate), 10))
	u.Query().Add("bits", "16")
	u.Query().Add("channel", strconv.FormatUint(uint64(s.channel), 10))

	connection, _, err := websocket.DefaultDialer.Dial(u.String(), header)
	if err != nil {
		log.Println("speech open error", err)
		return err
	}

	connection.SetCloseHandler(func(code int, text string) error {
		s.Close()
		return nil
	})

	s.connection = connection
	log.Println("speech open")
	return nil
}

func (s *Speech) closeConnection() error {
	if s.connection == nil {
		return fmt.Errorf("speech null connection")
	}

	err := s.connection.Close()
	s.connection = nil
	log.Println("speech close")

	return err
}

func (s *Speech) handle() {
	defer s.closeConnection()

	for s.isRunning() {
		_, msg, err := s.connection.ReadMessage()
		if err != nil {
			log.Println("speech read message error", err)
			continue
		}

		sr, err := UnmarshalSpeechResult(msg)
		if err != nil {
			log.Println("speech result unmarshal error", err)
			continue
		}

		s.results = append(s.results, sr)

		if s.onText != nil {
			t, err := s.useResultsText()
			if err != nil {
				log.Println("speech text error", err)
				continue
			}
			s.onText(t)
		}
	}
}

func (s *Speech) useResultsText() (string, error) {
	if s.results == nil {
		return "", fmt.Errorf("speech null results")
	}

	if len(s.results) == 0 {
		return "", nil
	}

	res := make([]SpeechResult, len(s.results))
	copy(res, s.results)

	sort.Slice(res, func(i, j int) bool {
		return res[i].Time < res[j].Time
	})

	texts := make([]string, 0, len(res))
	for _, r := range res {
		t := strings.TrimSpace(r.Text)
		if t != "" {
			texts = append(texts, t)
		}
	}

	return strings.Join(texts, " "), nil
}

func (s *Speech) send(audio []byte) error {
	if s.connection == nil {
		return fmt.Errorf("speech null connection")
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	err := s.connection.WriteMessage(websocket.BinaryMessage, audio)

	return err
}

func (s *Speech) openSocketListener() error {
	if s.socketListener != nil {
		return fmt.Errorf("speech socket listener exitst")
	}

	// delete exits
	os.Remove(s.socketPath)

	// start listen
	l, err := net.Listen("unix", s.socketPath)
	if err != nil {
		log.Println("speech socket listener open error", err)
		return err
	}
	s.socketListener = &l

	return nil
}

func (s *Speech) closeSocketListener() error {
	if s.socketListener == nil {
		return fmt.Errorf("speech null socket listener")
	}

	err := (*s.socketListener).Close()
	if err != nil {
		log.Println("speech socket listener close error", err)
	}
	s.socketListener = nil
	os.Remove(s.socketPath)

	return err
}

func (s *Speech) openSocketConnection() error { // avoid null listener
	if s.socketListener == nil {
		return fmt.Errorf("speech null socket listener")
	} else if s.socketConnection != nil {
		return fmt.Errorf("speech socket connection exists")
	}

	c, err := (*s.socketListener).Accept()
	if err != nil {
		log.Println("speech socket connection open error", err)
		return err
	}
	s.socketConnection = &c

	return nil
}

func (s *Speech) closeSocketConnection() error {
	if s.socketConnection == nil {
		return fmt.Errorf("speech null socket connection")
	}

	err := (*s.socketConnection).Close()
	if err != nil {
		log.Println("speech socket connection close error", err)
	}
	s.socketConnection = nil

	return err
}

func (s *Speech) startCmd() error {
	// avoid null connection
	if s.socketConnection == nil {
		return fmt.Errorf("speech null socket connection")
	} else if s.cmd != nil {
		return fmt.Errorf("speech cmd exists")
	}

	s.cmd = exec.Command(s.binPath,
		"-d", s.hardware,
		"-s", s.socketPath,
		// S16_LE
		"-f", "2",
		"-r", strconv.FormatUint(uint64(s.sampleRate), 10),
		"-c", strconv.FormatUint(uint64(s.channel), 10),
	)

	err := s.cmd.Start()
	if err != nil {
		log.Println("speech cmd start error", err)
		return err
	}

	return nil
}

func (s *Speech) stopCmd() error {
	if s.cmd != nil {
		err := s.cmd.Process.Signal(os.Interrupt)
		if err != nil {
			log.Println("speech cmd stop error", err)
		}
		s.cmd = nil

		return err
	}

	return nil
}

func (s *Speech) handleSocket() {
	defer s.closeSocket()

	headerBuffer := make([]byte, recordHeaderLength)

	for s.isRunning() {
		err := s.readSocket(headerBuffer)
		if err != nil {
			log.Println("speech socket read header error", err)
			return
		}

		// parse header
		header := ParseSpeechRecordHeader(headerBuffer)

		// check size
		if header.size == 0 {
			log.Println("speech socket", header.id, "size is 0")
			continue
		}

		recordBuffer := make([]byte, header.size)
		err = s.readSocket(recordBuffer)
		if err != nil {
			log.Println("speech socket read data error", err)
			return
		}

		// send
		s.send(recordBuffer)
	}
}

func (s *Speech) readSocket(buffer []byte) error {
	// avoid null connection
	if s.socketConnection == nil {
		return fmt.Errorf("speech null socket connection")
	}

	total := 0
	for total < len(buffer) && s.isRunning() {
		n, err := (*s.socketConnection).Read(buffer[total:])
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		total += n
	}

	if !s.isRunning() {
		return fmt.Errorf("media socket is closing")
	}

	if total != len(buffer) {
		return fmt.Errorf("media socket incomplete read: expected %d, got %d", len(buffer), total)
	}

	return nil
}

func (s *Speech) openSocket() error {
	err := s.openSocketListener()
	if err != nil {
		return err
	}

	err = s.startCmd()
	if err != nil {
		s.closeSocket()
		return err
	}

	err = s.openSocketConnection()
	if err != nil {
		s.closeSocket()
		return err
	}

	go s.handleSocket()

	return nil
}

func (s *Speech) closeSocket() {
	s.stopCmd()
	s.closeSocketConnection()
	s.closeSocketListener()
}

func (s *Speech) Open() error {
	s.setRunning(true)

	err := s.openConnection()
	if err != nil {
		return err
	}

	err = s.openSocket()
	if err != nil {
		s.closeConnection()
		return err
	}

	s.results = []SpeechResult{}

	go s.handle()

	return nil
}

func (s *Speech) Close() {
	s.setRunning(false)
	s.results = nil
}
