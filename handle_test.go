package imap_test

import (
	"testing"

	"github.com/emersion/go-imap"
)

func TestRespHandle_Accept(t *testing.T) {
	ch := make(chan bool, 1)
	hdlr := &imap.RespHandle{
		Accepts: ch,
	}

	hdlr.Accept()

	v := <-ch
	if v != true {
		t.Error("Invalid return value:", v)
	}
}

func TestRespHandle_Reject(t *testing.T) {
	ch := make(chan bool, 1)
	hdlr := &imap.RespHandle{
		Accepts: ch,
	}

	hdlr.Reject()

	v := <-ch
	if v != false {
		t.Error("Invalid return value:", v)
	}
}

func TestRespHandle_AcceptNamedResp_Matching(t *testing.T) {
	ch := make(chan bool, 1)
	hdlr := &imap.RespHandle{
		Resp: &imap.Resp{
			Tag:    "*",
			Fields: []interface{}{"SEARCH", "42"},
		},
		Accepts: ch,
	}

	fields, ok := hdlr.AcceptNamedResp("SEARCH")
	if ok != true {
		t.Error("Matching response not accepted")
	}
	if len(fields) != 1 {
		t.Error("Invalid fields")
	}
	if f, ok := fields[0].(string); !ok || f != "42" {
		t.Error("Invalid first field")
	}

	v := <-ch
	if v != true {
		t.Error("Invalid return value:", v)
	}
}

func TestRespHandle_AcceptNamedResp_NotMatching(t *testing.T) {
	ch := make(chan bool, 1)
	hdlr := &imap.RespHandle{
		Resp: &imap.Resp{
			Tag:    "*",
			Fields: []interface{}{"26", "EXISTS"},
		},
		Accepts: ch,
	}

	_, ok := hdlr.AcceptNamedResp("SEARCH")
	if ok != false {
		t.Error("Response not matching has been accepted")
	}

	v := <-ch
	if v != false {
		t.Error("Invalid return value:", v)
	}
}

func TestMultiRespHandler(t *testing.T) {
	mh := imap.NewMultiRespHandler()

	h1 := make(imap.RespHandler)
	mh.Add(h1)
	go func() {
		(<-h1).Accept()
		(<-h1).Reject()
		mh.Del(h1)
	}()

	h2 := make(imap.RespHandler)
	mh.Add(h2)
	go func() {
		(<-h2).Reject()
		(<-h2).Reject()
		mh.Del(h2)
	}()

	// Should not add it, or will block forever
	var h3 imap.RespHandler
	mh.Add(h3)

	rh1 := &imap.RespHandle{Accepts: make(chan bool, 1)}
	rh2 := &imap.RespHandle{Accepts: make(chan bool, 1)}

	h := make(imap.RespHandler, 2)
	h <- rh1
	h <- rh2
	close(h)

	if err := mh.HandleFrom(h); err != nil {
		t.Fatal("Expected no error while handling response, got:", err)
	}
	if accepted := <-rh1.Accepts; !accepted {
		t.Error("First response was not accepted")
	}
	if accepted := <-rh2.Accepts; accepted {
		t.Error("First response was not rejected")
	}
}
