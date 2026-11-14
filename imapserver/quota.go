package imapserver

import (
	"fmt"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

// SessionQuota is an IMAP session which supports the QUOTA extension
// (RFC 9208).
//
// SETQUOTA is not supported: it is always rejected with NOPERM and QUOTASET
// is not advertised.
type SessionQuota interface {
	Session

	// QuotaResourceTypes returns the resource types supported by the server.
	// Each is advertised as a QUOTA=RES-<type> capability. QUOTA responses
	// only list these resource types, in this order.
	QuotaResourceTypes() []imap.QuotaResourceType

	// GetQuota returns the usage and limits for a quota root. An unknown
	// root should be reported with an error.
	//
	// A resource without a limit must be left out of QuotaData.Resources. A
	// quota root must have at least one resource with a limit.
	GetQuota(root string) (*imap.QuotaData, error)

	// GetQuotaRoot returns the quota roots for a mailbox, along with their
	// usage and limits. The list may be empty if no quota root applies.
	GetQuotaRoot(mailbox string) ([]imap.QuotaData, error)
}

func (c *Conn) handleGetQuota(dec *imapwire.Decoder) error {
	var root string
	if !dec.ExpectSP() || !dec.ExpectAString(&root) || !dec.ExpectCRLF() {
		return dec.Err()
	}

	if err := c.checkState(imap.ConnStateAuthenticated); err != nil {
		return err
	}

	session, ok := c.session.(SessionQuota)
	if !ok {
		return newClientBugError("QUOTA is not supported")
	}

	data, err := session.GetQuota(root)
	if err != nil {
		return err
	}

	resources, err := quotaResources(session.QuotaResourceTypes(), data)
	if err != nil {
		return err
	}

	return c.writeQuota(data, resources)
}

func (c *Conn) handleGetQuotaRoot(dec *imapwire.Decoder) error {
	var mailbox string
	if !dec.ExpectSP() || !dec.ExpectMailbox(&mailbox) || !dec.ExpectCRLF() {
		return dec.Err()
	}

	if err := c.checkState(imap.ConnStateAuthenticated); err != nil {
		return err
	}

	session, ok := c.session.(SessionQuota)
	if !ok {
		return newClientBugError("QUOTA is not supported")
	}

	l, err := session.GetQuotaRoot(mailbox)
	if err != nil {
		return err
	}

	types := session.QuotaResourceTypes()
	resources := make([][]imap.QuotaResourceType, len(l))
	for i := range l {
		resources[i], err = quotaResources(types, &l[i])
		if err != nil {
			return err
		}
	}

	if err := c.writeQuotaRoot(mailbox, l); err != nil {
		return err
	}
	for i := range l {
		if err := c.writeQuota(&l[i], resources[i]); err != nil {
			return err
		}
	}
	return nil
}

func (c *Conn) handleSetQuota(dec *imapwire.Decoder) error {
	var root string
	if !dec.ExpectSP() || !dec.ExpectAString(&root) || !dec.ExpectSP() {
		return dec.Err()
	}
	err := dec.ExpectList(func() error {
		var (
			name  string
			limit int64
		)
		if !dec.ExpectAtom(&name) || !dec.ExpectSP() || !dec.ExpectNumber64(&limit) {
			return dec.Err()
		}
		return nil
	})
	if err != nil {
		return err
	}
	if !dec.ExpectCRLF() {
		return dec.Err()
	}

	if err := c.checkState(imap.ConnStateAuthenticated); err != nil {
		return err
	}

	if _, ok := c.session.(SessionQuota); !ok {
		return newClientBugError("QUOTA is not supported")
	}

	return &imap.Error{
		Type: imap.StatusResponseTypeNo,
		Code: imap.ResponseCodeNoPerm,
		Text: "Quota limits cannot be changed",
	}
}

func (c *Conn) writeQuotaRoot(mailbox string, l []imap.QuotaData) error {
	enc := newResponseEncoder(c)
	defer enc.end()
	enc.Atom("*").SP().Atom("QUOTAROOT").SP().Mailbox(mailbox)
	for _, data := range l {
		enc.SP().String(data.Root)
	}
	return enc.CRLF()
}

// quotaResources returns the resource types of data to write, in the order
// given by the session.
func quotaResources(types []imap.QuotaResourceType, data *imap.QuotaData) ([]imap.QuotaResourceType, error) {
	var l []imap.QuotaResourceType
	for _, typ := range types {
		res, ok := data.Resources[typ]
		if !ok {
			continue
		}
		if res.Usage < 0 || res.Limit < 0 {
			return nil, fmt.Errorf("imapserver: negative usage or limit for quota resource %v in root %q", typ, data.Root)
		}
		l = append(l, typ)
	}
	if len(l) == 0 {
		// RFC 9208 requires at least one resource in a QUOTA response
		return nil, fmt.Errorf("imapserver: no resource with a limit in quota root %q", data.Root)
	}
	return l, nil
}

func (c *Conn) writeQuota(data *imap.QuotaData, resources []imap.QuotaResourceType) error {
	enc := newResponseEncoder(c)
	defer enc.end()
	enc.Atom("*").SP().Atom("QUOTA").SP().String(data.Root).SP()
	enc.List(len(resources), func(i int) {
		res := data.Resources[resources[i]]
		enc.Atom(string(resources[i])).SP().Number64(res.Usage).SP().Number64(res.Limit)
	})
	return enc.CRLF()
}
