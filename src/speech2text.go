package src

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
)

type Speech2Text struct {
	region string
	key    string
	locale string

	client *http.Client
}

const host string = "api.cognitive.microsoft.com"
const apiVersion string = "2025-10-15"

func (s *Speech2Text) url(path string) url.URL {
	query := url.Values{}

	query.Set("api-version", apiVersion)

	return url.URL{
		Scheme:   "https",
		Host:     fmt.Sprintf("%s.%s", s.region, host),
		Path:     fmt.Sprintf("%s/%s", "speechtotext", path),
		RawQuery: query.Encode(),
	}
}

type Speech2TextPostTranscriptionsTranscribeDataDefinition struct {
	Locales []string `json:"locales,omitempty"`
}

type Speech2TextPostTranscriptionsTranscribeRes struct {
	DurationMilliseconds int                          `json:"durationMilliseconds"`
	CombinedPhrases      []Speech2TextCombinedPhrases `json:"combinedPhrases,omitempty"`
	Phrases              []Speech2TextPhrase          `json:"phrases,omitempty"`
}

type Speech2TextCombinedPhrases struct {
	Text string `json:"text"`
}

type Speech2TextPhrase struct {
	OffsetMilliseconds   int               `json:"offsetMilliseconds"`
	DurationMilliseconds int               `json:"durationMilliseconds"`
	Text                 string            `json:"text"`
	Words                []Speech2TextWord `json:"words,omitempty"`
	Locale               string            `json:"locale"`
	Confidence           float64           `json:"confidence"`
}

type Speech2TextWord struct {
	OffsetMilliseconds   int    `json:"offsetMilliseconds"`
	DurationMilliseconds int    `json:"durationMilliseconds"`
	Text                 string `json:"text"`
}

func UnmarshalSpeech2TextPostTranscriptionsTranscribeRes(data []byte) (Speech2TextPostTranscriptionsTranscribeRes, error) {
	r := Speech2TextPostTranscriptionsTranscribeRes{}

	err := json.Unmarshal(data, &r)

	return r, err
}

func (s *Speech2Text) PostTranscriptionsTranscribe(audio []byte) (Speech2TextPostTranscriptionsTranscribeRes, error) {
	u := s.url("transcriptions:transcribe")

	definition := Speech2TextPostTranscriptionsTranscribeDataDefinition{}
	if s.locale != "" {
		definition.Locales = []string{s.locale}
	}
	ds, err := json.Marshal(definition)
	if err != nil {
		log.Println("speech2text post transcriptions transcribe definition marshal error", err)
		return Speech2TextPostTranscriptionsTranscribeRes{}, err
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// add definition
	err = writer.WriteField("definition", string(ds))

	// add audio
	part, err := writer.CreateFormFile("audio", "audio")
	if err != nil {
		log.Println("speech2text post transcriptions transcribe create form file error", err)
		return Speech2TextPostTranscriptionsTranscribeRes{}, err
	}
	_, err = part.Write(audio)
	if err != nil {
		log.Println("speech2text post transcriptions transcribe part write error", err)
		return Speech2TextPostTranscriptionsTranscribeRes{}, err
	}
	err = writer.Close()
	if err != nil {
		log.Println("speech2text post transcriptions transcribe writer close error", err)
		return Speech2TextPostTranscriptionsTranscribeRes{}, err
	}

	// create req
	req, err := http.NewRequest("POST", u.String(), body)
	if err != nil {
		log.Println("speech2text post transcriptions transcribe create req error", err)
		return Speech2TextPostTranscriptionsTranscribeRes{}, err
	}

	// set header
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Ocp-Apim-Subscription-Key", s.key)

	if s.client == nil {
		s.client = &http.Client{
			Timeout: 10000,
		}
	}

	// send req
	res, err := s.client.Do(req)
	if err != nil {
		log.Println("speech2text post transcriptions transcribe send req error", err)
		return Speech2TextPostTranscriptionsTranscribeRes{}, err
	}

	// read res data
	defer req.Body.Close()
	data, err := io.ReadAll(res.Body)

	if res.StatusCode >= 300 {
		return Speech2TextPostTranscriptionsTranscribeRes{}, fmt.Errorf("speech2text post transcriptions transcribe http error", res.StatusCode)
	}

	rd, err := UnmarshalSpeech2TextPostTranscriptionsTranscribeRes(data)

	return rd, err
}
