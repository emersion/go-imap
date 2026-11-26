package imapclient_test

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/emersion/go-imap/v2/imapserver/imapmemserver"
)

// noQuotaSession hides the QUOTA support of the wrapped session.
type noQuotaSession struct {
	imapserver.SessionIMAP4rev2
}

// newQuotaClientServerPair returns a logged in client for a user with the
// given quota limits and one message in INBOX. If quota is false, the session
// doesn't implement imapserver.SessionQuota.
func newQuotaClientServerPair(t *testing.T, limits map[imap.QuotaResourceType]int64, quota bool) *imapclient.Client {
	memServer := imapmemserver.New()
	user := imapmemserver.NewUser(testUsername, testPassword)
	user.Create("INBOX", nil)
	user.SetQuota(limits)
	memServer.AddUser(user)

	conn, server := newMemServerConn(t, func() imapserver.Session {
		sess := memServer.NewSession()
		if !quota {
			return noQuotaSession{sess.(imapserver.SessionIMAP4rev2)}
		}
		return sess
	})
	t.Cleanup(func() { server.Close() })

	client := imapclient.New(conn, nil)
	t.Cleanup(func() { client.Close() })

	if err := client.Login(testUsername, testPassword).Wait(); err != nil {
		t.Fatalf("Login().Wait() = %v", err)
	}
	appendCmd := client.Append("INBOX", int64(len(simpleRawMessage)), nil)
	appendCmd.Write([]byte(simpleRawMessage))
	appendCmd.Close()
	if _, err := appendCmd.Wait(); err != nil {
		t.Fatalf("AppendCommand.Wait() = %v", err)
	}

	return client
}

var testQuotaLimits = map[imap.QuotaResourceType]int64{
	imap.QuotaResourceStorage: 1024,
	imap.QuotaResourceMessage: 100,
}

// testQuotaData is the quota of a user with testQuotaLimits and one message.
var testQuotaData = imap.QuotaData{
	Root: "",
	Resources: map[imap.QuotaResourceType]imap.QuotaResourceData{
		imap.QuotaResourceStorage: {Usage: 1, Limit: 1024},
		imap.QuotaResourceMessage: {Usage: 1, Limit: 100},
	},
}

func wantIMAPError(t *testing.T, err error, typ imap.StatusResponseType, code imap.ResponseCode) {
	t.Helper()
	var imapErr *imap.Error
	if !errors.As(err, &imapErr) {
		t.Fatalf("got error %v, want IMAP error", err)
	}
	if imapErr.Type != typ || imapErr.Code != code {
		t.Errorf("got %v [%v], want %v [%v]", imapErr.Type, imapErr.Code, typ, code)
	}
}

func TestQuota_capability(t *testing.T) {
	tests := []struct {
		name      string
		quota     bool
		wantQuota bool
		wantTypes []imap.QuotaResourceType
	}{
		{
			name:      "SessionQuota",
			quota:     true,
			wantQuota: true,
			wantTypes: []imap.QuotaResourceType{imap.QuotaResourceMessage, imap.QuotaResourceStorage}, // sorted
		},
		{
			name:  "no SessionQuota",
			quota: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := newQuotaClientServerPair(t, testQuotaLimits, tc.quota)

			caps := client.Caps()
			if got := caps.Has(imap.CapQuota); got != tc.wantQuota {
				t.Errorf("Caps().Has(QUOTA) = %v, want %v", got, tc.wantQuota)
			}
			if caps.Has(imap.CapQuotaSet) {
				t.Errorf("Caps().Has(QUOTASET) = true, want false")
			}
			got := caps.QuotaResourceTypes()
			sort.Slice(got, func(i, j int) bool { return got[i] < got[j] })
			if !reflect.DeepEqual(got, tc.wantTypes) {
				t.Errorf("Caps().QuotaResourceTypes() = %v, want %v", got, tc.wantTypes)
			}

			if !tc.wantQuota {
				_, err := client.GetQuota("").Wait()
				wantIMAPError(t, err, imap.StatusResponseTypeBad, imap.ResponseCodeClientBug)
			}
		})
	}
}

func TestGetQuotaRoot(t *testing.T) {
	client := newQuotaClientServerPair(t, testQuotaLimits, true)

	if err := client.Create("Quota Folder", nil).Wait(); err != nil {
		t.Fatalf("Create() = %v", err)
	}

	for _, mailbox := range []string{"INBOX", "Quota Folder"} {
		t.Run(mailbox, func(t *testing.T) {
			data, err := client.GetQuotaRoot(mailbox).Wait()
			if err != nil {
				t.Fatalf("GetQuotaRoot() = %v", err)
			}
			want := []imapclient.QuotaData{testQuotaData}
			if !reflect.DeepEqual(data, want) {
				t.Errorf("GetQuotaRoot() = %v, want %v", data, want)
			}
		})
	}

	t.Run("nonexistent mailbox", func(t *testing.T) {
		_, err := client.GetQuotaRoot("Nonexistent").Wait()
		wantIMAPError(t, err, imap.StatusResponseTypeNo, imap.ResponseCodeNonExistent)
	})
}

func TestGetQuota(t *testing.T) {
	client := newQuotaClientServerPair(t, testQuotaLimits, true)

	data, err := client.GetQuota("").Wait()
	if err != nil {
		t.Fatalf("GetQuota() = %v", err)
	}
	if !reflect.DeepEqual(data, &testQuotaData) {
		t.Errorf("GetQuota() = %v, want %v", data, &testQuotaData)
	}

	_, err = client.GetQuota("Unknown root").Wait()
	wantIMAPError(t, err, imap.StatusResponseTypeNo, imap.ResponseCodeNonExistent)
}

func TestQuota_unlimited(t *testing.T) {
	t.Run("one resource limited", func(t *testing.T) {
		client := newQuotaClientServerPair(t, map[imap.QuotaResourceType]int64{
			imap.QuotaResourceStorage: 1024,
		}, true)

		want := &imap.QuotaData{
			Root: "",
			Resources: map[imap.QuotaResourceType]imap.QuotaResourceData{
				imap.QuotaResourceStorage: {Usage: 1, Limit: 1024},
			},
		}

		data, err := client.GetQuota("").Wait()
		if err != nil {
			t.Fatalf("GetQuota() = %v", err)
		}
		if !reflect.DeepEqual(data, want) {
			t.Errorf("GetQuota() = %v, want %v", data, want)
		}

		l, err := client.GetQuotaRoot("INBOX").Wait()
		if err != nil {
			t.Fatalf("GetQuotaRoot() = %v", err)
		}
		if len(l) != 1 || !reflect.DeepEqual(&l[0], want) {
			t.Errorf("GetQuotaRoot() = %v, want [%v]", l, want)
		}
	})

	t.Run("no resource limited", func(t *testing.T) {
		client := newQuotaClientServerPair(t, nil, true)

		l, err := client.GetQuotaRoot("INBOX").Wait()
		if err != nil {
			t.Fatalf("GetQuotaRoot() = %v", err)
		}
		if len(l) != 0 {
			t.Errorf("GetQuotaRoot() = %v, want no quota root", l)
		}

		_, err = client.GetQuota("").Wait()
		wantIMAPError(t, err, imap.StatusResponseTypeNo, imap.ResponseCodeNonExistent)
	})
}

func TestSetQuota(t *testing.T) {
	client := newQuotaClientServerPair(t, testQuotaLimits, true)

	limits := map[imap.QuotaResourceType]int64{imap.QuotaResourceStorage: 2048}
	// A non-ASCII root is sent as a literal
	for _, root := range []string{"", "Kvóta"} {
		err := client.SetQuota(root, limits).Wait()
		wantIMAPError(t, err, imap.StatusResponseTypeNo, imap.ResponseCodeNoPerm)
	}

	// The connection is still usable and the quota is unchanged
	data, err := client.GetQuota("").Wait()
	if err != nil {
		t.Fatalf("GetQuota() = %v", err)
	}
	if !reflect.DeepEqual(data, &testQuotaData) {
		t.Errorf("GetQuota() = %v, want %v", data, &testQuotaData)
	}
}

func TestSetQuota_notAuthenticated(t *testing.T) {
	client, server := newClientServerPair(t, imap.ConnStateNotAuthenticated)
	defer client.Close()
	defer server.Close()

	err := client.SetQuota("", map[imap.QuotaResourceType]int64{imap.QuotaResourceStorage: 1}).Wait()
	wantIMAPError(t, err, imap.StatusResponseTypeBad, imap.ResponseCodeClientBug)

	// The connection is not closed, unlike after an unknown command
	if err := client.Login(testUsername, testPassword).Wait(); err != nil {
		t.Errorf("Login().Wait() = %v", err)
	}
}

// TestQuota_literal sends GETQUOTAROOT and SETQUOTA with synchronizing
// literals, which imapclient doesn't use for these commands.
func TestQuota_literal(t *testing.T) {
	memServer := imapmemserver.New()
	user := imapmemserver.NewUser(testUsername, testPassword)
	user.Create("INBOX", nil)
	user.Create("Quota Folder", nil)
	user.SetQuota(testQuotaLimits)
	memServer.AddUser(user)

	conn, server := newMemServerConn(t, memServer.NewSession)
	defer server.Close()
	defer conn.Close()

	br := bufio.NewReader(conn)
	readLine := func() string {
		t.Helper()
		line, err := br.ReadString('\n')
		if err != nil {
			t.Fatalf("ReadString() = %v", err)
		}
		return strings.TrimRight(line, "\r\n")
	}
	write := func(s string) {
		t.Helper()
		if _, err := io.WriteString(conn, s); err != nil {
			t.Fatalf("WriteString() = %v", err)
		}
	}
	// readResponse reads lines until the tagged response, and returns them
	readResponse := func(tag string) []string {
		t.Helper()
		var lines []string
		for {
			line := readLine()
			lines = append(lines, line)
			if strings.HasPrefix(line, tag+" ") {
				return lines
			}
		}
	}
	sendLiteral := func(prefix, lit, suffix string) {
		t.Helper()
		write(fmt.Sprintf("%v{%v}\r\n", prefix, len(lit)))
		if line := readLine(); !strings.HasPrefix(line, "+ ") {
			t.Fatalf("got %q, want continuation request", line)
		}
		write(lit + suffix + "\r\n")
	}

	readLine() // greeting
	write("A1 LOGIN " + testUsername + " " + testPassword + "\r\n")
	if lines := readResponse("A1"); !strings.HasPrefix(lines[len(lines)-1], "A1 OK") {
		t.Fatalf("LOGIN failed: %q", lines)
	}

	sendLiteral("A2 GETQUOTAROOT ", "Quota Folder", "")
	want := []string{
		`* QUOTAROOT "Quota Folder" ""`,
		`* QUOTA "" (STORAGE 0 1024 MESSAGE 0 100)`,
		"A2 OK GETQUOTAROOT completed",
	}
	if got := readResponse("A2"); !reflect.DeepEqual(got, want) {
		t.Errorf("GETQUOTAROOT response = %q, want %q", got, want)
	}

	sendLiteral("A3 SETQUOTA ", "root", " (STORAGE 1)")
	if got := readResponse("A3"); !strings.HasPrefix(got[len(got)-1], "A3 NO [NOPERM]") {
		t.Errorf("SETQUOTA response = %q, want NO [NOPERM]", got)
	}

	// The connection is still in sync
	write("A4 NOOP\r\n")
	if got := readResponse("A4"); !strings.HasPrefix(got[len(got)-1], "A4 OK") {
		t.Errorf("NOOP response = %q, want OK", got)
	}
}
