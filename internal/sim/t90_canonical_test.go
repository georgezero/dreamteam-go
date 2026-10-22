package sim_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/eval"
	"github.com/georgezero/dreamteam-go/internal/scenario"
	"github.com/georgezero/dreamteam-go/internal/session"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// The game's T90 evaluation (session.T90) is the canonical development
// runner's on one seed: the plan and the no-action baseline recorded by
// devRunStrategy to the common horizon strategyTail sets, evaluated by the
// policy in force, give the same report, byte for byte.
func TestT90MatchesCanonicalRunner(t *testing.T) {
	if testing.Short() {
		t.Skip("runs eight 200-day worlds")
	}
	p, digest, err := eval.LoadPolicy("../../" + eval.PolicyPath)
	if err != nil {
		t.Fatal(err)
	}
	prot := &eval.Protocol{Policy: p, StrategySHA: map[string]string{}}
	const seed = 905
	batch := []api.Action{
		{Type: api.ActSetReadingAllocation, Op: "set", Employee: "E07", Effective: "2026-10-27", SiteAllocationBP: 6500},
		{Type: api.ActSetReadingAllocation, Op: "set", Employee: "E08", Effective: "2026-10-27", SiteAllocationBP: 5500},
	}
	s, err := session.Open("sustainable", seed)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range s.Apply(batch...) {
		if !r.OK {
			t.Fatalf("%s: %s", r.Code, r.Message)
		}
	}
	s.AdvanceTo(s.Now().Add(95 * simtime.Day))
	got, err := s.T90()
	if err != nil {
		t.Fatal(err)
	}

	f, err := scenario.ResolveFile("sustainable")
	if err != nil {
		t.Fatal(err)
	}
	fp, _ := scenario.Fingerprint(f)
	cfg, err := scenario.Compile(f)
	if err != nil {
		t.Fatal(err)
	}
	plans := []eval.Strategy{{Name: session.T90Baseline}, {Name: session.T90Plan, Batches: []eval.StrategyBatch{{Time: "2026-10-13T06:00:00", Actions: batch}}}}
	horizon := int64(0)
	for _, st := range plans {
		h, err := strategyTail(prot, cfg, st, seed)
		if err != nil {
			t.Fatal(err)
		}
		horizon = max(horizon, h)
	}
	records := map[string][]eval.Record{}
	for _, st := range plans {
		r, err := devRunStrategy(prot, cfg, fp, st, seed, horizon)
		if err != nil {
			t.Fatal(err)
		}
		if need := eval.TailEnd(p, r.TakeoverSec, r.LatestEffectSec, r.LatestCommitmentSec); need != horizon {
			t.Fatalf("%s: the horizon moves (%d, %d)", st.Name, horizon, need)
		}
		records[st.Name] = []eval.Record{r}
	}
	want := eval.Evaluate(p, digest, eval.Input{Baseline: session.T90Baseline, Plans: records})
	a, _ := json.Marshal(got.Report)
	b, _ := json.Marshal(want)
	if !bytes.Equal(a, b) {
		t.Fatalf("the game's T90 report differs from the canonical runner's (%d and %d bytes)", len(a), len(b))
	}
	if len(want.Integrity) > 0 || want.Comparisons[0].Verdict == "" {
		t.Fatalf("integrity %v", want.Integrity)
	}
	t.Logf("verdict %s · %s", want.Comparisons[0].Verdict, want.Comparisons[0].Classification)
}
