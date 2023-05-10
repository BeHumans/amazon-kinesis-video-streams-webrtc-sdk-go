package v4_test

import (
	"os"
	"testing"
	"time"

	"github.com/BeHumans/amazon-kinesis-video-streams-webrtc-sdk-go/signer"
	signerV4 "github.com/BeHumans/amazon-kinesis-video-streams-webrtc-sdk-go/signer/v4"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/stretchr/testify/assert"
)

var region string
var date time.Time
var queryParams map[string]string
var credetialsValue credentials.Value
var testSigner signer.APII
var service string

// Expected Signed URLs
var expectedSignedURL = "wss://kvs.awsamazon.com/?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=AKIA4F7WJQR7FMMWMNXI%2F20191201%2Fus-west-2%2Fkinesisvideo%2Faws4_request&X-Amz-Date=20191201T000000Z&X-Amz-Expires=299&X-Amz-Security-Token=FakeSessionToken&X-Amz-Signature=fc268038be276315822b4f73eafd28ee3a5632a2a35fdb0a88db9a42b13d6c92&X-Amz-SignedHeaders=host&X-Amz-TestParam=test-param-value"
var expectedSignedURLFirehouse = "wss://kvs.awsamazon.com/?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=AKIA4F7WJQR7FMMWMNXI%2F20191201%2Fus-west-2%2Ffirehose%2Faws4_request&X-Amz-Date=20191201T000000Z&X-Amz-Expires=299&X-Amz-Security-Token=FakeSessionToken&X-Amz-Signature=f15308513d21a381d38b7607a0439f25fc2e6c9f5ff56a48c1664b486e6234d5&X-Amz-SignedHeaders=host&X-Amz-TestParam=test-param-value"
var expectedSignedURLWithPath = "wss://kvs.awsamazon.com/path/path/path?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=AKIA4F7WJQR7FMMWMNXI%2F20191201%2Fus-west-2%2Fkinesisvideo%2Faws4_request&X-Amz-Date=20191201T000000Z&X-Amz-Expires=299&X-Amz-Security-Token=FakeSessionToken&X-Amz-Signature=0bf3df6ca23d8d82f688e8dbfb90d69e74843d40038541b1721c545eef7612a4&X-Amz-SignedHeaders=host&X-Amz-TestParam=test-param-value"

// Necessary info for each test
func InitInfo() {
	region = "us-west-2"
	queryParams = map[string]string{
		"X-Amz-TestParam": "test-param-value",
	}
	date, _ = time.Parse("2006-01-02", "2019-12-01")

	credetialsValue = credentials.Value{
		AccessKeyID:     "AKIA4F7WJQR7FMMWMNXI",
		SecretAccessKey: "FakeSecretKey",
		SessionToken:    "FakeSessionToken",
	}
	service = "kinesisvideo"
	testSigner, _ = signerV4.New(
		signerV4.WithRegion(region), signerV4.WithCredentialsValue(&credetialsValue),
		signerV4.WithService(service))

}

// Check if it uses correctly ENV AWS Credentials if exists
func TestWithENVCredentials(t *testing.T) {
	// Load Initial values
	InitInfo()

	// Set ENV AWS ACCESS and SECRET values
	os.Setenv("AWS_ACCESS_KEY_ID", "AWS_ACCESS_KEY_ID_FAKE")
	os.Setenv("AWS_ACCESS_KEY", "AWS_ACCESS_KEY_FAKE")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "AWS_SECRET_ACCESS_KEY_FAKE")
	os.Setenv("AWS_SECRET_KEY", "AWS_SECRET_KEY_FAKE")

	// New signer without input credentials
	ownTestSigner, err := signerV4.New(
		signerV4.WithRegion(region),
		signerV4.WithService(service))

	// if err something wrong
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Get credentials value from signer
	value, err := ownTestSigner.Credentials.Get()

	// if err something wrong
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// ASSERTS
	assert.Equal(t, value.AccessKeyID, "AWS_ACCESS_KEY_ID_FAKE")
	assert.Equal(t, value.SecretAccessKey, "AWS_SECRET_ACCESS_KEY_FAKE")
	assert.Equal(t, value.ProviderName, "EnvProvider")
}

// Check if it uses .aws/credentials when exists and not exists ENV AWS Credentials
func TestWithSharedCredentials(t *testing.T) {
	// Load Initial values
	InitInfo()

	// Unset ENV AWS ACCESS and SECRET values for use Shared Credentials Provider
	os.Unsetenv("AWS_ACCESS_KEY_ID")
	os.Unsetenv("AWS_ACCESS_KEY")
	os.Unsetenv("AWS_SECRET_ACCESS_KEY")
	os.Unsetenv("AWS_SECRET_KEY")

	// New signer without input credentials
	ownTestSigner, err := signerV4.New(
		signerV4.WithRegion(region),
		signerV4.WithService(service))

	// if err something wrong
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Get credentials value from signer
	value, err := ownTestSigner.Credentials.Get()

	// if err verify if error msg if is because .aws/credentials not exist or it is nor valid
	if err != nil {
		expectedErrorMsg := "NoCredentialProviders: no valid providers in chain. Deprecated.\n\tFor verbose messaging see aws.Config.CredentialsChainVerboseErrors"
		assert.EqualErrorf(t, err, expectedErrorMsg, "Error should be: %v, got: %v", expectedErrorMsg, err)
	} else {
		// .aws/credentials exist
		assert.Equal(t, value.ProviderName, "SharedCredentialsProvider")
	}

}

// Check if error exists when endpoint is not wss protocol
func TestInvalidEndpoint(t *testing.T) {
	// Load Initial values
	InitInfo()

	// Check if error exists calling Get signed URL
	_, err := testSigner.GetSignedURL("https://kvs.awsamazon.com", queryParams, &date)
	expectedErrorMsg := "Endpoint 'https://kvs.awsamazon.com' is not a secure WebSocket endpoint. It should start with 'wss://'."
	assert.EqualErrorf(t, err, expectedErrorMsg, "Error should be: %v, got: %v", expectedErrorMsg, err)
}

// Check if error exists when endpoint contains query params
func TestInvalidEndpointQueryParams(t *testing.T) {
	// Load Initial values
	InitInfo()

	// Check if error exists calling Get signed URL
	_, err := testSigner.GetSignedURL("wss://kvs.awsamazon.com?a=b", queryParams, &date)
	expectedErrorMsg := "Endpoint 'wss://kvs.awsamazon.com?a=b' should not contain any query parameters"
	assert.EqualErrorf(t, err, expectedErrorMsg, "Error should be: %v, got: %v", expectedErrorMsg, err)
}

// Check if it is signed and is valid expected signed url
func TestValidSigned(t *testing.T) {
	// Load Initial values
	InitInfo()

	// Check if error exists calling Get signed URL
	url, err := testSigner.GetSignedURL("wss://kvs.awsamazon.com", queryParams, &date)

	// if err something wrong
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// ASSERTS
	assert.Equal(t, url, expectedSignedURL)
}

// Check if it is signed with input credentials into signer
func TestValidSignedDynamicCredentials(t *testing.T) {
	// Load Initial values
	InitInfo()

	// New signer with input credentials
	testOwnSigner, err := signerV4.New(
		signerV4.WithRegion(region),
		signerV4.WithCredentialsValue(&credetialsValue),
		signerV4.WithService(service))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Calling Get Signed URL with correct input parameters
	url, err := testOwnSigner.GetSignedURL("wss://kvs.awsamazon.com", queryParams, &date)
	// if something wrong happened
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// ASSERTS
	assert.Equal(t, url, expectedSignedURL)
}

// Check if it is signed without SessionToken value
func TestValidSignedWithoutSessionToken(t *testing.T) {
	// Load Initial values
	InitInfo()

	// Delete Session Token
	credetialsValue.SessionToken = ""

	// Calling Get Signed URL with correct input parameters
	url, err := testSigner.GetSignedURL("wss://kvs.awsamazon.com", queryParams, &date)
	// if something wrong happened
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// ASSERTS
	assert.Equal(t, url, expectedSignedURL)
}

// Check if it is signed for other service value
func TestValidSignedServiceOverride(t *testing.T) {
	// Load Initial values
	InitInfo()

	// Select Firehose service
	service := "firehose"

	// New signer with input credentials
	testOwnSigner, err := signerV4.New(
		signerV4.WithRegion(region),
		signerV4.WithCredentialsValue(&credetialsValue),
		signerV4.WithService(service))
	// if something wrong happened
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	// Calling Get Signed URL with correct input parameters
	url, err := testOwnSigner.GetSignedURL("wss://kvs.awsamazon.com", queryParams, &date)
	// if something wrong happened
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// ASSERTS
	assert.Equal(t, url, expectedSignedURLFirehouse)
}

// Check if it is signed with endpoint large path
func TestValidSignedWithPath(t *testing.T) {
	// Load Initial values
	InitInfo()

	// Calling Get Signed URL with correct input parameters
	url, err := testSigner.GetSignedURL("wss://kvs.awsamazon.com/path/path/path", queryParams, &date)

	// if something wrong happened
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// ASSERTS
	assert.Equal(t, url, expectedSignedURLWithPath)
}

// Check if it is signed with mockDate
func TestValidSignedWithoutMockedDate(t *testing.T) {
	// Load Initial values
	InitInfo()

	// Calling Get Signed URL with correct input parameters
	_, err := testSigner.GetSignedURL("wss://kvs.awsamazon.com", queryParams, &date)
	// if something wrong happened
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

}
