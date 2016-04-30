package main

import (
	"log"

	"github.com/emersion/imap/client"
	//imap "github.com/emersion/imap/common"
)

func main() {
	log.Println("Connecting to server...")

	c, err := client.DialTLS("mail.gandi.net:993", nil)
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

	/*mailboxes := make(chan *imap.MailboxInfo)
	defer close(mailboxes)

	go (func() {
		for m := range mailboxes {
			log.Println(m)
		}
	})()*/

	mailboxes, err := c.List("", "%")
	log.Println(mailboxes)
}
