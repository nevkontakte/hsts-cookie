package main

import (
	"context"
	"flag"
	"net"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/sync/errgroup"

	"github.com/apex/log"
	"github.com/digitalocean/godo"
	"github.com/pkg/errors"

	"github.com/nevkontakte/hsts-cookie/config"
)

var (
	domain     = flag.String("domain", "hsts.nevkontakte.com", "Base domain name for the service.")
	target     = flag.String("target", "0.0.0.0", "IP address to point domain and subdomains to. If a domain name is provided, it will be resolved into an IP address at setup time.")
	cookieBits = flag.Int("cookie_bits", config.CookieBits, "Cookie bit size.")

	apiKey = flag.String("api_key", "", "DigitalOcean API key.")
)

type tokenSource struct {
	T string
}

func (t *tokenSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{AccessToken: t.T}, nil
}

// DomainManager provides high-level API to manipulate domains managed by DO DNS.
type DomainManager struct {
	Client *godo.Client
	Domain string
}

// Exists checks whether a domain zone exists.
func (d *DomainManager) Exists(ctx context.Context) (bool, error) {
	_, r, err := d.Client.Domains.Get(ctx, d.Domain)
	if err != nil && r.StatusCode == 404 {
		return false, nil
	} else if err != nil {
		return false, errors.Wrap(err, "unable to check domain existence")
	}

	return true, nil
}

// Create a new domain zone.
func (d *DomainManager) Create(ctx context.Context) error {
	req := &godo.DomainCreateRequest{
		Name:      d.Domain,
		IPAddress: "0.0.0.0",
	}
	_, _, err := d.Client.Domains.Create(ctx, req)
	return err
}

// Ensure that domain zone exists. Creates it if not.
func (d *DomainManager) Ensure(ctx context.Context) error {
	if ok, err := d.Exists(ctx); err != nil {
		return errors.Wrap(err, "unable to ensure domain")
	} else if !ok {
		return d.Create(ctx)
	}
	return nil
}

// PurgeRecords deletes all A, AAA and CNAME records in the zone.
func (d *DomainManager) PurgeRecords(ctx context.Context) error {
	records, _, err := d.Client.Domains.Records(ctx, d.Domain, nil)
	if err != nil {
		return errors.Wrap(err, "unable to list existing records")
	}

	g, ctx := errgroup.WithContext(ctx)

	for _, r := range records {
		r := r

		if !(r.Type == "A" || r.Type == "AAA" || r.Type == "CNAME") {
			continue
		}

		g.Go(func() error {
			_, err := d.Client.Domains.DeleteRecord(ctx, d.Domain, r.ID)
			return errors.Wrapf(err, "unable to delete record %v", r)
		})
	}
	return g.Wait()
}

// TargetAt creates A records for the list of provided IPs.
func (d *DomainManager) TargetAt(ctx context.Context, ips []string) error {
	g, ctx := errgroup.WithContext(ctx)

	for _, ip := range ips {
		ip := ip
		g.Go(func() error {
			req := &godo.DomainRecordEditRequest{
				Type: "A",
				Name: "@",
				Data: ip,
			}
			_, _, err := d.Client.Domains.CreateRecord(ctx, d.Domain, req)
			return errors.Wrapf(err, "unable to create A record for %q", ip)
		})
	}
	return g.Wait()
}

// AddAliases as CNAME records pointing to the main domain.
func (d *DomainManager) AddAliases(ctx context.Context, aliases []string) error {
	g, ctx := errgroup.WithContext(ctx)

	for _, alias := range aliases {
		alias := alias

		g.Go(func() error {
			req := &godo.DomainRecordEditRequest{
				Type: "CNAME",
				Name: strings.TrimSuffix(alias, d.Domain),
				Data: d.Domain + ".",
			}
			_, _, err := d.Client.Domains.CreateRecord(ctx, d.Domain, req)
			return errors.Wrapf(err, "unable to create CNAME record for %q", alias)
		})
	}

	return g.Wait()
}

func main() {
	flag.Parse()

	ctx := context.Background()
	client := godo.NewClient(oauth2.NewClient(ctx, &tokenSource{T: *apiKey}))

	log.Infof("Setting up domains for %s to point to %s.", *domain, *target)

	ips, err := net.LookupHost(*target)
	if err != nil {
		log.Fatalf("Unable to look up target %q: %s", *target, err)
	} else if len(ips) == 0 {
		log.Fatalf("Unable to look up target %q: empty IP list.", *target)
	}

	log.Infof("Resolved %q into %q.", *target, ips)

	d := &DomainManager{
		Client: client,
		Domain: *domain,
	}

	log.Infof("Creating DNS zone for %q...", d.Domain)
	if err := d.Ensure(ctx); err != nil {
		log.Fatalf("Unable to create a new domain: %s", err)
	}

	log.Infof("Purging all existing A, AAA and CNAME records...")
	if err := d.PurgeRecords(ctx); err != nil {
		log.Fatalf("Unable to purge existing records: %s", err)
	}

	log.Infof("Creating A records for the main domain...")
	if err := d.TargetAt(ctx, ips); err != nil {
		log.Fatalf("Unable to create A records for the main domain: %s", err)
	}

	opts := config.Options{
		Domain:     d.Domain,
		CookieBits: uint8(*cookieBits),
	}
	aliases := append(opts.BitDomains(), opts.TagDomain())
	log.Infof("Creating CNAME records for subdomains: %v...", aliases)
	if err := d.AddAliases(ctx, aliases); err != nil {
		log.Fatalf("Unable to create CNAME record: %s", err)
	}
}
