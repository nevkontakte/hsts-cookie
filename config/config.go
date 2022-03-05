package config

import (
	"sort"
	"strconv"
	"time"
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
		subdomain := o.OffsetToDomain(i)
		subdomain = subdomain + "." + o.Domain
		domains = append(domains, subdomain)
	}
	sort.Strings(domains)

	return domains
}

func (o Options) AllDomains() []string {
	return append([]string{o.Domain, o.TagDomain()}, o.BitDomains()...)
}

func (o Options) OffsetToDomain(offset uint8) string {
	return strconv.FormatUint(uint64(offset), 36)
}

func (o Options) DomainToOffset(d string) (uint8, error) {
	offset, err := strconv.ParseUint(d, 36, 8)
	return uint8(offset), err
}
