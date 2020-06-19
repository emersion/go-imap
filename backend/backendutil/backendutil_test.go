package backendutil

import (
	"time"
)

var testDate, _ = time.Parse(time.RFC1123Z, "Sat, 18 Jun 2016 12:00:00 +0900")

const testHeaderString = "Content-Type: multipart/mixed; boundary=message-boundary\r\n" +
	"Date: Sat, 18 Jun 2016 12:00:00 +0900\r\n" +
	"Date: Sat, 19 Jun 2016 12:00:00 +0900\r\n" +
	"From: Mitsuha Miyamizu <mitsuha.miyamizu@example.org>\r\n" +
	"Reply-To: Mitsuha Miyamizu <mitsuha.miyamizu+replyto@example.org>\r\n" +
	"Message-Id: 42@example.org\r\n" +
	"Subject: Your Name.\r\n" +
	"To: Taki Tachibana <taki.tachibana@example.org>\r\n" +
	"\r\n"

const testHeaderFromToString = "From: Mitsuha Miyamizu <mitsuha.miyamizu@example.org>\r\n" +
	"To: Taki Tachibana <taki.tachibana@example.org>\r\n" +
	"\r\n"

const testHeaderDateString = "Date: Sat, 18 Jun 2016 12:00:00 +0900\r\n" +
	"Date: Sat, 19 Jun 2016 12:00:00 +0900\r\n" +
	"\r\n"

const testHeaderNoFromToString = "Content-Type: multipart/mixed; boundary=message-boundary\r\n" +
	"Date: Sat, 18 Jun 2016 12:00:00 +0900\r\n" +
	"Date: Sat, 19 Jun 2016 12:00:00 +0900\r\n" +
	"Reply-To: Mitsuha Miyamizu <mitsuha.miyamizu+replyto@example.org>\r\n" +
	"Message-Id: 42@example.org\r\n" +
	"Subject: Your Name.\r\n" +
	"\r\n"

const testAltHeaderString = "Content-Type: multipart/alternative; boundary=b2\r\n" +
	"\r\n"

const testTextHeaderString = "Content-Disposition: inline\r\n" +
	"Content-Type: text/plain\r\n" +
	"\r\n"

const testTextContentTypeString = "Content-Type: text/plain\r\n" +
	"\r\n"

const testTextNoContentTypeString = "Content-Disposition: inline\r\n" +
	"\r\n"

const testTextBodyString = "What's your name?"

const testTextString = testTextHeaderString + testTextBodyString

const testHTMLHeaderString = "Content-Disposition: inline\r\n" +
	"Content-Type: text/html\r\n" +
	"\r\n"

const testHTMLBodyString = "<div>What's <i>your\r\n</i> name?</div>"

const testHTMLString = testHTMLHeaderString + testHTMLBodyString

const testAttachmentHeaderString = "Content-Disposition: attachment; filename=note.txt\r\n" +
	"Content-Type: text/plain\r\n" +
	"\r\n"

const testAttachmentBodyString = "My name is Mitsuha."

const testAttachmentString = testAttachmentHeaderString + testAttachmentBodyString

const testBodyString = "--message-boundary\r\n" +
	testAltHeaderString +
	"\r\n--b2\r\n" +
	testTextString +
	"\r\n--b2\r\n" +
	testHTMLString +
	"\r\n--b2--\r\n" +
	"\r\n--message-boundary\r\n" +
	testAttachmentString +
	"\r\n--message-boundary--\r\n"

const testMailString = testHeaderString + testBodyString
