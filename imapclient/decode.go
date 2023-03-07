package imapclient

import (
	"fmt"

	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func readCapabilities(dec *imapwire.Decoder) (map[string]struct{}, error) {
	caps := make(map[string]struct{})
	for dec.SP() {
		var name string
		if !dec.ExpectAtom(&name) {
			return caps, fmt.Errorf("in capability-data: %v", dec.Err())
		}
		caps[name] = struct{}{}
	}
	return caps, nil
}

func readFlagList(dec *imapwire.Decoder) ([]string, error) {
	if !dec.ExpectSpecial('(') {
		return nil, dec.Err()
	}
	if dec.Special(')') {
		return nil, nil
	}

	flag, err := readFlag(dec)
	if err != nil {
		return nil, err
	}

	flags := []string{flag}
	for dec.SP() {
		flag, err := readFlag(dec)
		if err != nil {
			return flags, err
		}
		flags = append(flags, flag)
	}

	if !dec.ExpectSpecial(')') {
		return flags, dec.Err()
	}
	return flags, nil
}

func readFlag(dec *imapwire.Decoder) (string, error) {
	isSystem := dec.Special('\\')
	var name string
	if !dec.ExpectAtom(&name) {
		return "", fmt.Errorf("in flag: %v", dec.Err())
	}
	if isSystem {
		name = "\\" + name
	}
	return name, nil
}
