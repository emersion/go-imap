package internal

import (
	"encoding/base64"
)

// EncodeSASLInitialResp encodes a SASL initial response for the AUTHENTICATE
// command line. An empty response is encoded as "=" (RFC 4959).
func EncodeSASLInitialResp(b []byte) string {
	if len(b) == 0 {
		return "="
	} else {
		return base64.StdEncoding.EncodeToString(b)
	}
}

// DecodeSASLInitialResp decodes a SASL initial response from the AUTHENTICATE
// command line. "=" is decoded as an empty response (RFC 4959).
func DecodeSASLInitialResp(s string) ([]byte, error) {
	if s == "=" {
		// go-sasl treats nil as no challenge/response, so return a non-nil
		// empty byte slice
		return []byte{}, nil
	} else {
		return base64.StdEncoding.DecodeString(s)
	}
}

// DecodeSASLContinuation decodes a SASL challenge or response sent in a
// continuation. "=" is not valid there, but older versions of this library
// sent it for an empty payload, so it is still accepted as one.
func DecodeSASLContinuation(s string) ([]byte, error) {
	if s == "=" {
		return []byte{}, nil
	}
	return base64.StdEncoding.DecodeString(s)
}
