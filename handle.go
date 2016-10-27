package imap

import (
	"sync"
)

// A response that can be either accepted or rejected by a handler.
type RespHandle struct {
	Resp    interface{}
	Accepts chan bool
}

// Accept this response. This means that the handler will process it.
func (h *RespHandle) Accept() {
	h.Accepts <- true
}

// Reject this response. The handler cannot process it.
func (h *RespHandle) Reject() {
	h.Accepts <- false
}

// Accept this response if it has the specified name. If not, reject it.
func (h *RespHandle) AcceptNamedResp(name string) (fields []interface{}, accepted bool) {
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

	fields = res.Fields[1:]
	accepted = true
	return
}

// Delivers responses to handlers.
type RespHandler chan *RespHandle

// Handles responses from a handler.
type RespHandlerFrom interface {
	HandleFrom(hdlr RespHandler) error
}

// A RespHandlerFrom that forwards responses to multiple RespHandler.
type MultiRespHandler struct {
	handlers []RespHandler
	locker   sync.Locker
}

func NewMultiRespHandler() *MultiRespHandler {
	return &MultiRespHandler{
		locker: &sync.Mutex{},
	}
}

func (mh *MultiRespHandler) HandleFrom(ch RespHandler) error {
	for rh := range ch {
		mh.locker.Lock()

		accepted := false
		for i := len(mh.handlers) - 1; i >= 0; i-- {
			hdlr := mh.handlers[i]

			rh := &RespHandle{
				Resp:    rh.Resp,
				Accepts: make(chan bool),
			}

			hdlr <- rh
			if accepted = <-rh.Accepts; accepted {
				break
			}
		}

		mh.locker.Unlock()

		if accepted {
			rh.Accept()
		} else {
			rh.Reject()
		}
	}

	mh.locker.Lock()
	for _, hdlr := range mh.handlers {
		close(hdlr)
	}
	mh.handlers = nil
	mh.locker.Unlock()

	return nil
}

func (mh *MultiRespHandler) Add(hdlr RespHandler) {
	if hdlr == nil {
		return
	}

	mh.locker.Lock()
	mh.handlers = append(mh.handlers, hdlr)
	mh.locker.Unlock()
}

func (mh *MultiRespHandler) Del(hdlr RespHandler) {
	mh.locker.Lock()
	for i, h := range mh.handlers {
		if h == hdlr {
			close(hdlr)
			mh.handlers = append(mh.handlers[:i], mh.handlers[i+1:]...)
		}
	}
	mh.locker.Unlock()
}
