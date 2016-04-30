package main

import (
	"log"

	imap "github.com/emersion/imap/client"
)

func main() {
	log.Println("Connecting to server...")

	c, err := imap.DialTLS("mail.gandi.net:993", nil)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Connected")

	if c.Caps["STARTTLS"] {
		log.Println("Starting TLS")
		c.StartTLS(nil)
	}

	if err := c.Login("username", "password"); err != nil {
		log.Fatal(err)
	}
	log.Println("Logged in")
}
