package screepsws

import (
    "encoding/json"
    "fmt"
    "net/url"
    "strings"
    "sync"
    "context"

    "github.com/gorilla/websocket"
    "github.com/hinshun/screepsapi/screepstype"
    tomb "gopkg.in/tomb.v2"
)

type webSocket struct {
    conn          *websocket.Conn
    serverURL     *url.URL
    token         string
    authenticated bool
    authLock      sync.RWMutex
    sendQueue     []string
    subscriptions map[string]chan []byte
    subLock      sync.Mutex
    tomb          *tomb.Tomb
}

func NewWebSocket(rawServerURL, token string, lifetime context.Context) (WebSocket, error) {
    tomb, _ := tomb.WithContext(lifetime)
    
    serverURL, err := url.Parse(rawServerURL)
    if err != nil {
            return nil, fmt.Errorf("failed to parse server url '%s': %s", rawServerURL, err)
    }

    ws := &webSocket{
            serverURL:     serverURL,
            token:         token,
            subscriptions: make(map[string]chan []byte),
            tomb:          tomb,
    }

    err = ws.connect()
    if err != nil {
            return nil, fmt.Errorf("failed to connect to '%s': %s", serverURL.String(), err)
    }

    return ws, nil
}

func (ws *webSocket) Close() error {
        if ws.conn == nil {
                return fmt.Errorf("websocket is not connected")
        }

        ws.tomb.Killf("Closed")
        ws.authLock.Lock()
        ws.authenticated = false
        ws.sendQueue = ws.sendQueue[:0]
        ws.authLock.Unlock()

        err := ws.conn.Close()
        if err != nil {
                return err
        }
        ws.conn = nil

        return nil
}

func (ws *webSocket) Wait() error {
        return ws.tomb.Wait()
}

func (ws *webSocket) Subscribe(channel string) (<-chan []byte, error) {
        ws.subLock.Lock()
        defer ws.subLock.Unlock()
        
	_, exists := ws.subscriptions[channel]
	if exists {
		return nil, fmt.Errorf("channel '%s' already subscribed", channel)
	}

	err := ws.send(fmt.Sprintf(subscribeFormat, channel))
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe: %s", err)
	}

	dataChan := make(chan []byte, 32)
	ws.subscriptions[channel] = dataChan

	return dataChan, nil
}

func (ws *webSocket) Unsubscribe(channel string) error {
        ws.subLock.Lock()
        defer ws.subLock.Unlock()
        
	err := ws.send(fmt.Sprintf(unsubscribeFormat, channel))
	if err != nil {
		return fmt.Errorf("failed to unsubscribe: %s", err)
	}

	dataChan, exists := ws.subscriptions[channel]
	if exists {
		close(dataChan)
		delete(ws.subscriptions, channel)
	}

	return nil
}

func (ws *webSocket) connect() error {
	if ws.conn != nil {
		return fmt.Errorf("websocket is already connected")
	}

	websocketURL, _ := url.Parse(ws.serverURL.String())
	websocketURL.Scheme = strings.Replace(websocketURL.Scheme, "http", "ws", 1)
	websocketURL.Path = socketPath

	conn, _, err := websocket.DefaultDialer.Dial(websocketURL.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create websocket connection: %s", err)
	}
	ws.conn = conn

	err = ws.authenticate(ws.token)
	if err != nil {
		return fmt.Errorf("failed to authenticate: %s", err)
	}

	err = ws.setGZIP(true)
	if err != nil {
		return fmt.Errorf("failed to enable gzip: %s", err)
	}

	ws.tomb.Go(ws.listen)

	return nil
}

func (ws *webSocket) authenticate(token string) error {
	err := ws.conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf(authFormat, ws.token)))
	if err != nil {
		return fmt.Errorf("failed to authenticate: %s", err)
	}
	return nil
}

func (ws *webSocket) setGZIP(enable bool) error {
	arg := "off"
	if enable {
		arg = "on"
	}

	err := ws.conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf(gzipFormat, arg)))
	if err != nil {
		return fmt.Errorf("failed to authenticate: %s", err)
	}
	return nil
}

func (ws *webSocket) send(data string) error {
	ws.authLock.RLock()
	defer ws.authLock.RUnlock()
	if !ws.authenticated {
		ws.sendQueue = append(ws.sendQueue, data)
		return nil
	}

	// fmt.Printf("websocket: %s\n", data)
	err := ws.conn.WriteMessage(websocket.TextMessage, []byte(data))
	if err != nil {
		return fmt.Errorf("failed to send '%s': %s", data, err)
	}
	return nil
}

func (ws *webSocket) receive() (data []byte, err error) {
	_, data, err = ws.conn.ReadMessage()
	if err != nil {
		return
	}
	// fmt.Printf("websocket-data: %s\n", data)
	return
}

func (ws *webSocket) listen() error {
	for i := 0; i < 4; i++ {
		_, err := ws.receive()
		if err != nil {
			return fmt.Errorf("failed to receive connection handshake: %s\n", err)
		}
	}

	ws.authLock.Lock()
	ws.authenticated = true
	ws.authLock.Unlock()

	for len(ws.sendQueue) > 0 {
		data := ws.sendQueue[0]
		err := ws.send(data)
		if err != nil {
			return fmt.Errorf("failed to send data off queue '%s': %s\n", data, err)
		}
		ws.sendQueue = ws.sendQueue[1:]
	}

	for {
		select {
		case <-ws.tomb.Dying():
			return nil
		default:
			err := ws.receiveFrame()
			if err != nil {
				return fmt.Errorf("failed to receive frame: %s\n", err)
			}
		}
	}
	
	return nil
}

func (ws *webSocket) receiveFrame() error {
	// When the websocket connection is closed, a blocking receive will exit due
	// to the closed connection, however gorilla/websocket then panics with nil
	// pointer exception now that the connection is closed.
	defer func() {
		r := recover()
		if r != nil {
                    ws.tomb.Killf("%v", r)
		}
	}()

	data, err := ws.receive()
	if err != nil {
                return fmt.Errorf("failed to receive data: %s", err)
	}

	if len(data) < len(screepstype.GzipPrefix)+2 {
		return fmt.Errorf("frame data too small: %s", data)
	}

	if string(data[:len(screepstype.GzipPrefix)]) == screepstype.GzipPrefix {
		data, err = ws.handleGzippedData(data)
		if err != nil {
			return fmt.Errorf("failed to handle gzipped data: %s", err)
		}
	}

	err = ws.handleData(data)
	if err != nil {
		return fmt.Errorf("failed to handle data: %s", err)
	}

	return nil
}

func (ws *webSocket) handleData(data []byte) error {
	resp := make([]json.RawMessage, 2)
	err := json.Unmarshal(data, &resp)
        
	if err != nil {
		return fmt.Errorf("failed to unmarshal received data '%s': %s", data, err)
	}

	channel, err := resp[0].MarshalJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal channel name: %s", err)
	}
        var keys []string
        for sub, _ := range ws.subscriptions {
            keys = append(keys, sub)
        }

	channelData, err := resp[1].MarshalJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal channel data: %s", err)
	}

        ws.subLock.Lock()
        defer ws.subLock.Unlock()
	subscription, ok := ws.subscriptions[string(channel[1:len(channel)-1])]
	if ok {
		subscription <- channelData
	}

	return nil
}

func (ws *webSocket) handleGzippedData(data []byte) ([]byte, error) {
	unzippedData, err := screepstype.Unzip(string(data), screepstype.CompressionTypeZlib)
	if err != nil {
		return nil, fmt.Errorf("failed to unzip gzipped data: %s", err)
	}

	return unzippedData, nil
}
