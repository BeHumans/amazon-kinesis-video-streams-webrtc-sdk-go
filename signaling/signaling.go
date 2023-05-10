package signaling

import (
	"bytes"
	b64 "encoding/base64"
	"encoding/json"
	"errors"
	"time"

	"github.com/BeHumans/amazon-kinesis-video-streams-webrtc-sdk-go/signer"
	signerV4 "github.com/BeHumans/amazon-kinesis-video-streams-webrtc-sdk-go/signer/v4"
	"github.com/aws/aws-sdk-go/aws/credentials"
)

// Signaling msg format for Sending
type WebSocketSignalingMessageSend struct {
	MessageType       MessageType `json:"action,omitempty"`
	MessagePayload    string      `json:"messagePayload,omitempty"`
	RecipientClientID string      `json:"recipientClientId,omitempty"`
}

// Signaling msg format for Reception
type WebSocketSignalingMessageReceive struct {
	MessageType    MessageType `json:"messageType,omitempty"`
	MessagePayload string      `json:"messagePayload,omitempty"`
	SenderClientID string      `json:"senderClientId,omitempty"`
}

// Default service for v4 signature
var service = "kinesisvideo"

// Signaling configure
type Config struct {
	ChannelARN        *string            // ARN AWS Signaling Channel
	ChannelEndpoint   *string            // WSS URL AWS signaling
	Region            *string            // AWS Region
	Role              Role               // Master or viewer actor
	ClientID          *string            // Id Client for AWS signaling protocol
	SystemClockOffset int                // An offset value in milliseconds to apply to all signing times
	CredentialsValue  *credentials.Value // AWS Credentials
}

// Signaling Message Type
type MessageType string

const (
	sdpAnswer    MessageType = "SDP_ANSWER"
	sdpOffer     MessageType = "SDP_OFFER"
	iceCandidate MessageType = "ICE_CANDIDATE"
)

// Signaling connection State Type
type ReadyStateType string

const (
	connecting ReadyStateType = "CONNECTING"
	open       ReadyStateType = "OPEN"
	closing    ReadyStateType = "CLOSING"
	closed     ReadyStateType = "CLOSED"
)

// Signaling client
type Client struct {
	readyState                     ReadyStateType                               // Signaling cient connection status
	config                         Config                                       // Signaling client configuration
	signer                         signer.APII                                  // V4 AWS Signer
	dateProvider                   *signer.DateProvier                          // Date provider for V4 AWS Signer
	wsClient                       WebSocketClientI                             // Websocket client
	onOpen                         func()                                       // Function for Open Event
	onError                        func(err error)                              // Function for Error Event
	onClose                        func()                                       // Function for Close Event
	onSdpAnswer                    func(answer *string, clientID *string)       // Function for Sdp Answer Event
	onSdpOffer                     func(offer *string, remoteClientID *string)  // Function for Sdp Offer Event
	onIceCandidate                 func(iceCandidate *string, clientID *string) // Function for Ice Candidate Event
	hasReceivedRemoteSDPByClientID map[string]bool                              // Maps for manage receive remote SDP by clientID
	pendingIceCandidatesByClientID map[string][]string                          // Maps for manage pending Ice Candidate by clientID
}

// On Open Event Function
func (sc *Client) OnOpen(f func()) {
	sc.onOpen = f
}

// On Close Event Function
func (sc *Client) OnClose(f func()) {
	sc.onClose = f
}

// OnError Event Function
func (sc *Client) OnError(f func(err error)) {
	sc.onError = f
}

// OnMessage Event Function
func (sc *Client) onMessage(data []byte) {

	var messageParsed WebSocketSignalingMessageReceive

	// New decoder
	dec := json.NewDecoder(bytes.NewReader(data))

	// Decode data, unknown message ?
	if err := dec.Decode(&messageParsed); err != nil {
		return
	}

	// Decode Base 64 payload message property
	decodedMessagePayload, err := b64.StdEncoding.DecodeString(messageParsed.MessagePayload)

	// error payload?
	if err != nil {
		return
	}

	// Decode message to string
	var messagePayloadParsed = string(decodedMessagePayload)

	switch messageParsed.MessageType {
	// When receive a SDP Offer
	case sdpOffer:
		// Trigger on Sdp Offer Event
		sc.onSdpOffer(&messagePayloadParsed, &messageParsed.SenderClientID)
		sc.emitPendingIceCandidates(&messageParsed.SenderClientID)
		return
	// When receive a SDP Answer
	case sdpAnswer:
		// trigger on Sdp Answer Event
		sc.onSdpAnswer(&messagePayloadParsed, &messageParsed.SenderClientID)
		sc.emitPendingIceCandidates(&messageParsed.SenderClientID)
		return
	// When receive a Ice Candidate
	case iceCandidate:
		sc.emitOrQueueIceCandidate(&messagePayloadParsed, &messageParsed.SenderClientID)
		return
	default:
		// Unknown message
		return
	}
}

// On Sdp Answer Event Function
func (sc *Client) OnSdpAnswer(f func(answer *string, clientID *string)) {
	sc.onSdpAnswer = f
}

// On Sdp Offer Event Function
func (sc *Client) OnSdpOffer(f func(offer *string, remoteClientID *string)) {
	sc.onSdpOffer = f
}

// On ICE Candidate Event Function
func (sc *Client) OnIceCandidate(f func(iceCandidate *string, clientID *string)) {
	sc.onIceCandidate = f
}

// Optional parameters

// Use own websocket client implementation
func WithWebsocketClient(websocket WebSocketClientI) func(*Client) {
	return func(sc *Client) {
		sc.wsClient = websocket
	}
}

// Use own v4 AWS signer implementation
func WithSigner(signer signer.APII) func(*Client) {
	return func(sc *Client) {
		sc.signer = signer
	}
}

// Use own Date Provider implementation
func WithDateProvider(dateProvider signer.DateProvier) func(*Client) {
	return func(sc *Client) {
		sc.dateProvider = &dateProvider
	}
}

// New signaling client
func New(config *Config, options ...func(*Client)) (*Client, error) {

	// Config must never be nil
	if config == nil {
		return nil, errors.New("Config cannot be nil")
	}

	// New Signaling client with initial values
	sc := &Client{
		readyState:                     closed,
		config:                         *config,
		hasReceivedRemoteSDPByClientID: make(map[string]bool),
		pendingIceCandidatesByClientID: make(map[string][]string),
	}

	// Getting other optional parameters
	for _, o := range options {
		o(sc)
	}

	// Nil Checkers

	// When Viewer actor
	if config.Role == Viewer {
		// Config clientID must never be nil
		if config.ClientID == nil {
			return nil, errors.New("clientID cannot be nil")
		}
	}

	// When Master actor
	if config.Role == Master {
		// Config clientID always nil
		if config.ClientID != nil {
			return nil, errors.New("clientID should be nil when master selected")
		}
	}

	// Config ChannelARN must never be nil
	if config.ChannelARN == nil {
		return nil, errors.New("channelARN cannot be nil")
	}

	// Config Region must never be nil
	if config.Region == nil {
		return nil, errors.New("region cannot be nil")
	}

	// Config ChannelEndpoint must never be nil
	if config.ChannelEndpoint == nil {
		return nil, errors.New("channelEndpoint cannot be nil")
	}

	// If you are not using our signer
	if sc.signer == nil {
		var kinesisVideoSigner signer.APII
		var err error
		// Do you have own Credentials ?
		if config.CredentialsValue != nil {
			// Create new V4 signer with own Credentials
			kinesisVideoSigner, err = signerV4.New(signerV4.WithRegion(*config.Region),
				signerV4.WithService(service), signerV4.WithCredentialsValue(config.CredentialsValue))
		} else {
			// Create new V4 signer using AWS machine credentials provider
			kinesisVideoSigner, err = signerV4.New(signerV4.WithRegion(*config.Region),
				signerV4.WithService(service))
		}

		// if something wrong happened when sign
		if err != nil {
			return nil, err
		}

		// Assing signer
		sc.signer = kinesisVideoSigner
	}

	// If you are not using our date provider
	if sc.dateProvider == nil {
		// Assing date provider
		sc.dateProvider = &signer.DateProvier{
			ClockOffset: time.Duration(config.SystemClockOffset),
		}
	}
	// If you are not using our websocket client
	if sc.wsClient == nil {
		sc.wsClient = &WebSocketClient{}
	}

	// return signaling client
	return sc, nil
}

// Open Signaling Client
func (sc *Client) Open() error {
	// Check if reOpen action
	if sc.readyState != closed {
		err := errors.New("client is already open, opening, or closing")
		if sc.onError != nil {
			// Trigger Error Event
			sc.onError(err)
		}
		return err
	}

	// Change signaling client state
	sc.readyState = connecting

	// Go Rutine for connect to websocket signaling channel
	go func() {

		// Prepare data for sign request
		queryParams := signer.QueryParams{
			"X-Amz-channelARN": *sc.config.ChannelARN,
		}
		// If viewer actor
		if sc.config.Role == Viewer {
			queryParams["X-Amz-ClientID"] = *sc.config.ClientID
		}

		// AWS V4 Sing channel endpoint uri
		signedURL, err := sc.signer.GetSignedURL(*sc.config.ChannelEndpoint, queryParams, sc.dateProvider.GetDate())

		// if something wrong happened
		if err != nil {
			// Trigger Error Event
			sc.onError(err)
		}

		// If other process changed signaling client status nothing to do
		if sc.readyState != connecting {
			return
		}

		// Set signed url to websocket client
		err = sc.wsClient.SetURL(signedURL)
		if err != nil {
			sc.onError(err)
		}

		// When websocket Open event do
		sc.wsClient.OnOpen(func() {
			sc.readyState = open
			if sc.onOpen != nil {
				sc.onOpen()
			}
		})

		// When websocket Error event do
		sc.wsClient.OnError(func(err error) {
			sc.onError(err)
		})

		// When websocket Close event do
		sc.wsClient.OnClose(func() {
			sc.readyState = closed
			sc.wsClient = nil
			sc.onClose()
		})

		// Wait channel define for OnMessage lock until dial Ok
		dial := make(chan string)

		// When websocket Message event do
		sc.wsClient.OnMessage(dial, func(messageType int, data []byte) {
			sc.onMessage(data)
		})

		// Dial websocket client
		err = sc.wsClient.Dial()

		// if something wrong happened
		if err != nil {
			// Trigger Error Event
			sc.onError(err)
		}

		// Dial Ok, unlock channel
		dial <- "done"

	}()

	return nil
}

// Close signaling client
func (sc *Client) Close() {
	// If websocket exists
	if sc.wsClient != nil {
		// Change signaling client status
		sc.readyState = closing
		// Close websocket client
		sc.wsClient.Close()

	} else if sc.readyState != closed {
		// trigger Close event
		sc.onClose()
	}
	// Nothing to do when websocket client does not exist and signaling client status is different than CLOSED
}

// Use for emit Ice Candidate Messages when signaling client has recive SDP message
func (sc *Client) emitOrQueueIceCandidate(iceCandidate *string, clientID *string) {
	var clientIDKEY string

	// if you don't have client Id use Default Client Id
	if clientID == nil {
		clientIDKEY = string(DefaultClientID)
	} else {
		clientIDKEY = *clientID
	}

	// If signaling client has recive SDP message
	if sc.hasReceivedRemoteSDPByClientID[clientIDKEY] {
		// trigger Ice Candidate Event
		sc.onIceCandidate(iceCandidate, clientID)
	} else {
		// If it is the first Ice Candidate message for this client Id
		if sc.pendingIceCandidatesByClientID == nil || sc.pendingIceCandidatesByClientID[clientIDKEY] == nil {
			sc.pendingIceCandidatesByClientID[clientIDKEY] = []string{}
		}
		// Queue Ice Candidate Message for this client Id
		sc.pendingIceCandidatesByClientID[clientIDKEY] = append(sc.pendingIceCandidatesByClientID[clientIDKEY], *iceCandidate)
	}

}

// Use for emit Ice Candidate Messages
func (sc *Client) emitPendingIceCandidates(clientID *string) {
	var clientIDKEY string

	// if you don't have client Id use Default Client Id
	if clientID == nil {
		clientIDKEY = string(DefaultClientID)
	} else {
		clientIDKEY = *clientID
	}

	// Set Sdp message receive for this client id
	sc.hasReceivedRemoteSDPByClientID[clientIDKEY] = true

	// Get Ice Candidate messages queue
	pendingIceCandidates := sc.pendingIceCandidatesByClientID[clientIDKEY]

	// If empty nothing to do
	if pendingIceCandidates == nil {
		return
	}

	// Clean Ice Candidate queue
	sc.pendingIceCandidatesByClientID[clientIDKEY] = nil

	// trigger Ice Candidate events, one by one
	for i := range pendingIceCandidates {
		sc.onIceCandidate(&pendingIceCandidates[i], clientID)
	}

}

// Sender signalingSdp Offer Messages
func (sc *Client) SendSdpOffer(sdpOfferMsg string, recipientClientID *string) {
	var clientID string

	// Assing client Id if exists
	if recipientClientID != nil {
		clientID = *recipientClientID
	}
	// Send Message
	sc.sendMessage(sdpOffer, sdpOfferMsg, clientID)
}

// Sender signaling Ice Candidate Messages
func (sc *Client) SendIceCandidate(iceCandidateMsg string, recipientClientID *string) {
	var clientID string

	// Assing client Id if exists
	if recipientClientID != nil {
		clientID = *recipientClientID
	}
	// Send Message
	sc.sendMessage(iceCandidate, iceCandidateMsg, clientID)
}

// Sender signaling Sdp Answer Messages
func (sc *Client) SendSdpAnswer(sdpAnswerMsg string, recipientClientID *string) {
	var clientID string

	// Assing client Id if exists
	if recipientClientID != nil {
		clientID = *recipientClientID
	}
	// Send Message
	sc.sendMessage(sdpAnswer, sdpAnswerMsg, clientID)
}

// Generic Sender signaling Messages
func (sc *Client) sendMessage(msgType MessageType, payload string, recipientClientID string) {

	// If signaling client status is different to OPEN, you can't send message
	if sc.readyState != open {
		err := errors.New("could not send message because the connection to the signaling service is not open")
		sc.onError(err)
		return
	}

	// Check if Recipient Client is valid
	if sc.validateRecipientClientID(&recipientClientID) {
		// Create WebSocket Signaling Message
		wsMessage := WebSocketSignalingMessageSend{
			MessageType:       msgType,
			MessagePayload:    b64.StdEncoding.EncodeToString([]byte(payload)),
			RecipientClientID: recipientClientID,
		}
		// Encode Message
		wsMessageBytes, _ := json.Marshal(wsMessage)

		// Send Message over websocket
		err := sc.wsClient.Send(TextMessage, wsMessageBytes)
		if err != nil {
			sc.onError(err)
			return
		}
	}

}

// Error if Recipient Client Id exists and actor is viewer
func (sc *Client) validateRecipientClientID(recipientClientID *string) bool {
	if sc.config.Role == Viewer && recipientClientID != nil && *recipientClientID != "" {
		err := errors.New("unexpected recipient client id. As the VIEWER, messages must not be sent with a recipient client id")
		sc.onError(err)
		return false
	}
	return true
}
