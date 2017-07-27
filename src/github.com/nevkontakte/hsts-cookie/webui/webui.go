package webui

import (
	"github.com/nevkontakte/hsts-cookie/cookie"
)

type StartResolvingOp struct {
	token chan cookie.Token
}

type ResolveBitOp struct {
	token  cookie.Token
	offset uint32
	value  bool
	result chan cookie.MaybeCookie
}

func RunResolvingWorker() (chan *StartResolvingOp, chan *ResolveBitOp) {
	start := make(chan *StartResolvingOp)
	resolve := make(chan *ResolveBitOp)

	resolver := cookie.NewResolver()

	go func() {
		for {
			select {
			case s := <-start:
				s.token <- resolver.StartResolving()
			case r := <-resolve:
				rc, err := resolver.ResolveBit(r.token, r.offset, r.value)
				if err != nil {
					r.result <- cookie.MaybeCookie{Cookie: nil, Error: err}
				} else if rc.IsComplete() {
					r.result <- cookie.MaybeCookie{Cookie: &rc.Cookie, Error: nil}
				} else {
					r.result <- cookie.MaybeCookie{Cookie: nil, Error: nil}
				}
			}

		}
	}()

	return start, resolve
}

var start, resolve = RunResolvingWorker()
