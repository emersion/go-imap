package imap

import (
	"strconv"
	"strings"
)

// Cap represents an IMAP capability.
type Cap string

// Registered capabilities.
//
// See: https://www.iana.org/assignments/imap-capabilities/
const (
	CapIMAP4rev1 Cap = "IMAP4rev1" // RFC 3501
	CapIMAP4rev2 Cap = "IMAP4rev2" // RFC 9051

	CapAuthPlain Cap = "AUTH=PLAIN"

	CapStartTLS      Cap = "STARTTLS"
	CapLoginDisabled Cap = "LOGINDISABLED"

	// Folded in IMAP4rev2
	CapNamespace    Cap = "NAMESPACE"     // RFC 2342
	CapUnselect     Cap = "UNSELECT"      // RFC 3691
	CapUIDPlus      Cap = "UIDPLUS"       // RFC 4315
	CapESearch      Cap = "ESEARCH"       // RFC 4731
	CapSearchRes    Cap = "SEARCHRES"     // RFC 5182
	CapEnable       Cap = "ENABLE"        // RFC 5161
	CapIdle         Cap = "IDLE"          // RFC 2177
	CapSASLIR       Cap = "SASL-IR"       // RFC 4959
	CapListExtended Cap = "LIST-EXTENDED" // RFC 5258
	CapListStatus   Cap = "LIST-STATUS"   // RFC 5819
	CapMove         Cap = "MOVE"          // RFC 6851
	CapLiteralMinus Cap = "LITERAL-"      // RFC 7888
	CapStatusSize   Cap = "STATUS=SIZE"   // RFC 8438

	CapACL              Cap = "ACL"                // RFC 4314
	CapAppendLimit      Cap = "APPENDLIMIT"        // RFC 7889
	CapBinary           Cap = "BINARY"             // RFC 3516
	CapCatenate         Cap = "CATENATE"           // RFC 4469
	CapChildren         Cap = "CHILDREN"           // RFC 3348
	CapCondStore        Cap = "CONDSTORE"          // RFC 7162
	CapConvert          Cap = "CONVERT"            // RFC 5259
	CapCreateSpecialUse Cap = "CREATE-SPECIAL-USE" // RFC 6154
	CapESort            Cap = "ESORT"              // RFC 5267
	CapFilters          Cap = "FILTERS"            // RFC 5466
	CapID               Cap = "ID"                 // RFC 2971
	CapLanguage         Cap = "LANGUAGE"           // RFC 5255
	CapListMyRights     Cap = "LIST-MYRIGHTS"      // RFC 8440
	CapLiteralPlus      Cap = "LITERAL+"           // RFC 7888
	CapLoginReferrals   Cap = "LOGIN-REFERRALS"    // RFC 2221
	CapMailboxReferrals Cap = "MAILBOX-REFERRALS"  // RFC 2193
	CapMetadata         Cap = "METADATA"           // RFC 5464
	CapMetadataServer   Cap = "METADATA-SERVER"    // RFC 5464
	CapMultiAppend      Cap = "MULTIAPPEND"        // RFC 3502
	CapMultiSearch      Cap = "MULTISEARCH"        // RFC 7377
	CapNotify           Cap = "NOTIFY"             // RFC 5465
	CapObjectID         Cap = "OBJECTID"           // RFC 8474
	CapPreview          Cap = "PREVIEW"            // RFC 8970
	CapQResync          Cap = "QRESYNC"            // RFC 7162
	CapQuota            Cap = "QUOTA"              // RFC 9208
	CapQuotaSet         Cap = "QUOTASET"           // RFC 9208
	CapReplace          Cap = "REPLACE"            // RFC 8508
	CapSaveDate         Cap = "SAVEDATE"           // RFC 8514
	CapSearchFuzzy      Cap = "SEARCH=FUZZY"       // RFC 6203
	CapSort             Cap = "SORT"               // RFC 5256
	CapSortDisplay      Cap = "SORT=DISPLAY"       // RFC 5957
	CapSpecialUse       Cap = "SPECIAL-USE"        // RFC 6154
	CapUnauthenticate   Cap = "UNAUTHENTICATE"     // RFC 8437
	CapURLPartial       Cap = "URL-PARTIAL"        // RFC 5550
	CapURLAuth          Cap = "URLAUTH"            // RFC 4467
	CapUTF8Accept       Cap = "UTF8=ACCEPT"        // RFC 6855
	CapUTF8Only         Cap = "UTF8=ONLY"          // RFC 6855
	CapWithin           Cap = "WITHIN"             // RFC 5032
	CapUIDOnly          Cap = "UIDONLY"            // RFC 9586
	CapListMetadata     Cap = "LIST-METADATA"      // RFC 9590
	CapInProgress       Cap = "INPROGRESS"         // RFC 9585
)

var imap4rev2Caps = CapSet{
	CapNamespace:    {},
	CapUnselect:     {},
	CapUIDPlus:      {},
	CapESearch:      {},
	CapSearchRes:    {},
	CapEnable:       {},
	CapIdle:         {},
	CapSASLIR:       {},
	CapListExtended: {},
	CapListStatus:   {},
	CapMove:         {},
	CapLiteralMinus: {},
	CapStatusSize:   {},
}

// CapSet is a set of capabilities.
type CapSet struct {
	IMAP4rev1 bool // RFC 3501
	IMAP4rev2 bool // RFC 9051

	Auth map[string]struct{}

	StartTLS      bool
	LoginDisabled bool

	// Folded in IMAP4rev2
	Namespace    bool // RFC 2342
	Unselect     bool // RFC 3691
	UIDPlus      bool // RFC 4315
	ESearch      bool // RFC 4731
	SearchRes    bool // RFC 5182
	Enable       bool // RFC 5161
	Idle         bool // RFC 2177
	SASLIR       bool // RFC 4959
	ListExtended bool // RFC 5258
	ListStatus   bool // RFC 5819
	Move         bool // RFC 6851
	LiteralMinus bool // RFC 7888
	StatusSize   bool // RFC 8438

	ACL              bool // RFC 4314
	AppendLimit      bool // RFC 7889
	Binary           bool // RFC 3516
	Catenate         bool // RFC 4469
	Children         bool // RFC 3348
	CondStore        bool // RFC 7162
	Convert          bool // RFC 5259
	CreateSpecialUse bool // RFC 6154
	ESort            bool // RFC 5267
	Filters          bool // RFC 5466
	ID               bool // RFC 2971
	Language         bool // RFC 5255
	ListMyRights     bool // RFC 8440
	LiteralPlus      bool // RFC 7888
	LoginReferrals   bool // RFC 2221
	MailboxReferrals bool // RFC 2193
	Metadata         bool // RFC 5464
	MetadataServer   bool // RFC 5464
	MultiAppend      bool // RFC 3502
	MultiSearch      bool // RFC 7377
	Notify           bool // RFC 5465
	ObjectID         bool // RFC 8474
	Preview          bool // RFC 8970
	QResync          bool // RFC 7162
	Quota            bool // RFC 9208
	QuotaSet         bool // RFC 9208
	Replace          bool // RFC 8508
	SaveDate         bool // RFC 8514
	SearchFuzzy      bool // RFC 6203
	Sort             bool // RFC 5256
	SortDisplay      bool // RFC 5957
	SpecialUse       bool // RFC 6154
	Unauthenticate   bool // RFC 8437
	URLPartial       bool // RFC 5550
	URLAuth          bool // RFC 4467
	UTF8Accept       bool // RFC 6855
	UTF8Only         bool // RFC 6855
	Within           bool // RFC 5032
	UIDOnly          bool // RFC 9586
	ListMetadata     bool // RFC 9590
	InProgress       bool // RFC 9585
}

// Has checks whether a capability is supported.
//
// Some capabilities are implied by others, as such Has may return true even if
// the capability is not in the map.
func (set CapSet) Has(c Cap) bool {
	if set.has(c) {
		return true
	}

	if set.has(CapIMAP4rev2) && imap4rev2Caps.has(c) {
		return true
	}

	if c == CapLiteralMinus && set.has(CapLiteralPlus) {
		return true
	}
	if c == CapCondStore && set.has(CapQResync) {
		return true
	}
	if c == CapUTF8Accept && set.has(CapUTF8Only) {
		return true
	}
	if c == CapAppendLimit {
		_, ok := set.AppendLimit()
		return ok
	}

	return false
}

// AuthMechanisms returns the list of supported SASL mechanisms for
// authentication.
func (set CapSet) AuthMechanisms() []string {
	var l []string
	for c := range set {
		if !strings.HasPrefix(string(c), "AUTH=") {
			continue
		}
		mech := strings.TrimPrefix(string(c), "AUTH=")
		l = append(l, mech)
	}
	return l
}

// AppendLimit checks the APPENDLIMIT capability.
//
// If the server supports APPENDLIMIT, ok is true. If the server doesn't have
// the same upload limit for all mailboxes, limit is nil and per-mailbox
// limits must be queried via STATUS.
func (set CapSet) AppendLimit() (limit *uint32, ok bool) {
	if set.has(CapAppendLimit) {
		return nil, true
	}

	for c := range set {
		if !strings.HasPrefix(string(c), "APPENDLIMIT=") {
			continue
		}

		limitStr := strings.TrimPrefix(string(c), "APPENDLIMIT=")
		limit64, err := strconv.ParseUint(limitStr, 10, 32)
		if err == nil && limit64 > 0 {
			limit32 := uint32(limit64)
			return &limit32, true
		}
	}

	limit32 := ^uint32(0)
	return &limit32, false
}

// QuotaResourceTypes returns the list of supported QUOTA resource types.
func (set CapSet) QuotaResourceTypes() []QuotaResourceType {
	var l []QuotaResourceType
	for c := range set {
		if !strings.HasPrefix(string(c), "QUOTA=RES-") {
			continue
		}
		t := strings.TrimPrefix(string(c), "QUOTA=RES-")
		l = append(l, QuotaResourceType(t))
	}
	return l
}

// ThreadAlgorithms returns the list of supported threading algorithms.
func (set CapSet) ThreadAlgorithms() []ThreadAlgorithm {
	var l []ThreadAlgorithm
	for c := range set {
		if !strings.HasPrefix(string(c), "THREAD=") {
			continue
		}
		alg := strings.TrimPrefix(string(c), "THREAD=")
		l = append(l, ThreadAlgorithm(alg))
	}
	return l
}
