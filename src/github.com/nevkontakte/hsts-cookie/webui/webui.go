package webui

//go:generate go-bindata -o assets.go -pkg webui -prefix ../public ../public/...

import (
	"html/template"
	"net/http"

	"github.com/nevkontakte/hsts-cookie/config"
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

var templates = template.Must(template.New("index.html").Parse(string(MustAsset("index.html"))))

func IndexHandler(response http.ResponseWriter, request *http.Request) {
	if request.TLS != nil {
		http.Redirect(response, request, "http://"+config.Domain, 302)
		return
	}
	data := struct {
		Domain string
	}{
		Domain: config.Domain,
	}
	err := templates.ExecuteTemplate(response, "index.html", data)
	if err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
	}

}
