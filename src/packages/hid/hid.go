package hid

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"sync"
)

const (
	HidKeyboardReportId = 0x01
	HidMouseReportId    = 0x02
)

type HidController struct {
	path    string
	udcPath string

	fd   *os.File
	fdMu sync.RWMutex
}

func NewHidController(path string, udcPath string) HidController {
	return HidController{path: path, udcPath: udcPath}
}

func (h *HidController) openFile() error {
	h.fdMu.Lock()
	defer h.fdMu.Unlock()

	if h.fd != nil {
		return fmt.Errorf("hid fd exists")
	}

	fd, err := os.OpenFile(h.path, os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	h.fd = fd

	return nil
}

func (h *HidController) closeFile() error {
	h.fdMu.Lock()
	defer h.fdMu.Unlock()

	if h.fd == nil {
		return fmt.Errorf("hid null fd")
	}

	err := h.fd.Close()
	h.fd = nil
	return err
}

func (h *HidController) write(id byte, data []byte) error {
	h.fdMu.RLock()
	defer h.fdMu.RUnlock()

	if h.fd == nil {
		return fmt.Errorf("hid null fd")
	}

	r := append([]byte{id}, data...)

	_, err := h.fd.Write(r)

	return err
}

func (h *HidController) writeMouse(
	btn1, btn2, btn3 bool,
	x, y uint16,
) error {
	data := make([]byte, 5)

	// set buttons
	var bs byte
	if btn1 {
		bs |= 1 << 0
	}
	if btn2 {
		bs |= 1 << 1
	}
	if btn3 {
		bs |= 1 << 2
	}
	data[0] = bs

	// set pos
	binary.LittleEndian.PutUint16(data[1:3], x)
	binary.LittleEndian.PutUint16(data[3:5], y)

	return h.write(HidMouseReportId, data)
}

func (h *HidController) writeKeyboard(
	ctrl, shift, alt bool,
	key1, key2, key3, key4, key5, key6 string,
) error {
	data := make([]byte, 7)

	// set modifiers
	var ms byte
	if ctrl {
		ms |= 1 << 0
	}
	if shift {
		ms |= 1 << 1
	}
	if alt {
		ms |= 1 << 2
	}
	data[0] = ms

	data[1] = findKeyCode(key1)
	data[2] = findKeyCode(key2)
	data[3] = findKeyCode(key3)
	data[4] = findKeyCode(key4)
	data[5] = findKeyCode(key5)
	data[6] = findKeyCode(key6)

	return h.write(HidKeyboardReportId, data)
}

func (h *HidController) Open() error {
	return h.openFile()
}

func (h *HidController) Close() {
	h.closeFile()
}

func (h *HidController) ReadStatus() bool {
	// is running
	if h.fd != nil {
		return true
	}

	// check file exists
	_, err := os.Stat(h.path)
	if err != nil {
		return false
	}

	// avoid null udc path
	if h.udcPath == "" {
		return false
	}

	udc, err := os.ReadFile(h.udcPath)
	if err != nil {
		log.Println("hid controller read status udc error", err)
		return false
	}

	statePath := fmt.Sprintf("/sys/class/udc/%s/state", string(udc))
	state, err := os.ReadFile(statePath)
	if err != nil {
		log.Println("hid controller read status state error", err)
		return false
	}

	return string(state) != "not attached"
}

func (h *HidController) Send(b []byte) error {
	hd, err := UnmarshalHidData(b)
	if err != nil {
		return err
	}

	switch hd.Category {
	case HidDataCategoryMouse:
		{
			d := hd.Data.(HidMouseData)
			return h.writeMouse(
				d.Button1,
				d.Button2,
				d.Button3,
				uint16(d.X),
				uint16(d.Y),
			)
		}
	case HidDataCategoryKeyboard:
		{
			d := hd.Data.(HidKeyboardData)
			return h.writeKeyboard(
				d.Ctrl,
				d.Shift,
				d.Alt,
				d.Key1,
				d.Key2,
				d.Key3,
				d.Key4,
				d.Key5,
				d.Key6,
			)
		}
	default:
		return fmt.Errorf("hid controller send error, unknown data category %s", hd.Category)
	}
}
