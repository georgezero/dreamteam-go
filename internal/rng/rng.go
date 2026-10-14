// Package rng provides keyed deterministic randomness and integer-only
// samplers.
//
// Every random attribute in the simulation has its own key: a domain name
// and origin keys that do not depend on player actions. The key is hashed
// with SHA-256 over a byte-exact encoding, so independent implementations
// agree and observed outputs do not reveal the seed.
//
// Most attributes need one to four values: Hash returns them straight from
// the digest (four 64-bit lanes). Attributes needing more draws use New, a
// ChaCha8 stream seeded with the same digest. No floating point is used.
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

// Source supplies raw 64-bit values. *Stream and *Draws implement it.
type Source interface {
	Uint64() uint64
}

// Stream is one keyed random stream with unlimited draws.
type Stream struct {
	c *rand.ChaCha8
}

// New returns the stream for (seed, domain, keys).
func New(seed Seed, domain string, keys ...int64) *Stream {
	return &Stream{c: rand.NewChaCha8(sha256.Sum256(KeyBytes(seed, domain, keys...)))}
}

// Uint64 returns the next raw 64-bit value.
func (s *Stream) Uint64() uint64 { return s.c.Uint64() }

// Uniform returns a value in [0, n); see Uniform.
func (s *Stream) Uniform(n uint64) uint64 { return Uniform(s, n) }

// Bernoulli returns true with probability p; see Bernoulli.
func (s *Stream) Bernoulli(p PPM) bool { return Bernoulli(s, p) }

// Weighted picks an index; see Weighted.
func (s *Stream) Weighted(weights []uint64) int { return Weighted(s, weights) }

// Draws holds the four 64-bit lanes of one key's SHA-256 digest: up to
// four values for about the cost of one hash.
type Draws struct {
	lanes [4]uint64
	next  int
}

// Hash returns the draws for (seed, domain, keys). The lanes are the digest
// read as big-endian uint64s; it is the same digest that seeds New.
func Hash(seed Seed, domain string, keys ...int64) *Draws {
	sum := sha256.Sum256(KeyBytes(seed, domain, keys...))
	d := &Draws{}
	for i := range d.lanes {
		d.lanes[i] = binary.BigEndian.Uint64(sum[8*i:])
	}
	return d
}

// Lanes returns the four lanes.
func (d *Draws) Lanes() [4]uint64 { return d.lanes }

// FromLanes rebuilds draws from lanes returned by Lanes (for callers that
// memoize a digest instead of recomputing it).
func FromLanes(l [4]uint64) *Draws { return &Draws{lanes: l} }

// Uint64 returns the next lane. It panics after four: an attribute that
// needs more draws must use a Stream.
func (d *Draws) Uint64() uint64 {
	if d.next == len(d.lanes) {
		panic("rng: more than four draws from one Hash; use a Stream")
	}
	v := d.lanes[d.next]
	d.next++
	return v
}

// Lane returns a one-draw source reading lane i (0–3) directly, so an
// attribute keeps its fixed lane whether or not other lanes are used
// (SIMULATION_DESIGN §4.3). It does not affect Uint64's sequence.
func (d *Draws) Lane(i int) Source { return &lane{v: d.lanes[i]} }

type lane struct {
	v    uint64
	used bool
}

func (l *lane) Uint64() uint64 {
	if l.used {
		panic("rng: a lane yields one draw")
	}
	l.used = true
	return l.v
}

// Uniform returns a value in [0, n); see Uniform.
func (d *Draws) Uniform(n uint64) uint64 { return Uniform(d, n) }

// Bernoulli returns true with probability p; see Bernoulli.
func (d *Draws) Bernoulli(p PPM) bool { return Bernoulli(d, p) }

// Weighted picks an index; see Weighted.
func (d *Draws) Weighted(weights []uint64) int { return Weighted(d, weights) }

// Uniform returns a value in [0, n) using the high word of a 64×64-bit
// product (one draw, no rejection). The bias is at most n/2^64, which is
// negligible for the ranges used here; powers of two are exact.
// Uniform panics if n == 0.
func Uniform(src Source, n uint64) uint64 {
	if n == 0 {
		panic("rng: Uniform(0)")
	}
	hi, _ := bits.Mul64(src.Uint64(), n)
	return hi
}

// PPM is a probability in parts per million.
type PPM uint32

// OneMillion is probability 1 in PPM.
const OneMillion PPM = 1_000_000

// Bernoulli returns true with probability p (one draw).
func Bernoulli(src Source, p PPM) bool {
	return Uniform(src, uint64(OneMillion)) < uint64(p)
}

// Weighted picks an index with probability proportional to weights (one
// draw). It panics if all weights are zero.
func Weighted(src Source, weights []uint64) int {
	var total uint64
	for _, w := range weights {
		total += w
	}
	u := Uniform(src, total)
	for i, w := range weights {
		if u < w {
			return i
		}
		u -= w
	}
	panic("rng: unreachable")
}
