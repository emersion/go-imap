package imap

// MyRightsData is the data returned by the MYRIGHTS command.
type MyRightsData struct {
	Mailbox string
	Rights  RightSet
}
