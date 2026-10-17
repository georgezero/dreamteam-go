package sim_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/eval"
	"github.com/georgezero/dreamteam-go/internal/rng"
	"github.com/georgezero/dreamteam-go/internal/sim"
	"github.com/georgezero/dreamteam-go/internal/simtime"
	"github.com/georgezero/dreamteam-go/internal/version"
)

// The holdout protocol end to end on a synthetic protocol: a test
// holdout of two seeds (its own label and range, not the repository's
// holdout), the mini practice, the short policy, a no-action baseline and
// a teleradiology strategy. Nothing here touches the sealed holdout.

type stubRepo struct {
	err    error
	author string
}

func (s *stubRepo) Committed(...string) error              { return s.err }
func (s *stubRepo) FirstAuthor(string) (string, error)     { return s.author, nil }
func sha256Hex(raw []byte) string                          { h := sha256.Sum256(raw); return hex.EncodeToString(h[:]) }
func (f *protoFixture) path(rel string) string             { return filepath.Join(f.root, filepath.FromSlash(rel)) }
func (f *protoFixture) authorize() (*eval.Protocol, error) { return f.authorizeAs(version.Number) }

type protoFixture struct {
	t    *testing.T
	root string
	repo *stubRepo
}

func (f *protoFixture) authorizeAs(engine string) (*eval.Protocol, error) {
	return eval.Authorize(f.root, eval.RepoPaths, f.repo, engine)
}

// write marshals v to rel and returns the bytes.
func (f *protoFixture) write(rel string, v any) []byte {
	f.t.Helper()
	raw, err := json.MarshalIndent(v, "", " ")
	if err != nil {
		f.t.Fatal(err)
	}
	raw = append(raw, '\n')
	if err := os.MkdirAll(filepath.Dir(f.path(rel)), 0o755); err != nil {
		f.t.Fatal(err)
	}
	if err := os.WriteFile(f.path(rel), raw, 0o644); err != nil {
		f.t.Fatal(err)
	}
	return raw
}

// edit decodes rel into v, applies change and rewrites it. Whatever pins
// rel by digest is left as it was: the change is out of line with it.
func (f *protoFixture) edit(rel string, v any, change func()) {
	f.t.Helper()
	raw, _ := os.ReadFile(f.path(rel))
	if err := json.Unmarshal(raw, v); err != nil {
		f.t.Fatal(err)
	}
	change()
	f.write(rel, v)
}

const (
	fixPolicy   = "docs/evaluation/policy-test.json"
	fixScenario = "docs/evaluation/scenario-mini.json"
	fixSet      = "docs/evaluation/strategies/set.json"
)

func newProtoFixture(t *testing.T) *protoFixture {
	t.Helper()
	f := &protoFixture{t: t, root: t.TempDir(), repo: &stubRepo{author: "Owner"}}
	p, _ := shortPolicy(t, miniConfig(t))
	praw := f.write(fixPolicy, p)
	seeds, err := eval.HoldoutSeeds("dreamteam/test/holdout-runner", 2, 500000, 599999, nil)
	if err != nil {
		t.Fatal(err)
	}
	parts := make([]string, len(seeds))
	for i, s := range seeds {
		parts[i] = strconv.FormatInt(s, 10)
	}
	hraw := f.write(eval.RepoPaths.Holdout, eval.Holdout{Schema: eval.HoldoutSchema, ID: "test-holdout", Method: "as v0.2-holdout-1",
		Label: "dreamteam/test/holdout-runner", Count: 2, Lo: 500000, Hi: 599999, Excluded: []string{},
		SeedsSHA256: sha256Hex([]byte(strings.Join(parts, ","))), Sealed: true, Protocol: []string{"test"}, Unseal: eval.RepoPaths.Unseal})
	f.write(fixScenario, map[string]string{"scenario": "mini"})
	w := sim.New(miniConfig(t), rng.SeedFromInt(1))
	start := w.StartTime()
	cal := w.Calendar()
	strategies := map[string]eval.Strategy{
		"A": {Schema: eval.StrategySchema, Name: "A", Description: "no action"},
		"T": {Schema: eval.StrategySchema, Name: "T", Description: "weeknight and weekend teleradiology from day 14", TeleOnlyOf: "A",
			Batches: []eval.StrategyBatch{{Time: cal.ISO(start), Actions: []api.Action{{Type: api.ActContractTeleradiology, Op: api.TeleOpStart,
				RequestID: "eval", Plan: "standard-v1", Windows: []string{"weeknights", "saturday", "sunday"}, Scope: "all",
				Effective: cal.Date(start.Add(14 * simtime.Day)).String()}}}}},
	}
	set := eval.StrategySet{Schema: eval.StrategySetSchema, ID: "test-set", Description: "synthetic", ScenarioManifest: fixScenario, Baseline: "A"}
	for _, n := range []string{"A", "T"} {
		rel := "docs/evaluation/strategies/" + n + ".json"
		raw := f.write(rel, strategies[n])
		set.Plans = append(set.Plans, struct {
			Name   string `json:"name"`
			Path   string `json:"path"`
			SHA256 string `json:"sha256"`
		}{n, rel, sha256Hex(raw)})
	}
	sraw := f.write(fixSet, set)
	f.write(eval.RepoPaths.Unseal, eval.Unseal{Schema: eval.UnsealSchema, Holdout: "test-holdout", HoldoutSHA256: sha256Hex(hraw),
		Policy: fixPolicy, PolicyDigest: sha256Hex(praw), StrategySet: fixSet, StrategiesSHA256: sha256Hex(sraw), Engine: version.Number,
		Primary: "T", ApprovedBy: "Owner", ApprovedOn: "2026-10-17", Statement: "Run the frozen strategy set once on the test holdout."})
	return f
}

// repinUnseal changes a field of the unseal record (which nothing pins).
func (f *protoFixture) repinUnseal(change func(u *eval.Unseal)) {
	var u eval.Unseal
	f.edit(eval.RepoPaths.Unseal, &u, func() { change(&u) })
}

func TestHoldoutProtocolAuthorize(t *testing.T) {
	f := newProtoFixture(t)
	p, err := f.authorize()
	if err != nil {
		t.Fatal(err)
	}
	if !p.Authorized() || len(p.Seeds) != 2 || p.Committer != "Owner" || !p.Covers(p.Seeds[1]) || p.Covers(901) {
		t.Fatalf("%+v", p)
	}
	if (&eval.Protocol{Seeds: p.Seeds}).Covers(p.Seeds[0]) {
		t.Fatal("a protocol not made by Authorize covers a seed")
	}
	if _, err := f.authorizeAs("0.0.0-other"); err == nil || !strings.Contains(err.Error(), "engine") {
		t.Fatalf("another engine authorized: %v", err)
	}
	// Each case breaks one element of the contract; every one refuses.
	cases := []struct {
		want   string
		break_ func(f *protoFixture)
	}{
		{"no unseal record", func(f *protoFixture) { os.Remove(f.path(eval.RepoPaths.Unseal)) }},
		{"not at its pinned digest", func(f *protoFixture) {
			var st eval.Strategy
			f.edit("docs/evaluation/strategies/T.json", &st, func() { st.Description += " (edited after approval)" })
		}},
		{"not at the approved digest", func(f *protoFixture) {
			var s eval.StrategySet
			f.edit(fixSet, &s, func() { s.Description += "!" })
		}},
		{"not the approved", func(f *protoFixture) {
			raw, _ := os.ReadFile(f.path(fixPolicy))
			os.WriteFile(f.path(fixPolicy), append(raw, ' '), 0o644)
		}},
		{"manifest's digest", func(f *protoFixture) {
			raw, _ := os.ReadFile(f.path(eval.RepoPaths.Holdout))
			os.WriteFile(f.path(eval.RepoPaths.Holdout), append(raw, ' '), 0o644)
		}},
		{"is not a candidate", func(f *protoFixture) { f.repinUnseal(func(u *eval.Unseal) { u.Primary = "A" }) }},
		{"is not a candidate", func(f *protoFixture) { f.repinUnseal(func(u *eval.Unseal) { u.Primary = "Z" }) }},
		{"is not a date", func(f *protoFixture) { f.repinUnseal(func(u *eval.Unseal) { u.ApprovedOn = "soon" }) }},
		{"no approver", func(f *protoFixture) { f.repinUnseal(func(u *eval.Unseal) { u.Statement = " " }) }},
		{"schema", func(f *protoFixture) {
			f.repinUnseal(func(u *eval.Unseal) { u.Schema = "dreamteam.holdout-unseal/v0" })
		}},
		{"committed by", func(f *protoFixture) { f.repo.author = "Someone else" }},
		{"not committed", func(f *protoFixture) { f.repo.err = errors.New("docs/evaluation/strategies/T.json changed") }},
		{"unknown field", func(f *protoFixture) {
			raw, _ := os.ReadFile(f.path(eval.RepoPaths.Unseal))
			os.WriteFile(f.path(eval.RepoPaths.Unseal), []byte(strings.Replace(string(raw), `"primary"`, `"extra": 1, "primary"`, 1)), 0o644)
		}},
	}
	for _, c := range cases {
		f := newProtoFixture(t)
		c.break_(f)
		p, err := f.authorize()
		if err == nil || p != nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v", c.want, err)
		}
		if c.want == "no unseal record" && !errors.Is(err, eval.ErrSealed) {
			t.Errorf("a missing unseal record is not ErrSealed: %v", err)
		}
	}
}

// The one-time run: written once, reproduced exactly, never overwritten,
// and a tampered output is a failure that writes nothing.
func TestHoldoutRunnerOneTime(t *testing.T) {
	f := newProtoFixture(t)
	p, err := f.authorize()
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(f.root, "runs")
	mode, rep, err := runHoldout(t, p, out)
	if err != nil || mode != eval.RunWritten {
		t.Fatalf("%s %v", mode, err)
	}
	if len(rep.Integrity) != 0 || rep.Primary != "T" || len(rep.Comparisons) != 1 {
		t.Fatalf("%v %+v", rep.Integrity, rep.Comparisons)
	}
	if c := rep.Comparisons[0]; !c.Valid || c.Role != "primary" || c.Verdict == "" {
		t.Fatalf("%+v", c)
	}
	raw, _ := os.ReadFile(filepath.Join(out, "run.json"))
	var run eval.Run
	if err := json.Unmarshal(raw, &run); err != nil {
		t.Fatal(err)
	}
	// The whole matrix, on the horizon the contract's commitment sets.
	if len(run.Records) != 4 || run.Primary != "T" || run.Engine != version.Number || run.HorizonEndSec != rep.HorizonEndSec ||
		run.HorizonEndSec <= eval.TailEnd(p.Policy, rep.Plans[0].PerSeed[0].Measures.Windows[eval.WinT90].From, 0, 0) {
		t.Fatalf("%+v", run)
	}
	if st, _ := os.Stat(filepath.Join(out, "report.json")); st.Mode().Perm()&0o222 != 0 {
		t.Fatal("the committed report is writable")
	}
	// A rerun reproduces it.
	if mode, _, err := runHoldout(t, p, out); err != nil || mode != eval.RunReproduced {
		t.Fatalf("%s %v", mode, err)
	}
	// A tampered report: the rerun refuses, and writes nothing.
	repPath := filepath.Join(out, "report.json")
	os.Chmod(repPath, 0o644)
	tampered := append(raw[:0:0], []byte("{}\n")...)
	os.WriteFile(repPath, tampered, 0o444)
	if _, _, err := runHoldout(t, p, out); !errors.Is(err, eval.ErrNotReproduced) {
		t.Fatalf("a tampered run reproduced: %v", err)
	}
	if got, _ := os.ReadFile(repPath); string(got) != string(tampered) {
		t.Fatal("the failed rerun wrote")
	}
	// A strategy acting at or after the primary endpoint is refused.
	w := sim.New(miniConfig(t), rng.SeedFromInt(p.Seeds[0]))
	late := eval.Strategy{Name: "late", Batches: []eval.StrategyBatch{{Time: w.Calendar().ISO(w.StartTime().Add(90 * simtime.Day))}}}
	if _, err := strategyBatches(p, w, late); err == nil {
		t.Fatal("a batch at T90 accepted")
	}
	// The matrix must be complete and every record the protocol's.
	recs := map[string][]eval.Record{}
	for _, n := range p.Plans() {
		for _, s := range p.Seeds {
			recs[n] = append(recs[n], eval.Record{Plan: n, Seed: s, PlanSHA256: p.StrategySHA[n]})
		}
	}
	if _, err := p.NewRun(recs, 0, nil); err != nil {
		t.Fatal(err)
	}
	short := map[string][]eval.Record{"A": recs["A"], "T": recs["T"][:1]}
	if _, err := p.NewRun(short, 0, nil); err == nil {
		t.Fatal("a missing seed accepted")
	}
	recs["T"][1].PlanSHA256 = "other"
	if _, err := p.NewRun(recs, 0, nil); err == nil {
		t.Fatal("a record of another strategy file accepted")
	}
	if _, err := (&eval.Protocol{}).NewRun(recs, 0, nil); !errors.Is(err, eval.ErrSealed) {
		t.Fatal("an unauthorized protocol made a run")
	}
	// A directory with other files but no manifest is not written over.
	stray := filepath.Join(f.root, "stray")
	os.MkdirAll(stray, 0o755)
	os.WriteFile(filepath.Join(stray, "notes.txt"), []byte("x"), 0o644)
	if _, err := eval.CommitRun(stray, &run, []byte("{}")); err == nil {
		t.Fatal("wrote into a directory with other files")
	}
}

// Outside the runner, no recorder takes a holdout seed; the runner's own
// recorder needs an authorized protocol.
func TestHoldoutRecordersGuarded(t *testing.T) {
	_, seeds, err := eval.LoadHoldout("../../" + eval.HoldoutPath)
	if err != nil {
		t.Fatal(err)
	}
	panics := func(name, want string, f func()) {
		t.Helper()
		defer func() {
			if r := recover(); r == nil || !strings.Contains(fmt.Sprint(r), want) {
				t.Errorf("%s: refusal %v, want %q", name, r, want)
			}
		}()
		f()
	}
	cfg := *miniConfig(t)
	cfg.EvaluationDays = 0
	w := sim.New(&cfg, rng.SeedFromInt(seeds[0]))
	w.Engine().RunUntil(w.StartTime(), nil)
	panics("a holdout seed through NewEvalRecorder", "canonical holdout runner", func() { w.NewEvalRecorder("A", "", "", seeds[0], 14) })
	// A protocol not made by Authorize, otherwise complete, is refused.
	pol, _ := shortPolicy(t, &cfg)
	forged := &eval.Protocol{Seeds: seeds, Policy: pol, StrategySHA: map[string]string{}}
	panics("an unauthorized protocol", "authorized protocol", func() { w.NewHoldoutRecorder(forged, "A", "", "", seeds[0]) })
}
