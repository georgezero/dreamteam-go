package sim_test

import (
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/eval"
	"github.com/georgezero/dreamteam-go/internal/rng"
	"github.com/georgezero/dreamteam-go/internal/sim"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// The development runner of Slice 5C.1 (calib_strategy_test.go drives it
// on the development seeds): the canonical holdout runner's run loop with
// the development recorder. TestDevelopmentRunnerMatchesCanonical shows
// that it records exactly what the canonical runner records.

// devRun is a development run's manifest: what ran, and the digest of
// every record and of the report.
type devRun struct {
	Schema        string           `json:"schema"`
	Set           string           `json:"set"`
	SetSHA256     string           `json:"setSHA256"`
	PolicyDigest  string           `json:"policyDigest"`
	Engine        string           `json:"engine"`
	Seeds         []int64          `json:"seeds"`
	HorizonEndSec int64            `json:"horizonEndSec"`
	Records       []eval.RunRecord `json:"records"`
	ReportSHA256  string           `json:"reportSHA256"`
	// TrajectorySHA256: the daily samples' digest (dreamteam.devrun/v2,
	// whose report and trajectory are kept outside the repository).
	TrajectorySHA256 string `json:"trajectorySHA256,omitempty"`
}

// devRunStrategy is runStrategy (the canonical runner's) with the
// development recorder.
func devRunStrategy(p *eval.Protocol, cfg *sim.Config, fp string, st eval.Strategy, seed, horizon int64) (eval.Record, error) {
	w := sim.New(cfg, rng.SeedFromInt(seed))
	start := w.StartTime()
	w.Engine().RunUntil(start, nil)
	rec := w.NewEvalRecorder(st.Name, p.StrategySHA[st.Name], fp, seed, p.Policy.Finance.EmployeeCommitmentDays)
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
				} else {
					return eval.Record{}, fmt.Errorf("%s seed %d: batch at %s refused: %s %s", st.Name, seed, bs[m.batch].b.Time, res.Code, res.Message)
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

// devMatrix runs every plan on every seed to the common horizon, as
// runHoldout does: the horizon from the first seed's accepted actions,
// confirmed by every record (and every plan rerun once if a seed reaches
// further).
func devMatrix(t *testing.T, p *eval.Protocol, parallel int, keep func(eval.Record)) (map[string][]eval.Record, int64) {
	t.Helper()
	cfg, fp, err := strategyConfig(p)
	if err != nil {
		t.Fatal(err)
	}
	horizon := int64(0)
	for _, name := range p.Plans() {
		h, err := strategyTail(p, cfg, p.Strategies[name], p.Seeds[0])
		if err != nil {
			t.Fatal(err)
		}
		horizon = max(horizon, h)
	}
	for pass := 0; ; pass++ {
		records := map[string][]eval.Record{}
		for _, name := range p.Plans() {
			records[name] = make([]eval.Record, len(p.Seeds))
		}
		var (
			mu   sync.Mutex
			wg   sync.WaitGroup
			errs []error
			sem  = make(chan struct{}, parallel)
		)
		for _, name := range p.Plans() {
			for i, seed := range p.Seeds {
				wg.Add(1)
				go func() {
					defer wg.Done()
					sem <- struct{}{}
					defer func() { <-sem }()
					r, err := devRunStrategy(p, cfg, fp, p.Strategies[name], seed, horizon)
					mu.Lock()
					defer mu.Unlock()
					if err != nil {
						errs = append(errs, err)
						return
					}
					records[name][i] = r
					t.Logf("%s seed %d recorded", name, seed)
				}()
			}
		}
		wg.Wait()
		if len(errs) > 0 {
			t.Fatal(errors.Join(errs...))
		}
		need := horizon
		for _, name := range p.Plans() {
			for _, r := range records[name] {
				need = max(need, eval.TailEnd(p.Policy, r.TakeoverSec, r.LatestEffectSec, r.LatestCommitmentSec))
			}
		}
		if need == horizon {
			for _, name := range p.Plans() {
				for _, r := range records[name] {
					keep(r)
				}
			}
			return records, horizon
		}
		if pass > 0 {
			t.Fatalf("the common horizon does not settle (%d, then %d)", horizon, need)
		}
		t.Logf("the horizon moves from %d to %d, and every plan runs again", horizon, need)
		horizon = need
	}
}

// writeOnce writes files into dir if it holds none of them; otherwise
// every one must already hold exactly these bytes.
func writeOnce(dir string, files map[string][]byte) (string, error) {
	names := slices.Sorted(func(yield func(string) bool) {
		for n := range files {
			if !yield(n) {
				return
			}
		}
	})
	present := 0
	for _, n := range names {
		if _, err := os.Stat(filepath.Join(dir, n)); err == nil {
			present++
		}
	}
	if present > 0 {
		for _, n := range names {
			old, err := os.ReadFile(filepath.Join(dir, n))
			if err != nil || !bytes.Equal(old, files[n]) {
				return "", fmt.Errorf("%s: %s does not reproduce the committed run", dir, n)
			}
		}
		return eval.RunReproduced, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	for _, n := range names {
		if err := os.WriteFile(filepath.Join(dir, n), files[n], 0o644); err != nil {
			return "", err
		}
	}
	return eval.RunWritten, nil
}

// The development runner records, plan by plan and seed by seed, exactly
// the canonical runner's records (same bytes), and the matrix settles on
// the same horizon.
func TestDevelopmentRunnerMatchesCanonical(t *testing.T) {
	f := newProtoFixture(t)
	p, err := f.authorize()
	if err != nil {
		t.Fatal(err)
	}
	cfg, fp, err := strategyConfig(p)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]map[int64][]byte{}
	records, horizon := devMatrix(t, p, 2, func(r eval.Record) {
		raw, _ := json.Marshal(r)
		if got[r.Plan] == nil {
			got[r.Plan] = map[int64][]byte{}
		}
		got[r.Plan][r.Seed] = raw
	})
	if len(records) != 2 || horizon <= 0 {
		t.Fatalf("%d plans, horizon %d", len(records), horizon)
	}
	for _, name := range p.Plans() {
		for _, seed := range p.Seeds {
			want, err := runStrategy(p, cfg, fp, p.Strategies[name], seed, horizon)
			if err != nil {
				t.Fatal(err)
			}
			raw, _ := json.Marshal(want)
			if !bytes.Equal(raw, got[name][seed]) {
				t.Fatalf("%s seed %d: the development record differs from the canonical runner's", name, seed)
			}
		}
	}
	// A batch at a sample instant (06:00 on day 3) applies before the
	// sample, as in the canonical runner.
	w := sim.New(cfg, rng.SeedFromInt(p.Seeds[0]))
	start := w.StartTime()
	late := eval.Strategy{Schema: eval.StrategySchema, Name: "L", Batches: []eval.StrategyBatch{{Time: w.Calendar().ISO(start.Add(3 * simtime.Day)),
		Actions: []api.Action{{Type: api.ActContractTeleradiology, Op: api.TeleOpStart, RequestID: "late", Plan: "standard-v1",
			Windows: []string{"weeknights"}, Scope: "all", Effective: w.Calendar().Date(start.Add(17 * simtime.Day)).String()}}}}}
	want, err := runStrategy(p, cfg, fp, late, p.Seeds[0], horizon)
	if err != nil {
		t.Fatal(err)
	}
	dev, err := devRunStrategy(p, cfg, fp, late, p.Seeds[0], horizon)
	if err != nil {
		t.Fatal(err)
	}
	wr, _ := json.Marshal(want)
	dr, _ := json.Marshal(dev)
	if !bytes.Equal(wr, dr) || want.LastActionSec != int64(start.Add(3*simtime.Day)) {
		t.Fatal("a batch at a sample instant: the development record differs from the canonical runner's")
	}
	if _, _, err := runHoldout(t, p, filepath.Join(f.root, "runs")); err != nil {
		t.Fatal(err)
	}
	var run eval.Run
	raw, _ := os.ReadFile(filepath.Join(f.root, "runs", "run.json"))
	json.Unmarshal(raw, &run)
	if run.HorizonEndSec != horizon {
		t.Fatalf("horizon %d, the canonical run's %d", horizon, run.HorizonEndSec)
	}
	for _, x := range run.Records {
		if sha256Hex(got[x.Plan][x.Seed]) != x.SHA256 {
			t.Fatalf("%s seed %d: digest differs from the canonical run's", x.Plan, x.Seed)
		}
	}
}

// eval.Development loads a set as the holdout contract does, for
// development seeds only, and never yields an authorized protocol.
func TestDevelopmentSet(t *testing.T) {
	f := newProtoFixture(t)
	pol, digest := shortPolicy(t, miniConfig(t))
	dev, err := eval.DevelopmentSeeds(pol)
	if err != nil || len(dev) != 10 || dev[0] != 901 || dev[9] != 910 {
		t.Fatalf("%v %v", dev, err)
	}
	p, setSHA, err := eval.Development(f.root, fixSet, pol, digest, dev)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(f.path(fixSet))
	if setSHA != sha256Hex(raw) || p.Authorized() || p.Covers(dev[0]) || len(p.Plans()) != 2 || p.Strategies["T"].TeleOnlyOf != "A" {
		t.Fatalf("%+v", p)
	}
	w := sim.New(miniConfig(t), rng.SeedFromInt(901))
	w.Engine().RunUntil(w.StartTime(), nil)
	func() {
		defer func() {
			if r := recover(); r == nil || !strings.Contains(fmt.Sprint(r), "authorized protocol") {
				t.Errorf("a development protocol's holdout recorder: refusal %v", r)
			}
		}()
		w.NewHoldoutRecorder(p, "A", "", "", 901)
	}()
	refuse := func(what, want string, root, set string, seeds []int64) {
		t.Helper()
		if _, _, err := eval.Development(root, set, pol, digest, seeds); err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("%s: %v", what, err)
		}
	}
	refuse("a reference seed", "not a development seed", f.root, fixSet, []int64{901, 1001})
	refuse("a seed past the range", "not a development seed", f.root, fixSet, []int64{911})
	refuse("no set", "no such file", f.root, "docs/evaluation/strategies/none.json", dev)
	var st eval.Strategy
	f.edit("docs/evaluation/strategies/T.json", &st, func() { st.Description = "changed" })
	refuse("a strategy off its pin", "pinned digest", f.root, fixSet, dev)
	g := newProtoFixture(t)
	var set eval.StrategySet
	g.edit(fixSet, &set, func() { set.Baseline = "T" })
	refuse("a baseline that acts", "takes actions", g.root, fixSet, dev)
	h := newProtoFixture(t)
	h.edit(fixSet, &set, func() { set.Schema = "dreamteam.strategyset/v0" })
	refuse("another schema", "schema", h.root, fixSet, dev)
	k := newProtoFixture(t)
	var ts eval.Strategy
	k.edit("docs/evaluation/strategies/T.json", &ts, func() { ts.TeleOnlyOf = "Z" })
	traw, _ := os.ReadFile(k.path("docs/evaluation/strategies/T.json"))
	k.edit(fixSet, &set, func() { set.Plans[1].SHA256 = sha256Hex(traw) })
	refuse("teleOnlyOf naming no strategy", "teleOnlyOf", k.root, fixSet, dev)
}

// Every committed development set loads strictly, and every batch of
// every strategy is in [takeover, T90) on the benchmark's practice.
func TestDevelopmentSetsCommitted(t *testing.T) {
	pol, digest, err := eval.LoadPolicy("../../" + eval.PolicyPath)
	if err != nil {
		t.Fatal(err)
	}
	dev, _ := eval.DevelopmentSeeds(pol)
	sets, _ := filepath.Glob("../../docs/calibration/strategy-development/sets/*.json")
	if len(sets) == 0 {
		t.Fatal("no development sets")
	}
	for _, path := range sets {
		rel := "docs/calibration/strategy-development/sets/" + filepath.Base(path)
		p, _, err := eval.Development("../..", rel, pol, digest, dev)
		if err != nil {
			t.Fatal(err)
		}
		cfg, _, err := strategyConfig(p)
		if err != nil {
			t.Fatal(err)
		}
		w := sim.New(cfg, rng.SeedFromInt(dev[0]))
		for _, name := range p.Plans() {
			if _, err := strategyBatches(p, w, p.Strategies[name]); err != nil {
				t.Fatalf("%s: %v", rel, err)
			}
		}
	}
}

func TestWriteOnce(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "run")
	files := map[string][]byte{"run.json": []byte("a\n"), "report.json": []byte("b\n")}
	if mode, err := writeOnce(dir, files); err != nil || mode != eval.RunWritten {
		t.Fatalf("%s %v", mode, err)
	}
	if mode, err := writeOnce(dir, files); err != nil || mode != eval.RunReproduced {
		t.Fatalf("%s %v", mode, err)
	}
	if _, err := writeOnce(dir, map[string][]byte{"run.json": []byte("a\n"), "report.json": []byte("c\n")}); err == nil {
		t.Fatal("a different report reproduced")
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "report.json")); string(got) != "b\n" {
		t.Fatal("a failed reproduction wrote")
	}
	if _, err := writeOnce(dir, map[string][]byte{"run.json": []byte("a\n"), "report.json": []byte("b\n"), "more.csv": nil}); err == nil {
		t.Fatal("a partial directory reproduced")
	}
}
