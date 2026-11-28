package imapserver

import (
	"strings"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func (c *Conn) handleStore(tag string, dec *imapwire.Decoder, numKind NumKind) error {
	var (
		numSet imap.NumSet
		item   string
	)
	if !dec.ExpectSP() || !dec.ExpectNumSet(numKind.wire(), &numSet) {
		return dec.Err()
	}

	if !dec.ExpectSP() {
		return dec.Err()
	}

	// RFC 4466:
	//    store           = "STORE" SP sequence-set [store-modifiers]
	//                     SP store-att-flags
	options, err := readStoreOptions(dec)
	if err != nil {
		return err
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

	if options.UnchangedSince != 0 {
		c.enable(imap.CapCondStore)
	}

	w := &FetchWriter{conn: c}
	data, err := c.session.Store(w, numSet, &imap.StoreFlags{
		Op:     op,
		Silent: silent,
		Flags:  flags,
	}, &options)
	if err != nil {
		return err
	}

	return c.writeStoreOK(tag, data)
}

func (c *Conn) writeStoreOK(tag string, data *imap.StoreData) error {
	enc := newResponseEncoder(c)
	defer enc.end()

	enc.Atom(tag).SP().Atom("OK").SP()
	if data != nil && data.Modified != nil && !isNumSetEmpty(data.Modified) && c.enabled.Has(imap.CapCondStore) {
		enc.Special('[')
		enc.Atom("MODIFIED").SP().NumSet(data.Modified)
		enc.Special(']').SP()
	}
	enc.Text("STORE completed")
	return enc.CRLF()
}

func readStoreOptions(dec *imapwire.Decoder) (imap.StoreOptions, error) {
	options := imap.StoreOptions{}
	hasList, err := dec.List(func() error {
		// store-modifier  = store-modifier-name [SP store-modif-params]
		// store-modif-params = tagged-ext-val
		// store-modifier-name = tagged-ext-label

		var modifierName string
		if !dec.ExpectAtom(&modifierName) {
			return dec.Err()
		}

		switch strings.ToUpper(modifierName) {
		case "UNCHANGEDSINCE":
			if !dec.ExpectSP() {
				return dec.Err()
			}
			if !dec.ExpectModSeq(&options.UnchangedSince) {
				return dec.Err()
			}
			return nil
		default:
			return newClientBugError("Unknown STORE modifier: " + modifierName)
		}
	})
	if !hasList || err != nil {
		return options, err
	}

	if !dec.ExpectSP() {
		return imap.StoreOptions{}, dec.Err()
	}
	return options, nil
}
