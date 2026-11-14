package imap

// QuotaResourceType is a QUOTA resource type.
//
// See RFC 9208 section 5.
type QuotaResourceType string

const (
	QuotaResourceStorage           QuotaResourceType = "STORAGE"
	QuotaResourceMessage           QuotaResourceType = "MESSAGE"
	QuotaResourceMailbox           QuotaResourceType = "MAILBOX"
	QuotaResourceAnnotationStorage QuotaResourceType = "ANNOTATION-STORAGE"
)

// QuotaData is the data returned by a QUOTA response.
//
// Resources only contains resources which have a limit: a resource without a
// limit is not listed.
type QuotaData struct {
	Root      string
	Resources map[QuotaResourceType]QuotaResourceData
}

// QuotaResourceData contains the usage and limit for a quota resource.
//
// For STORAGE, the values are in units of 1024 octets.
type QuotaResourceData struct {
	Usage int64
	Limit int64
}
