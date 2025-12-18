package src

import (
	"log"
	"sync/atomic"

	"github.com/Microsoft/cognitive-services-speech-sdk-go/audio"
	"github.com/Microsoft/cognitive-services-speech-sdk-go/speech"
)

type SpeechOnText func(text string)

type Speech struct {
	key    string
	region string

	running uint32

	audioConfig      *audio.AudioConfig
	speechConfig     *speech.SpeechConfig
	speechRecognizer *speech.SpeechRecognizer

	onText SpeechOnText
}

func (s *Speech) isRunning() bool {
	return atomic.LoadUint32(&s.running) == 1
}

func (s *Speech) setRunning(running bool) {
	if running {
		atomic.StoreUint32(&s.running, 1)
	} else {
		atomic.StoreUint32(&s.running, 0)
	}
}

func (s *Speech) open() error {
	audioConfig, err := audio.NewAudioConfigFromDefaultMicrophoneInput()
	if err != nil {
		log.Println("speech open audio config error", err)
		return err
	}
	s.audioConfig = audioConfig

	speechConfig, err := speech.NewSpeechConfigFromSubscription(s.key, s.region)
	if err != nil {
		log.Println("speeck open speech config error", err)
		return err
	}
	s.speechConfig = speechConfig

	speechRecognizer, err := speech.NewSpeechRecognizerFromConfig(speechConfig, audioConfig)
	if err != nil {
		log.Println("speech open speech recognizer error", err)
		return err
	}
	s.speechRecognizer = speechRecognizer

	// set callback
	if s.onText != nil {
		speechRecognizer.Recognizing(func(event speech.SpeechRecognitionEventArgs) {
			defer event.Close()
			s.onText(event.Result.Text)
		})
		speechRecognizer.Recognized(func(event speech.SpeechRecognitionEventArgs) {
			defer event.Close()
			s.onText(event.Result.Text)
		})
	}

	// start
	speechRecognizer.StartContinuousRecognitionAsync()

	return nil
}

func (s *Speech) close() {
	if s.speechRecognizer != nil {
		if s.isRunning() {
			s.speechRecognizer.StopContinuousRecognitionAsync()
			s.setRunning(false)
		}
		s.speechRecognizer.Close()
		s.speechRecognizer = nil
	}

	if s.speechConfig != nil {
		s.speechConfig.Close()
		s.speechConfig = nil
	}

	if s.audioConfig != nil {
		s.audioConfig.Close()
		s.audioConfig = nil
	}
}
