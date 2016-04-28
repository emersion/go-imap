package common

import (
	"bufio"
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

type Literal struct {
	Len int
	Str string
}

func (l *Literal) String() string {
	return l.Str
}

func (l *Literal) WriteTo(w io.Writer) (N int64, err error) {
	n, err := io.WriteString(w, string(literalStart) + strconv.Itoa(l.Len) + string(literalEnd))
	return int64(n), err
}

func trimSuffix(str string, suffix rune) string {
	return str[:len(str)-1]
}

func parseAtom(r bufio.Reader) (interface{}, error) {
	atom, err := r.ReadString(byte(delim))
	if err != nil && err != io.EOF {
		return nil, err
	}
	atom = trimSuffix(atom, delim)

	if atom == "NIL" {
		return nil, nil
	}
	return atom, nil
}

func parseLiteral(r bufio.Reader) (literal *imap.Literal, err error) {
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

func parseQuotedString(r bufio.Reader) (str string, err error) {
	char, _, err := r.ReadRune()
	if err != nil {
		return
	}
	if char != quote {
		err = errors.New("Quoted string doesn't start with a quote")
		return
	}

	str, err := r.ReadString(byte(quote))
	if err != nil {
		return
	}
	str = trimSuffix(str, quote)
	return
}

func parseList(r bufio.Reader) (fields []interface{}, err error) {
	return // TODO
}

func parseLine(r bufio.Reader) (fields []interface{}, err error) {
	var char rune
	for {
		chars, err = r.Peek(1)
		if err != nil {
			return
		}
		char = rune(chars[0])

		var field interface{}
		switch char {
		case '\n':
			return
		case literalStart:
			field, err = parseLiteral(r)
		case quote:
			field, err = parseQuotedString(r)
		case listStart:
			field, err = parseList(r)
		default:
			field, err = parseAtom(r)
		}

		if err != nil {
			return
		}

		fields = append(fields, field)
	}
}
