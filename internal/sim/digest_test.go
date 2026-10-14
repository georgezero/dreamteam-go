package sim_test

import (
	"reflect"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/rng"
	"github.com/georgezero/dreamteam-go/internal/sim"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// The state digest is a function of outcomes: unrelated events (handled
// or stale), reading observations and previewing actions change none of
// it, nor do rejected actions once the clock moves on.
func TestStateDigestIndependentOfNoiseAndReads(t *testing.T) {
	cfg := miniConfig(t)
	plain := sim.New(cfg, rng.SeedFromInt(3))
	noisy := sim.New(cfg, rng.SeedFromInt(3))
	cal := plain.Calendar()
	for d := int64(0); d < 3; d++ {
		start := cal.StartOfDay(d)
		for _, w := range []*sim.World{plain, noisy} {
			if err := runTo(w, start.Add(1)); err != nil {
				t.Fatal(err)
			}
		}
		for x := start.Add(7 * simtime.Minute); x < start.Add(simtime.Day-2); x = x.Add(7 * simtime.Minute) {
			noisy.PushLiveNoise(x)
			noisy.PushStaleNoise(x.Add(3))
		}
		for _, w := range []*sim.World{plain, noisy} {
			if err := runTo(w, start.Add(13*simtime.Hour)); err != nil {
				t.Fatal(err)
			}
		}
		_ = noisy.Observe(500)
		_ = noisy.History(10)
		_ = noisy.Evaluation()
		_ = noisy.Validate(extend("E06", 60))
		noisy.Apply([]api.Action{extend("E99", 5)}) // rejected
		for _, w := range []*sim.World{plain, noisy} {
			if err := runTo(w, cal.StartOfDay(d+1)); err != nil {
				t.Fatal(err)
			}
		}
		if plain.StateDigest() != noisy.StateDigest() || plain.LogHash() != noisy.LogHash() {
			pa, pb := plain.StateDigestParts(), noisy.StateDigestParts()
			for k := range pa {
				if pa[k] != pb[k] {
					t.Log("differs:", k)
				}
			}
			t.Fatalf("day %d: noise, reads or a rejected action changed the state digest", d)
		}
	}
}

// The state digest covers hidden state: two worlds with the same past and
// a different hidden future (orders not yet received) observe identically
// but digest differently, because the future events are pending.
func TestStateDigestCoversHiddenState(t *testing.T) {
	cfg := miniConfig(t)
	a := sim.New(cfg, rng.SeedFromInt(4))
	b := sim.New(cfg, rng.SeedFromInt(4))
	T := a.Calendar().StartOfDate(cfg.Start.AddDays(2)).Add(13 * simtime.Hour)
	b.DropOrdersFrom(T)
	for _, w := range []*sim.World{a, b} {
		w.Engine().RunUntil(T-1, nil)
	}
	if !reflect.DeepEqual(a.Observe(2000), b.Observe(2000)) || a.LogHash() != b.LogHash() || a.LedgerHash() != b.LedgerHash() {
		t.Fatal("the observable pasts differ: the test is vacuous")
	}
	if a.StateDigest() == b.StateDigest() {
		t.Fatal("the state digest missed a difference in hidden pending events")
	}
}

// Same seed and history, same digest; another seed, another digest.
func TestStateDigestDeterministic(t *testing.T) {
	cfg := miniConfig(t)
	x, y, z := run(t, cfg, 5, 4), run(t, cfg, 5, 4), run(t, cfg, 6, 4)
	if x.StateDigest() != y.StateDigest() || x.StateDigest() == z.StateDigest() || len(x.StateDigest()) != 64 {
		t.Fatal("state digest not a deterministic function of the run")
	}
}
