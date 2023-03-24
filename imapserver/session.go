package imapserver

// Session is an IMAP session.
type Session interface {
	Close() error
	Login(username, password string) error
}
