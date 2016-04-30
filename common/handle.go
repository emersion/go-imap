package common

type RespHandling struct {
	Resp interface{}
	Accepts chan bool
}

func (h *RespHandling) Accept() {
	h.Accepts <- true
}

func (h *RespHandling) Reject() {
	h.Accepts <- false
}

type RespHandler chan *RespHandling

type RespHandlerFrom interface {
	HandleFrom(hdlr RespHandler) error
}
