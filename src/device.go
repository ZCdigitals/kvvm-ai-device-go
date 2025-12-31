package src

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	WEBRTC "github.com/pion/webrtc/v4"

	"device-go/src/apis"
	"device-go/src/libs/webrtc"
	"device-go/src/libs/websocket"
	"device-go/src/packages/front"
	"device-go/src/packages/gstreamer"
	"device-go/src/packages/hid"
	"device-go/src/packages/mqtt"
	"device-go/src/packages/video"
	"device-go/src/packages/wake_on_lan"
)

const videoWidth uint = 1920
const videoHeight uint = 1080
const bitRate uint = 10 * 1024
const gop uint = 60

const DeviceMediaSourceVideo uint = 1
const DeviceMediaSourceGst uint = 2

type Device struct {
	cancel context.CancelFunc
	wg     sync.WaitGroup

	cf ConfigFile

	// signal resources
	api        apis.ServeApi
	mqttUrl    string
	mqtt       *mqtt.Mqtt
	responseWs *websocket.WebSocket

	// webrtc
	wrtc *webrtc.WebRTC

	// device resources
	mediaSource     uint
	videoPath       string
	videoBinPath    string
	videoSocketPath string
	mv              *video.Video
	mg              *gstreamer.Gstreamer
	vm              video.VideoMonitor
	hid             hid.HidController
	front           front.Front
}

func NewDevice(args Args) Device {
	return Device{
		cf: ConfigFile{
			path: args.ConfigPath,
		},

		// signal resources
		api:     apis.NewServeApi(args.ServeUrl, args.ServeClientId),
		mqttUrl: args.MqttUrl,

		// device resources
		mediaSource:     args.MediaSource,
		videoPath:       args.VideoPath,
		videoBinPath:    args.VideoBinPath,
		videoSocketPath: args.VideoSocketPath,
		hid: hid.NewHidController(
			args.HidPath,
			args.HidUdcPath,
		),
		vm: video.NewVideoMonitor(
			args.VideoMonitorPath,
			args.VideoMonitorBinPath,
			args.VideoMonitorSocketPath,
		),
		front: front.NewFront(
			args.FrontBinPath,
			args.FrontSocketPath,
			Version,
		),
	}
}

// webrtc start
func (d *Device) wrtcStart(msg DeviceMessage) error {
	if d.wrtc != nil {
		return fmt.Errorf("device webrtc exists")
	}

	// create webrtc
	wrtc := webrtc.WebRTC{
		OnIceCandidate: d.sendIceCandidate,
		OnDataChannel:  d.useDataChannel,
		OnClose: func() {
			log.Println("device webrtc close")
			d.wsStop()
			d.mediaStop()
			d.hid.Close()
			d.wrtc = nil
		},
	}
	d.wrtc = &wrtc

	// use ice servers
	iss := make([]WEBRTC.ICEServer, len(msg.IceServers))
	for i, v := range msg.IceServers {
		iss[i] = v.ToWebrtcIceServer()
	}

	// open wrtc
	err := wrtc.Open(iss)
	if err != nil {
		return err
	}

	// media start
	return d.mediaStart()
}

// webrt stop
func (d *Device) wrtcStop() error {
	if d.wrtc == nil {
		return fmt.Errorf("device null webrtc")
	}

	d.wrtc.Close()
	d.wrtc = nil

	return nil
}

func (d *Device) useDataChannel(dc *WEBRTC.DataChannel) bool {
	switch dc.Label() {
	case "hid":
		{
			d.hid.Open()

			dc.OnOpen(func() {
				log.Println("data channel hid open", *dc.ID())
			})

			dc.OnMessage(func(dcmsg WEBRTC.DataChannelMessage) {
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

func (d *Device) sendIceCandidate(candidate *WEBRTC.ICECandidateInit) {
	m := NewDeviceMessage(WebRTCIceCandidate)
	m.IceCandidate = candidate

	err := d.wsSend(m)

	if err != nil {
		log.Println("device send ice cadidate error", err)
	}
}

// media start
func (d *Device) mediaStart() error {
	if d.wrtc == nil {
		return fmt.Errorf("device null webrtc")
	}

	switch d.mediaSource {
	case DeviceMediaSourceVideo:
		{
			if d.mv != nil {
				return fmt.Errorf("device mv exists")
			}

			mv := video.NewVideo(
				d.videoPath,
				d.videoBinPath,
				d.videoSocketPath,
				videoWidth,
				videoHeight,
				bitRate,
				gop,
			)
			d.mv = &mv

			// use video
			d.wrtc.AddVideoTrackSample(WEBRTC.RTPCodecCapability{MimeType: WEBRTC.MimeTypeH264})

			// set callback
			d.mv.OnData = func(id uint32, timestamp uint64, frame []byte) {
				d.wrtc.WriteVideoTrackSample(frame, timestamp)
			}

			d.mv.Open()
			break
		}
	case DeviceMediaSourceGst:
		{
			if d.mg != nil {
				return fmt.Errorf("device mg exists")
			}

			mg := gstreamer.NewGstreamer(
				d.videoPath,
				"localhost",
				10000,
				videoWidth,
				videoHeight,
				bitRate,
				gop,
			)
			d.mg = &mg

			// use video
			d.wrtc.AddVideoTrackRtp(WEBRTC.RTPCodecCapability{MimeType: WEBRTC.MimeTypeH264})

			d.mg.OnData = func(frame []byte) {
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

// media stop
func (d *Device) mediaStop() {
	if d.mv != nil {
		d.mv.Close()
		d.mv = nil
	} else if d.mg != nil {
		d.mg.Close()
		d.mg = nil
	}
}

// websocket start
func (d *Device) wsStart() error {
	if d.responseWs != nil {
		return fmt.Errorf("device websocket exists")
	}

	ws, err := d.api.UseDeviceResponse(d.cf.Config.ID)
	if err != nil {
		return err
	}

	ws.OnMessage = func(messageType int, message []byte) {
		m := d.handleWsMessage(message)
		ws.Send(m)
	}

	err = ws.Open()
	if err != nil {
		return err
	}
	log.Println("device ws open")

	// send first message
	ws.Send(NewDeviceMessage(""))

	d.responseWs = ws

	return nil
}

// websocket stop
func (d *Device) wsStop() error {
	if d.responseWs == nil {
		return fmt.Errorf("device null websocket")
	}

	d.responseWs.Close()
	d.responseWs = nil

	return nil
}

// websocket send data
func (d *Device) wsSend(m any) error {
	if d.responseWs == nil {
		return fmt.Errorf("device null websocket")
	}

	return d.responseWs.Send(m)
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

	mqttIsConnected := false
	if d.mqtt != nil {
		mqttIsConnected = d.mqtt.IsConnected()
	}

	log.Println("status", mqttIsConnected, d.vm.IsConnected, d.hid.ReadStatus())
}

func (d *Device) sendWOL() error {
	if d.cf.Config.WakeOnLanMac == "" {
		return fmt.Errorf("device wake on lan mac is empty")
	}

	return wake_on_lan.SendWOL(d.cf.Config.WakeOnLanMac)
}

// handle mqtt message
func (d *Device) handleMqttMessage(msg []byte) DeviceMessage {
	m, err := UnmarshalDeviceMessage(msg)
	log.Println("device mqtt request", m.Type)

	if err != nil {
		return NewDeviceMessage(Error)
	}

	switch m.Type {
	case WebSocketStart:
		{
			err = d.wsStart()
			if err != nil {
				log.Println("device ws start error", err)
				return NewDeviceMessage(Error)
			}
			return NewDeviceMessage(WebSocketStart)
		}
	case WebSocketStop:
		{
			err = d.wsStop()
			if err != nil {
				log.Println("device ws stop error", err)
				return NewDeviceMessage(Error)
			}
			return NewDeviceMessage(WebSocketStop)
		}
	case Error, "":
		{
			return NewDeviceMessage("")
		}
	default:
		{
			log.Println("unknown request type", m.Type)
			return NewDeviceMessage(Error)
		}
	}
}

// handle ws message
func (d *Device) handleWsMessage(msg []byte) DeviceMessage {
	m, err := UnmarshalDeviceMessage(msg)
	log.Println("device ws message", m.Type)

	if err != nil {
		return NewDeviceMessage(Error)
	}

	switch m.Type {
	case WebRTCStart:
		{
			err = d.wrtcStart(m)
			if err != nil {
				log.Println("device wrtc start error", err)
				return NewDeviceMessage(Error)
			}
			return NewDeviceMessage(WebRTCStart)
		}
	case WebRTCStop:
		{
			err = d.wrtcStop()
			if err != nil {
				log.Println("device wrtc stop error", err)
				return NewDeviceMessage(Error)
			}
			return NewDeviceMessage(WebRTCStop)
		}
	case WebRTCIceCandidate:
		{
			if d.wrtc == nil {
				return NewDeviceMessage(Error)
			}

			err = d.wrtc.AddIceCandidate(m.IceCandidate)
			if err != nil {
				log.Println("device wrtc add ice candidtae error", err)
				return NewDeviceMessage(Error)
			}
			return NewDeviceMessage(WebRTCIceCandidate)
		}
	case WebRTCOffer:
		{
			if d.wrtc == nil {
				return NewDeviceMessage(Error)
			}

			mm := NewDeviceMessage(WebRTCAnswer)
			answer, err := d.wrtc.UseOffer(m.Offer)
			if err != nil {
				log.Println("device wrtc use offer error", err)
				return NewDeviceMessage(Error)
			}

			mm.Answer = answer
			return mm
		}
	case Error, "":
		{
			return NewDeviceMessage("")
		}
	default:
		{
			log.Println("unknown request", m.Type)
			return NewDeviceMessage(Error)
		}
	}
}

func (d *Device) loop(ctx context.Context) {
	d.wg.Add(1)
	defer func() {
		d.wg.Done()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			{
				d.sendStatus()
			}
		}
		time.Sleep(3 * time.Second)
	}
}

// open
func (d *Device) Open() {
	// load config
	err := d.cf.Load()
	if os.IsNotExist(err) {
		err = d.cf.Save()
		if err != nil {
			log.Println("device config save error", err)
		}
	} else if err != nil {
		log.Println("device config load error", err)
	}

	// set api auth
	d.api.SetOAuthToken(
		d.cf.Config.AccessToken,
		d.cf.Config.AccessTokenExpiresAt,
		d.cf.Config.RefreshToken,
		d.cf.Config.RefreshTokenExpiresAt,
	)
	d.api.OnUpdateToken = func(accessToken string, accessTokenExpiresAt time.Time, refreshToken string, refreshTokenExpiresAt time.Time) {
		d.cf.Config.AccessToken = accessToken
		d.cf.Config.AccessTokenExpiresAt = accessTokenExpiresAt
		d.cf.Config.RefreshToken = refreshToken
		d.cf.Config.RefreshTokenExpiresAt = refreshTokenExpiresAt
		err := d.cf.Save()
		if err != nil {
			log.Println("device config save error", err)
		}
	}

	// if id exists, use mqtt
	if d.cf.Config.ID != "" {
		mqtt := mqtt.NewMqtt(d.cf.Config.ID, d.mqttUrl)
		mqtt.OnRequest = func(msg []byte) {
			m := d.handleMqttMessage(msg)
			d.mqtt.Send(m)
		}
		err = mqtt.Open()
		if err != nil {
			log.Println("device mqtt open error", err)
		} else {
			d.mqtt = &mqtt
		}
	}

	// err := d.front.Open()
	// if err != nil {
	// 	log.Printf("device front open error %v\n", err)
	// }

	err = d.vm.Open()
	if err != nil {
		log.Println("device video monitor open error", err)
	}

	// start
	// ctx, cancel := context.WithCancel(context.Background())
	// d.cancel = cancel

	// go d.loop(ctx)
}

// close
func (d *Device) Close() {
	if d.cancel != nil {
		d.cancel()
		d.cancel = nil
	}

	d.wg.Wait()

	if d.mqtt != nil {
		d.mqtt.Close()
	}

	d.wsStop()
	d.mediaStop()
	d.vm.Close()
	d.hid.Close()

	d.wrtcStop()
}

func (d *Device) SendWol() {
	err := d.sendWOL()
	if err != nil {
		log.Println("device send wol error", err)
	}
}
