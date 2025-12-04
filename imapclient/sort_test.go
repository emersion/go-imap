package imapclient_test

import (
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

func TestSort(t *testing.T) {
	// Messages with varying attributes
	// msg1: smallest, oldest arrival, subject "C"
	// msg2: medium, middle arrival, subject "B"
	// msg3: largest, newest arrival, subject "A"
	msg1 := []byte("From: B <b@example.org>\r\nDate: Mon, 2 Jan 2006 15:04:05 -0700\r\nSubject: C\r\n\r\nSmall")
	msg2 := []byte("From: C <c@example.org>\r\nDate: Tue, 3 Jan 2006 15:04:05 -0700\r\nSubject: B\r\n\r\nMedium body")
	msg3 := []byte("From: A <a@example.org>\r\nDate: Wed, 4 Jan 2006 15:04:05 -0700\r\nSubject: A\r\n\r\nMuch larger body content here")

	client, server := newClientServerPair(t, imap.ConnStateSelected)

	defer client.Close()
	defer server.Close()

	const mboxName = "SORT-TEST"
	if err := client.Create(mboxName, nil).Wait(); err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	if _, err := client.Select(mboxName, nil).Wait(); err != nil {
		t.Fatalf("Select() failed: %v", err)
	}

	appendMsg := func(msgData []byte) {
		cmd := client.Append(mboxName, int64(len(msgData)), nil)
		if _, err := cmd.Write(msgData); err != nil {
			t.Fatalf("Append.Write() failed: %v", err)
		}
		if err := cmd.Close(); err != nil {
			t.Fatalf("Append.Close() failed: %v", err)
		}
		if _, err := cmd.Wait(); err != nil {
			t.Fatalf("Append.Wait() failed: %v", err)
		}
	}

	// Append messages, ensuring different arrival times
	appendMsg(msg1)
	time.Sleep(10 * time.Millisecond)
	appendMsg(msg2)
	time.Sleep(10 * time.Millisecond)
	appendMsg(msg3)

	allMsgs := &imap.SearchCriteria{}

	t.Run("SORT", func(t *testing.T) {
		tests := []struct {
			name    string
			options *imapclient.SortOptions
			want    []uint32
		}{
			{
				name: "ARRIVAL",
				options: &imapclient.SortOptions{
					SortCriteria:   []imap.SortCriterion{{Key: imap.SortKeyArrival}},
					SearchCriteria: allMsgs,
				},
				want: []uint32{1, 2, 3},
			},
			{
				name: "REVERSE SUBJECT",
				options: &imapclient.SortOptions{
					SortCriteria:   []imap.SortCriterion{{Key: imap.SortKeySubject, Reverse: true}},
					SearchCriteria: allMsgs,
				},
				want: []uint32{1, 2, 3}, // C, B, A (reversed)
			},
			{
				name: "SIZE",
				options: &imapclient.SortOptions{
					SortCriteria:   []imap.SortCriterion{{Key: imap.SortKeySize}},
					SearchCriteria: allMsgs,
				},
				want: []uint32{1, 2, 3},
			},
			{
				name: "SUBJECT",
				options: &imapclient.SortOptions{
					SortCriteria:   []imap.SortCriterion{{Key: imap.SortKeySubject}},
					SearchCriteria: allMsgs,
				},
				want: []uint32{3, 2, 1}, // A, B, C
			},
			{
				name: "DISPLAY",
				options: &imapclient.SortOptions{
					SortCriteria:   []imap.SortCriterion{{Key: imap.SortKeyDisplay}},
					SearchCriteria: allMsgs,
				},
				want: []uint32{3, 1, 2}, // A, B, C
			},
			{
				name: "FROM",
				options: &imapclient.SortOptions{
					SortCriteria:   []imap.SortCriterion{{Key: imap.SortKeyFrom}},
					SearchCriteria: allMsgs,
				},
				want: []uint32{3, 1, 2}, // a@, b@, c@
			},
			{
				name: "DATE then SUBJECT",
				options: &imapclient.SortOptions{
					SortCriteria: []imap.SortCriterion{
						{Key: imap.SortKeyDate},
						{Key: imap.SortKeySubject}, // tie-breaker, but not needed here
					},
					SearchCriteria: allMsgs,
				},
				want: []uint32{1, 2, 3},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Skip DISPLAY test for Dovecot as it doesn't support SORT=DISPLAY
				if tt.name == "DISPLAY" && os.Getenv("GOIMAP_TEST_DOVECOT") == "1" {
					t.Skip("Dovecot doesn't support SORT=DISPLAY")
				}

				data, err := client.Sort(tt.options).Wait()
				if err != nil {
					t.Fatalf("Sort() failed: %v", err)
				}
				if !reflect.DeepEqual(data.SeqNums, tt.want) {
					t.Errorf("Sort() SeqNums = %v, want %v", data.SeqNums, tt.want)
				}
				if len(data.UIDs) > 0 {
					t.Errorf("Sort() returned UIDs unexpectedly: %v", data.UIDs)
				}
			})
		}
	})

	t.Run("UID SORT", func(t *testing.T) {
		// Use SUBJECT for sorting as it's more deterministic than ARRIVAL across servers
		options := &imapclient.SortOptions{
			SortCriteria:   []imap.SortCriterion{{Key: imap.SortKeySubject}},
			SearchCriteria: allMsgs,
		}
		data, err := client.UIDSort(options).Wait()
		if err != nil {
			t.Fatalf("UIDSort() failed: %v", err)
		}

		// Subjects are A, B, C -> UIDs 3, 2, 1
		wantUIDs := []imap.UID{3, 2, 1}
		if !reflect.DeepEqual(data.UIDs, wantUIDs) {
			t.Errorf("UIDSort() UIDs = %v, want %v", data.UIDs, wantUIDs)
		}
		if len(data.SeqNums) > 0 {
			t.Errorf("UIDSort() returned SeqNums unexpectedly: %v", data.SeqNums)
		}
	})

	t.Run("ESORT", func(t *testing.T) {
		// Skip for Dovecot as it doesn't support ESORT extension
		if os.Getenv("GOIMAP_TEST_DOVECOT") == "1" {
			t.Skip("Dovecot doesn't support ESORT")
		}

		options := &imapclient.SortOptions{
			SortCriteria:   []imap.SortCriterion{{Key: imap.SortKeyArrival}},
			SearchCriteria: allMsgs,
			Return: imap.SortOptions{
				ReturnCount: true,
				ReturnMin:   true,
				ReturnMax:   true,
				ReturnAll:   true,
			},
		}
		data, err := client.Sort(options).Wait()
		if err != nil {
			t.Fatalf("ESort() failed: %v", err)
		}

		if data.Count != 3 {
			t.Errorf("ESort() Count = %v, want 3", data.Count)
		}
		if data.Min != 1 {
			t.Errorf("ESort() Min = %v, want 1", data.Min)
		}
		if data.Max != 3 {
			t.Errorf("ESort() Max = %v, want 3", data.Max)
		}
		wantSeqNums := []uint32{1, 2, 3}
		if !reflect.DeepEqual(data.SeqNums, wantSeqNums) {
			t.Errorf("ESort() SeqNums = %v, want %v", data.SeqNums, wantSeqNums)
		}
	})

	t.Run("Empty result", func(t *testing.T) {
		// Search for a non-existent message
		options := &imapclient.SortOptions{
			SortCriteria: []imap.SortCriterion{{Key: imap.SortKeyArrival}},
			SearchCriteria: &imap.SearchCriteria{
				Header: []imap.SearchCriteriaHeaderField{{Key: "Subject", Value: "non-existent"}},
			},
			Return: imap.SortOptions{
				ReturnCount: true,
				ReturnAll:   true,
			},
		}

		data, err := client.Sort(options).Wait()
		if err != nil {
			t.Fatalf("Sort() with empty result failed: %v", err)
		}

		if data.Count != 0 {
			t.Errorf("Count = %v, want 0", data.Count)
		}
		if len(data.SeqNums) != 0 {
			t.Errorf("SeqNums = %v, want empty", data.SeqNums)
		}
	})

	// RFC 5267 compliance tests:
	// - Empty RETURN list defaults to ALL
	// - Unknown RETURN options are silently ignored
	// These are tested at the server/protocol level in imapserver package
}
