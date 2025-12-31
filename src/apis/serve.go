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

type ServeApiOnUpdateToken func(
	accessToken string,
	accessTokenExpiresAt time.Time,
	refreshToken string,
	refreshTokenExpiresAt time.Time,
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

	OnUpdateToken ServeApiOnUpdateToken
}

func NewServeApi(url string, clientId string) ServeApi {
	return ServeApi{baseUrl: url, clientId: clientId}
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
	req.Header = api.buildHeader("application/x-www-form-urlencoded")

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

	n := time.Now()
	if api.accessToken != "" && api.accessTokenExpiresAt.After(n) {
		api.tokenMu.RUnlock()
		return api.accessToken, nil
	} else if api.refreshToken != "" && api.refreshTokenExpiresAt.After(n) {
		api.tokenMu.RUnlock()

		err := api.PostOAuthTokenRefreshToken(api.refreshToken)
		if err != nil {
			return "", err
		}

		return api.accessToken, nil
	}

	api.tokenMu.RUnlock()
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

	if api.OnUpdateToken != nil {
		log.Println("serve api update token", accessToken, refreshToken)
		api.OnUpdateToken(accessToken, accessTokenExpiresAt, refreshToken, refreshTokenExpiresAt)
	}
}

func (api *ServeApi) PostOAuthTokenCode(code string, state string) error {
	status, body, err := api.postForm(
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
	} else if status != 200 {
		return fmt.Errorf("serve api http error", status)
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
	status, body, err := api.postForm(
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
	} else if status != 200 {
		return fmt.Errorf("serve api http error", status)
	}

	data := PostOAuthTokenCodeRes{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return err
	}

	api.SetOAuthToken(data.Data.AccessToken, data.Data.AccessTokenExpiresAt, data.Data.RefreshToken, data.Data.RefreshTokenExpiresAt)

	return nil
}

func useWsUrl(u *url.URL) *url.URL {
	if u == nil {
		return nil
	}

	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	case "http":
		u.Scheme = "ws"
	default:
		log.Fatalln("unknown url schema %s", u.Scheme)
	}

	return u
}

func (api *ServeApi) UseDeviceResponse(id string) (*websocket.WebSocket, error) {
	u, err := api.buildUrl(
		fmt.Sprintf("/ws/device/%s/response", id),
		url.Values{},
	)
	if err != nil {
		return nil, err
	}

	u = useWsUrl(u)

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

	u = useWsUrl(u)

	accessToken, err := api.GetAccessToken()
	if err != nil {
		return nil, err
	}

	ws := websocket.NewWebSocket(u.String(), accessToken)

	return &ws, nil
}
