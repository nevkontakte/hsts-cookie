package webui

//go:generate go-bindata -o assets.go -pkg webui -prefix ../public ../public/...

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/nevkontakte/hsts-cookie/config"
	"github.com/nevkontakte/hsts-cookie/cookie"
)

var templates = template.Must(template.New("index.html").Parse(string(MustAsset("index.html"))))

type WebApp struct {
	Opts config.Options

	resolver *cookie.Resolver
}

func New(o config.Options) *WebApp {
	return &WebApp{
		Opts:     o,
		resolver: cookie.NewResolver(),
	}
}

func (wa WebApp) GetHandler() http.Handler {
	r := mux.NewRouter()
	r.HandleFunc("/dispatch.css", wa.TagDispatchHandler).Host(wa.Opts.TagDomain())
	r.HandleFunc("/setup.css", wa.TagSetupHandler).Host(wa.Opts.TagDomain())
	r.HandleFunc("/reset.css", wa.TagResetHandler).Host(wa.Opts.TagDomain())
	r.HandleFunc("/get/{token:[0-9]+}.css", wa.GetBitHandler).Host("{subdomain:[0-9a-z]}." + wa.Opts.Domain)
	r.HandleFunc("/set/{switch:(?:on|off)}.css", wa.SetBitHandler).Host("{subdomain:[0-9a-z]}." + wa.Opts.Domain)

	r.HandleFunc("/", wa.IndexHandler)

	return r
}

// IndexHandler Displays index page and triggers request to dispatch.css to perform demonstration.
func (wa *WebApp) IndexHandler(response http.ResponseWriter, request *http.Request) {
	if request.TLS != nil {
		http.Redirect(response, request, "http://"+wa.Opts.Domain, 302)
		return
	}
	data := struct {
		Domain string
	}{
		Domain: wa.Opts.Domain,
	}
	err := templates.ExecuteTemplate(response, "index.html", data)
	if err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
	}

}

// TagDispatchHandler serves requests to dispatch.css. It determines whether the user has been
// already assigned a cookie and either set up a new one or initiates cookie resolution process.
func (wa *WebApp) TagDispatchHandler(response http.ResponseWriter, request *http.Request) {
	if request.TLS == nil {
		http.Redirect(response, request, fmt.Sprintf("https://tag.%s/setup.css", wa.Opts.Domain), 302)
	} else {
		token := wa.resolver.Begin(wa.Opts.CookieBits)

		response.Header().Add("Content-Type", "text/css")
		for _, subdomain := range wa.Opts.BitDomains() {
			fmt.Fprintf(response, "@import url(\"http://%s/get/%d.css\") all;\n", subdomain, token)
		}
	}
}

// TagSetupHandler serves requests to setup.css. It generates a new random cookie for a user
// and generates a series of requests which would set up the fingerprint.
func (wa *WebApp) TagSetupHandler(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("Content-Type", "text/css")
	response.Header().Set("Strict-Transport-Security", fmt.Sprintf("max-age=%.0f", wa.Opts.CookieLifeTime.Seconds()))
	domains := wa.Opts.BitDomains()
	c := cookie.RandomCookie(wa.Opts.CookieBits)

	for i, use_https := range c.Export() {
		var filename string
		if use_https {
			filename = "on"
		} else {
			filename = "off"
		}
		subdomain := domains[i]

		fmt.Fprintf(response, "@import url(\"https://%s/set/%s.css\") all;\n", subdomain, filename)
	}

	fmt.Fprintf(response, ".set {display: block}")
	fmt.Fprintf(response, ".set:after {content: \"%04X\"}", c.Id)
}

// TagResetHandler serves requests to reset.css. It resets cookie assignment bit for a user,
// so that next visit to dispatch.css will result in a new fingerprint created.
func (wa *WebApp) TagResetHandler(response http.ResponseWriter, request *http.Request) {
	response.Header().Add("Content-Type", "text/css")
	response.Header().Set("Strict-Transport-Security", "max-age=0")
}

// SetBitHandler serves requests to /set/(on|of).css on bit domains. It issues HSTS headers to
// set corresponding cookie bit to the requested state: on or off.
func (wa *WebApp) SetBitHandler(response http.ResponseWriter, request *http.Request) {
	params := mux.Vars(request)

	response.Header().Add("Content-Type", "text/css")
	if params["switch"] == "on" {
		response.Header().Set("Strict-Transport-Security", fmt.Sprintf("max-age=%.0f", wa.Opts.CookieLifeTime.Seconds()))
	} else {
		response.Header().Set("Strict-Transport-Security", "max-age=0")
	}

	fmt.Fprintf(response, "/* %s -> %s */", params["subdomain"], params["switch"])
}

// GetBitHandler serves requests to /get/{token}.css on bit domains. It passes another bit to
// cookie resolver and returns cookie value to a user if the cookie has been fully resolved.
func (wa *WebApp) GetBitHandler(response http.ResponseWriter, request *http.Request) {
	params := mux.Vars(request)

	bit_offset, _ := wa.Opts.DomainToOffset(params["subdomain"])

	token64, _ := strconv.ParseUint(params["token"], 0, 32)
	token := cookie.Token(token64)

	p, ok := wa.resolver.Get(token)

	if !ok {
		http.Error(response, "Unknown cookie resolution token", 404)
		return
	}

	p.SetBit(uint8(bit_offset), request.TLS != nil)
	c, complete := p.Resolved()

	response.Header().Add("Content-Type", "text/css")
	if complete {
		wa.resolver.End(token)

		fmt.Fprintf(response, "/* Cookie: %d */\n", c.Id)
		fmt.Fprintf(response, ".get {display: block}")
		fmt.Fprintf(response, ".get:after {content: \"%04X\"}", c.Id)
	} else {
		fmt.Fprintf(response, "/* Keep resolving... */")
	}
}
