package src

import (
	"log"
	"sync/atomic"
	"time"

	"github.com/pion/webrtc/v4"
)

const videoWidth uint = 1920
const videoHeight uint = 1080
const bitRate uint = 10 * 1024
const gop uint = 60

type DeviceMediaSource uint

const DeviceMediaSourceVideo DeviceMediaSource = 1
const DeviceMediaSourceGst DeviceMediaSource = 2

type Device struct {
	Id          string
	WsUrl       string
	WsKey       string
	MqttUrl     string
	MediaSource DeviceMediaSource

	running uint32

	// signal resources
	mqtt *Mqtt
	ws   *WebSocket

	// webrtc
	wrtc WebRTC

	// device resources
	mv  *MediaVideo
	mg  *MediaGst
	hid HidController

	// other resources
	front Front
}

func (d *Device) isRunning() bool {
	return atomic.LoadUint32(&d.running) == 1
}

func (d *Device) setRunning(running bool) {
	if running {
		atomic.StoreUint32(&d.running, 1)
	} else {
		atomic.StoreUint32(&d.running, 0)
	}
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

			if d.mv != nil {
				d.mv.Close()
			}
			if d.mg != nil {
				d.mg.Close()
			}
			d.hid.Close()
		},
	}

	// create resources
	switch d.MediaSource {
	case DeviceMediaSourceVideo:
		{
			mv := NewMediaVideo(videoWidth, videoHeight, "/dev/video0", "/var/run/capture.sock", bitRate, gop)
			d.mv = &mv
			break
		}
	case DeviceMediaSourceGst:
		{
			mg := NewMediaGst(videoWidth, videoHeight, "/dev/video0", "localhost", 10000, bitRate, gop)
			d.mg = &mg
			break
		}
	default:
		log.Fatalf("unknown media source %d", d.MediaSource)
	}
	d.hid = NewHidController("/dev/hidg1", "/sys/class/udc")
	d.front = NewFront("/var/run/front.sock")

	// start
	d.setRunning(true)
	go d.loop()
}

func (d *Device) Close() {
	d.setRunning(false)

	if d.mqtt != nil {
		d.mqtt.Close()
		d.mqtt = nil
	}
	if d.ws != nil {
		d.ws.Close()
		d.ws = nil
	}

	if d.mv != nil {
		d.mv.Close()
	}
	if d.mg != nil {
		d.mg.Close()
	}
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

	if d.mv != nil {
		// use video
		d.wrtc.UseVideoTrackSample(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264})

		d.mv.onData = func(header *MediaFrameHeader, frame []byte) {
			d.wrtc.WriteVideoTrackSample(frame, header.timestamp)
		}

		d.mv.Open()
	} else if d.mg != nil {
		// use video
		d.wrtc.UseVideoTrackRtp(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264})

		d.mg.onData = func(frame []byte) {
			d.wrtc.WriteVideoTrackRtp(frame)
		}

		d.mg.Open()
	}
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

func (d *Device) sendStatus() {
	d.front.SendStatus(
		d.mqtt.client.IsConnected(),
		true,
		d.hid.ReadStatus(),
		false,
		false,
	)
}

func (d *Device) loop() {
	for d.isRunning() {
		d.sendStatus()
		time.Sleep(3 * time.Second)
	}
}
