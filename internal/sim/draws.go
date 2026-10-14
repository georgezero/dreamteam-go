package sim

import "github.com/georgezero/dreamteam-go/internal/rng"

// drawKey identifies a keyed draw: domain and up to five keys.
type drawKey struct {
	domain string
	n      int
	k      [5]int64
}

// hash returns rng.Hash(seed, domain, keys...), memoized for the current
// day. A draw is a pure function of its key, so the memo changes no
// outcome; it saves the repeat hashes where one attribute group is read at
// two moments of the same day (an order scheduled at day start and
// disclosed later, an arrival scheduled and then happening, an exam
// started and ended). The memo is cleared at each day start.
func (st *state) hash(domain string, keys ...int64) *rng.Draws {
	if len(keys) > 5 {
		return rng.Hash(st.seed, domain, keys...)
	}
	k := drawKey{domain: domain, n: len(keys)}
	copy(k.k[:], keys)
	if l, ok := st.drawMemo[k]; ok {
		return rng.FromLanes(l)
	}
	d := rng.Hash(st.seed, domain, keys...)
	if st.drawMemo == nil {
		st.drawMemo = map[drawKey][4]uint64{}
	}
	st.drawMemo[k] = d.Lanes()
	return d
}
