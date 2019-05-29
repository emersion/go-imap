package backendutil

import (
	"bufio"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-message/textproto"
)

var bodyTests = []struct {
	section string
	body    string
}{
	{
		section: "BODY[]",
		body:    testMailString,
	},
	{
		section: "BODY[1.1]",
		body:    testTextBodyString,
	},
	{
		section: "BODY[1.2]",
		body:    testHTMLBodyString,
	},
	{
		section: "BODY[2]",
		body:    testAttachmentBodyString,
	},
	{
		section: "BODY[HEADER]",
		body:    testHeaderString,
	},
	{
		section: "BODY[HEADER.FIELDS (From To)]",
		body:    testHeaderFromToString,
	},
	{
		section: "BODY[HEADER.FIELDS (FROM to)]",
		body:    testHeaderFromToString,
	},
	{
		section: "BODY[HEADER.FIELDS.NOT (From To)]",
		body:    testHeaderNoFromToString,
	},
	{
		section: "BODY[HEADER.FIELDS (Date)]",
		body:    testHeaderDateString,
	},
	{
		section: "BODY[1.1.HEADER]",
		body:    testTextHeaderString,
	},
	{
		section: "BODY[1.1.HEADER.FIELDS (Content-Type)]",
		body:    testTextContentTypeString,
	},
	{
		section: "BODY[1.1.HEADER.FIELDS.NOT (Content-Type)]",
		body:    testTextNoContentTypeString,
	},
	{
		section: "BODY[2.HEADER]",
		body:    testAttachmentHeaderString,
	},
	{
		section: "BODY[2.MIME]",
		body:    testAttachmentHeaderString,
	},
	{
		section: "BODY[TEXT]",
		body:    testBodyString,
	},
	{
		section: "BODY[1.1.TEXT]",
		body:    testTextBodyString,
	},
	{
		section: "BODY[2.TEXT]",
		body:    testAttachmentBodyString,
	},
	{
		section: "BODY[2.1]",
		body:    "",
	},
	{
		section: "BODY[3]",
		body:    "",
	},
	{
		section: "BODY[2.TEXT]<0.9>",
		body:    testAttachmentBodyString[:9],
	},
}

func TestFetchBodySection(t *testing.T) {
	for _, test := range bodyTests {
		test := test
		t.Run(test.section, func(t *testing.T) {
			bufferedBody := bufio.NewReader(strings.NewReader(testMailString))

			header, err := textproto.ReadHeader(bufferedBody)
			if err != nil {
				t.Fatal("Expected no error while reading mail, got:", err)
			}

			section, err := imap.ParseBodySectionName(imap.FetchItem(test.section))
			if err != nil {
				t.Fatal("Expected no error while parsing body section name, got:", err)
			}

			r, err := FetchBodySection(header, bufferedBody, section)
			if test.body == "" {
				if err == nil {
					t.Error("Expected an error while extracting non-existing body section")
				}
			} else {
				if err != nil {
					t.Fatal("Expected no error while extracting body section, got:", err)
				}

				b, err := ioutil.ReadAll(r)
				if err != nil {
					t.Fatal("Expected no error while reading body section, got:", err)
				}

				if s := string(b); s != test.body {
					t.Errorf("Expected body section %q to be \n%s\n but got \n%s", test.section, test.body, s)
				}
			}
		})
	}
}
