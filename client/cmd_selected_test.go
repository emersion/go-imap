package client_test

import (
	"io"
	//"fmt"
	"net"
	"testing"

	"github.com/emersion/imap/common"
	"github.com/emersion/imap/client"
)

func TestClient_Check(t *testing.T) {
	ct := func(c *client.Client) (err error) {
		c.State = common.SelectedState

		err = c.Check()
		return
	}

	st := func(c net.Conn) {
		scanner := NewCmdScanner(c)

		tag, cmd := scanner.Scan()
		if cmd != "CHECK" {
			t.Fatal("Bad command:", cmd)
		}

		io.WriteString(c, tag + " OK CHECK completed\r\n")
	}

	testClient(t, ct, st)
}
