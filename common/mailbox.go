package common

type MailboxInfo struct {
	Flags []string
	Delimiter string
	Name string
}

type MailboxStatus struct {
	Name string
	ReadOnly bool
	Flags []string
	PermanentFlags []string
	Total uint32
	Recent uint32
	Unseen uint32
	UidNext uint32
	UidValidity uint32
}
