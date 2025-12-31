package hid

import (
	"encoding/json"
	"fmt"
)

const (
	HidDataCategoryKeyboard string = "keyboard"
	HidDataCategoryMouse    string = "mouse"
)

type HidData struct {
	Category string `json:"category"`
	Data     any    `json:"data"`
}

func UnmarshalHidData(data []byte) (HidData, error) {
	h := HidData{}

	// use raw
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return h, err
	}

	// use category
	if err := json.Unmarshal(raw["category"], &h.Category); err != nil {
		return h, err
	}

	switch h.Category {
	case HidDataCategoryMouse:
		{
			m, err := UnmarshalHidMouseData(raw["data"])
			if err != nil {
				return h, err
			}
			h.Data = m
			break
		}
	case HidDataCategoryKeyboard:
		{
			k, err := UnmarshalHidKeyboardData(raw["data"])
			if err != nil {
				return h, err
			}
			h.Data = k
			break
		}
	default:
		{
			return h, fmt.Errorf("hid data unmarshal error, unknown category %s", h.Category)
		}
	}

	return h, nil
}

const (
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
	Ctrl  bool   `json:"ctrl"`
	Shift bool   `json:"shift"`
	Alt   bool   `json:"alt"`
	Key1  string `json:"key1"`
	Key2  string `json:"key2"`
	Key3  string `json:"key3"`
	Key4  string `json:"key4"`
	Key5  string `json:"key5"`
	Key6  string `json:"key6"`
}

func UnmarshalHidKeyboardData(data []byte) (HidKeyboardData, error) {
	k := HidKeyboardData{}
	err := json.Unmarshal(data, &k)

	if err != nil {
		return k, err
	}

	return k, err
}
