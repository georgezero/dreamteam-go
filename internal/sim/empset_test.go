package sim

import (
	"math/rand/v2"
	"slices"
	"testing"
)

// The growable employee set agrees with a plain set for indices on both
// sides of the inline word, never aliases a copy's words, and orders sets
// as the V0.1 mask did for the first 64 indices.
func TestEmpSetAgainstReference(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	members := func(s empSet) []int {
		var out []int
		for i := s.next(0); i >= 0; i = s.next(i + 1) {
			out = append(out, i)
		}
		return out
	}
	for trial := range 500 {
		var s, u empSet
		ref, refU := map[int]bool{}, map[int]bool{}
		for range r.IntN(40) {
			i := r.IntN(300)
			switch r.IntN(3) {
			case 0, 1:
				s = s.with(i)
				ref[i] = true
			case 2:
				before := s
				snapshot := members(before)
				s = s.without(i)
				delete(ref, i)
				if !slices.Equal(members(before), snapshot) {
					t.Fatalf("trial %d: without changed the original set", trial)
				}
			}
			j := r.IntN(300)
			u = u.with(j)
			refU[j] = true
		}
		var want []int
		meets := false
		for i := range 300 {
			if ref[i] != s.has(i) {
				t.Fatalf("trial %d: has(%d) = %v", trial, i, s.has(i))
			}
			if ref[i] {
				want = append(want, i)
				meets = meets || refU[i]
			}
		}
		if got := members(s); !slices.Equal(got, want) || s.empty() != (len(want) == 0) || s.meets(u) != meets {
			t.Fatalf("trial %d: members %v, want %v", trial, got, want)
		}
		uSnap := members(u)
		all := s.union(u)
		for i := range 300 {
			if all.has(i) != (ref[i] || refU[i]) {
				t.Fatalf("trial %d: union at %d", trial, i)
			}
		}
		if !slices.Equal(members(u), uSnap) {
			t.Fatalf("trial %d: union changed an operand", trial)
		}
	}
	// Below 64 the order is the V0.1 mask comparison.
	a, b := empSet{}.with(1).with(5), empSet{}.with(2)
	if got, want := cmpSets(a, b), int(a.lo)-int(b.lo); got != want {
		t.Fatalf("cmpSets %d, mask order %d", got, want)
	}
	if cmpSets(empSet{}.with(70), empSet{}.with(71)) >= 0 || cmpSets(empSet{}.with(3), empSet{}.with(3)) != 0 {
		t.Fatal("cmpSets beyond the inline word")
	}
}

// A displaced patient's acceptance is measured from the tolerance anchor
// (the start of their current voluntary attempt), not from the displaced
// appointment's day, so a displacement never renews patience (§7.2).
func TestAcceptReplacementAnchor(t *testing.T) {
	const anchor, tol = 10, 3 // the displaced appointment was on day 15
	at := func(d int64) slotRef { return slotRef{day: d, idx: 1} }
	for _, c := range []struct {
		name            string
		pref, any       slotRef
		hasPref, hasAny bool
		want            string
		wantDay         int64
	}{
		{"preferred within the anchor's tolerance", at(12), at(11), true, true, resRebooked, 12},
		{"preferred too late, any in time", at(16), at(13), true, true, resRebooked, 13},
		{"within 3 days of the displaced day, not of the anchor", at(16), at(16), true, true, resDeclined, 0},
		{"only an any-window slot, too late", slotRef{}, at(14), false, true, resDeclined, 0},
		{"nothing in the horizon", slotRef{}, slotRef{}, false, false, resNoSlot, 0},
	} {
		pick, res := acceptReplacement(anchor, tol, c.pref, c.any, c.hasPref, c.hasAny)
		if res != c.want || (res == resRebooked && pick.day != c.wantDay) {
			t.Errorf("%s: %s day %d, want %s day %d", c.name, res, pick.day, c.want, c.wantDay)
		}
	}
}
