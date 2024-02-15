package imapclient_test

import (
	"bufio"
	"fmt"
	"net"
	"testing"

	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func TestMyrights(t *testing.T) {
	type testcaseType struct {
		Mailbox string
		Rights  string
	}

	clientConn, serverConn := net.Pipe()
	client := imapclient.New(clientConn, nil)

	defer client.Close()
	defer serverConn.Close()

	// for server
	serverErrChan := make(chan error)
	testcaseChan := make(chan testcaseType)

	defer close(testcaseChan)

	// for client
	clientErrChan := make(chan error)

	// server
	go func() {
		// send greeting with PREAUTH
		_, err := serverConn.Write([]byte("* PREAUTH BibiServer IMAP4rev1 happy to serve\r\n"))
		if err != nil {
			serverErrChan <- fmt.Errorf("write error: %v", err)
			return
		}

		srvDec := imapwire.NewDecoder(bufio.NewReader(serverConn), imapwire.ConnSideServer)

		// handle requests
		for {
			var tag, cmdName string
			if !srvDec.ExpectAtom(&tag) || !srvDec.ExpectSP() || !srvDec.ExpectAtom(&cmdName) {
				serverErrChan <- fmt.Errorf("error reading cmd: %v, tag=%v, cmdName=%v", srvDec.Err(), tag, cmdName)
				return
			}

			switch cmdName {
			case "CAPABILITY":
				if !srvDec.ExpectCRLF() {
					serverErrChan <- fmt.Errorf("error reading cmd: %v", err)
					return
				}
				_, err := serverConn.Write([]byte(fmt.Sprintf("* CAPABILITY IMAP4rev1 ACL\r\n%v OK CAPABILITY Completed.\r\n", tag)))
				if err != nil {
					serverErrChan <- fmt.Errorf("write error: %v", err)
					return
				}
			case "MYRIGHTS":
				testcase, ok := <-testcaseChan
				if !ok {
					serverErrChan <- nil
					return
				}

				var mailbox string
				if !srvDec.ExpectSP() || !srvDec.ExpectMailbox(&mailbox) || !srvDec.ExpectCRLF() {
					serverErrChan <- fmt.Errorf("error reading cmd: %v", srvDec.Err())
					return
				}

				if mailbox != testcase.Mailbox {
					serverErrChan <- fmt.Errorf("incorrect mailbox: %v - %v", testcase.Mailbox, mailbox)
					return
				}

				_, err := serverConn.Write([]byte(fmt.Sprintf("* MYRIGHTS %v %v\r\n%v OK MYRIGHTS\r\n", testcase.Mailbox, testcase.Rights, tag)))
				if err != nil {
					serverErrChan <- fmt.Errorf("write error: %v", err)
					return
				}
			default:
				serverErrChan <- fmt.Errorf("unsupported cmd: %v", cmdName)
				return
			}
		}
	}()

	if err := client.WaitGreeting(); err != nil {
		t.Fatalf("WaitGreeting: %v", err)
	}

	for _, testcase := range []testcaseType{
		// Testcase 1
		// Common use
		{
			Mailbox: "MyFolder",
			Rights:  "rwiptsldaex",
		},
		// Testcase 2
		// Common use with child
		{
			Mailbox: "MyFolder/Child",
			Rights:  "rwi",
		},
		// Testcase 3
		// Rights with spaces
		{
			Mailbox: "MyFolder",
			Rights:  "r wi p",
		},
	} {
		// execute MYRIGHTS command
		go func() {
			myrightsCmd := client.Myrights(testcase.Mailbox)

			data, err := myrightsCmd.Wait()
			if err != nil {
				clientErrChan <- fmt.Errorf("wait error: %v", err)
				return
			}

			if data.Mailbox != testcase.Mailbox || data.Rights != testcase.Rights {
				clientErrChan <- fmt.Errorf("client received incorrect mailbox or rights: %v - %v, %v - %v",
					testcase.Mailbox, data.Mailbox, testcase.Rights, data.Rights)
				return
			}
			clientErrChan <- nil
		}()

		testcaseChan <- testcase

		select {
		case err := <-serverErrChan:
			t.Fatalf("server error: %v", err)
		case err := <-clientErrChan:
			if err != nil {
				t.Fatalf("client error: %v", err)
			}
		}
	}
}
