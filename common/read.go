package common

import (
	"errors"
	"io"
	"strconv"
	"strings"
)

const (
	sp = ' '
	dquote = '"'
	literalStart = '{'
	literalEnd = '}'
	listStart = '('
	listEnd = ')'
	respCodeStart = '['
	respCodeEnd = ']'
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

func (r *Reader) ReadSp() error {
	char, _, err := r.ReadRune()
	if err != nil {
		return err
	}
	if char != sp {
		return errors.New("Not a space")
	}
	return nil
}

func (r *Reader) ReadAtom() (interface{}, error) {
	var atom string
	for {
		char, _, err := r.ReadRune()
		if err != nil {
			return nil, err
		}

		// TODO: list-wildcards and \
		if char == listStart || char == literalStart || char == dquote {
			return nil, errors.New("Atom contains forbidden char: " + string(char))
		}
		if char == sp || char == listEnd || char == respCodeEnd || char == '\n' {
			break
		}

		atom += string(char)
	}

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
	if char != dquote {
		err = errors.New("Quoted string doesn't start with a double quote")
		return
	}

	str, err = r.ReadString(byte(dquote))
	if err != nil {
		return
	}
	str = trimSuffix(str, dquote)
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
		case literalStart:
			field, err = r.ReadLiteral()
		case dquote:
			field, err = r.ReadQuotedString()
		case listStart:
			field, err = r.ReadList()
		default:
			field, err = r.ReadAtom()
			r.UnreadRune()
		}

		if err != nil {
			return
		}
		fields = append(fields, field)

		if char, _, err = r.ReadRune(); err != nil {
			return
		}
		if char == '\n' || char == listEnd || char == respCodeEnd {
			return
		}
		if char != sp {
			err = errors.New("Fields are not separated by a space")
			return
		}
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

	r.UnreadRune()
	char, _, err = r.ReadRune()
	if err != nil {
		return
	}
	if char != listEnd {
		err = errors.New("List doesn't end with a close parenthesis")
	}
	return
}

func (r *Reader) ReadLine() (fields []interface{}, err error) {
	fields, err = r.ReadFields()
	if err != nil {
		return
	}

	r.UnreadRune()
	char, _, err := r.ReadRune()
	if err != nil {
		return
	}
	if char != '\n' {
		err = errors.New("Line doesn't end with a newline character")
	}
	return
}

func (r *Reader) ReadRespCode() (code StatusRespCode, fields []interface{}, err error) {
	char, _, err := r.ReadRune()
	if err != nil {
		return
	}
	if char != respCodeStart {
		err = errors.New("Response code doesn't start with an open bracket")
		return
	}

	fields, err = r.ReadFields()
	if err != nil {
		return
	}

	if len(fields) == 0 {
		err = errors.New("Response code doesn't contain any field")
		return
	}

	codeStr, ok := fields[0].(string)
	if !ok {
		err = errors.New("Response code doesn't start with a string atom")
		return
	}

	code = StatusRespCode(codeStr)
	fields = fields[1:]

	r.UnreadRune()
	char, _, err = r.ReadRune()
	if err != nil {
		return
	}
	if char != respCodeEnd {
		err = errors.New("Response code doesn't end with a close bracket")
	}
	return
}

func (r *Reader) ReadInfo() (info string, err error) {
	info, err = r.ReadString(byte('\n'))
	if err != nil {
		return
	}
	info = strings.TrimSuffix(info, "\n")
	info = strings.TrimLeft(info, " ")
	return
}

func NewReader(r reader) *Reader {
	return &Reader{r}
}

type ReaderFrom interface {
	ReadFrom(r Reader) error
}
