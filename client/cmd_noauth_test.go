package client

import (
	"crypto/tls"
	"io"
	"testing"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/internal"
	"github.com/emersion/go-sasl"
)

func TestClient_StartTLS(t *testing.T) {
	c, s := newTestClient(t)
	defer s.Close()

	cert, err := tls.X509KeyPair(internal.LocalhostCert, internal.LocalhostKey)
	if err != nil {
		t.Fatal("cannot load test certificate:", err)
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{cert},
	}

	if c.IsTLS() {
		t.Fatal("Client has TLS enabled before STARTTLS")
	}

	if ok, err := c.SupportStartTLS(); err != nil {
		t.Fatalf("c.SupportStartTLS() = %v", err)
	} else if !ok {
		t.Fatalf("c.SupportStartTLS() = %v, want true", ok)
	}

	done := make(chan error, 1)
	go func() {
		done <- c.StartTLS(tlsConfig)
	}()

	tag, cmd := s.ScanCmd()
	if cmd != "STARTTLS" {
		t.Fatalf("client sent command %v, want STARTTLS", cmd)
	}
	s.WriteString(tag + " OK Begin TLS negotiation now\r\n")

	ss := tls.Server(s.Conn, tlsConfig)
	if err := ss.Handshake(); err != nil {
		t.Fatal("cannot perform TLS handshake:", err)
	}

	if err := <-done; err != nil {
		t.Error("c.StartTLS() =", err)
	}

	if !c.IsTLS() {
		t.Errorf("Client has not TLS enabled after STARTTLS")
	}

	go func() {
		_, err := c.Capability()
		done <- err
	}()

	tag, cmd = newCmdScanner(ss).ScanCmd()
	if cmd != "CAPABILITY" {
		t.Fatalf("client sent command %v, want CAPABILITY", cmd)
	}
	io.WriteString(ss, "* CAPABILITY IMAP4rev1 AUTH=PLAIN\r\n")
	io.WriteString(ss, tag+" OK CAPABILITY completed.\r\n")
}

func TestClient_Authenticate(t *testing.T) {
	c, s := newTestClient(t)
	defer s.Close()

	if ok, err := c.SupportAuth(sasl.Plain); err != nil {
		t.Fatalf("c.SupportAuth(sasl.Plain) = %v", err)
	} else if !ok {
		t.Fatalf("c.SupportAuth(sasl.Plain) = %v, want true", ok)
	}

	sasl := sasl.NewPlainClient("", "username", "password")

	done := make(chan error, 1)
	go func() {
		done <- c.Authenticate(sasl)
	}()

	tag, cmd := s.ScanCmd()
	if cmd != "AUTHENTICATE PLAIN" {
		t.Fatalf("client sent command %v, want AUTHENTICATE PLAIN", cmd)
	}

	s.WriteString("+ \r\n")

	wantLine := "AHVzZXJuYW1lAHBhc3N3b3Jk"
	if line := s.ScanLine(); line != wantLine {
		t.Fatalf("client sent auth %v, want %v", line, wantLine)
	}

	s.WriteString(tag + " OK AUTHENTICATE completed\r\n")

	if err := <-done; err != nil {
		t.Fatalf("c.Authenticate() = %v", err)
	}

	if state := c.State(); state != imap.AuthenticatedState {
		t.Errorf("c.State() = %v, want %v", state, imap.AuthenticatedState)
	}
}

func TestClient_Login_Success(t *testing.T) {
	c, s := newTestClient(t)
	defer s.Close()

	done := make(chan error, 1)
	go func() {
		done <- c.Login("username", "password")
	}()

	tag, cmd := s.ScanCmd()
	if cmd != "LOGIN username password" {
		t.Fatalf("client sent command %v, want LOGIN username password", cmd)
	}
	s.WriteString(tag + " OK LOGIN completed\r\n")

	if err := <-done; err != nil {
		t.Fatalf("c.Login() = %v", err)
	}

	if state := c.State(); state != imap.AuthenticatedState {
		t.Errorf("c.State() = %v, want %v", state, imap.AuthenticatedState)
	}
}

func TestClient_Login_Error(t *testing.T) {
	c, s := newTestClient(t)
	defer s.Close()

	done := make(chan error, 1)
	go func() {
		done <- c.Login("username", "password")
	}()

	tag, cmd := s.ScanCmd()
	if cmd != "LOGIN username password" {
		t.Fatalf("client sent command %v, want LOGIN username password", cmd)
	}
	s.WriteString(tag + " NO LOGIN incorrect\r\n")

	if err := <-done; err == nil {
		t.Fatal("c.Login() = nil, want LOGIN incorrect")
	}

	if state := c.State(); state != imap.NotAuthenticatedState {
		t.Errorf("c.State() = %v, want %v", state, imap.NotAuthenticatedState)
	}
}
