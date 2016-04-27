package common

import (
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
	N += n

	if len(c.Fields) > 0 {
		var fields string
		fields, err = formatList(r.Fields)
		if err != nil {
			return
		}

		n, err = w.Write([]byte(" " + fields))
		if err != nil {
			return
		}
		N += n
	}

	return
}

// A continuation response.
type ContinuationResp struct {
	Info string
}

func (r *ContinuationResp) WriteTo(w io.Writer) (int64, error) {
	res := &Response{Tag: "+"}

	if r.Info != "" {
		res.Fields = append(res.Fields, r.Info)
	}

	return res.WriteTo(w)
}
