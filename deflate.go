package imap

// A CompressUnsuppportedError is returned by Client.Compress when the provided
// compression mechanism is not supported.
type CompressUnsupportedError struct {
	Mechanism string
}

func (err CompressUnsupportedError) Error() string {
	return "COMPRESS mechanism " + err.Mechanism + " not supported"
}

// Compression algorithms for use with COMPRESS extension (RFC 4978).
const (
	// The DEFLATE algorithm, defined in RFC 1951.
	CompressDeflate = "DEFLATE"
)
