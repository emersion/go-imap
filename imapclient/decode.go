package imapclient

import (
	"fmt"

	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func readCapabilities(dec *imapwire.Decoder) (map[string]struct{}, error) {
	caps := make(map[string]struct{})
	for dec.SP() {
		var name string
		if !dec.ExpectAtom(&name) {
			return caps, fmt.Errorf("in capability-data: %v", dec.Err())
		}
		caps[name] = struct{}{}
	}
	return caps, nil
}
