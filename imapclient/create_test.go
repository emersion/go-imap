package imapclient_test

import (
	"testing"

	"github.com/emersion/go-imap/v2"
)

func testCreate(t *testing.T, name string, utf8Accept bool) {
	client, server := newClientServerPair(t, imap.ConnStateAuthenticated)
	defer client.Close()
	defer server.Close()

	if utf8Accept {
		if !client.Caps().Has(imap.CapUTF8Accept) {
			t.Skipf("missing UTF8=ACCEPT support")
		}
		if data, err := client.Enable(imap.CapUTF8Accept).Wait(); err != nil {
			t.Fatalf("Enable(CapUTF8Accept) = %v", err)
		} else if !data.Caps.Has(imap.CapUTF8Accept) {
			t.Fatalf("server refused to enable UTF8=ACCEPT")
		}
	}

	if err := client.Create(name, nil).Wait(); err != nil {
		t.Fatalf("Create() = %v", err)
	}

	listCmd := client.List("", name, nil)
	mailboxes, err := listCmd.Collect()
	if err != nil {
		t.Errorf("List() = %v", err)
	} else if len(mailboxes) != 1 || mailboxes[0].Mailbox != name {
		t.Errorf("List() = %v, want exactly one entry with correct name", mailboxes)
	}
}

func TestCreate(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		testCreate(t, "Test mailbox", false)
	})

	t.Run("unicode_utf7", func(t *testing.T) {
		testCreate(t, "Cafè", false)
	})
	t.Run("unicode_utf8", func(t *testing.T) {
		testCreate(t, "Cafè", true)
	})

	// '&' is the UTF-7 escape character
	t.Run("ampersand_utf7", func(t *testing.T) {
		testCreate(t, "Angus & Julia", false)
	})
	t.Run("ampersand_utf8", func(t *testing.T) {
		testCreate(t, "Angus & Julia", true)
	})
}

func testCreateThenEnable(t *testing.T, name string, capability imap.Cap) {
	client, server := newClientServerPair(t, imap.ConnStateAuthenticated)
	defer client.Close()
	defer server.Close()

	if !client.Caps().Has(capability) {
		t.Skipf("missing %v support", capability)
	}

	if err := client.Create(name, nil).Wait(); err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if data, err := client.Enable(capability).Wait(); err != nil {
		t.Fatalf("Enable(%v) = %v", capability, err)
	} else if !data.Caps.Has(capability) {
		t.Fatalf("server refused to enable %v", capability)
	}

	mailboxes, err := client.List("", name, nil).Collect()
	if err != nil {
		t.Errorf("List() = %v", err)
	} else if len(mailboxes) != 1 || mailboxes[0].Mailbox != name {
		t.Errorf("List() = %v, want exactly one entry with correct name", mailboxes)
	}

	if _, err := client.Select(name, nil).Wait(); err != nil {
		t.Errorf("Select() = %v", err)
	}
}

func TestCreateThenEnable(t *testing.T) {
	t.Run("unicode_utf8", func(t *testing.T) {
		testCreateThenEnable(t, "Cafè", imap.CapUTF8Accept)
	})
	t.Run("ampersand_utf8", func(t *testing.T) {
		testCreateThenEnable(t, "Angus & Julia", imap.CapUTF8Accept)
	})
	t.Run("unicode_imap4rev2", func(t *testing.T) {
		testCreateThenEnable(t, "Cafè", imap.CapIMAP4rev2)
	})
	t.Run("ampersand_imap4rev2", func(t *testing.T) {
		testCreateThenEnable(t, "Angus & Julia", imap.CapIMAP4rev2)
	})
}
