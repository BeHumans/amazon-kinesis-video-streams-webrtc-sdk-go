package webrtcSignaling

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var region string
var date time.Time
var queryParams map[string]string
var credetials Credentials
var signer SigV4RequestSigner

func IntInfo() {
	region = "us-west-2"
	queryParams = map[string]string{
		"X-Amz-TestParam": "test-param-value",
	}
	date, _ = time.Parse("2006-01-02", "2019-12-01")

	credetials = Credentials{
		accessKeyId:     "AKIA4F7WJQR7FMMWMNXI",
		secretAccessKey: "FakeSecretKey",
		sessionToken:    "FakeSessionToken",
	}

	signer = SigV4RequestSigner{
		region:      "us-west-2",
		credentials: credetials,
		service:     "kinesisvideo",
	}
}

var expectedSignedURL = "wss://kvs.awsamazon.com/?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=AKIA4F7WJQR7FMMWMNXI%2F20191201%2Fus-west-2%2Fkinesisvideo%2Faws4_request&X-Amz-Date=20191201T000000Z&X-Amz-Expires=299&X-Amz-Security-Token=FakeSessionToken&X-Amz-Signature=fc268038be276315822b4f73eafd28ee3a5632a2a35fdb0a88db9a42b13d6c92&X-Amz-SignedHeaders=host&X-Amz-TestParam=test-param-value"
var expectedSignedURLFirehouse = "wss://kvs.awsamazon.com/?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=AKIA4F7WJQR7FMMWMNXI%2F20191201%2Fus-west-2%2Ffirehose%2Faws4_request&X-Amz-Date=20191201T000000Z&X-Amz-Expires=299&X-Amz-Security-Token=FakeSessionToken&X-Amz-Signature=f15308513d21a381d38b7607a0439f25fc2e6c9f5ff56a48c1664b486e6234d5&X-Amz-SignedHeaders=host&X-Amz-TestParam=test-param-value"
var expectedSignedURLWithPath = "wss://kvs.awsamazon.com/path/path/path?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=AKIA4F7WJQR7FMMWMNXI%2F20191201%2Fus-west-2%2Fkinesisvideo%2Faws4_request&X-Amz-Date=20191201T000000Z&X-Amz-Expires=299&X-Amz-Security-Token=FakeSessionToken&X-Amz-Signature=0bf3df6ca23d8d82f688e8dbfb90d69e74843d40038541b1721c545eef7612a4&X-Amz-SignedHeaders=host&X-Amz-TestParam=test-param-value"

func TestInvalidEndpoint(t *testing.T) {
	IntInfo()
	_, err := signer.getSignedURL("https://kvs.awsamazon.com", queryParams, &date)
	expectedErrorMsg := "Endpoint 'https://kvs.awsamazon.com' is not a secure WebSocket endpoint. It should start with 'wss://'."
	assert.EqualErrorf(t, err, expectedErrorMsg, "Error should be: %v, got: %v", expectedErrorMsg, err)
}
func TestInvalidEndpointQueryParams(t *testing.T) {
	IntInfo()
	_, err := signer.getSignedURL("wss://kvs.awsamazon.com?a=b", queryParams, &date)
	expectedErrorMsg := "Endpoint 'wss://kvs.awsamazon.com?a=b' should not contain any query parameters"
	assert.EqualErrorf(t, err, expectedErrorMsg, "Error should be: %v, got: %v", expectedErrorMsg, err)
}
func TestValidSigned(t *testing.T) {
	IntInfo()
	url, _ := signer.getSignedURL("wss://kvs.awsamazon.com", queryParams, &date)
	assert.Equal(t, url, expectedSignedURL)
}

func TestValidSignedDynamicCredentials(t *testing.T) {
	IntInfo()
	ownCredentials := Credentials{
		accessKeyId:     "AKIA4F7WJQR7FMMWMNXI",
		secretAccessKey: "FakeSecretKey",
		sessionToken:    "FakeSessionToken",
	}

	signer := NewRequestSigner(region, ownCredentials, nil)

	url, _ := signer.getSignedURL("wss://kvs.awsamazon.com", queryParams, &date)
	assert.Equal(t, url, expectedSignedURL)
}
func TestValidSignedWithoutSessionToken(t *testing.T) {
	IntInfo()
	credetials.sessionToken = ""
	url, _ := signer.getSignedURL("wss://kvs.awsamazon.com", queryParams, &date)
	assert.Equal(t, url, expectedSignedURL)
}

func TestValidSignedServiceOverride(t *testing.T) {
	IntInfo()
	service := "firehose"

	signer := NewRequestSigner(region, credetials, &service)

	url, _ := signer.getSignedURL("wss://kvs.awsamazon.com", queryParams, &date)
	assert.Equal(t, url, expectedSignedURLFirehouse)
}
func TestValidSignedWithPath(t *testing.T) {
	IntInfo()
	url, _ := signer.getSignedURL("wss://kvs.awsamazon.com/path/path/path", queryParams, &date)
	assert.Equal(t, url, expectedSignedURLWithPath)
}
func TestValidSignedWithoutMockedDate(t *testing.T) {
	IntInfo()
	_, err := signer.getSignedURL("wss://kvs.awsamazon.com", queryParams, &date)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}
