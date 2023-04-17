package webrtcSignaling

import (
	cHMAC "crypto/hmac"
	cSHA256 "crypto/sha256"
	"encoding/hex"
	"errors"
	"net/url"
	"sort"
	"strings"
	"time"
)

const (
	DEFAULT_ALGORITHM = "AWS4-HMAC-SHA256"
)

type QueryParams map[string]string

type Credentials struct {
	accessKeyId     string
	secretAccessKey string
	sessionToken    string
}

type RequestSigner interface {
	getSignedURL(endpoint string, queryParams QueryParams, date *time.Time) (string, error)
}

type SigV4RequestSigner struct {
	region      string
	credentials Credentials
	service     string
}

func mergeMaps(m1 map[string]string, m2 map[string]string) map[string]string {
	merged := make(map[string]string)
	for k, v := range m1 {
		merged[k] = v
	}
	for key, value := range m2 {
		merged[key] = value
	}
	return merged
}

func getDateTimeString(date *time.Time) string {
	dateString := date.UTC().Format(time.RFC3339)
	return strings.ReplaceAll(strings.ReplaceAll(dateString, ":", ""), "-", "")
}
func getDateString(date *time.Time) string {
	return getDateTimeString(date)[:8]
}

func createQueryString(queryParams QueryParams) string {

	keys := make([]string, 0, len(queryParams))

	for k := range queryParams {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	sortedQueryParams := url.Values{}
	for _, k := range keys {
		sortedQueryParams.Add(k, queryParams[k])
	}

	return sortedQueryParams.Encode()
}

func createHeadersString(headers map[string]string) string {

	var headersString string = ""
	for k := range headers {
		headersString = headersString + k + ":" + headers[k] + "\n"
	}
	return headersString
}
func sha256(message string) string {
	h := cSHA256.New()
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}

func hmac(key []byte, message string) []byte {

	sig := cHMAC.New(cSHA256.New, key)
	sig.Write([]byte(message))
	return sig.Sum(nil)
}

func (s *SigV4RequestSigner) getSignatureKey(dateString string) []byte {
	kDate := hmac([]byte("AWS4"+s.credentials.secretAccessKey), dateString)
	kRegion := hmac(kDate, s.region)
	kService := hmac(kRegion, s.service)
	return hmac(kService, "aws4_request")
}

// NewService - our constructor function
func NewRequestSigner(region string, credentials Credentials, service *string) *SigV4RequestSigner {
	var defaultService string = "kinesisvideo"
	if service == nil {
		service = &defaultService
	}
	rs := &SigV4RequestSigner{
		region:      region,
		credentials: credentials,
		service:     *service,
	}
	return rs
}

func (s *SigV4RequestSigner) getSignedURL(endpoint string, queryParams QueryParams, date *time.Time) (string, error) {

	var now time.Time = time.Now()

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

	var protocol string = "wss"
	var urlProtocol string = protocol + "://"
	if u.Scheme != protocol {
		return "", errors.New("Endpoint '" + endpoint + "' is not a secure WebSocket endpoint. It should start with '" + urlProtocol + "'.")
	}
	if strings.Contains(endpoint, "?") {
		return "", errors.New("Endpoint '" + endpoint + "' should not contain any query parameters")
	}
	host := u.Host
	path := u.Path
	if path == "" {
		path = "/"
	}

	signedHeaders := strings.Join([]string{"host"}, ";")

	// Prepare method
	method := "GET" // Method is always GET for signed URLs

	// Prepare canonical query string
	credentialScope := dateString + "/" + s.region + "/" + s.service + "/" + "aws4_request"

	canonicalQueryParams := mergeMaps(queryParams, map[string]string{
		"X-Amz-Algorithm":     DEFAULT_ALGORITHM,
		"X-Amz-Credential":    s.credentials.accessKeyId + "/" + credentialScope,
		"X-Amz-Date":          datetimeString,
		"X-Amz-Expires":       "299",
		"X-Amz-SignedHeaders": signedHeaders,
	})

	if s.credentials.sessionToken != "" {
		canonicalQueryParams["X-Amz-Security-Token"] = s.credentials.sessionToken
	}
	canonicalQueryString := createQueryString(canonicalQueryParams)

	// Prepare canonical headers
	canonicalHeaders := map[string]string{"host": host}
	canonicalHeadersString := createHeadersString(canonicalHeaders)

	// Prepare payload hash
	payloadHash := sha256("")

	// Combine canonical request parts into a canonical request string and hash
	canonicalRequest := strings.Join([]string{method, path, canonicalQueryString, canonicalHeadersString, signedHeaders, payloadHash}, "\n")
	canonicalRequestHash := sha256(canonicalRequest)

	// Create signature
	stringToSign := strings.Join([]string{DEFAULT_ALGORITHM, datetimeString, credentialScope, canonicalRequestHash}, "\n")
	signingKey := s.getSignatureKey(dateString)
	signature := hmac(signingKey, stringToSign)

	// Add signature to query params

	signedQueryParams := mergeMaps(canonicalQueryParams, map[string]string{
		"X-Amz-Signature": hex.EncodeToString(signature),
	})

	// Create signed URL
	return protocol + "://" + host + path + "?" + createQueryString(signedQueryParams), nil
}
