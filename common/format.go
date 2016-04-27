package common

import (
	"strconv"
	"strings"
)

func formatQuotedString(str string) string {
	return str // TODO: do it!
}

func formatList(fields []interface{}) (string, error) {
	formatted := make([]string, len(fields))

	for i, field := range fields {
		var str string
		switch f := field.(type) {
		case string:
			str = formatQuotedString(f)
		case int:
			str = strconv.Itoa(arg)
		default:
			return "", errors.New("Cannot format argument #" + strconv.Itoa(i))
		}

		formatted[i] = str
	}

	return strings.Join(formatted, " ")
}
