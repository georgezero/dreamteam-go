package session

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Two evaluations shared by the T90 tests (each runs two ~200-day worlds):
// no action measured to day 90 on seed 901, and a reading-allocation plan
// projected from day 30 on seed 902.
var t90Fix struct {
	once        sync.Once
	full, proj  *T90
	err         error
	before, aft Digests
}

func t90Fixture(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("runs four 200-day worlds")
	}
	t90Fix.once.Do(func() {
		s, err := Open("sustainable", 901)
		if err != nil {
			t90Fix.err = err
			return
		}
		s.AdvanceTo(s.Now().Add(90 * simtime.Day))
		t90Fix.before = s.Digests()
		if t90Fix.full, t90Fix.err = s.T90(); t90Fix.err != nil {
			return
		}
		t90Fix.aft = s.Digests()
		p, err := Open("sustainable", 902)
		if err != nil {
			t90Fix.err = err
			return
		}
		for _, r := range p.Apply(
			api.Action{Type: api.ActSetReadingAllocation, Op: "set", Employee: "E07", Effective: "2026-10-27", SiteAllocationBP: 6250},
			api.Action{Type: api.ActSetReadingAllocation, Op: "set", Employee: "E08", Effective: "2026-10-27", SiteAllocationBP: 5250}) {
			if !r.OK {
				t90Fix.err = fmt.Errorf("%s: %s", r.Code, r.Message)
				return
			}
		}
		p.AdvanceTo(p.Now().Add(30 * simtime.Day))
		t90Fix.proj, t90Fix.err = p.T90()
	})
	if t90Fix.err != nil {
		t.Fatal(t90Fix.err)
	}
}

// The evaluation is the policy's, valid, and shows every gate of the plan's
// seed with its status and value: pass, fail and insufficient all occur on
// the no-action seed, and each is displayed as the report decides it.
func TestT90Gates(t *testing.T) {
	t90Fixture(t)
	r := t90Fix.full
	if r.PolicyID != "v0.2-eval-4" || r.PolicyDigest != "4e277c7263dcff7ddabe63741beb44abfc9a46d3bc23f6b6418715b58ffb6fb3" || len(r.Report.Integrity) > 0 {
		t.Fatalf("policy %s %s, integrity %v", r.PolicyID, r.PolicyDigest, r.Report.Integrity)
	}
	plan := t90PlanReport(r.Report, T90Plan)
	if plan == nil || t90PlanReport(r.Report, T90Baseline) == nil {
		t.Fatal("plan or baseline missing")
	}
	text := strings.Join(r.Lines(160), "\n")
	seen := map[string]bool{}
	for _, g := range plan.PerSeed[0].Gates {
		seen[g.Status] = true
		want := fmt.Sprintf("%-8s %-7s %-12s", g.Stratum, g.Window, strings.ToUpper(g.Status))
		if g.Den > 0 {
			want += fmt.Sprintf(" %-19s", fmt.Sprintf("%d/%d = %.1f%%", g.Num, g.Den, 100*g.Value))
		}
		if !strings.Contains(text, want) {
			t.Errorf("gate %s %s %s not shown as %q", g.Stratum, g.Window, g.Status, want)
		}
	}
	for _, st := range []string{"pass", "fail", "insufficient"} {
		if !seen[st] {
			t.Errorf("no %s gate on the no-action seed: %v", st, seen)
		}
	}
	// The same plan as the baseline: no difference anywhere, and the plan
	// gates as the report sets them.
	cmp := r.Report.Comparisons[0]
	if !strings.Contains(text, "VERDICT on this seed, by the policy's rules: "+cmp.Verdict+" · "+cmp.Classification) ||
		!strings.Contains(text, fmt.Sprintf("  %-17s %-13s", "Service", strings.ToUpper(plan.ServiceStatus))) {
		t.Errorf("verdict or plan gates not shown:\n%s", text)
	}
	if !strings.Contains(text, "MEASURED to T90") || strings.Contains(text, "PROJECTED") || r.Projected {
		t.Error("a session at day 90 is measured, not projected")
	}
	// Compact layout: every line within 80 columns.
	for _, l := range r.Lines(79) {
		if n := len([]rune(l)); n > 79 {
			t.Errorf("%d columns: %q", n, l)
		}
	}
}

// A session not yet at T90 is projected, and says so; evaluating leaves
// the session exactly as it was.
func TestT90ProjectedAndSessionUnchanged(t *testing.T) {
	t90Fixture(t)
	p := t90Fix.proj
	if !p.Projected || p.Decisions != 1 || p.Later != 0 || int(p.AsOfDay) != 30 {
		t.Fatalf("projected %v, decisions %d, later %d, day %.1f", p.Projected, p.Decisions, p.Later, p.AsOfDay)
	}
	if text := strings.Join(p.Lines(160), "\n"); !strings.Contains(text, "PROJECTED: you are on day 30.0") {
		t.Error("the projection is not stated")
	}
	if t90Fix.before != t90Fix.aft {
		t.Fatal("evaluating changed the session")
	}
}

// The evaluation is defined for the policy's scenario only, after play has
// started; decisions at or after T90 belong to a later period.
func TestT90Availability(t *testing.T) {
	e, err := Open("established", 1)
	if err != nil {
		t.Fatal(err)
	}
	e.AdvanceTo(e.Now().Add(simtime.Day))
	if _, err := e.T90Job(); !errors.Is(err, ErrT90Scenario) {
		t.Errorf("established: %v", err)
	}
	s, err := Open("sustainable", 903)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.T90Job(); !errors.Is(err, ErrT90Started) {
		t.Errorf("at the takeover: %v", err)
	}
	s.AdvanceTo(s.Now().Add(91 * simtime.Day))
	s.Apply(api.Action{Type: api.ActSetReadingAllocation, Op: "set", Employee: "E07", Effective: "2027-02-01", SiteAllocationBP: 6250})
	j, err := s.T90Job()
	if err != nil {
		t.Fatal(err)
	}
	if len(j.plan) != 0 || j.later != 1 {
		t.Fatalf("plan %d batches, later %d", len(j.plan), j.later)
	}
}

// The bundled sustainable scenario is reproducible: the same 90 days twice
// give the same digests, and its saved record replays them.
func TestSustainableDeterministic(t *testing.T) {
	if testing.Short() {
		t.Skip("two 90-day runs")
	}
	run := func() *Session {
		s, err := Open("sustainable", 904)
		if err != nil {
			t.Fatal(err)
		}
		s.Apply(api.Action{Type: api.ActSetReadingAllocation, Op: "set", Employee: "E08", Effective: "2026-10-27", SiteAllocationBP: 5250})
		s.AdvanceTo(s.Now().Add(90 * simtime.Day))
		return s
	}
	a, b := run(), run()
	if a.Digests() != b.Digests() {
		t.Fatal("two runs differ")
	}
	r, err := a.Record()
	if err != nil {
		t.Fatal(err)
	}
	if r.Scenario.Fingerprint != "e8718dbd1031feb326b4934d291f81eb14fd867d4a8fa6acc1673a75b55b76f8" {
		t.Fatalf("recorded scenario %s", r.Scenario.Fingerprint)
	}
	s, rep, err := Replay(r, ReplayOptions{})
	if err != nil || !rep.OK || s.Digests() != a.Digests() {
		t.Fatalf("replay: %v %s", err, repString(rep))
	}
}
