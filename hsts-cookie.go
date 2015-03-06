package main

import (
	"github.com/gorilla/mux"
	"html/template"
	"github.com/nevkontakte/hsts-cookie/config"
	"github.com/nevkontakte/hsts-cookie/cookie"
	"github.com/nevkontakte/hsts-cookie/webui"
	"net/http"
)

var templates = template.Must(template.ParseFiles(
	"resources/index.html"))

func indexHandler(response http.ResponseWriter, request *http.Request) {
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
		http.ListenAndServeTLS(":4343", "critical/1_nevkontakte.com_bundle.crt", "critical/hsts.key", nil)
		aborted <- 0
	}()
	println("Up and running")
	<-aborted
	println("Aborted")
}
