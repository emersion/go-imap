package internal

import (
	"fmt"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func ReadFlagList(dec *imapwire.Decoder) ([]imap.Flag, error) {
	var flags []imap.Flag
	err := dec.ExpectList(func() error {
		flag, err := ReadFlag(dec)
		if err != nil {
			return err
		}
		flags = append(flags, imap.Flag(flag))
		return nil
	})
	return flags, err
}

func ReadFlag(dec *imapwire.Decoder) (string, error) {
	isSystem := dec.Special('\\')
	if isSystem && dec.Special('*') {
		return "\\*", nil // flag-perm
	}
	var name string
	if !dec.ExpectAtom(&name) {
		return "", fmt.Errorf("in flag: %w", dec.Err())
	}
	if isSystem {
		name = "\\" + name
	}
	return name, nil
}
