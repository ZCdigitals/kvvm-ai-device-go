package src

import (
	"log"

	// This is required to register screen adapter
	// "github.com/pion/mediadevices/pkg/driver/screen"

	// This is required to register camera adapter
	// "github.com/pion/mediadevices/pkg/driver/camera"

	"github.com/pion/webrtc/v4"
)

type WebRTCOnIceCandidate func(candidate webrtc.ICECandidateInit)
type WebRTCOnHidMessage func(msg string)

type WebRTC struct {
	pc        *webrtc.PeerConnection
	vt        *webrtc.TrackLocalStaticRTP
	rtpSender *webrtc.RTPSender

	// ice candidate
	onIceCandidate WebRTCOnIceCandidate

	// hid data channel
	dc           *webrtc.DataChannel
	onHidMessage WebRTCOnHidMessage
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
		log.Fatal("create peer connection error ", err)
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
			wrtc.onIceCandidate(cddt.ToJSON())
		} else {
			// there cloud be null candidate, this is legal, just ignore it
			// wrtc.onIceCandidate(nil)
		}
	})

	// 处理data channel
	pc.OnDataChannel(func(dc *webrtc.DataChannel) {
		// only allow hid data
		if dc.Label() != "hid" {
			log.Println("unknown data channel", dc.Label())
			return
		}
		wrtc.dc = dc

		dc.OnClose(func() {
			log.Println("data channel close")
			wrtc.dc.Close()
		})

		dc.OnOpen(func() {
			log.Println("data channel open")
		})

		dc.OnMessage(func(msg webrtc.DataChannelMessage) {
			if wrtc.onHidMessage == nil {
				return
			}
			// handle hid data
			wrtc.onHidMessage(string(msg.Data))
		})
	})
}

func (wrtc *WebRTC) Close() error {
	if wrtc.dc != nil {
		err := wrtc.dc.Close()
		if err != nil {
			return err
		}
	}

	if wrtc.pc != nil {
		err := wrtc.pc.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

func (wrtc *WebRTC) UseOffer(offer webrtc.SessionDescription) webrtc.SessionDescription {
	err := wrtc.pc.SetRemoteDescription(offer)
	if err != nil {
		log.Fatal("use remote offer error ", err)
	}
	log.Println("use remote offer")

	answer, err := wrtc.pc.CreateAnswer(nil)
	if err != nil {
		log.Fatal("create answer error ", err)
	}

	err = wrtc.pc.SetLocalDescription(answer)
	if err != nil {
		log.Fatal("use local answer error ", err)
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
	vt, err := webrtc.NewTrackLocalStaticRTP(
		capability,
		"video",
		"pion",
	)

	if err != nil {
		log.Fatal("create video track error ", err)
	}
	log.Println("create video track")

	rtpSender, err := wrtc.pc.AddTrack(vt)
	if err != nil {
		log.Fatal("add track error ", err)
	}
	log.Println("add track")

	// Read incoming RTCP packets
	// Before these packets are returned they are processed by interceptors. For things
	// like NACK this needs to be called.
	go func() {
		rtcpBuf := make([]byte, 1500)
		for {
			if _, _, rtcpErr := rtpSender.Read(rtcpBuf); rtcpErr != nil {
				return
			}
		}
	}()

	wrtc.vt = vt
	wrtc.rtpSender = rtpSender
}

func (wrtc *WebRTC) WriteVideoTrack(b []byte) error {
	_, err := wrtc.vt.Write(b)

	return err
}
