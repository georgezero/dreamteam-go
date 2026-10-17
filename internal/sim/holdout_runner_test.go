package sim_test

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/eval"
	"github.com/georgezero/dreamteam-go/internal/rng"
	"github.com/georgezero/dreamteam-go/internal/scenario"
	"github.com/georgezero/dreamteam-go/internal/sim"
	"github.com/georgezero/dreamteam-go/internal/simtime"
	"github.com/georgezero/dreamteam-go/internal/version"
)

// The canonical holdout runner (docs/evaluation/README.md §9, Slice
// 5C.0b). It is the only path that runs a holdout seed: it takes an
// authorized protocol (eval.Authorize), runs every strategy of the frozen
// set on every holdout seed to one common horizon, evaluates the matrix
// with the frozen policy, and commits the run manifest and report once;
// a later run must reproduce them byte for byte.
//
//	DT_HOLDOUT_RUN=1 go test ./internal/sim -run 'TestHoldoutRun$' -v -timeout 600m
//
// Without DT_HOLDOUT_RUN, TestHoldoutRun only checks that the repository's
// holdout is sealed.

// runnerManifest is the scenario of a strategy set: a calibration
// manifest's scenario and overlay.
type runnerManifest struct {
	Scenario string          `json:"scenario"`
	Overlay  json.RawMessage `json:"overlay,omitempty"`
}

// strategyConfig compiles the strategy set's scenario: its config, with
// an open-ended evaluation window, and its fingerprint.
func strategyConfig(p *eval.Protocol) (*sim.Config, string, error) {
	raw, err := os.ReadFile(filepath.Join(p.Root, filepath.FromSlash(p.Set.ScenarioManifest)))
	if err != nil {
		return nil, "", err
	}
	var m runnerManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, "", err
	}
	f, err := scenario.ResolveFile(m.Scenario)
	if err != nil {
		return nil, "", err
	}
	if len(m.Overlay) > 0 {
		canon, err := scenario.Canonical(f)
		if err != nil {
			return nil, "", err
		}
		var b, o any
		json.Unmarshal(canon, &b)
		if err := json.Unmarshal(m.Overlay, &o); err != nil {
			return nil, "", err
		}
		merged, _ := json.Marshal(scenario.Merge(b, o))
		if f, err = scenario.Decode(merged); err != nil {
			return nil, "", err
		}
	}
	fp, err := scenario.Fingerprint(f)
	if err != nil {
		return nil, "", err
	}
	cfg, err := scenario.Compile(f)
	if err != nil {
		return nil, "", err
	}
	cfg.EvaluationDays = 0
	return cfg, fp, nil
}

type timedBatch struct {
	at simtime.Time
	b  eval.StrategyBatch
}

// strategyBatches are a strategy's batches in time order, each from the
// takeover and before the primary endpoint.
func strategyBatches(p *eval.Protocol, w *sim.World, st eval.Strategy) ([]timedBatch, error) {
	start := w.StartTime()
	t90 := start.Add(simtime.Duration(p.Policy.Horizons.PrimaryDays) * simtime.Day)
	var out []timedBatch
	for _, b := range st.Batches {
		at, err := w.Calendar().ParseISO(b.Time)
		if err != nil || at < start || at >= t90 {
			return nil, fmt.Errorf("%s: batch time %q is not in [takeover, T90): %v", st.Name, b.Time, err)
		}
		out = append(out, timedBatch{at, b})
	}
	slices.SortStableFunc(out, func(x, y timedBatch) int { return cmp.Compare(x.at, y.at) })
	return out, nil
}

// strategyTail is a strategy's own tail end on seed: run to its last
// batch, apply them all, and read the latest effect and commitment.
func strategyTail(p *eval.Protocol, cfg *sim.Config, st eval.Strategy, seed int64) (int64, error) {
	w := sim.New(cfg, rng.SeedFromInt(seed))
	w.Engine().RunUntil(w.StartTime(), nil)
	bs, err := strategyBatches(p, w, st)
	if err != nil {
		return 0, err
	}
	var last int64
	for _, b := range bs {
		w.Engine().RunUntil(b.at, nil)
		for _, res := range w.Apply(b.b.Actions) {
			if res.OK {
				last = int64(b.at)
			}
		}
	}
	return eval.TailEnd(p.Policy, int64(w.StartTime()), w.LatestEffect(), w.LatestCommitment(last)), nil
}

// runStrategy records strategy st on seed to the common horizon: daily
// samples, the captures, its batches at their instants (before a sample
// or capture at the same instant).
func runStrategy(p *eval.Protocol, cfg *sim.Config, fp string, st eval.Strategy, seed, horizon int64) (eval.Record, error) {
	w := sim.New(cfg, rng.SeedFromInt(seed))
	start := w.StartTime()
	w.Engine().RunUntil(start, nil)
	rec := w.NewHoldoutRecorder(p, st.Name, p.StrategySHA[st.Name], fp, seed)
	bs, err := strategyBatches(p, w, st)
	if err != nil {
		return eval.Record{}, err
	}
	h := p.Policy.Horizons
	day := func(n int) simtime.Time { return start.Add(simtime.Duration(n) * simtime.Day) }
	end := simtime.Time(eval.ObservationEnd(p.Policy, int64(start), w.Calendar().Date(start).String(), horizon))
	type mark struct {
		at    simtime.Time
		order int // 0 batch, 1 sample, 2 capture
		batch int
		day   int
		name  string
	}
	var marks []mark
	for i, b := range bs {
		marks = append(marks, mark{at: b.at, batch: i})
	}
	for d := 1; day(d) <= end; d++ {
		marks = append(marks, mark{at: day(d), order: 1, day: d})
	}
	for _, c := range []mark{{at: day(h.FeedbackDays), name: "T30"}, {at: day(h.PrimaryDays), name: "T90"}, {at: simtime.Time(horizon), name: "tailEnd"},
		{at: day(h.ValidationDays), name: "T180"}, {at: end, name: "end"}} {
		c.order = 2
		marks = append(marks, c)
	}
	slices.SortStableFunc(marks, func(x, y mark) int { return cmp.Or(cmp.Compare(x.at, y.at), cmp.Compare(x.order, y.order)) })
	var last int64
	for _, m := range marks {
		w.Engine().RunUntil(m.at, nil)
		switch m.order {
		case 0:
			for _, res := range w.Apply(bs[m.batch].b.Actions) {
				if res.OK {
					last = int64(m.at)
				}
			}
		case 1:
			rec.Sample(m.day)
		default:
			rec.Capture(m.name)
		}
	}
	if err := w.Check(); err != nil {
		return eval.Record{}, err
	}
	return rec.Finish(last, w.LatestEffect(), horizon, st.TeleOnlyOf), nil
}

// runHoldout runs an authorized protocol's whole matrix and commits it to
// out: eval.RunWritten the first time, eval.RunReproduced on an identical
// rerun, an error (nothing written) otherwise.
func runHoldout(t *testing.T, p *eval.Protocol, out string) (string, *eval.Report, error) {
	t.Helper()
	cfg, fp, err := strategyConfig(p)
	if err != nil {
		return "", nil, err
	}
	// The common horizon: every strategy's own tail end, from its first
	// seed's accepted actions; the records then confirm it on every seed.
	horizon := int64(0)
	for _, name := range p.Plans() {
		h, err := strategyTail(p, cfg, p.Strategies[name], p.Seeds[0])
		if err != nil {
			return "", nil, err
		}
		horizon = max(horizon, h)
	}
	var records map[string][]eval.Record
	for pass := 0; ; pass++ {
		records = map[string][]eval.Record{}
		need := horizon
		for _, name := range p.Plans() {
			for _, seed := range p.Seeds {
				r, err := runStrategy(p, cfg, fp, p.Strategies[name], seed, horizon)
				if err != nil {
					return "", nil, err
				}
				records[name] = append(records[name], r)
				need = max(need, eval.TailEnd(p.Policy, r.TakeoverSec, r.LatestEffectSec, r.LatestCommitmentSec))
			}
		}
		if need == horizon {
			break
		}
		if pass > 0 {
			return "", nil, fmt.Errorf("the common horizon does not settle (%d, then %d)", horizon, need)
		}
		t.Logf("a seed's actions reach past the first seed's: the horizon moves from %d to %d, and every plan runs again", horizon, need)
		horizon = need
	}
	rep := eval.Evaluate(p.Policy, p.PolicyDigest, eval.Input{Baseline: p.Set.Baseline, Plans: records, Seeds: p.Seeds,
		PlanSHA: p.StrategySHA, Primary: p.Unseal.Primary})
	report, err := json.MarshalIndent(rep, "", " ")
	if err != nil {
		return "", nil, err
	}
	report = append(report, '\n')
	run, err := p.NewRun(records, horizon, report)
	if err != nil {
		return "", nil, err
	}
	mode, err := eval.CommitRun(out, run, report)
	return mode, rep, err
}

// runDir is where a protocol's run is committed.
func runDir(p *eval.Protocol) string {
	return filepath.Join(p.Root, "docs", "evaluation", "holdout-runs", p.Holdout.ID)
}

// The repository's canonical run. It needs the owner's committed unseal
// record; until then it only confirms the holdout is sealed.
func TestHoldoutRun(t *testing.T) {
	p, err := eval.Authorize("../..", eval.RepoPaths, eval.GitRepo{Root: "../.."}, version.Number)
	if errors.Is(err, eval.ErrSealed) {
		if _, statErr := os.Stat("../../" + eval.RepoPaths.Unseal); errors.Is(statErr, os.ErrNotExist) {
			if os.Getenv("DT_HOLDOUT_RUN") != "" {
				t.Fatal(err)
			}
			t.Skip(err)
		}
	}
	if err != nil {
		t.Fatal(err) // an unseal record that does not check out is never a skip
	}
	if os.Getenv("DT_HOLDOUT_RUN") == "" {
		t.Skip("the holdout is unsealed; set DT_HOLDOUT_RUN=1 for the one-time run (or its reproduction)")
	}
	mode, rep, err := runHoldout(t, p, runDir(p))
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range rep.Comparisons {
		t.Logf("%s (%s) against %s: %s, %s", c.Candidate, c.Role, c.Against, c.Verdict, c.Classification)
	}
	t.Logf("holdout run %s in %s", mode, runDir(p))
}
