package v4

import (
	"encoding/hex"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/BeHumans/amazon-kinesis-video-streams-webrtc-sdk-go/signer"
	"github.com/aws/aws-sdk-go/aws/credentials"
)

const (
	// DefaultAlgorithm used for AWS V4 Signed
	DefaultAlgorithm = "AWS4-HMAC-SHA256"
)

// AWS V4 Signer
type Signer struct {
	Region      string
	Credentials *credentials.Credentials
	Service     string
}

// Get Signature from datestring, region and service using credentials
func (s *Signer) getSignatureKey(dateString string) []byte {
	cred, _ := s.Credentials.Get()
	date := signer.HMAC([]byte("AWS4"+cred.SecretAccessKey), dateString)
	region := signer.HMAC(date, s.Region)
	service := signer.HMAC(region, s.Service)
	return signer.HMAC(service, "aws4_request")
}

// Add credentials value as option
func WithCredentialsValue(value *credentials.Value) func(*Signer) {
	return func(sc *Signer) {
		// Create a static credential provider with input credentials value
		sc.Credentials = credentials.NewChainCredentials(
			[]credentials.Provider{
				&credentials.StaticProvider{
					Value: *value,
				},
			})
	}
}

// Add region value as option
func WithRegion(region string) func(*Signer) {
	return func(sc *Signer) {
		sc.Region = region
	}
}

// Add service value as option
func WithService(service string) func(*Signer) {
	return func(sc *Signer) {
		sc.Service = service
	}
}

// New Signer
func New(options ...func(*Signer)) (*Signer, error) {

	// Signed empty
	rs := &Signer{}

	// Add With Options
	for _, o := range options {
		o(rs)
	}

	// Check if it added credentials
	if rs.Credentials == nil {
		// Create Chain in order
		// 1.Env Provider
		// 2.Shared Credentials Provider
		rs.Credentials = credentials.NewChainCredentials(
			[]credentials.Provider{
				&credentials.EnvProvider{},
				&credentials.SharedCredentialsProvider{},
			})
	}

	// Return signer
	return rs, nil
}

// GetSignedURL function that sign input url, input query params and input date
func (s *Signer) GetSignedURL(endpoint string, queryParams signer.QueryParams, date *time.Time) (string, error) {
	// Get credentials to use
	cred, err := s.Credentials.Get()
	if err != nil {
		return "", errors.New("credentials for sign invalid because they are non-existent or expired")
	}

	// Get date now
	var now = time.Now()

	// if you don't give me the date, now is the date
	if date == nil {
		date = &now
	}

	// Prepare date strings
	datetimeString := getDateTimeString(date)
	dateString := getDateString(date)

	// Validate and parse endpoint
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", errors.New("Endpoint '" + endpoint + "' is not a valid uri.")
	}

	var protocol = "wss"
	var urlProtocol = protocol + "://"

	// Check protocol
	if u.Scheme != protocol {
		return "", errors.New("Endpoint '" + endpoint + "' is not a secure WebSocket endpoint. It should start with '" + urlProtocol + "'.")
	}

	// Check if it contains params
	if strings.Contains(endpoint, "?") {
		return "", errors.New("Endpoint '" + endpoint + "' should not contain any query parameters")
	}

	// Get host and path from url
	host := u.Host
	path := u.Path

	// Default path if nil
	if path == "" {
		path = "/"
	}

	// Add default host header
	signedHeaders := strings.Join([]string{"host"}, ";")

	// Prepare method
	method := "GET" // Method is always GET for signed URLs

	// Prepare canonical query string
	credentialScope := dateString + "/" + s.Region + "/" + s.Service + "/" + "aws4_request"

	canonicalQueryParams := signer.MergeMaps(queryParams, map[string]string{
		"X-Amz-Algorithm":     DefaultAlgorithm,
		"X-Amz-Credential":    cred.AccessKeyID + "/" + credentialScope,
		"X-Amz-Date":          datetimeString,
		"X-Amz-Expires":       "299",
		"X-Amz-SignedHeaders": signedHeaders,
	})

	// Add SessionToken as query params if it exists
	if cred.SessionToken != "" {
		canonicalQueryParams["X-Amz-Security-Token"] = cred.SessionToken
	}

	// Query params to query param string
	canonicalQueryString := signer.CreateQueryString(canonicalQueryParams)

	// Prepare canonical headers
	canonicalHeaders := map[string]string{"host": host}
	canonicalHeadersString := signer.CreateHeadersString(canonicalHeaders)

	// Prepare payload hash
	payloadHash := signer.SHA256("")

	// Combine canonical request parts into a canonical request string and hash
	canonicalRequest := strings.Join([]string{method, path, canonicalQueryString, canonicalHeadersString, signedHeaders, payloadHash}, "\n")
	canonicalRequestHash := signer.SHA256(canonicalRequest)

	// Create signature
	stringToSign := strings.Join([]string{DefaultAlgorithm, datetimeString, credentialScope, canonicalRequestHash}, "\n")
	signingKey := s.getSignatureKey(dateString)
	signature := signer.HMAC(signingKey, stringToSign)

	// Add signature to query params
	signedQueryParams := signer.MergeMaps(canonicalQueryParams, map[string]string{
		"X-Amz-Signature": hex.EncodeToString(signature),
	})

	// Create signed URL
	return protocol + "://" + host + path + "?" + signer.CreateQueryString(signedQueryParams), nil
}

// Get datetime as a valid string AWS V4 signature date format
func getDateTimeString(date *time.Time) string {
	dateString := date.UTC().Format(time.RFC3339)
	return strings.ReplaceAll(strings.ReplaceAll(dateString, ":", ""), "-", "")
}

// Get only date info, first 8 chars, from a valid string AWS V4 signature date format
func getDateString(date *time.Time) string {
	return getDateTimeString(date)[:8]
}
