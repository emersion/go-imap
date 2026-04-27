package imap_test

import (
	"reflect"
	"testing"

	"github.com/emersion/go-imap/v2"
)

func sampleFetchOptions() *imap.FetchOptions {
	return &imap.FetchOptions{
		BodyStructure: &imap.FetchItemBodyStructure{Extended: true},
		Envelope:      true,
		Flags:         true,
		UID:           true,
		ModSeq:        true,
		ChangedSince:  42,
		BodySection: []*imap.FetchItemBodySection{
			{
				Specifier:       imap.PartSpecifierHeader,
				Part:            []int{1, 2},
				HeaderFields:    []string{"Subject", "From"},
				HeaderFieldsNot: []string{"Bcc"},
				Partial:         &imap.SectionPartial{Offset: 0, Size: 1024},
				Peek:            true,
			},
		},
		BinarySection: []*imap.FetchItemBinarySection{
			{Part: []int{3}, Partial: &imap.SectionPartial{Offset: 1, Size: 2}, Peek: true},
		},
		BinarySectionSize: []*imap.FetchItemBinarySectionSize{
			{Part: []int{4, 5}},
		},
		CustomAttributes: []string{"X-GM-MSGID", "X-GM-THRID"},
	}
}

func TestFetchOptions_Clone_Equal(t *testing.T) {
	orig := sampleFetchOptions()
	cloned := orig.Clone()
	if !reflect.DeepEqual(orig, cloned) {
		t.Fatalf("Clone() not deep-equal to original\norig:   %#v\ncloned: %#v", orig, cloned)
	}
}

func TestFetchOptions_Clone_Independent(t *testing.T) {
	orig := sampleFetchOptions()
	cloned := orig.Clone()

	// Mutating the clone must not affect the original. Touch every place
	// where Clone has to allocate to be useful.
	cloned.CustomAttributes[0] = "MUTATED"
	cloned.CustomAttributes = append(cloned.CustomAttributes, "EXTRA")
	cloned.BodyStructure.Extended = false
	cloned.BodySection[0].Part[0] = 999
	cloned.BodySection[0].HeaderFields[0] = "MUTATED"
	cloned.BodySection[0].HeaderFieldsNot[0] = "MUTATED"
	cloned.BodySection[0].Partial.Offset = 999
	cloned.BinarySection[0].Part[0] = 999
	cloned.BinarySection[0].Partial.Offset = 999
	cloned.BinarySectionSize[0].Part[0] = 999
	cloned.ChangedSince = 0

	want := sampleFetchOptions()
	if !reflect.DeepEqual(orig, want) {
		t.Fatalf("mutating clone bled into original\norig: %#v", orig)
	}
}

func TestFetchOptions_Clone_Nil(t *testing.T) {
	var orig *imap.FetchOptions
	if got := orig.Clone(); got != nil {
		t.Errorf("nil.Clone() = %#v, want nil", got)
	}
}

func TestFetchOptions_WithAttributes_AppendsAndDoesNotMutate(t *testing.T) {
	orig := &imap.FetchOptions{
		UID:              true,
		CustomAttributes: []string{"X-GM-MSGID"},
	}
	got := orig.WithAttributes("X-GM-THRID", "X-GM-LABELS")

	wantAttrs := []string{"X-GM-MSGID", "X-GM-THRID", "X-GM-LABELS"}
	if !reflect.DeepEqual(got.CustomAttributes, wantAttrs) {
		t.Errorf("WithAttributes attrs = %v, want %v", got.CustomAttributes, wantAttrs)
	}
	if !got.UID {
		t.Error("WithAttributes dropped UID flag")
	}
	if !reflect.DeepEqual(orig.CustomAttributes, []string{"X-GM-MSGID"}) {
		t.Errorf("WithAttributes mutated original: %v", orig.CustomAttributes)
	}
}

func TestFetchOptions_WithAttributes_NilReceiver(t *testing.T) {
	var orig *imap.FetchOptions
	got := orig.WithAttributes("X-GM-MSGID")
	if got == nil {
		t.Fatal("WithAttributes on nil receiver returned nil")
	}
	if !reflect.DeepEqual(got.CustomAttributes, []string{"X-GM-MSGID"}) {
		t.Errorf("CustomAttributes = %v, want [X-GM-MSGID]", got.CustomAttributes)
	}
}

func TestFetchOptions_WithAttributes_NoAttrs(t *testing.T) {
	orig := sampleFetchOptions()
	got := orig.WithAttributes()
	if !reflect.DeepEqual(got, orig) {
		t.Errorf("WithAttributes() with no args should return a clone equal to original")
	}
	if got == orig {
		t.Errorf("WithAttributes() must return a fresh pointer, not the original")
	}
}

func TestFetchItemBodySection_Clone_Independent(t *testing.T) {
	orig := &imap.FetchItemBodySection{
		Specifier:       imap.PartSpecifierHeader,
		Part:            []int{1, 2},
		HeaderFields:    []string{"a"},
		HeaderFieldsNot: []string{"b"},
		Partial:         &imap.SectionPartial{Offset: 5, Size: 6},
		Peek:            true,
	}
	clone := orig.Clone()
	clone.Part[0] = 99
	clone.HeaderFields[0] = "z"
	clone.HeaderFieldsNot[0] = "z"
	clone.Partial.Offset = 99

	if orig.Part[0] != 1 || orig.HeaderFields[0] != "a" || orig.HeaderFieldsNot[0] != "b" || orig.Partial.Offset != 5 {
		t.Fatalf("Clone bled into original: %+v", orig)
	}
}

func TestFetchItemBodySection_Clone_Nil(t *testing.T) {
	var s *imap.FetchItemBodySection
	if got := s.Clone(); got != nil {
		t.Errorf("(*FetchItemBodySection)(nil).Clone() = %v, want nil", got)
	}
}
