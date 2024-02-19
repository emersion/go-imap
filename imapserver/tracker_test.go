package imapserver_test

import (
	"testing"

	"github.com/opsxolc/go-imap/v2/imapserver"
)

type trackerUpdate struct {
	expunge     uint32
	numMessages uint32
}

var sessionTrackerSeqNumTests = []struct {
	name                       string
	pending                    []trackerUpdate
	clientSeqNum, serverSeqNum uint32
}{
	{
		name:         "noop",
		pending:      nil,
		clientSeqNum: 20,
		serverSeqNum: 20,
	},
	{
		name:         "noop_last",
		pending:      nil,
		clientSeqNum: 42,
		serverSeqNum: 42,
	},
	{
		name:         "noop_client_oob",
		pending:      nil,
		clientSeqNum: 43,
		serverSeqNum: 0,
	},
	{
		name:         "noop_server_oob",
		pending:      nil,
		clientSeqNum: 0,
		serverSeqNum: 43,
	},
	{
		name:         "expunge_eq",
		pending:      []trackerUpdate{{expunge: 20}},
		clientSeqNum: 20,
		serverSeqNum: 0,
	},
	{
		name:         "expunge_lt",
		pending:      []trackerUpdate{{expunge: 20}},
		clientSeqNum: 10,
		serverSeqNum: 10,
	},
	{
		name:         "expunge_gt",
		pending:      []trackerUpdate{{expunge: 10}},
		clientSeqNum: 20,
		serverSeqNum: 19,
	},
	{
		name:         "append_eq",
		pending:      []trackerUpdate{{numMessages: 43}},
		clientSeqNum: 0,
		serverSeqNum: 43,
	},
	{
		name:         "append_lt",
		pending:      []trackerUpdate{{numMessages: 43}},
		clientSeqNum: 42,
		serverSeqNum: 42,
	},
	{
		name: "expunge_append",
		pending: []trackerUpdate{
			{expunge: 42},
			{numMessages: 42},
		},
		clientSeqNum: 42,
		serverSeqNum: 0,
	},
	{
		name: "expunge_append",
		pending: []trackerUpdate{
			{expunge: 42},
			{numMessages: 42},
		},
		clientSeqNum: 0,
		serverSeqNum: 42,
	},
	{
		name: "append_expunge",
		pending: []trackerUpdate{
			{numMessages: 43},
			{expunge: 42},
		},
		clientSeqNum: 42,
		serverSeqNum: 0,
	},
	{
		name: "append_expunge",
		pending: []trackerUpdate{
			{numMessages: 43},
			{expunge: 42},
		},
		clientSeqNum: 0,
		serverSeqNum: 42,
	},
	{
		name: "multi_expunge_middle",
		pending: []trackerUpdate{
			{expunge: 3},
			{expunge: 1},
		},
		clientSeqNum: 2,
		serverSeqNum: 1,
	},
	{
		name: "multi_expunge_after",
		pending: []trackerUpdate{
			{expunge: 3},
			{expunge: 1},
		},
		clientSeqNum: 4,
		serverSeqNum: 2,
	},
}

func TestSessionTracker(t *testing.T) {
	for _, tc := range sessionTrackerSeqNumTests {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			mboxTracker := imapserver.NewMailboxTracker(42)
			sessTracker := mboxTracker.NewSession()
			for _, update := range tc.pending {
				switch {
				case update.expunge != 0:
					mboxTracker.QueueExpunge(update.expunge)
				case update.numMessages != 0:
					mboxTracker.QueueNumMessages(update.numMessages)
				}
			}

			serverSeqNum := sessTracker.DecodeSeqNum(tc.clientSeqNum)
			if tc.clientSeqNum != 0 && serverSeqNum != tc.serverSeqNum {
				t.Errorf("DecodeSeqNum(%v): got %v, want %v", tc.clientSeqNum, serverSeqNum, tc.serverSeqNum)
			}

			clientSeqNum := sessTracker.EncodeSeqNum(tc.serverSeqNum)
			if tc.serverSeqNum != 0 && clientSeqNum != tc.clientSeqNum {
				t.Errorf("EncodeSeqNum(%v): got %v, want %v", tc.serverSeqNum, clientSeqNum, tc.clientSeqNum)
			}
		})
	}
}
