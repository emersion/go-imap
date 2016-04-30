package main

import (
	"log"

	imap "github.com/emersion/imap/client"
)

func main() {
	_, err := imap.DialTLS("mail.gandi.net:993", nil)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Connected!")
}
