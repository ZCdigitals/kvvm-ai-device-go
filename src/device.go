package src

import (
	"encoding/json"
	"log"
	"time"

	"github.com/pion/webrtc/v4"
)

type Device struct {
	Id      string
	WsUrl   string
	WsKey   string
	MqttUrl string

	// signal resources
	mqtt *Mqtt
	ws   *WebSocket

	// webrtc
	wrtc WebRTC

	// device resources
	ms  MediaSocket
	hid HidController
}

func (d *Device) Init() {
	if d.MqttUrl != "" {
		// create mqtt
		d.mqtt = &Mqtt{
			id:  d.Id,
			url: d.MqttUrl,
			onRequest: func(msg []byte) {
				m := d.onMessage(msg)
				d.mqtt.publish("request", m)
			},
		}
	} else if d.WsUrl != "" {
		// create webscoket
		d.ws = &WebSocket{
			id:  d.Id,
			url: d.WsUrl,
			key: d.WsKey,
			onMessage: func(msg []byte) {
				m := d.onMessage(msg)
				d.ws.Send(m)
			},
		}
	}

	if d.mqtt != nil {
		d.mqtt.Init()
	} else if d.ws != nil {
		d.ws.Init()
	} else {
		log.Fatalln("Must set mqtt or ws")
	}

	// create webrtc
	d.wrtc = WebRTC{
		onIceCandidate: d.sendIceCandidate,
		onClose: func() {
			if d.mqtt != nil {
				d.ws.Close()
				d.ws = nil
			}
			d.ms.Close()
			d.hid.Close()
		},
	}

	// create resources
	d.ms = NewMediaSocket("/var/run/capture.sock")
	// d.ms = *NewMediaSocket("/tmp/capture.sock")
	d.hid = HidController{Path: "/dev/hidg0"}

}

func (d *Device) Close() {
	if d.mqtt != nil {
		d.mqtt.Close()
	}
	if d.ws != nil {
		d.ws.Close()
	}
	d.ms.Close()
	d.hid.Close()
	d.wrtc.Close()
}

type DeviceMessageType string

const (
	WebSocketStart     DeviceMessageType = "websocket-start"
	WebRTCStart        DeviceMessageType = "webrtc-start"
	WebRTCStop         DeviceMessageType = "webrtc-stop"
	WebRTCIceCandidate DeviceMessageType = "webrtc-ice-candidate"
	WebRTCOffer        DeviceMessageType = "webrtc-offer"
	WebRTCAnswer       DeviceMessageType = "webrtc-answer"
	Error              DeviceMessageType = "error"
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

	// websocket url
	WebSocketUrl string `json:"websocketUrl,omitempty"`

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

func (d *Device) onMessage(msg []byte) DeviceMessage {
	var m DeviceMessage
	err := json.Unmarshal(msg, &m)
	if err != nil {
		log.Printf("json parse message error %s", err)
		return DeviceMessage{
			Time: time.Now().Unix(),
		}
	}

	switch m.Type {
	case WebSocketStart:
		d.onWebSocketStart(m)
		return DeviceMessage{
			Time: time.Now().Unix(),
			Type: WebSocketStart,
		}
	case WebRTCStart:
		d.onWebRTCStart(m)
		return DeviceMessage{
			Time: time.Now().Unix(),
			Type: WebRTCStart,
		}
	case WebRTCIceCandidate:
		d.wrtc.UseIceCandidate(m.IceCandidate)
		return DeviceMessage{
			Time: time.Now().Unix(),
			Type: WebRTCIceCandidate,
		}
	case WebRTCOffer:
		{
			answer := d.wrtc.UseOffer(m.Offer)
			return DeviceMessage{
				Time:   time.Now().Unix(),
				Type:   WebRTCAnswer,
				Answer: answer,
			}
		}
	case Error:
	case "":
		{
			return DeviceMessage{
				Time: time.Now().Unix(),
			}
		}
	default:
		{
			log.Println("unknown request", m.Type)
			return DeviceMessage{
				Time: time.Now().Unix(),
			}
		}
	}

	return DeviceMessage{
		Time: time.Now().Unix(),
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
	d.wrtc.UseTrack(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264})
	// log.Println("webrtc use track")

	// d.ms.onData = func(header *MediaFrameHeader, frame []byte) {
	// 	d.wrtc.WriteVideoTrack(frame, header.timestamp)
	// }

	// err := d.ms.Init()
	// if err != nil {
	// 	log.Printf("media init error %s", err)
	// 	return
	// }

	// // use hid
	// err = d.hid.Open()
	// if err != nil {
	// 	log.Printf("hid open error %s", err)
	// 	return
	// }
	dc := d.wrtc.CreateDataChannel("hid")
	if dc != nil {
		log.Println("hid created")
	}
	// if dc == nil {
	// 	dc.OnMessage(func(dcmsg webrtc.DataChannelMessage) {
	// 		d.hid.Send(dcmsg.Data)
	// 	})
	// }
}

func (d *Device) onWebSocketStart(msg DeviceMessage) {
	if d.ws == nil {
		return
	}
	d.ws = &WebSocket{
		url: msg.WebSocketUrl,
		onMessage: func(msg []byte) {
			m := d.onMessage(msg)
			d.ws.Send(m)
		},
	}
}

func (d *Device) sendIceCandidate(candidate *webrtc.ICECandidateInit) {
	if d.mqtt != nil {
		d.mqtt.PublishResponse(
			DeviceMessage{
				Time:         time.Now().Unix(),
				Type:         WebRTCIceCandidate,
				IceCandidate: candidate,
			},
		)
	} else if d.ws != nil {
		d.ws.Send(DeviceMessage{
			Time:         time.Now().Unix(),
			Type:         WebRTCIceCandidate,
			IceCandidate: candidate,
		})
	} else {
		log.Printf("can not send ice candidate")
	}
}
