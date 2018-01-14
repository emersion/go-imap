package client

import (
	"testing"

	"github.com/emersion/go-imap"
)

func TestClient_Capability(t *testing.T) {
	c, s := newTestClient(t)
	defer s.Close()

	var caps map[string]bool
	done := make(chan error, 1)
	go func() {
		var err error
		caps, err = c.Capability()
		done <- err
	}()

	tag, cmd := s.ScanCmd()
	if cmd != "CAPABILITY" {
		t.Fatalf("client sent command %v, want CAPABILITY", cmd)
	}
	s.WriteString("* CAPABILITY IMAP4rev1 XTEST\r\n")
	s.WriteString(tag + " OK CAPABILITY completed.\r\n")

	if err := <-done; err != nil {
		t.Error("c.Capability() = ", err)
	}

	if !caps["XTEST"] {
		t.Error("XTEST capability missing")
	}
}

func TestClient_Noop(t *testing.T) {
	c, s := newTestClient(t)
	defer s.Close()

	done := make(chan error, 1)
	go func() {
		done <- c.Noop()
	}()

	tag, cmd := s.ScanCmd()
	if cmd != "NOOP" {
		t.Fatalf("client sent command %v, want NOOP", cmd)
	}
	s.WriteString(tag + " OK NOOP completed\r\n")

	if err := <-done; err != nil {
		t.Error("c.Noop() = ", err)
	}
}

func TestClient_Logout(t *testing.T) {
	c, s := newTestClient(t)
	defer s.Close()

	done := make(chan error, 1)
	go func() {
		done <- c.Logout()
	}()

	tag, cmd := s.ScanCmd()
	if cmd != "LOGOUT" {
		t.Fatalf("client sent command %v, want LOGOUT", cmd)
	}
	s.WriteString("* BYE Client asked to close the connection.\r\n")
	s.WriteString(tag + " OK LOGOUT completed\r\n")

	if err := <-done; err != nil {
		t.Error("c.Logout() =", err)
	}

	if state := c.State(); state != imap.LogoutState {
		t.Errorf("c.State() = %v, want %v", state, imap.LogoutState)
	}
}
