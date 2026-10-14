//go:build acceptance

package sim_test

import (
	"os"
	"strconv"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/rng"
	"github.com/georgezero/dreamteam-go/internal/sim"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// V0.1 acceptance criteria 9 and 10 at full scale (ROADMAP §2). Slow, so
// behind the acceptance build tag:
//
//	go test -tags acceptance ./internal/sim -run Acceptance -v -timeout 120m
//
// DT_SEEDS and DT_TRIALS shrink them for a quick look.

func envInt(name string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(name)); err == nil && v > 0 {
		return v
	}
	return def
}

// Criterion 10: every invariant (SIMULATION_DESIGN §10) holds after every
// timestamp transaction, 50 seeds × 90 days of the established practice
// from the takeover. The full check walks every record, so it costs
// minutes per seed; seeds run in parallel.
func TestAcceptanceInvariants(t *testing.T) {
	cfg := established(t)
	for seed := int64(1); seed <= int64(envInt("DT_SEEDS", 50)); seed++ {
		t.Run(strconv.FormatInt(seed, 10), func(t *testing.T) {
			t.Parallel()
			w := sim.New(cfg, rng.SeedFromInt(seed))
			w.Engine().RunUntil(w.StartTime(), nil)
			w.Engine().CheckInvariants = true
			if err := runTo(w, w.StartTime().Add(90*simtime.Day)); err != nil {
				t.Fatalf("seed %d: %v", seed, err)
			}
		})
	}
}

// Criterion 9: random chunkings of the real simulation (warm-up, takeover
// and one day of operations of the mini practice) give the same log,
// state and ledger as one uninterrupted run — 1,000 chunkings.
func TestAcceptanceChunking(t *testing.T) {
	cfg := miniConfig(t)
	ref := sim.New(cfg, rng.SeedFromInt(9))
	end := ref.StartTime().Add(simtime.Day)
	ref.Engine().RunUntil(end, nil)
	wantLog, wantState := ref.LogHash(), ref.StateDigest()
	chunker := rng.New(rng.SeedFromInt(9), "acceptance.chunks")
	for trial := 0; trial < envInt("DT_TRIALS", 1000); trial++ {
		w := sim.New(cfg, rng.SeedFromInt(9))
		for w.Engine().Now() < end {
			var step simtime.Duration
			switch chunker.Uniform(4) {
			case 0:
				step = 0
			case 1:
				step = simtime.Duration(chunker.Uniform(60))
			case 2:
				step = simtime.Duration(chunker.Uniform(uint64(6 * simtime.Hour)))
			default:
				step = simtime.Duration(chunker.Uniform(uint64(20 * simtime.Day)))
			}
			w.Engine().RunUntil(min(w.Engine().Now().Add(step), end), nil)
		}
		if w.LogHash() != wantLog || w.StateDigest() != wantState {
			t.Fatalf("trial %d: chunked run differs", trial)
		}
	}
}
