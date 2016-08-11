package imap

// A literal.
type Literal struct {
	// The literal length.
	len int
	// The literal contents.
	contents []byte
}

func (l *Literal) Len() int {
	return l.len
}

func (l *Literal) Bytes() []byte {
	return l.contents
}

func (l *Literal) String() string {
	return string(l.contents)
}

// Create a new literal.
func NewLiteral(b []byte) *Literal {
	return &Literal{len: len(b), contents: b}
}
