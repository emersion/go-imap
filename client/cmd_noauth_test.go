package client_test

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"testing"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-imap/internal"
	"github.com/emersion/go-sasl"
)

func TestClient_StartTLS(t *testing.T) {
	cert, err := tls.X509KeyPair(internal.LocalhostCert, internal.LocalhostKey)
	if err != nil {
		t.Fatal(err)
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{cert},
	}

	ct := func(c *client.Client) error {
		if c.IsTLS() {
			return fmt.Errorf("Client has TLS enabled before STARTTLS")
		}

		if ok, err := c.SupportStartTLS(); err != nil {
			return err
		} else if !ok {
			return fmt.Errorf("Server doesn't support STARTTLS")
		}

		if err := c.StartTLS(tlsConfig); err != nil {
			return err
		}

		if !c.IsTLS() {
			return fmt.Errorf("Client has not TLS enabled after STARTTLS")
		}

		_, err := c.Capability()
		return err
	}

	st := func(c net.Conn) {
		scanner := NewCmdScanner(c)

		tag, cmd := scanner.Scan()
		if cmd != "STARTTLS" {
			t.Fatal("Bad command:", cmd)
		}

		io.WriteString(c, tag+" OK Begin TLS negotiation now\r\n")

		sc := tls.Server(c, tlsConfig)
		if err = sc.Handshake(); err != nil {
			t.Fatal(err)
		}

		scanner = NewCmdScanner(sc)

		tag, cmd = scanner.Scan()
		if cmd != "CAPABILITY" {
			t.Fatal("Bad command:", cmd)
		}

		io.WriteString(sc, "* CAPABILITY IMAP4rev1 AUTH=PLAIN\r\n")
		io.WriteString(sc, tag+" OK CAPABILITY completed.\r\n")
	}

	testClient(t, ct, st)
}

func TestClient_Authenticate(t *testing.T) {
	ct := func(c *client.Client) error {
		if ok, err := c.SupportAuth(sasl.Plain); err != nil {
			return err
		} else if !ok {
			return fmt.Errorf("Server doesn't support AUTH=PLAIN")
		}

		sasl := sasl.NewPlainClient("", "username", "password")

		if err := c.Authenticate(sasl); err != nil {
			return err
		}

		if c.State != imap.AuthenticatedState {
			return fmt.Errorf("Client is not in authenticated state after AUTENTICATE")
		}

		return nil
	}

	st := func(c net.Conn) {
		scanner := NewCmdScanner(c)

		tag, cmd := scanner.Scan()
		if cmd != "AUTHENTICATE PLAIN" {
			t.Fatal("Bad command:", cmd)
		}

		io.WriteString(c, "+ \r\n")

		line := scanner.ScanLine()
		b, err := base64.StdEncoding.DecodeString(line)
		if err != nil {
			t.Fatal(err)
		}

		parts := bytes.Split(b, []byte("\x00"))
		if string(parts[0]) != "" {
			t.Fatal("Bad identity")
		}
		if string(parts[1]) != "username" {
			t.Fatal("Bad username")
		}
		if string(parts[2]) != "password" {
			t.Fatal("Bad password")
		}

		io.WriteString(c, tag+" OK AUTHENTICATE completed\r\n")
	}

	testClient(t, ct, st)
}

func TestClient_Login_Success(t *testing.T) {
	ct := func(c *client.Client) (err error) {
		err = c.Login("username", "password")
		if err != nil {
			return
		}

		if c.State != imap.AuthenticatedState {
			return fmt.Errorf("Client is not in authenticated state after login")
		}

		return
	}

	st := func(c net.Conn) {
		scanner := NewCmdScanner(c)

		tag, cmd := scanner.Scan()
		if cmd != "LOGIN username password" {
			t.Fatal("Bad command:", cmd)
		}

		io.WriteString(c, tag+" OK LOGIN completed\r\n")
	}

	testClient(t, ct, st)
}

func TestClient_Login_Error(t *testing.T) {
	ct := func(c *client.Client) error {
		err := c.Login("username", "password")
		if err == nil {
			return fmt.Errorf("Failed login didn't returned an error: %v", err)
		}

		if c.State != imap.NotAuthenticatedState {
			return fmt.Errorf("Client state must be NotAuthenticated after failed login, but is: %v", c.State)
		}

		return nil
	}

	st := func(c net.Conn) {
		scanner := NewCmdScanner(c)

		tag, cmd := scanner.Scan()
		if cmd != "LOGIN username password" {
			t.Fatal("Bad command:", cmd)
		}

		io.WriteString(c, tag+" NO LOGIN incorrect\r\n")
	}

	testClient(t, ct, st)
}
