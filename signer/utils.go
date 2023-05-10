package signer

import (
	cHMAC "crypto/hmac"
	cSHA256 "crypto/sha256"
	"encoding/hex"
	"net/url"
	"sort"
)

// Util for merge two maps map[string]string into one map[string]string
func MergeMaps(m1 map[string]string, m2 map[string]string) map[string]string {
	merged := make(map[string]string)
	for k, v := range m1 {
		merged[k] = v
	}
	for key, value := range m2 {
		merged[key] = value
	}
	return merged
}

// Apply SHA256 to an input text
func SHA256(text string) string {
	h := cSHA256.New()
	h.Write([]byte(text))
	return hex.EncodeToString(h.Sum(nil))
}

// Apply HMAC to an input text with an input key
func HMAC(key []byte, text string) []byte {
	sig := cHMAC.New(cSHA256.New, key)
	sig.Write([]byte(text))
	return sig.Sum(nil)
}

// Build from headers map the string with headers line by line
func CreateHeadersString(headers map[string]string) string {
	var headersString = ""
	for k := range headers {
		headersString = headersString + k + ":" + headers[k] + "\n"
	}
	return headersString
}

// Build from QueryParams the URI encoded string
func CreateQueryString(queryParams QueryParams) string {
	keys := make([]string, 0, len(queryParams))

	for k := range queryParams {
		keys = append(keys, k)
	}

	// If the same data is not ordered, it gives a different signature!!!
	sort.Strings(keys)

	sortedQueryParams := url.Values{}
	for _, k := range keys {
		sortedQueryParams.Add(k, queryParams[k])
	}

	return sortedQueryParams.Encode()
}
