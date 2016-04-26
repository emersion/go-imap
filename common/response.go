package common

// An untagged response.
// See https://tools.ietf.org/html/rfc3501#section-2.2.2
type Response struct {
	Tag string
	Fields []interface{}
}
