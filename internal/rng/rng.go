// Package rng provides keyed deterministic random streams and integer-only
// samplers.
//
// Every random attribute in the simulation draws from its own stream, keyed
// by a domain name and origin keys that do not depend on player actions.
// Streams are ChaCha8 generators seeded with SHA-256 over a byte-exact key
// encoding, so independent implementations agree and observed outputs do not
// reveal the seed. No floating point is used.
package rng

import (
	"crypto/sha256"
	"encoding/binary"
	"math/bits"
	"math/rand/v2"
)

// Seed is a 256-bit simulation seed.
type Seed [32]byte

// SeedFromInt expands a small integer (game and development seeds) into a
// Seed: SHA-256("dreamteam-seed" || 8-byte big-endian n).
func SeedFromInt(n int64) Seed {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], uint64(n))
	h := sha256.New()
	h.Write([]byte("dreamteam-seed"))
	h.Write(buf[:])
	var s Seed
	copy(s[:], h.Sum(nil))
	return s
}

// KeyBytes returns the byte string hashed to seed a stream:
//
//	"dreamteam/v1" || seed || enc(domain) || enc(k1) || ... || enc(kn)
//	enc(string) = uvarint(len) || utf8 bytes
//	enc(int64)  = 0x01 || 8-byte big-endian
func KeyBytes(seed Seed, domain string, keys ...int64) []byte {
	b := make([]byte, 0, 12+32+binary.MaxVarintLen64+len(domain)+9*len(keys))
	b = append(b, "dreamteam/v1"...)
	b = append(b, seed[:]...)
	b = binary.AppendUvarint(b, uint64(len(domain)))
	b = append(b, domain...)
	for _, k := range keys {
		b = append(b, 0x01)
		b = binary.BigEndian.AppendUint64(b, uint64(k))
	}
	return b
}

// Stream is one keyed random stream.
type Stream struct {
	c *rand.ChaCha8
}

// New returns the stream for (seed, domain, keys).
func New(seed Seed, domain string, keys ...int64) *Stream {
	return &Stream{c: rand.NewChaCha8(sha256.Sum256(KeyBytes(seed, domain, keys...)))}
}

// Uint64 returns the next raw 64-bit value.
func (s *Stream) Uint64() uint64 { return s.c.Uint64() }

// Uniform returns a value in [0, n) using the high word of a 64×64-bit
// product (one draw, no rejection). The bias is at most n/2^64, which is
// negligible for the ranges used here; powers of two are exact.
// Uniform panics if n == 0.
func (s *Stream) Uniform(n uint64) uint64 {
	if n == 0 {
		panic("rng: Uniform(0)")
	}
	hi, _ := bits.Mul64(s.c.Uint64(), n)
	return hi
}

// PPM is a probability in parts per million.
type PPM uint32

// OneMillion is probability 1 in PPM.
const OneMillion PPM = 1_000_000

// Bernoulli returns true with probability p.
func (s *Stream) Bernoulli(p PPM) bool {
	return s.Uniform(uint64(OneMillion)) < uint64(p)
}

// Weighted picks an index with probability proportional to weights.
// It panics if all weights are zero.
func (s *Stream) Weighted(weights []uint64) int {
	var total uint64
	for _, w := range weights {
		total += w
	}
	u := s.Uniform(total)
	for i, w := range weights {
		if u < w {
			return i
		}
		u -= w
	}
	panic("rng: unreachable")
}
