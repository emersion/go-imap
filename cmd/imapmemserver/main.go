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
	flag.StringVar(&username, "password", "user", "Password")
	flag.BoolVar(&debug, "debug", false, "Print all commands and responses")
	flag.BoolVar(&insecureAuth, "insecure-auth", false, "Allow authentication without TLS")
	flag.Parse()

	var tlsConfig *tls.Config
	if tlsCert != "" || tlsKey != "" {
		cert, err := tls.LoadX509KeyPair("../tlstunnel/cert.pem", "../tlstunnel/key.pem")
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
		user.Create("INBOX")
		memServer.AddUser(user)
	}

	var debugWriter io.Writer
	if debug {
		debugWriter = os.Stdout
	}

	server := imapserver.New(&imapserver.Options{
		NewSession: func(conn *imapserver.Conn) (imapserver.Session, error) {
			return memServer.NewSession(), nil
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
