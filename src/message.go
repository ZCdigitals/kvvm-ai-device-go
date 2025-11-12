package src

import (
	"encoding/json"
	"time"

	"github.com/pion/webrtc/v4"
)

type DeviceMessageType string

const (
	WebSocketStart     DeviceMessageType = "websocket-start"
	WebSocketStop      DeviceMessageType = "websocket-stop"
	WebRTCStart        DeviceMessageType = "webrtc-start"
	WebRTCStop         DeviceMessageType = "webrtc-stop"
	WebRTCIceCandidate DeviceMessageType = "webrtc-ice-candidate"
	WebRTCOffer        DeviceMessageType = "webrtc-offer"
	WebRTCAnswer       DeviceMessageType = "webrtc-answer"
	Error              DeviceMessageType = "error"
)

type DeviceMessage struct {
	Time int64             `json:"time,omitempty"`
	Type DeviceMessageType `json:"type,omitempty"`

	// webrtc start
	IceServers []DeviceMessageIceServer `json:"iceServers,omitempty"`

	// webrtc ice candidate
	IceCandidate  *webrtc.ICECandidateInit  `json:"iceCandidate,omitempty"`
	IceCandidates []webrtc.ICECandidateInit `json:"iceCandidates,omitempty"`

	// webrtc offer
	Offer *webrtc.SessionDescription `json:"offer,omitempty"`

	// webrtc answer
	Answer *webrtc.SessionDescription `json:"answer,omitempty"`
}

func NewDeviceMessage(t DeviceMessageType) DeviceMessage {
	return DeviceMessage{
		Time: time.Now().Unix(),
		Type: t,
	}
}

func UnmarshalDeviceMessage(data []byte) (DeviceMessage, error) {
	m := NewDeviceMessage("")

	err := json.Unmarshal(data, &m)

	if err != nil {
		m.Type = Error
	}

	return m, err
}
