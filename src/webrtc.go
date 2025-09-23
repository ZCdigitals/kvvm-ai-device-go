package src

import (
	"log"

	"github.com/pion/mediadevices"
	"github.com/pion/mediadevices/pkg/codec/vpx"

	// This is required to register screen adapter
	"github.com/pion/mediadevices/pkg/driver/screen"

	// This is required to register camera adapter
	// "github.com/pion/mediadevices/pkg/driver/camera"

	"github.com/pion/mediadevices/pkg/frame"
	"github.com/pion/mediadevices/pkg/prop"
	"github.com/pion/webrtc/v4"
)

type WebRTCOnIceCandidate func(candidate webrtc.ICECandidateInit)
type WebRTCOnHidMessage func(msg string)

type WebRTC struct {
	me        webrtc.MediaEngine
	api       *webrtc.API
	pc        *webrtc.PeerConnection
	ms        *mediadevices.MediaStream
	cs        *mediadevices.CodecSelector
	rtpSender *webrtc.RTPSender

	// ice candidate
	onIceCandidate WebRTCOnIceCandidate

	// hid data channel
	dc           *webrtc.DataChannel
	onHidMessage WebRTCOnHidMessage
}

func (wrtc *WebRTC) Init() {
	// 注册摄像头驱动
	screen.Initialize()
	// camera.Initialize()

	// 创建媒体引擎
	wrtc.me = webrtc.MediaEngine{}

	// 配置编解码器
	vp8Params, err := vpx.NewVP8Params()
	if err != nil {
		log.Fatal("create vp8 params error ", err)
	}
	codecSelector := mediadevices.NewCodecSelector(
		mediadevices.WithVideoEncoders(&vp8Params),
	)
	codecSelector.Populate(&wrtc.me)
	wrtc.cs = codecSelector

	// 创建API对象
	wrtc.api = webrtc.NewAPI(webrtc.WithMediaEngine(&wrtc.me))
}

func (wrtc *WebRTC) Open(iceServers []webrtc.ICEServer, device string, width int, height int) {
	// 准备配置
	config := webrtc.Configuration{
		ICEServers: iceServers,
	}

	// 创建PeerConnection
	pc, err := wrtc.api.NewPeerConnection(config)
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
			// wrtc.onIceCandidate(nil)
		}
	})

	// 设置媒体流
	// ms, err := mediadevices.GetUserMedia(mediadevices.MediaStreamConstraints{
	// 	Video: func(constraint *mediadevices.MediaTrackConstraints) {
	// 		constraint.DeviceID = prop.String("video1") // 使用/dev/video1
	// constraint.Width = prop.Int(width)
	// constraint.Height = prop.Int(height)
	// 	},
	// 	Codec: wrtc.cs,
	// })
	ms, err := mediadevices.GetDisplayMedia(mediadevices.MediaStreamConstraints{
		Video: func(constraint *mediadevices.MediaTrackConstraints) {
			constraint.FrameFormat = prop.FrameFormat(frame.FormatI420)
			constraint.Width = prop.Int(width)
			constraint.Height = prop.Int(height)
		},
		Codec: wrtc.cs,
	})
	if err != nil {
		log.Fatal("create media stream error ", err)
	}
	wrtc.ms = &ms
	log.Println("create media steam")

	// 添加视频轨
	vts := ms.GetVideoTracks()
	if len(vts) == 0 {
		log.Fatal("there is no video track")
	}

	rtpSender, err := pc.AddTrack(vts[0])
	if err != nil {
		log.Fatal("add track error ", err)
	}
	wrtc.rtpSender = rtpSender

	// 处理RTCP包
	go func() {
		rtcpBuf := make([]byte, 1500)
		for {
			if _, _, rtcpErr := rtpSender.Read(rtcpBuf); rtcpErr != nil {
				return
			}
		}
	}()

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

func (wrtc *WebRTC) Close() {
	if wrtc.dc != nil {
		wrtc.dc.Close()
	}
	if wrtc.rtpSender != nil {
		wrtc.rtpSender.Stop()
	}
	if wrtc.pc != nil {
		wrtc.pc.Close()
	}
}

func (wrtc *WebRTC) UseOffer(offer webrtc.SessionDescription) webrtc.SessionDescription {
	wrtc.pc.SetRemoteDescription(offer)
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
