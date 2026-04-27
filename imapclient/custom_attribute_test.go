package imapclient_test

import (
	"reflect"
	"testing"

	"github.com/emersion/go-imap/v2/imapclient"
)

func TestCustomAttribute_Constructors_AndPredicates(t *testing.T) {
	cases := []struct {
		name string
		attr imapclient.CustomAttribute
		want func(imapclient.CustomAttribute) bool
	}{
		{"Number", imapclient.NumberAttribute(42), imapclient.CustomAttribute.IsNumber},
		{"String", imapclient.StringAttribute("x"), imapclient.CustomAttribute.IsString},
		{"List", imapclient.ListAttribute([]imapclient.CustomAttribute{
			imapclient.NumberAttribute(1),
			imapclient.StringAttribute("a"),
		}), imapclient.CustomAttribute.IsList},
		{"NumberList", imapclient.NumberListAttribute([]uint64{1, 2, 3}), imapclient.CustomAttribute.IsNumberList},
		{"StringList", imapclient.StringListAttribute([]string{"a", "b"}), imapclient.CustomAttribute.IsStringList},
		{"Raw", imapclient.RawAttribute(struct{ X int }{X: 1}), imapclient.CustomAttribute.IsRaw},
	}
	predicates := map[string]func(imapclient.CustomAttribute) bool{
		"Number":     imapclient.CustomAttribute.IsNumber,
		"String":     imapclient.CustomAttribute.IsString,
		"List":       imapclient.CustomAttribute.IsList,
		"NumberList": imapclient.CustomAttribute.IsNumberList,
		"StringList": imapclient.CustomAttribute.IsStringList,
		"Raw":        imapclient.CustomAttribute.IsRaw,
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !tc.want(tc.attr) {
				t.Fatalf("predicate Is%s returned false for %+v", tc.name, tc.attr)
			}
			for predName, pred := range predicates {
				if predName == tc.name {
					continue
				}
				if pred(tc.attr) {
					t.Errorf("Is%s = true on %s-typed attribute", predName, tc.name)
				}
			}
		})
	}
}

func TestCustomAttribute_ZeroValue(t *testing.T) {
	var a imapclient.CustomAttribute
	if a.IsNumber() || a.IsString() || a.IsList() || a.IsNumberList() || a.IsStringList() || a.IsRaw() {
		t.Errorf("zero CustomAttribute should not match any Is* predicate")
	}
	if _, ok := a.GetNumber(); ok {
		t.Errorf("zero CustomAttribute GetNumber ok = true, want false")
	}
	if _, ok := a.GetString(); ok {
		t.Errorf("zero CustomAttribute GetString ok = true, want false")
	}
}

func TestCustomAttribute_Getters(t *testing.T) {
	t.Run("Number", func(t *testing.T) {
		got, ok := imapclient.NumberAttribute(42).GetNumber()
		if !ok || got != 42 {
			t.Errorf("GetNumber = (%d, %v), want (42, true)", got, ok)
		}
	})
	t.Run("Number_zero_is_present", func(t *testing.T) {
		// Number=0 is a legal value distinct from "not set"; pointer
		// semantics must report it as present.
		got, ok := imapclient.NumberAttribute(0).GetNumber()
		if !ok || got != 0 {
			t.Errorf("GetNumber = (%d, %v), want (0, true)", got, ok)
		}
	})
	t.Run("String_empty_is_present", func(t *testing.T) {
		got, ok := imapclient.StringAttribute("").GetString()
		if !ok || got != "" {
			t.Errorf("GetString = (%q, %v), want (\"\", true)", got, ok)
		}
	})
	t.Run("List_empty_is_present", func(t *testing.T) {
		got, ok := imapclient.ListAttribute(nil).GetList()
		if !ok || got == nil {
			t.Errorf("GetList = (%v, %v), want (non-nil empty, true)", got, ok)
		}
	})
	t.Run("NumberList", func(t *testing.T) {
		want := []uint64{7, 8, 9}
		got, ok := imapclient.NumberListAttribute(want).GetNumberList()
		if !ok || !reflect.DeepEqual(got, want) {
			t.Errorf("GetNumberList = (%v, %v), want (%v, true)", got, ok, want)
		}
	})
	t.Run("StringList", func(t *testing.T) {
		want := []string{"a", "b"}
		got, ok := imapclient.StringListAttribute(want).GetStringList()
		if !ok || !reflect.DeepEqual(got, want) {
			t.Errorf("GetStringList = (%v, %v), want (%v, true)", got, ok, want)
		}
	})
	t.Run("Raw", func(t *testing.T) {
		got, ok := imapclient.RawAttribute("verbatim").GetRaw()
		if !ok || got != "verbatim" {
			t.Errorf("GetRaw = (%v, %v), want (verbatim, true)", got, ok)
		}
	})
}
