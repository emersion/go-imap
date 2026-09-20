package imapserver

import (
	"fmt"
	"strings"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func (c *Conn) handleID(tag string, dec *imapwire.Decoder) error {
	idData, err := readID(dec)
	if err != nil {
		return fmt.Errorf("in id: %v", err)
	}

	if !dec.ExpectCRLF() {
		return dec.Err()
	}

	var serverIDData *imap.IDData
	if idSess, ok := c.session.(SessionID); ok {
		serverIDData = idSess.ID(idData)
	}

	enc := newResponseEncoder(c)
	enc.Atom("*").SP().Atom("ID")

	if serverIDData == nil {
		enc.SP().NIL()
	} else {
		enc.SP().Special('(')
		isFirstKey := true
		if serverIDData.Name != "" {
			addIDKeyValue(enc.Encoder, &isFirstKey, "name", serverIDData.Name)
		}
		if serverIDData.Version != "" {
			addIDKeyValue(enc.Encoder, &isFirstKey, "version", serverIDData.Version)
		}
		if serverIDData.OS != "" {
			addIDKeyValue(enc.Encoder, &isFirstKey, "os", serverIDData.OS)
		}
		if serverIDData.OSVersion != "" {
			addIDKeyValue(enc.Encoder, &isFirstKey, "os-version", serverIDData.OSVersion)
		}
		if serverIDData.Vendor != "" {
			addIDKeyValue(enc.Encoder, &isFirstKey, "vendor", serverIDData.Vendor)
		}
		if serverIDData.SupportURL != "" {
			addIDKeyValue(enc.Encoder, &isFirstKey, "support-url", serverIDData.SupportURL)
		}
		if serverIDData.Address != "" {
			addIDKeyValue(enc.Encoder, &isFirstKey, "address", serverIDData.Address)
		}
		if serverIDData.Date != "" {
			addIDKeyValue(enc.Encoder, &isFirstKey, "date", serverIDData.Date)
		}
		if serverIDData.Command != "" {
			addIDKeyValue(enc.Encoder, &isFirstKey, "command", serverIDData.Command)
		}
		if serverIDData.Arguments != "" {
			addIDKeyValue(enc.Encoder, &isFirstKey, "arguments", serverIDData.Arguments)
		}
		if serverIDData.Environment != "" {
			addIDKeyValue(enc.Encoder, &isFirstKey, "environment", serverIDData.Environment)
		}
		if serverIDData.Raw != nil {
			stdKeys := map[string]struct{}{
				"name": {}, "version": {}, "os": {}, "os-version": {}, "vendor": {},
				"support-url": {}, "address": {}, "date": {}, "command": {},
				"arguments": {}, "environment": {},
			}
			for k, v := range serverIDData.Raw {
				if _, ok := stdKeys[strings.ToLower(k)]; !ok {
					addIDKeyValue(enc.Encoder, &isFirstKey, k, v)
				}
			}
		}
		enc.Special(')')
	}

	err = enc.CRLF()
	enc.end()
	if err != nil {
		return err
	}

	return c.writeStatusResp(tag, &imap.StatusResponse{
		Type: imap.StatusResponseTypeOK,
		Text: "ID completed",
	})
}

func readID(dec *imapwire.Decoder) (*imap.IDData, error) {
	if !dec.ExpectSP() {
		return nil, dec.Err()
	}

	if dec.ExpectNIL() {
		return nil, nil
	}

	data := &imap.IDData{
		Raw: make(map[string]string),
	}
	currKey := ""
	err := dec.ExpectList(func() error {
		var keyOrValue string
		if !dec.String(&keyOrValue) {
			return fmt.Errorf("in id key-val list: %v", dec.Err())
		}

		if currKey == "" {
			currKey = keyOrValue
			return nil
		}

		lowerKey := strings.ToLower(currKey)
		data.Raw[lowerKey] = keyOrValue

		switch lowerKey {
		case "name":
			data.Name = keyOrValue
		case "version":
			data.Version = keyOrValue
		case "os":
			data.OS = keyOrValue
		case "os-version":
			data.OSVersion = keyOrValue
		case "vendor":
			data.Vendor = keyOrValue
		case "support-url":
			data.SupportURL = keyOrValue
		case "address":
			data.Address = keyOrValue
		case "date":
			data.Date = keyOrValue
		case "command":
			data.Command = keyOrValue
		case "arguments":
			data.Arguments = keyOrValue
		case "environment":
			data.Environment = keyOrValue
		default:
			// Unknown key, already stored in Raw
		}
		currKey = ""

		return nil
	})

	if err != nil {
		return nil, err
	}

	return data, nil
}

func addIDKeyValue(enc *imapwire.Encoder, isFirstKey *bool, key, value string) {
	if *isFirstKey {
		enc.Quoted(key).SP().Quoted(value)
	} else {
		enc.SP().Quoted(key).SP().Quoted(value)
	}
	*isFirstKey = false
}

// SessionID is an interface for sessions that can provide server ID information.
type SessionID interface {
	// ID returns server information in response to a client ID command.
	// The client's ID information is provided if available.
	ID(clientID *imap.IDData) *imap.IDData
}
