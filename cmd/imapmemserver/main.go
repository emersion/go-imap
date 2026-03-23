package main

import (
	"crypto/tls"
	"flag"
	"io"
	"log"
	"net"
	"os"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/emersion/go-imap/v2/imapserver/imapmemserver"
)

var (
	listen       string
	tlsCert      string
	tlsKey       string
	username     string
	password     string
	debug        bool
	insecureAuth bool
)

func main() {
	flag.StringVar(&listen, "listen", "localhost:143", "listening address")
	flag.StringVar(&tlsCert, "tls-cert", "", "TLS certificate")
	flag.StringVar(&tlsKey, "tls-key", "", "TLS key")
	flag.StringVar(&username, "username", "user", "Username")
	flag.StringVar(&password, "password", "user", "Password")
	flag.BoolVar(&debug, "debug", false, "Print all commands and responses")
	flag.BoolVar(&insecureAuth, "insecure-auth", false, "Allow authentication without TLS")
	flag.Parse()

	var tlsConfig *tls.Config
	if tlsCert != "" || tlsKey != "" {
		cert, err := tls.LoadX509KeyPair(tlsCert, tlsKey)
		if err != nil {
			log.Fatalf("Failed to load TLS key pair: %v", err)
		}
		tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
	}

	ln, err := net.Listen("tcp", listen)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	log.Printf("IMAP server listening on %v", ln.Addr())

	memServer := imapmemserver.New()

	if username != "" || password != "" {
		user := imapmemserver.NewUser(username, password)

		// Create standard mailboxes with special-use attributes as per RFC 6154
		if err := user.Create("INBOX", nil); err != nil {
			log.Printf("Failed to create INBOX: %v", err)
		}

		if err := user.Create("Drafts", &imap.CreateOptions{
			SpecialUse: []imap.MailboxAttr{imap.MailboxAttrDrafts},
		}); err != nil {
			log.Printf("Failed to create Drafts mailbox: %v", err)
		}

		if err := user.Create("Sent", &imap.CreateOptions{
			SpecialUse: []imap.MailboxAttr{imap.MailboxAttrSent},
		}); err != nil {
			log.Printf("Failed to create Sent mailbox: %v", err)
		}

		if err := user.Create("Archive", &imap.CreateOptions{
			SpecialUse: []imap.MailboxAttr{imap.MailboxAttrArchive},
		}); err != nil {
			log.Printf("Failed to create Archive mailbox: %v", err)
		}

		if err := user.Create("Junk", &imap.CreateOptions{
			SpecialUse: []imap.MailboxAttr{imap.MailboxAttrJunk},
		}); err != nil {
			log.Printf("Failed to create Junk mailbox: %v", err)
		}

		if err := user.Create("Trash", &imap.CreateOptions{
			SpecialUse: []imap.MailboxAttr{imap.MailboxAttrTrash},
		}); err != nil {
			log.Printf("Failed to create Trash mailbox: %v", err)
		}

		if err := user.Create("Flagged", &imap.CreateOptions{
			SpecialUse: []imap.MailboxAttr{imap.MailboxAttrFlagged},
		}); err != nil {
			log.Printf("Failed to create Flagged mailbox: %v", err)
		}

		// Subscribe to the most commonly used mailboxes
		_ = user.Subscribe("INBOX")
		_ = user.Subscribe("Drafts")
		_ = user.Subscribe("Sent")
		_ = user.Subscribe("Trash")

		memServer.AddUser(user)
	}

	var debugWriter io.Writer
	if debug {
		debugWriter = os.Stdout
	}

	server := imapserver.New(&imapserver.Options{
		NewSession: func(conn *imapserver.Conn) (imapserver.Session, *imapserver.GreetingData, error) {
			return memServer.NewSession(), nil, nil
		},
		Caps: imap.CapSet{
			imap.CapIMAP4rev1: {},
			imap.CapIMAP4rev2: {},
		},
		TLSConfig:    tlsConfig,
		InsecureAuth: insecureAuth,
		DebugWriter:  debugWriter,
	})
	if err := server.Serve(ln); err != nil {
		log.Fatalf("Serve() = %v", err)
	}
}
