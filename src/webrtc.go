package src

import (
	"log"

	"github.com/pion/mediadevices"
	"github.com/pion/mediadevices/pkg/driver/camera"
	"github.com/pion/mediadevices/pkg/prop"
	"github.com/pion/webrtc/v4"
)

type WebRTC struct {
	mediaEngine webrtc.MediaEngine
	api         *webrtc.API
}

func (wrtc *WebRTC) Init(iceServers []webrtc.ICEServer) {
	// 注册摄像头驱动
	camera.Initialize()

	// 创建媒体引擎
	wrtc.mediaEngine = webrtc.MediaEngine{}

	// 配置编解码器
	codecSelector := mediadevices.NewCodecSelector()
	codecSelector.Populate(&wrtc.mediaEngine)

	// 创建API对象
	wrtc.api = webrtc.NewAPI(webrtc.WithMediaEngine(&wrtc.mediaEngine))

	config := webrtc.Configuration{
		ICEServers: iceServers,
	}

	pc, err := wrtc.api.NewPeerConnection(config)
	if err != nil {
		log.Fatal("create peer connection error", err)
	}

	ms, err := mediadevices.GetUserMedia(mediadevices.MediaStreamConstraints{
		Video: func(constraint *mediadevices.MediaTrackConstraints) {
			constraint.DeviceID = prop.String("video1") // 使用/dev/video1
			constraint.Width = prop.Int(640)
			constraint.Height = prop.Int(480)
		},
		Codec: codecSelector,
	})
	if err != nil {
		log.Fatal("create media stream error", err)
	}

	vts := ms.GetVideoTracks()
	if len(vts) == 0 {
		log.Fatal("there is no video tracks")
	}

	rtpSender, err := pc.AddTrack(vts[0])
	if err != nil {
		log.Fatal("add track error", err)
	}

}
