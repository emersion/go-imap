package backendutil

import (
	"reflect"
	"strings"
	"testing"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-message"
)

var testBodyStructure = &imap.BodyStructure{
	MimeType:    "multipart",
	MimeSubType: "mixed",
	Params:      map[string]string{"boundary": "message-boundary"},
	Parts: []*imap.BodyStructure{
		{
			MimeType:          "text",
			MimeSubType:       "plain",
			Params:            map[string]string{},
			Extended:          true,
			Disposition:       "inline",
			DispositionParams: map[string]string{},
		},
		{
			MimeType:          "text",
			MimeSubType:       "plain",
			Params:            map[string]string{},
			Extended:          true,
			Disposition:       "attachment",
			DispositionParams: map[string]string{"filename": "note.txt"},
		},
	},
	Extended: true,
}

func TestFetchBodyStructure(t *testing.T) {
	e, err := message.Read(strings.NewReader(testMailString))
	if err != nil {
		t.Fatal("Expected no error while reading mail, got:", err)
	}

	bs, err := FetchBodyStructure(e, true)
	if err != nil {
		t.Fatal("Expected no error while fetching body structure, got:", err)
	}

	if !reflect.DeepEqual(testBodyStructure, bs) {
		t.Errorf("Expected body structure \n%+v\n but got \n%+v", testBodyStructure, bs)
	}
}
