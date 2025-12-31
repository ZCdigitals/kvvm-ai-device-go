package webrtc

import (
	"fmt"
	"time"

	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
)

type WebRTCOnIceCandidate func(candidate *webrtc.ICECandidateInit)
type WebRTCOnDataChannel func(dataChannel *webrtc.DataChannel) bool

type WebRTC struct {
	pc *webrtc.PeerConnection

	// video track
	lastFrameTime time.Time
	vtSample      *webrtc.TrackLocalStaticSample
	vtRtp         *webrtc.TrackLocalStaticRTP

	// callback
	OnIceCandidate WebRTCOnIceCandidate
	OnDataChannel  WebRTCOnDataChannel
	OnClose        func()
}

func (wrtc *WebRTC) Open(iceServers []webrtc.ICEServer) error {
	if wrtc.pc != nil {
		return fmt.Errorf("wrtc pc exists")
	}

	config := webrtc.Configuration{
		ICEServers: iceServers,
	}

	// create peer connection
	pc, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return err
	}

	// close by peer
	pc.OnConnectionStateChange(func(pcs webrtc.PeerConnectionState) {
		switch pcs {
		case webrtc.PeerConnectionStateClosed:
		case webrtc.PeerConnectionStateFailed:
			{
				wrtc.Close()
				break
			}
		}
	})

	// ice candidate
	pc.OnICECandidate(func(i *webrtc.ICECandidate) {
		// ice candidtae cloud be null candidate
		//  this is legal, just ignore it
		if i == nil {
			return
		} else if wrtc.OnIceCandidate == nil {
			return
		}

		cj := i.ToJSON()
		wrtc.OnIceCandidate(&cj)
	})

	// data channel
	pc.OnDataChannel(func(dc *webrtc.DataChannel) {
		if wrtc.OnDataChannel == nil {
			dc.Close()
			return
		}

		using := wrtc.OnDataChannel(dc)
		if !using {
			dc.Close()
		}
	})

	wrtc.pc = pc

	return nil
}

func (wrtc *WebRTC) Close() error {
	if wrtc.OnClose != nil {
		wrtc.OnClose()
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
			return err
		}
	}

	return nil
}

func (wrtc *WebRTC) UseOffer(offer *webrtc.SessionDescription) (*webrtc.SessionDescription, error) {
	err := wrtc.pc.SetRemoteDescription(*offer)
	if err != nil {
		return nil, err
	}

	answer, err := wrtc.pc.CreateAnswer(nil)
	if err != nil {
		return nil, err
	}

	err = wrtc.pc.SetLocalDescription(answer)
	if err != nil {
		return nil, err
	}

	return &answer, nil
}

func (wrtc *WebRTC) AddIceCandidate(candidate *webrtc.ICECandidateInit) error {
	return wrtc.pc.AddICECandidate(*candidate)
}

func (wrtc *WebRTC) AddVideoTrackSample(capability webrtc.RTPCodecCapability) error {
	// Create a video track
	vt, err := webrtc.NewTrackLocalStaticSample(
		capability,
		"video",
		"kvvm",
	)
	if err != nil {
		return err
	}

	_, err = wrtc.pc.AddTrack(vt)
	if err != nil {
		return err
	}

	wrtc.vtSample = vt
	wrtc.lastFrameTime = time.Now()

	return nil
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

func (wrtc *WebRTC) AddVideoTrackRtp(capability webrtc.RTPCodecCapability) error {
	// Create a video track
	vt, err := webrtc.NewTrackLocalStaticRTP(
		capability,
		"video",
		"kvvm",
	)
	if err != nil {
		return err
	}

	_, err = wrtc.pc.AddTrack(vt)
	if err != nil {
		return err
	}

	wrtc.vtRtp = vt

	return nil
}

func (wrtc *WebRTC) WriteVideoTrackRtp(b []byte) error {
	if wrtc.vtRtp == nil {
		return nil
	}

	_, err := wrtc.vtRtp.Write(b)

	return err
}

func (wrtc *WebRTC) CreateDataChannel(label string) (*webrtc.DataChannel, error) {
	return wrtc.pc.CreateDataChannel(label, nil)
}
