package eval

import (
	"math"
	"strings"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Adversarial fixtures for every loophole of the Slice 5C.0 audit
// (docs/audits/slice-5c0-eval-policy-audit.md), each a synthetic plan
// that passed or was misjudged under v0.2-eval-1 and is caught by
// v0.2-eval-2. No simulation and no candidate strategy is involved.

func reasons(c Comparison) string { return strings.Join(c.Reasons, "\n") }

// H5: six seeds, each failing a different per-seed gate. No gate has more
// failing seeds than the 4-of-40 limit (v0.2-eval-1 passed it), and every
// pooled rate passes; the union of failing seeds (6) exceeds the limit.
func TestJointSeedReliability(t *testing.T) {
	p, d := testPolicy(t)
	bursts := map[int64]burst{
		901: {"soon", 0, 90, 450},   // soon fails in T90 only
		902: {"soon", 90, 150, 450}, // soon fails in the tail only
		903: {"routine", 0, 90, 200},
		904: {"routine", 90, 150, 200},
		905: {"urgent", 0, 150, 300}, // urgent fails over the gated period
		906: {"urgent", 0, 150, 300},
	}
	mk := func(name string, bad bool) func(int64) synth {
		return func(s int64) synth {
			x := base(name, s)
			x.profitPD = 30_000
			if b, ok := bursts[s]; ok && bad {
				x.bursts = []burst{b}
			}
			return x
		}
	}
	rep := Evaluate(p, d, Input{Baseline: "A", Plans: map[string][]Record{"A": records(40, mk("A", false)), "B": records(40, mk("B", true))}})
	pr := plan(rep, "B")
	lim := failLimit(bp(p.Service.MaxFailingSeedFraction), 40)
	perGate := map[string]int{}
	for _, s := range pr.PerSeed {
		for _, g := range s.Gates {
			if g.PerSeed && role(p, g.Window) == "gated" && g.Status != Pass {
				perGate[g.Window+"/"+g.Stratum]++
			}
		}
	}
	if len(perGate) != 5 {
		t.Fatalf("failing gates %v: the fixture should fail five different gates", perGate)
	}
	for k, n := range perGate {
		if n > lim {
			t.Fatalf("gate %s fails on %d seeds: each gate alone must be within the limit %d", k, n, lim)
		}
	}
	for _, g := range pr.ServiceGates {
		if g.Role == "gated" && g.Status == Fail {
			t.Fatalf("pooled %s %s fails: only the union should", g.Window, g.Stratum)
		}
	}
	j := pr.ServiceJoint
	if j.Status != Fail || len(j.FailingSeeds) != 6 || j.FailLimit != 4 || j.Gates != 5 || pr.ServiceStatus != Fail {
		t.Fatalf("%+v %s", j, pr.ServiceStatus)
	}
	if c := comparison(t, rep, "B"); c.Verdict != NOGO || !strings.Contains(reasons(c), "service seed reliability") {
		t.Fatalf("%s %v", c.Verdict, c.Reasons)
	}
	// Four seeds failing (the limit) pass.
	delete(bursts, 905)
	delete(bursts, 906)
	rep = Evaluate(p, d, Input{Baseline: "A", Plans: map[string][]Record{"A": records(40, mk("A", false)), "B": records(40, mk("B", true))}})
	if j := plan(rep, "B").ServiceJoint; j.Status != Pass || len(j.FailingSeeds) != 4 {
		t.Fatalf("%+v", j)
	}
}

// H2: a late-effective candidate has a longer tail. Its baseline must be
// observed over the same calendar interval: unequal horizons are invalid,
// a horizon short of a record's tail rule is invalid, and with the common
// horizon both plans' tail windows are identical.
func TestCommonHorizon(t *testing.T) {
	p, d := testPolicy(t)
	late := func(h int64) func(int64) synth {
		return func(s int64) synth {
			x := base("E", s)
			x.lastAction, x.effect, x.profitPD, x.horizon = 20*daySec, 140*daySec, 30_000, h // own tail: day 170
			return x
		}
	}
	baseAt := func(h int64) func(int64) synth {
		return func(s int64) synth { x := base("A", s); x.horizon = h; return x }
	}
	// Unequal tails under the same metric name: invalid.
	rep := Evaluate(p, d, Input{Baseline: "A", Plans: map[string][]Record{"A": records(10, baseAt(150*daySec)), "E": records(10, late(170*daySec))}})
	if c := comparison(t, rep, "E"); c.Verdict != INVALID || len(rep.Integrity) == 0 || !strings.Contains(reasons(c), "share one horizon") {
		t.Fatalf("%s %v", c.Verdict, c.Reasons)
	}
	// A common horizon that does not cover the candidate's tail rule.
	rep = Evaluate(p, d, Input{Baseline: "A", Plans: map[string][]Record{"A": records(10, baseAt(150*daySec)), "E": records(10, late(150*daySec))}})
	if c := comparison(t, rep, "E"); c.Verdict != INVALID || !strings.Contains(reasons(c), "does not cover") {
		t.Fatalf("%s %v", c.Verdict, c.Reasons)
	}
	// The common horizon: the same windows, the baseline's tail 80 days.
	rep = Evaluate(p, d, Input{Baseline: "A", Plans: map[string][]Record{"A": records(10, baseAt(170*daySec)), "E": records(10, late(170*daySec))}})
	c := comparison(t, rep, "E")
	if c.Verdict != GO || rep.HorizonEndSec != t0+170*daySec {
		t.Fatalf("%s %v", c.Verdict, c.Reasons)
	}
	a, e := plan(rep, "A").PerSeed[0].Measures.Windows[WinTail], plan(rep, "E").PerSeed[0].Measures.Windows[WinTail]
	if a.From != e.From || a.To != e.To || a.To-a.From != 80*daySec || a.Cohorts["urgent"].Acquired != 80 {
		t.Fatalf("baseline tail (%d, %d], candidate (%d, %d]", a.From, a.To, e.From, e.To)
	}
}

// H3: orders received on day 89 and wanted on day 100. Under v0.2-eval-1
// they were in no gated access denominator (received before the tail, due
// after T90); now they are due in the tail wherever received, and a plan
// that leaves them open fails case selection.
func TestCarriedForwardOrders(t *testing.T) {
	p, d := testPolicy(t)
	mk := func(name string, served bool) func(int64) synth {
		return func(s int64) synth {
			x := base(name, s)
			out := OutOpen
			if served {
				out = OutServed
			}
			x.late, x.profitPD = []lateOrders{{n: 40, recv: 89, want: 100, resolveDay: 100, out: out}}, 30_000
			return x
		}
	}
	rep := Evaluate(p, d, Input{Baseline: "A", Plans: map[string][]Record{"A": records(10, mk("A", true)), "B": records(10, mk("B", false))}})
	m := plan(rep, "B").PerSeed[0].Measures
	t90, tail := m.Windows[WinT90].Access, m.Windows[WinTail].Access
	if t90.DueBy["priority:soon"] != 180 || tail.DueBy["priority:soon"] != 120+40 || tail.DueReceivedEarlier != 40 || tail.UnresolvedDue != 40 {
		t.Fatalf("T90 due %d; tail due %d, received earlier %d, unresolved %d", t90.DueBy["priority:soon"], tail.DueBy["priority:soon"], tail.DueReceivedEarlier, tail.UnresolvedDue)
	}
	c := comparison(t, rep, "B")
	if c.CaseSelectionStatus != Fail || c.Verdict != NOGO || !strings.Contains(reasons(c), "served_fraction_drop_per_seed tail priority:soon") ||
		!strings.Contains(reasons(c), "served_fraction_drop_per_seed tail modality:CT") || !strings.Contains(reasons(c), "served_fraction_drop_pooled tail priorityModality:soon/CT") {
		t.Fatalf("%s %s %v", c.CaseSelectionStatus, c.Verdict, c.Reasons)
	}
	// Served, they are no loss.
	rep = Evaluate(p, d, Input{Baseline: "A", Plans: map[string][]Record{"A": records(10, mk("A", true)), "B": records(10, mk("B", true))}})
	if c := comparison(t, rep, "B"); c.CaseSelectionStatus != Pass {
		t.Fatalf("%s %v", c.CaseSelectionStatus, c.Reasons)
	}
	// Referrals received late in T90 and lost after it (day 92) count in
	// T90's receipt cohort: taken at the window's end they were open, and
	// in no later receipt cohort.
	lost := func(name, out string) func(int64) synth {
		return func(s int64) synth {
			x := base(name, s)
			x.late, x.profitPD = []lateOrders{{n: 40, recv: 85, want: 86, resolveDay: 92, out: out}}, 30_000
			return x
		}
	}
	rep = Evaluate(p, d, Input{Baseline: "A", Plans: map[string][]Record{"A": records(10, lost("A", OutServed)), "B": records(10, lost("B", OutLost))}})
	c = comparison(t, rep, "B")
	if x := plan(rep, "B").PerSeed[0].Measures.Windows[WinT90].Access; x.Lost != 40 || x.LostDue != 40 {
		t.Fatalf("T90 lost %d, lost due %d", x.Lost, x.LostDue)
	}
	if c.CaseSelectionStatus != Fail || !strings.Contains(reasons(c), "lost_referral_rate_increase T90") {
		t.Fatalf("%s %v", c.CaseSelectionStatus, c.Reasons)
	}
}

// Inherited orders are followed to the end of observation as their own
// stratum: a plan that serves fewer of them than the baseline fails.
func TestInheritedOrdersGuarded(t *testing.T) {
	p, d := testPolicy(t)
	mk := func(name string) func(int64) synth {
		return func(s int64) synth { x := base(name, s); x.inherited, x.profitPD = 40, 30_000; return x }
	}
	recs := records(10, mk("B"))
	for i := range recs {
		// Ten inherited orders a seed lost instead of served.
		for k := range 10 {
			for c := range recs[i].Captures {
				recs[i].Captures[c].OrderOutcomes[k] = OutLost
			}
		}
	}
	rep := Evaluate(p, d, Input{Baseline: "A", Plans: map[string][]Record{"A": records(10, mk("A")), "B": recs}})
	m := plan(rep, "B").PerSeed[0].Measures.Inherited
	if m.Due != 40 || m.Served != 30 || m.Lost != 10 {
		t.Fatalf("%+v", m)
	}
	if c := comparison(t, rep, "B"); c.CaseSelectionStatus != Fail || !strings.Contains(reasons(c), InheritedStratum) {
		t.Fatalf("%s %v", c.CaseSelectionStatus, c.Reasons)
	}
}

// backlogRecord is a record of constant daily samples (backlog level) and
// the given studies, observed to day 181 with a day-150 horizon.
func backlogRecord(level func(d int) int, studies []Study) *Record {
	r := &Record{TakeoverSec: t0, TakeoverDate: t0Date, HorizonEndSec: t0 + 150*daySec, TATTargetMin: testTargets, Studies: studies}
	date, _ := simtime.ParseDate(t0Date)
	for d := 1; d <= 181; d++ {
		r.Samples = append(r.Samples, Sample{Day: d, AtSec: t0 + int64(d)*daySec, Date: date.AddDays(int64(d)).String(), Unread: level(d)})
	}
	return r
}

func checkFailed(bm BacklogMeasures, name string) bool {
	for _, c := range bm.Checks {
		if c.Check == name && !c.Pass {
			return true
		}
	}
	return false
}

// H4: each way old or growing work escaped the single final-versus-
// reference comparison of v0.2-eval-1.
func TestBacklogSafeguards(t *testing.T) {
	p, _ := testPolicy(t)
	t90 := t0 + 90*daySec
	flat := func(int) int { return 20 }
	// The rule of v0.2-eval-1, for contrast: the last whole block ending by
	// the horizon (111–138) against the reference, mean only.
	legacyStable := func(bm BacklogMeasures) bool {
		for _, b := range bm.Later {
			if b.LastDay <= 150 && b.LastDay > 138-28 && b.LastDay == 138 {
				return b.Mean-bm.Reference.Mean <= max(0.1*bm.Reference.Mean, 2)
			}
		}
		return false
	}
	if bm := measureBacklog(p, backlogRecord(flat, nil), t90); bm.Status != "stable" || len(bm.Reasons) != 0 {
		t.Fatalf("%+v", bm)
	}

	// Persistent old work: three inherited routine studies never reported,
	// in both blocks with zero growth.
	var old []Study
	for k := range 3 {
		acq := t0 - 2*hour + int64(k)*60
		old = append(old, Study{ID: "I", Priority: "routine", AcquiredSec: acq, EnteredSec: acq})
	}
	bm := measureBacklog(p, backlogRecord(flat, old), t90)
	if bm.Status != "unstable" || bm.Persistent != 3 || !legacyStable(bm) {
		t.Fatalf("persistent: %+v", bm)
	}

	// A persistent stream of old work, in the reference block as in every
	// later one: each day one routine study is reported 20 days past due.
	// Every block's counts and exposure equal the reference's.
	var stream []Study
	for k := -40; k <= 181; k++ {
		acq := t0 + int64(k)*daySec + 3*hour
		fin := acq + 1440*60 + 20*daySec
		stream = append(stream, Study{ID: "P", Priority: "routine", AcquiredSec: acq, EnteredSec: acq, ClaimedSec: fin - hour, ReadStartSec: fin - hour, FinalSec: fin})
	}
	bm = measureBacklog(p, backlogRecord(flat, stream), t90)
	for _, c := range bm.Checks {
		if !c.Pass {
			t.Fatalf("stream: only the persistent rule should fail: %+v", c)
		}
	}
	if bm.Status != "unstable" || bm.Persistent == 0 || !legacyStable(bm) {
		t.Fatalf("stream: %+v", bm)
	}

	// A spike that recovers within a block: +12 for three days (days
	// 97–99) leaves the means within tolerance, not the peak.
	spike := func(d int) int {
		if d >= 97 && d <= 99 {
			return 32
		}
		return 20
	}
	bm = measureBacklog(p, backlogRecord(spike, nil), t90)
	if bm.Status != "unstable" || !checkFailed(bm, "max") || checkFailed(bm, "mean") || !legacyStable(bm) {
		t.Fatalf("spike: %+v", bm)
	}

	// Late-tail deterioration after the last block ending by the horizon:
	// days 139–150 (and on) are in the covering block 139–166.
	late := func(d int) int {
		if d >= 139 {
			return 35
		}
		return 20
	}
	bm = measureBacklog(p, backlogRecord(late, nil), t90)
	if bm.Status != "unstable" || !checkFailed(bm, "mean") || !legacyStable(bm) {
		t.Fatalf("late tail: %+v", bm)
	}

	// Old work replacing fresh work: the same count, but from day 115 ten
	// routine studies stay unfinished for a week (stale, not persistent).
	var stale []Study
	for k := range 10 {
		acq := t0 + 113*daySec + int64(k)*60
		stale = append(stale, Study{ID: "S", Priority: "routine", AcquiredSec: acq, EnteredSec: acq, ClaimedSec: acq + 7*daySec,
			ReadStartSec: acq + 7*daySec, FinalSec: acq + 8*daySec})
	}
	bm = measureBacklog(p, backlogRecord(flat, stale), t90)
	if bm.Status != "unstable" || !checkFailed(bm, "stale") || !checkFailed(bm, "exposure") || bm.Persistent != 0 || !legacyStable(bm) {
		t.Fatalf("stale: %+v", bm)
	}

	// An inflated reference does not license unbounded later growth: the
	// tolerance is 10% of the reference, and old work is persistent
	// whatever the reference.
	inflated := func(d int) int {
		if d < 83 {
			return 40
		}
		return 45
	}
	if bm := measureBacklog(p, backlogRecord(inflated, nil), t90); bm.Status != "unstable" {
		t.Fatalf("inflated: %+v", bm)
	}
}

// M1: the tolerances hold at equality, exactly; the float subtraction
// v0.2-eval-1 used fails each of these.
func TestExactTolerances(t *testing.T) {
	// Run-time float64 arithmetic, as v0.2-eval-1 did it (Go would fold
	// constant expressions exactly).
	fr := func(a, b float64) float64 { return a / b }
	cases := []struct {
		name      string
		exact     bool // within tolerance, exactly
		float     bool // v0.2-eval-1's float test: within?
		exactPast bool // one count further: past the tolerance
	}{
		{"2% acquisitions (4,900 of 5,000)", !relDropAbove(4900, 5000, 200), 1-fr(4900, 5000) <= 0.02, relDropAbove(4899, 5000, 200)},
		{"2% acquisitions (98 of 100)", !relDropAbove(98, 100, 200), 1-fr(98, 100) <= 0.02, relDropAbove(97, 100, 200)},
		{"5 points per seed (80% to 75%)", !dropAbove(80, 100, 75, 100, 500), fr(80, 100)-fr(75, 100) <= 0.05, dropAbove(80, 100, 74, 100, 500)},
		{"2 points pooled (90% to 88%)", !dropAbove(90, 100, 88, 100, 200), fr(90, 100)-fr(88, 100) <= 0.02, dropAbove(90, 100, 87, 100, 200)},
		{"2 points loss rise (5% to 7%)", !dropAbove(7, 100, 5, 100, 200), fr(7, 100)-fr(5, 100) <= 0.02, dropAbove(8, 100, 5, 100, 200)},
		{"12% breach gate (12 of 100)", !rateAbove(12, 100, 1200), fr(12, 100) <= 0.12, rateAbove(13, 100, 1200)},
		{"6% breach gate (3 of 50)", !rateAbove(3, 50, 600), fr(3, 50) <= 0.06, rateAbove(4, 50, 600)},
	}
	floatFails := 0
	for _, c := range cases {
		if !c.exact || !c.exactPast {
			t.Errorf("%s: exactly at the tolerance within %v, one past %v", c.name, c.exact, c.exactPast)
		}
		if !c.float {
			floatFails++
		}
	}
	if floatFails < 4 {
		t.Fatalf("the float comparisons failed only %d boundaries: the fixture does not show the defect", floatFails)
	}
	// Backlog growth at its tolerance: mean +10% of the reference, or the
	// 2-study floor, passes.
	if growthAbove(28*22, 28*20, 28, 1000, 2) || !growthAbove(28*22+1, 28*20, 28, 1000, 2) || growthAbove(28*33, 28*30, 28, 1000, 2) {
		t.Fatal("backlog tolerance not exact")
	}
	if failLimit(1000, 40) != 4 || failLimit(1000, 10) != 1 || failLimit(1000, 20) != 2 || failLimit(1000, 9) != 0 {
		t.Fatal("fail limit")
	}
}

// Exact equality end to end: a per-seed served-fraction drop of exactly
// five points passes the per-seed gate; one more order fails it.
func TestCaseSelectionAtEquality(t *testing.T) {
	p, _ := testPolicy(t)
	mk := func(name string, served int64) *PlanReport {
		w := &WindowMeasures{Cohorts: map[string]*Cohort{}, AcquiredBy: map[string]int64{"priority:soon": 100, "modality:CT": 100},
			Access: Access{DueBy: map[string]int64{"priority:soon": 100, "modality:CT": 100}, ServedBy: map[string]int64{"priority:soon": served, "modality:CT": served}}}
		m := &SeedMeasures{Windows: map[string]*WindowMeasures{WinT90: w, WinTail: w}}
		return &PlanReport{Plan: name, PerSeed: []SeedResult{{Seed: 901, Measures: m}}}
	}
	find := func(gs []GateSummary, gate string) GateSummary {
		for _, g := range gs {
			if g.Gate == gate && g.Window == WinT90 && g.Stratum == "priority:soon" {
				return g
			}
		}
		t.Fatalf("no %s", gate)
		return GateSummary{}
	}
	gs, _ := caseSelection(p, mk("B", 75), mk("A", 80))
	if g := find(gs, "served_fraction_drop_per_seed"); g.Status != Pass || g.Failing != 0 {
		t.Fatalf("%+v", g)
	}
	if g := find(gs, "served_fraction_drop_pooled"); g.Status != Fail {
		t.Fatalf("pooled 5 points > 2: %+v", g)
	}
	gs, _ = caseSelection(p, mk("B", 74), mk("A", 80))
	if g := find(gs, "served_fraction_drop_per_seed"); g.Status != Fail {
		t.Fatalf("%+v", g)
	}
	gs, _ = caseSelection(p, mk("B", 78), mk("A", 80))
	if g := find(gs, "served_fraction_drop_pooled"); g.Status != Pass {
		t.Fatalf("pooled exactly 2 points: %+v", g)
	}
}

// M2: a rare modality is never passed silently: served by both, it is
// reported insufficient (not judged); dropped by the candidate, it fails.
func TestRareStrata(t *testing.T) {
	p, d := testPolicy(t)
	mk := func(name string, lost bool) func(int64) synth {
		return func(s int64) synth { x := base(name, s); x.rare, x.rareLost, x.profitPD = 30, lost, 30_000; return x }
	}
	find := func(c Comparison, w string) GateSummary {
		for _, g := range c.CaseSelection {
			if g.Gate == "served_fraction_drop_pooled" && g.Window == w && g.Stratum == "modality:US" {
				return g
			}
		}
		t.Fatalf("no US stratum in %s", w)
		return GateSummary{}
	}
	rep := Evaluate(p, d, Input{Baseline: "A", Plans: map[string][]Record{"A": records(10, mk("A", false)), "B": records(10, mk("B", false))}})
	c := comparison(t, rep, "B")
	// Ten seeds × 3 US orders in T90: 30 due, judged; 2 a seed in the tail
	// (20 pooled): insufficient, reported, and not blocking? Modality is a
	// blocking kind, so the plan cannot be GO on that evidence.
	if g := find(c, WinT90); g.Status != Pass || g.Pooled.Den != 30 {
		t.Fatalf("%+v", g)
	}
	if g := find(c, WinTail); g.Status != Insufficient || !g.Blocking || c.Verdict != NOGO {
		t.Fatalf("%+v %s %v", g, c.Verdict, c.Reasons)
	}
	// The intersection kind is not blocking: reported as not judged.
	for _, g := range c.CaseSelection {
		if g.Stratum == "priorityModality:routine/US" && g.Window == WinTail && (g.Status != Insufficient || g.Blocking) {
			t.Fatalf("%+v", g)
		}
	}
	// Dropped entirely: fails, in both windows.
	rep = Evaluate(p, d, Input{Baseline: "A", Plans: map[string][]Record{"A": records(10, mk("A", false)), "B": records(10, mk("B", true))}})
	c = comparison(t, rep, "B")
	if g := find(c, WinTail); g.Status != Fail {
		t.Fatalf("%+v", g)
	}
	if !strings.Contains(reasons(c), "modality:US") || c.Verdict != NOGO {
		t.Fatalf("%s %v", c.Verdict, c.Reasons)
	}
}

// H7: a plan that beats a loss-making baseline but loses money itself is
// at most a relative improvement, never GO.
func TestLossMakingImprovement(t *testing.T) {
	p, d := testPolicy(t)
	mk := func(name string, pd int64) func(int64) synth {
		return func(s int64) synth { x := base(name, s); x.profitPD, x.profitNoise = pd, 300; return x }
	}
	rep := Evaluate(p, d, Input{Baseline: "A", Plans: map[string][]Record{"A": records(10, mk("A", -20_000)), "B": records(10, mk("B", -5_000))}})
	c := comparison(t, rep, "B")
	if c.Profit[WinT90].Lo <= 0 {
		t.Fatalf("the fixture's increment must be significant: %+v", c.Profit[WinT90])
	}
	if c.Verdict != CONDITIONAL || c.Classification != ClassRelative || !strings.Contains(reasons(c), "not profitable itself") {
		t.Fatalf("%s (%s) %v", c.Verdict, c.Classification, c.Reasons)
	}
	if pr := plan(rep, "B"); pr.OperatingProfit[WinT90].Mean >= 0 {
		t.Fatalf("%+v", pr.OperatingProfit)
	}
}

// H7: an insolvent baseline whose service passes is not a service
// restoration: a costlier candidate is NO GO, not CONDITIONAL (v0.2-eval-1
// gave CONDITIONAL GO to any candidate of an ineligible baseline).
func TestInsolventBaselineNotRestoration(t *testing.T) {
	p, d := testPolicy(t)
	broke := func(s int64) synth {
		x := base("A", s)
		x.minCash, x.minCashSet = -1, true
		return x
	}
	costly := func(s int64) synth { x := base("B", s); x.profitPD = 4_000; return x }
	rep := Evaluate(p, d, Input{Baseline: "A", Plans: map[string][]Record{"A": records(10, broke), "B": records(10, costly)}})
	if a := plan(rep, "A"); a.Eligible || a.ServiceStatus != Pass {
		t.Fatalf("baseline eligible %v service %s", a.Eligible, a.ServiceStatus)
	}
	if c := comparison(t, rep, "B"); c.Verdict != NOGO || c.Classification != ClassNoBenefit {
		t.Fatalf("%s (%s) %v", c.Verdict, c.Classification, c.Reasons)
	}
}

// M3: a funding gap anywhere in the observation is a hard gate, not only
// at the tail's end: obligations above cash for the first weeks (cash
// growing past them later) is NO GO.
func TestFundingGapThroughHorizon(t *testing.T) {
	p, d := testPolicy(t)
	tight := func(s int64) synth {
		x := base("B", s)
		x.profitPD, x.cash0, x.payables = 30_000, 100_000, 200_000 // cash passes the payables on day 20
		return x
	}
	rep := Evaluate(p, d, Input{Baseline: "A", Plans: map[string][]Record{"A": records(10, func(s int64) synth { return base("A", s) }), "B": records(10, tight)}})
	pr := plan(rep, "B")
	m := pr.PerSeed[0].Measures
	if tw := m.Windows[WinTail]; tw.Money.FundingGapCents != 0 {
		t.Fatalf("no gap at the tail's end expected: %d", tw.Money.FundingGapCents)
	}
	if m.FundingGap != 100_000-5_000 || m.FundingGapAt != t0+daySec || pr.FundingStatus != Fail || len(pr.FundingGapSeeds) != 10 {
		t.Fatalf("gap %d at %d, %s %v", m.FundingGap, m.FundingGapAt-t0, pr.FundingStatus, pr.FundingGapSeeds)
	}
	if c := comparison(t, rep, "B"); c.Verdict != NOGO || !strings.Contains(reasons(c), "funding") {
		t.Fatalf("%s %v", c.Verdict, c.Reasons)
	}
}

// M4: incomplete or inconsistent records are invalid, never partial.
func TestRecordIntegrity(t *testing.T) {
	p, d := testPolicy(t)
	cases := map[string]func(r *Record){
		"daily samples":           func(r *Record) { r.Samples = append(r.Samples[:40], r.Samples[41:]...) },
		"no capture T90":          func(r *Record) { r.Captures = append(r.Captures[:1], r.Captures[2:]...) },
		"capture T30 twice":       func(r *Record) { r.Captures = append(r.Captures, r.Captures[0]) },
		"capture T90 at":          func(r *Record) { r.Captures[1].AtSec++ },
		"timestamps out of order": func(r *Record) { r.Studies[5].FinalSec = r.Studies[5].AcquiredSec - 1 },
		"disagrees with its":      func(r *Record) { r.Orders[3].ResolvedSec = 0 },
		"observed to":             func(r *Record) { r.ObservedUntilSec -= daySec; r.Captures[len(r.Captures)-1].AtSec -= daySec },
		"below the recorded":      func(r *Record) { r.Captures[4].Money.MinCashCents = r.Samples[0].CashCents + 1 },
		"minimum cash at":         func(r *Record) { r.Captures[4].Money.MinCashSec = 0 },
		"does not cover":          func(r *Record) { r.HorizonEndSec -= daySec },
		"sample 3 is day":         func(r *Record) { r.Samples[3].Date = "2026-01-01" },
		"is not a horizon":        func(r *Record) { r.Captures[0].Name = "T31" },
		"order outcomes for":      func(r *Record) { r.Captures[2].OrderOutcomes = r.Captures[2].OrderOutcomes[1:] },
		"sample backlog":          func(r *Record) { r.Samples[10].Unread++ },
		"cash bridge":             func(r *Record) { r.Captures[0].Money.CashBridgeReconciles = false },
		"an action at":            func(r *Record) { r.LastActionSec = t0 + 90*daySec },
		"remote reports' fees":    func(r *Record) { r.Captures[1].Money.RemoteReadsCents++ },
	}
	for want, breakIt := range cases {
		r := base("B", 901).record()
		breakIt(&r)
		m := Measure(p, &r)
		if !strings.Contains(strings.Join(m.Integrity, "\n"), want) {
			t.Errorf("%s: %v", want, m.Integrity)
		}
	}
	// A missing sample invalidates the comparison (not only payback's
	// denominator).
	recs := records(10, func(s int64) synth { x := base("B", s); x.profitPD = 30_000; return x })
	recs[2].Samples = recs[2].Samples[:100]
	rep := Evaluate(p, d, Input{Baseline: "A", Plans: map[string][]Record{"A": records(10, func(s int64) synth { return base("A", s) }), "B": recs}})
	if c := comparison(t, rep, "B"); c.Verdict != INVALID || !strings.Contains(reasons(c), "lack daily samples") {
		t.Fatalf("%s %v", c.Verdict, c.Reasons)
	}
}

// M5: a displacement is counted in the window it happened in, not in
// every window its order is in.
func TestDisplacementByEventTime(t *testing.T) {
	p, _ := testPolicy(t)
	r := base("A", 901).record()
	k := 5 // an order received on day 0
	r.Orders[k].DisplacementSec = []int64{r.Orders[k].ReceivedSec + 120*daySec}
	m := Measure(p, &r)
	if len(m.Integrity) != 0 {
		t.Fatal(m.Integrity)
	}
	if m.Windows[WinT90].Access.Displacements != 0 || m.Windows[WinTail].Access.Displacements != 1 || m.Windows[WinT180].Access.DisplacedOrders != 1 {
		t.Fatalf("T90 %d, tail %d", m.Windows[WinT90].Access.Displacements, m.Windows[WinTail].Access.Displacements)
	}
}

// Low: worst seeds are identified, paired increases are per seed, and
// undefined values never enter means as zero.
func TestWorstSeedsIdentified(t *testing.T) {
	p, d := testPolicy(t)
	cand := func(s int64) synth {
		x := base("B", s)
		x.profitPD, x.profitNoise = 20_000, 1_000
		if s == 907 {
			x.bursts = []burst{{"soon", 0, 90, 300}}
		}
		return x
	}
	rep := Evaluate(p, d, Input{Baseline: "A", Plans: map[string][]Record{"A": records(10, func(s int64) synth { return base("A", s) }), "B": records(10, cand)}})
	c := comparison(t, rep, "B")
	if e := c.WorstBreach[WinT90]["soon"]; e.LargestIncrease.Seed != 907 || !e.LargestIncrease.Defined || e.Candidate.Seed != 907 {
		t.Fatalf("%+v", e)
	}
	// profitNoise: (seed % 7 − 3) × noise a day is lowest for seed % 7 = 0.
	if w := c.WorstProfit[WinT90]; w.Seed%7 != 0 || !w.Defined {
		t.Fatalf("%+v", w)
	}
	// An empty urgent stratum: no defined seed, mean not a zero rate.
	empty := func(s int64) synth { x := base("C", s); x.perDay["urgent"] = 0; return x }
	rep = Evaluate(p, d, Input{Baseline: "A", Plans: map[string][]Record{"A": records(10, func(s int64) synth { return base("A", s) }), "C": records(10, empty)}})
	g := gate(t, plan(rep, "C"), "breach_rate", WinT90, "urgent")
	if g.DefinedSeeds != 0 || g.Worst.Defined || g.Mean != 0 || g.Status != Insufficient {
		t.Fatalf("%+v", g)
	}
	if math.IsNaN(g.Mean) {
		t.Fatal("NaN mean")
	}
}

// The obligation bridge counts each component once and nothing else:
// receivables and unbilled work are never set against cash.
func TestObligationBridge(t *testing.T) {
	m := Money{PayablesCents: 1, AccruedPayrollCents: 2, SeverancePayableCents: 4, EarnedUnpostedWagesCents: 8,
		EmployeeCommitmentCents: 16, RemoteInFlightCents: 32, ContractMinimumCents: 64, PastDueCents: 128,
		ReceivablesCents: 1 << 10, UnbilledCompletedCents: 1 << 11, CashCents: 1 << 12, RemoteRetainerCents: 1 << 13, RemoteReadsCents: 1 << 14}
	if m.Obligations() != 255 {
		t.Fatalf("obligations %d", m.Obligations())
	}
}

// Dispersion is 0 for identical seeds and large for scattered ones.
func TestDispersion(t *testing.T) {
	even := []SeedGate{{Num: 10, Den: 100}, {Num: 10, Den: 100}, {Num: 10, Den: 100}}
	if dispersion(even) != 0 {
		t.Fatal("identical seeds have no variance")
	}
	spread := []SeedGate{{Num: 0, Den: 100}, {Num: 20, Den: 100}, {Num: 10, Den: 100}, {Num: 10, Den: 100}}
	if d := dispersion(spread); d < 5 {
		t.Fatalf("dispersion %v", d)
	}
}
