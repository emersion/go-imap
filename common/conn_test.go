package common_test

import (
	"bytes"
	"io"
	"net"
	"testing"

	"github.com/emersion/go-imap/common"
)

func TestNewConn(t *testing.T) {
	b := &bytes.Buffer{}
	c, s := net.Pipe()

	done := make(chan error)
	go (func() {
		_, err := io.Copy(b, s)
		done <- err
	})()

	r := common.NewReader(nil)
	w := common.NewWriter(nil)

	ic := common.NewConn(c, r, w)

	sent := []byte("hi")
	ic.Write(sent)
	ic.Flush()
	ic.Close()

	if err := <-done; err != nil {
		t.Fatal(err)
	}

	s.Close()

	received := b.Bytes()
	if string(sent) != string(received) {
		t.Errorf("Sent %v but received %v", sent, received)
	}
}

func transform(b []byte) []byte {
	bb := make([]byte, len(b))

	for i, c := range b {
		if rune(c) == 'c' {
			bb[i] = byte('d')
		} else {
			bb[i] = c
		}
	}

	return bb
}

type upgraded struct {
	net.Conn
}

func (c *upgraded) Write(b []byte) (int, error) {
	return c.Conn.Write(transform(b))
}

func TestConn_Upgrade(t *testing.T) {
	b := &bytes.Buffer{}
	c, s := net.Pipe()

	done := make(chan error)
	go (func() {
		_, err := io.Copy(b, s)
		done <- err
	})()

	r := common.NewReader(nil)
	w := common.NewWriter(nil)

	ic := common.NewConn(c, r, w)

	began := make(chan struct{})
	go ic.Upgrade(func(conn net.Conn) (net.Conn, error) {
		began <- struct{}{}
		return &upgraded{conn}, nil
	})
	<-began

	ic.Wait()

	sent := []byte("abcd")
	expected := transform(sent)
	ic.Write(sent)
	ic.Flush()
	ic.Close()

	if err := <-done; err != nil {
		t.Fatal(err)
	}

	s.Close()

	received := b.Bytes()
	if string(expected) != string(received) {
		t.Errorf("Expected %v but received %v", expected, received)
	}
}
