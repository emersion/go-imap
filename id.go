package imap

type IDData struct {
	Name        string
	Version     string
	OS          string
	OSVersion   string
	Vendor      string
	SupportURL  string
	Address     string
	Date        string
	Command     string
	Arguments   string
	Environment string

	// Raw contains all raw key-value pairs. Standard keys are also present
	// in this map. Keys are case-insensitive and are normalized to lowercase.
	Raw map[string]string
}
