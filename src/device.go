package src

import (
	"log"

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
				m := d.handleMessage(msg)
				d.mqtt.Send(m)
			},
		}
		d.mqtt.Open()
	} else if d.WsUrl != "" {
		// create webscoket
		d.ws = &WebSocket{
			id:  d.Id,
			url: d.WsUrl,
			key: d.WsKey,
			onMessage: func(msg []byte) {
				m := d.handleMessage(msg)
				d.ws.Send(m)
			},
		}
		d.ws.Open()
	} else {
		log.Fatalln("Must set mqtt or ws")
	}

	// create webrtc
	d.wrtc = WebRTC{
		onIceCandidate: d.sendIceCandidate,
		onDataChannel:  d.useDataChannel,
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
	d.hid = NewHidController("/dev/hidg1")
}

func (d *Device) Close() {
	if d.mqtt != nil {
		d.mqtt.Close()
		d.mqtt = nil
	}
	if d.ws != nil {
		d.ws.Close()
		d.ws = nil
	}
	d.ms.Close()
	d.hid.Close()
	d.wrtc.Close()
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

func (d *Device) handleMessage(msg []byte) DeviceMessage {
	m, err := UnmarshalDeviceMessage(msg)

	if err != nil {
		return m
	}

	switch m.Type {
	case WebSocketStart:
		d.webSocketStart()
		return NewDeviceMessage(WebSocketStart)
	case WebSocketStop:
		d.webSocketStop()
		return NewDeviceMessage(WebSocketStop)
	case WebRTCStart:
		d.webRTCStart(m)
		return NewDeviceMessage(WebRTCStart)
	case WebRTCStop:
		d.webRTCStop()
		return NewDeviceMessage(WebRTCStop)
	case WebRTCIceCandidate:
		d.wrtc.UseIceCandidate(m.IceCandidate)
		return NewDeviceMessage(WebRTCIceCandidate)
	case WebRTCOffer:
		{
			mm := NewDeviceMessage(WebRTCAnswer)
			mm.Answer = d.wrtc.UseOffer(m.Offer)
			return mm
		}
	case Error:
	case "":
		return NewDeviceMessage("")
	default:
		log.Println("unknown request", m.Type)
		return NewDeviceMessage("")
	}

	return NewDeviceMessage("")
}

func (d *Device) webRTCStart(msg DeviceMessage) {
	// use ice servers
	iss := make([]webrtc.ICEServer, len(msg.IceServers))
	for i, v := range msg.IceServers {
		iss[i] = v.ToWebrtcIceServer()
	}

	// open wrtc
	d.wrtc.Open(iss)

	// use video
	d.wrtc.UseVideoTrackSample(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264})

	d.ms.onData = func(header *MediaFrameHeader, frame []byte) {
		d.wrtc.WriteVideoTrackSample(frame, header.timestamp)
	}

	d.ms.Open()
}

func (d *Device) webRTCStop() {
	d.wrtc.Close()
}

func (d *Device) webSocketStart() error {
	ws := &WebSocket{
		id:  d.Id,
		url: d.WsUrl,
		key: d.WsKey,
		onMessage: func(msg []byte) {
			m := d.handleMessage(msg)
			d.ws.Send(m)
		},
	}

	err := ws.Open()
	if err != nil {
		return err
	}

	d.ws = ws

	return nil
}

func (d *Device) webSocketStop() {
	if d.ws == nil {
		return
	}
	d.ws.Close()
	d.ws = nil
}

func (d *Device) useDataChannel(dc *webrtc.DataChannel) bool {
	switch dc.Label() {
	case "hid":
		{
			err := d.hid.Open()
			if err != nil {
				return false
			}

			dc.OnOpen(func() {
				log.Printf("data channel hid open %d\n", *dc.ID())
			})

			dc.OnMessage(func(dcmsg webrtc.DataChannelMessage) {
				d.hid.Send(dcmsg.Data)
			})

			return true
		}
	default:
		log.Printf("data channel unknown %d %s", *dc.ID(), dc.Label())
		dc.Close()
		return false
	}
}

func (d *Device) sendIceCandidate(candidate *webrtc.ICECandidateInit) {
	m := NewDeviceMessage(WebRTCIceCandidate)
	m.IceCandidate = candidate
	if d.ws != nil {
		d.ws.Send(m)
	} else if d.mqtt != nil {
		d.mqtt.Send(m)
	} else {
		log.Printf("can not send ice candidate")
	}
}
