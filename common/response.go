package common

import (
	"errors"
)

// A response.
// See https://tools.ietf.org/html/rfc3501#section-2.2.2
type Resp struct {
	// The response tag. Can be either * for untagged responses, + for continuation
	// requests or a previous command's tag.
	Tag string
	// The parsed response fields.
	Fields []interface{}
}

// A continuation request.
type ContinuationResp struct {
	// The info message sent with the continuation request.
	Info string
}

func (r *ContinuationResp) Resp() *Resp {
	res := &Resp{Tag: "+"}

	if r.Info != "" {
		res.Fields = append(res.Fields, r.Info)
	}

	return res
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
