package src

import (
	"encoding/json"
	"log"
	"time"

	"github.com/pion/webrtc/v4"
)

const (
	DeviceMessageTypeWebSocketStart     string = "websocket-start"
	DeviceMessageTypeWebSocketStop      string = "websocket-stop"
	DeviceMessageTypeWebRTCStart        string = "webrtc-start"
	DeviceMessageTypeWebRTCStop         string = "webrtc-stop"
	DeviceMessageTypeWebRTCIceCandidate string = "webrtc-ice-candidate"
	DeviceMessageTypeWebRTCOffer        string = "webrtc-offer"
	DeviceMessageTypeWebRTCAnswer       string = "webrtc-answer"
	DeviceMessageTypeWakeOnLan          string = "wake-on-lan"
	DeviceMessageTypeError              string = "error"
)

type DeviceMessage struct {
	Time int64  `json:"time"`
	Type string `json:"type"`

	// webrtc start
	IceServers []DeviceMessageIceServer `json:"iceServers,omitempty"`

	// webrtc ice candidate
	IceCandidate  *webrtc.ICECandidateInit  `json:"iceCandidate,omitempty"`
	IceCandidates []webrtc.ICECandidateInit `json:"iceCandidates,omitempty"`

	// webrtc offer
	Offer *webrtc.SessionDescription `json:"offer,omitempty"`

	// webrtc answer
	Answer *webrtc.SessionDescription `json:"answer,omitempty"`

	// wake on lan mac
	WakeOnLanMacAddress string `json:"wakeOnLanMacAddress"`
}

func NewDeviceMessage(t string) DeviceMessage {
	return DeviceMessage{
		Time: time.Now().Unix(),
		Type: t,
	}
}

func UnmarshalDeviceMessage(data []byte) (DeviceMessage, error) {
	m := DeviceMessage{}

	err := json.Unmarshal(data, &m)

	if err != nil {
		log.Println("message json unmarshal error", err)
		m.Type = DeviceMessageTypeError
	}

	return m, err
}

type DeviceMessageIceServer struct {
	Credential string   `json:"credential"`
	Urls       []string `json:"urls"`
	Username   string   `json:"username"`
}

func (iceServer *DeviceMessageIceServer) ToWebrtcIceServer() webrtc.ICEServer {
	return webrtc.ICEServer{
		URLs:       iceServer.Urls,
		Username:   iceServer.Username,
		Credential: iceServer.Credential,
	}
}
