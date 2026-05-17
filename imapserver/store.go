package imapserver

import (
	"strings"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func (c *Conn) handleStore(dec *imapwire.Decoder, numKind NumKind) error {
	var (
		numSet  imap.NumSet
		item    string
		options imap.StoreOptions
	)
	if !dec.ExpectSP() || !dec.ExpectNumSet(numKind.wire(), &numSet) || !dec.ExpectSP() {
		return dec.Err()
	}
	// Optional store-modifiers, RFC 7162 §3.1.3:
	//   store-modifiers = "(" store-modifier *(SP store-modifier) ")" SP
	if dec.Special('(') {
		if err := readStoreModifiers(dec, &options); err != nil {
			return err
		}
		if !dec.ExpectSpecial(')') || !dec.ExpectSP() {
			return dec.Err()
		}
	}
	if !dec.ExpectAtom(&item) || !dec.ExpectSP() {
		return dec.Err()
	}
	var flags []imap.Flag
	isList, err := dec.List(func() error {
		flag, err := internal.ExpectFlag(dec)
		if err != nil {
			return err
		}
		flags = append(flags, flag)
		return nil
	})
	if err != nil {
		return err
	} else if !isList {
		for {
			flag, err := internal.ExpectFlag(dec)
			if err != nil {
				return err
			}
			flags = append(flags, flag)

			if !dec.SP() {
				break
			}
		}
	}
	if !dec.ExpectCRLF() {
		return dec.Err()
	}

	item = strings.ToUpper(item)
	silent := strings.HasSuffix(item, ".SILENT")
	item = strings.TrimSuffix(item, ".SILENT")

	var op imap.StoreFlagsOp
	switch {
	case strings.HasPrefix(item, "+"):
		op = imap.StoreFlagsAdd
		item = strings.TrimPrefix(item, "+")
	case strings.HasPrefix(item, "-"):
		op = imap.StoreFlagsDel
		item = strings.TrimPrefix(item, "-")
	default:
		op = imap.StoreFlagsSet
	}

	if item != "FLAGS" {
		return newClientBugError("STORE can only change FLAGS")
	}

	if err := c.checkState(imap.ConnStateSelected); err != nil {
		return err
	}

	w := &FetchWriter{conn: c}
	return c.session.Store(w, numSet, &imap.StoreFlags{
		Op:     op,
		Silent: silent,
		Flags:  flags,
	}, &options)
}

// readStoreModifiers parses the parenthesised modifier list that may
// precede the STORE flag operation, RFC 7162 §3.1.3. Currently
// supported: UNCHANGEDSINCE (CONDSTORE).
func readStoreModifiers(dec *imapwire.Decoder, options *imap.StoreOptions) error {
	for {
		var name string
		if !dec.ExpectAtom(&name) {
			return dec.Err()
		}
		switch strings.ToUpper(name) {
		case "UNCHANGEDSINCE":
			if !dec.ExpectSP() || !dec.ExpectModSeq(&options.UnchangedSince) {
				return dec.Err()
			}
		default:
			return newClientBugError("unknown STORE modifier")
		}
		if !dec.SP() {
			return nil
		}
	}
}
