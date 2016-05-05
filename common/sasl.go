package common

// Note:
//   Most of this code was copied, with some modifications, from net/smtp. It
//   would be better if Go provided a standard package (e.g. crypto/sasl) that
//   could be shared by SMTP, IMAP, and other packages.

// Interface for performing challenge-response authentication.
type Sasl interface {
	// Start begins SASL authentication with the server. It returns the
	// authentication mechanism name and "initial response" data (if required by
	// the selected mechanism). A non-nil error causes the client to abort the
	// authentication attempt.
	//
	// A nil ir value is different from a zero-length value. The nil value
	// indicates that the selected mechanism does not use an initial response,
	// while a zero-length value indicates an empty initial response, which must
	// be sent to the server.
	Start() (mech string, ir []byte, err error)

	// Next continues challenge-response authentication. A non-nil error causes
	// the client to abort the authentication attempt.
	Next(challenge []byte) (response []byte, err error)
}
