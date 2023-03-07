package imapwire

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"unicode"
)

type Decoder struct {
	r   *bufio.Reader
	err error
}

func NewDecoder(r *bufio.Reader) *Decoder {
	return &Decoder{r: r}
}

func (dec *Decoder) mustUnreadByte() {
	if err := dec.r.UnreadByte(); err != nil {
		panic(fmt.Errorf("imapwire: failed to unread byte: %v", err))
	}
}

func (dec *Decoder) Err() error {
	return dec.err
}

func (dec *Decoder) returnErr(err error) bool {
	if err == nil {
		return true
	}
	if dec.err == nil {
		dec.err = err
	}
	return false
}

func (dec *Decoder) readByte() (byte, bool) {
	b, err := dec.r.ReadByte()
	if err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return b, dec.returnErr(err)
	}
	return b, true
}

func (dec *Decoder) acceptByte(want byte) bool {
	got, ok := dec.readByte()
	if !ok {
		return false
	} else if got != want {
		dec.mustUnreadByte()
		return false
	}
	return true
}

func (dec *Decoder) EOF() bool {
	_, err := dec.r.ReadByte()
	if err == io.EOF {
		return true
	} else if err != nil {
		return dec.returnErr(err)
	}
	dec.mustUnreadByte()
	return false
}

func (dec *Decoder) Expect(ok bool, name string) bool {
	if !ok {
		return dec.returnErr(fmt.Errorf("expected %v", name))
	}
	return true
}

func (dec *Decoder) SP() bool {
	return dec.acceptByte(' ')
}

func (dec *Decoder) ExpectSP() bool {
	return dec.Expect(dec.SP(), "SP")
}

func (dec *Decoder) CRLF() bool {
	return dec.acceptByte('\r') && dec.acceptByte('\n')
}

func (dec *Decoder) ExpectCRLF() bool {
	return dec.Expect(dec.CRLF(), "CRLF")
}

func (dec *Decoder) Atom(ptr *string) bool {
	var sb strings.Builder
	for {
		b, ok := dec.readByte()
		if !ok {
			return false
		}

		var valid bool
		switch b {
		case '(', ')', '{', ' ', '%', '*', '"', '\\', ']':
			valid = false
		default:
			valid = !unicode.IsControl(rune(b))
		}
		if !valid {
			dec.mustUnreadByte()
			break
		}

		sb.WriteByte(b)
	}
	if sb.Len() == 0 {
		return false
	}
	*ptr = sb.String()
	return true
}

func (dec *Decoder) ExpectAtom(ptr *string) bool {
	return dec.Expect(dec.Atom(ptr), "atom")
}

func (dec *Decoder) Special(b byte) bool {
	return dec.acceptByte(b)
}

func (dec *Decoder) ExpectSpecial(b byte) bool {
	return dec.Expect(dec.Special(b), fmt.Sprintf("'%v'", string(b)))
}

func (dec *Decoder) Text(ptr *string) bool {
	var sb strings.Builder
	for {
		b, ok := dec.readByte()
		if !ok {
			return false
		} else if b == '\r' || b == '\n' {
			dec.mustUnreadByte()
			break
		}
		sb.WriteByte(b)
	}
	if sb.Len() == 0 {
		return false
	}
	*ptr = sb.String()
	return true
}

func (dec *Decoder) ExpectText(ptr *string) bool {
	return dec.Expect(dec.Text(ptr), "text")
}
