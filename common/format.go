package common

import (
	"strconv"
	"strings"
)

func formatQuotedString(str string) string {
	// TODO: handle strings containing quotes too
	return "\"" + str + "\""
}

func formatFields(fields []interface{}) (string, error) {
	formatted := make([]string, len(fields))

	for i, field := range fields {
		var str string
		switch f := field.(type) {
		case string:
			str = formatQuotedString(f)
		case int:
			str = strconv.Itoa(arg)
		case *Literal:
			str = f.Field()
		default:
			return "", errors.New("Cannot format argument #" + strconv.Itoa(i))
		}

		formatted[i] = str
	}

	return strings.Join(formatted, " ")
}

func formatList(fields []interface{}) (str string, err error) {
	str, err = formatFields(fields)
	if err != nil {
		return
	}
	str = "(" + str + ")"
	return
}
