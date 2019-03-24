package screepsapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const (
	contentType    = "application/json"
	tokenHeader    = "X-Token"
	usernameHeader = "X-Username"
)

var (
	DefaultHTTPTimeout = 10 * time.Second
)

type Client struct {
	httpClient *http.Client
	serverURL  *url.URL
	token      string
	tokenLock  sync.RWMutex
}

type Credentials struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	ServerURL string `json:"serverURL"`
	Token     string `json:"token"`
}

func NewClient(credentials Credentials) (*Client, error) {
	httpClient := &http.Client{
		Timeout: DefaultHTTPTimeout,
	}

	serverURL, err := url.Parse(credentials.ServerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse server url '%s': %s", credentials.ServerURL, err)
	}

	client := &Client{
		httpClient: httpClient,
		serverURL:  serverURL,
	}
        
        if credentials.Token == "" {
            err = client.SignIn(credentials.Email, credentials.Password)
            if err != nil {
                    return nil, fmt.Errorf("failed to login: %s", err)
            }
        } else {
            client.tokenLock.Lock()
            client.token = credentials.Token
            client.tokenLock.Unlock()
        }

	return client, nil
}

func (c *Client) Token() string {
	return c.token
}

func (c *Client) get(path string, resp interface{}, values url.Values, statusCode int) error {
	getURL, _ := url.Parse(c.serverURL.String())
	getURL.Path = fmt.Sprintf("%s/%s", apiPath, path)
	getURL.RawQuery = values.Encode()

	req, err := http.NewRequest("GET", getURL.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create new GET request: %s", err)
	}

	httpResp, err := c.do(req)
	if err != nil {
		return fmt.Errorf("failed to GET '%s': %s", getURL, err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != statusCode {
		return fmt.Errorf("failed to GET '%s': status %d", getURL, httpResp.StatusCode)
	}

	data, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return fmt.Errorf("failed to read data: %s", err)
	}
	// fmt.Printf("get-data: %s\n", data)

	err = json.Unmarshal(data, resp)
	if err != nil {
		return fmt.Errorf("failed to unmarshal data: %s", err)
	}

	apiResp, ok := resp.(APIResponse)
	if !ok || !apiResp.IsOk() {
		return fmt.Errorf("bad response")
	}

	return nil
}

func (c *Client) post(path string, req, resp interface{}, values url.Values, statusCode int) error {
	var body io.Reader
	if req != nil {
		reqJSON, err := json.Marshal(req)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %s", err)
		}
		// fmt.Printf("post: %s\n", reqJSON)
		body = bytes.NewReader(reqJSON)
	}
	postURL, _ := url.Parse(c.serverURL.String())
	postURL.Path = fmt.Sprintf("%s/%s", apiPath, path)
	if values != nil {
		postURL.RawQuery = values.Encode()
	}

	httpReq, err := http.NewRequest("POST", postURL.String(), body)
	if err != nil {
		return fmt.Errorf("failed to create new POST request: %s", err)
	}
	httpReq.Header.Set("Content-Type", contentType)

	httpResp, err := c.do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to POST '%s': %s", postURL, err)
	}
	defer httpResp.Body.Close()

	// if httpResp.StatusCode != statusCode {
	// 	return fmt.Errorf("failed to POST '%s': status %d", postURL, httpResp.StatusCode)
	// }

	data, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return fmt.Errorf("failed to read data: %s", err)
	}
	// fmt.Printf("post-data: %s\n", data)

	err = json.Unmarshal(data, resp)
	if err != nil {
		return fmt.Errorf("failed to unmarshal data: %s", err)
	}

	apiResp, ok := resp.(APIResponse)
	if !ok || !apiResp.IsOk() {
		return fmt.Errorf("bad response")
	}

	return nil
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	c.tokenLock.RLock()
	req.Header.Set(tokenHeader, c.token)
	req.Header.Set(usernameHeader, c.token)
	c.tokenLock.RUnlock()

	return c.httpClient.Do(req)
}
