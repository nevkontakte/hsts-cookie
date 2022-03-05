package cookie

import (
	"math/rand"
	"sync"
)

type Token uint32

type Cookie struct {
	Id   uint32
	Size uint8
}

func RandomCookie(size uint8) Cookie {
	return Cookie{
		Id:   rand.Uint32() % (1 << size),
		Size: size,
	}
}

func (c *Cookie) Export() []bool {
	bits := []bool{}

	for offset := uint8(0); offset < c.Size; offset++ {
		bits = append(bits, (c.Id&(1<<offset)) != 0)
	}

	return bits
}

type Partial struct {
	mu   sync.Mutex
	c    Cookie
	mask uint32
}

func (p *Partial) Resolved() (Cookie, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.mask == ((1 << p.c.Size) - 1) {
		return p.c, true
	} else {
		return Cookie{}, false
	}
}

func (p *Partial) SetBit(offset uint8, value bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.mask |= (1 << offset)
	if value {
		p.c.Id |= (1 << offset)
	}
}

type Resolver struct {
	mu    sync.Mutex
	state map[Token]*Partial
}

func NewResolver() *Resolver {
	return &Resolver{state: make(map[Token]*Partial)}
}

func (r *Resolver) Begin(size uint8) Token {
	r.mu.Lock()
	defer r.mu.Unlock()

	for {
		t := Token(rand.Uint32())
		if _, ok := r.state[t]; !ok {
			r.state[t] = &Partial{c: Cookie{Size: size}}
			return t
		}
	}
}

func (r *Resolver) End(t Token) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.state, t)
}

func (r *Resolver) Get(t Token) (*Partial, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, ok := r.state[t]
	return p, ok
}
