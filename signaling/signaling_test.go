package signaling_test

import (
	"errors"
	"testing"
	"time"

	"github.com/BeHumans/amazon-kinesis-video-streams-webrtc-sdk-go/signaling"
	"github.com/BeHumans/amazon-kinesis-video-streams-webrtc-sdk-go/signer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// var for test porpouse
var (
	CHANNEL_ARN = "arn:aws:kinesisvideo:us-west-2:123456789012:channel/testChannel/1234567890"
	CLIENT_ID   = "TestClientId"
	REGION      = "us-west-2"
	ENDPOINT    = "wss://endpoint.kinesisvideo.amazonaws.com"

	SDP_OFFER = `{"sdp":"offer= true\nvideo= true","type":"offer"}`

	SDP_OFFER_VIEWER_STRING  = `{"action":"SDP_OFFER","messagePayload":"eyJzZHAiOiJvZmZlcj0gdHJ1ZVxudmlkZW89IHRydWUiLCJ0eXBlIjoib2ZmZXIifQ=="}`
	SDP_OFFER_VIEWER_MESSAGE = `{"messageType":"SDP_OFFER","messagePayload":"eyJzZHAiOiJvZmZlcj0gdHJ1ZVxudmlkZW89IHRydWUiLCJ0eXBlIjoib2ZmZXIifQ==","senderClientId":"TestClientId"}`
	SDP_OFFER_MASTER_STRING  = `{"action":"SDP_OFFER","messagePayload":"eyJzZHAiOiJvZmZlcj0gdHJ1ZVxudmlkZW89IHRydWUiLCJ0eXBlIjoib2ZmZXIifQ==","recipientClientId":"TestClientId"}`
	SDP_OFFER_MASTER_MESSAGE = `{"messageType":"SDP_OFFER","messagePayload":"eyJzZHAiOiJvZmZlcj0gdHJ1ZVxudmlkZW89IHRydWUiLCJ0eXBlIjoib2ZmZXIifQ=="}`

	SDP_ANSWER = `{"sdp":"offer= true\nvideo= true","type":"answer"}`

	SDP_ANSWER_VIEWER_MESSAGE = `{"messageType":"SDP_ANSWER","messagePayload":"eyJzZHAiOiJvZmZlcj0gdHJ1ZVxudmlkZW89IHRydWUiLCJ0eXBlIjoiYW5zd2VyIn0=","senderClientId":"TestClientId"}'`
	SDP_ANSWER_VIEWER_STRING  = `{"action":"SDP_ANSWER","messagePayload":"eyJzZHAiOiJvZmZlcj0gdHJ1ZVxudmlkZW89IHRydWUiLCJ0eXBlIjoiYW5zd2VyIn0="}`
	SDP_ANSWER_MASTER_MESSAGE = `{"messageType":"SDP_ANSWER","messagePayload":"eyJzZHAiOiJvZmZlcj0gdHJ1ZVxudmlkZW89IHRydWUiLCJ0eXBlIjoiYW5zd2VyIn0="}'`
	SDP_ANSWER_MASTER_STRING  = `{"action":"SDP_ANSWER","messagePayload":"eyJzZHAiOiJvZmZlcj0gdHJ1ZVxudmlkZW89IHRydWUiLCJ0eXBlIjoiYW5zd2VyIn0=","recipientClientId":"TestClientId"}`

	ICE_CANDIDATE = `{"candidate":"upd 10.111.34.88","sdpMid":"1","sdpMLineIndex":1}`

	ICE_CANDIDATE_VIEWER_STRING  = `{"action":"ICE_CANDIDATE","messagePayload":"eyJjYW5kaWRhdGUiOiJ1cGQgMTAuMTExLjM0Ljg4Iiwic2RwTWlkIjoiMSIsInNkcE1MaW5lSW5kZXgiOjF9"}`
	ICE_CANDIDATE_VIEWER_MESSAGE = `{"messageType":"ICE_CANDIDATE","messagePayload":"eyJjYW5kaWRhdGUiOiJ1cGQgMTAuMTExLjM0Ljg4Iiwic2RwTWlkIjoiMSIsInNkcE1MaW5lSW5kZXgiOjF9","senderClientId":"TestClientId"}`
	ICE_CANDIDATE_MASTER_MESSAGE = `{"messageType":"ICE_CANDIDATE","messagePayload":"eyJjYW5kaWRhdGUiOiJ1cGQgMTAuMTExLjM0Ljg4Iiwic2RwTWlkIjoiMSIsInNkcE1MaW5lSW5kZXgiOjF9"}`
	ICE_CANDIDATE_MASTER_STRING  = `{"action":"ICE_CANDIDATE","messagePayload":"eyJjYW5kaWRhdGUiOiJ1cGQgMTAuMTExLjM0Ljg4Iiwic2RwTWlkIjoiMSIsInNkcE1MaW5lSW5kZXgiOjF9","recipientClientId":"TestClientId"}`
)

// Testing signaling configures
var configMaster signaling.Config
var configViewer signaling.Config

var dateProvider signer.DateProvier

// Signer mock
type mockSigner struct {
	mock.Mock
}

// Get SignedURL Mock for simulate url Signed with AWS V4 signature
func (m *mockSigner) GetSignedURL(endpoint string, queryParams signer.QueryParams, date *time.Time) (string, error) {
	args := m.Called(endpoint, queryParams, date)
	return args.String(0), args.Error(1)
}

// WebSocket mock
type mockWebSocket struct {
	mock.Mock
	onClose   func()
	onOpen    func()
	onError   func(err error)
	onMessage func(messageType int, data []byte)
}

// Mock On Open Event Function
func (m *mockWebSocket) OnOpen(f func()) {
	m.onOpen = f
}

// Mock On Close Event Function
func (m *mockWebSocket) OnClose(f func()) {
	m.onClose = f
}

// Mock On Error Event Function
func (m *mockWebSocket) OnError(f func(err error)) {
	m.onError = f
}

// Mock On Message Event Function
func (m *mockWebSocket) OnMessage(dial chan string, f func(messageType int, data []byte)) {
	m.onMessage = f
	m.Called(dial, f)
}

// Mock Websocket Close Function
func (m *mockWebSocket) Close() {
	m.Called()
	if m.onClose != nil {
		m.onClose()
	}
	return
}

// Mock Websocket Dial Function
func (m *mockWebSocket) Dial() error {
	args := m.Called()
	m.onOpen()
	return args.Error(0)
}

// Mock Websocket Send Function
func (m *mockWebSocket) Send(msgType int, data []byte) error {
	args := m.Called(msgType, data)
	return args.Error(0)
}

// Mock set url Function
func (m *mockWebSocket) SetUrl(url string) error {
	args := m.Called(url)
	return args.Error(0)
}

// Initial tests state
func InitInfo() {

	// Signaling config for Viewer connect
	configViewer = signaling.Config{
		ChannelARN:      &CHANNEL_ARN,
		Region:          &REGION,
		ChannelEndpoint: &ENDPOINT,
		Role:            signaling.VIEWER,
		ClientId:        &CLIENT_ID,
	}

	// Signaling config for Viewer connect
	configMaster = signaling.Config{
		ChannelARN:      &CHANNEL_ARN,
		Region:          &REGION,
		ChannelEndpoint: &ENDPOINT,
		Role:            signaling.MASTER,
	}

}

// Testing New Signaling as a Viewer
func TestConstructorViewer(t *testing.T) {
	// Load Initial values
	InitInfo()

	// New Signaling
	_, err := signaling.New(&configViewer)

	//if something wrong happened
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

// Testing New Signaling as a Master
func TestConstructorMaster(t *testing.T) {
	// Load Initial values
	InitInfo()

	// Delete ClientId from config
	configMaster.ClientId = nil

	// New Signaling
	_, err := signaling.New(&configMaster)

	//if something wrong happened
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

// Testing New Signaling error when no config
func TestConstructorNoConfig(t *testing.T) {
	// Load Initial values
	InitInfo()

	// New Signaling
	_, err := signaling.New(nil)

	// Expected Error
	expectedErrorMsg := "Config cannot be nil"
	assert.EqualErrorf(t, err, expectedErrorMsg, "Error should be: %v, got: %v", expectedErrorMsg, err)
}

// Testing New Signaling error when config without ClientId
func TestConstructorNoClientId(t *testing.T) {
	// Load Initial values
	InitInfo()

	// Delete ClientId from config
	configViewer.ClientId = nil

	// New Signaling
	_, err := signaling.New(&configViewer)

	// Expected Error
	expectedErrorMsg := "clientId cannot be nil"
	assert.EqualErrorf(t, err, expectedErrorMsg, "Error should be: %v, got: %v", expectedErrorMsg, err)
}

// Testing New Signaling error when config master with ClientId
func TestConstructorMasterWithClientId(t *testing.T) {
	// Load Initial values
	InitInfo()

	// Add ClientId to config
	var clientId string = "FAKECLIENT"
	configMaster.ClientId = &clientId

	// New Signaling
	_, err := signaling.New(&configMaster)

	// Expected Error
	expectedErrorMsg := "clientId should be nil when master selected"
	assert.EqualErrorf(t, err, expectedErrorMsg, "Error should be: %v, got: %v", expectedErrorMsg, err)
}

// Testing New Signaling error when config without ChannelARN
func TestConstructorWithoutChannelARN(t *testing.T) {
	// Load Initial values
	InitInfo()

	// Delete ChannelARN from config
	configMaster.ChannelARN = nil

	// New Signaling
	_, err := signaling.New(&configMaster)

	// Expected Error
	expectedErrorMsg := "channelARN cannot be nil"
	assert.EqualErrorf(t, err, expectedErrorMsg, "Error should be: %v, got: %v", expectedErrorMsg, err)
}

// Testing New Signaling error when config without Region
func TestConstructorWithoutRegion(t *testing.T) {
	// Load Initial values
	InitInfo()

	// Delete Region from config
	configMaster.Region = nil

	// New Signaling
	_, err := signaling.New(&configMaster)

	// Expected Error
	expectedErrorMsg := "region cannot be nil"
	assert.EqualErrorf(t, err, expectedErrorMsg, "Error should be: %v, got: %v", expectedErrorMsg, err)
}

// Testing New Signaling error when config without ChannelEndpoint
func TestConstructorWithoutChannelEndpoint(t *testing.T) {
	// Load Initial values
	InitInfo()

	// Delete ChannelEndpoint from config
	configMaster.ChannelEndpoint = nil

	// New Signaling
	_, err := signaling.New(&configMaster)

	// Expected Error
	expectedErrorMsg := "channelEndpoint cannot be nil"
	assert.EqualErrorf(t, err, expectedErrorMsg, "Error should be: %v, got: %v", expectedErrorMsg, err)
}

// Testing Signaling Open as Viwer
func TestOpenConnectionAsViewer(t *testing.T) {
	// Load Initial values
	InitInfo()

	// Create channel for control flow
	c := make(chan string)

	// Create mock Signer
	ownMockSigner := &mockSigner{}

	// Expected GetSignedURL function
	ownMockSigner.On("GetSignedURL", mock.Anything, mock.Anything, mock.Anything).
		Return(mock.Anything, nil)

	// Create mock WebSocket
	ownMockWebsocket := &mockWebSocket{}

	// Expected mock WebSocket functions
	ownMockWebsocket.On("Dial").Return(nil)
	ownMockWebsocket.On("SetUrl", mock.Anything).Return(nil)
	ownMockWebsocket.On("OnMessage", mock.Anything, mock.Anything).Return()
	ownMockWebsocket.On("OnOpen", mock.AnythingOfType("func")).Return()

	// New Signaling with mock
	client, err := signaling.New(&configViewer, signaling.WithSigner(ownMockSigner), signaling.WithWebsocketClient(ownMockWebsocket))

	//if something wrong happened
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// If Open Event
	client.OnOpen(func() {
		// Expected Called with
		ownMockSigner.AssertCalled(t, "GetSignedURL", ENDPOINT, signer.QueryParams{
			"X-Amz-ChannelARN": CHANNEL_ARN,
			"X-Amz-ClientId":   CLIENT_ID},
			mock.Anything)
		c <- "done"
	})

	// Signaling Open Connection
	err = client.Open()

	// if something wrong happened
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Wait until done
	if <-c != "done" {
		t.Errorf("Unexpected error")
	}
}

// Testing Signaling Open as Master
func TestOpenConnectionAsMaster(t *testing.T) {
	// Load Initial values
	InitInfo()

	// Create channel for control flow
	c := make(chan string)

	// Create mock Signer
	ownMockSigner := &mockSigner{}
	// Expected GetSignedURL function
	ownMockSigner.On("GetSignedURL", mock.Anything, mock.Anything, mock.Anything).
		Return(mock.Anything, nil)

	// Create mock WebSocket
	ownMockWebsocket := &mockWebSocket{}

	// Expected mock WebSocket functions
	ownMockWebsocket.On("Dial").Return(nil)
	ownMockWebsocket.On("SetUrl", mock.Anything).Return(nil)
	ownMockWebsocket.On("OnMessage", mock.Anything, mock.Anything).Return()

	// New Signaling with mock
	client, err := signaling.New(&configMaster, signaling.WithSigner(ownMockSigner), signaling.WithWebsocketClient(ownMockWebsocket))

	//if something wrong happened
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// If Open Event
	client.OnOpen(func() {
		// Expected Called with
		ownMockSigner.AssertCalled(t, "GetSignedURL", ENDPOINT, signer.QueryParams{
			"X-Amz-ChannelARN": CHANNEL_ARN},
			mock.Anything)
		c <- "done"
	})

	// Signaling Open Connection
	err = client.Open()

	// if something wrong happened
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Wait until done
	if <-c != "done" {
		t.Errorf("Unexpected error")
	}
}

// Testing Signaling Open with ClockOffset
func TestOpenConnectionClockOffset(t *testing.T) {
	// Load Initial values
	InitInfo()

	// Add SystemClockOffset to config
	configViewer.SystemClockOffset = 1000000

	// Create channel for control flow
	c := make(chan string)

	// Create mock Signer
	ownMockSigner := &mockSigner{}
	// Expected GetSignedURL function
	ownMockSigner.On("GetSignedURL", mock.Anything, mock.Anything, mock.Anything).
		Return(mock.Anything, nil)

	// Create mock WebSocket
	ownMockWebsocket := &mockWebSocket{}

	// Expected mock WebSocket functions
	ownMockWebsocket.On("Dial").Return(nil)
	ownMockWebsocket.On("SetUrl", mock.Anything).Return(nil)
	ownMockWebsocket.On("OnMessage", mock.Anything, mock.Anything).Return()

	// New Signaling with mock
	client, err := signaling.New(&configViewer, signaling.WithSigner(ownMockSigner), signaling.WithWebsocketClient(ownMockWebsocket))

	//if something wrong happened
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// If Open Event
	client.OnOpen(func() {
		// Expected Called with
		ownMockSigner.AssertCalled(t, "GetSignedURL", ENDPOINT, signer.QueryParams{
			"X-Amz-ChannelARN": CHANNEL_ARN,
			"X-Amz-ClientId":   CLIENT_ID},
			mock.Anything) // TODO: Test GeDate()
		c <- "done"
	})

	// Signaling Open Connection
	err = client.Open()

	// if something wrong happened
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Wait until done
	if <-c != "done" {
		t.Errorf("Unexpected error")
	}
}

// Testing Signaling Open twice
func TestOpen2Times(t *testing.T) {
	// Load Initial values
	InitInfo()

	// Create mock Signer
	ownMockSigner := &mockSigner{}
	// Expected GetSignedURL function
	ownMockSigner.On("GetSignedURL", mock.Anything, mock.Anything, mock.Anything).Return(mock.Anything, nil)

	// Create mock WebSocket
	ownMockWebsocket := &mockWebSocket{}

	// Expected mock WebSocket functions
	ownMockWebsocket.On("Dial").Return(nil)
	ownMockWebsocket.On("SetUrl", mock.Anything).Return(nil)
	ownMockWebsocket.On("OnMessage", mock.Anything, mock.Anything).Return()

	// New Signaling with mock
	client, err := signaling.New(&configViewer, signaling.WithSigner(ownMockSigner), signaling.WithWebsocketClient(ownMockWebsocket))

	//if something wrong happened
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// If Error Event
	client.OnError(func(err error) {
		expectedErrorMsg := "Client is already open, opening, or closing"
		assert.EqualErrorf(t, err, expectedErrorMsg, "Error should be: %v, got: %v", expectedErrorMsg, err)
	})

	// Signaling Open Connection
	err = client.Open()

	// if something wrong happened
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	err = client.Open()
	// Expected Error
	expectedErrorMsg := "Client is already open, opening, or closing"
	assert.EqualErrorf(t, err, expectedErrorMsg, "Error should be: %v, got: %v", expectedErrorMsg, err)

}

// Testing Signaling Open with error
func TestOpenEmitError(t *testing.T) {
	// Load Initial values
	InitInfo()

	// Create channel for control flow
	c := make(chan string)

	// Create mock Signer
	ownMockSigner := &mockSigner{}
	// Expected GetSignedURL function
	ownMockSigner.On("GetSignedURL", mock.Anything, mock.Anything,
		mock.Anything).Return(mock.Anything, errors.New("MockError"))

	// Create mock WebSocket
	ownMockWebsocket := &mockWebSocket{}

	// Expected mock WebSocket functions
	ownMockWebsocket.On("Dial").Return(nil)
	ownMockWebsocket.On("SetUrl", mock.Anything).Return(nil)
	ownMockWebsocket.On("OnMessage", mock.Anything, mock.Anything).Return()

	// New Signaling with mock
	client, err := signaling.New(&configViewer, signaling.WithSigner(ownMockSigner), signaling.WithWebsocketClient(ownMockWebsocket))

	// if something wrong happened
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// If Error Event
	client.OnError(func(err error) {
		expectedErrorMsg := "MockError"
		assert.EqualErrorf(t, err, expectedErrorMsg, "Error should be: %v, got: %v", expectedErrorMsg, err)
		c <- "done"
	})

	// Signaling Open Connection
	err = client.Open()

	// Wait until done
	if <-c != "done" {
		t.Errorf("Unexpected error")
	}
}

// Testing Signaling Close Open connection
func TestCloseOpenConnection(t *testing.T) {
	// Load Initial values
	InitInfo()

	// Create channel for control flow
	c := make(chan string)

	// Create mock Signer
	ownMockSigner := &mockSigner{}
	// Expected GetSignedURL function
	ownMockSigner.On("GetSignedURL", mock.Anything, mock.Anything, mock.Anything).Return(mock.Anything, nil)

	// Create mock WebSocket
	ownMockWebsocket := &mockWebSocket{}

	// Expected mock WebSocket functions
	ownMockWebsocket.On("Dial").Return(nil)
	ownMockWebsocket.On("SetUrl", mock.Anything).Return(nil)
	ownMockWebsocket.On("OnMessage", mock.Anything, mock.Anything).Return()
	ownMockWebsocket.On("Close").Return()

	// New Signaling with mock
	client, err := signaling.New(&configViewer, signaling.WithSigner(ownMockSigner), signaling.WithWebsocketClient(ownMockWebsocket))

	// if something wrong happened
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// if error event
	client.OnOpen(func() {
		client.Close()
	})

	// if close event
	client.OnClose(func() {
		ownMockWebsocket.AssertCalled(t, "Close")
		c <- "done"
	})

	// Signaling Open Connection
	client.Open()

	// Wait until done
	if <-c != "done" {
		t.Errorf("Unexpected error")
	}
}

// Testing Signaling twice Close
func TestCloseOpenConnectionDoNothing(t *testing.T) {
	// Load Initial values
	InitInfo()

	// Create channels for control flow
	c := make(chan string)
	d := make(chan string)

	// Create mock Signer
	ownMockSigner := &mockSigner{}
	// Expected GetSignedURL function
	ownMockSigner.On("GetSignedURL", mock.Anything, mock.Anything, mock.Anything).Return(mock.Anything, nil)

	// Create mock WebSocket
	ownMockWebsocket := &mockWebSocket{}

	// Expected mock WebSocket functions
	ownMockWebsocket.On("Dial").Return(nil)
	ownMockWebsocket.On("SetUrl", mock.Anything).Return(nil)
	ownMockWebsocket.On("OnMessage", mock.Anything, mock.Anything).Return()
	ownMockWebsocket.On("Close").Return()

	// New Signaling with mock
	client, err := signaling.New(&configViewer, signaling.WithSigner(ownMockSigner), signaling.WithWebsocketClient(ownMockWebsocket))

	// if something wrong happened
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// if open event
	client.OnOpen(func() {
		// Twice Close
		client.Close()
		client.Close()
		d <- "done"
	})

	// if close event
	client.OnClose(func() {
		c <- "done"
	})

	// Signaling Open Connection
	client.Open()

	// Wait until done
	if <-c != "done" {
		t.Errorf("Unexpected error")
	}

	// Wait until done
	if <-d != "done" {
		t.Errorf("Unexpected error")
	}
	// Expected number of calls
	ownMockWebsocket.AssertNumberOfCalls(t, "Close", 1)

}

// Testing Signaling close without open
func TestCloseNotOpenConnectionDoNothing(t *testing.T) {
	// Load Initial values
	InitInfo()

	// Create mock Signer
	ownMockSigner := &mockSigner{}
	// Expected GetSignedURL function
	ownMockSigner.On("GetSignedURL", mock.Anything, mock.Anything, mock.Anything).Return(mock.Anything, nil)

	// Create mock WebSocket
	ownMockWebsocket := &mockWebSocket{}

	// Expected mock WebSocket functions
	ownMockWebsocket.On("Dial").Return(nil)
	ownMockWebsocket.On("SetUrl", mock.Anything).Return(nil)
	ownMockWebsocket.On("OnMessage", mock.Anything, mock.Anything).Return()
	ownMockWebsocket.On("Close").Return()

	// New Signaling with mock
	client, err := signaling.New(&configViewer, signaling.WithSigner(ownMockSigner), signaling.WithWebsocketClient(ownMockWebsocket))

	// if something wrong happened
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Close without open
	client.Close()

	// Expected number of calls
	ownMockWebsocket.AssertNumberOfCalls(t, "OnClose", 0)
}

// Testing Sdp Offer from Viewer
func TestSendSdpOfferFromViewer(t *testing.T) {
	// Load Initial values
	InitInfo()

	// Create channel for control flow
	c := make(chan string)

	// Create mock Signer
	ownMockSigner := &mockSigner{}
	// Expected GetSignedURL function
	ownMockSigner.On("GetSignedURL", mock.Anything, mock.Anything, mock.Anything).Return(mock.Anything, nil)

	// Create mock WebSocket
	ownMockWebsocket := &mockWebSocket{}
	// Expected mock WebSocket functions
	ownMockWebsocket.On("Dial").Return(nil)
	ownMockWebsocket.On("SetUrl", mock.Anything).Return(nil)
	ownMockWebsocket.On("OnMessage", mock.Anything, mock.Anything).Return()
	ownMockWebsocket.On("Close").Return()
	ownMockWebsocket.On("Send", mock.Anything, mock.Anything).Return(nil)

	// New Signaling with mock
	client, err := signaling.New(&configViewer, signaling.WithSigner(ownMockSigner), signaling.WithWebsocketClient(ownMockWebsocket))

	// if something wrong happened
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// if open event
	client.OnOpen(func() {
		client.SendSdpOffer(SDP_OFFER, nil)
		ownMockWebsocket.AssertCalled(t, "Send", signaling.TextMessage, []byte(SDP_OFFER_VIEWER_STRING))
		c <- "done"
	})

	// Signaling Open Connection
	client.Open()

	// Wait until done
	if <-c != "done" {
		t.Errorf("Unexpected error")
	}
}

// Testing Sdp Offer from Master
func TestSendSdpOfferFromMaster(t *testing.T) {
	// Load Initial values
	InitInfo()

	// Create channel for control flow
	c := make(chan string)

	// Create mock Signer
	ownMockSigner := &mockSigner{}
	// Expected GetSignedURL function
	ownMockSigner.On("GetSignedURL", mock.Anything, mock.Anything, mock.Anything).Return(mock.Anything, nil)

	// Create mock WebSocket
	ownMockWebsocket := &mockWebSocket{}
	// Expected mock WebSocket functions
	ownMockWebsocket.On("Dial").Return(nil)
	ownMockWebsocket.On("SetUrl", mock.Anything).Return(nil)
	ownMockWebsocket.On("OnMessage", mock.Anything, mock.Anything).Return()
	ownMockWebsocket.On("Close").Return()
	ownMockWebsocket.On("Send", mock.Anything, mock.Anything).Return(nil)

	// New Signaling with mock
	client, err := signaling.New(&configMaster, signaling.WithSigner(ownMockSigner), signaling.WithWebsocketClient(ownMockWebsocket))

	// if something wrong happened
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// if open event
	client.OnOpen(func() {
		client.SendSdpOffer(SDP_OFFER, &CLIENT_ID)
		ownMockWebsocket.AssertCalled(t, "Send", signaling.TextMessage, []byte(SDP_OFFER_MASTER_STRING))
		c <- "done"
	})

	// Signaling Open Connection
	client.Open()

	// Wait until done
	if <-c != "done" {
		t.Errorf("Unexpected error")
	}
}

// Testing Sdp Offer without open connection
func TestSendSdpOfferIfConnectionNotOpen(t *testing.T) {
	// Load Initial values
	InitInfo()

	// Create mock Signer
	ownMockSigner := &mockSigner{}
	// Expected GetSignedURL function
	ownMockSigner.On("GetSignedURL", mock.Anything, mock.Anything, mock.Anything).Return(mock.Anything, nil)

	// Create mock WebSocket
	ownMockWebsocket := &mockWebSocket{}
	// Expected mock WebSocket functions
	ownMockWebsocket.On("Dial").Return(nil)
	ownMockWebsocket.On("SetUrl", mock.Anything).Return(nil)
	ownMockWebsocket.On("OnMessage", mock.Anything, mock.Anything).Return()
	ownMockWebsocket.On("Close").Return()
	ownMockWebsocket.On("Send", mock.Anything, mock.Anything).Return(nil)

	// New Signaling with mock
	client, err := signaling.New(&configMaster, signaling.WithSigner(ownMockSigner), signaling.WithWebsocketClient(ownMockWebsocket))

	// if something wrong happened
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// if error event
	client.OnError(func(err error) {
		expectedErrorMsg := "Could not send message because the connection to the signaling service is not open."
		assert.EqualErrorf(t, err, expectedErrorMsg, "Error should be: %v, got: %v", expectedErrorMsg, err)
	})
	// Send Offer
	client.SendSdpOffer(SDP_OFFER_VIEWER_STRING, &CLIENT_ID)
}

// Testing Sdp Offer error from viewer add clientid
func TestSendSdpOfferIdAsViewer(t *testing.T) {
	// Load Initial values
	InitInfo()

	// Create channel for control flow
	c := make(chan string)

	// Create mock Signer
	ownMockSigner := &mockSigner{}
	// Expected GetSignedURL function
	ownMockSigner.On("GetSignedURL", mock.Anything, mock.Anything, mock.Anything).Return(mock.Anything, nil)

	// Create mock WebSocket
	ownMockWebsocket := &mockWebSocket{}
	// Expected mock WebSocket functions
	ownMockWebsocket.On("Dial").Return(nil)
	ownMockWebsocket.On("SetUrl", mock.Anything).Return(nil)
	ownMockWebsocket.On("OnMessage", mock.Anything, mock.Anything).Return()
	ownMockWebsocket.On("Close").Return()
	ownMockWebsocket.On("Send", mock.Anything, mock.Anything).Return(nil)

	// New Signaling with mock
	client, err := signaling.New(&configViewer, signaling.WithSigner(ownMockSigner), signaling.WithWebsocketClient(ownMockWebsocket))

	// if something wrong happened
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// if open event
	client.OnOpen(func() {
		client.SendSdpOffer(SDP_OFFER, &CLIENT_ID)
	})

	// if error event
	client.OnError(func(err error) {
		expectedErrorMsg := "Unexpected recipient client id. As the VIEWER, messages must not be sent with a recipient client id."
		assert.EqualErrorf(t, err, expectedErrorMsg, "Error should be: %v, got: %v", expectedErrorMsg, err)
		c <- "done"
	})

	// Signaling Open Connection
	client.Open()

	// Wait until done
	if <-c != "done" {
		t.Errorf("Unexpected error")
	}

}

// Testing Sdp Answer from viewer
func TestSendSdpAnswerfromViewer(t *testing.T) {
	// Load Initial values
	InitInfo()

	// Create channel for control flow
	c := make(chan string)

	// Create mock Signer
	ownMockSigner := &mockSigner{}
	// Expected GetSignedURL function
	ownMockSigner.On("GetSignedURL", mock.Anything, mock.Anything, mock.Anything).Return(mock.Anything, nil)

	// Create mock WebSocket
	ownMockWebsocket := &mockWebSocket{}
	// Expected mock WebSocket functions
	ownMockWebsocket.On("Dial").Return(nil)
	ownMockWebsocket.On("SetUrl", mock.Anything).Return(nil)
	ownMockWebsocket.On("OnMessage", mock.Anything, mock.Anything).Return()
	ownMockWebsocket.On("Close").Return()
	ownMockWebsocket.On("Send", mock.Anything, mock.Anything).Return(nil)

	// New Signaling with mock
	client, err := signaling.New(&configViewer, signaling.WithSigner(ownMockSigner), signaling.WithWebsocketClient(ownMockWebsocket))

	// if something wrong happened
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// if open event
	client.OnOpen(func() {
		client.SendSdpAnswer(SDP_ANSWER, nil)
		ownMockWebsocket.AssertCalled(t, "Send", signaling.TextMessage, []byte(SDP_ANSWER_VIEWER_STRING))
		c <- "done"
	})

	// Signaling Open Connection
	client.Open()

	// Wait until done
	if <-c != "done" {
		t.Errorf("Unexpected error")
	}
}

// Testing Sdp Answer from master
func TestSendSdpAnswerAsMaster(t *testing.T) {
	// Load Initial values
	InitInfo()

	// Create channel for control flow
	c := make(chan string)

	// Create mock Signer
	ownMockSigner := &mockSigner{}
	// Expected GetSignedURL function
	ownMockSigner.On("GetSignedURL", mock.Anything, mock.Anything, mock.Anything).Return(mock.Anything, nil)

	// Create mock WebSocket
	ownMockWebsocket := &mockWebSocket{}
	// Expected mock WebSocket functions
	ownMockWebsocket.On("Dial").Return(nil)
	ownMockWebsocket.On("SetUrl", mock.Anything).Return(nil)
	ownMockWebsocket.On("OnMessage", mock.Anything, mock.Anything).Return()
	ownMockWebsocket.On("Close").Return()
	ownMockWebsocket.On("Send", mock.Anything, mock.Anything).Return(nil)

	// New Signaling with mock
	client, err := signaling.New(&configMaster, signaling.WithSigner(ownMockSigner), signaling.WithWebsocketClient(ownMockWebsocket))

	// if something wrong happened
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// if open event
	client.OnOpen(func() {
		client.SendSdpAnswer(SDP_ANSWER, &CLIENT_ID)
		ownMockWebsocket.AssertCalled(t, "Send", signaling.TextMessage, []byte(SDP_ANSWER_MASTER_STRING))
		c <- "done"
	})

	// Signaling Open Connection
	client.Open()

	// Wait until done
	if <-c != "done" {
		t.Errorf("Unexpected error")
	}
}

// Testing Sdp Answer from master error without open connection
func TestSendSdpAnswerIfConnectionNotOpen(t *testing.T) {
	// Load Initial values
	InitInfo()

	// Create mock Signer
	ownMockSigner := &mockSigner{}
	// Expected GetSignedURL function
	ownMockSigner.On("GetSignedURL", mock.Anything, mock.Anything, mock.Anything).Return(mock.Anything, nil)

	// Create mock WebSocket
	ownMockWebsocket := &mockWebSocket{}
	// Expected mock WebSocket functions
	ownMockWebsocket.On("Dial").Return(nil)
	ownMockWebsocket.On("SetUrl", mock.Anything).Return(nil)
	ownMockWebsocket.On("OnMessage", mock.Anything, mock.Anything).Return()
	ownMockWebsocket.On("Close").Return()
	ownMockWebsocket.On("Send", mock.Anything, mock.Anything).Return(nil)

	// New Signaling with mock
	client, err := signaling.New(&configMaster, signaling.WithSigner(ownMockSigner), signaling.WithWebsocketClient(ownMockWebsocket))

	// if something wrong happened
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// if error event
	client.OnError(func(err error) {
		expectedErrorMsg := "Could not send message because the connection to the signaling service is not open."
		assert.EqualErrorf(t, err, expectedErrorMsg, "Error should be: %v, got: %v", expectedErrorMsg, err)
	})

	// Send SDP
	client.SendSdpAnswer(SDP_ANSWER, &CLIENT_ID)

}

// Testing Sdp Answer error from viewer add clientid
func TestSendSdpAnswerIdAsViewer(t *testing.T) {
	// Load Initial values
	InitInfo()

	// Create channel for control flow
	c := make(chan string)

	// Create mock Signer
	ownMockSigner := &mockSigner{}
	// Expected GetSignedURL function
	ownMockSigner.On("GetSignedURL", mock.Anything, mock.Anything, mock.Anything).Return(mock.Anything, nil)

	// Create mock WebSocket
	ownMockWebsocket := &mockWebSocket{}
	// Expected mock WebSocket functions
	ownMockWebsocket.On("Dial").Return(nil)
	ownMockWebsocket.On("SetUrl", mock.Anything).Return(nil)
	ownMockWebsocket.On("OnMessage", mock.Anything, mock.Anything).Return()
	ownMockWebsocket.On("Close").Return()
	ownMockWebsocket.On("Send", mock.Anything, mock.Anything).Return(nil)

	// New Signaling with mock
	client, err := signaling.New(&configViewer, signaling.WithSigner(ownMockSigner), signaling.WithWebsocketClient(ownMockWebsocket))

	// if something wrong happened
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// if open event
	client.OnOpen(func() {
		client.SendSdpAnswer(SDP_ANSWER, &CLIENT_ID)
	})

	// if error event
	client.OnError(func(err error) {
		expectedErrorMsg := "Unexpected recipient client id. As the VIEWER, messages must not be sent with a recipient client id."
		assert.EqualErrorf(t, err, expectedErrorMsg, "Error should be: %v, got: %v", expectedErrorMsg, err)
		c <- "done"
	})

	// Signaling Open Connection
	client.Open()

	// Wait until done
	if <-c != "done" {
		t.Errorf("Unexpected error")
	}

}

// Testing Ice Andidate from viewer
func TestSendIceCandidateAsViewer(t *testing.T) {
	// Load Initial values
	InitInfo()

	// Create channel for control flow
	c := make(chan string)

	// Create mock Signer
	ownMockSigner := &mockSigner{}
	// Expected GetSignedURL function
	ownMockSigner.On("GetSignedURL", mock.Anything, mock.Anything, mock.Anything).Return(mock.Anything, nil)

	// Create mock WebSocket
	ownMockWebsocket := &mockWebSocket{}
	// Expected mock WebSocket functions
	ownMockWebsocket.On("Dial").Return(nil)
	ownMockWebsocket.On("SetUrl", mock.Anything).Return(nil)
	ownMockWebsocket.On("OnMessage", mock.Anything, mock.Anything).Return()
	ownMockWebsocket.On("Close").Return()
	ownMockWebsocket.On("Send", mock.Anything, mock.Anything).Return(nil)

	// New Signaling with mock
	client, err := signaling.New(&configViewer, signaling.WithSigner(ownMockSigner), signaling.WithWebsocketClient(ownMockWebsocket))

	// if something wrong happened
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// if open event
	client.OnOpen(func() {
		client.SendIceCandidate(ICE_CANDIDATE, nil)
		ownMockWebsocket.AssertCalled(t, "Send", signaling.TextMessage, []byte(ICE_CANDIDATE_VIEWER_STRING))
		c <- "done"
	})

	// Signaling Open Connection
	client.Open()

	// Wait until done
	if <-c != "done" {
		t.Errorf("Unexpected error")
	}
}

// Testing Ice Andidate from master
func TestSendIceCandidateAsMaster(t *testing.T) {
	// Load Initial values
	InitInfo()

	// Create channel for control flow
	c := make(chan string)

	// Create mock Signer
	ownMockSigner := &mockSigner{}
	// Expected GetSignedURL function
	ownMockSigner.On("GetSignedURL", mock.Anything, mock.Anything, mock.Anything).Return(mock.Anything, nil)

	// Create mock WebSocket
	ownMockWebsocket := &mockWebSocket{}
	// Expected mock WebSocket functions
	ownMockWebsocket.On("Dial").Return(nil)
	ownMockWebsocket.On("SetUrl", mock.Anything).Return(nil)
	ownMockWebsocket.On("OnMessage", mock.Anything, mock.Anything).Return()
	ownMockWebsocket.On("Close").Return()
	ownMockWebsocket.On("Send", mock.Anything, mock.Anything).Return(nil)

	// New Signaling with mock
	client, err := signaling.New(&configMaster, signaling.WithSigner(ownMockSigner), signaling.WithWebsocketClient(ownMockWebsocket))

	// if something wrong happened
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// if open event
	client.OnOpen(func() {
		client.SendIceCandidate(ICE_CANDIDATE, &CLIENT_ID)
		ownMockWebsocket.AssertCalled(t, "Send", signaling.TextMessage, []byte(ICE_CANDIDATE_MASTER_STRING))
		c <- "done"
	})

	// Signaling Open Connection
	client.Open()

	// Wait until done
	if <-c != "done" {
		t.Errorf("Unexpected error")
	}
}

// Testing Ice Andidate from master without open connection
func TestSendIceCandidateIfConnectionNotOpen(t *testing.T) {
	// Load Initial values
	InitInfo()

	// Create mock Signer
	ownMockSigner := &mockSigner{}
	// Expected GetSignedURL function
	ownMockSigner.On("GetSignedURL", mock.Anything, mock.Anything, mock.Anything).Return(mock.Anything, nil)

	// Create mock WebSocket
	ownMockWebsocket := &mockWebSocket{}
	// Expected mock WebSocket functions
	ownMockWebsocket.On("Dial").Return(nil)
	ownMockWebsocket.On("SetUrl", mock.Anything).Return(nil)
	ownMockWebsocket.On("OnMessage", mock.Anything, mock.Anything).Return()
	ownMockWebsocket.On("Close").Return()
	ownMockWebsocket.On("Send", mock.Anything, mock.Anything).Return(nil)

	// New Signaling with mock
	client, err := signaling.New(&configMaster, signaling.WithSigner(ownMockSigner), signaling.WithWebsocketClient(ownMockWebsocket))

	// if something wrong happened
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// if error event
	client.OnError(func(err error) {
		expectedErrorMsg := "Could not send message because the connection to the signaling service is not open."
		assert.EqualErrorf(t, err, expectedErrorMsg, "Error should be: %v, got: %v", expectedErrorMsg, err)
	})

	// Send ICE
	client.SendIceCandidate(ICE_CANDIDATE, &CLIENT_ID)
}

// Testing Ice Candidate error from viewer add clientid
func TestSendIceCandidateIdAsViewer(t *testing.T) {
	// Load Initial values
	InitInfo()

	// Create channel for control flow
	c := make(chan string)

	// Create mock Signer
	ownMockSigner := &mockSigner{}
	// Expected GetSignedURL function
	ownMockSigner.On("GetSignedURL", mock.Anything, mock.Anything, mock.Anything).Return(mock.Anything, nil)

	// Create mock WebSocket
	ownMockWebsocket := &mockWebSocket{}
	// Expected mock WebSocket functions
	ownMockWebsocket.On("Dial").Return(nil)
	ownMockWebsocket.On("SetUrl", mock.Anything).Return(nil)
	ownMockWebsocket.On("OnMessage", mock.Anything, mock.Anything).Return()
	ownMockWebsocket.On("Close").Return()
	ownMockWebsocket.On("Send", mock.Anything, mock.Anything).Return(nil)

	// New Signaling with mock
	client, err := signaling.New(&configViewer, signaling.WithSigner(ownMockSigner), signaling.WithWebsocketClient(ownMockWebsocket))

	// if something wrong happened
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// if open event
	client.OnOpen(func() {
		client.SendIceCandidate(ICE_CANDIDATE, &CLIENT_ID)
	})

	// if error event
	client.OnError(func(err error) {
		expectedErrorMsg := "Unexpected recipient client id. As the VIEWER, messages must not be sent with a recipient client id."
		assert.EqualErrorf(t, err, expectedErrorMsg, "Error should be: %v, got: %v", expectedErrorMsg, err)
		c <- "done"
	})

	// Signaling Open Connection
	client.Open()

	// Wait until done
	if <-c != "done" {
		t.Errorf("Unexpected error")
	}

}

// Testing Events On Message ignoring unknown msgs
func TestEventIgnoreMsg(t *testing.T) {
	// Load Initial values
	InitInfo()

	// Create channel for control flow
	c := make(chan string)

	// Create mock Signer
	ownMockSigner := &mockSigner{}
	// Expected GetSignedURL function
	ownMockSigner.On("GetSignedURL", mock.Anything, mock.Anything, mock.Anything).Return(mock.Anything, nil)

	// Create mock WebSocket
	ownMockWebsocket := &mockWebSocket{}
	// Expected mock WebSocket functions
	ownMockWebsocket.On("Dial").Return(nil)
	ownMockWebsocket.On("SetUrl", mock.Anything).Return(nil)
	ownMockWebsocket.On("Close").Return()
	ownMockWebsocket.On("Send", mock.Anything, mock.Anything).Return(nil)
	ownMockWebsocket.On("OnMessage", mock.Anything, mock.Anything).Return()
	ownMockWebsocket.On("OnOpen", mock.Anything).Return()

	// New Signaling with mock
	client, err := signaling.New(&configViewer, signaling.WithSigner(ownMockSigner), signaling.WithWebsocketClient(ownMockWebsocket))

	// if something wrong happened
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// if sdp Offer event
	client.OnSdpOffer(func(offer *string, remoteClientId *string) {
		c <- "done"
	})

	// if open event
	client.OnOpen(func() {
		// Invalid Message
		ownMockWebsocket.On("OnMessage", mock.Anything).Return([]byte("not valid JSON"))
		ownMockWebsocket.onMessage(signaling.TextMessage, []byte("not valid JSON"))

		// Valid Message
		ownMockWebsocket.On("OnMessage", mock.Anything).Return([]byte(SDP_OFFER_MASTER_MESSAGE))
		ownMockWebsocket.onMessage(signaling.TextMessage, []byte(SDP_OFFER_MASTER_MESSAGE))
	})

	// Signaling Open Connection
	client.Open()

	// Wait until done
	if <-c != "done" {
		t.Errorf("Unexpected error")
	}
}

// Testing Events Sdp Offer from Master
func TestEventSdpOfferFromMaster(t *testing.T) {
	// Load Initial values
	InitInfo()

	// Create channel for control flow
	c := make(chan string)

	// Create mock Signer
	ownMockSigner := &mockSigner{}
	// Expected GetSignedURL function
	ownMockSigner.On("GetSignedURL", mock.Anything, mock.Anything, mock.Anything).Return(mock.Anything, nil)

	// Create mock WebSocket
	ownMockWebsocket := &mockWebSocket{}
	// Expected mock WebSocket functions
	ownMockWebsocket.On("Dial").Return(nil)
	ownMockWebsocket.On("SetUrl", mock.Anything).Return(nil)
	ownMockWebsocket.On("Close").Return()
	ownMockWebsocket.On("Send", mock.Anything, mock.Anything).Return(nil)
	ownMockWebsocket.On("OnMessage", mock.Anything, mock.Anything).Return()

	// New Signaling with mock
	client, err := signaling.New(&configViewer, signaling.WithSigner(ownMockSigner), signaling.WithWebsocketClient(ownMockWebsocket))

	// if something wrong happened
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// if sdp Offer event
	client.OnSdpOffer(func(offer *string, senderClientId *string) {
		assert.Equal(t, *offer, SDP_OFFER)
		assert.Equal(t, *senderClientId, "")
		c <- "done"
	})

	// if open event
	client.OnOpen(func() {
		ownMockWebsocket.On("OnMessage", mock.Anything).Return([]byte(SDP_OFFER_MASTER_MESSAGE))
		ownMockWebsocket.onMessage(signaling.TextMessage, []byte(SDP_OFFER_MASTER_MESSAGE))
	})

	// Signaling Open Connection
	client.Open()

	// Wait until done
	if <-c != "done" {
		t.Errorf("Unexpected error")
	}

}

// Testing Events Sdp Offer from Viewer
func TestEventSdpOfferFromViewer(t *testing.T) {
	// Load Initial values
	InitInfo()

	// Create channel for control flow
	c := make(chan string)

	// Create mock Signer
	ownMockSigner := &mockSigner{}
	// Expected GetSignedURL function
	ownMockSigner.On("GetSignedURL", mock.Anything, mock.Anything, mock.Anything).Return(mock.Anything, nil)

	// Create mock WebSocket
	ownMockWebsocket := &mockWebSocket{}
	// Expected mock WebSocket functions
	ownMockWebsocket.On("Dial").Return(nil)
	ownMockWebsocket.On("SetUrl", mock.Anything).Return(nil)
	ownMockWebsocket.On("Close").Return()
	ownMockWebsocket.On("Send", mock.Anything, mock.Anything).Return(nil)
	ownMockWebsocket.On("OnMessage", mock.Anything, mock.Anything).Return()

	// New Signaling with mock
	client, err := signaling.New(&configMaster, signaling.WithSigner(ownMockSigner), signaling.WithWebsocketClient(ownMockWebsocket))

	// if something wrong happened
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// if sdp Offer event
	client.OnSdpOffer(func(offer *string, senderClientId *string) {
		assert.Equal(t, *offer, SDP_OFFER)
		assert.Equal(t, *senderClientId, CLIENT_ID)
		c <- "done"
	})

	// if open event
	client.OnOpen(func() {
		ownMockWebsocket.On("OnMessage", mock.Anything).Return([]byte(SDP_OFFER_VIEWER_MESSAGE))
		ownMockWebsocket.onMessage(signaling.TextMessage, []byte(SDP_OFFER_VIEWER_MESSAGE))
	})

	// Signaling Open Connection
	client.Open()

	// Wait until done
	if <-c != "done" {
		t.Errorf("Unexpected error")
	}

}

// Testing Events Sdp Offer from Master and Pending ICE
func TestEventSdpOfferFromMasterAndPendingICE(t *testing.T) {
	// Load Initial values
	InitInfo()

	// Create channel for control flow
	c := make(chan string)

	// Create mock Signer
	ownMockSigner := &mockSigner{}
	// Expected GetSignedURL function
	ownMockSigner.On("GetSignedURL", mock.Anything, mock.Anything, mock.Anything).Return(mock.Anything, nil)

	// Create mock WebSocket
	ownMockWebsocket := &mockWebSocket{}
	// Expected mock WebSocket functions
	ownMockWebsocket.On("Dial").Return(nil)
	ownMockWebsocket.On("SetUrl", mock.Anything).Return(nil)
	ownMockWebsocket.On("Close").Return()
	ownMockWebsocket.On("Send", mock.Anything, mock.Anything).Return(nil)
	ownMockWebsocket.On("OnMessage", mock.Anything, mock.Anything).Return()

	// New Signaling with mock
	client, err := signaling.New(&configViewer, signaling.WithSigner(ownMockSigner), signaling.WithWebsocketClient(ownMockWebsocket))

	// if something wrong happened
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// if sdp Offer event
	client.OnSdpOffer(func(offer *string, senderClientId *string) {
		assert.Equal(t, *offer, SDP_OFFER)
		assert.Equal(t, *senderClientId, "")
		client.OnIceCandidate(func(iceCandidate *string, clientId *string) {
			assert.Equal(t, *iceCandidate, ICE_CANDIDATE)
			assert.Equal(t, *clientId, "")
		})
		c <- "done"
	})

	// if open event
	client.OnOpen(func() {
		ownMockWebsocket.On("OnMessage", mock.Anything).Return([]byte(ICE_CANDIDATE_MASTER_MESSAGE))
		ownMockWebsocket.onMessage(signaling.TextMessage, []byte(ICE_CANDIDATE_MASTER_MESSAGE))
		ownMockWebsocket.On("OnMessage", mock.Anything).Return([]byte(ICE_CANDIDATE_MASTER_MESSAGE))
		ownMockWebsocket.onMessage(signaling.TextMessage, []byte(ICE_CANDIDATE_MASTER_MESSAGE))
		ownMockWebsocket.On("OnMessage", mock.Anything).Return([]byte(SDP_OFFER_MASTER_MESSAGE))
		ownMockWebsocket.onMessage(signaling.TextMessage, []byte(SDP_OFFER_MASTER_MESSAGE))
	})

	// Signaling Open Connection
	client.Open()

	// Wait until done
	if <-c != "done" {
		t.Errorf("Unexpected error")
	}
}

// Testing Events Sdp Answer from Master
func TestEventSdpAnswerFromMaster(t *testing.T) {
	// Load Initial values
	InitInfo()

	// Create channel for control flow
	c := make(chan string)

	// Create mock Signer
	ownMockSigner := &mockSigner{}
	// Expected GetSignedURL function
	ownMockSigner.On("GetSignedURL", mock.Anything, mock.Anything, mock.Anything).Return(mock.Anything, nil)

	// Create mock WebSocket
	ownMockWebsocket := &mockWebSocket{}
	// Expected mock WebSocket functions
	ownMockWebsocket.On("Dial").Return(nil)
	ownMockWebsocket.On("SetUrl", mock.Anything).Return(nil)
	ownMockWebsocket.On("Close").Return()
	ownMockWebsocket.On("Send", mock.Anything, mock.Anything).Return(nil)
	ownMockWebsocket.On("OnMessage", mock.Anything, mock.Anything).Return()

	// New Signaling with mock
	client, err := signaling.New(&configViewer, signaling.WithSigner(ownMockSigner), signaling.WithWebsocketClient(ownMockWebsocket))

	// if something wrong happened
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// if sdp Answer event
	client.OnSdpAnswer(func(answer *string, clientId *string) {
		assert.Equal(t, *answer, SDP_ANSWER)
		assert.Equal(t, *clientId, "")
		c <- "done"
	})

	// if open event
	client.OnOpen(func() {
		ownMockWebsocket.On("OnMessage", mock.Anything).Return([]byte(SDP_ANSWER_MASTER_MESSAGE))
		ownMockWebsocket.onMessage(signaling.TextMessage, []byte(SDP_ANSWER_MASTER_MESSAGE))
	})

	// Signaling Open Connection
	client.Open()

	// Wait until done
	if <-c != "done" {
		t.Errorf("Unexpected error")
	}

}

// Testing Events Sdp Answer from Viewer
func TestEventSdpAnswerFromViewer(t *testing.T) {
	// Load Initial values
	InitInfo()

	// Create channel for control flow
	c := make(chan string)

	// Create mock Signer
	ownMockSigner := &mockSigner{}
	// Expected GetSignedURL function
	ownMockSigner.On("GetSignedURL", mock.Anything, mock.Anything, mock.Anything).Return(mock.Anything, nil)

	// Create mock WebSocket
	ownMockWebsocket := &mockWebSocket{}
	// Expected mock WebSocket functions
	ownMockWebsocket.On("Dial").Return(nil)
	ownMockWebsocket.On("SetUrl", mock.Anything).Return(nil)
	ownMockWebsocket.On("Close").Return()
	ownMockWebsocket.On("Send", mock.Anything, mock.Anything).Return(nil)
	ownMockWebsocket.On("OnMessage", mock.Anything, mock.Anything).Return()

	// New Signaling with mock
	client, err := signaling.New(&configMaster, signaling.WithSigner(ownMockSigner), signaling.WithWebsocketClient(ownMockWebsocket))

	// if something wrong happened
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// if sdp Answer event
	client.OnSdpAnswer(func(answer *string, clientId *string) {
		assert.Equal(t, *answer, SDP_ANSWER)
		assert.Equal(t, *clientId, CLIENT_ID)
		c <- "done"
	})

	// if open event
	client.OnOpen(func() {
		ownMockWebsocket.On("OnMessage", mock.Anything).Return([]byte(SDP_ANSWER_VIEWER_MESSAGE))
		ownMockWebsocket.onMessage(signaling.TextMessage, []byte(SDP_ANSWER_VIEWER_MESSAGE))
	})

	// Signaling Open Connection
	client.Open()

	// Wait until done
	if <-c != "done" {
		t.Errorf("Unexpected error")
	}

}

// Testing Events Sdp Answer from Master and Pending Ice
func TestEventSdpAnswerFromMasterAndPendingICE(t *testing.T) {
	// Load Initial values
	InitInfo()

	// Create channel for control flow
	c := make(chan string)

	// Create mock Signer
	ownMockSigner := &mockSigner{}
	// Expected GetSignedURL function
	ownMockSigner.On("GetSignedURL", mock.Anything, mock.Anything, mock.Anything).Return(mock.Anything, nil)

	// Create mock WebSocket
	ownMockWebsocket := &mockWebSocket{}
	// Expected mock WebSocket functions
	ownMockWebsocket.On("Dial").Return(nil)
	ownMockWebsocket.On("SetUrl", mock.Anything).Return(nil)
	ownMockWebsocket.On("Close").Return()
	ownMockWebsocket.On("Send", mock.Anything, mock.Anything).Return(nil)
	ownMockWebsocket.On("OnMessage", mock.Anything, mock.Anything).Return()

	// New Signaling with mock
	client, err := signaling.New(&configViewer, signaling.WithSigner(ownMockSigner), signaling.WithWebsocketClient(ownMockWebsocket))

	// if something wrong happened
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// if sdp Answer event
	client.OnSdpAnswer(func(answer *string, clientId *string) {
		assert.Equal(t, *answer, SDP_ANSWER)
		assert.Equal(t, *clientId, "")
		client.OnIceCandidate(func(iceCandidate *string, clientId *string) {
			assert.Equal(t, *iceCandidate, ICE_CANDIDATE)
			assert.Equal(t, *clientId, "")
		})
		c <- "done"
	})

	// if open event
	client.OnOpen(func() {
		ownMockWebsocket.On("OnMessage", mock.Anything).Return([]byte(ICE_CANDIDATE_MASTER_MESSAGE))
		ownMockWebsocket.onMessage(signaling.TextMessage, []byte(ICE_CANDIDATE_MASTER_MESSAGE))
		ownMockWebsocket.On("OnMessage", mock.Anything).Return([]byte(ICE_CANDIDATE_MASTER_MESSAGE))
		ownMockWebsocket.onMessage(signaling.TextMessage, []byte(ICE_CANDIDATE_MASTER_MESSAGE))
		ownMockWebsocket.On("OnMessage", mock.Anything).Return([]byte(SDP_ANSWER_MASTER_MESSAGE))
		ownMockWebsocket.onMessage(signaling.TextMessage, []byte(SDP_ANSWER_MASTER_MESSAGE))
	})

	// Signaling Open Connection
	client.Open()

	// Wait until done
	if <-c != "done" {
		t.Errorf("Unexpected error")
	}

}

// Testing Events Ice Candidate from Master
func TestEventIceCandidateFromMaster(t *testing.T) {
	// Load Initial values
	InitInfo()

	// Create channel for control flow
	c := make(chan string)

	// Create mock Signer
	ownMockSigner := &mockSigner{}
	// Expected GetSignedURL function
	ownMockSigner.On("GetSignedURL", mock.Anything, mock.Anything, mock.Anything).Return(mock.Anything, nil)

	// Create mock WebSocket
	ownMockWebsocket := &mockWebSocket{}
	// Expected mock WebSocket functions
	ownMockWebsocket.On("Dial").Return(nil)
	ownMockWebsocket.On("SetUrl", mock.Anything).Return(nil)
	ownMockWebsocket.On("Close").Return()
	ownMockWebsocket.On("Send", mock.Anything, mock.Anything).Return(nil)
	ownMockWebsocket.On("OnMessage", mock.Anything, mock.Anything).Return()

	// New Signaling with mock
	client, err := signaling.New(&configViewer, signaling.WithSigner(ownMockSigner), signaling.WithWebsocketClient(ownMockWebsocket))

	// if something wrong happened
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	// if SDP Answer Event
	client.OnSdpAnswer(func(answer *string, clientId *string) {

	})
	// if ICE  Event
	client.OnIceCandidate(func(iceCandidate *string, clientId *string) {
		assert.Equal(t, *iceCandidate, ICE_CANDIDATE)
		assert.Equal(t, *clientId, "")
		c <- "done"
	})
	// if open event
	client.OnOpen(func() {
		ownMockWebsocket.On("OnMessage", mock.Anything).Return([]byte(SDP_ANSWER_MASTER_MESSAGE))
		ownMockWebsocket.onMessage(signaling.TextMessage, []byte(SDP_ANSWER_MASTER_MESSAGE))
		ownMockWebsocket.On("OnMessage", mock.Anything).Return([]byte(ICE_CANDIDATE_MASTER_MESSAGE))
		ownMockWebsocket.onMessage(signaling.TextMessage, []byte(ICE_CANDIDATE_MASTER_MESSAGE))
	})

	// Signaling Open Connection
	client.Open()

	// Wait until done
	if <-c != "done" {
		t.Errorf("Unexpected error")
	}

}

// Testing Events Ice Candidate from Viewer
func TestEventIceCandidateFromViewer(t *testing.T) {
	// Load Initial values
	InitInfo()

	// Create channel for control flow
	c := make(chan string)

	// Create mock Signer
	ownMockSigner := &mockSigner{}
	// Expected GetSignedURL function
	ownMockSigner.On("GetSignedURL", mock.Anything, mock.Anything, mock.Anything).Return(mock.Anything, nil)

	// Create mock WebSocket
	ownMockWebsocket := &mockWebSocket{}
	// Expected mock WebSocket functions
	ownMockWebsocket.On("Dial").Return(nil)
	ownMockWebsocket.On("SetUrl", mock.Anything).Return(nil)
	ownMockWebsocket.On("Close").Return()
	ownMockWebsocket.On("Send", mock.Anything, mock.Anything).Return(nil)
	ownMockWebsocket.On("OnMessage", mock.Anything, mock.Anything).Return()

	// New Signaling with mock
	client, err := signaling.New(&configMaster, signaling.WithSigner(ownMockSigner), signaling.WithWebsocketClient(ownMockWebsocket))

	// if something wrong happened
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// if SDP Answer Event
	client.OnSdpAnswer(func(answer *string, clientId *string) {

	})

	// if ICE Event
	client.OnIceCandidate(func(iceCandidate *string, clientId *string) {
		assert.Equal(t, *iceCandidate, ICE_CANDIDATE)
		assert.Equal(t, *clientId, CLIENT_ID)
		c <- "done"
	})

	// if open Event
	client.OnOpen(func() {
		ownMockWebsocket.On("OnMessage", mock.Anything).Return([]byte(SDP_ANSWER_VIEWER_MESSAGE))
		ownMockWebsocket.onMessage(signaling.TextMessage, []byte(SDP_ANSWER_VIEWER_MESSAGE))
		ownMockWebsocket.On("OnMessage", mock.Anything).Return([]byte(ICE_CANDIDATE_VIEWER_MESSAGE))
		ownMockWebsocket.onMessage(signaling.TextMessage, []byte(ICE_CANDIDATE_VIEWER_MESSAGE))
	})

	// Signaling Open Connection
	client.Open()

	// Wait until done
	if <-c != "done" {
		t.Errorf("Unexpected error")
	}

}
