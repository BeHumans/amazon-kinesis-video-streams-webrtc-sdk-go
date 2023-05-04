package signaling

import (
	"errors"

	"github.com/gorilla/websocket"
)

// WebSocket Gorilla Client implementation
type WebSocketClient struct {
	url       *string
	conn      *websocket.Conn
	isClosed  bool
	onClose   func()
	onOpen    func()
	onError   func(err error)
	onMessage func(messageType int, data []byte)
}

// On Open Event Function
func (ws *WebSocketClient) OnOpen(f func()) {
	ws.onOpen = f
}

// On Close Event Function
func (ws *WebSocketClient) OnClose(f func()) {
	ws.onClose = f
}

// On Error Event Function
func (ws *WebSocketClient) OnError(f func(err error)) {
	ws.onError = f
}

// On OnMessage Event Function
func (ws *WebSocketClient) OnMessage(dial chan string, f func(messageType int, data []byte)) {
	// GoRutine waiting to receive messages
	go func() {
		// wait until dial finish ok
		// ensures that conn has courage before waiting for msgs
		<-dial
		for {
			messageType, message, err := ws.conn.ReadMessage()
			if err != nil {
				// Error Event triggered
				ws.onError(err)

				// Close Websocket connection
				ws.Close()
				return
			}
			f(messageType, message)
		}
	}()
}

// Open connection to websocket
func (ws *WebSocketClient) Dial() error {
	// Try to connect
	conn, _, err := websocket.DefaultDialer.Dial(*ws.url, nil)
	// Something wrong?
	if err != nil {
		// Error Event triggered
		ws.onError(err)
		return err
	}
	ws.conn = conn
	// Open Event triggered
	ws.onOpen()
	return nil
}

// Function to send data to websocket
func (ws *WebSocketClient) Send(msgType int, data []byte) error {
	err := ws.conn.WriteMessage(msgType, data)
	// Something wrong?
	if err != nil {
		// Error Event triggered
		ws.onError(err)
		return err
	}
	return nil
}

// Function to close websocket greatfully
func (ws *WebSocketClient) Close() {
	// Check if it calls when is closed
	if ws.isClosed {
		return
	}

	ws.isClosed = true

	// Websocket conection close
	ws.conn.Close()

	// Don't generate the closing event if there is no associated function
	if ws.onClose != nil {
		// Open Event triggered
		ws.onClose()
	}
}

// Set url of Websocket
func (ws *WebSocketClient) SetUrl(url string) error {
	// Check if conn exists
	if ws.conn != nil {
		return errors.New("You already have an open connection")
	}
	// Assign url value
	ws.url = &url
	return nil
}
