package src

import (
	"log"
	"time"

	// This is required to register screen adapter
	// "github.com/pion/mediadevices/pkg/driver/screen"

	// This is required to register camera adapter
	// "github.com/pion/mediadevices/pkg/driver/camera"

	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
)

type WebRTCOnIceCandidate func(candidate webrtc.ICECandidateInit)
type WebRTCOnDataChannelMessage func(msg []byte)

type WebRTC struct {
	pc *webrtc.PeerConnection

	// video track
	vt            *webrtc.TrackLocalStaticSample
	lastFrameTime time.Time

	// close
	onClose func()

	// ice candidate
	onIceCandidate WebRTCOnIceCandidate

	// hid data channel
	hidChannel   *webrtc.DataChannel
	onHidMessage WebRTCOnDataChannelMessage

	// http data channel
	httpChannel   *webrtc.DataChannel
	onHttpMessage WebRTCOnDataChannelMessage
}

func (wrtc *WebRTC) Init() {}

func (wrtc *WebRTC) Open(iceServers []webrtc.ICEServer) {
	// 准备配置
	config := webrtc.Configuration{
		ICEServers: iceServers,
	}

	// 创建PeerConnection
	pc, err := webrtc.NewPeerConnection(config)
	if err != nil {
		log.Fatalf("create peer connection error %s", err)
	}

	wrtc.pc = pc

	pc.OnConnectionStateChange(func(st webrtc.PeerConnectionState) {
		log.Println("webrtc connect state change", st)
		switch st {
		case webrtc.PeerConnectionStateClosed:
			wrtc.Close()
		case webrtc.PeerConnectionStateFailed:
			wrtc.Close()
		}
	})

	pc.OnICEConnectionStateChange(func(st webrtc.ICEConnectionState) {
		log.Println("webrtc ice connection state change", st)
	})

	// handle ice candidate
	pc.OnICECandidate(func(cddt *webrtc.ICECandidate) {
		log.Println("got local ice candidate")
		if wrtc.onIceCandidate == nil {
			return
		}
		if cddt != nil {
			cj := cddt.ToJSON()
			wrtc.onIceCandidate(cj)
		} else {
			// there cloud be null candidate, this is legal, just ignore it
			// wrtc.onIceCandidate(nil)
		}
	})

	// handle data channel
	pc.OnDataChannel(func(dc *webrtc.DataChannel) {
		switch dc.Label() {
		// hid data
		case "hid":
			{
				wrtc.hidChannel = dc

				dc.OnClose(func() {
					log.Println("hid data channel close")
					wrtc.hidChannel.Close()
					wrtc.hidChannel = nil
				})

				dc.OnOpen(func() {
					log.Println("hid data channel open")
				})

				if wrtc.onHidMessage != nil {
					dc.OnMessage(func(msg webrtc.DataChannelMessage) {
						wrtc.onHidMessage(msg.Data)
					})
				}
			}
		case "http":
			{
				wrtc.httpChannel = dc

				dc.OnClose(func() {
					log.Println("http data channel close")
					wrtc.httpChannel.Close()
					wrtc.httpChannel = nil
				})

				dc.OnOpen(func() {
					log.Println("http data channel open")
				})

				if wrtc.onHttpMessage != nil {
					dc.OnMessage(func(msg webrtc.DataChannelMessage) {
						wrtc.onHttpMessage(msg.Data)
					})
				}
			}
		default:
			log.Println("unknown data channel", dc.Label())
		}
	})
}

func (wrtc *WebRTC) Close() {
	if wrtc.onClose != nil {
		wrtc.onClose()
	}

	if wrtc.hidChannel != nil {
		err := wrtc.hidChannel.Close()
		if err != nil {
			log.Printf("wrtc hid channel close error %s", err)
		}

		wrtc.hidChannel = nil
	}

	if wrtc.httpChannel != nil {
		err := wrtc.httpChannel.Close()
		if err != nil {
			log.Printf("wrtc http channel close error %s", err)
		}

		wrtc.httpChannel = nil
	}

	if wrtc.vt != nil {
		wrtc.vt = nil
	}

	if wrtc.pc != nil {
		err := wrtc.pc.Close()
		if err != nil {
			log.Printf("wrtc peer connection close error %s", err)
		}
	}
}

func (wrtc *WebRTC) UseOffer(offer webrtc.SessionDescription) webrtc.SessionDescription {
	err := wrtc.pc.SetRemoteDescription(offer)
	if err != nil {
		log.Fatalf("use remote offer error %s", err)
	}
	log.Println("use remote offer")

	answer, err := wrtc.pc.CreateAnswer(nil)
	if err != nil {
		log.Fatalf("create answer error %s", err)
	}

	err = wrtc.pc.SetLocalDescription(answer)
	if err != nil {
		log.Fatalf("use local answer error %s", err)
	}

	log.Println("create local answer")

	return answer
}

func (wrtc *WebRTC) UseIceCandidate(candidate webrtc.ICECandidateInit) {
	wrtc.pc.AddICECandidate(candidate)
	log.Println("add remote ice candidate")
}

func (wrtc *WebRTC) UseTrack(capability webrtc.RTPCodecCapability) {
	// Create a video track
	vt, err := webrtc.NewTrackLocalStaticSample(
		capability,
		"video",
		"kvvm",
	)

	if err != nil {
		log.Fatalf("create video track error %s", err)
	}
	log.Println("create video track")

	_, err = wrtc.pc.AddTrack(vt)
	if err != nil {
		log.Fatalf("add track error %s", err)
	}
	log.Println("add track")

	wrtc.vt = vt
	wrtc.lastFrameTime = time.Now()
}

func (wrtc *WebRTC) WriteVideoTrack(b []byte, timestamp uint64) error {
	t := time.UnixMicro(int64(timestamp))
	err := wrtc.vt.WriteSample(media.Sample{Data: b, Duration: wrtc.lastFrameTime.Sub(t)})
	wrtc.lastFrameTime = t

	return err
}

func (wrtc *WebRTC) SendHttpMessage(b []byte) error {
	if wrtc.httpChannel == nil {
		return nil
	}

	err := wrtc.hidChannel.Send(b)

	return err
}
