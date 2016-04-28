package common

import (
	"bufio"
	"errors"
	"io"
)

// A response.
// See https://tools.ietf.org/html/rfc3501#section-2.2.2
type Response struct {
	Tag string
	Fields []interface{}
}

// Implements io.WriterTo interface.
func (r *Response) WriteTo(w io.Writer) (N int64, err error) {
	n, err := w.Write([]byte(r.Tag))
	if err != nil {
		return
	}
	N += int64(n)

	if len(r.Fields) > 0 {
		var fields string
		fields, err = formatList(r.Fields)
		if err != nil {
			return
		}

		n, err = w.Write([]byte(" " + fields))
		if err != nil {
			return
		}
		N += int64(n)
	}

	return
}

// Implements io.ReaderFrom interface.
func (r *Response) ReadFrom(rd io.Reader) (n int64, err error) {
	br := bufio.NewReader(rd)
	// TODO: set n

	fields, err := parseLine(br)
	if err != nil {
		return
	}

	if len(fields) == 0 {
		err = errors.New("Cannot read response: line has no fields")
		return
	}

	r.Tag = fields[0].(string)
	r.Fields = fields[1:]
	return
}

// A continuation response.
type ContinuationResp struct {
	Info string
}

func (r *ContinuationResp) Response() *Response {
	res := &Response{Tag: "+"}

	if r.Info != "" {
		res.Fields = append(res.Fields, r.Info)
	}

	return res
}

func (r *ContinuationResp) WriteTo(w io.Writer) (int64, error) {
	return r.Response().WriteTo(w)
}

func ParseContinuationResp(res *Response) *ContinuationResp {
	cont := &ContinuationResp{}

	if len(res.Fields) > 0 {
		cont.Info = res.Fields[0].(string)
	}

	return cont
}

func readResp(r io.Reader) (out interface{}, size int, err error) {
	res := &Response{}

	n, err := res.ReadFrom(r)
	size = int(n)
	if err != nil {
		return
	}

	switch res.Tag {
	case "+":
		out = ParseContinuationResp(res)
	case "*":
		out = res
	default:
		// TODO: can be a generic response too? Check if name is a StatusRespType
		out = ParseStatusResp(res)
	}
	return
}
