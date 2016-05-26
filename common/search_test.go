package common_test

import (
	"testing"

	"github.com/emersion/go-imap/common"
)

// TODO: improve these tests
func TestSearchCriteria_Format(t *testing.T) {
	seqset, _ := common.NewSeqSet("1:2")

	criteria := &common.SearchCriteria{
		SeqSet: seqset,
		Text: "hello",
	}

	fields := criteria.Format()
	if len(fields) != 3 {
		t.Fatal("Invalid fields list length:", fields)
	}
}
