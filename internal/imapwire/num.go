package imapwire

import (
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapnum"
)

type NumKind int

const (
	NumKindSeq NumKind = iota + 1
	NumKindUID
)

func NumSetKind(numSet imap.NumSet) NumKind {
	switch numSet.(type) {
	case imap.SeqSet:
		return NumKindSeq
	case imap.UIDSet:
		return NumKindUID
	default:
		panic("imap: invalid NumSet type")
	}
}

func ParseSeqSet(s string) (imap.SeqSet, error) {
	set, err := imapnum.ParseSet[uint32](s)
	if err != nil {
		return nil, err
	}
	return imap.SeqSet(set), nil
}
