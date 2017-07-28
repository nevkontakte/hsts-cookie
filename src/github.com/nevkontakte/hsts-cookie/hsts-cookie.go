package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/apex/log"
	"github.com/gorilla/mux"
	"github.com/nevkontakte/hsts-cookie/config"
	"github.com/nevkontakte/hsts-cookie/webui"
)

var (
	makeHosts = flag.Bool("make_hosts", false, "Generate /etc/hosts line for debugging and exit.")

	domain = flag.String("domain", "hsts.nevkontakte.com", "Base domain name for the service.")

	addr      = flag.String("addr", "0.0.0.0", "IP address to bind on.")
	portHTTP  = flag.Int("port_http", 8080, "Port number for plain HTTP requests.")
	portHTTPS = flag.Int("port_https", 4343, "Port number for HTTPS requests.")
)

type Server struct {
	Addr      string
	PortHTTP  int
	PortHTTPS int

	CertFile string
	KeyFile  string

	Handler http.Handler
}

func (s *Server) ListenAndServe() error {
	plain := http.Server{
		Addr:    fmt.Sprintf("%s:%d", s.Addr, s.PortHTTP),
		Handler: s.Handler,
	}

	secure := http.Server{
		Addr:    fmt.Sprintf("%s:%d", s.Addr, s.PortHTTPS),
		Handler: s.Handler,
	}

	g, ctx := errgroup.WithContext(context.Background())

	g.Go(func() error {
		log.Infof("Starting HTTP server at %q...", plain.Addr)
		return plain.ListenAndServe()
	})

	g.Go(func() error {
		log.Infof("Starting HTTPS server at %q...", secure.Addr)
		return secure.ListenAndServeTLS(s.CertFile, s.KeyFile)
	})

	go func() {
		select {
		case <-ctx.Done():
			log.Infof("Shutting down HTTP servers...")
			plain.Close()
			secure.Close()
		}
	}()

	return g.Wait()
}

func printHosts(opts config.Options) {
	fmt.Println("# Hosts file line for debugging...")
	fmt.Printf("%s\t%s %s", *addr, opts.Domain, opts.TagDomain())
	for _, v := range opts.BitDomains() {
		print(" ", v)
	}
	println()
}

func main() {
	flag.Parse()

	opts := config.Options{
		Domain:         *domain,
		CookieLifeTime: time.Hour * 24,
		CookieBits:     16,
	}

	var err error
	switch {
	case *makeHosts:
		printHosts(opts)
	default:
		s := &Server{
			Addr:      *addr,
			PortHTTP:  *portHTTP,
			PortHTTPS: *portHTTPS,
		}
		err = s.ListenAndServe()
	}

	if err != nil {
		log.Errorf("Fatal error: %s", err)
		os.Exit(1)
	}
	os.Exit(0)

	r := mux.NewRouter()
	r.HandleFunc("/dispatch.css", webui.TagDispatchHandler).Host("tag." + config.Domain)
	r.HandleFunc("/setup.css", webui.TagSetupHandler).Host("tag." + config.Domain)
	r.HandleFunc("/reset.css", webui.TagResetHandler).Host("tag." + config.Domain)
	r.HandleFunc("/get/{token:[0-9]+}.css", webui.GetBitHandler).Host("{subdomain:[0-9a-z]}." + config.Domain)
	r.HandleFunc("/set/{switch:(?:on|off)}.css", webui.SetBitHandler).Host("{subdomain:[0-9a-z]}." + config.Domain)

	r.HandleFunc("/", webui.IndexHandler)
	http.Handle("/", r)

	aborted := make(chan int)
	go func() {
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Errorf("Error serving HTTP requests: %s", err)
		}
		aborted <- 0
	}()
	go func() {
		if err := http.ListenAndServeTLS(":4343", "secret/hsts.crt", "secret/hsts.key", nil); err != nil {
			log.Errorf("Error serving HTTPS requests: %s", err)
		}
		aborted <- 0
	}()
	println("Up and running")
	<-aborted
	println("Aborted")
}
