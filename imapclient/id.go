package imapclient

import (
	"errors"
	"fmt"
	"strings"

	"github.com/emersion/go-imap/v2/internal/imapwire"
)

const (
	idMaxFieldValuePairs = 30
	idMaxFieldLength     = 30
	idMaxValueLength     = 1024
)

type IDCommand struct {
	commandBase
	data map[string]*string
	err  error
}

func (command *IDCommand) Wait() (map[string]*string, error) {
	if command.err != nil {
		return nil, command.err
	}
	err := command.wait()
	return command.data, err
}

func validateIDPayload(fields map[string]*string) error {
	if len(fields) > idMaxFieldValuePairs {
		return fmt.Errorf(
			"cannot send %d field-value pairs as only up to %d are allowed",
			len(fields),
			idMaxFieldValuePairs)
	}

	fieldsSeen := map[string]bool{}

	for field, value := range fields {
		if len(field) > idMaxFieldLength {
			return fmt.Errorf(
				"cannot send a %d-octet field as fields only up to %d octets are allowed",
				len(field),
				idMaxFieldLength)
		}
		if value != nil && len(*value) > idMaxValueLength {
			return fmt.Errorf(
				"cannot send a %d-octet value as values only up to %d octets are allowed",
				len(*value),
				idMaxValueLength)
		}

		fieldLowered := strings.ToLower(field)
		if fieldsSeen[fieldLowered] {
			return fmt.Errorf("cannot send field %q twice", field)
		}
		fieldsSeen[fieldLowered] = true
	}

	return nil
}

// ID sends an ID command.
//
// This command requires support for the ID extension (RFC 2971).
//
// Fields (map keys) are case-insensitive in both requests and responses.
// Fields' case in requests might not be preserved. Fields are brought to
// lower case in responses.
//
// NIL lists are presented as nil maps, empty lists are presented as empty maps,
// sent or received.
func (client *Client) ID(fields map[string]*string) *IDCommand {
	if err := validateIDPayload(fields); err != nil {
		// Remain consistent with *imap.Error being the only non-fatal error
		client.closeWithError(err)
		return &IDCommand{err: err}
	}

	command := &IDCommand{}
	encoder := client.beginCommand("ID", command)
	encoder.SP()

	if fields == nil {
		encoder.NIL()
		encoder.end()
		return command
	}

	list := encoder.BeginList()
	for field, value := range fields {
		list.Item().String(field)
		if value != nil {
			list.Item().String(*value)
		} else {
			list.Item().NIL()
		}
	}
	list.End()

	encoder.end()
	return command
}

// Attempt to read NIL for an optional value that can't be an atom and doesn't start with an
// ATOM-CHAR. If a non-NIL atom is scanned, an error is set on decoder.
func tryReadingNil(decoder *imapwire.Decoder) bool {
	var atom string
	return decoder.Atom(&atom) && decoder.Expect(atom == "NIL", "NIL")
}

func fieldReader(decoder *imapwire.Decoder, result map[string]*string) func() (string, error) {
	return func() (string, error) {
		field := ""
		if !decoder.ExpectString(&field) {
			return "", decoder.Err()
		}

		if len(field) > idMaxFieldLength {
			return "", fmt.Errorf(
				"illegal %d-octet field as fields only up to %d octets are allowed",
				len(field),
				idMaxFieldLength)
		}

		fieldLowered := strings.ToLower(field)
		if _, found := result[fieldLowered]; found {
			return "", fmt.Errorf("received field %q twice", field)
		}

		if len(result) == idMaxFieldValuePairs {
			return "", errors.New("too many field-value pairs")
		}

		return fieldLowered, nil
	}
}

func valueReader(decoder *imapwire.Decoder, result map[string]*string) func(string) error {
	return func(key string) error {
		if tryReadingNil(decoder) {
			result[key] = nil
			return nil
		}

		value := ""
		if !decoder.ExpectString(&value) {
			return decoder.Err()
		}

		if len(value) > idMaxValueLength {
			return fmt.Errorf(
				"illegal %d-octet value as values only up to %d octets are allowed",
				len(value),
				idMaxValueLength)
		}

		result[key] = &value
		return nil
	}
}

func expectMap(decoder *imapwire.Decoder, onKey func() error, onValue func() error) error {
	seenKey := false

	err := decoder.ExpectList(func() error {
		if seenKey {
			seenKey = false
			return onValue()
		}
		seenKey = true
		return onKey()
	})
	if err != nil {
		return err
	}

	if seenKey {
		return errors.New("uneven number of elements in list")
	}
	return nil
}

func readIDResponse(decoder *imapwire.Decoder) (map[string]*string, error) {
	if !decoder.ExpectSP() {
		return nil, decoder.Err()
	}
	if tryReadingNil(decoder) {
		return nil, nil
	}
	if decoder.Err() != nil {
		return nil, decoder.Err()
	}

	result := map[string]*string{}
	readField := fieldReader(decoder, result)
	readValue := valueReader(decoder, result)
	field := ""

	err := expectMap(decoder,
		func() (err error) {
			field, err = readField()
			return
		},
		func() error {
			return readValue(field)
		})

	return result, err
}

func (client *Client) handleID() error {
	response, err := readIDResponse(client.dec)
	if err != nil {
		return fmt.Errorf("in id: %v", err)
	}

	if cmd := findPendingCmdByType[*IDCommand](client); cmd != nil {
		cmd.data = response
	}

	return nil
}
