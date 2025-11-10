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

type WebRTCOnIceCandidate func(candidate *webrtc.ICECandidateInit)
type WebRTCOnDataChannelMessage func(msg []byte)

type WebRTC struct {
	pc *webrtc.PeerConnection

	// video track
	vt            *webrtc.TrackLocalStaticSample
	lastFrameTime time.Time

	// ice candidate
	onIceCandidate WebRTCOnIceCandidate

	// data channels
	dataChannels []*webrtc.DataChannel

	// on close
	onClose func()
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
			wrtc.onIceCandidate(&cj)
		} else {
			// there cloud be null candidate, this is legal, just ignore it
			// wrtc.onIceCandidate(nil)
		}
	})
}

func (wrtc *WebRTC) Close() {
	if wrtc.onClose != nil {
		wrtc.onClose()
	}

	for _, dc := range wrtc.dataChannels {
		dc.Close()
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

func (wrtc *WebRTC) UseOffer(offer *webrtc.SessionDescription) *webrtc.SessionDescription {
	err := wrtc.pc.SetRemoteDescription(*offer)
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

	return &answer
}

func (wrtc *WebRTC) UseIceCandidate(candidate *webrtc.ICECandidateInit) {
	wrtc.pc.AddICECandidate(*candidate)
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
	// log.Println("write video track", timestamp, int(timestamp), t)
	err := wrtc.vt.WriteSample(media.Sample{Data: b, Duration: t.Sub(wrtc.lastFrameTime)})
	wrtc.lastFrameTime = t

	return err
}

func (wrtc *WebRTC) CreateDataChannel(label string) *webrtc.DataChannel {
	dc, err := wrtc.pc.CreateDataChannel(label, &webrtc.DataChannelInit{})
	if err != nil {
		log.Printf("wrtc data channel create error %s", err)
		return nil
	}

	wrtc.dataChannels = append(wrtc.dataChannels, dc)

	return dc
}
