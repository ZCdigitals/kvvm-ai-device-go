package src

import (
	"io"
	"net/http"
)

type HttpRequestData struct {
	Id     string      `json:"id"`
	Method string      `json:"method,omitempty"`
	Url    string      `json:"url,omitempty"`
	Header http.Header `json:"header,omitempty"`

	body io.ReadCloser
}

type HttpResponseData struct {
	Id     string      `json:"id"`
	Status int         `json:"status"`
	Header http.Header `json:"header,omitempty"`

	body io.ReadCloser
}

func SendHttpRequest(r HttpRequestData) (*HttpResponseData, error) {
	req, err := http.NewRequest(r.Method, r.Url, r.body)
	if err != nil {
		return nil, err
	}
	defer r.body.Close()

	// set header
	req.Header = r.Header

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	rr := HttpResponseData{
		Id:     r.Id,
		Status: res.StatusCode,
		Header: res.Header,

		body: res.Body,
	}

	// defer res.Body.Close()

	return &rr, nil
}
