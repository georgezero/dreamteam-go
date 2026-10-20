package eval

import (
	"math"
	"math/rand/v2"
	"slices"
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

// backlogRecord is a record of daily samples (backlog level, and a
// constant workload of 40 acquisitions a day) and the given studies,
// observed to day 181 with a day-150 horizon.
func backlogRecord(level func(d int) int, studies []Study) *Record {
	return workloadRecord(level, func(int) int { return 40 }, studies)
}

// workloadRecord is backlogRecord with the daily acquisitions given.
func workloadRecord(level, acq func(d int) int, studies []Study) *Record {
	r := &Record{TakeoverSec: t0, TakeoverDate: t0Date, HorizonEndSec: t0 + 150*daySec, TATTargetMin: testTargets, Studies: studies}
	date, _ := simtime.ParseDate(t0Date)
	cum := 0
	for d := 1; d <= 181; d++ {
		cum += acq(d)
		r.Samples = append(r.Samples, Sample{Day: d, AtSec: t0 + int64(d)*daySec, Date: date.AddDays(int64(d)).String(), Unread: level(d), AcquiredCum: cum})
	}
	return r
}

// 5C.2, under v0.2-eval-3's rule (ocEval3: the mean and peak hard at 10% /
// 2 studies, late work by block means): the backlog checks scale with the
// block's own workload (Little's law), so a seasonal rise in demand that a
// stable queue follows in proportion is not instability; stale and
// persistent work never scales, and growth beyond the workload, spikes,
// late-tail deterioration and a backlog that does not fall with demand
// fail. Each case beside the unadjusted (v0.2-eval-2) rule, for contrast.
// TestBacklogStability is v0.2-eval-4's.
func TestSeasonalBacklog(t *testing.T) {
	p0, _ := testPolicy(t)
	p := ocEval3(p0, true)
	v2 := *p
	v2.Backlog.WorkloadAdjustedChecks, v2.Backlog.AbsoluteChecks = nil, slices.Clone(backlogChecks)
	t90 := t0 + 90*daySec
	// Demand 40 a day through the reference block (days 55–82), then 50
	// (+25%, the order of the scenario's December-to-January rise).
	rising := func(d int) int {
		if d <= 82 {
			return 40
		}
		return 50
	}
	follows := func(lo, hi int) func(int) int {
		return func(d int) int {
			if d <= 82 {
				return lo
			}
			return hi
		}
	}
	run := func(pol *Policy, level, acq func(int) int, studies []Study) BacklogMeasures {
		return measureBacklog(pol, workloadRecord(level, acq, studies), t90)
	}
	// Stable service under rising seasonal demand: 20 → 25 with the
	// workload.
	if bm := run(p, follows(20, 25), rising, nil); bm.Status != "stable" {
		t.Fatalf("seasonal: %v", bm.Reasons)
	}
	if bm := run(&v2, follows(20, 25), rising, nil); bm.Status != "unstable" || !checkFailed(bm, "mean") {
		t.Fatalf("seasonal under v0.2-eval-2: %v", bm.Reasons)
	}
	// Genuine capacity deterioration: flat demand, 20 → 25; and rising
	// demand, 20 → 32 (+60% against +25%).
	if bm := run(p, follows(20, 25), func(int) int { return 40 }, nil); bm.Status != "unstable" || !checkFailed(bm, "mean") {
		t.Fatalf("deterioration: %v", bm.Reasons)
	}
	if bm := run(p, follows(20, 32), rising, nil); bm.Status != "unstable" || !checkFailed(bm, "mean") || !checkFailed(bm, "max") {
		t.Fatalf("deterioration beyond the workload: %v", bm.Reasons)
	}
	// A backlog that does not fall when demand falls (40 → 30): the
	// scaled reference is 15.
	if bm := run(p, follows(20, 20), follows(40, 30), nil); bm.Status != "unstable" || !checkFailed(bm, "mean") {
		t.Fatalf("falling demand: %v", bm.Reasons)
	}
	// Persistent old backlog: proportional growth, three inherited studies
	// never reported.
	var old []Study
	for k := range 3 {
		acq := t0 - 2*hour + int64(k)*60
		old = append(old, Study{ID: "I", Priority: "routine", AcquiredSec: acq, EnteredSec: acq})
	}
	if bm := run(p, follows(20, 25), rising, old); bm.Status != "unstable" || bm.Persistent != 3 {
		t.Fatalf("persistent: %+v", bm.Reasons)
	}
	// A growing stale burden under rising demand: from day 115 ten routine
	// studies wait a week. Stale never scales with the workload.
	var stale []Study
	for k := range 10 {
		acq := t0 + 113*daySec + int64(k)*60
		stale = append(stale, Study{ID: "S", Priority: "routine", AcquiredSec: acq, EnteredSec: acq, ClaimedSec: acq + 7*daySec,
			ReadStartSec: acq + 7*daySec, FinalSec: acq + 8*daySec})
	}
	if bm := run(p, follows(20, 25), rising, stale); bm.Status != "unstable" || !checkFailed(bm, "stale") || bm.Persistent != 0 {
		t.Fatalf("stale: %v", bm.Reasons)
	}
	// Stale work never scales: a stale burden present in the reference
	// that grows with the workload (14 studies a week waiting a week in the
	// reference, 18 later: a stale mean of about 10, then 12.9) fails the
	// absolute stale check; scaled by the workload it would pass.
	var burden []Study
	for start := 55; start < 167; start += 7 {
		n := 14
		if start > 82 {
			n = 18
		}
		for k := range n {
			acq := t0 + int64(start)*daySec + int64(k)*60
			burden = append(burden, Study{ID: "B", Priority: "routine", AcquiredSec: acq, EnteredSec: acq, ClaimedSec: acq + 7*daySec,
				ReadStartSec: acq + 7*daySec, FinalSec: acq + 7*daySec + hour})
		}
	}
	if bm := run(p, follows(20, 25), rising, burden); bm.Status != "unstable" || !checkFailed(bm, "stale") {
		t.Fatalf("proportional stale burden: %v", bm.Reasons)
	}
	// No recorded workload: compared unscaled, never waved through.
	if bm := run(p, follows(20, 25), func(int) int { return 0 }, nil); bm.Status != "unstable" || !checkFailed(bm, "mean") {
		t.Fatalf("no workload: %v", bm.Reasons)
	}
	// A temporary spike: three days at 31 against a scaled peak of 25.
	spike := func(d int) int {
		switch {
		case d <= 82:
			return 20
		case d >= 97 && d <= 99:
			return 31
		}
		return 25
	}
	if bm := run(p, spike, rising, nil); bm.Status != "unstable" || !checkFailed(bm, "max") || checkFailed(bm, "mean") {
		t.Fatalf("spike: %v", bm.Reasons)
	}
	// Late-tail deterioration in the covering block (139–166) alone.
	late := func(d int) int {
		switch {
		case d <= 82:
			return 20
		case d >= 139:
			return 33
		}
		return 25
	}
	bm := run(p, late, rising, nil)
	if bm.Status != "unstable" || !checkFailed(bm, "mean") {
		t.Fatalf("late tail: %v", bm.Reasons)
	}
	for _, c := range bm.Checks {
		if !c.Pass && c.FirstDay != 139 {
			t.Fatalf("late tail: block %d failed %s", c.FirstDay, c.Check)
		}
	}
	// The scaled reference is reported with its ratio; an absolute check
	// keeps ratio 1.
	for _, c := range run(p, follows(20, 25), rising, nil).Checks {
		want := 1.25
		if c.Check == "stale" {
			want = 1
		}
		if c.WorkloadRatio != want {
			t.Fatalf("%s ratio %v", c.Check, c.WorkloadRatio)
		}
	}
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
// reference comparison of v0.2-eval-1, under v0.2-eval-3's rule (ocEval3)
// and, where it differs, v0.2-eval-4's (v4).
func TestBacklogSafeguards(t *testing.T) {
	v4, _ := testPolicy(t)
	p := ocEval3(v4, true)
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
		if !c.Pass && !c.Informational {
			t.Fatalf("stream: only the persistent rule should fail: %+v", c)
		}
	}
	if bm.Status != "unstable" || bm.Persistent == 0 || !legacyStable(bm) {
		t.Fatalf("stream: %+v", bm)
	}
	// v0.2-eval-4: the stream is also above the stale cap in every block,
	// the reference included.
	if bm := measureBacklog(v4, backlogRecord(flat, stream), t90); bm.Status != "unstable" || !checkFailed(bm, "staleCap") || checkFailed(bm, "stale") {
		t.Fatalf("stream under v4: %v", bm.Reasons)
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
	// v0.2-eval-4: the peak is reported, never gated.
	if bm := measureBacklog(v4, backlogRecord(spike, nil), t90); bm.Status != "stable" || !checkFailed(bm, "max") {
		t.Fatalf("spike under v4: %v", bm.Reasons)
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
	// v0.2-eval-4: a step to a higher level with no late work is within
	// the level tolerance; growth without bound in the same block is not
	// (TestBacklogStability).
	if bm := measureBacklog(v4, backlogRecord(late, nil), t90); bm.Status != "stable" || !checkFailed(bm, "mean") {
		t.Fatalf("late step under v4: %v", bm.Reasons)
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
	// v0.2-eval-4: ten studies a week past due on every sample of days
	// 115–122 are a minority of the block's mornings: the medians do not
	// move. Late work on most mornings does (TestBacklogStability).
	if bm := measureBacklog(v4, backlogRecord(flat, stale), t90); bm.Status != "stable" {
		t.Fatalf("stale week under v4: %v", bm.Reasons)
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
	// v0.2-eval-4: a level within the tolerance passes whatever the
	// reference; old work in an inflated reference is capped
	// (TestBacklogStability).
	if bm := measureBacklog(v4, backlogRecord(inflated, nil), t90); bm.Status != "stable" {
		t.Fatalf("inflated under v4: %v", bm.Reasons)
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
// 5C.2: a solvent, clinically eligible plan that has not repaid its
// transition cost by T90 but is ahead over the operating window is
// CONDITIONAL GO (transition, not yet repaid), whatever T180 shows; a
// standing cost is a service purchase (when the baseline fails service)
// or no benefit. Nothing here is financial GO.
func TestTransitionClassification(t *testing.T) {
	p, d := testPolicy(t)
	baseline := func(soonPM int) func(int64) synth {
		return func(s int64) synth { x := base("A", s); x.breachPM["soon"] = soonPM; return x }
	}
	cand := func(pd, upfront int64) func(int64) synth {
		return func(s int64) synth {
			x := base("B", s)
			x.profitPD, x.upfront, x.profitNoise = pd, upfront, 100
			return x
		}
	}
	eval := func(a, b func(int64) synth) Comparison {
		rep := Evaluate(p, d, Input{Baseline: "A", Plans: map[string][]Record{"A": records(10, a), "B": records(10, b)}})
		return comparison(t, rep, "B")
	}
	// +$20 a day after a $3,000 transition: T90 −1,200 dollars, days 31–90
	// +1,200, T180 +600.
	invest := cand(12_000, 300_000)
	c := eval(baseline(200), invest)
	if c.Verdict != CONDITIONAL || c.Classification != ClassTransition || c.ServiceRestored ||
		!strings.Contains(reasons(c), "transition not yet repaid") || c.Financial.T90Increment.Mean >= 0 ||
		c.Financial.OperatingIncrement.Mean <= 0 || c.Financial.T180Increment.Mean <= 0 {
		t.Fatalf("transition: %s (%s) %v %+v", c.Verdict, c.Classification, c.Reasons, c.Financial)
	}
	if !c.Financial.T90Absolute.Defined || !c.Financial.T180Absolute.Defined || !c.Financial.TransitionObligationsT90.Defined {
		t.Fatalf("financial summary %+v", c.Financial)
	}
	// The same against a baseline failing service: still a transition, the
	// restoration stated beside it.
	c = eval(baseline(400), invest)
	if c.Classification != ClassTransition || !c.ServiceRestored || !strings.Contains(reasons(c), "restores service") {
		t.Fatalf("transition restoring service: %s %v", c.Classification, c.Reasons)
	}
	// A standing cost (behind in every window): a service purchase when the
	// baseline fails service, no benefit when it passes.
	costly := cand(8_000, 0)
	if c = eval(baseline(400), costly); c.Verdict != CONDITIONAL || c.Classification != ClassServicePurchase {
		t.Fatalf("service purchase: %s (%s)", c.Verdict, c.Classification)
	}
	if c = eval(baseline(200), costly); c.Verdict != NOGO || c.Classification != ClassNoBenefit {
		t.Fatalf("no benefit: %s (%s)", c.Verdict, c.Classification)
	}
	// The restoration is a fact about the pair whatever the verdict: an
	// insolvent candidate is NO GO, and still passes the service the
	// baseline fails.
	broke := func(s int64) synth { x := cand(8_000, 0)(s); x.minCash, x.minCashSet = -1, true; return x }
	if c = eval(baseline(400), broke); c.Verdict != NOGO || !c.ServiceRestored {
		t.Fatalf("restoration under NO GO: %s %v", c.Verdict, c.ServiceRestored)
	}
	if c = eval(baseline(200), broke); c.ServiceRestored {
		t.Fatal("a baseline that passes service is not restored")
	}
	// A large T180 gain never makes an unrepaid T90 financial GO.
	if c = eval(baseline(200), cand(30_000, 2_000_000)); c.Verdict == GO || c.Financial.T180Increment.Lo <= 0 || c.Financial.T90Increment.Mean >= 0 {
		t.Fatalf("T180 promoted: %s (%s) %+v", c.Verdict, c.Classification, c.Financial)
	}
}

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

// 5C.2: funding is a dated settlement test. Obligations above cash at an
// instant, the forward employee commitment included, are a reported
// indicator; what fails is a payment not made on its date (past due),
// cash below the floor at any posting, observation stopping before the
// horizon's obligations fall due, or committed liabilities outside the
// payment cycle that cash at the end does not cover.
func TestDatedFunding(t *testing.T) {
	p, d := testPolicy(t)
	eval := func(mk func(int64) synth, edit func(*Record)) (*PlanReport, Comparison) {
		t.Helper()
		recs := records(10, mk)
		for i := range recs {
			if edit != nil {
				edit(&recs[i])
			}
		}
		rep := Evaluate(p, d, Input{Baseline: "A", Plans: map[string][]Record{"A": records(10, func(s int64) synth { return base("A", s) }), "B": recs}})
		return plan(rep, "B"), comparison(t, rep, "B")
	}
	atEnd := func(f func(*Money)) func(*Record) {
		return func(r *Record) {
			for i := range r.Captures {
				if r.Captures[i].Name == "end" {
					f(&r.Captures[i].Money)
				}
			}
		}
	}
	// Obligations above cash for the first weeks (cash grows past them
	// later), every payment made on its date: the indicator reports the
	// gap; funding passes.
	tight := func(s int64) synth {
		x := base("B", s)
		x.profitPD, x.cash0, x.payables = 30_000, 100_000, 200_000
		return x
	}
	pr, c := eval(tight, nil)
	m := pr.PerSeed[0].Measures
	if pr.FundingStatus != Pass || m.FundingGap != 0 || m.CommitmentGap != 100_000-5_000 || m.CommitmentGapAt != t0+daySec ||
		len(pr.CommitmentGapSeeds) != 10 || !pr.CommitmentGapFraction.Defined || pr.CommitmentGapFraction.Num == 0 {
		t.Fatalf("indicator: %s gap %d, commitment %d at %d, %v %+v", pr.FundingStatus, m.FundingGap, m.CommitmentGap, m.CommitmentGapAt-t0,
			pr.CommitmentGapSeeds, pr.CommitmentGapFraction)
	}
	if strings.Contains(reasons(c), "funding") {
		t.Fatalf("the indicator gates: %v", c.Reasons)
	}
	// A deferred payment (past due at a capture) cannot hide behind cash.
	pr, c = eval(func(s int64) synth { return base("B", s) }, func(r *Record) { r.Captures[1].Money.PastDueCents = 1 })
	if pr.SolvencyStatus != Fail || c.Verdict != NOGO || !strings.Contains(reasons(c), "solvency") {
		t.Fatalf("past due: %s %s %v", pr.SolvencyStatus, c.Verdict, c.Reasons)
	}
	// Cash below zero at one posting between samples, every sample and
	// capture positive: insolvent.
	pr, c = eval(func(s int64) synth { x := base("B", s); x.minCash, x.minCashSet = -1, true; return x }, nil)
	if pr.SolvencyStatus != Fail || c.Verdict != NOGO {
		t.Fatalf("negative posting: %s %s", pr.SolvencyStatus, c.Verdict)
	}
	// Observation stopping before the horizon's obligations fall due:
	// the record is not evidence.
	pr, c = eval(func(s int64) synth { x := base("B", s); x.observeShort, x.followUp = daySec, 0; return x }, nil)
	if pr.Valid || c.Verdict != INVALID {
		t.Fatalf("truncated: valid %v %s", pr.Valid, c.Verdict)
	}
	// A late horizon (day 175): the cohorts' closure and the covering
	// backlog block end by day 194, the settlement window at day 206.
	late := Record{TakeoverSec: t0, TakeoverDate: t0Date, HorizonEndSec: t0 + 175*daySec}
	if got := observationNeeded(p, &late); got != late.HorizonEndSec+int64(p.Finance.SettlementDays)*daySec {
		t.Fatalf("observation to day %d does not end with the settlement window", (got-t0)/daySec)
	}
	// Committed liabilities outside the cycle above cash at the end:
	// severance payable, a contract's minimum retainers, fees in flight.
	for _, k := range []struct {
		name string
		set  func(*Money)
	}{
		{"severance", func(m *Money) { m.SeverancePayableCents = m.CashCents + 1 }},
		{"contract minimum", func(m *Money) { m.ContractMinimumCents = m.CashCents + 1 }},
		{"fees in flight", func(m *Money) { m.RemoteInFlightCents = m.CashCents/2 + 1; m.ContractMinimumCents = m.CashCents / 2 }},
	} {
		pr, c = eval(func(s int64) synth { return base("B", s) }, atEnd(k.set))
		if pr.FundingStatus != Fail || pr.PerSeed[0].Measures.FundingGap != 1 || c.Verdict != NOGO || !strings.Contains(reasons(c), "funding") {
			t.Fatalf("%s: %s gap %d %s", k.name, pr.FundingStatus, pr.PerSeed[0].Measures.FundingGap, c.Verdict)
		}
	}
	// Exactly covered passes; the recurring cycle's accrued liabilities
	// are never part of the terminal test.
	pr, _ = eval(func(s int64) synth { return base("B", s) }, atEnd(func(m *Money) {
		m.SeverancePayableCents, m.PayablesCents, m.AccruedPayrollCents = m.CashCents, 5*m.CashCents, m.CashCents
	}))
	if pr.FundingStatus != Pass {
		t.Fatalf("covered: %s %v", pr.FundingStatus, pr.FundingGapSeeds)
	}
	if l := pr.PerSeed[0].Measures.Liquidity["end"]; l.Committed != l.CashCents || l.Accrued != 6*l.CashCents {
		t.Fatalf("liquidity split %+v", l)
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
		b := base("B", 901)
		b.followUp = 0 // observed exactly as the v4 policy needs
		r := b.record()
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

// 5C.2: the cluster statistics. A design effect is about 1 for
// independent outcomes and the cluster size for perfectly clustered ones;
// a fixed weekday pattern adds nothing at the day level once removed; the
// cluster-robust pooled interval widens with seed-level variation; the
// exact upper bound on a failing-seed fraction is the Clopper–Pearson one.
func TestClusterStatistics(t *testing.T) {
	var indep, clustered []clusterGroup
	rng := rand.New(rand.NewPCG(5, 2))
	for g := range 200 {
		ones := int64(0)
		for range 20 {
			if rng.Float64() < 0.1 {
				ones++
			}
		}
		indep = append(indep, clusterGroup{n: 20, ones: ones})
		c := int64(0)
		if g%10 == 0 {
			c = 20
		}
		clustered = append(clustered, clusterGroup{n: 20, ones: c})
	}
	if de := designEffect(indep); de < 0.5 || de > 1.6 {
		t.Fatalf("independent design effect %v", de)
	}
	if de := designEffect(clustered); de < 19 || de > 20.01 {
		t.Fatalf("clustered design effect %v", de)
	}
	if designEffect(indep[:1]) != 0 {
		t.Fatal("one group has no design effect")
	}
	// Every Saturday breaches, no other day: a fixed weekday pattern, no
	// random day effect.
	var xs []*Study
	for d := int64(0); d < 70; d++ {
		for i := range int64(10) {
			acq := t0 + d*daySec + 9*hour + i*60
			fin := acq + 10*60
			if ((acq/daySec)+4)%7 == 6 { // Saturday
				fin = acq + 2*hour
			}
			xs = append(xs, &Study{Priority: "urgent", AcquiredSec: acq, EnteredSec: acq, ClaimedSec: acq, ReadStartSec: acq, FinalSec: fin})
		}
	}
	breach := func(s *Study) (bool, bool) { return s.FinalSec > s.AcquiredSec+3600, true }
	day, week := clusterEffects(xs, breach)
	if day != 0 && day > 1.01 {
		t.Fatalf("weekday pattern counted as day clustering: %v", day)
	}
	if week > 1.01 && week != 0 {
		t.Fatalf("a pattern repeating every week clusters by week: %v", week)
	}
	// Seeds alike against seeds scattered, same pooled rate.
	alike := clusterRobustRate([]int64{10, 10, 10, 10}, []int64{100, 100, 100, 100})
	spread := clusterRobustRate([]int64{0, 20, 0, 20}, []int64{100, 100, 100, 100})
	if alike.Mean != 0.1 || spread.Mean != 0.1 || alike.Hi-alike.Lo > 1e-12 || spread.Hi-spread.Lo < 0.2 {
		t.Fatalf("robust intervals %+v %+v", alike, spread)
	}
	for _, k := range []struct {
		x, n int
		want float64
	}{{0, 10, 1 - math.Pow(0.05, 0.1)}, {0, 40, 1 - math.Pow(0.05, 1.0/40)}, {10, 10, 1}} {
		if got := binomUpper(k.x, k.n, 0.95); math.Abs(got-k.want) > 1e-9 {
			t.Fatalf("upper bound %d of %d: %v, want %v", k.x, k.n, got, k.want)
		}
	}
	// At the bound, P(X ≤ x) is 5%.
	u := binomUpper(4, 40, 0.95)
	var cdf float64
	for k := range 5 {
		cdf += math.Exp(lpmf(40, k, u))
	}
	if math.Abs(cdf-0.05) > 1e-9 || u < 0.2 || u > 0.23 {
		t.Fatalf("upper bound 4 of 40: %v (cdf %v)", u, cdf)
	}
}
