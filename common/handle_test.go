package common_test

import (
	"testing"

	"github.com/emersion/go-imap/common"
)

func TestRespHandling_Accept(t *testing.T) {
	ch := make(chan bool, 1)
	hdlr := &common.RespHandling{
		Accepts: ch,
	}

	hdlr.Accept()

	v := <-ch
	if v != true {
		t.Error("Invalid return value:", v)
	}
}

func TestRespHandling_Reject(t *testing.T) {
	ch := make(chan bool, 1)
	hdlr := &common.RespHandling{
		Accepts: ch,
	}

	hdlr.Reject()

	v := <-ch
	if v != false {
		t.Error("Invalid return value:", v)
	}
}

func TestRespHandling_AcceptNamedResp_Matching(t *testing.T) {
	ch := make(chan bool, 1)
	hdlr := &common.RespHandling{
		Resp: &common.Resp{
			Tag: "*",
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

func TestRespHandling_AcceptNamedResp_NotMatching(t *testing.T) {
	ch := make(chan bool, 1)
	hdlr := &common.RespHandling{
		Resp: &common.Resp{
			Tag: "*",
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
