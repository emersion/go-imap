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
	locker   sync.RWMutex
}

func NewMultiRespHandler() *MultiRespHandler {
	return &MultiRespHandler{}
}

func (mh *MultiRespHandler) HandleFrom(ch RespHandler) error {
	for rh := range ch {
		mh.locker.RLock()

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

		mh.locker.RUnlock()

		if accepted {
			rh.Accept()
		} else {
			rh.Reject()
		}
	}

	// Close and remove all handlers
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
	// Find the handler's index
	mh.locker.RLock()
	found := -1
	for i, h := range mh.handlers {
		if h == hdlr {
			found = i
			break
		}
	}
	mh.locker.RUnlock()

	if found < 0 {
		// Not found
		return
	}

	// Close handler and remove it
	// Make sure to close it before locking, because of deadlocks
	close(hdlr)
	mh.locker.Lock()
	mh.handlers = append(mh.handlers[:found], mh.handlers[found+1:]...)
	mh.locker.Unlock()
}
