package sim

import "math/bits"

// empSet is a set of employee registry indices (V0.2_DESIGN §5.1): a bit
// per employee, the first 64 inline (no allocation, as the V0.1 uint64
// mask), any beyond in hi. The registry is append-only and never reuses an
// index, so a practice may employ any number of people over its lifetime.
// Values are immutable once shared: operations that change a set with
// words in hi copy them first.
type empSet struct {
	lo uint64
	hi []uint64 // hi[k] holds indices 64(k+1) … 64(k+1)+63
}

func (s empSet) has(i int) bool {
	if i < 64 {
		return s.lo&(1<<uint(i)) != 0
	}
	k := i/64 - 1
	return k < len(s.hi) && s.hi[k]&(1<<uint(i%64)) != 0
}

func (s empSet) empty() bool {
	if s.lo != 0 {
		return false
	}
	for _, w := range s.hi {
		if w != 0 {
			return false
		}
	}
	return true
}

// with returns s plus i.
func (s empSet) with(i int) empSet {
	if i < 64 {
		s.lo |= 1 << uint(i)
		return s
	}
	k := i/64 - 1
	hi := make([]uint64, max(len(s.hi), k+1))
	copy(hi, s.hi)
	hi[k] |= 1 << uint(i%64)
	s.hi = hi
	return s
}

// without returns s less i.
func (s empSet) without(i int) empSet {
	if i < 64 {
		s.lo &^= 1 << uint(i)
		return s
	}
	k := i/64 - 1
	if k >= len(s.hi) || s.hi[k]&(1<<uint(i%64)) == 0 {
		return s
	}
	s.hi = append([]uint64(nil), s.hi...)
	s.hi[k] &^= 1 << uint(i%64)
	return s
}

// union returns s ∪ t.
func (s empSet) union(t empSet) empSet {
	s.lo |= t.lo
	if len(t.hi) == 0 {
		return s
	}
	hi := make([]uint64, max(len(s.hi), len(t.hi)))
	copy(hi, s.hi)
	for k, w := range t.hi {
		hi[k] |= w
	}
	s.hi = hi
	return s
}

// meets reports whether s and t share a member.
func (s empSet) meets(t empSet) bool {
	if s.lo&t.lo != 0 {
		return true
	}
	for k := 0; k < len(s.hi) && k < len(t.hi); k++ {
		if s.hi[k]&t.hi[k] != 0 {
			return true
		}
	}
	return false
}

// next returns the smallest member ≥ from, or -1.
func (s empSet) next(from int) int {
	if from < 64 {
		if w := s.lo >> uint(from); w != 0 {
			return from + bits.TrailingZeros64(w)
		}
		from = 64
	}
	for k := from/64 - 1; k < len(s.hi); k++ {
		w := s.hi[k]
		if base := 64 * (k + 1); from > base {
			w >>= uint(from - base)
			if w != 0 {
				return from + bits.TrailingZeros64(w)
			}
			continue
		}
		if w != 0 {
			return 64*(k+1) + bits.TrailingZeros64(w)
		}
	}
	return -1
}

// cmpSets orders sets deterministically: by the first 64 indices as the
// V0.1 mask compared (int(a) − int(b)), then by the remaining words.
func cmpSets(a, b empSet) int {
	if a.lo != b.lo {
		return int(a.lo) - int(b.lo)
	}
	for k := 0; k < max(len(a.hi), len(b.hi)); k++ {
		var x, y uint64
		if k < len(a.hi) {
			x = a.hi[k]
		}
		if k < len(b.hi) {
			y = b.hi[k]
		}
		if x != y {
			if x < y {
				return -1
			}
			return 1
		}
	}
	return 0
}
