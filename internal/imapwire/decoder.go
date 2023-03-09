package imapwire

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"
)

// A Decoder reads IMAP data.
//
// There are multiple families of methods:
//
//   - Methods directly named after IMAP grammar elements attempt to decode
//     said element, and return false if it's another element.
//   - "Expect" methods do the same, but set the decoder error (see Err) on
//     failure.
type Decoder struct {
	r       *bufio.Reader
	err     error
	literal bool
}

// NewDecoder creates a new decoder.
func NewDecoder(r *bufio.Reader) *Decoder {
	return &Decoder{r: r}
}

func (dec *Decoder) mustUnreadByte() {
	if err := dec.r.UnreadByte(); err != nil {
		panic(fmt.Errorf("imapwire: failed to unread byte: %v", err))
	}
}

// Err returns the decoder error, if any.
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
	if dec.literal {
		return 0, dec.returnErr(fmt.Errorf("imapwire: cannot decode while a literal is open"))
	}
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

// EOF returns true if end-of-file is reached.
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

// Expect sets the decoder error if ok is false.
func (dec *Decoder) Expect(ok bool, name string) bool {
	if !ok {
		err := fmt.Errorf("expected %v", name)
		if dec.r.Buffered() > 0 {
			b, _ := dec.r.Peek(1)
			err = fmt.Errorf("%v, got '%v'", err, string(b))
		}
		return dec.returnErr(err)
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

func (dec *Decoder) Skip(untilCh byte) {
	for {
		ch, ok := dec.readByte()
		if !ok {
			return
		} else if ch == untilCh {
			dec.mustUnreadByte()
			return
		}
	}
}

func (dec *Decoder) Number64() (v int64, ok bool) {
	var sb strings.Builder
	for {
		ch, ok := dec.readByte()
		if !ok {
			return 0, false
		} else if ch < '0' || ch > '9' {
			dec.mustUnreadByte()
			break
		}
		sb.WriteByte(ch)
	}
	if sb.Len() == 0 {
		return 0, false
	}
	v, err := strconv.ParseInt(sb.String(), 10, 64)
	if err != nil {
		return 0, false // can happen on overflow
	}
	return v, true
}

func (dec *Decoder) ExpectNumber64() (v int64, ok bool) {
	v, ok = dec.Number64()
	dec.Expect(ok, "number64")
	return v, ok
}

func (dec *Decoder) ExpectNString() (lit *LiteralReader, nonSync, ok bool) {
	// TODO: quoted
	var s string
	if dec.Atom(&s) {
		if s == "NIL" {
			return nil, true, true
		}
		return newLiteralReaderFromString(s), true, true
	}
	if lit, nonSync, ok = dec.Literal(); ok {
		return lit, nonSync, true
	} else {
		return nil, false, dec.Expect(false, "nstring")
	}
}

func (dec *Decoder) Literal() (lit *LiteralReader, nonSync, ok bool) {
	if !dec.Special('{') {
		return nil, false, false
	}
	size, ok := dec.ExpectNumber64()
	if !ok {
		return nil, false, false
	}
	nonSync = dec.acceptByte('+')
	if !dec.ExpectSpecial('}') || !dec.ExpectCRLF() {
		return nil, false, false
	}
	dec.literal = true
	lit = &LiteralReader{
		dec:  dec,
		size: size,
		r:    io.LimitReader(dec.r, size),
	}
	return lit, nonSync, true
}

type LiteralReader struct {
	dec  *Decoder
	size int64
	r    io.Reader
}

func newLiteralReaderFromString(s string) *LiteralReader {
	return &LiteralReader{
		size: int64(len(s)),
		r:    strings.NewReader(s),
	}
}

func (lit *LiteralReader) Size() int64 {
	return lit.size
}

func (lit *LiteralReader) Read(b []byte) (int, error) {
	n, err := lit.r.Read(b)
	if err == io.EOF && lit.dec != nil {
		lit.dec.literal = false
		lit.dec = nil
	}
	return n, err
}
