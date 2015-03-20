package webui

import (
	"fmt"
	"github.com/nevkontakte/hsts-cookie/config"
	"github.com/nevkontakte/hsts-cookie/cookie"
	"net/http"
    "strconv"
)

func TagDispatchHandler(response http.ResponseWriter, request *http.Request) {
	if request.TLS == nil {
		http.Redirect(response, request, fmt.Sprintf("https://tag.%s/setup.css", config.Domain), 302)
	} else {
		op := StartResolvingOp{token: make(chan cookie.Token)}
		start <- &op
		token := <-op.token

		response.Header().Add("Content-Type", "text/css")
		for _, subdomain := range cookie.ExportSubdomains(config.Domain) {
			fmt.Fprintf(response, "@import url(\"http://%s/get/%d.css\") all;\n", subdomain, token)
		}
	}
}

func TagSetupHandler(response http.ResponseWriter, request *http.Request) {
	c := cookie.RandomCookie()
	for subdomain, use_https := range c.Export(config.Domain) {
		var filename string
		if use_https {
			filename = "on"
		} else {
			filename = "off"
		}

		response.Header().Add("Content-Type", "text/css")
		response.Header().Set("Strict-Transport-Security", "max-age="+strconv.Itoa(config.CookieLifetime))
		fmt.Fprintf(response, "@import url(\"https://%s/set/%s.css\") all;\n", subdomain, filename)
	}

	fmt.Fprintf(response, ".set {display: block}")
	fmt.Fprintf(response, ".set:after {content: \"%04X\"}", c.Id)
}

func TagResetHandler(response http.ResponseWriter, request *http.Request) {
	response.Header().Add("Content-Type", "text/css")
	response.Header().Set("Strict-Transport-Security", "max-age=0")
}
