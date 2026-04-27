package imapclient

import (
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

// AttributeDecoder is the value-decoding API exposed to user-supplied
// CustomAttributeDecoderFunc implementations. It wraps the internal IMAP wire
// decoder so callers can register custom FETCH attribute decoders without
// taking a dependency on internal packages.
//
// Methods named after IMAP grammar elements return false when the next token
// does not match. "Expect" variants additionally record an error retrievable
// via Err.
//
// AttributeDecoder is constructed by the library; user code receives an
// *AttributeDecoder argument from CustomAttributeDecoderFunc.
type AttributeDecoder struct {
	dec *imapwire.Decoder
}

func newAttributeDecoder(dec *imapwire.Decoder) *AttributeDecoder {
	return &AttributeDecoder{dec: dec}
}

func (d *AttributeDecoder) Err() error {
	return d.dec.Err()
}

func (d *AttributeDecoder) SP() bool {
	return d.dec.SP()
}

func (d *AttributeDecoder) ExpectSP() bool {
	return d.dec.ExpectSP()
}

func (d *AttributeDecoder) Atom(ptr *string) bool {
	return d.dec.Atom(ptr)
}

func (d *AttributeDecoder) ExpectAtom(ptr *string) bool {
	return d.dec.ExpectAtom(ptr)
}

func (d *AttributeDecoder) Quoted(ptr *string) bool {
	return d.dec.Quoted(ptr)
}

// String decodes a quoted string or a literal.
func (d *AttributeDecoder) String(ptr *string) bool {
	return d.dec.String(ptr)
}

func (d *AttributeDecoder) ExpectString(ptr *string) bool {
	return d.dec.ExpectString(ptr)
}

// ExpectAString decodes an astring (atom, quoted string, or literal).
func (d *AttributeDecoder) ExpectAString(ptr *string) bool {
	return d.dec.ExpectAString(ptr)
}

// ExpectNString decodes an nstring (string or NIL).
func (d *AttributeDecoder) ExpectNString(ptr *string) bool {
	return d.dec.ExpectNString(ptr)
}

func (d *AttributeDecoder) ExpectNIL() bool {
	return d.dec.ExpectNIL()
}

func (d *AttributeDecoder) Number(ptr *uint32) bool {
	return d.dec.Number(ptr)
}

func (d *AttributeDecoder) ExpectNumber(ptr *uint32) bool {
	return d.dec.ExpectNumber(ptr)
}

func (d *AttributeDecoder) Number64(ptr *int64) bool {
	return d.dec.Number64(ptr)
}

func (d *AttributeDecoder) ExpectNumber64(ptr *int64) bool {
	return d.dec.ExpectNumber64(ptr)
}

func (d *AttributeDecoder) Special(b byte) bool {
	return d.dec.Special(b)
}

func (d *AttributeDecoder) ExpectSpecial(b byte) bool {
	return d.dec.ExpectSpecial(b)
}

// List decodes a parenthesized list, calling f for each element. The first
// return reports whether a list was present at all (vs. a different token).
func (d *AttributeDecoder) List(f func() error) (bool, error) {
	return d.dec.List(f)
}

func (d *AttributeDecoder) ExpectList(f func() error) error {
	return d.dec.ExpectList(f)
}

// ExpectNList decodes a parenthesized list or NIL.
func (d *AttributeDecoder) ExpectNList(f func() error) error {
	return d.dec.ExpectNList(f)
}

// DiscardValue consumes and discards a value of any type (atom, string, list).
func (d *AttributeDecoder) DiscardValue() bool {
	return d.dec.DiscardValue()
}
