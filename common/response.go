package common

import (
	"errors"
)

// A response.
// See RFC 3501 section 2.2.2
type Resp struct {
	// The response tag. Can be either * for untagged responses, + for continuation
	// requests or a previous command's tag.
	Tag string
	// The parsed response fields.
	Fields []interface{}
}

func (r *Resp) WriteTo(w *Writer) (err error) {
	fields := []interface{}{r.Tag}
	fields = append(fields, r.Fields...)

	if _, err = w.WriteFields(fields); err != nil {
		return
	}

	_, err = w.WriteCrlf()
	return
}

// Create a new untagged response.
func NewUntaggedResp(fields []interface{}) *Resp {
	return &Resp{
		Tag: "*",
		Fields: fields,
	}
}

// A continuation request.
type ContinuationResp struct {
	// The info message sent with the continuation request.
	Info string
}

func (r *ContinuationResp) WriteTo(w *Writer) (err error) {
	if _, err = w.WriteString("+"); err != nil {
		return
	}

	if r.Info != "" {
		if _, err = w.WriteSp(); err != nil {
			return
		}

		_, err = w.WriteInfo(r.Info)
	} else {
		_, err = w.WriteCrlf()
	}
	return
}

// A value that can be converted to a Resp.
type Responser interface {
	Response() *Resp
}

// Read a single response from a Reader. Returns either a continuation request,
// a status response or a raw response.
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
		if err := r.ReadSp(); err != nil {
			r.UnreadRune()
		}

		res := &ContinuationResp{}
		res.Info, err = r.ReadInfo()
		if err != nil {
			return
		}

		out = res
		return
	}

	if err = r.ReadSp(); err != nil {
		return
	}

	// Can be either data or status
	// Try to parse a status
	isStatus := false
	var fields []interface{}

	if atom, err = r.ReadAtom(); err == nil {
		fields = append(fields, atom)

		if err = r.ReadSp(); err == nil {
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
		} else {
			r.UnreadRune()
		}
	} else {
		r.UnreadRune()
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
