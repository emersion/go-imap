# imap

An IMAP library written in Go.

## Why?

Other IMAP implementations in Go:
* Require to make many type assertions
* Are not idiomatic

## Usage

```go
import (
	"log"

	"github.com/emersion/imap/client"
	imap "github.com/emersion/imap/common"
)

func main() {
	log.Println("Connecting to server...")

	c, err := client.DialTLS("mail.example.org:993", nil)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Connected")

	if err := c.Login("username", "password"); err != nil {
		log.Fatal(err)
	}
	log.Println("Logged in")

	mailboxes := make(chan *imap.MailboxInfo)
	go (func () {
		err = c.List("", "%", mailboxes)
	})()

	log.Println("Mailboxes:")
	for m := range mailboxes {
		log.Println(m.Name)
	}

	if err != nil {
		log.Fatal(err)
	}

	log.Println("Done")
}
```

## License

MIT
