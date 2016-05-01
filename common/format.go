package common

import (
	"errors"
	"strconv"
	"strings"
)

func formatAtomString(str string) string {
	return str
}

func formatQuotedString(str string) string {
	// TODO: handle strings containing quotes too
	return "\"" + str + "\""
}

func formatString(str string) string {
	// TODO: decide between atom and quoted (choose atom when possible)
	return formatAtomString(str)
}

func formatFields(fields []interface{}) (string, error) {
	formatted := make([]string, len(fields))

	for i, field := range fields {
		var str string
		var err error
		switch f := field.(type) {
		case string:
			str = formatString(f)
		case int:
			str = strconv.Itoa(f)
		case *Literal:
			str = f.Field()
		case []interface{}:
			str, err = formatFields(f)
		case *SeqSet:
			str = f.String()
		default:
			return "", errors.New("Cannot format argument #" + strconv.Itoa(i))
		}

		if err != nil {
			return "", err
		}

		formatted[i] = str
	}

	return strings.Join(formatted, " "), nil
}

func formatList(fields []interface{}) (str string, err error) {
	str, err = formatFields(fields)
	if err != nil {
		return
	}
	str = "(" + str + ")"
	return
}
