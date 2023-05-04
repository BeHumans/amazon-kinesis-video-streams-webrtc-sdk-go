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
	RecipientClientId string      `json:"recipientClientId,omitempty"`
}

// Signaling msg format for Reception
type WebSocketSignalingMessageReceive struct {
	MessageType    MessageType `json:"messageType,omitempty"`
	MessagePayload string      `json:"messagePayload,omitempty"`
	SenderClientId string      `json:"senderClientId,omitempty"`
}

// Default service for v4 signature
var service string = "kinesisvideo"

// Signaling configure
type Config struct {
	ChannelARN        *string            // ARN AWS Signaling Channel
	ChannelEndpoint   *string            // WSS URL AWS signaling
	Region            *string            // AWS Region
	Role              Role               // Master or viewer actor
	ClientId          *string            // Id Client for AWS signaling protocol
	SystemClockOffset int                // An offset value in milliseconds to apply to all signing times
	CredentialsValue  *credentials.Value // AWS Credentials
}

// Signaling Message Type
type MessageType string

const (
	SDP_ANSWER    MessageType = "SDP_ANSWER"
	SDP_OFFER     MessageType = "SDP_OFFER"
	ICE_CANDIDATE MessageType = "ICE_CANDIDATE"
)

// Signaling connection State Type
type ReadyStateType string

const (
	CONNECTING ReadyStateType = "CONNECTING"
	OPEN       ReadyStateType = "OPEN"
	CLOSING    ReadyStateType = "CLOSING"
	CLOSED     ReadyStateType = "CLOSED"
)

// Signaling client
type SignalingClient struct {
	readyState                     ReadyStateType                               // Signaling cient connection status
	config                         *Config                                      // Signaling client configuration
	signer                         signer.SignerAPI                             // V4 AWS Signer
	dateProvider                   *signer.DateProvier                          // Date provider for V4 AWS Signer
	wsClient                       WebSocketClientI                             // Websocket client
	onOpen                         func()                                       // Function for Open Event
	onError                        func(err error)                              // Function for Error Event
	onClose                        func()                                       // Function for Close Event
	onSdpAnswer                    func(answer *string, clientId *string)       // Function for Sdp Answer Event
	onSdpOffer                     func(offer *string, remoteClientId *string)  // Function for Sdp Offer Event
	onIceCandidate                 func(iceCandidate *string, clientId *string) // Function for Ice Candidate Event
	hasReceivedRemoteSDPByClientId map[string]bool                              // Maps for manage receive remote SDP by clientId
	pendingIceCandidatesByClientId map[string][]string                          // Maps for manage pending Ice Candidate by clientId
}

// On Open Event Function
func (sc *SignalingClient) OnOpen(f func()) {
	sc.onOpen = f
}

// On Close Event Function
func (sc *SignalingClient) OnClose(f func()) {
	sc.onClose = f
}

// On Error Event Function
func (sc *SignalingClient) OnError(f func(err error)) {
	sc.onError = f
}

// On Message Event Function
func (sc *SignalingClient) onMessage(data []byte) {

	var messageParsed WebSocketSignalingMessageReceive

	// New decoder
	dec := json.NewDecoder(bytes.NewReader(data))

	// Decode data, unkown message ?
	if err := dec.Decode(&messageParsed); err != nil {
		return
	}

	// Decode Base 64 payload message property
	decodedMessagePayload, err := b64.StdEncoding.DecodeString(messageParsed.MessagePayload)

	//error payload?
	if err != nil {
		return
	}

	// Decode message to string
	var messagePayloadParsed string = string(decodedMessagePayload)

	switch messageParsed.MessageType {
	// When receive a SDP Offer
	case SDP_OFFER:
		// Trigger on Sdp Offer Event
		sc.onSdpOffer(&messagePayloadParsed, &messageParsed.SenderClientId)
		sc.emitPendingIceCandidates(&messageParsed.SenderClientId)
		return
	// When receive a SDP Answer
	case SDP_ANSWER:
		// trigger on Sdp Answer Event
		sc.onSdpAnswer(&messagePayloadParsed, &messageParsed.SenderClientId)
		sc.emitPendingIceCandidates(&messageParsed.SenderClientId)
		return
	// When receive a Ice Candidate
	case ICE_CANDIDATE:
		sc.emitOrQueueIceCandidate(&messagePayloadParsed, &messageParsed.SenderClientId)
		return
	default:
		// Unkown message
		return
	}
}

// On Sdp Answer Event Function
func (sc *SignalingClient) OnSdpAnswer(f func(answer *string, clientId *string)) {
	sc.onSdpAnswer = f
}

// On Sdp Offer Event Function
func (sc *SignalingClient) OnSdpOffer(f func(offer *string, remoteClientId *string)) {
	sc.onSdpOffer = f
}

// On ICE Candidate Event Function
func (sc *SignalingClient) OnIceCandidate(f func(iceCandidate *string, clientId *string)) {
	sc.onIceCandidate = f
}

// Optional parameters

// Use own websocket client implementation
func WithWebsocketClient(websocket WebSocketClientI) func(*SignalingClient) {
	return func(sc *SignalingClient) {
		sc.wsClient = websocket
	}
}

// Use own v4 AWS signer implementation
func WithSigner(signer signer.SignerAPI) func(*SignalingClient) {
	return func(sc *SignalingClient) {
		sc.signer = signer
	}
}

// Use own Date Provider implementation
func WithDateProvider(dateProvider signer.DateProvier) func(*SignalingClient) {
	return func(sc *SignalingClient) {
		sc.dateProvider = &dateProvider
	}
}

// New signaling client
func New(config *Config, options ...func(*SignalingClient)) (*SignalingClient, error) {

	// Config must never be nil
	if config == nil {
		return nil, errors.New("Config cannot be nil")
	}

	// New Signaling client with initial values
	sc := &SignalingClient{
		readyState:                     CLOSED,
		config:                         config,
		hasReceivedRemoteSDPByClientId: make(map[string]bool),
		pendingIceCandidatesByClientId: make(map[string][]string),
	}

	// Getting other optional parameters
	for _, o := range options {
		o(sc)
	}

	// Nil Checkers

	// When Viewer actor
	if config.Role == VIEWER {
		// Config clientId must never be nil
		if config.ClientId == nil {
			return nil, errors.New("clientId cannot be nil")
		}
	}

	// When Master actor
	if config.Role == MASTER {
		// Config clientId always nil
		if config.ClientId != nil {
			return nil, errors.New("clientId should be nil when master selected")
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
		var kinesisVideoSigner signer.SignerAPI
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

		//if something wrong happened when sign
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
			ClockOffsetMs: time.Duration(config.SystemClockOffset),
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
func (sc *SignalingClient) Open() error {
	// Check if reOpen action
	if sc.readyState != CLOSED {
		err := errors.New("Client is already open, opening, or closing")
		if sc.onError != nil {
			// Trigger Error Event
			sc.onError(err)
		}
		return err
	}

	// Change signaling client state
	sc.readyState = CONNECTING

	// Go Rutine for connect to websocket signaling channel
	go func() {

		// Prepare data for sign request
		queryParams := signer.QueryParams{
			"X-Amz-ChannelARN": *sc.config.ChannelARN,
		}
		// If viewer actor
		if sc.config.Role == VIEWER {
			queryParams["X-Amz-ClientId"] = *sc.config.ClientId
		}

		// AWS V4 Sing channel endpoint uri
		signedURL, err := sc.signer.GetSignedURL(*sc.config.ChannelEndpoint, queryParams, sc.dateProvider.GetDate())

		//if something wrong happened
		if err != nil {
			// Trigger Error Event
			sc.onError(err)
		}

		// If other process changed signaling client status nothing to do
		if sc.readyState != CONNECTING {
			return
		}

		// Set signed url to websocket client
		err = sc.wsClient.SetUrl(signedURL)
		if err != nil {
			sc.onError(err)
		}

		// When websocket Open event do
		sc.wsClient.OnOpen(func() {
			sc.readyState = OPEN
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
			sc.readyState = CLOSED
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

		//if something wrong happened
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
func (sc *SignalingClient) Close() {
	// If websocket exists
	if sc.wsClient != nil {
		// Change signaling client status
		sc.readyState = CLOSING
		// Close websocket client
		sc.wsClient.Close()

	} else if sc.readyState != CLOSED {
		// trigger Close event
		sc.onClose()
	}
	// Nothing to do when websocket client does not exist and signaling client status is different than CLOSED
}

// Use for emit Ice Candidate Messages when signaling client has recive SDP message
func (sc *SignalingClient) emitOrQueueIceCandidate(iceCandidate *string, clientId *string) {
	var clientIdKey string

	// if you don't have client Id use Default Client Id
	if clientId == nil {
		clientIdKey = string(DEFAULT_CLIENT_ID)
	} else {
		clientIdKey = *clientId
	}

	// If signaling client has recive SDP message
	if sc.hasReceivedRemoteSDPByClientId[clientIdKey] {
		// trigger Ice Candidate Event
		sc.onIceCandidate(iceCandidate, clientId)
	} else {
		// If it is the first Ice Candidate message for this client Id
		if sc.pendingIceCandidatesByClientId == nil || sc.pendingIceCandidatesByClientId[clientIdKey] == nil {
			sc.pendingIceCandidatesByClientId[clientIdKey] = []string{}
		}
		// Queue Ice Candidate Message for this client Id
		sc.pendingIceCandidatesByClientId[clientIdKey] = append(sc.pendingIceCandidatesByClientId[clientIdKey], *iceCandidate)
	}

}

// Use for emit Ice Candidate Messages
func (sc *SignalingClient) emitPendingIceCandidates(clientId *string) {
	var clientIdKey string

	// if you don't have client Id use Default Client Id
	if clientId == nil {
		clientIdKey = string(DEFAULT_CLIENT_ID)
	} else {
		clientIdKey = *clientId
	}

	// Set Sdp message receive for this client id
	sc.hasReceivedRemoteSDPByClientId[clientIdKey] = true

	// Get Ice Candidate messages queue
	pendingIceCandidates := sc.pendingIceCandidatesByClientId[clientIdKey]

	// If empty nothing to do
	if pendingIceCandidates == nil {
		return
	}

	// Clean Ice Candidate queue
	sc.pendingIceCandidatesByClientId[clientIdKey] = nil

	// trigger Ice Candidate events, one by one
	for _, iceCandidate := range pendingIceCandidates {
		sc.onIceCandidate(&iceCandidate, clientId)
	}

}

// Sender signalingSdp Offer Messages
func (sc *SignalingClient) SendSdpOffer(offer string, recipientClientId *string) {
	var clientId string

	// Assing client Id if exists
	if recipientClientId != nil {
		clientId = *recipientClientId
	}
	// Send Message
	sc.sendMessage(SDP_OFFER, offer, clientId)
}

// Sender signaling Ice Candidate Messages
func (sc *SignalingClient) SendIceCandidate(iceCandidate string, recipientClientId *string) {
	var clientId string

	// Assing client Id if exists
	if recipientClientId != nil {
		clientId = *recipientClientId
	}
	// Send Message
	sc.sendMessage(ICE_CANDIDATE, iceCandidate, clientId)
}

// Sender signaling Sdp Answer Messages
func (sc *SignalingClient) SendSdpAnswer(sdpAnswer string, recipientClientId *string) {
	var clientId string

	// Assing client Id if exists
	if recipientClientId != nil {
		clientId = *recipientClientId
	}
	// Send Message
	sc.sendMessage(SDP_ANSWER, sdpAnswer, clientId)
}

// Generic Sender signaling Messages
func (sc *SignalingClient) sendMessage(msgType MessageType, payload string, recipientClientId string) {

	// If signaling client status is different to OPEN, you can't send message
	if sc.readyState != OPEN {
		err := errors.New("Could not send message because the connection to the signaling service is not open.")
		sc.onError(err)
		return
	}

	// Check if Recipient Client is valid
	if sc.validateRecipientClientId(&recipientClientId) {
		// Create WebSocket Signaling Message
		wsMessage := WebSocketSignalingMessageSend{
			MessageType:       msgType,
			MessagePayload:    b64.StdEncoding.EncodeToString([]byte(payload)),
			RecipientClientId: recipientClientId,
		}
		// Encode Message
		wsMessageBytes, _ := json.Marshal(wsMessage)

		// Send Message over websocket
		sc.wsClient.Send(TextMessage, wsMessageBytes)
	}

}

// Error if Recipient Client Id exists and actor is viewer
func (sc *SignalingClient) validateRecipientClientId(recipientClientId *string) bool {
	if sc.config.Role == VIEWER && recipientClientId != nil && *recipientClientId != "" {
		err := errors.New("Unexpected recipient client id. As the VIEWER, messages must not be sent with a recipient client id.")
		sc.onError(err)
		return false
	}
	return true
}
