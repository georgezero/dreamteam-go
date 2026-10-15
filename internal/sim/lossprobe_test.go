package sim_test

import (
	"testing"

	"github.com/georgezero/dreamteam-go/internal/rng"
	"github.com/georgezero/dreamteam-go/internal/sim"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// The booking-loss probe (V0.2 Step 0B) is passive: with it set, the log,
// ledger, state and action digests are those of the same run without it,
// every lost order is diagnosed exactly once, and no diagnosis finds a slot
// FirstFit should have booked.
func TestLossProbeIsPassive(t *testing.T) {
	cfg := established(t)
	run := func(probe bool) (*sim.World, []sim.LossDiagnosis) {
		w := sim.New(cfg, rng.SeedFromInt(3))
		var got []sim.LossDiagnosis
		if probe {
			w.SetLossProbe(func(d sim.LossDiagnosis) { got = append(got, d) })
		}
		w.Engine().RunUntil(w.StartTime().Add(21*simtime.Day), nil)
		return w, got
	}
	plain, _ := run(false)
	probed, diags := run(true)
	if plain.LogHash() != probed.LogHash() || plain.LedgerHash() != probed.LedgerHash() ||
		plain.StateDigest() != probed.StateDigest() || plain.ActionDigest() != probed.ActionDigest() {
		t.Fatal("the loss probe changed the run")
	}
	lost := map[string]bool{}
	for _, o := range probed.Orders() {
		if o.State == "lost" {
			lost[o.ID] = true
		}
	}
	seen := map[string]bool{}
	for _, d := range diags {
		if seen[d.Order] || !lost[d.Order] {
			t.Fatalf("%s: diagnosed twice or not lost", d.Order)
		}
		seen[d.Order] = true
		if d.Unexplained != 0 || d.Primary == "unexplained" {
			t.Fatalf("%s: a free bookable slot in the window: %+v", d.Order, d)
		}
		if d.Slots != d.Booked+d.Unstaffed+d.NoSupervision+d.Notice+d.SharedConflict || len(d.SharedAt) != d.SharedConflict {
			t.Fatalf("%s: slot reasons do not add up: %+v", d.Order, d)
		}
	}
	if len(seen) != len(lost) || len(lost) == 0 {
		t.Fatalf("%d lost orders, %d diagnosed", len(lost), len(seen))
	}
}
