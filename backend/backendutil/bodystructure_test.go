package backendutil

import (
	"bufio"
	"reflect"
	"strings"
	"testing"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-message/textproto"
)

var testBodyStructure = &imap.BodyStructure{
	MIMEType:    "multipart",
	MIMESubType: "mixed",
	Params:      map[string]string{"boundary": "message-boundary"},
	Parts: []*imap.BodyStructure{
		{
			MIMEType:    "multipart",
			MIMESubType: "alternative",
			Params:      map[string]string{"boundary": "b2"},
			Extended:    true,
			Parts: []*imap.BodyStructure{
				{
					MIMEType:          "text",
					MIMESubType:       "plain",
					Params:            map[string]string{},
					Extended:          true,
					Disposition:       "inline",
					DispositionParams: map[string]string{},
					Lines:             1,
					Size:              17,
				},
				{
					MIMEType:          "text",
					MIMESubType:       "html",
					Params:            map[string]string{},
					Extended:          true,
					Disposition:       "inline",
					DispositionParams: map[string]string{},
					Lines:             2,
					Size:              37,
				},
			},
		},
		{
			MIMEType:          "text",
			MIMESubType:       "plain",
			Params:            map[string]string{},
			Extended:          true,
			Disposition:       "attachment",
			DispositionParams: map[string]string{"filename": "note.txt"},
			Lines:             1,
			Size:              19,
		},
	},
	Extended: true,
}

func TestFetchBodyStructure(t *testing.T) {
	bufferedBody := bufio.NewReader(strings.NewReader(testMailString))

	header, err := textproto.ReadHeader(bufferedBody)
	if err != nil {
		t.Fatal("Expected no error while reading mail, got:", err)
	}

	bs, err := FetchBodyStructure(header, bufferedBody, true)
	if err != nil {
		t.Fatal("Expected no error while fetching body structure, got:", err)
	}

	if !reflect.DeepEqual(testBodyStructure, bs) {
		t.Errorf("Expected body structure \n%+v\n but got \n%+v", testBodyStructure, bs)
	}
}
