package src

import (
	"encoding/json"
	"log"
	"time"

	"github.com/pion/webrtc/v4"
)

type Device struct {
	Id      string
	MqttUrl string

	mqtt Mqtt

	wrtc WebRTC
	ms   MediaSocket
	hid  HidController
}

func (d *Device) Init() {
	// create mqtt
	d.mqtt = Mqtt{
		id:        d.Id,
		url:       d.MqttUrl,
		onRequest: d.onMqttRequest,
	}
	d.mqtt.Init()

	// create webrtc
	d.wrtc = WebRTC{
		onIceCandidate: d.sendIceCandidate,
		onClose: func() {
			d.ms.Close()
			d.hid.Close()
		},
	}

	// create rtp
	d.ms = *NewMediaSocket("/tmp/capture.sock")

	// create hid
	d.hid = HidController{Path: "/dev/hidg0"}
}

func (d *Device) Close() {
	d.mqtt.Close()

	d.wrtc.Close()

	d.ms.Close()

	d.hid.Close()
}

type DeviceMessageType string

const (
	WebRTCStart        DeviceMessageType = "webrtc-start"
	WebRTCStop         DeviceMessageType = "webrtc-stop"
	WebRTCIceCandidate DeviceMessageType = "webrtc-ice-candidate"
	WebRTCOffer        DeviceMessageType = "webrtc-offer"
	WebRTCAnswer       DeviceMessageType = "webrtc-answer"
)

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

type DeviceMessage struct {
	Time int64             `json:"time,omitempty"`
	Type DeviceMessageType `json:"type,omitempty"`

	// webrtc start
	IceServers []DeviceMessageIceServer `json:"iceServers,omitempty"`
	Video      bool                     `json:"video,omitempty"`
	Hid        bool                     `json:"hid,omitempty"`

	// webrtc ice candidate
	IceCandidate  webrtc.ICECandidateInit   `json:"iceCandidate,omitempty"`
	IceCandidates []webrtc.ICECandidateInit `json:"iceCandidates,omitempty"`

	// webrtc offer
	Offer webrtc.SessionDescription `json:"offer,omitempty"`

	// webrtc answer
	Answer webrtc.SessionDescription `json:"answer,omitempty"`
}

func (d *Device) onMqttRequest(msg []byte) {
	var m DeviceMessage
	err := json.Unmarshal(msg, &m)
	if err != nil {
		log.Fatalf("json parse message error %s", err)
	}

	switch m.Type {
	case WebRTCStart:
		d.onWebRTCStart(m)
	case WebRTCIceCandidate:
		d.wrtc.UseIceCandidate(m.IceCandidate)
	case WebRTCOffer:
		{
			answer := d.wrtc.UseOffer(m.Offer)
			d.mqtt.PublishResponse(
				DeviceMessage{
					Time:   time.Now().Unix(),
					Type:   WebRTCAnswer,
					Answer: answer,
				},
			)
		}
	case "":
		{
			d.mqtt.PublishResponse(DeviceMessage{
				Time: time.Now().Unix(),
			})
		}
	default:
		{
			log.Println("unknown request", m.Type)
			return
		}
	}
}

type DeviceHttpData struct {
	Method string            `json:"method"`
	Url    string            `json:"url"`
	Header map[string]string `json:"header"`
	Body   string            `json:"body,omitempty"`
}

func (d *Device) onWebRTCStart(msg DeviceMessage) {
	// use ice servers
	iss := make([]webrtc.ICEServer, len(msg.IceServers))
	for i, v := range msg.IceServers {
		iss[i] = v.ToWebrtcIceServer()
	}

	// open wrtc
	d.wrtc.Open(iss)

	// use video
	if msg.Video {
		// create track after open, because this cloud be optional
		d.wrtc.UseTrack(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264})
		log.Println("webrtc use track")

		d.ms.onData = func(header *MediaFrameHeader, frame []byte) {
			println("track data")
			d.wrtc.WriteVideoTrack(frame, header.timestamp)
		}

		err := d.ms.Init()
		if err != nil {
			log.Printf("media init error %s", err)
		}
	}

	// use hid
	if msg.Hid {
		d.hid.Open()

		d.wrtc.onHidMessage = func(msg []byte) {
			d.hid.Send(msg)
		}
	}

	d.wrtc.onHttpMessage = func(msg []byte) {
		var req HttpRequestData
		json.Unmarshal(msg, &req)

		// todo, http body

		res, err := SendHttpRequest(req)
		if err != nil {
			log.Printf("http send error %s", err)
			return
		}

		mm, err := json.Marshal(res)
		if err != nil {
			log.Printf("http json error %s", err)
			return
		}

		d.wrtc.SendHttpMessage(mm)

		// todo, http body
	}

	// send start to peer
	d.mqtt.PublishResponse(
		DeviceMessage{
			Time: time.Now().Unix(),
			Type: WebRTCStart,
		},
	)
}

func (d *Device) sendIceCandidate(candidate webrtc.ICECandidateInit) {
	d.mqtt.PublishResponse(
		DeviceMessage{
			Time:         time.Now().Unix(),
			Type:         WebRTCIceCandidate,
			IceCandidate: candidate,
		},
	)
}
