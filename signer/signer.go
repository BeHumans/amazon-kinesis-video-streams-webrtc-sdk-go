package signer

import (
	"time"
)

// Query Params from uri
type QueryParams map[string]string

// Signer API provides an interface to enable AWS Signature
type SignerAPI interface {
	// Return an enpoint url signed
	GetSignedURL(endpoint string, queryParams QueryParams, date *time.Time) (string, error)
}
