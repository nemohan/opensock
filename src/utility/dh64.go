package utility

import (
	"math/rand"
	"time"
	//"math/rand"
	//"time"
)

const (
	dh64p = 0xffffffffffffffc5
	dh64g = 5
)

//
type DH64 struct {
	source rand.Source
	r      *rand.Rand
}

//
func (dh *DH64) Init() {
	dh.source = rand.NewSource(time.Now().UnixNano())
	dh.r = rand.New(dh.source)
}

func (dh *DH64) Secret(privateKey, pubKey uint64) uint64 {
	return dh.powModP(pubKey, privateKey)
}

//
func (dh *DH64) DH64KeyPair() (private, pub uint64) {
	r := dh.r
	a := r.Uint64()
	b := r.Uint64() & 0xffff
	c := r.Uint64() & 0xffff
	d := (rand.Uint64() & 0xffff) + 1

	privateKey := a<<48 | b<<32 | c<<16 | d
	pubKey := dh.powModP(dh64g, privateKey)
	return privateKey, pubKey
}

func (dh *DH64) mulModP(a, b uint64) uint64 {
	var m uint64
	for b != 0 {
		if (b & 1) != 0 {
			t := dh64p - a
			if m >= t {
				m -= t
			} else {
				m += a
			}
		}

		if a >= dh64p-a {
			a = a*2 - dh64p
		} else {
			a = a * 2
		}
		b >>= 1
	}
	return m
}

func (dh *DH64) powModP2(a, b uint64) uint64 {
	if b == 1 {
		return a
	}
	t := dh.powModP2(a, b>>1)
	t = dh.mulModP(t, t)
	if (b % 2) != 0 {
		t = dh.mulModP(t, a)
	}
	return t
}

func (dh *DH64) powModP(a, b uint64) uint64 {
	if a == 0 || b == 0 {
		return 1
	}
	if a > dh64p {
		a %= dh64p
	}
	return dh.powModP2(a, b)
}
