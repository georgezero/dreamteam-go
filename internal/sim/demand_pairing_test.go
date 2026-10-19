package sim_test

import (
	"cmp"
	"maps"
	"slices"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/eval"
)

// The premise of v0.2-eval-5's lost-demand pairing (docs/calibration/
// clinical-5c4c/access-gates.md §3): the simulator's demand is exogenous.
// On one seed, plans that act differently (no action; a teleradiology
// contract; walk-ins scheduled later and another worklist policy, with
// their different bookings, losses and readings) receive the same orders: the same receipt and wanted times,
// priorities, modalities and walk-in flags, and since Slice 5C.4d each
// under the same immutable origin identity, unique within a record. The
// mini practice and the short policy (test fixtures, no development
// strategy).
func TestDemandExogenousAcrossPlans(t *testing.T) {
	type key struct {
		recv, want int64
		q, mod     string
		walk       bool
	}
	demand := func(r eval.Record) []key {
		var ks []key
		for _, o := range r.Orders {
			if o.ReceivedSec > r.TakeoverSec {
				ks = append(ks, key{o.ReceivedSec, o.WantedSec, o.Priority, o.Modality, o.WalkIn})
			}
		}
		slices.SortFunc(ks, func(a, b key) int {
			return cmp.Or(cmp.Compare(a.recv, b.recv), cmp.Compare(a.want, b.want), cmp.Compare(a.q, b.q), cmp.Compare(a.mod, b.mod))
		})
		return ks
	}
	outcomes := func(r eval.Record) map[string]int {
		m := map[string]int{}
		for _, o := range r.Captures[len(r.Captures)-1].OrderOutcomes {
			m[o]++
		}
		return m
	}
	// byOrigin: each order received after the takeover by its origin, its
	// request; every origin present and unique.
	byOrigin := func(r eval.Record) map[string]key {
		m := map[string]key{}
		for _, o := range r.Orders {
			if o.Origin == "" {
				t.Fatalf("seed %d: order %s without an origin", r.Seed, o.ID)
			}
			if _, dup := m[o.Origin]; dup {
				t.Fatalf("seed %d: origin %s twice", r.Seed, o.Origin)
			}
			if o.ReceivedSec > r.TakeoverSec {
				m[o.Origin] = key{o.ReceivedSec, o.WantedSec, o.Priority, o.Modality, o.WalkIn}
			}
		}
		return m
	}
	p, _ := shortPolicy(t, miniConfig(t))
	for _, seed := range []int64{41, 42} {
		a, _ := recordMini(t, p, seed, false, true)
		b, _ := recordMini(t, p, seed, true, true)
		da, db := demand(a), demand(b)
		if len(da) == 0 || !slices.Equal(da, db) {
			t.Fatalf("seed %d: a teleradiology contract changed the demand (%d and %d orders)", seed, len(da), len(db))
		}
		// Walk-ins scheduled later and another worklist policy from the
		// takeover: different bookings, losses and readings, the same demand.
		c, _ := recordMiniActs(t, p, seed, false, true, []api.Action{{Type: api.ActSetWalkInPolicy, Policy: api.WalkInScheduleLater},
			{Type: api.ActSetWorklistPolicy, Policy: api.WorklistShortestFirst}})
		if dc := demand(c); !slices.Equal(da, dc) {
			t.Fatalf("seed %d: the walk-in and worklist policies changed the demand (%d and %d orders)", seed, len(da), len(dc))
		}
		// The same origins, each the same request, under all three plans:
		// identity, not just the same demand population.
		oa := byOrigin(a)
		if !maps.Equal(oa, byOrigin(b)) || !maps.Equal(oa, byOrigin(c)) {
			t.Fatalf("seed %d: origins or their requests differ between plans", seed)
		}
		if maps.Equal(outcomes(a), outcomes(c)) {
			t.Fatalf("seed %d: the policies changed no outcome: no test", seed)
		}
		t.Logf("seed %d: %d orders after the takeover; outcomes %v, %v (contract), %v (policies)", seed, len(da), outcomes(a), outcomes(b), outcomes(c))
	}
}
