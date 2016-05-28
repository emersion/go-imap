package common

import (
	"net"
)

// A function that upgrades a connection.
//
// This should only be used by libraries implementing an IMAP extension (e.g.
// COMPRESS).
type ConnUpgrader func(conn net.Conn) (net.Conn, error)
