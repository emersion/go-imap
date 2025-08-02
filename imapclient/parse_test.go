package imapclient

import (
	"bufio"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

var bodyStructureCases = []struct {
	description string
	data        string
	parsed      imap.BodyStructure
}{
	{
		description: "example from RFC 3501",
		data:        `(("TEXT" "PLAIN" ("CHARSET" "US-ASCII") NIL NIL "7BIT" 1152 23)("TEXT" "PLAIN" ("CHARSET" "US-ASCII" "NAME" "cc.diff") "<960723163407.20117h@cac.washington.edu>" "Compiler diff" "BASE64" 4554 73) "MIXED") `,
		parsed: &imap.BodyStructureMultiPart{
			Children: []imap.BodyStructure{
				&imap.BodyStructureSinglePart{
					Type:        "TEXT",
					Subtype:     "PLAIN",
					Params:      map[string]string{"charset": "US-ASCII"},
					ID:          "",
					Description: "",
					Encoding:    "7BIT",
					Size:        1152,
					Text: &imap.BodyStructureText{
						NumLines: 23,
					},
				},
				&imap.BodyStructureSinglePart{
					Type:    "TEXT",
					Subtype: "PLAIN",
					Params: map[string]string{
						"charset": "US-ASCII",
						"name":    "cc.diff",
					},
					ID:          "<960723163407.20117h@cac.washington.edu>",
					Description: "Compiler diff",
					Encoding:    "BASE64",
					Size:        4554,
					Text: &imap.BodyStructureText{
						NumLines: 73,
					},
				},
			},
			Subtype: "MIXED",
		},
	},
	{
		description: "issue #678",
		data:        `("text" "html" ("charset" "utf-8") NIL NIL "quoted-printable" 5606 133 NIL NIL NIL NIL)`,
		parsed: &imap.BodyStructureSinglePart{
			Type:        "text",
			Subtype:     "html",
			Params:      map[string]string{"charset": "utf-8"},
			ID:          "",
			Description: "",
			Encoding:    "quoted-printable",
			Size:        5606,
			Text: &imap.BodyStructureText{
				NumLines: 133,
			},
			Extended: &imap.BodyStructureSinglePartExt{},
		},
	},
	{
		description: "message/global attachment in Dovecot",
		data:        `(((("text" "plain" ("charset" "UTF-8") NIL NIL "quoted-printable" 576 31 NIL NIL NIL NIL)("text" "html" ("charset" "UTF-8") NIL NIL "quoted-printable" 4691 112 NIL NIL NIL NIL) "alternative" ("boundary" "----=_NextPart_002_0063_01D85E63.43189F50") NIL NIL NIL)("image" "png" ("name" "image001.png") "<image001.png@01D85E63.36D4F950>" NIL "base64" 3832 NIL NIL NIL NIL) "related" ("boundary" "----=_NextPart_001_0062_01D85E63.43189F50") NIL NIL NIL)("message" "delivery-status" ("name" "details.txt") NIL NIL "7bit" 594 NIL ("attachment" ("filename" "details.txt")) NIL NIL)("message" "global" ("name" "Untitled attachment 00019.dat") NIL NIL "7bit" 6726 NIL ("attachment" ("filename" "Untitled attachment 00019.dat")) NIL NIL) "mixed" ("boundary" "----=_NextPart_000_0061_01D85E63.43189F50") NIL ("en-us") NIL)`,
		parsed: &imap.BodyStructureMultiPart{
			Children: []imap.BodyStructure{
				&imap.BodyStructureMultiPart{
					Children: []imap.BodyStructure{
						&imap.BodyStructureMultiPart{
							Children: []imap.BodyStructure{
								&imap.BodyStructureSinglePart{
									Type:    "text",
									Subtype: "plain",
									Params: map[string]string{
										"charset": "UTF-8",
									},
									ID:          "",
									Description: "",
									Encoding:    "quoted-printable",
									Size:        576,
									Text: &imap.BodyStructureText{
										NumLines: 31,
									},
									Extended: &imap.BodyStructureSinglePartExt{},
								},
								&imap.BodyStructureSinglePart{
									Type:    "text",
									Subtype: "html",
									Params: map[string]string{
										"charset": "UTF-8",
									},
									ID:          "",
									Description: "",
									Encoding:    "quoted-printable",
									Size:        4691,
									Text: &imap.BodyStructureText{
										NumLines: 112,
									},
									Extended: &imap.BodyStructureSinglePartExt{},
								},
							},
							Subtype: "alternative",
							Extended: &imap.BodyStructureMultiPartExt{
								Params: map[string]string{
									"boundary": "----=_NextPart_002_0063_01D85E63.43189F50",
								},
							},
						},
						&imap.BodyStructureSinglePart{
							Type:    "image",
							Subtype: "png",
							Params: map[string]string{
								"name": "image001.png",
							},
							ID:          "<image001.png@01D85E63.36D4F950>",
							Description: "",
							Encoding:    "base64",
							Size:        3832,
							Extended:    &imap.BodyStructureSinglePartExt{},
						},
					},
					Subtype: "related",
					Extended: &imap.BodyStructureMultiPartExt{
						Params: map[string]string{
							"boundary": "----=_NextPart_001_0062_01D85E63.43189F50",
						},
					},
				},
				&imap.BodyStructureSinglePart{
					Type:    "message",
					Subtype: "delivery-status",
					Params: map[string]string{
						"name": "details.txt",
					},
					ID:          "",
					Description: "",
					Encoding:    "7bit",
					Size:        594,
					Extended: &imap.BodyStructureSinglePartExt{
						Disposition: &imap.BodyStructureDisposition{
							Value: "attachment",
							Params: map[string]string{
								"filename": "details.txt",
							},
						},
					},
				},
				&imap.BodyStructureSinglePart{
					Type:    "message",
					Subtype: "global",
					Params: map[string]string{
						"name": "Untitled attachment 00019.dat",
					},
					ID:            "",
					Description:   "",
					Encoding:      "7bit",
					Size:          6726,
					MessageRFC822: nil,
					Extended: &imap.BodyStructureSinglePartExt{
						Disposition: &imap.BodyStructureDisposition{
							Value: "attachment",
							Params: map[string]string{
								"filename": "Untitled attachment 00019.dat",
							},
						},
					},
				},
			},
			Subtype: "mixed",
			Extended: &imap.BodyStructureMultiPartExt{
				Params: map[string]string{
					"boundary": "----=_NextPart_000_0061_01D85E63.43189F50",
				},
				Language: []string{
					"en-us",
				},
			},
		},
	},
}

func TestParseBodyStructure(t *testing.T) {
	for _, c := range bodyStructureCases {
		dec := imapwire.NewDecoder(bufio.NewReader(strings.NewReader(c.data)), imapwire.ConnSideClient)
		s, err := readBody(dec, &Options{})
		if err != nil {
			t.Fatalf("%s: error parsing body structure: %v", c.description, err)
		}
		if !reflect.DeepEqual(s, c.parsed) {
			t.Fatalf("%s: parsed structure doesn't match: want %s, got %s", c.description, toJSON(c.parsed), toJSON(s))
		}
	}
}

func toJSON(v any) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}
