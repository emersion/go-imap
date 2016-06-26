package common

import (
	"errors"
)

// A status response type.
type StatusRespType string

const (
	// The OK response indicates an information message from the server.  When
	// tagged, it indicates successful completion of the associated command.
	// The untagged form indicates an information-only message.
	OK StatusRespType = "OK"

	// The NO response indicates an operational error message from the
	// server.  When tagged, it indicates unsuccessful completion of the
	// associated command.  The untagged form indicates a warning; the
	// command can still complete successfully.
	NO = "NO"

	// The BAD response indicates an error message from the server.  When
	// tagged, it reports a protocol-level error in the client's command;
	// the tag indicates the command that caused the error.  The untagged
	// form indicates a protocol-level error for which the associated
	// command can not be determined; it can also indicate an internal
	// server failure.
	BAD = "BAD"

	// The PREAUTH response is always untagged, and is one of three
	// possible greetings at connection startup.  It indicates that the
	// connection has already been authenticated by external means; thus
	// no LOGIN command is needed.
	PREAUTH = "PREAUTH"

	// The BYE response is always untagged, and indicates that the server
	// is about to close the connection.
	BYE = "BYE"
)

// A status response.
// See RFC 3501 section 7.1
type StatusResp struct {
	// The response tag. If empty, it defaults to *.
	Tag string

	// The status type.
	Type StatusRespType

	// The status code.
	// See https://www.iana.org/assignments/imap-response-codes/imap-response-codes.xhtml
	Code string

	// Arguments provided with the status code.
	Arguments []interface{}

	// The status info.
	Info string
}

// If this status is NO or BAD, returns an error with the status info.
// Otherwise, returns nil.
func (r *StatusResp) Err() error {
	if r == nil {
		// No status response, connection closed before we get one
		return errors.New("Connection closed during command execution")
	}

	if r.Type == NO || r.Type == BAD {
		return errors.New(r.Info)
	}
	return nil
}

func (r *StatusResp) WriteTo(w *Writer) (err error) {
	tag := r.Tag
	if tag == "" {
		tag = "*"
	}

	if _, err = w.WriteFields([]interface{}{tag, string(r.Type)}); err != nil {
		return
	}

	if _, err = w.WriteSp(); err != nil {
		return
	}

	if r.Code != "" {
		if _, err = w.WriteRespCode(r.Code, r.Arguments); err != nil {
			return
		}

		if _, err = w.WriteSp(); err != nil {
			return
		}
	}

	_, err = w.WriteInfo(r.Info)
	return
}
