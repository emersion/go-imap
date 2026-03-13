package imapclient_test

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"io"
	"net"
	"strings"
	"testing"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/emersion/go-imap/v2/imapserver/imapmemserver"
)

// newCompressClientServerPair creates a client/server pair suitable for COMPRESS testing.
// It does NOT set TLSConfig so that canCompress() returns true on cleartext connections.
func newCompressClientServerPair(t *testing.T, initialState imap.ConnState) (*imapclient.Client, io.Closer) {
	memServer := imapmemserver.New()

	user := imapmemserver.NewUser(testUsername, testPassword)
	user.Create("INBOX", nil)
	memServer.AddUser(user)

	server := imapserver.New(&imapserver.Options{
		NewSession: func(conn *imapserver.Conn) (imapserver.Session, *imapserver.GreetingData, error) {
			return memServer.NewSession(), nil, nil
		},
		// No TLSConfig so canCompress() returns true on cleartext connections
		InsecureAuth: true,
		Caps: imap.CapSet{
			imap.CapIMAP4rev1: {},
			imap.CapIMAP4rev2: {},
			imap.CapCondStore:  {},
			imap.CapQResync:    {},
		},
	})

	ln, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("net.Listen() = %v", err)
	}

	go func() {
		if err := server.Serve(ln); err != nil {
			t.Errorf("Serve() = %v", err)
		}
	}()

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("net.Dial() = %v", err)
	}

	client := imapclient.New(conn, nil)

	if initialState >= imap.ConnStateAuthenticated {
		if err := client.Login(testUsername, testPassword).Wait(); err != nil {
			t.Fatalf("Login().Wait() = %v", err)
		}

		appendCmd := client.Append("INBOX", int64(len(simpleRawMessage)), nil)
		appendCmd.Write([]byte(simpleRawMessage))
		appendCmd.Close()
		if _, err := appendCmd.Wait(); err != nil {
			t.Fatalf("AppendCommand.Wait() = %v", err)
		}
	}
	if initialState >= imap.ConnStateSelected {
		if _, err := client.Select("INBOX", nil).Wait(); err != nil {
			t.Fatalf("Select().Wait() = %v", err)
		}
	}

	return client, server
}

// newCompressTLSClientServerPair creates a TLS client/server pair for testing COMPRESS over TLS.
func newCompressTLSClientServerPair(t *testing.T) (net.Conn, io.Closer) {
	memServer := imapmemserver.New()

	user := imapmemserver.NewUser(testUsername, testPassword)
	user.Create("INBOX", nil)
	memServer.AddUser(user)

	cert, err := tls.X509KeyPair([]byte(rsaCertPEM), []byte(rsaKeyPEM))
	if err != nil {
		t.Fatalf("tls.X509KeyPair() = %v", err)
	}

	server := imapserver.New(&imapserver.Options{
		NewSession: func(conn *imapserver.Conn) (imapserver.Session, *imapserver.GreetingData, error) {
			return memServer.NewSession(), nil, nil
		},
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
		},
		InsecureAuth: true,
		Caps: imap.CapSet{
			imap.CapIMAP4rev1: {},
			imap.CapIMAP4rev2: {},
		},
	})

	ln, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("net.Listen() = %v", err)
	}

	go func() {
		if err := server.Serve(ln); err != nil {
			t.Errorf("Serve() = %v", err)
		}
	}()

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("net.Dial() = %v", err)
	}

	return conn, server
}

func TestCompress(t *testing.T) {
	client, server := newCompressClientServerPair(t, imap.ConnStateAuthenticated)
	defer client.Close()
	defer server.Close()

	if err := client.Compress(nil); err != nil {
		t.Fatalf("Compress() = %v", err)
	}

	// Verify connection still works after compression
	if err := client.Noop().Wait(); err != nil {
		t.Fatalf("Noop() after Compress = %v", err)
	}
}

func TestCompress_ThenSelect(t *testing.T) {
	client, server := newCompressClientServerPair(t, imap.ConnStateAuthenticated)
	defer client.Close()
	defer server.Close()

	if err := client.Compress(nil); err != nil {
		t.Fatalf("Compress() = %v", err)
	}

	data, err := client.Select("INBOX", nil).Wait()
	if err != nil {
		t.Fatalf("Select() after Compress = %v", err)
	}
	if data.NumMessages == 0 {
		t.Errorf("NumMessages = 0, want > 0")
	}
}

func TestCompress_ThenFetch(t *testing.T) {
	client, server := newCompressClientServerPair(t, imap.ConnStateSelected)
	defer client.Close()
	defer server.Close()

	if err := client.Compress(nil); err != nil {
		t.Fatalf("Compress() = %v", err)
	}

	// Fetch envelope
	messages, err := client.Fetch(imap.SeqSetNum(1), &imap.FetchOptions{
		Envelope: true,
	}).Collect()
	if err != nil {
		t.Fatalf("Fetch(Envelope) after Compress = %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("len(messages) = %v, want 1", len(messages))
	}
	if messages[0].Envelope == nil {
		t.Error("Envelope is nil")
	}
}

func TestCompress_ThenFetchBody(t *testing.T) {
	client, server := newCompressClientServerPair(t, imap.ConnStateSelected)
	defer client.Close()
	defer server.Close()

	if err := client.Compress(nil); err != nil {
		t.Fatalf("Compress() = %v", err)
	}

	// Fetch full body section
	messages, err := client.Fetch(imap.SeqSetNum(1), &imap.FetchOptions{
		BodySection: []*imap.FetchItemBodySection{
			{Specifier: imap.PartSpecifierNone, Peek: true},
		},
	}).Collect()
	if err != nil {
		t.Fatalf("Fetch(BodySection) after Compress = %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("len(messages) = %v, want 1", len(messages))
	}

	if len(messages[0].BodySection) == 0 {
		t.Fatal("no body sections returned")
	}
	body := messages[0].BodySection[0].Bytes
	if len(body) == 0 {
		t.Error("body is empty after Compress + Fetch")
	}
	if !strings.Contains(string(body), "This is my letter!") {
		t.Errorf("body does not contain expected text, got: %q", string(body))
	}
}

func TestCompress_ThenFetchFlags(t *testing.T) {
	client, server := newCompressClientServerPair(t, imap.ConnStateSelected)
	defer client.Close()
	defer server.Close()

	if err := client.Compress(nil); err != nil {
		t.Fatalf("Compress() = %v", err)
	}

	messages, err := client.Fetch(imap.SeqSetNum(1), &imap.FetchOptions{
		Flags: true,
		UID:   true,
	}).Collect()
	if err != nil {
		t.Fatalf("Fetch(Flags+UID) after Compress = %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("len(messages) = %v, want 1", len(messages))
	}
	if messages[0].UID == 0 {
		t.Error("UID is 0")
	}
}

func TestCompress_ThenStore(t *testing.T) {
	client, server := newCompressClientServerPair(t, imap.ConnStateSelected)
	defer client.Close()
	defer server.Close()

	if err := client.Compress(nil); err != nil {
		t.Fatalf("Compress() = %v", err)
	}

	// Store flags
	storeFlags := imap.StoreFlags{
		Op:    imap.StoreFlagsAdd,
		Flags: []imap.Flag{imap.FlagSeen},
	}
	results, err := client.Store(imap.SeqSetNum(1), &storeFlags, nil).Collect()
	if err != nil {
		t.Fatalf("Store() after Compress = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %v, want 1", len(results))
	}

	// Verify flags were set
	hasSeen := false
	for _, flag := range results[0].Flags {
		if flag == imap.FlagSeen {
			hasSeen = true
		}
	}
	if !hasSeen {
		t.Error("\\Seen flag not set after Store")
	}
}

func TestCompress_ThenAppend(t *testing.T) {
	client, server := newCompressClientServerPair(t, imap.ConnStateAuthenticated)
	defer client.Close()
	defer server.Close()

	if err := client.Compress(nil); err != nil {
		t.Fatalf("Compress() = %v", err)
	}

	msg := "MIME-Version: 1.0\r\nSubject: compressed append\r\n\r\nBody after compress"
	appendCmd := client.Append("INBOX", int64(len(msg)), nil)
	appendCmd.Write([]byte(msg))
	appendCmd.Close()
	if _, err := appendCmd.Wait(); err != nil {
		t.Fatalf("Append() after Compress = %v", err)
	}

	// Verify message count
	data, err := client.Select("INBOX", nil).Wait()
	if err != nil {
		t.Fatalf("Select() = %v", err)
	}
	// Should have 2 messages: the one added during setup + the one we just appended
	if data.NumMessages != 2 {
		t.Errorf("NumMessages = %v, want 2", data.NumMessages)
	}
}

func TestCompress_ThenSearch(t *testing.T) {
	client, server := newCompressClientServerPair(t, imap.ConnStateSelected)
	defer client.Close()
	defer server.Close()

	if err := client.Compress(nil); err != nil {
		t.Fatalf("Compress() = %v", err)
	}

	criteria := &imap.SearchCriteria{
		Header: []imap.SearchCriteriaHeaderField{
			{Key: "Message-Id", Value: "191101702316132@example.com"},
		},
	}
	results, err := client.Search(criteria, nil).Wait()
	if err != nil {
		t.Fatalf("Search() after Compress = %v", err)
	}
	if len(results.AllSeqNums()) == 0 {
		t.Error("Search returned no results, expected at least 1")
	}
}

func TestCompress_ThenCopy(t *testing.T) {
	client, server := newCompressClientServerPair(t, imap.ConnStateAuthenticated)
	defer client.Close()
	defer server.Close()

	// Create a destination mailbox
	if err := client.Create("Archive", nil).Wait(); err != nil {
		t.Fatalf("Create(Archive) = %v", err)
	}

	if err := client.Compress(nil); err != nil {
		t.Fatalf("Compress() = %v", err)
	}

	// Select INBOX and copy messages
	if _, err := client.Select("INBOX", nil).Wait(); err != nil {
		t.Fatalf("Select(INBOX) = %v", err)
	}

	copyData, err := client.Copy(imap.SeqSetNum(1), "Archive").Wait()
	if err != nil {
		t.Fatalf("Copy() after Compress = %v", err)
	}
	if copyData.UIDValidity == 0 {
		t.Error("CopyData.UIDValidity is 0")
	}
}

func TestCompress_ThenExpunge(t *testing.T) {
	client, server := newCompressClientServerPair(t, imap.ConnStateSelected)
	defer client.Close()
	defer server.Close()

	if err := client.Compress(nil); err != nil {
		t.Fatalf("Compress() = %v", err)
	}

	// Mark message as deleted
	storeFlags := imap.StoreFlags{
		Op:    imap.StoreFlagsAdd,
		Flags: []imap.Flag{imap.FlagDeleted},
	}
	if _, err := client.Store(imap.SeqSetNum(1), &storeFlags, nil).Collect(); err != nil {
		t.Fatalf("Store(+Deleted) = %v", err)
	}

	// Expunge
	if _, err := client.Expunge().Collect(); err != nil {
		t.Fatalf("Expunge() after Compress = %v", err)
	}

	// Verify message count is now 0
	data, err := client.Status("INBOX", &imap.StatusOptions{
		NumMessages: true,
	}).Wait()
	if err != nil {
		t.Fatalf("Status() = %v", err)
	}
	if data.NumMessages != nil && *data.NumMessages != 0 {
		t.Errorf("NumMessages = %v, want 0", *data.NumMessages)
	}
}

func TestCompress_ThenList(t *testing.T) {
	client, server := newCompressClientServerPair(t, imap.ConnStateAuthenticated)
	defer client.Close()
	defer server.Close()

	if err := client.Compress(nil); err != nil {
		t.Fatalf("Compress() = %v", err)
	}

	mailboxes, err := client.List("", "%", nil).Collect()
	if err != nil {
		t.Fatalf("List() after Compress = %v", err)
	}
	if len(mailboxes) == 0 {
		t.Error("List returned no mailboxes")
	}

	found := false
	for _, mb := range mailboxes {
		if mb.Mailbox == "INBOX" {
			found = true
		}
	}
	if !found {
		t.Error("INBOX not found in List results")
	}
}

func TestCompress_ThenStatus(t *testing.T) {
	client, server := newCompressClientServerPair(t, imap.ConnStateAuthenticated)
	defer client.Close()
	defer server.Close()

	if err := client.Compress(nil); err != nil {
		t.Fatalf("Compress() = %v", err)
	}

	data, err := client.Status("INBOX", &imap.StatusOptions{
		NumMessages: true,
		UIDNext:     true,
		UIDValidity: true,
	}).Wait()
	if err != nil {
		t.Fatalf("Status() after Compress = %v", err)
	}
	if data.NumMessages == nil {
		t.Error("NumMessages is nil")
	}
	if data.UIDNext == 0 {
		t.Error("UIDNext is 0")
	}
}

func TestCompress_ThenCreate_Delete(t *testing.T) {
	client, server := newCompressClientServerPair(t, imap.ConnStateAuthenticated)
	defer client.Close()
	defer server.Close()

	if err := client.Compress(nil); err != nil {
		t.Fatalf("Compress() = %v", err)
	}

	// Create
	if err := client.Create("TestFolder", nil).Wait(); err != nil {
		t.Fatalf("Create() after Compress = %v", err)
	}

	// Verify it exists
	mailboxes, err := client.List("", "*", nil).Collect()
	if err != nil {
		t.Fatalf("List() = %v", err)
	}
	found := false
	for _, mb := range mailboxes {
		if mb.Mailbox == "TestFolder" {
			found = true
		}
	}
	if !found {
		t.Error("TestFolder not found after Create")
	}

	// Delete
	if err := client.Delete("TestFolder").Wait(); err != nil {
		t.Fatalf("Delete() after Compress = %v", err)
	}
}

func TestCompress_ThenRename(t *testing.T) {
	client, server := newCompressClientServerPair(t, imap.ConnStateAuthenticated)
	defer client.Close()
	defer server.Close()

	if err := client.Create("OldName", nil).Wait(); err != nil {
		t.Fatalf("Create(OldName) = %v", err)
	}

	if err := client.Compress(nil); err != nil {
		t.Fatalf("Compress() = %v", err)
	}

	if err := client.Rename("OldName", "NewName", nil).Wait(); err != nil {
		t.Fatalf("Rename() after Compress = %v", err)
	}

	// Verify rename
	mailboxes, err := client.List("", "*", nil).Collect()
	if err != nil {
		t.Fatalf("List() = %v", err)
	}
	foundOld := false
	foundNew := false
	for _, mb := range mailboxes {
		if mb.Mailbox == "OldName" {
			foundOld = true
		}
		if mb.Mailbox == "NewName" {
			foundNew = true
		}
	}
	if foundOld {
		t.Error("OldName still exists after Rename")
	}
	if !foundNew {
		t.Error("NewName not found after Rename")
	}
}

func TestCompress_DoubleCompress_Fails(t *testing.T) {
	client, server := newCompressClientServerPair(t, imap.ConnStateAuthenticated)
	defer client.Close()
	defer server.Close()

	if err := client.Compress(nil); err != nil {
		t.Fatalf("First Compress() = %v", err)
	}

	// Second compress should fail
	err := client.Compress(nil)
	if err == nil {
		t.Error("Second Compress() should fail, got nil")
	}
}

func TestCompress_BeforeLogin(t *testing.T) {
	client, server := newCompressClientServerPair(t, imap.ConnStateNotAuthenticated)
	defer client.Close()
	defer server.Close()

	// COMPRESS should work before authentication
	if err := client.Compress(nil); err != nil {
		t.Fatalf("Compress() before login = %v", err)
	}

	// Then login should work over compressed connection
	if err := client.Login(testUsername, testPassword).Wait(); err != nil {
		t.Fatalf("Login() after Compress = %v", err)
	}

	if err := client.Noop().Wait(); err != nil {
		t.Fatalf("Noop() after login over compressed connection = %v", err)
	}
}

func TestCompress_Capability(t *testing.T) {
	client, server := newCompressClientServerPair(t, imap.ConnStateNotAuthenticated)
	defer client.Close()
	defer server.Close()

	// Check COMPRESS=DEFLATE is advertised
	caps := client.Caps()
	algos := caps.CompressAlgorithms()
	found := false
	for _, algo := range algos {
		if algo == "DEFLATE" {
			found = true
		}
	}
	if !found {
		t.Error("COMPRESS=DEFLATE not found in capabilities")
	}
}

func TestCompress_CapabilityDisappearsAfterCompress(t *testing.T) {
	client, server := newCompressClientServerPair(t, imap.ConnStateAuthenticated)
	defer client.Close()
	defer server.Close()

	// Before compress, capability should be present
	capCmd := client.Capability()
	caps, err := capCmd.Wait()
	if err != nil {
		t.Fatalf("Capability() = %v", err)
	}
	algos := caps.CompressAlgorithms()
	if len(algos) == 0 {
		t.Fatal("COMPRESS=DEFLATE not in capabilities before compress")
	}

	// Compress
	if err := client.Compress(nil); err != nil {
		t.Fatalf("Compress() = %v", err)
	}

	// After compress, capability should no longer be present
	capCmd = client.Capability()
	caps, err = capCmd.Wait()
	if err != nil {
		t.Fatalf("Capability() after Compress = %v", err)
	}
	algos = caps.CompressAlgorithms()
	if len(algos) != 0 {
		t.Error("COMPRESS=DEFLATE should not be in capabilities after compress")
	}
}

func TestCompress_ThenSelectAndMultipleFetches(t *testing.T) {
	client, server := newCompressClientServerPair(t, imap.ConnStateAuthenticated)
	defer client.Close()
	defer server.Close()

	// Append multiple messages before compressing
	for i := 0; i < 5; i++ {
		msg := "MIME-Version: 1.0\r\nSubject: test message\r\n\r\nBody content for testing compression"
		appendCmd := client.Append("INBOX", int64(len(msg)), nil)
		appendCmd.Write([]byte(msg))
		appendCmd.Close()
		if _, err := appendCmd.Wait(); err != nil {
			t.Fatalf("Append[%d]() = %v", i, err)
		}
	}

	if err := client.Compress(nil); err != nil {
		t.Fatalf("Compress() = %v", err)
	}

	data, err := client.Select("INBOX", nil).Wait()
	if err != nil {
		t.Fatalf("Select() after Compress = %v", err)
	}
	// 1 from setup + 5 appended = 6
	if data.NumMessages != 6 {
		t.Errorf("NumMessages = %v, want 6", data.NumMessages)
	}

	// Fetch all messages one at a time over compressed connection
	for i := uint32(1); i <= 6; i++ {
		messages, err := client.Fetch(imap.SeqSetNum(i), &imap.FetchOptions{
			Envelope: true,
			Flags:    true,
			UID:      true,
		}).Collect()
		if err != nil {
			t.Fatalf("Fetch(%d) = %v", i, err)
		}
		if len(messages) != 1 {
			t.Fatalf("Fetch(%d): len(messages) = %v, want 1", i, len(messages))
		}
	}

	// Fetch all at once
	messages, err := client.Fetch(imap.SeqSetNum(1, 2, 3, 4, 5, 6), &imap.FetchOptions{
		Envelope: true,
		Flags:    true,
		UID:      true,
	}).Collect()
	if err != nil {
		t.Fatalf("Fetch(1:6) = %v", err)
	}
	if len(messages) != 6 {
		t.Errorf("Fetch(1:6): len(messages) = %v, want 6", len(messages))
	}
}

func TestCompress_ThenIdle(t *testing.T) {
	client, server := newCompressClientServerPair(t, imap.ConnStateSelected)
	defer client.Close()
	defer server.Close()

	if err := client.Compress(nil); err != nil {
		t.Fatalf("Compress() = %v", err)
	}

	// Start IDLE
	idleCmd, err := client.Idle()
	if err != nil {
		t.Fatalf("Idle() after Compress = %v", err)
	}

	// Stop IDLE
	if err := idleCmd.Close(); err != nil {
		t.Fatalf("IdleCommand.Close() = %v", err)
	}
	if err := idleCmd.Wait(); err != nil {
		t.Fatalf("IdleCommand.Wait() = %v", err)
	}

	// Verify connection still works
	if err := client.Noop().Wait(); err != nil {
		t.Fatalf("Noop() after Idle over compressed connection = %v", err)
	}
}

func TestCompress_ThenMove(t *testing.T) {
	client, server := newCompressClientServerPair(t, imap.ConnStateAuthenticated)
	defer client.Close()
	defer server.Close()

	// Create destination
	if err := client.Create("Trash", nil).Wait(); err != nil {
		t.Fatalf("Create(Trash) = %v", err)
	}

	if err := client.Compress(nil); err != nil {
		t.Fatalf("Compress() = %v", err)
	}

	if _, err := client.Select("INBOX", nil).Wait(); err != nil {
		t.Fatalf("Select(INBOX) = %v", err)
	}

	moveData, err := client.Move(imap.SeqSetNum(1), "Trash").Wait()
	if err != nil {
		t.Fatalf("Move() after Compress = %v", err)
	}
	if moveData.UIDValidity == 0 {
		t.Error("MoveData.UIDValidity is 0")
	}
}

func TestCompress_ThenUnselect(t *testing.T) {
	client, server := newCompressClientServerPair(t, imap.ConnStateSelected)
	defer client.Close()
	defer server.Close()

	if err := client.Compress(nil); err != nil {
		t.Fatalf("Compress() = %v", err)
	}

	if err := client.Unselect().Wait(); err != nil {
		t.Fatalf("Unselect() after Compress = %v", err)
	}

	// Re-select should work
	if _, err := client.Select("INBOX", nil).Wait(); err != nil {
		t.Fatalf("Select() after Unselect over compressed = %v", err)
	}
}

func TestCompress_ThenLogout(t *testing.T) {
	client, server := newCompressClientServerPair(t, imap.ConnStateAuthenticated)
	defer server.Close()

	if err := client.Compress(nil); err != nil {
		t.Fatalf("Compress() = %v", err)
	}

	if err := client.Logout().Wait(); err != nil {
		t.Fatalf("Logout() after Compress = %v", err)
	}
	// Close may return an error due to the compressed stream being torn down,
	// which is expected after LOGOUT
	client.Close()
}

func TestCompress_ThenCondStore(t *testing.T) {
	client, server := newCompressClientServerPair(t, imap.ConnStateAuthenticated)
	defer client.Close()
	defer server.Close()

	if err := client.Compress(nil); err != nil {
		t.Fatalf("Compress() = %v", err)
	}

	// SELECT with CONDSTORE over compressed connection
	options := &imap.SelectOptions{CondStore: true}
	data, err := client.Select("INBOX", options).Wait()
	if err != nil {
		t.Fatalf("Select(CONDSTORE) after Compress = %v", err)
	}
	if data.HighestModSeq == 0 {
		t.Error("HighestModSeq is 0")
	}

	// FETCH with MODSEQ over compressed connection
	messages, err := client.Fetch(imap.SeqSetNum(1), &imap.FetchOptions{
		ModSeq: true,
	}).Collect()
	if err != nil {
		t.Fatalf("Fetch(ModSeq) after Compress = %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("len(messages) = %v, want 1", len(messages))
	}
	if messages[0].ModSeq == 0 {
		t.Error("ModSeq is 0")
	}
}

func TestCompress_LargeMessage(t *testing.T) {
	client, server := newCompressClientServerPair(t, imap.ConnStateAuthenticated)
	defer client.Close()
	defer server.Close()

	if err := client.Compress(nil); err != nil {
		t.Fatalf("Compress() = %v", err)
	}

	// Append a large message (should compress well since it's repetitive text)
	body := strings.Repeat("This is a test line for compression testing. ", 1000)
	msg := "MIME-Version: 1.0\r\nSubject: large message\r\nContent-Type: text/plain\r\n\r\n" + body
	appendCmd := client.Append("INBOX", int64(len(msg)), nil)
	appendCmd.Write([]byte(msg))
	appendCmd.Close()
	if _, err := appendCmd.Wait(); err != nil {
		t.Fatalf("Append(large) after Compress = %v", err)
	}

	// Select and fetch it back
	if _, err := client.Select("INBOX", nil).Wait(); err != nil {
		t.Fatalf("Select() = %v", err)
	}

	// Fetch the large message (should be seq 2, since setup added one)
	messages, err := client.Fetch(imap.SeqSetNum(2), &imap.FetchOptions{
		BodySection: []*imap.FetchItemBodySection{
			{Specifier: imap.PartSpecifierNone, Peek: true},
		},
	}).Collect()
	if err != nil {
		t.Fatalf("Fetch(large) = %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("len(messages) = %v, want 1", len(messages))
	}

	if len(messages[0].BodySection) == 0 {
		t.Fatal("no body sections returned")
	}
	data := messages[0].BodySection[0].Bytes
	if !strings.Contains(string(data), "compression testing") {
		t.Error("Large message body doesn't contain expected text")
	}
	if len(data) < len(body) {
		t.Errorf("Body too small: got %d bytes, expected at least %d", len(data), len(body))
	}
}

func TestCompress_FullWorkflow(t *testing.T) {
	// Full email client workflow over compressed connection
	client, server := newCompressClientServerPair(t, imap.ConnStateNotAuthenticated)
	defer client.Close()
	defer server.Close()

	// 1. Login
	if err := client.Login(testUsername, testPassword).Wait(); err != nil {
		t.Fatalf("Login() = %v", err)
	}

	// 2. Compress
	if err := client.Compress(nil); err != nil {
		t.Fatalf("Compress() = %v", err)
	}

	// 3. List mailboxes
	mailboxes, err := client.List("", "*", nil).Collect()
	if err != nil {
		t.Fatalf("List() = %v", err)
	}
	if len(mailboxes) == 0 {
		t.Fatal("No mailboxes found")
	}

	// 4. Create a folder
	if err := client.Create("Drafts", nil).Wait(); err != nil {
		t.Fatalf("Create(Drafts) = %v", err)
	}

	// 5. Append a message
	msg := "MIME-Version: 1.0\r\nSubject: Test\r\n\r\nHello from compressed"
	appendCmd := client.Append("INBOX", int64(len(msg)), nil)
	appendCmd.Write([]byte(msg))
	appendCmd.Close()
	if _, err := appendCmd.Wait(); err != nil {
		t.Fatalf("Append() = %v", err)
	}

	// 6. Select INBOX
	selectData, err := client.Select("INBOX", nil).Wait()
	if err != nil {
		t.Fatalf("Select(INBOX) = %v", err)
	}
	if selectData.NumMessages < 1 {
		t.Fatalf("NumMessages = %v, want >= 1", selectData.NumMessages)
	}

	// 7. Fetch messages
	messages, err := client.Fetch(imap.SeqSetNum(1), &imap.FetchOptions{
		Envelope: true,
		Flags:    true,
		UID:      true,
	}).Collect()
	if err != nil {
		t.Fatalf("Fetch() = %v", err)
	}
	if len(messages) == 0 {
		t.Fatal("No messages fetched")
	}

	// 8. Mark as read
	storeFlags := imap.StoreFlags{
		Op:    imap.StoreFlagsAdd,
		Flags: []imap.Flag{imap.FlagSeen},
	}
	if _, err := client.Store(imap.SeqSetNum(1), &storeFlags, nil).Collect(); err != nil {
		t.Fatalf("Store(+Seen) = %v", err)
	}

	// 9. Copy to Drafts
	if _, err := client.Copy(imap.SeqSetNum(1), "Drafts").Wait(); err != nil {
		t.Fatalf("Copy() = %v", err)
	}

	// 10. Status of Drafts
	draftStatus, err := client.Status("Drafts", &imap.StatusOptions{
		NumMessages: true,
	}).Wait()
	if err != nil {
		t.Fatalf("Status(Drafts) = %v", err)
	}
	if draftStatus.NumMessages == nil || *draftStatus.NumMessages == 0 {
		t.Error("Drafts should have at least 1 message after Copy")
	}

	// 11. IDLE briefly
	idleCmd, err := client.Idle()
	if err != nil {
		t.Fatalf("Idle() = %v", err)
	}
	if err := idleCmd.Close(); err != nil {
		t.Fatalf("Idle.Close() = %v", err)
	}
	if err := idleCmd.Wait(); err != nil {
		t.Fatalf("Idle.Wait() = %v", err)
	}

	// 12. Noop after idle
	if err := client.Noop().Wait(); err != nil {
		t.Fatalf("Noop() = %v", err)
	}

	// 13. Unselect
	if err := client.Unselect().Wait(); err != nil {
		t.Fatalf("Unselect() = %v", err)
	}

	// 14. Delete the folder
	if err := client.Delete("Drafts").Wait(); err != nil {
		t.Fatalf("Delete(Drafts) = %v", err)
	}

	// 15. Logout
	if err := client.Logout().Wait(); err != nil {
		t.Fatalf("Logout() = %v", err)
	}
}

func TestCompress_WithStartTLS(t *testing.T) {
	conn, server := newCompressTLSClientServerPair(t)
	defer conn.Close()
	defer server.Close()

	options := imapclient.Options{
		TLSConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client, err := imapclient.NewStartTLS(conn, &options)
	if err != nil {
		t.Fatalf("NewStartTLS() = %v", err)
	}
	defer client.Close()

	// After STARTTLS, COMPRESS should still work
	if err := client.Compress(nil); err != nil {
		t.Fatalf("Compress() after StartTLS = %v", err)
	}

	// Login over TLS + compressed connection
	if err := client.Login(testUsername, testPassword).Wait(); err != nil {
		t.Fatalf("Login() after TLS+Compress = %v", err)
	}

	// Noop to verify connection
	if err := client.Noop().Wait(); err != nil {
		t.Fatalf("Noop() after TLS+Compress = %v", err)
	}
}

func TestCompress_MultipleNoop(t *testing.T) {
	client, server := newCompressClientServerPair(t, imap.ConnStateAuthenticated)
	defer client.Close()
	defer server.Close()

	if err := client.Compress(nil); err != nil {
		t.Fatalf("Compress() = %v", err)
	}

	// Send many NOOPs to stress-test the compressed connection
	for i := 0; i < 100; i++ {
		if err := client.Noop().Wait(); err != nil {
			t.Fatalf("Noop()[%d] = %v", i, err)
		}
	}
}

func TestCompress_RapidSelectUnselect(t *testing.T) {
	client, server := newCompressClientServerPair(t, imap.ConnStateAuthenticated)
	defer client.Close()
	defer server.Close()

	if err := client.Compress(nil); err != nil {
		t.Fatalf("Compress() = %v", err)
	}

	// Rapidly select and unselect to stress test compressed I/O
	for i := 0; i < 10; i++ {
		if _, err := client.Select("INBOX", nil).Wait(); err != nil {
			t.Fatalf("Select()[%d] = %v", i, err)
		}
		if err := client.Unselect().Wait(); err != nil {
			t.Fatalf("Unselect()[%d] = %v", i, err)
		}
	}
}

func TestCompress_AppendMultipleMessages(t *testing.T) {
	client, server := newCompressClientServerPair(t, imap.ConnStateAuthenticated)
	defer client.Close()
	defer server.Close()

	if err := client.Compress(nil); err != nil {
		t.Fatalf("Compress() = %v", err)
	}

	// Append 20 messages over compressed connection
	for i := 0; i < 20; i++ {
		msg := "MIME-Version: 1.0\r\nSubject: msg\r\n\r\nBody"
		appendCmd := client.Append("INBOX", int64(len(msg)), nil)
		appendCmd.Write([]byte(msg))
		appendCmd.Close()
		if _, err := appendCmd.Wait(); err != nil {
			t.Fatalf("Append[%d]() = %v", i, err)
		}
	}

	// Verify count
	data, err := client.Status("INBOX", &imap.StatusOptions{NumMessages: true}).Wait()
	if err != nil {
		t.Fatalf("Status() = %v", err)
	}
	// 1 from setup + 20 appended = 21
	if data.NumMessages == nil || *data.NumMessages != 21 {
		t.Errorf("NumMessages = %v, want 21", data.NumMessages)
	}
}

func TestCompress_InvalidAlgorithm(t *testing.T) {
	memServer := imapmemserver.New()
	user := imapmemserver.NewUser(testUsername, testPassword)
	user.Create("INBOX", nil)
	memServer.AddUser(user)

	server := imapserver.New(&imapserver.Options{
		NewSession: func(conn *imapserver.Conn) (imapserver.Session, *imapserver.GreetingData, error) {
			return memServer.NewSession(), nil, nil
		},
		InsecureAuth: true,
	})

	ln, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("net.Listen() = %v", err)
	}
	go server.Serve(ln)
	defer server.Close()

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("net.Dial() = %v", err)
	}
	defer conn.Close()

	// Read greeting
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("Read greeting = %v", err)
	}
	greeting := string(buf[:n])
	if !strings.Contains(greeting, "OK") {
		t.Fatalf("unexpected greeting: %s", greeting)
	}

	// Send COMPRESS with unsupported algorithm
	if _, err := conn.Write([]byte("A1 COMPRESS LZO\r\n")); err != nil {
		t.Fatalf("Write COMPRESS LZO = %v", err)
	}

	n, err = conn.Read(buf)
	if err != nil {
		t.Fatalf("Read response = %v", err)
	}
	resp := string(buf[:n])
	if !strings.Contains(resp, "BAD") {
		t.Errorf("expected BAD response for invalid algorithm, got: %s", resp)
	}

	// Connection should still work
	if _, err := conn.Write([]byte("A2 NOOP\r\n")); err != nil {
		t.Fatalf("Write NOOP = %v", err)
	}
	n, err = conn.Read(buf)
	if err != nil {
		t.Fatalf("Read NOOP response = %v", err)
	}
	noopResp := string(buf[:n])
	if !strings.Contains(noopResp, "OK") {
		t.Errorf("NOOP after rejected COMPRESS should succeed, got: %s", noopResp)
	}
}

func TestCompress_CleartextWithTLSConfigured(t *testing.T) {
	// When TLSConfig is set but connection is cleartext, canCompress() returns false
	conn, server := newCompressTLSClientServerPair(t)
	defer conn.Close()
	defer server.Close()

	// Don't upgrade to TLS — just use the raw cleartext connection
	client := imapclient.New(conn, nil)
	defer client.Close()

	// COMPRESS should fail because TLS is configured but we're on cleartext
	err := client.Compress(nil)
	if err == nil {
		t.Fatal("COMPRESS on cleartext with TLS configured should fail, got nil")
	}

	// Connection should still work
	if err := client.Noop().Wait(); err != nil {
		t.Fatalf("Noop() after rejected COMPRESS = %v", err)
	}
}

func TestCompress_ThenUIDFetch(t *testing.T) {
	client, server := newCompressClientServerPair(t, imap.ConnStateSelected)
	defer client.Close()
	defer server.Close()

	if err := client.Compress(nil); err != nil {
		t.Fatalf("Compress() = %v", err)
	}

	// UID FETCH
	messages, err := client.Fetch(imap.UIDSetNum(1), &imap.FetchOptions{
		Envelope: true,
		Flags:    true,
		UID:      true,
	}).Collect()
	if err != nil {
		t.Fatalf("UID Fetch() after Compress = %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("len(messages) = %v, want 1", len(messages))
	}
	if messages[0].UID != 1 {
		t.Errorf("UID = %v, want 1", messages[0].UID)
	}
}

func TestCompress_ThenUIDStore(t *testing.T) {
	client, server := newCompressClientServerPair(t, imap.ConnStateSelected)
	defer client.Close()
	defer server.Close()

	if err := client.Compress(nil); err != nil {
		t.Fatalf("Compress() = %v", err)
	}

	storeFlags := imap.StoreFlags{
		Op:    imap.StoreFlagsAdd,
		Flags: []imap.Flag{imap.FlagFlagged},
	}
	results, err := client.Store(imap.UIDSetNum(1), &storeFlags, nil).Collect()
	if err != nil {
		t.Fatalf("UID Store() after Compress = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %v, want 1", len(results))
	}
	hasFlagged := false
	for _, flag := range results[0].Flags {
		if flag == imap.FlagFlagged {
			hasFlagged = true
		}
	}
	if !hasFlagged {
		t.Error("\\Flagged not set after UID Store")
	}
}

func TestCompress_ThenUIDCopy(t *testing.T) {
	client, server := newCompressClientServerPair(t, imap.ConnStateAuthenticated)
	defer client.Close()
	defer server.Close()

	if err := client.Create("Archive", nil).Wait(); err != nil {
		t.Fatalf("Create(Archive) = %v", err)
	}

	if err := client.Compress(nil); err != nil {
		t.Fatalf("Compress() = %v", err)
	}

	if _, err := client.Select("INBOX", nil).Wait(); err != nil {
		t.Fatalf("Select() = %v", err)
	}

	copyData, err := client.Copy(imap.UIDSetNum(1), "Archive").Wait()
	if err != nil {
		t.Fatalf("UID Copy() after Compress = %v", err)
	}
	if copyData.UIDValidity == 0 {
		t.Error("CopyData.UIDValidity is 0")
	}
}

func TestCompress_ThenExamine(t *testing.T) {
	client, server := newCompressClientServerPair(t, imap.ConnStateAuthenticated)
	defer client.Close()
	defer server.Close()

	if err := client.Compress(nil); err != nil {
		t.Fatalf("Compress() = %v", err)
	}

	// EXAMINE is read-only SELECT
	data, err := client.Select("INBOX", &imap.SelectOptions{ReadOnly: true}).Wait()
	if err != nil {
		t.Fatalf("Examine() after Compress = %v", err)
	}
	if data.NumMessages == 0 {
		t.Error("NumMessages = 0, want > 0")
	}

	// Should be able to fetch but not store
	messages, err := client.Fetch(imap.SeqSetNum(1), &imap.FetchOptions{
		Envelope: true,
	}).Collect()
	if err != nil {
		t.Fatalf("Fetch() after Examine over compressed = %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("len(messages) = %v, want 1", len(messages))
	}
}

func TestCompress_ThenSubscribeUnsubscribe(t *testing.T) {
	client, server := newCompressClientServerPair(t, imap.ConnStateAuthenticated)
	defer client.Close()
	defer server.Close()

	if err := client.Create("Subscribed", nil).Wait(); err != nil {
		t.Fatalf("Create() = %v", err)
	}

	if err := client.Compress(nil); err != nil {
		t.Fatalf("Compress() = %v", err)
	}

	if err := client.Subscribe("Subscribed").Wait(); err != nil {
		t.Fatalf("Subscribe() after Compress = %v", err)
	}

	if err := client.Unsubscribe("Subscribed").Wait(); err != nil {
		t.Fatalf("Unsubscribe() after Compress = %v", err)
	}
}

func TestCompress_NonCompressibleData(t *testing.T) {
	client, server := newCompressClientServerPair(t, imap.ConnStateAuthenticated)
	defer client.Close()
	defer server.Close()

	if err := client.Compress(nil); err != nil {
		t.Fatalf("Compress() = %v", err)
	}

	// Generate random bytes that won't compress well
	randomBytes := make([]byte, 8192)
	if _, err := rand.Read(randomBytes); err != nil {
		t.Fatalf("rand.Read() = %v", err)
	}
	randomBody := base64.StdEncoding.EncodeToString(randomBytes)

	msg := "MIME-Version: 1.0\r\nSubject: binary\r\nContent-Transfer-Encoding: base64\r\n\r\n" + randomBody
	appendCmd := client.Append("INBOX", int64(len(msg)), nil)
	appendCmd.Write([]byte(msg))
	appendCmd.Close()
	if _, err := appendCmd.Wait(); err != nil {
		t.Fatalf("Append(random) after Compress = %v", err)
	}

	// Fetch it back
	if _, err := client.Select("INBOX", nil).Wait(); err != nil {
		t.Fatalf("Select() = %v", err)
	}

	messages, err := client.Fetch(imap.SeqSetNum(2), &imap.FetchOptions{
		BodySection: []*imap.FetchItemBodySection{
			{Specifier: imap.PartSpecifierNone, Peek: true},
		},
	}).Collect()
	if err != nil {
		t.Fatalf("Fetch(random) = %v", err)
	}
	if len(messages) != 1 || len(messages[0].BodySection) == 0 {
		t.Fatal("no body returned")
	}
	body := string(messages[0].BodySection[0].Bytes)
	if !strings.Contains(body, randomBody[:64]) {
		t.Error("Random data not preserved through compress/decompress round-trip")
	}
}

func TestCompress_StartTLS_ThenFullOperations(t *testing.T) {
	conn, server := newCompressTLSClientServerPair(t)
	defer conn.Close()
	defer server.Close()

	options := imapclient.Options{
		TLSConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client, err := imapclient.NewStartTLS(conn, &options)
	if err != nil {
		t.Fatalf("NewStartTLS() = %v", err)
	}
	defer client.Close()

	// Login over TLS
	if err := client.Login(testUsername, testPassword).Wait(); err != nil {
		t.Fatalf("Login() = %v", err)
	}

	// Compress over TLS
	if err := client.Compress(nil); err != nil {
		t.Fatalf("Compress() after StartTLS = %v", err)
	}

	// Append a message over TLS+COMPRESS
	msg := "MIME-Version: 1.0\r\nSubject: TLS compressed\r\n\r\nBody over TLS+DEFLATE"
	appendCmd := client.Append("INBOX", int64(len(msg)), nil)
	appendCmd.Write([]byte(msg))
	appendCmd.Close()
	if _, err := appendCmd.Wait(); err != nil {
		t.Fatalf("Append() over TLS+Compress = %v", err)
	}

	// Select and fetch over TLS+COMPRESS
	data, err := client.Select("INBOX", nil).Wait()
	if err != nil {
		t.Fatalf("Select() over TLS+Compress = %v", err)
	}
	if data.NumMessages == 0 {
		t.Error("NumMessages = 0 after append")
	}

	messages, err := client.Fetch(imap.SeqSetNum(1), &imap.FetchOptions{
		Envelope: true,
		BodySection: []*imap.FetchItemBodySection{
			{Specifier: imap.PartSpecifierNone, Peek: true},
		},
	}).Collect()
	if err != nil {
		t.Fatalf("Fetch() over TLS+Compress = %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("len(messages) = %v, want 1", len(messages))
	}
	if len(messages[0].BodySection) == 0 {
		t.Fatal("no body sections returned")
	}
	if !strings.Contains(string(messages[0].BodySection[0].Bytes), "TLS+DEFLATE") {
		t.Error("body doesn't contain expected text over TLS+Compress")
	}

	// Store flags over TLS+COMPRESS
	storeFlags := imap.StoreFlags{
		Op:    imap.StoreFlagsAdd,
		Flags: []imap.Flag{imap.FlagSeen},
	}
	if _, err := client.Store(imap.SeqSetNum(1), &storeFlags, nil).Collect(); err != nil {
		t.Fatalf("Store() over TLS+Compress = %v", err)
	}

	// List over TLS+COMPRESS
	mailboxes, err := client.List("", "*", nil).Collect()
	if err != nil {
		t.Fatalf("List() over TLS+Compress = %v", err)
	}
	if len(mailboxes) == 0 {
		t.Error("List returned no mailboxes over TLS+Compress")
	}
}

func TestCompress_EmptyMailboxFetch(t *testing.T) {
	client, server := newCompressClientServerPair(t, imap.ConnStateAuthenticated)
	defer client.Close()
	defer server.Close()

	// Create an empty mailbox
	if err := client.Create("Empty", nil).Wait(); err != nil {
		t.Fatalf("Create(Empty) = %v", err)
	}

	if err := client.Compress(nil); err != nil {
		t.Fatalf("Compress() = %v", err)
	}

	// Select the empty mailbox
	data, err := client.Select("Empty", nil).Wait()
	if err != nil {
		t.Fatalf("Select(Empty) after Compress = %v", err)
	}
	if data.NumMessages != 0 {
		t.Errorf("NumMessages = %v, want 0", data.NumMessages)
	}

	// Fetch from empty mailbox — should return 0 messages, no error
	messages, err := client.Fetch(imap.SeqSetNum(1), &imap.FetchOptions{
		Envelope: true,
	}).Collect()
	if err != nil {
		t.Fatalf("Fetch() on empty mailbox after Compress = %v", err)
	}
	if len(messages) != 0 {
		t.Errorf("len(messages) = %v, want 0", len(messages))
	}

	// Status on empty mailbox
	status, err := client.Status("Empty", &imap.StatusOptions{
		NumMessages: true,
		UIDNext:     true,
	}).Wait()
	if err != nil {
		t.Fatalf("Status(Empty) after Compress = %v", err)
	}
	if status.NumMessages != nil && *status.NumMessages != 0 {
		t.Errorf("Status.NumMessages = %v, want 0", *status.NumMessages)
	}

	// Search empty mailbox
	criteria := &imap.SearchCriteria{
		Header: []imap.SearchCriteriaHeaderField{
			{Key: "Subject", Value: "nonexistent"},
		},
	}
	searchResults, err := client.Search(criteria, nil).Wait()
	if err != nil {
		t.Fatalf("Search() on empty mailbox after Compress = %v", err)
	}
	if len(searchResults.AllSeqNums()) != 0 {
		t.Errorf("Search on empty mailbox returned %d results, want 0", len(searchResults.AllSeqNums()))
	}
}

func TestCompress_ThenClose(t *testing.T) {
	client, server := newCompressClientServerPair(t, imap.ConnStateSelected)
	defer client.Close()
	defer server.Close()

	// Mark message as deleted before compressing
	storeFlags := imap.StoreFlags{
		Op:    imap.StoreFlagsAdd,
		Flags: []imap.Flag{imap.FlagDeleted},
	}
	if _, err := client.Store(imap.SeqSetNum(1), &storeFlags, nil).Collect(); err != nil {
		t.Fatalf("Store(+Deleted) = %v", err)
	}

	if err := client.Compress(nil); err != nil {
		t.Fatalf("Compress() = %v", err)
	}

	// CLOSE (UnselectAndExpunge) should implicitly expunge deleted messages
	if err := client.UnselectAndExpunge().Wait(); err != nil {
		t.Fatalf("UnselectAndExpunge() after Compress = %v", err)
	}

	// Re-select and verify message was expunged
	data, err := client.Select("INBOX", nil).Wait()
	if err != nil {
		t.Fatalf("Select() after Close = %v", err)
	}
	if data.NumMessages != 0 {
		t.Errorf("NumMessages = %v after CLOSE, want 0 (message should have been expunged)", data.NumMessages)
	}
}
