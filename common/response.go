package common

import (
	"errors"
	"io"
)

// A response.
// See https://tools.ietf.org/html/rfc3501#section-2.2.2
type Resp struct {
	Tag string
	Fields []interface{}
}

// Implements io.WriterTo interface.
func (r *Resp) WriteTo(w io.Writer) (N int64, err error) {
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

// A continuation response.
type ContinuationResp struct {
	Info string
}

func (r *ContinuationResp) Resp() *Resp {
	res := &Resp{Tag: "+"}

	if r.Info != "" {
		res.Fields = append(res.Fields, r.Info)
	}

	return res
}

func (r *ContinuationResp) WriteTo(w io.Writer) (int64, error) {
	return r.Resp().WriteTo(w)
}

func ReadResp(r *Reader) (out interface{}, err error) {
	atom, err := r.ReadAtom()
	if err != nil {
		return
	}
	tag, ok := atom.(string)
	if !ok {
		err = errors.New("Response tag is not an atom")
		return
	}

	if tag == "+" {
		res := &ContinuationResp{}
		res.Info, err = r.ReadInfo()
		if err != nil {
			return
		}

		out = res
		return
	}

	// Can be either data or status
	// Try to parse a status
	isStatus := false
	var fields []interface{}

	if atom, err = r.ReadAtom(); err == nil {
		fields = append(fields, atom)

		if name, ok := atom.(string); ok {
			status := StatusRespType(name)
			if status == OK || status == NO || status == BAD || status == PREAUTH || status == BYE {
				isStatus = true

				res := &StatusResp{
					Tag: tag,
					Type: status,
				}

				var char rune
				if char, _, err = r.ReadRune(); err != nil {
					return
				}
				r.UnreadRune()

				if char == '[' {
					// Contains code & arguments
					res.Code, res.Arguments, err = r.ReadRespCode()
					if err != nil {
						return
					}
				}

				res.Info, err = r.ReadInfo()
				if err != nil {
					return
				}

				out = res
			}
		}
	}

	if !isStatus {
		// Not a status so it's data
		res := &Resp{Tag: tag}

		var remaining []interface{}
		remaining, err = r.ReadLine()
		if err != nil {
			return
		}

		res.Fields = append(fields, remaining...)
		out = res
	}

	return
}
