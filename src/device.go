package src

import (
	"encoding/json"
	"log"
	"net/url"
	"strconv"
	"time"

	"github.com/pion/webrtc/v4"
)

type Device struct {
	Id         string
	MqttBroker string

	mqtt Mqtt
	wrtc WebRTC
}

func (d *Device) Init() {
	uu, err := url.Parse(d.MqttBroker)
	if err != nil {
		log.Fatal(err)
	}

	p, err := strconv.ParseInt(uu.Port(), 0, 64)
	if err != nil {
		log.Fatal(err)
	}
	up, _ := uu.User.Password()

	d.mqtt = Mqtt{
		broker:    uu.Hostname(),
		port:      p,
		deviceId:  d.Id,
		username:  uu.User.Username(),
		password:  up,
		OnRequest: d.onRequest,
	}
	d.mqtt.Init()

	d.wrtc = WebRTC{
		onIceCandidate: d.onIceCandidate,
	}
	d.wrtc.Init()
}

func (d *Device) Close() {
	d.wrtc.Close()
	d.mqtt.Close()
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
}

type DeviceMessageWebRTCStart struct {
	DeviceMessage
	IceServers []DeviceMessageIceServer `json:"iceServers,omitempty"`
}

type DeviceMessageWebRTCIceCandidate struct {
	DeviceMessage
	IceCandidate webrtc.ICECandidateInit `json:"iceCandidate"`
}

type DeviceMessageWebRTCOffer struct {
	DeviceMessage
	Offer webrtc.SessionDescription `json:"offer"`
}

type DeviceMessageWebRTCAnswer struct {
	DeviceMessage
	Answer webrtc.SessionDescription `json:"answer"`
}

func (d *Device) onRequest(msg []byte) {
	var m DeviceMessage
	err := json.Unmarshal(msg, &m)
	if err != nil {
		log.Fatal("json parse message error ", err)
	}

	switch m.Type {
	case WebRTCStart:
		{
			var m DeviceMessageWebRTCStart
			err := json.Unmarshal(msg, &m)
			if err != nil {
				log.Fatal("json parse message error ", err)
			}

			d.wrtc.Open(Map(m.IceServers, func(ics DeviceMessageIceServer) webrtc.ICEServer {
				return ics.ToWebrtcIceServer()
			}), "video1", 1920, 1080)

			d.mqtt.PublishResponse(DeviceMessageWebRTCStart{
				DeviceMessage: DeviceMessage{
					Time: time.Now().Unix(),
					Type: WebRTCStart,
				},
			})
		}
	case WebRTCIceCandidate:
		{
			var m DeviceMessageWebRTCIceCandidate
			err := json.Unmarshal(msg, &m)
			if err != nil {
				log.Fatal("json parse message error ", err)
			}

			d.wrtc.UseIceCandidate(m.IceCandidate)
		}
	case WebRTCOffer:
		{
			var m DeviceMessageWebRTCOffer
			err := json.Unmarshal(msg, &m)
			if err != nil {
				log.Fatal("json parse message error ", err)
			}

			answer := d.wrtc.UseOffer(m.Offer)
			d.mqtt.PublishResponse(DeviceMessageWebRTCAnswer{
				DeviceMessage: DeviceMessage{
					Time: time.Now().Unix(),
					Type: WebRTCAnswer,
				},
				Answer: answer,
			})
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

func (d *Device) onIceCandidate(candidate webrtc.ICECandidateInit) {
	d.mqtt.PublishResponse(DeviceMessageWebRTCIceCandidate{
		DeviceMessage: DeviceMessage{
			Time: time.Now().Unix(),
			Type: WebRTCIceCandidate,
		},
		IceCandidate: candidate,
	})
}
