package apis

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"device-go/src/libs/websocket"
)

type ServeApi struct {
	// client
	client http.Client

	// base url
	baseUrl string

	// oauth
	clientId              string
	accessToken           string
	accessTokenExpiresAt  time.Time
	refreshToken          string
	refreshTokenExpiresAt time.Time
	tokenMu               sync.RWMutex
}

func NewServeApi(url string) ServeApi {
	return ServeApi{baseUrl: url}
}

func (api *ServeApi) buildUrl(path string, query url.Values) (*url.URL, error) {
	u, err := url.Parse(api.baseUrl)
	if err != nil {
		log.Println("serve api build url error", err)
		return nil, err
	}

	u.Path = path

	u.RawQuery = query.Encode()

	return u, nil
}

func (api *ServeApi) buildHeader(contentType string) http.Header {
	api.tokenMu.RLock()
	defer api.tokenMu.RUnlock()

	h := http.Header{}

	// use access token
	if api.accessToken != "" && api.accessTokenExpiresAt.After(time.Now()) {
		h.Add("Authorization", fmt.Sprintf("Bearer %s", api.accessToken))
	}

	// content type cloud be empty
	if contentType != "" {
		h.Add("Content-Type", contentType)
	}
	h.Add("Accept", "application/json")

	return h
}

func (api *ServeApi) doRequest(req *http.Request) (int, []byte, error) {
	// do http
	res, err := api.client.Do(req)
	if err != nil {
		return res.StatusCode, nil, err
	}

	// read res body
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return res.StatusCode, nil, err
	}

	return res.StatusCode, body, nil

}

func (api *ServeApi) get(path string, query url.Values) (int, []byte, error) {
	// build url
	u, err := api.buildUrl(path, query)
	if err != nil {
		return 0, nil, err
	}

	// build req
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return 0, nil, err
	}

	// build header
	req.Header = api.buildHeader("")

	// do http
	return api.doRequest(req)
}

func (api *ServeApi) post(path string, query url.Values, data any) (int, []byte, error) {
	// build url
	u, err := api.buildUrl(path, query)
	if err != nil {
		return 0, nil, err
	}

	// build body
	jd, err := json.Marshal(data)
	if err != nil {
		return 0, nil, err
	}
	var bodyReader io.Reader
	if data != nil {
		bodyReader = bytes.NewReader(jd)
	}

	// build req
	req, err := http.NewRequest("POST", u.String(), bodyReader)
	if err != nil {
		return 0, nil, err
	}

	// build header
	req.Header = api.buildHeader("application/json")

	// do http
	return api.doRequest(req)
}

func (api *ServeApi) postForm(path string, query url.Values, form url.Values) (int, []byte, error) {
	// build url
	u, err := api.buildUrl(path, query)
	if err != nil {
		return 0, nil, err
	}

	// build req
	req, err := http.NewRequest("POST", u.String(), bytes.NewBufferString(form.Encode()))
	if err != nil {
		return 0, nil, err
	}

	// build header
	req.Header = api.buildHeader("application/json")

	// do http
	return api.doRequest(req)
}

type PostOAuthTokenCodeResData struct {
	AccessToken           string    `json:"accessToken"`
	AccessTokenExpiresAt  time.Time `json:"accessTokenExpiresAt"`
	RefreshToken          string    `json:"refreshToken"`
	RefreshTokenExpiresAt time.Time `json:"refreshTokenExpiresAt"`
}

type PostOAuthTokenCodeRes struct {
	Code int                        `json:"code"`
	Msg  string                     `json:"msg"`
	Data *PostOAuthTokenCodeResData `json:"data,omitempty"`
}

func (api *ServeApi) GetAccessToken() (string, error) {
	api.tokenMu.RLock()
	defer api.tokenMu.RUnlock()

	n := time.Now()
	if api.accessToken != "" && api.accessTokenExpiresAt.After(n) {
		return api.accessToken, nil
	} else if api.refreshToken != "" && api.refreshTokenExpiresAt.After(n) {
		err := api.PostOAuthTokenRefreshToken(api.refreshToken)
		if err != nil {
			return "", err
		}
		return api.accessToken, nil
	}

	return "", fmt.Errorf("serve api oauth token null auth")
}

func (api *ServeApi) SetOAuthToken(
	accessToken string,
	accessTokenExpiresAt time.Time,
	refreshToken string,
	refreshTokenExpiresAt time.Time,
) {
	api.tokenMu.Lock()
	defer api.tokenMu.Unlock()

	api.accessToken = accessToken
	api.accessTokenExpiresAt = accessTokenExpiresAt

	if refreshToken != "" {
		api.refreshToken = refreshToken
		api.refreshTokenExpiresAt = refreshTokenExpiresAt
	}
}

func (api *ServeApi) PostOAuthTokenCode(code string, state string) error {
	_, body, err := api.postForm(
		"/oauth/token",
		url.Values{},
		url.Values{
			"code":       []string{code},
			"state":      []string{state},
			"grant_type": []string{"authorization_code"},
			"client_id":  []string{api.clientId},
		},
	)

	if err != nil {
		return err
	}

	data := PostOAuthTokenCodeRes{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return err
	}

	api.SetOAuthToken(data.Data.AccessToken, data.Data.AccessTokenExpiresAt, data.Data.RefreshToken, data.Data.RefreshTokenExpiresAt)

	return nil
}

func (api *ServeApi) PostOAuthTokenRefreshToken(refreshToken string) error {
	_, body, err := api.postForm(
		"/oauth/token",
		url.Values{},
		url.Values{
			"refresh_token": []string{refreshToken},
			"grant_type":    []string{"refresh_token"},
			"client_id":     []string{api.clientId},
		},
	)

	if err != nil {
		api.SetOAuthToken("", time.Time{}, "", time.Time{})
		return err
	}

	data := PostOAuthTokenCodeRes{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return err
	}

	api.SetOAuthToken(data.Data.AccessToken, data.Data.AccessTokenExpiresAt, data.Data.RefreshToken, data.Data.RefreshTokenExpiresAt)

	return nil
}

func (api *ServeApi) UseDeviceResponse(id string) (*websocket.WebSocket, error) {
	u, err := api.buildUrl(
		fmt.Sprintf("/ws/device/%s/response", id),
		url.Values{},
	)
	if err != nil {
		return nil, err
	}

	accessToken, err := api.GetAccessToken()
	if err != nil {
		return nil, err
	}

	ws := websocket.NewWebSocket(u.String(), accessToken)

	return &ws, nil
}

func (api *ServeApi) UseDeviceSTT(id string) (*websocket.WebSocket, error) {
	u, err := api.buildUrl(
		fmt.Sprintf("/ws/device/%s/stt", id),
		url.Values{},
	)
	if err != nil {
		return nil, err
	}

	accessToken, err := api.GetAccessToken()
	if err != nil {
		return nil, err
	}

	ws := websocket.NewWebSocket(u.String(), accessToken)

	return &ws, nil
}
