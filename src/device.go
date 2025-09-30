package src

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"time"

	"github.com/pion/webrtc/v4"
)

type Device struct {
	Id      string
	MqttUrl string

	mqtt Mqtt
	wrtc WebRTC
	rtp  Rtp
	hid  HidController
}

func (d *Device) Init() {
	// create mqtt
	d.mqtt = Mqtt{
		Url:       d.MqttUrl,
		OnRequest: d.onRequest,
	}
	d.mqtt.Init()

	// create webrtc
	d.wrtc = WebRTC{
		onIceCandidate: d.onIceCandidate,
		onHidMessage:   d.onHidMessage,
		onClose: func() {
			d.rtp.Close()
			d.hid.Close()
		},
	}

	// create rtp
	d.rtp = Rtp{Ip: "0.0.0.0", Port: 5004}

	// create hid
	d.hid = HidController{Path: "/dev/hidg0"}
}

func (d *Device) Close() error {
	d.mqtt.Close()

	err := d.wrtc.Close()
	if err != nil {
		return err
	}

	err = d.rtp.Close()
	if err != nil {
		return err
	}

	err = d.hid.Close()
	if err != nil {
		return err
	}

	return nil
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

			// open wrtc
			d.wrtc.Open(
				Map(
					m.IceServers,
					func(ics DeviceMessageIceServer) webrtc.ICEServer {
						return ics.ToWebrtcIceServer()
					},
				),
			)
			log.Println("webrtc open")

			// create track after open, because this cloud be optional
			d.wrtc.UseTrack(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264})
			log.Println("webrtc use track")

			go func() {
				// Read RTP packets forever and send them to the WebRTC Client
				inboundRTPPacket := make([]byte, 1600) // UDP MTU

				for {
					n, err := d.rtp.Read(inboundRTPPacket)

					if err != nil {
						log.Fatal("device rtp read error ", err)
					}

					err = d.wrtc.WriteVideoTrack(inboundRTPPacket[:n])

					if err != nil {
						if errors.Is(err, io.ErrClosedPipe) {
							// The peerConnection has been closed.
							return
						}

						log.Fatal("device rtp write error ", err)
					}
				}
			}()

			// start hid
			d.hid.Open()

			// init rtp
			d.rtp.Init()

			// send start to peer
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

func (d *Device) onHidMessage(msg string) {
	d.hid.Send(msg)
}
