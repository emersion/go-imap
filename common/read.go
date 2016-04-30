package common

import (
	"errors"
	"io"
	"strconv"
)

const (
	delim = ' '
	quote = '"'
	literalStart = '{'
	literalEnd = '}'
	listStart = '('
	listEnd = ')'
)

type StringReader interface {
	ReadString(delim byte) (line string, err error)
}

type reader interface {
	io.RuneScanner
	StringReader
}

type Reader struct {
	reader
}

func trimSuffix(str string, suffix rune) string {
	return str[:len(str)-1]
}

func (r *Reader) ReadAtom() (interface{}, error) {
	atom, err := r.ReadString(byte(delim))
	if err != nil {
		return nil, err
	}
	atom = trimSuffix(atom, delim)

	if atom == "NIL" {
		return nil, nil
	}
	return atom, nil
}

func (r *Reader) ReadLiteral() (literal *Literal, err error) {
	char, _, err := r.ReadRune()
	if err != nil {
		return
	}
	if char != literalStart {
		err = errors.New("Literal string doesn't start with an open brace")
		return
	}

	lstr, err := r.ReadString(byte(literalEnd))
	if err != nil {
		return
	}
	lstr = trimSuffix(lstr, literalEnd)

	l, err := strconv.Atoi(lstr)
	if err != nil {
		return
	}

	literal = &Literal{Len: l}
	return
}

func (r *Reader) ReadQuotedString() (str string, err error) {
	char, _, err := r.ReadRune()
	if err != nil {
		return
	}
	if char != quote {
		err = errors.New("Quoted string doesn't start with a quote")
		return
	}

	str, err = r.ReadString(byte(quote))
	if err != nil {
		return
	}
	str = trimSuffix(str, quote)
	return
}

func (r *Reader) ReadFields() (fields []interface{}, err error) {
	var char rune
	for {
		if char, _, err = r.ReadRune(); err != nil {
			return
		}
		if err = r.UnreadRune(); err != nil {
			return
		}

		var field interface{}
		switch char {
		case '\n', ')', ']': // TODO: more generic check
			return
		case literalStart:
			field, err = r.ReadLiteral()
		case quote:
			field, err = r.ReadQuotedString()
		case listStart:
			field, err = r.ReadList()
		default:
			field, err = r.ReadAtom()
		}

		if err != nil {
			return
		}

		fields = append(fields, field)
	}
}

func (r *Reader) ReadList() (fields []interface{}, err error) {
	char, _, err := r.ReadRune()
	if err != nil {
		return
	}
	if char != listStart {
		err = errors.New("List doesn't start with an open parenthesis")
		return
	}

	fields, err = r.ReadFields()
	if err != nil {
		return
	}

	char, _, err = r.ReadRune()
	if err != nil {
		return
	}
	if char != listStart {
		err = errors.New("List doesn't end with a close parenthesis")
	}
	return
}

func (r *Reader) ReadLine() (fields []interface{}, err error) {
	fields, err = r.ReadFields()
	if err != nil {
		return
	}

	char, _, err := r.ReadRune()
	if err != nil {
		return
	}
	if char != '\n' {
		err = errors.New("Line doesn't end with a newline character")
	}
	return
}

func (r *Reader) ReadInfo() (info string, err error) {
	info, err = r.ReadString(byte('\n'))
	if err != nil {
		return
	}
	info = trimSuffix(info, '\n')
	return
}

func NewReader(r reader) *Reader {
	return &Reader{r}
}

type ReaderFrom interface {
	ReadFrom(r Reader) error
}
