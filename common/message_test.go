package common_test

import (
	"testing"
	"time"

	"github.com/emersion/imap/common"
)

func TestParseDate(t *testing.T) {
	tests := []struct {
		dateStr string
		exp     time.Time
	}{
		{
			"21-Nov-1997 09:55:06 -0600",
			time.Date(1997, 11, 21, 9, 55, 6, 0, time.FixedZone("", -6*60*60)),
		},
	}
	for _, test := range tests {
		date, err := common.ParseDate(test.dateStr)
		if err != nil {
			t.Errorf("Failed parsing %q: %v", test.dateStr, err)
			continue
		}
		if !date.Equal(test.exp) {
			t.Errorf("Parse of %q: got %+v, want %+v", test.dateStr, date, test.exp)
		}
	}
}

func TestAddress_Parse(t *testing.T) {
	addr := &common.Address{}
	fields := []interface{}{"The NSA", nil, "root", "nsa.gov"}

	if err := addr.Parse(fields); err != nil {
		t.Fatal(err)
	}

	if addr.PersonalName != "The NSA" {
		t.Error("Invalid personal name:", addr.PersonalName)
	}
	if addr.AtDomainList != "" {
		t.Error("Invalid at-domain-list:", addr.AtDomainList)
	}
	if addr.MailboxName != "root" {
		t.Error("Invalid mailbox name:", addr.MailboxName)
	}
	if addr.HostName != "nsa.gov" {
		t.Error("Invalid host name:", addr.HostName)
	}
}

func TestAddress_Format(t *testing.T) {
	addr := &common.Address{
		PersonalName: "The NSA",
		MailboxName: "root",
		HostName: "nsa.gov",
	}

	fields := addr.Format()
	if len(fields) != 4 {
		t.Fatal("Invalid fields list length")
	}
	if fields[0] != "The NSA" {
		t.Error("Invalid personal name:", fields[0])
	}
	if fields[1] != nil {
		t.Error("Invalid at-domain-list:", fields[1])
	}
	if fields[2] != "root" {
		t.Error("Invalid mailbox name:", fields[2])
	}
	if fields[3] != "nsa.gov" {
		t.Error("Invalid host name:", fields[3])
	}
}
