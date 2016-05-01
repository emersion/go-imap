package common

import (
	"errors"
	"io"
)

// A status response type.
type StatusRespType string

const (
	OK StatusRespType = "OK"
	NO = "NO"
	BAD = "BAD"
	PREAUTH = "PREAUTH"
	BYE = "BYE"
)

// A status response.
// See https://tools.ietf.org/html/rfc3501#section-7.1
type StatusResp struct {
	// The response tag.
	Tag string

	// The status type.
	Type StatusRespType

	// The status code.
	Code string

	// Arguments provided with the status code.
	Arguments []interface{}

	// The status info.
	Info string
}

// If this status is NO or BAD, returns an error with the status info.
// Otherwise, returns nil.
func (r *StatusResp) Err() error {
	if r.Type == NO || r.Type == BAD {
		return errors.New(r.Info)
	}
	return nil
}

// Implements io.WriterTo.
func (r *StatusResp) WriteTo(w io.Writer) (int64, error) {
	fields := []interface{}{r.Type}

	if r.Code != "" {
		code := r.Code

		if len(r.Arguments) > 0 {
			// TODO: convert Arguments to []string
			//code += " " + strings.Join(" ", r.Arguments)
		}

		fields = append(fields, "[" + code + "]")
	}

	if r.Info != "" {
		fields = append(fields, r.Info)
	}

	res := &Resp{
		Tag: r.Tag,
		Fields: fields,
	}

	return res.WriteTo(w)
}
