package signaling

// WebSocket API provides an interface to enable mock
type WebSocketClientI interface {
	OnClose(func())
	OnOpen(func())
	OnError(func(err error))
	Dial() error
	Close()
	OnMessage(chan string, func(messageType int, data []byte))
	Send(int, []byte) error
	SetURL(string) error
}

// For using webosocket text msg
const TextMessage int = 1
