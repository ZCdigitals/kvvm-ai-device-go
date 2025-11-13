package src

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"os"
)

const (
	HidKeyboardReportId = 0x01
	HidMouseReportId    = 0x02
	HidMousePositionMin = 0
	HidMousePositionMax = 32768
)

type HidMouseData struct {
	X       int  `json:"x"`
	Y       int  `json:"y"`
	Button1 bool `json:"button1"`
	Button2 bool `json:"button2"`
	Button3 bool `json:"button3"`
}

func UnmarshalHidMouseData(data []byte) (HidMouseData, error) {
	m := HidMouseData{}
	err := json.Unmarshal(data, &m)

	if err != nil {
		log.Printf("hid mouse data unmarshal error %v", err)
		return m, err
	}

	if m.X < HidMousePositionMin || m.X >= HidMousePositionMax {
		return m, fmt.Errorf("hid mouse data unmarshal error, x must be in [%d, %d)", HidMousePositionMin, HidMousePositionMax)
	} else if m.Y < HidMousePositionMin || m.Y >= HidMousePositionMax {
		return m, fmt.Errorf("hid mouse data unmarshal error, y must be in [%d, %d)", HidMousePositionMin, HidMousePositionMax)
	}

	return m, err
}

type HidKeyboardData struct {
	Ctrl  bool    `json:"ctrl"`
	Shift bool    `json:"shift"`
	Alt   bool    `json:"alt"`
	Key1  *string `json:"key1,omitempty"`
	Key2  *string `json:"key2,omitempty"`
	Key3  *string `json:"key3,omitempty"`
	Key4  *string `json:"key4,omitempty"`
	Key5  *string `json:"key5,omitempty"`
	Key6  *string `json:"key6,omitempty"`
}

func UnmarshalHidKeyboardData(data []byte) (HidKeyboardData, error) {
	k := HidKeyboardData{}
	err := json.Unmarshal(data, &k)

	if err != nil {
		log.Printf("hid keyboard data unmarshal error %v", err)
		return k, err
	}

	return k, err
}

type HidDataCategory string

const (
	HidDataCategoryKeyboard HidDataCategory = "keyboard"
	HidDataCategoryMouse    HidDataCategory = "mouse"
)

type HidData struct {
	Category HidDataCategory `json:"category"`
	Data     any             `json:"data"`
}

func UnmarshalHidData(data []byte) (HidData, error) {
	h := HidData{}

	// use raw
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		log.Printf("hid data unmarshal error %v", err)
		return h, err
	}

	// use category
	if err := json.Unmarshal(raw["category"], &h.Category); err != nil {
		return h, fmt.Errorf("hid data unmarshal error, category unmarshal error %v", err)
	}

	switch h.Category {
	case HidDataCategoryMouse:
		{
			m, err := UnmarshalHidMouseData(raw["data"])
			if err != nil {
				return h, err
			}
			h.Data = m
		}
	case HidDataCategoryKeyboard:
		{
			k, err := UnmarshalHidKeyboardData(raw["data"])
			if err != nil {
				return h, err
			}
			h.Data = k
		}
	default:
		return h, fmt.Errorf("hid data unmarshal error, unknown category %s", h.Category)
	}

	return h, nil
}

type HidController struct {
	path string
	fd   *os.File
}

func NewHidController(path string) HidController {
	return HidController{path: path}
}

func (h *HidController) Open() error {
	fd, err := os.OpenFile(h.path, os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("hid controller open error %v", err)
		return err
	}

	h.fd = fd
	log.Println("hid controller open")

	return err
}

func (h *HidController) Close() error {
	if h.fd == nil {
		return nil
	}

	// set all key up
	h.SendMouse(HidMouseData{X: 100, Y: 100})
	h.SendKeyboard(HidKeyboardData{})

	err := h.fd.Close()
	if err != nil {
		log.Printf("hid controller close error %v", err)
	}
	h.fd = nil
	log.Println("hid controller close")

	return err
}

func (h *HidController) WriteReport(reportID byte, data []byte) error {
	if h.fd == nil {
		return fmt.Errorf("hid controller write error null fd")
	}

	// add id
	report := append([]byte{reportID}, data...)

	if _, err := h.fd.Write(report); err != nil {
		log.Printf("hid controller write error %v", err)
		return err
	}

	return nil
}

func (h *HidController) WriteMouseReport(btn1, btn2, btn3 bool, x, y int) error {
	if x < 0 || x >= 32768 {
		return fmt.Errorf("x must be [0-32768)")
	} else if y < 0 || y >= 32768 {
		return fmt.Errorf("y must be [0-32768)")
	}

	// buttons
	buttons := BoolsToInt(btn1, btn2, btn3, false, false, false, false, false)

	data := make([]byte, 5)
	data[0] = byte(buttons)

	binary.LittleEndian.PutUint16(data[1:3], uint16(x))
	binary.LittleEndian.PutUint16(data[3:5], uint16(y))

	return h.WriteReport(HidMouseReportId, data)
}

func (h *HidController) WriteKeyboardReport(
	ctrl, shift, alt bool,
	key1, key2, key3, key4, key5, key6 *string,
) error {
	keys := []*string{key1, key2, key3, key4, key5, key6}
	keyCodes := make([]byte, 6)

	for i, key := range keys {
		keyCodes[i] = FindKeyCode(key)
	}

	// control buttons
	modifiers := BoolsToInt(
		ctrl,  // left ctrl
		shift, // left shift
		alt,   // left alt
		false, // left gui
		false, // right ctrl
		false, // right shift
		false, // right alt
		false, // right gui
	)

	data := make([]byte, 7)
	data[0] = byte(modifiers)
	copy(data[1:], keyCodes)

	return h.WriteReport(HidKeyboardReportId, data)
}

func (h *HidController) SendMouse(data HidMouseData) error {
	return h.WriteMouseReport(
		data.Button1,
		data.Button2,
		data.Button3,
		data.X,
		data.Y,
	)
}

func (h *HidController) SendKeyboard(data HidKeyboardData) error {
	return h.WriteKeyboardReport(
		data.Ctrl,
		data.Shift,
		data.Alt,
		data.Key1,
		data.Key2,
		data.Key3,
		data.Key4,
		data.Key5,
		data.Key6,
	)
}

func (h *HidController) Send(b []byte) error {
	hd, err := UnmarshalHidData(b)
	if err != nil {
		return err
	}

	// log.Printf("hid data %s", hidData.Category)

	switch data := hd.Data.(type) {
	case HidMouseData:
		return h.SendMouse(data)
	case HidKeyboardData:
		return h.SendKeyboard(data)
	default:
		return fmt.Errorf("unknown data type: %T", data)
	}
}

// https://usb.org/sites/default/files/hut1_22.pdf
// https://developer.mozilla.org/zh-CN/docs/Web/API/KeyboardEvent/key
var HIDKeyboardUsageTable = map[string]byte{
	// a-z
	"a": 0x04, "A": 0x04,
	"b": 0x05, "B": 0x05,
	"c": 0x06, "C": 0x06,
	"d": 0x07, "D": 0x07,
	"e": 0x08, "E": 0x08,
	"f": 0x09, "F": 0x09,
	"g": 0x0A, "G": 0x0A,
	"h": 0x0B, "H": 0x0B,
	"i": 0x0C, "I": 0x0C,
	"j": 0x0D, "J": 0x0D,
	"k": 0x0E, "K": 0x0E,
	"l": 0x0F, "L": 0x0F,
	"m": 0x10, "M": 0x10,
	"n": 0x11, "N": 0x11,
	"o": 0x12, "O": 0x12,
	"p": 0x13, "P": 0x13,
	"q": 0x14, "Q": 0x14,
	"r": 0x15, "R": 0x15,
	"s": 0x16, "S": 0x16,
	"t": 0x17, "T": 0x17,
	"u": 0x18, "U": 0x18,
	"v": 0x19, "V": 0x19,
	"w": 0x1A, "W": 0x1A,
	"x": 0x1B, "X": 0x1B,
	"y": 0x1C, "Y": 0x1C,
	"z": 0x1D, "Z": 0x1D,

	// numbers
	"1": 0x1E, "!": 0x1E,
	"2": 0x1F, "@": 0x1F,
	"3": 0x20, "#": 0x20,
	"4": 0x21, "$": 0x21,
	"5": 0x22, "%": 0x22,
	"6": 0x23, "^": 0x23,
	"7": 0x24, "&": 0x24,
	"8": 0x25, "*": 0x25,
	"9": 0x26, "(": 0x26,
	"0": 0x27, ")": 0x27,

	//  return
	"Enter": 0x28,
	// esc
	"Escape": 0x29,
	// backspace
	"Backspace": 0x2A,
	// tab
	"Tab": 0x2B,
	// space
	" ": 0x2C,

	// special
	"-": 0x2D, "_": 0x2D,
	"=": 0x2E, "+": 0x2E,
	"[": 0x2F, "{": 0x2F,
	"]": 0x30, "}": 0x30,
	"\\": 0x31, "|": 0x31,
	";": 0x33, ":": 0x33,
	"'": 0x34, "\"": 0x34,
	",": 0x36, "<": 0x36,
	".": 0x37, ">": 0x37,
	"/": 0x38, "?": 0x38,

	// CapsLock
	"CapsLock": 0x39,

	// f1-f12
	"F1":  0x3A,
	"F2":  0x3B,
	"F3":  0x3C,
	"F4":  0x3D,
	"F5":  0x3E,
	"F6":  0x3F,
	"F7":  0x40,
	"F8":  0x41,
	"F9":  0x42,
	"F10": 0x43,
	"F11": 0x44,
	"F12": 0x45,

	// pause
	// "":0x48
	// insert
	"Insert": 0x49,
	// page
	"Home":     0x4A,
	"PageUp":   0x4B,
	"PageDown": 0x4D,
	"END":      0x4E,
	// delete
	"Delete": 0x4C,
	// arrow
	"ArrowRight": 0x4F,
	"ArrowLeft":  0x50,
	"ArrowDown":  0x51,
	"ArrowUp":    0x52,
}

func FindKeyCode(key *string) byte {
	if key == nil || *key == "" {
		return 0x00
	}

	if code, exists := HIDKeyboardUsageTable[*key]; exists {
		return code
	}

	// 如果键不存在，记录警告并返回0
	log.Printf("Key %s not found in keyboard table", *key)
	return 0x00
}
