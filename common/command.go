package common

import (
	"errors"
	"io"
	"strconv"
)

const (
	Capability string = "CAPABILITY"
	Noop = "NOOP"
	Logout = "LOGOUT"
	Starttls = "STARTTLS"
)

// A command.
type Command struct {
	Tag string
	Name string
	Arguments []interface{}
}

// Implements io.WriterTo interface.
func (c *Command) WriteTo(w io.Writer) (N int64, err error) {
	n, err := w.Write([]byte(c.Tag + " " + c.Name))
	if err != nil {
		return
	}
	N += n

	for i, argi := range c.Arguments {
		if argw, ok := argi.(io.WriterTo); ok {
			n, err := argw.WriteTo(w)
			if err != nil {
				return N, err
			}
			N += n
		}

		var b []byte
		switch arg := argi.(type) {
		case []byte:
			b = arg
		case string:
			b = []byte(arg)
		case int:
			b = []byte(strconv.Itoa(arg))
		default:
			return N, errors.New("Cannot format argument #" + strconv.Itoa(i))
		}

		n, err = w.Write([]byte(" "))
		if err != nil {
			return
		}
		N += n

		n, err = w.Write(b)
		if err != nil {
			return
		}
		N += n
	}

	return
}
