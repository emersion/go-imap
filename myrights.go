package imap

// MyrightsData is the data returned by the MYRIGHTS command.
type MyrightsData struct {
	Mailbox string
	Rights  string
}
