package main

//go:generate go-bindata -o assets.go -pkg main public/...

import (
	"html/template"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/nevkontakte/hsts-cookie/config"
	"github.com/nevkontakte/hsts-cookie/cookie"
	"github.com/nevkontakte/hsts-cookie/webui"
)

var templates = template.Must(template.New("index.html").Parse(string(MustAsset("public/index.html"))))

func indexHandler(response http.ResponseWriter, request *http.Request) {
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

func main() {
	println("# Hosts file line for debugging...")
	print("127.0.0.1\t", config.Domain, " tag.", config.Domain)
	for _, v := range cookie.ExportSubdomains(config.Domain) {
		print(" ", v)
	}
	println()
	r := mux.NewRouter()
	r.HandleFunc("/dispatch.css", webui.TagDispatchHandler).Host("tag." + config.Domain)
	r.HandleFunc("/setup.css", webui.TagSetupHandler).Host("tag." + config.Domain)
	r.HandleFunc("/reset.css", webui.TagResetHandler).Host("tag." + config.Domain)
	r.HandleFunc("/get/{token:[0-9]+}.css", webui.GetBitHandler).Host("{subdomain:[0-9a-z]}." + config.Domain)
	r.HandleFunc("/set/{switch:(on|off)}.css", webui.SetBitHandler).Host("{subdomain:[0-9a-z]}." + config.Domain)

	r.HandleFunc("/", indexHandler)
	http.Handle("/", r)

	aborted := make(chan int)
	go func() {
		http.ListenAndServe(":8080", nil)
		aborted <- 0
	}()
	go func() {
		http.ListenAndServeTLS(":4343", "secret/hsts.crt", "secret/hsts.key", nil)
		aborted <- 0
	}()
	println("Up and running")
	<-aborted
	println("Aborted")
}
