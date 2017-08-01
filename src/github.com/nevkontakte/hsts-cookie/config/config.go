package config

import (
	"sort"
	"strconv"
	"time"
)

const (
	Domain         = "hsts.nevkontakte.com"
	CookieLifetime = 60 * 60 * 24 // 1 day lifetime
	CookieBits     = 30
)

type Options struct {
	Domain         string
	CookieLifeTime time.Duration
	CookieBits     uint8
}

func (o Options) TagDomain() string {
	return "tag." + o.Domain
}

func (o Options) BitDomains() []string {
	var domains []string

	for i := uint8(0); i < o.CookieBits; i++ {
		subdomain := strconv.FormatUint(uint64(i), int(o.CookieBits))
		subdomain = subdomain + "." + o.Domain
		domains = append(domains, subdomain)
	}
	sort.Strings(domains)

	return domains
}

func (o Options) AllDomains() []string {
	return append([]string{o.Domain, o.TagDomain()}, o.BitDomains()...)
}
