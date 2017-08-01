package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/sync/errgroup"

	"github.com/apex/log"
	"github.com/nevkontakte/hsts-cookie/config"
	"github.com/nevkontakte/hsts-cookie/webui"
)

var (
	makeHosts = flag.Bool("make_hosts", false, "Generate /etc/hosts line for debugging and exit.")

	domain = flag.String("domain", "hsts.nevkontakte.com", "Base domain name for the service.")

	addr      = flag.String("addr", "0.0.0.0", "IP address to bind on.")
	portHTTP  = flag.Int("port_http", 80, "Port number for plain HTTP requests.")
	portHTTPS = flag.Int("port_https", 443, "Port number for HTTPS requests.")

	acmeDir      = flag.String("acme_dir", "./.acme-cache", "Path for ACME cache directory.")
	useProdCerts = flag.Bool("use_production_certs", false, "Use Let's Encrypt production service. If not specified, will use staging instead.")
)

const LetsEncryptStagingURL = "https://acme-staging.api.letsencrypt.org/directory"

func AcceptTOS(url string) bool {
	log.Infof("Using this service implies acceptance of ToS at %s", url)
	return true
}

type Server struct {
	Addr      string
	PortHTTP  int
	PortHTTPS int
	Domains   []string

	CacheDir     string
	UseProdCerts bool

	Handler http.Handler
}

func (s *Server) ListenAndServe() error {
	plain := http.Server{
		Addr:    fmt.Sprintf("%s:%d", s.Addr, s.PortHTTP),
		Handler: s.Handler,
	}

	acmeDirectory := LetsEncryptStagingURL
	if s.UseProdCerts {
		acmeDirectory = acme.LetsEncryptURL
	}

	m := autocert.Manager{
		Prompt:     AcceptTOS,
		HostPolicy: autocert.HostWhitelist(s.Domains...),
		Cache:      autocert.DirCache(s.CacheDir),
		Client:     &acme.Client{DirectoryURL: acmeDirectory},
	}

	secure := http.Server{
		Addr:      fmt.Sprintf("%s:%d", s.Addr, s.PortHTTPS),
		Handler:   s.Handler,
		TLSConfig: &tls.Config{GetCertificate: m.GetCertificate},
	}

	g, ctx := errgroup.WithContext(context.Background())

	g.Go(func() error {
		log.Infof("Starting HTTP server at %q...", plain.Addr)
		return plain.ListenAndServe()
	})

	g.Go(func() error {
		log.Infof("Starting HTTPS server at %q...", secure.Addr)
		return secure.ListenAndServeTLS("", "")
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
	fmt.Printf("%s\t", *addr)
	for _, v := range opts.AllDomains() {
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
			Addr:         *addr,
			PortHTTP:     *portHTTP,
			PortHTTPS:    *portHTTPS,
			Domains:      opts.AllDomains(),
			UseProdCerts: *useProdCerts,
			CacheDir:     *acmeDir,
			Handler:      webui.New(opts).GetHandler(),
		}
		err = s.ListenAndServe()
	}

	if err != nil {
		log.Errorf("Fatal error: %s", err)
		os.Exit(1)
	}
	os.Exit(0)
}
