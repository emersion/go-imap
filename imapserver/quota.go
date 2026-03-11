package imapserver

import (
	"fmt"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

// QuotaResourceData contains the usage and limit for a quota resource.
type QuotaResourceData struct {
	Usage int64
	Limit int64
}

// QuotaData is the data returned by a QUOTA response.
type QuotaData struct {
	Root      string
	Resources map[imap.QuotaResourceType]QuotaResourceData
}

// SessionQuota is an IMAP session which supports QUOTA (RFC 9208).
type SessionQuota interface {
	Session

	// GetQuota returns the quota for the given root.
	GetQuota(root string) (*QuotaData, error)
	// GetQuotaRoot returns the quota roots for the given mailbox and
	// the quota data for each root.
	GetQuotaRoot(mailbox string) ([]string, []*QuotaData, error)
}

func (c *Conn) handleGetQuota(tag string, dec *imapwire.Decoder) error {
	var root string
	if !dec.ExpectSP() || !dec.ExpectAString(&root) || !dec.ExpectCRLF() {
		return dec.Err()
	}

	if err := c.checkState(imap.ConnStateAuthenticated); err != nil {
		return err
	}

	quotaSession, ok := c.session.(SessionQuota)
	if !ok {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Code: imap.ResponseCodeCannot,
			Text: "GETQUOTA command is not supported by this session",
		}
	}

	data, err := quotaSession.GetQuota(root)
	if err != nil {
		return err
	}

	if err := c.writeQuota(data); err != nil {
		return err
	}

	return c.writeStatusResp(tag, &imap.StatusResponse{
		Type: imap.StatusResponseTypeOK,
		Text: "GETQUOTA completed",
	})
}

func (c *Conn) handleGetQuotaRoot(tag string, dec *imapwire.Decoder) error {
	var mailbox string
	if !dec.ExpectSP() || !dec.ExpectMailbox(&mailbox) || !dec.ExpectCRLF() {
		return dec.Err()
	}

	if err := c.checkState(imap.ConnStateAuthenticated); err != nil {
		return err
	}

	quotaSession, ok := c.session.(SessionQuota)
	if !ok {
		return &imap.Error{
			Type: imap.StatusResponseTypeNo,
			Code: imap.ResponseCodeCannot,
			Text: "GETQUOTAROOT command is not supported by this session",
		}
	}

	roots, quotas, err := quotaSession.GetQuotaRoot(mailbox)
	if err != nil {
		return err
	}

	// Write QUOTAROOT response
	enc := newResponseEncoder(c)
	enc.Atom("*").SP().Atom("QUOTAROOT").SP().Mailbox(mailbox)
	for _, root := range roots {
		enc.SP().String(root)
	}
	if err := enc.CRLF(); err != nil {
		enc.end()
		return err
	}
	enc.end()

	// Write QUOTA response for each root
	for _, data := range quotas {
		if err := c.writeQuota(data); err != nil {
			return err
		}
	}

	return c.writeStatusResp(tag, &imap.StatusResponse{
		Type: imap.StatusResponseTypeOK,
		Text: fmt.Sprintf("GETQUOTAROOT completed for %v", mailbox),
	})
}

func (c *Conn) writeQuota(data *QuotaData) error {
	enc := newResponseEncoder(c)
	defer enc.end()

	enc.Atom("*").SP().Atom("QUOTA").SP().String(data.Root).SP()
	enc.Special('(')
	i := 0
	for typ, res := range data.Resources {
		if i > 0 {
			enc.SP()
		}
		enc.Atom(string(typ)).SP().Number64(res.Usage).SP().Number64(res.Limit)
		i++
	}
	enc.Special(')')
	return enc.CRLF()
}
