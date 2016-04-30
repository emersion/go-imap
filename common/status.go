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

// A status response code.
type StatusRespCode string

const (
	ALERT StatusRespCode = "ALERT"
	BADCHARSET = "BADCHARSET"
	CAPABILITY = "CAPABILITY"
	PARSE = "PARSE"
	PERMANENTFLAGS = "PERMANENTFLAGS"
	READ_ONLY = "READ-ONLY"
	READ_WRITE = "READ-WRITE"
	TRYCREATE = "TRYCREATE"
	UIDNEXT = "UIDNEXT"
	UIDVALIDITY = "UIDVALIDITY"
	UNSEEN = "UNSEEN"
)

// A status response.
// See https://tools.ietf.org/html/rfc3501#section-7.1
type StatusResp struct {
	Tag string
	Type StatusRespType
	Code StatusRespCode
	Arguments []interface{}
	Info string
}

func (r *StatusResp) Err() error {
	if r.Type == NO || r.Type == BAD {
		return errors.New(r.Info)
	}
	return nil
}

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
