package src

import (
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/pion/webrtc/v4"
)

const videoWidth uint = 1920
const videoHeight uint = 1080
const bitRate uint = 10 * 1024
const gop uint = 60

const DeviceMediaSourceVideo uint = 1
const DeviceMediaSourceGst uint = 2

type Device struct {
	// input args
	Args Args

	// internal
	wsKey   string
	running uint32

	// signal resources
	mqtt Mqtt
	ws   *WebSocket

	// webrtc
	wrtc *WebRTC

	// device resources
	mv    *MediaVideo
	mg    *MediaGst
	hid   *HidController
	front Front
}

// device is running
func (d *Device) isRunning() bool {
	return atomic.LoadUint32(&d.running) == 1
}

// device set running
func (d *Device) setRunning(running bool) {
	if running {
		atomic.StoreUint32(&d.running, 1)
	} else {
		atomic.StoreUint32(&d.running, 0)
	}
}

// open
func (d *Device) Open() {
	// create mqtt
	d.mqtt = Mqtt{
		id:  d.Args.Id,
		url: d.Args.MqttUrl,
		onRequest: func(msg []byte) {
			m := d.handleMqttMessage(msg)
			d.mqtt.Send(m)
		},
	}
	d.mqtt.Open()

	d.wsKey = "ca612056d72344c07211e1eed4634ac593b4704ce65c4febc1bd336bd656404d"

	d.front = NewFront("/var/run/front.sock")
	// err := d.front.Open()
	// if err != nil {
	// 	log.Printf("device front open error %v\n", err)
	// }

	// start
	d.setRunning(true)
	// go d.loop()
}

// close
func (d *Device) Close() {
	d.setRunning(false)

	d.mqtt.Close()

	d.wsStop()
	d.mediaStop()
	d.hidStop()

	d.wrtcStop()
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

// handle mqtt message
func (d *Device) handleMqttMessage(msg []byte) DeviceMessage {
	m, err := UnmarshalDeviceMessage(msg)

	if err != nil {
		return m
	}

	log.Printf("device mqtt message %s\n", m.Type)

	switch m.Type {
	case WebSocketStart:
		{
			d.wsStart()
			return NewDeviceMessage(WebSocketStart)
		}
	case WebSocketStop:
		{
			d.wsStop()
			return NewDeviceMessage(WebSocketStop)
		}
	case Error:
	case "":
		{
			return NewDeviceMessage("")
		}
	default:
		{
			log.Println("unknown request", m.Type)
			return NewDeviceMessage("")
		}
	}

	return NewDeviceMessage("")
}

// handle ws message
func (d *Device) handleWsMessage(msg []byte) DeviceMessage {
	m, err := UnmarshalDeviceMessage(msg)

	if err != nil {
		return m
	}

	log.Printf("device ws message %s\n", m.Type)

	switch m.Type {
	case WebRTCStart:
		{
			d.wrtcStart(m)
			return NewDeviceMessage(WebRTCStart)
		}
	case WebRTCStop:
		{
			d.wrtcStop()
			return NewDeviceMessage(WebRTCStop)
		}
	case WebRTCIceCandidate:
		{
			if d.wrtc == nil {
				return NewDeviceMessage(Error)
			}

			d.wrtc.UseIceCandidate(m.IceCandidate)
			return NewDeviceMessage(WebRTCIceCandidate)
		}
	case WebRTCOffer:
		{
			if d.wrtc == nil {
				return NewDeviceMessage(Error)
			}

			mm := NewDeviceMessage(WebRTCAnswer)
			mm.Answer = d.wrtc.UseOffer(m.Offer)
			return mm
		}
	case Error:
	case "":
		{
			return NewDeviceMessage("")
		}
	default:
		{
			log.Println("unknown request", m.Type)
			return NewDeviceMessage("")
		}
	}

	return NewDeviceMessage("")
}

// webrtc start
func (d *Device) wrtcStart(msg DeviceMessage) {
	if d.wrtc != nil {
		return
	}

	// create webrtc
	wrtc := WebRTC{
		onIceCandidate: d.sendIceCandidate,
		onDataChannel:  d.useDataChannel,
		onClose: func() {
			d.wsStop()
			d.mediaStop()
			d.hidStop()
			d.wrtc = nil
		},
	}
	d.wrtc = &wrtc

	// use ice servers
	iss := make([]webrtc.ICEServer, len(msg.IceServers))
	for i, v := range msg.IceServers {
		iss[i] = v.ToWebrtcIceServer()
	}

	// open wrtc
	wrtc.Open(iss)

	d.mediaStart()
}

func (d *Device) wrtcStop() {
	if d.wrtc == nil {
		return
	}

	d.wrtc.Close()
	d.wrtc = nil
}

func (d *Device) useDataChannel(dc *webrtc.DataChannel) bool {
	switch dc.Label() {
	case "hid":
		{
			d.hidStart()

			dc.OnOpen(func() {
				log.Printf("data channel hid open %d\n", *dc.ID())
			})

			dc.OnMessage(func(dcmsg webrtc.DataChannelMessage) {
				d.hidSend(dcmsg.Data)
			})

			return true
		}
	default:
		{
			log.Printf("data channel unknown %d %s", *dc.ID(), dc.Label())
			dc.Close()
			return false
		}
	}
}

func (d *Device) sendIceCandidate(candidate *webrtc.ICECandidateInit) {
	m := NewDeviceMessage(WebRTCIceCandidate)
	m.IceCandidate = candidate

	err := d.wsSend(m)

	if err != nil {
		log.Printf("device send ice cadidate error %v\n", err)
	}
}

func (d *Device) mediaStart() error {
	if d.wrtc == nil {
		return fmt.Errorf("device wrtc is nil")
	}

	switch d.Args.MediaSource {
	case DeviceMediaSourceVideo:
		{
			if d.mv != nil {
				return nil
			}

			mv := NewMediaVideo(videoWidth, videoHeight, d.Args.VideoPath, d.Args.VideoBinPath, d.Args.VideoSocketPath, bitRate, gop)
			d.mv = &mv

			// use video
			d.wrtc.UseVideoTrackSample(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264})

			d.mv.onData = func(header *MediaFrameHeader, frame []byte) {
				d.wrtc.WriteVideoTrackSample(frame, header.timestamp)
			}

			d.mv.Open()
			break
		}
	case DeviceMediaSourceGst:
		{
			if d.mg != nil {
				return nil
			}

			mg := NewMediaGst(videoWidth, videoHeight, d.Args.VideoPath, "localhost", 10000, bitRate, gop)
			d.mg = &mg

			// use video
			d.wrtc.UseVideoTrackRtp(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264})

			d.mg.onData = func(frame []byte) {
				d.wrtc.WriteVideoTrackRtp(frame)
			}

			d.mg.Open()
			break
		}
	default:
		return fmt.Errorf("unknown media source %d", d.Args.MediaSource)
	}

	return nil
}

func (d *Device) mediaStop() {
	if d.mv != nil {
		d.mv.Close()
		d.mv = nil
	} else if d.mg != nil {
		d.mg.Close()
		d.mg = nil
	}
}

func (d *Device) hidStart() error {
	if d.hid != nil {
		return nil
	}

	hid := NewHidController(d.Args.HidPath, d.Args.HidUdcPath)
	d.hid = &hid

	return hid.Open()
}

func (d *Device) hidSend(b []byte) {
	err := d.hid.Send(b)
	if err != nil {
		log.Printf("device hid send error %v\n", err)
	}
}

func (d *Device) hidStop() {
	if d.hid == nil {
		return
	}

	d.hid.Close()
	d.hid = nil
}

func (d *Device) wsStart() error {
	if d.ws != nil {
		return nil
	}

	ws := WebSocket{
		id:  d.Args.Id,
		url: d.Args.WsUrl,
		key: d.wsKey,
		onMessage: func(msg []byte) {
			if d.ws == nil {
				return
			}

			m := d.handleWsMessage(msg)
			d.ws.Send(m)
		},
		onClose: func() {
			d.ws = nil
		},
	}

	err := ws.Open()
	if err != nil {
		return err
	}

	d.ws = &ws

	return nil
}

func (d *Device) wsSend(m any) error {
	if d.ws == nil {
		return fmt.Errorf("device ws is nil")
	}

	return d.ws.Send(m)
}

func (d *Device) wsStop() {
	if d.ws == nil {
		return
	}

	d.ws.Close()
	d.ws = nil
}

// send status to front
func (d *Device) sendStatus() {
	d.front.SendStatus(
		d.mqtt.client.IsConnected(),
		true,
		d.hid.ReadStatus(),
		WifiStatus{
			Enable:    true,
			Connected: true,
		},
	)
}

func (d *Device) loop() {
	for d.isRunning() {
		d.sendStatus()
		time.Sleep(3 * time.Second)
	}
}
