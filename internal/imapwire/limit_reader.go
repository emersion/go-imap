package imapwire

import "io"

// A reader that returns io.EOF after a specified number of bytes.
//
// If EOF is received before the requested number of bytes, io.ErrUnexpectedEOF is returned.
type limitReader struct {
	wrapped io.Reader
	count   int
}

func (reader *limitReader) Read(destination []byte) (int, error) {
	if reader.count == 0 {
		return 0, io.EOF
	}
	if len(destination) == 0 {
		return 0, nil
	}
	if len(destination) > reader.count {
		destination = destination[:reader.count]
	}
	read, err := reader.wrapped.Read(destination)
	reader.count -= read
	if err == nil {
		if reader.count == 0 {
			return read, io.EOF
		}
		return read, nil
	}
	if err == io.EOF {
		if reader.count == 0 {
			return read, io.EOF
		}
		return read, io.ErrUnexpectedEOF
	}
	return read, err
}

func newLimitReader(reader io.Reader, count int) *limitReader {
	return &limitReader{
		wrapped: reader,
		count:   count,
	}
}
