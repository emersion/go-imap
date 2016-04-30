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

func (h *RespHandling) AcceptNamedResp(name string) (fields []interface{}) {
	res, ok := h.Resp.(*Resp)
	if !ok || len(res.Fields) == 0 {
		h.Reject()
		return
	}

	n, ok := res.Fields[0].(string)
	if !ok || n != name {
		h.Reject()
		return
	}

	h.Accept()
	return res.Fields[1:]
}

type RespHandler chan *RespHandling

type RespHandlerFrom interface {
	HandleFrom(hdlr RespHandler) error
}
