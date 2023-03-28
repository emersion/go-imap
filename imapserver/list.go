package imapserver

import (
	"fmt"
	"strings"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func (c *Conn) handleList(dec *imapwire.Decoder) error {
	ref, pattern, options, err := readListCmd(dec)
	if err != nil {
		return err
	}

	if err := c.checkState(imap.ConnStateAuthenticated); err != nil {
		return err
	}

	w := &ListWriter{
		conn:    c,
		options: options,
	}
	return c.session.List(w, ref, pattern, options)
}

func (c *Conn) writeList(data *imap.ListData) error {
	enc := newResponseEncoder(c)
	defer enc.end()

	enc.Atom("*").SP().Atom("LIST").SP()
	enc.List(len(data.Attrs), func(i int) {
		enc.Atom(string(data.Attrs[i])) // TODO: validate attr
	})
	enc.SP()
	if data.Delim == 0 {
		enc.NIL()
	} else {
		enc.String(string(data.Delim)) // TODO: ensure we always use DQUOTE QUOTED-CHAR DQUOTE
	}
	enc.SP().Mailbox(data.Mailbox)

	var ext []string
	if data.ChildInfo != nil {
		ext = append(ext, "CHILDINFO")
	}
	if data.OldName != "" {
		ext = append(ext, "OLDNAME")
	}

	// TODO: omit extended data if the client didn't ask for it
	if len(ext) > 0 {
		enc.SP().List(len(ext), func(i int) {
			name := ext[i]
			enc.Atom(name).SP()
			switch name {
			case "CHILDINFO":
				enc.Special('(')
				if data.ChildInfo.Subscribed {
					enc.String("SUBSCRIBED") // TODO: ensure always quoted
				}
				enc.Special(')')
			case "OLDNAME":
				enc.Special('(').Mailbox(data.OldName).Special(')')
			default:
				panic(fmt.Errorf("imapserver: unknown LIST extended-item %v", name))
			}
		})
	}

	return enc.CRLF()
}

func readListCmd(dec *imapwire.Decoder) (ref, pattern string, options *imap.ListOptions, err error) {
	options = &imap.ListOptions{}

	if !dec.ExpectSP() {
		return "", "", nil, dec.Err()
	}

	hasSelectOpts, err := dec.List(func() error {
		var selectOpt string
		if !dec.ExpectAString(&selectOpt) {
			return dec.Err()
		}
		switch selectOpt {
		case "SUBSCRIBED":
			options.SelectSubscribed = true
		case "REMOTE":
			options.SelectRemote = true
		case "RECURSIVEMATCH":
			options.SelectRecursiveMatch = true
		}
		return nil
	})
	if err != nil {
		return "", "", nil, fmt.Errorf("in list-select-opts: %w", err)
	}
	if hasSelectOpts && !dec.ExpectSP() {
		return "", "", nil, dec.Err()
	}

	if !dec.ExpectMailbox(&ref) || !dec.ExpectSP() {
		return "", "", nil, dec.Err()
	}

	numPatterns := 0
	hasPatterns, err := dec.List(func() error {
		if numPatterns > 0 {
			return newClientBugError("Only a single LIST pattern can be supplied")
		}
		numPatterns++
		pattern, err = readListMailbox(dec)
		return err
	})
	if err != nil {
		return "", "", nil, err
	} else if !hasPatterns {
		pattern, err = readListMailbox(dec)
		if err != nil {
			return "", "", nil, err
		}
	}

	if dec.SP() { // list-return-opts
		var atom string
		if !dec.ExpectAtom(&atom) || !dec.Expect(atom == "RETURN", "RETURN") || !dec.ExpectSP() {
			return "", "", nil, dec.Err()
		}

		err := dec.ExpectList(func() error {
			return readReturnOption(dec, options)
		})
		if err != nil {
			return "", "", nil, fmt.Errorf("in list-return-opts: %w", err)
		}
	}

	if !dec.ExpectCRLF() {
		return "", "", nil, dec.Err()
	}

	if options.SelectRecursiveMatch && !options.SelectSubscribed {
		return "", "", nil, newClientBugError("The LIST RECURSIVEMATCH select option requires SUBSCRIBED")
	}

	return ref, pattern, options, nil
}

func readListMailbox(dec *imapwire.Decoder) (string, error) {
	var mailbox string
	if dec.String(&mailbox) {
		return mailbox, nil
	}
	if !dec.Expect(dec.Func(&mailbox, isListChar), "list-char") {
		return "", dec.Err()
	}
	return mailbox, nil
}

func isListChar(ch byte) bool {
	switch ch {
	case '%', '*': // list-wildcards
		return true
	case ']': // resp-specials
		return true
	default:
		return imapwire.IsAtomChar(ch)
	}
}

func readReturnOption(dec *imapwire.Decoder, options *imap.ListOptions) error {
	var name string
	if !dec.ExpectAtom(&name) {
		return dec.Err()
	}

	switch name {
	case "SUBSCRIBED":
		options.ReturnSubscribed = true
	case "CHILDREN":
		options.ReturnChildren = true
	case "STATUS":
		if !dec.ExpectSP() {
			return dec.Err()
		}
		return dec.ExpectList(func() error {
			item, err := readStatusItem(dec)
			if err != nil {
				return err
			}
			options.ReturnStatus = append(options.ReturnStatus, item)
			return nil
		})
	default:
		return newClientBugError("Unknown LIST RETURN options")
	}
	return nil
}

// ListWriter writes LIST responses.
type ListWriter struct {
	conn    *Conn
	options *imap.ListOptions
}

// WriteList writes a single LIST response for a mailbox.
func (w *ListWriter) WriteList(data *imap.ListData) error {
	if err := w.conn.writeList(data); err != nil {
		return err
	}
	if len(w.options.ReturnStatus) > 0 && data.Status != nil {
		if err := w.conn.writeStatus(data.Status, w.options.ReturnStatus); err != nil {
			return err
		}
	}
	return nil
}

// MatchList checks whether a reference and a pattern matches a mailbox.
func MatchList(name string, delim rune, reference, pattern string) bool {
	var delimStr string
	if delim != 0 {
		delimStr = string(delim)
	}

	if delimStr != "" && strings.HasPrefix(pattern, delimStr) {
		reference = ""
		pattern = strings.TrimPrefix(pattern, delimStr)
	}
	if reference != "" {
		if delimStr != "" && !strings.HasSuffix(reference, delimStr) {
			reference += delimStr
		}
		if !strings.HasPrefix(name, reference) {
			return false
		}
		name = strings.TrimPrefix(name, reference)
	}

	return matchList(name, delimStr, pattern)
}

func matchList(name, delim, pattern string) bool {
	// TODO: optimize

	i := strings.IndexAny(pattern, "*%")
	if i == -1 {
		// No more wildcards
		return name == pattern
	}

	// Get parts before and after wildcard
	chunk, wildcard, rest := pattern[0:i], pattern[i], pattern[i+1:]

	// Check that name begins with chunk
	if len(chunk) > 0 && !strings.HasPrefix(name, chunk) {
		return false
	}
	name = strings.TrimPrefix(name, chunk)

	// Expand wildcard
	var j int
	for j = 0; j < len(name); j++ {
		if wildcard == '%' && string(name[j]) == delim {
			break // Stop on delimiter if wildcard is %
		}
		// Try to match the rest from here
		if matchList(name[j:], delim, rest) {
			return true
		}
	}

	return matchList(name[j:], delim, rest)
}
