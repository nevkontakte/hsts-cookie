package cookie

import (
	"errors"
	"github.com/nevkontakte/hsts-cookie/config"
	"math/rand"
	"strconv"
)

type Token uint32

type Cookie struct {
	Id uint32
}

func RandomCookie() Cookie {
	return Cookie{Id: rand.Uint32() % (1 << config.CookieBits)}
}

type MaybeCookie struct {
	Cookie *Cookie
	Error  error
}

func (c *Cookie) Export(domain string) map[string]bool {
	domains := make(map[string]bool)

	for offset := uint64(0); offset < config.CookieBits; offset++ {
		subdomain := strconv.FormatUint(offset, config.CookieBits)
		subdomain = subdomain + "." + domain
		domains[subdomain] = (c.Id & (1 << offset)) != 0
	}

	return domains
}

func ExportSubdomains(domain string) []string {
	domains := make([]string, config.CookieBits)

	for i := uint64(0); i < config.CookieBits; i++ {
		subdomain := strconv.FormatUint(i, config.CookieBits)
		subdomain = subdomain + "." + domain
		domains[i] = subdomain
	}

	return domains
}

type ResolvingCookie struct {
	Cookie Cookie
	Mask   uint32
}

func (rc *ResolvingCookie) IsComplete() bool {
	return rc.Mask == ((1 << config.CookieBits) - 1)
}

type CookieResolver struct {
	resolvingCache map[Token]*ResolvingCookie
}

func NewResolver() CookieResolver {
	return CookieResolver{
		resolvingCache: make(map[Token]*ResolvingCookie),
	}
}

func (r *CookieResolver) StartResolving() Token {
	token := Token(rand.Uint32())
	r.resolvingCache[token] = new(ResolvingCookie)
	return token
}

func (r *CookieResolver) ResolveBit(token Token, offset uint32, value bool) (*ResolvingCookie, error) {
	if _, ok := r.resolvingCache[token]; !ok {
		return nil, errors.New("Invalid resolving token")
	}

	r.resolvingCache[token].Mask |= (1 << offset)
	if value {
		r.resolvingCache[token].Cookie.Id |= (1 << offset)
	}

	if r.resolvingCache[token].IsComplete() {
		rc := r.resolvingCache[token]
		delete(r.resolvingCache, token)
		return rc, nil
	} else {
		return r.resolvingCache[token], nil
	}
}
