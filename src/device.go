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
	running uint32

	cf ConfigFile

	// signal resources
	serveUrl   string
	api        ServeApi
	mqttUrl    string
	mqtt       *Mqtt
	responseWs *WebSocket

	// webrtc
	wrtc *WebRTC

	// device resources
	mediaSource     uint
	videoPath       string
	videoBinPath    string
	videoSocketPath string
	mv              *MediaVideo
	mg              *MediaGst
	hid             HidController
	vm              VideoMonitor
	front           Front
}

func NewDevice(args Args) Device {
	return Device{
		cf: ConfigFile{
			path: args.ConfigPath,
		},

		// signal resources
		serveUrl: args.ServeUrl,
		api: ServeApi{
			baseUrl: args.ServeUrl,
		},
		mqttUrl: args.MqttUrl,

		// device resources
		mediaSource: args.MediaSource,
		hid: HidController{
			path: args.HidPath,
		},
		vm: VideoMonitor{
			path:       args.VideoMonitorPath,
			binPath:    args.VideoMonitorBinPath,
			socketPath: args.VideoMonitorSocketPath,
		},
		front: Front{
			binPath:    args.FrontBinPath,
			socketPath: args.FrontSocketPath,
		},
	}
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
	// load config
	d.cf.Load()

	// set api auth
	d.api.accessToken = d.cf.config.AccessToken
	d.api.accessTokenExpiresAt = d.cf.config.AccessTokenExpiresAt
	d.api.refreshToken = d.cf.config.RefreshToken
	d.api.refreshTokenExpiresAt = d.cf.config.RefreshTokenExpiresAt

	// if id exists, use mqtt
	if d.cf.config.ID != "" {
		mqtt := Mqtt{
			id:  d.cf.config.ID,
			url: d.mqttUrl,
			onRequest: func(msg []byte) {
				m := d.handleMqttMessage(msg)
				d.mqtt.Send(m)
			},
		}
		mqtt.Open()
		d.mqtt = &mqtt
	}

	// err := d.front.Open()
	// if err != nil {
	// 	log.Printf("device front open error %v\n", err)
	// }

	d.vm.Open()

	// start
	d.setRunning(true)
	go d.loop()
}

// close
func (d *Device) Close() {
	d.setRunning(false)

	d.mqtt.Close()

	d.wsStop()
	d.mediaStop()
	d.vm.Close()
	d.hid.Close()

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

	log.Println("device mqtt message", m.Type)

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

	log.Println("device ws message", m.Type)

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
			d.hid.Close()
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
			d.hid.Open()

			dc.OnOpen(func() {
				log.Println("data channel hid open", *dc.ID())
			})

			dc.OnMessage(func(dcmsg webrtc.DataChannelMessage) {
				d.hid.Send(dcmsg.Data)
			})

			return true
		}
	default:
		{
			log.Println("data channel unknown", *dc.ID(), dc.Label())
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
		log.Println("device send ice cadidate error", err)
	}
}

func (d *Device) mediaStart() error {
	if d.wrtc == nil {
		return fmt.Errorf("device wrtc is nil")
	}

	switch d.mediaSource {
	case DeviceMediaSourceVideo:
		{
			if d.mv != nil {
				return nil
			}

			mv := MediaVideo{
				width:      videoWidth,
				height:     videoHeight,
				path:       d.videoPath,
				binPath:    d.videoBinPath,
				socketPath: d.videoSocketPath,
				bitRate:    bitRate,
				gop:        gop,
			}
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

			mg := MediaGst{
				width:      videoWidth,
				height:     videoHeight,
				inputPath:  d.videoPath,
				outputIp:   "localhost",
				outputPort: 10000,
				bitRate:    bitRate,
				gop:        gop,
			}
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
		return fmt.Errorf("unknown media source %d", d.mediaSource)
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

func (d *Device) wsStart() error {
	if d.responseWs != nil {
		return nil
	}

	ws, err := d.api.UseDeviceResponse(d.cf.config.ID)
	if err != nil {
		return err
	}

	err = ws.Open()
	if err != nil {
		return err
	}

	d.responseWs = ws

	return nil
}

func (d *Device) wsSend(m any) error {
	if d.responseWs == nil {
		return fmt.Errorf("device response ws is nil")
	}

	return d.responseWs.Send(m)
}

func (d *Device) wsStop() {
	if d.responseWs == nil {
		return
	}

	d.responseWs.Close()
	d.responseWs = nil
}

// send status to front
func (d *Device) sendStatus() {
	// d.front.SendStatus(
	// 	d.mqtt.client.IsConnected(),
	// 	true,
	// 	hidStatus,
	// 	WifiStatus{
	// 		Enable:    true,
	// 		Connected: true,
	// 	},
	// )

	log.Println("status", d.mqtt.client.IsConnected(), d.vm.isConnected, d.hid.ReadStatus())
}

func (d *Device) loop() {
	for d.isRunning() {
		d.sendStatus()
		time.Sleep(3 * time.Second)
	}
}

func (d *Device) sendWOL() error {
	if d.cf.config.WakeOnLanMac == "" {
		return fmt.Errorf("device wake on lan mac is empty")
	}

	return SendWOL(d.cf.config.WakeOnLanMac)
}

func (d *Device) SendWol() {
	err := d.sendWOL()
	if err != nil {
		log.Println("device send wol error", err)
	}
}
