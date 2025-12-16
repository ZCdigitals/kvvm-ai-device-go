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
type WebRTCOnDataChannel func(dataChannel *webrtc.DataChannel) bool

type WebRTC struct {
	pc *webrtc.PeerConnection

	// video track
	vtSample      *webrtc.TrackLocalStaticSample
	lastFrameTime time.Time
	vtRtp         *webrtc.TrackLocalStaticRTP

	// ice candidate
	onIceCandidate WebRTCOnIceCandidate

	// data channel
	onDataChannel WebRTCOnDataChannel

	// on close
	onClose func()
}

func (wrtc *WebRTC) Open(iceServers []webrtc.ICEServer) {
	// 准备配置
	config := webrtc.Configuration{
		ICEServers: iceServers,
	}

	// 创建PeerConnection
	pc, err := webrtc.NewPeerConnection(config)
	if err != nil {
		log.Fatalf("webrtc create peer connection error %v", err)
	}

	wrtc.pc = pc

	pc.OnConnectionStateChange(func(st webrtc.PeerConnectionState) {
		log.Println("webrtc connect state change", st)
		switch st {
		case webrtc.PeerConnectionStateClosed:
			{
				wrtc.Close()
				break
			}
		case webrtc.PeerConnectionStateFailed:
			{
				wrtc.Close()
				break
			}
		}
	})

	pc.OnICEConnectionStateChange(func(st webrtc.ICEConnectionState) {
		log.Println("webrtc ice connection state change", st)
	})

	// handle ice candidate
	pc.OnICECandidate(func(cddt *webrtc.ICECandidate) {
		// there cloud be null candidate, this is legal, just ignore it
		if cddt == nil {
			return
		}
		log.Println("webrtc got local ice candidate")

		if wrtc.onIceCandidate == nil {
			return
		}

		cj := cddt.ToJSON()
		wrtc.onIceCandidate(&cj)
	})

	// handle data channel
	pc.OnDataChannel(func(dc *webrtc.DataChannel) {
		if wrtc.onDataChannel == nil {
			return
		}

		using := wrtc.onDataChannel(dc)
		if !using {
			dc.Close()
		}
	})
}

func (wrtc *WebRTC) Close() {
	if wrtc.onClose != nil {
		wrtc.onClose()
	}

	if wrtc.vtSample != nil {
		wrtc.vtSample = nil
	}
	if wrtc.vtRtp != nil {
		wrtc.vtRtp = nil
	}

	if wrtc.pc != nil {
		err := wrtc.pc.Close()
		if err != nil {
			log.Println("wrtc peer connection close error", err)
		}
	}
}

func (wrtc *WebRTC) UseOffer(offer *webrtc.SessionDescription) *webrtc.SessionDescription {
	err := wrtc.pc.SetRemoteDescription(*offer)
	if err != nil {
		log.Fatalf("use remote offer error %v", err)
	}
	log.Println("use remote offer")

	answer, err := wrtc.pc.CreateAnswer(nil)
	if err != nil {
		log.Fatalf("create answer error %v", err)
	}

	err = wrtc.pc.SetLocalDescription(answer)
	if err != nil {
		log.Fatalf("use local answer error %v", err)
	}

	log.Println("create local answer")

	return &answer
}

func (wrtc *WebRTC) UseIceCandidate(candidate *webrtc.ICECandidateInit) {
	wrtc.pc.AddICECandidate(*candidate)
	log.Println("add remote ice candidate")
}

func (wrtc *WebRTC) UseVideoTrackSample(capability webrtc.RTPCodecCapability) {
	// Create a video track
	vt, err := webrtc.NewTrackLocalStaticSample(
		capability,
		"video",
		"kvvm",
	)

	if err != nil {
		log.Fatalf("create video track error %v", err)
	}
	log.Println("create video track")

	_, err = wrtc.pc.AddTrack(vt)
	if err != nil {
		log.Fatalf("add track error %v", err)
	}
	log.Println("add track")

	wrtc.vtSample = vt
	wrtc.lastFrameTime = time.Now()
}

func (wrtc *WebRTC) WriteVideoTrackSample(b []byte, timestamp uint64) error {
	if wrtc.vtSample == nil {
		return nil
	}

	t := time.UnixMicro(int64(timestamp))
	// log.Println("write video track", timestamp)
	err := wrtc.vtSample.WriteSample(media.Sample{Data: b, Duration: t.Sub(wrtc.lastFrameTime)})
	wrtc.lastFrameTime = t

	return err
}

func (wrtc *WebRTC) UseVideoTrackRtp(capability webrtc.RTPCodecCapability) {
	// Create a video track
	vt, err := webrtc.NewTrackLocalStaticRTP(
		capability,
		"video",
		"kvvm",
	)

	if err != nil {
		log.Fatalf("create video track error %V", err)
	}
	log.Println("create video track")

	_, err = wrtc.pc.AddTrack(vt)
	if err != nil {
		log.Fatalf("add track error %v", err)
	}
	log.Println("add track")

	wrtc.vtRtp = vt
}

func (wrtc *WebRTC) WriteVideoTrackRtp(b []byte) error {
	if wrtc.vtRtp == nil {
		return nil
	}

	_, err := wrtc.vtRtp.Write(b)

	return err
}

func (wrtc *WebRTC) CreateDataChannel(label string) *webrtc.DataChannel {
	dc, err := wrtc.pc.CreateDataChannel(label, nil)
	if err != nil {
		log.Println("wrtc data channel create error", err)
		return nil
	}

	return dc
}
