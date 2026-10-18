package eval

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/simtime"
)

func gate(t *testing.T, pr *PlanReport, g, w, s string) GateSummary {
	t.Helper()
	for _, x := range pr.ServiceGates {
		if x.Gate == g && x.Window == w && x.Stratum == s {
			return x
		}
	}
	t.Fatalf("%s: no gate %s %s %s", pr.Plan, g, w, s)
	return GateSummary{}
}

func comparison(t *testing.T, rep *Report, cand string) Comparison {
	t.Helper()
	for _, c := range rep.Comparisons {
		if c.Candidate == cand && c.Against == rep.Baseline {
			return c
		}
	}
	t.Fatalf("no comparison for %s", cand)
	return Comparison{}
}

func plan(rep *Report, name string) *PlanReport {
	for i := range rep.Plans {
		if rep.Plans[i].Plan == name {
			return &rep.Plans[i]
		}
	}
	return nil
}

// The synthetic baseline checks out: no integrity failure, every gate
// decided, and its measured rates are the fixture's.
func TestSyntheticRecordChecksOut(t *testing.T) {
	p, d := testPolicy(t)
	rep := Evaluate(p, d, Input{Baseline: "A", Plans: map[string][]Record{"A": records(10, func(s int64) synth { return base("A", s) })}})
	pr := plan(rep, "A")
	if !pr.Valid || len(pr.Integrity) != 0 || !pr.Eligible {
		t.Fatalf("%v %v", pr.Integrity, pr.Eligible)
	}
	g := gate(t, pr, "breach_rate", "T90", "urgent")
	if g.Status != Pass || g.Pooled.Den != 900 || math.Abs(g.Pooled.Value-0.1) > 0.02 {
		t.Fatalf("%+v", g)
	}
}

// 1. An empty priority is not a 0% breach rate: it lacks evidence, the
// seed does not pass, the pooled rate is undefined, and the plan cannot
// be GO.
func TestEmptyStratumDoesNotPass(t *testing.T) {
	p, d := testPolicy(t)
	cand := func(s int64) synth {
		x := base("B", s)
		x.perDay["urgent"] = 0
		x.profitPD = 30_000
		return x
	}
	rep := Evaluate(p, d, Input{Baseline: "A", Plans: map[string][]Record{
		"A": records(10, func(s int64) synth { return base("A", s) }), "B": records(10, cand)}})
	pr := plan(rep, "B")
	g := gate(t, pr, "breach_rate", "T90", "urgent")
	if g.Status != Insufficient || g.Pooled.Defined || g.InsufficientSeeds != 10 || g.Passing != 0 {
		t.Fatalf("%+v", g)
	}
	if pr.PerSeed[0].Service != Insufficient || pr.ServiceStatus != Insufficient || pr.Eligible {
		t.Fatalf("seed %s, plan %s, eligible %v", pr.PerSeed[0].Service, pr.ServiceStatus, pr.Eligible)
	}
	if c := comparison(t, rep, "B"); c.Verdict != NOGO {
		t.Fatalf("%s %v", c.Verdict, c.Reasons)
	}
}

// 2. Too few acquisitions, or studies still undecided when observation
// ends, are identified as insufficient evidence, with the reason.
func TestInsufficientCohortsIdentified(t *testing.T) {
	p, _ := testPolicy(t)
	if p.Service.MinCohort["urgent"] != 43 {
		t.Fatalf("minimum urgent cohort %d", p.Service.MinCohort["urgent"])
	}
	thin := base("A", 901)
	thin.perDay["urgent"] = 0
	r := thin.record()
	// 40 urgent studies in the first 90 days: three short of the minimum.
	for k := range 40 {
		acq := t0 + int64(k)*2*daySec + hour
		r.Studies = append(r.Studies, Study{Priority: "urgent", Modality: "MR", AcquiredSec: acq, EnteredSec: acq, ClaimedSec: acq, ReadStartSec: acq, FinalSec: acq + 1800})
	}
	r.Samples = nil
	for dd := 1; int64(dd)*daySec <= r.ObservedUntilSec-t0; dd++ {
		at := t0 + int64(dd)*daySec
		r.Samples = append(r.Samples, thin.sample(&r, dd, at, thin.money(at)))
	}
	m := Measure(p, &r)
	if len(m.Integrity) != 0 {
		t.Fatal(m.Integrity)
	}
	gs := serviceGates(p, m)
	var g SeedGate
	for _, x := range gs {
		if x.Window == WinT90 && x.Stratum == "urgent" {
			g = x
		}
	}
	if g.Status != Insufficient || !strings.Contains(g.Reason, "40 acquired, fewer than 43") || g.Value != 0 {
		t.Fatalf("%+v", g)
	}
	// Observation stopping before the tail cohort closes leaves studies
	// undecided: insufficient, never a pass, and it is an integrity failure.
	short := base("A", 902)
	short.observeShort = 30*daySec + 20*hour
	sr := short.record()
	sr.Studies = append(sr.Studies, Study{Priority: "routine", Modality: "XR", AcquiredSec: short.tailEnd() - hour})
	m = Measure(p, &sr)
	c := m.Windows[WinTail].Cohorts["routine"]
	if c.Unresolved == 0 || c.BreachRate.Defined || !hasPrefix(m.Integrity, "observed to") {
		t.Fatalf("%+v %v", c, m.Integrity)
	}
}

func ptr(r Record) *Record { return &r }

func hasPrefix(xs []string, p string) bool {
	return slices.ContainsFunc(xs, func(s string) bool { return strings.HasPrefix(s, p) })
}

// 3. An unfinished study counts as a breach once it is past its target,
// is reported as overdue at the window's end with its overdue minutes, and
// a study acquired just before the end that is reported within target
// after it is not a breach (the historical as-of-end rule differs).
func TestUnfinishedOverdueCounted(t *testing.T) {
	r := &Record{TATTargetMin: testTargets, ObservedUntilSec: t0 + 10*daySec}
	b := t0 + 5*daySec
	xs := []*Study{
		{Priority: "soon", AcquiredSec: b - 10*hour},                      // never final: overdue 6 h at b
		{Priority: "soon", AcquiredSec: b - 1*hour, FinalSec: b + 2*hour}, // final after b within 4 h: met
		{Priority: "soon", AcquiredSec: b - 3*hour, FinalSec: b + 2*hour}, // final 5 h after: breach, decided after b
		{Priority: "soon", AcquiredSec: b - 8*hour, FinalSec: b - 5*hour}, // final in 3 h: met
		{Priority: "soon", AcquiredSec: r.ObservedUntilSec - 3*hour},      // due after observation: undecided
	}
	c := cohort(r, xs[:4], b)
	if c.Acquired != 4 || c.Finalized != 1 || c.Unfinished != 3 || c.Breaches != 2 || c.OverdueUnfinished != 1 {
		t.Fatalf("%+v", c)
	}
	// Overdue minutes within the window: 6 h for the first; the third is
	// due at b + 1 h, after the window's end.
	if c.OverdueStudyMin != 6*60 || c.UnfinishedAgeMin.Max != 10*60 || !c.BreachRate.Defined || c.BreachRate.Value != 0.5 {
		t.Fatalf("%+v", c)
	}
	// As of b, the legacy rule sees one breach (the overdue one).
	if c.LegacyBreaches != 1 {
		t.Fatalf("legacy %d", c.LegacyBreaches)
	}
	if c := cohort(r, xs, b+10*hour); c.Unresolved != 1 || c.BreachRate.Defined {
		t.Fatalf("%+v", c)
	}
}

// 4. Backlog stability compares whole 28-day blocks that start on a rota
// cycle boundary (week-A Monday): a backlog with a two-week pattern is
// stable, though single samples on different weekdays (the historical
// rule) see growth; real growth is unstable.
func TestBacklogTrendRotaAligned(t *testing.T) {
	p, _ := testPolicy(t)
	anchor, _ := simtime.ParseDate(p.Scenario.RotaWeekA)
	mk := func(level func(d int) int) *Record {
		r := &Record{TakeoverSec: t0, TakeoverDate: t0Date, HorizonEndSec: t0 + 150*daySec, TATTargetMin: testTargets}
		date, _ := simtime.ParseDate(t0Date)
		for d := 1; d <= 181; d++ {
			x := date.AddDays(int64(d))
			r.Samples = append(r.Samples, Sample{Day: d, AtSec: t0 + int64(d)*daySec, Date: x.String(), Unread: level(d)})
		}
		return r
	}
	final := func(bm BacklogMeasures) BlockCheck {
		for _, c := range bm.Checks {
			if c.Check == "mean" && c.FirstDay == bm.Later[len(bm.Later)-1].FirstDay {
				return c
			}
		}
		t.Fatalf("no final mean check: %+v", bm)
		return BlockCheck{}
	}
	cycle := func(d int) int {
		x, _ := simtime.ParseDate(t0Date)
		k := mod(x.AddDays(int64(d)).Ordinal()-anchor.Ordinal(), 14)
		return 10 + int(k)*3 // 10 on week-A Mondays … 49 on week-B Sundays
	}
	bm := measureBacklog(p, mk(cycle), t0+90*daySec)
	if bm.Status != "stable" || final(bm).Value != final(bm).Reference {
		t.Fatalf("%+v", bm)
	}
	for _, b := range append([]Block{bm.Reference}, bm.Later...) {
		x, _ := simtime.ParseDate(b.FirstDate)
		if mod(x.Ordinal()-anchor.Ordinal(), 14) != 0 || b.LastDay-b.FirstDay != 27 {
			t.Fatalf("block %+v not rota-aligned", b)
		}
	}
	// The later blocks follow the reference through the one covering the
	// horizon (day 150): 83–110, 111–138, 139–166.
	if bm.Reference.LastDay > 90 || len(bm.Later) != 3 || bm.Later[0].FirstDay != bm.Reference.LastDay+1 || bm.Later[2].LastDay != 166 {
		t.Fatalf("blocks %+v %+v", bm.Reference, bm.Later)
	}
	// Day 90 is a week-B Monday (level 31), day 180 a week-B Sunday (49):
	// the single-sample rule reports growth that is only the rota.
	if !bm.LegacyDefined || bm.LegacyHolds {
		t.Fatalf("legacy %d → %d holds %v", bm.LegacyDay90, bm.LegacyDay180, bm.LegacyHolds)
	}
	// Slow growth: unstable under v0.2-eval-3's mean (10% / 2 studies); under
	// v0.2-eval-4 within the level tolerance, its mean check and slope
	// reported. Growth without bound fails the level.
	grow := func(d int) int { return cycle(d) + d/10 }
	if bm := measureBacklog(ocEval3(p, true), mk(grow), t0+90*daySec); bm.Status != "unstable" || bm.SlopePerDay <= 0 {
		t.Fatalf("%+v", bm)
	}
	if bm := measureBacklog(p, mk(grow), t0+90*daySec); bm.Status != "stable" || !checkFailed(bm, "mean") || bm.SlopePerDay <= 0 {
		t.Fatalf("%+v", bm)
	}
	fast := func(d int) int { return cycle(d) + max(0, d-82) }
	if bm := measureBacklog(p, mk(fast), t0+90*daySec); bm.Status != "unstable" || !checkFailed(bm, "level") {
		t.Fatalf("%+v", bm)
	}
	// Within tolerance: max(10%, 2 studies) of the reference mean.
	small := func(d int) int {
		if d > 100 {
			return cycle(d) + 2
		}
		return cycle(d)
	}
	if bm := measureBacklog(p, mk(small), t0+90*daySec); bm.Status != "stable" || final(bm).Value-final(bm).Reference != 2 {
		t.Fatalf("%+v", bm)
	}
}

// 5. Window boundaries are (a, b]: a study acquired exactly at the window's
// end is in it, one at its start is not; a report exactly at its due time
// meets the target, a second later breaches; turnaround spans midnight.
func TestMidnightBoundaries(t *testing.T) {
	r := &Record{TATTargetMin: testTargets, ObservedUntilSec: t0 + 3*daySec}
	a, b := t0, t0+daySec
	xs := []Study{
		{Priority: "urgent", AcquiredSec: a, FinalSec: a + 1800},                        // at the start: excluded
		{Priority: "urgent", AcquiredSec: b, FinalSec: b + 3600},                        // at the end, final exactly at due: met
		{Priority: "urgent", AcquiredSec: a + 18*hour - 60, FinalSec: a + 19*hour},      // 23:59 to 01:00: 61 min, breach
		{Priority: "urgent", AcquiredSec: a + 18*hour - 30, FinalSec: a + 18*hour + 30}, // 23:59:30 to 00:00:30
	}
	r.Studies = xs
	w := measureWindow(r, a, b, nil, &Capture{}, &Capture{})
	c := w.Cohorts["urgent"]
	if c.Acquired != 3 || c.Breaches != 1 || c.Finalized != 2 {
		t.Fatalf("%+v", c)
	}
	if c.CompletedTATMin.Max != 61 || c.CompletedTATMin.P50 != 1 {
		t.Fatalf("%+v", c.CompletedTATMin)
	}
}

// 6. Studies imaged before the takeover and orders received before it are
// the inherited cohort: never in a new-demand window.
func TestInheritedBacklogSeparate(t *testing.T) {
	p, _ := testPolicy(t)
	x := base("A", 901)
	x.inherited = 7
	r := x.record()
	m := Measure(p, &r)
	if len(m.Integrity) != 0 {
		t.Fatal(m.Integrity)
	}
	plain := Measure(p, ptr(base("A", 901).record()))
	if m.Windows[WinT90].Cohorts["routine"].Acquired != plain.Windows[WinT90].Cohorts["routine"].Acquired ||
		m.Windows[WinT90].Access.Received != plain.Windows[WinT90].Access.Received {
		t.Fatal("inherited work counted as new")
	}
	if m.Inherited.Studies["routine"].Acquired != 7 || m.Inherited.Orders != 7 || m.Inherited.UnfinishedAtT90 != 0 {
		t.Fatalf("%+v", m.Inherited)
	}
	// They are in the backlog until reported (the samples include them).
	if pr := base("A", 901).record(); r.Samples[0].Backlog() != pr.Samples[0].Backlog()+7 {
		t.Fatal("no inherited backlog in the samples")
	}
}

// 7. Local and remote reports add up to every report, and each remote
// report's locked fee sums to the ledger's read-fee expense; a record
// whose ledger disagrees is invalid.
func TestRemoteAndLocalReconcile(t *testing.T) {
	p, d := testPolicy(t)
	tele := func(s int64) synth {
		x := base("T", s)
		x.remotePM, x.fee, x.retainerPD = 300, 1_200, 4_000
		return x
	}
	r := tele(901).record()
	m := Measure(p, &r)
	if len(m.Integrity) != 0 {
		t.Fatal(m.Integrity)
	}
	w := m.Windows[WinT90]
	var finals int64
	for _, s := range r.Studies {
		if s.FinalSec > t0 && s.FinalSec <= t0+90*daySec {
			finals++
		}
	}
	rm := w.Remote
	if rm.RemoteReports == 0 || rm.LocalReports+rm.RemoteReports != finals || rm.ReadFeeCents != rm.RemoteReports*1_200 {
		t.Fatalf("%+v (%d finals)", rm, finals)
	}
	// 8. Vendor cost: retainer and read fees from the ledger, per report.
	if rm.RetainerCents != 90*4_000 || rm.TotalCostCents != rm.RetainerCents+rm.ReadFeeCents ||
		rm.CostPerRemoteCent.Value != float64(rm.TotalCostCents)/float64(rm.RemoteReports) || rm.HandoffWaitMin.P50 != 10 {
		t.Fatalf("%+v", rm)
	}
	if tw := m.Windows[WinTail].Remote; tw.RetainerCents != 60*4_000 {
		t.Fatalf("tail retainer %d", tw.RetainerCents)
	}
	recs := records(10, tele)
	recs[3].Captures[1].Money.RemoteReadsCents += 1
	rep := Evaluate(p, d, Input{Baseline: "A", Plans: map[string][]Record{"A": records(10, func(s int64) synth { return base("A", s) }), "T": recs}})
	if c := comparison(t, rep, "T"); c.Verdict != INVALID || !strings.Contains(strings.Join(c.Reasons, " "), "") {
		t.Fatalf("%s", c.Verdict)
	}
	if pr := plan(rep, "T"); pr.Valid || !hasPrefix(pr.Integrity, "seed 904: window T90: remote reports' fees") {
		t.Fatal(pr.Integrity)
	}
}

// 9. Solvency uses the lowest cash after any posting (every transaction),
// not the daily samples: a dip between samples fails it.
func TestTransactionLevelCashMinimum(t *testing.T) {
	p, d := testPolicy(t)
	dip := func(s int64) synth {
		x := base("B", s)
		x.profitPD = 30_000
		if s == 905 {
			x.minCash, x.minCashSet = -1, true // one cent below zero, between samples
		}
		return x
	}
	recs := records(10, dip)
	for _, smp := range recs[4].Samples {
		if smp.CashCents < 0 {
			t.Fatal("the fixture's samples must stay positive")
		}
	}
	rep := Evaluate(p, d, Input{Baseline: "A", Plans: map[string][]Record{"A": records(10, func(s int64) synth { return base("A", s) }), "B": recs}})
	pr := plan(rep, "B")
	if pr.SolvencyStatus != Fail || !slices.Equal(pr.InsolventSeeds, []int64{905}) || pr.WorstMinCash != -1 {
		t.Fatalf("%s %v %d", pr.SolvencyStatus, pr.InsolventSeeds, pr.WorstMinCash)
	}
	if c := comparison(t, rep, "B"); c.Verdict != NOGO {
		t.Fatalf("%s", c.Verdict)
	}
}

// 10. Payback is the sustained break-even: a later dip below zero resets
// it, a seed below zero at the horizon is censored, and the median is
// reported only when at least half the seeds have repaid.
func TestPaybackCensoring(t *testing.T) {
	p, _ := testPolicy(t)
	mk := func(seed int64, inc func(d int) int64) (Record, Record) {
		a := Record{Seed: seed}
		b := Record{Seed: seed}
		for d := 1; d <= 90; d++ {
			a.Samples = append(a.Samples, Sample{Day: d, ProfitCents: inc(d)})
			b.Samples = append(b.Samples, Sample{Day: d})
		}
		return a, b
	}
	cases := []func(d int) int64{
		func(d int) int64 { return int64(d - 30) },                          // repaid day 30
		func(d int) int64 { return int64(d - 50) },                          // day 50
		func(d int) int64 { return []int64{1, -1}[(d/40)%2] * int64(d-10) }, // dips at 40–79, repaid at 80
		func(d int) int64 { return int64(d - 120) },                         // censored
		func(d int) int64 { return -5 },                                     // censored
	}
	var cr []Record
	ri := map[int64]*Record{}
	for i, f := range cases {
		a, b := mk(int64(i+1), f)
		cr = append(cr, a)
		ri[a.Seed] = &b
	}
	pb, _ := payback(p, 90, cr, ri, func(s Sample) int64 { return s.ProfitCents })
	if !slices.Equal(pb.Days, []int{30, 50, 80, 0, 0}) || pb.Repaid != 3 || pb.Censored != 2 || !pb.MedianDefined || pb.Median != 80 {
		t.Fatalf("%+v", pb)
	}
	// Two of four: exactly half have repaid, by day 80.
	pb, _ = payback(p, 90, cr[1:], ri, func(s Sample) int64 { return s.ProfitCents })
	if !pb.MedianDefined || pb.Median != 80 || pb.Repaid != 2 || pb.Earliest != 50 || pb.Latest != 80 {
		t.Fatalf("%+v", pb)
	}
	// One of three: fewer than half, no median.
	pb, _ = payback(p, 90, cr[2:], ri, func(s Sample) int64 { return s.ProfitCents })
	if pb.MedianDefined || pb.Median != 0 || pb.Repaid != 1 || pb.Censored != 2 {
		t.Fatalf("%+v", pb)
	}
	// A seed without samples through the horizon is not left out
	// silently: it is returned as missing (the comparison is then
	// invalid), and no median is reported.
	if pb, missing := payback(p, 180, cr, ri, func(s Sample) int64 { return s.ProfitCents }); len(pb.Days) != 0 || pb.MedianDefined || len(missing) != 5 || pb.Seeds != 5 {
		t.Fatalf("%+v %v", pb, missing)
	}
}

// 11. A plan cannot pass the service gates by declining cases: dropping
// routine demand (or hard urgent cases) shows in the case-selection
// gates and the plan is NO GO, whatever its profit and breach rates.
func TestCaseSelectionCannotBeBypassed(t *testing.T) {
	p, d := testPolicy(t)
	baseA := func(s int64) synth { return base("A", s) }
	declineRoutine := func(s int64) synth {
		x := base("B", s)
		x.declinePM["routine"] = 60
		x.breachPM = map[string]int{"urgent": 50, "soon": 100, "routine": 10}
		x.profitPD = 50_000
		return x
	}
	dropUrgent := func(s int64) synth {
		x := base("C", s)
		x.declinePM["urgent"] = 200
		x.profitPD = 50_000
		return x
	}
	rep := Evaluate(p, d, Input{Baseline: "A", Plans: map[string][]Record{"A": records(10, baseA), "B": records(10, declineRoutine), "C": records(10, dropUrgent)}})
	for _, name := range []string{"B", "C"} {
		c := comparison(t, rep, name)
		pr := plan(rep, name)
		if pr.ServiceStatus != Pass {
			t.Fatalf("%s: the breach gates should pass on their own (%s)", name, pr.ServiceStatus)
		}
		if c.CaseSelectionStatus != Fail || c.Verdict != NOGO || !strings.Contains(strings.Join(c.Reasons, "\n"), "case selection") {
			t.Fatalf("%s: %s %s %v", name, c.CaseSelectionStatus, c.Verdict, c.Reasons)
		}
	}
	// Walk-ins turned away likewise.
	leave := func(s int64) synth {
		x := base("D", s)
		x.walkIns, x.walkLeft, x.profitPD = 2, 100, 50_000
		return x
	}
	walk := func(s int64) synth {
		x := base("A", s)
		x.walkIns = 2
		return x
	}
	rep = Evaluate(p, d, Input{Baseline: "A", Plans: map[string][]Record{"A": records(10, walk), "D": records(10, leave)}})
	if c := comparison(t, rep, "D"); c.CaseSelectionStatus != Fail || c.Verdict != NOGO {
		t.Fatalf("%s %s", c.CaseSelectionStatus, c.Verdict)
	}
}

// 12. The historical results the evaluation must not touch are byte for
// byte as recorded in the policy.
func TestHistoricalReferencesUnchanged(t *testing.T) {
	p, _ := testPolicy(t)
	if len(p.Historical) < 30 {
		t.Fatalf("%d references", len(p.Historical))
	}
	for _, h := range p.Historical {
		raw, err := os.ReadFile("../../" + h.Path)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(raw)
		if hex.EncodeToString(sum[:]) != h.SHA256 {
			t.Errorf("%s changed", h.Path)
		}
	}
}

// 13. The output depends only on the policy and records: the same input,
// in any order, gives the same bytes.
func TestEvaluationDeterministic(t *testing.T) {
	p, d := testPolicy(t)
	mk := func() Input {
		return Input{Baseline: "A", Pairs: [][2]string{{"T", "B"}}, Plans: map[string][]Record{
			"A": records(10, func(s int64) synth { return base("A", s) }),
			"B": records(10, func(s int64) synth { x := base("B", s); x.profitPD = 20_000; x.profitNoise = 900; return x }),
			"T": records(10, func(s int64) synth {
				x := base("T", s)
				x.remotePM, x.fee, x.retainerPD, x.breachPM["soon"] = 300, 1_200, 4_000, 100
				return x
			})}}
	}
	a, _ := json.Marshal(Evaluate(p, d, mk()))
	in := mk()
	for k := range in.Plans {
		slices.Reverse(in.Plans[k])
	}
	b, _ := json.Marshal(Evaluate(p, d, in))
	if !bytes.Equal(a, b) || len(a) < 1000 {
		t.Fatal("evaluation output differs between runs")
	}
}

// The verdict rules (policy "verdict"): GO, the two CONDITIONAL GOs, NO GO
// without benefit, and INVALID for an action at the primary endpoint; a
// late effect's tail is every plan's (the common horizon).
func TestVerdictRules(t *testing.T) {
	p, d := testPolicy(t)
	const h = 170 * daySec // E's tail end: every record's horizon
	failingBase := func(s int64) synth {
		x := base("A", s)
		x.breachPM["soon"] = 400 // 40% > 32%: the baseline fails service
		x.horizon = h
		return x
	}
	good := func(name string, pd, noise int64) func(int64) synth {
		return func(s int64) synth {
			x := base(name, s)
			x.profitPD, x.profitNoise, x.horizon = pd, noise, h
			return x
		}
	}
	late := func(s int64) synth {
		x := base("L", s)
		x.lastAction, x.horizon = 90*daySec, h
		return x
	}
	effect := func(s int64) synth {
		x := base("E", s)
		x.lastAction, x.effect = 20*daySec, 140*daySec // the tail runs to day 170
		x.profitPD = 30_000
		return x
	}
	rep := Evaluate(p, d, Input{Baseline: "A", Plans: map[string][]Record{
		"A": records(10, failingBase), "G": records(10, good("G", 30_000, 0)), "N": records(10, good("N", 10_500, 4_000)),
		"S": records(10, good("S", 4_000, 0)), "L": records(10, late), "E": records(10, effect)}})
	want := map[string][2]string{"G": {GO, ClassFinancial}, "N": {CONDITIONAL, ClassRelative}, "S": {CONDITIONAL, ClassServicePurchase},
		"L": {INVALID, ClassNotJudged}, "E": {GO, ClassFinancial}}
	for n, v := range want {
		if c := comparison(t, rep, n); c.Verdict != v[0] || c.Classification != v[1] {
			t.Errorf("%s: %s (%s), want %s (%s) %v", n, c.Verdict, c.Classification, v[0], v[1], c.Reasons)
		}
	}
	if pr := plan(rep, "E"); pr.PerSeed[0].Measures.TailEnd != t0+170*daySec {
		t.Fatalf("tail end %d", pr.PerSeed[0].Measures.TailEnd-t0)
	}
	// A passing baseline: a costlier plan with no benefit is NO GO.
	rep = Evaluate(p, d, Input{Baseline: "A", Plans: map[string][]Record{
		"A": records(10, func(s int64) synth { x := base("A", s); x.horizon = h; return x }), "S": records(10, good("S", 4_000, 0))}})
	if c := comparison(t, rep, "S"); c.Verdict != NOGO || c.Classification != ClassNoBenefit {
		t.Fatalf("%s %v", c.Verdict, c.Reasons)
	}
}

// A teleradiology-only variant must book exactly what its plan books.
func TestTeleOnlyBookingInvariance(t *testing.T) {
	p, d := testPolicy(t)
	tele := func(sha string) func(int64) synth {
		return func(s int64) synth {
			x := base("T", s)
			x.teleOnlyOf, x.apptSHA = "A", sha
			if s == 907 && sha == "" {
				x.apptSHA = "other"
			} else {
				x.apptSHA = "same"
			}
			return x
		}
	}
	for _, broken := range []bool{false, true} {
		sha := "same"
		if broken {
			sha = ""
		}
		rep := Evaluate(p, d, Input{Baseline: "A", Plans: map[string][]Record{"A": records(10, func(s int64) synth { return base("A", s) }), "T": records(10, tele(sha))}})
		pr := plan(rep, "T")
		if pr.BookingInvariant == broken || pr.Valid == broken || (comparison(t, rep, "T").Verdict == INVALID) != broken {
			t.Fatalf("broken %v: invariant %v valid %v %v", broken, pr.BookingInvariant, pr.Valid, pr.Integrity)
		}
	}
}

// The frozen policy: the owner's gates unchanged, the evidence minimums
// from their arithmetic (binomial standard error at the gate ≤ 5 points),
// and the closure covering the longest target.
func TestPolicyFrozen(t *testing.T) {
	p, _ := testPolicy(t)
	want := map[string]float64{"urgent": 0.12, "soon": 0.32, "routine": 0.06}
	for q, g := range want {
		if p.Service.BreachGate[q] != g {
			t.Fatalf("%s gate %v", q, p.Service.BreachGate[q])
		}
		if n := int64(math.Ceil(g * (1 - g) / (0.05 * 0.05))); p.Service.MinCohort[q] != n {
			t.Fatalf("%s minimum %d, the rule gives %d", q, p.Service.MinCohort[q], n)
		}
	}
	if p.Horizons.ClosureMin != 1440 || !slices.Equal(p.Service.Gated, []string{"T90", "tail"}) || p.Finance.MinCashFloorCents != 0 {
		t.Fatalf("%+v", p.Horizons)
	}
	if !slices.Equal(p.Service.PerSeedWindows["urgent"], []string{WinGated}) || p.Service.PlanningRate["urgent"] != 0.08 {
		t.Fatalf("%+v", p.Service)
	}
	raw, _ := os.ReadFile("../../" + PolicyPath)
	bad := bytes.Replace(raw, []byte(`"closureMin": 1440`), []byte(`"closureMin": 1440, "extra": 1`), 1)
	if _, err := DecodePolicy(bad); err == nil {
		t.Fatal("an unknown field is accepted")
	}
	// A tolerance that is not a whole number of basis points is refused.
	frac := bytes.Replace(raw, []byte(`"pooledMaxServedDrop": 0.02`), []byte(`"pooledMaxServedDrop": 0.02005`), 1)
	if _, err := DecodePolicy(frac); err == nil || !strings.Contains(err.Error(), "basis points") {
		t.Fatalf("a fractional basis point is accepted: %v", err)
	}
	// The superseded policies are kept byte for byte, each named by digest
	// by its successor, and refused as policies to evaluate under.
	type sup struct {
		Supersedes struct{ Path, SHA256 string }
	}
	chain := []string{PolicyV3Path, PolicyV2Path, PolicyV1Path}
	next := sup{}
	next.Supersedes.Path, next.Supersedes.SHA256 = p.Supersedes.Path, p.Supersedes.SHA256
	if p.Supersedes.SHA256 != "0cf4884f216170c7bba4d869039a86ba8fba70d76f31c2e36aebf58620213915" {
		t.Fatal("v0.2-eval-3 changed")
	}
	for _, path := range chain {
		old, err := os.ReadFile("../../" + path)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(old)
		if next.Supersedes.Path != path || hex.EncodeToString(sum[:]) != next.Supersedes.SHA256 {
			t.Fatalf("%s is not the policy its successor supersedes (%s %s)", path, next.Supersedes.Path, next.Supersedes.SHA256)
		}
		if _, err := DecodePolicy(old); err == nil || !strings.Contains(err.Error(), "superseded") {
			t.Fatalf("superseded %s decodes: %v", path, err)
		}
		next = sup{}
		json.Unmarshal(old, &next)
	}
	if len(p.Changes) != 4 || len(p.Deferred) == 0 || p.Finance.SettlementDays != 31 || p.Backlog.LateStatistic != "median" ||
		!slices.Equal(p.Backlog.InformationalChecks, []string{"mean", "max"}) {
		t.Fatalf("%d changes, %d deferred, %+v", len(p.Changes), len(p.Deferred), p.Backlog)
	}
	// Old and persistent work cannot be normalized, nor the age limit made
	// informational.
	for _, x := range [][2]string{
		{`"absoluteChecks": [
   "stale",`, `"absoluteChecks": [`},
		{`"informationalChecks": [
   "mean",`, `"informationalChecks": [
   "persistent",
   "mean",`},
	} {
		bad := bytes.Replace(raw, []byte(x[0]), []byte(x[1]), 1)
		if bytes.Equal(bad, raw) {
			t.Fatalf("no %q in the policy", x[0])
		}
		if x[1] == `"absoluteChecks": [` {
			bad = bytes.Replace(bad, []byte(`"workloadAdjustedChecks": [`), []byte(`"workloadAdjustedChecks": [
   "stale",`), 1)
		}
		if _, err := DecodePolicy(bad); err == nil {
			t.Fatalf("accepted: %s → %s", x[0], x[1])
		}
	}
	// A block statistic is the mean or the median.
	if _, err := DecodePolicy(bytes.Replace(raw, []byte(`"lateWorkStatistic": "median"`), []byte(`"lateWorkStatistic": "mode"`), 1)); err == nil {
		t.Fatal("an unknown block statistic is accepted")
	}
	// Every backlog check is adjusted or absolute, exactly once.
	dup := bytes.Replace(raw, []byte(`"absoluteChecks": [
   "stale",`), []byte(`"absoluteChecks": [
   "overdue",
   "stale",`), 1)
	if _, err := DecodePolicy(dup); err == nil || !strings.Contains(err.Error(), "exactly once") {
		t.Fatalf("a check both adjusted and absolute is accepted: %v", err)
	}
}

// The locked holdout: regenerated from its method it matches the
// manifest's digest, has its fixed size, overlaps no earlier seed set,
// has no unseal record, and every harness is refused its seeds — sealed
// or not, they run only through the canonical runner — without the
// refusal naming them.
func TestHoldoutSealed(t *testing.T) {
	h, seeds, err := LoadHoldout("../../" + HoldoutPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(seeds) != 40 || h.Count != 40 || !h.Sealed {
		t.Fatalf("%d seeds", len(seeds))
	}
	seen := map[int64]bool{}
	for _, s := range seeds {
		if s < 100_000 || (s >= 901 && s <= 910) || (s >= 1001 && s <= 1020) || seen[s] {
			t.Fatal("a seed overlaps an earlier set or repeats")
		}
		seen[s] = true
	}
	if _, err := os.Stat("../../" + RepoPaths.Unseal); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("the holdout has an unseal record: %v", err)
	}
	if h.Unseal != RepoPaths.Unseal {
		t.Fatalf("the manifest names %s", h.Unseal)
	}
	dev := []int64{901, 902, 903}
	if err := GuardSeeds(dev, h, seeds); err != nil {
		t.Fatal(err)
	}
	err = GuardSeeds(append(slices.Clone(dev), seeds[7]), h, seeds)
	if !errors.Is(err, ErrSealed) {
		t.Fatalf("a holdout seed ran: %v", err)
	}
	if strings.Contains(err.Error(), strconv.FormatInt(seeds[7], 10)) {
		t.Fatal("the refusal names a holdout seed")
	}
	// The manifest commits to the method: another label is other seeds.
	other, _ := HoldoutSeeds(h.Label+"x", h.Count, h.Lo, h.Hi, h.Excluded)
	if seedsDigest(other) == h.SeedsSHA256 {
		t.Fatal("digest does not bind the label")
	}
}

// The statistics against known values.
func TestStatistics(t *testing.T) {
	for df, want := range map[int]float64{1: 12.706, 9: 2.262, 19: 2.093, 39: 2.023, 1000: 1.962} {
		if got := tQuantile975(df); math.Abs(got-want) > 0.001 {
			t.Errorf("t(0.975, %d) = %.4f, want %.3f", df, got, want)
		}
	}
	w := wilson(4, 40)
	if math.Abs(w.Lo-0.0396) > 0.001 || math.Abs(w.Hi-0.2305) > 0.001 {
		t.Errorf("wilson %+v", w)
	}
	if r := ratio(0, 0); r.Defined {
		t.Error("0/0 defined")
	}
	if d := dist([]int64{5, 1, 4, 2, 3, 6, 7, 8, 9, 10}); d.P50 != 5 || d.P90 != 9 || d.Max != 10 {
		t.Errorf("%+v", d)
	}
}

// The repository guard every calibration harness calls: development
// seeds run; a holdout seed is refused.
func TestGuardRepo(t *testing.T) {
	_, seeds, err := LoadHoldout("../../" + HoldoutPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := GuardRepo("../..", []int64{901, 1001}); err != nil {
		t.Fatal(err)
	}
	if err := GuardRepo("../..", []int64{901, seeds[0]}); !errors.Is(err, ErrSealed) {
		t.Fatalf("%v", err)
	}
}

// A holdout evaluation's matrix: every plan on exactly the required
// seeds, every record at its pinned strategy digest, one digest a plan,
// and the primary named; other candidates are exploratory.
func TestEvaluateMatrix(t *testing.T) {
	p, d := testPolicy(t)
	mk := func(name string, pd int64) []Record {
		recs := records(10, func(s int64) synth { x := base(name, s); x.profitPD = pd; return x })
		for i := range recs {
			recs[i].PlanSHA256 = "sha-" + name
		}
		return recs
	}
	seeds := []int64{901, 902, 903, 904, 905, 906, 907, 908, 909, 910}
	pins := map[string]string{"A": "sha-A", "B": "sha-B", "C": "sha-C"}
	in := func() Input {
		return Input{Baseline: "A", Plans: map[string][]Record{"A": mk("A", 10_000), "B": mk("B", 30_000), "C": mk("C", 20_000)},
			Seeds: seeds, PlanSHA: pins, Primary: "B"}
	}
	rep := Evaluate(p, d, in())
	if len(rep.Integrity) != 0 || comparison(t, rep, "B").Role != "primary" || comparison(t, rep, "C").Role != "exploratory" {
		t.Fatalf("%v", rep.Integrity)
	}
	cases := map[string]func(x *Input){
		"seeds, not the required": func(x *Input) { x.Plans["C"] = x.Plans["C"][1:] },
		"the set pins":            func(x *Input) { x.Plans["B"][3].PlanSHA256 = "sha-B2" },
		"strategy digests":        func(x *Input) { x.PlanSHA = nil; x.Plans["B"][3].PlanSHA256 = "sha-B2" },
		"has no records":          func(x *Input) { delete(x.Plans, "C") },
		"not in the strategy set": func(x *Input) { delete(x.PlanSHA, "C") },
		"is not a candidate":      func(x *Input) { x.Primary = "A" },
	}
	for want, f := range cases {
		x := in()
		f(&x)
		rep := Evaluate(p, d, x)
		if !strings.Contains(strings.Join(rep.Integrity, "\n"), want) {
			t.Errorf("%s: %v", want, rep.Integrity)
		}
		for _, c := range rep.Comparisons {
			if c.Verdict != INVALID {
				t.Errorf("%s: %s is %s", want, c.Candidate, c.Verdict)
			}
		}
	}
}

// The repository's holdout, through the canonical path: sealed (no unseal
// record). And GitRepo answers the protocol's questions, on a scratch
// repository: committed and unchanged, changed, untracked, first author.
func TestRepositoryProtocol(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git")
	}
	if _, err := os.Stat("../../" + RepoPaths.Unseal); errors.Is(err, os.ErrNotExist) {
		if p, err := Authorize("../..", RepoPaths, GitRepo{Root: "../.."}, "any"); p != nil || !errors.Is(err, ErrSealed) {
			t.Fatalf("%v", err)
		}
	}
	dir := t.TempDir()
	g := GitRepo{Root: dir}
	run := func(args ...string) {
		t.Helper()
		if out, err := g.git(args...); err != nil {
			t.Fatalf("git %v: %s %v", args, out, err)
		}
	}
	run("init", "-q")
	os.WriteFile(filepath.Join(dir, "a.json"), []byte("{}\n"), 0o644)
	run("add", "a.json")
	run("-c", "user.name=Tester", "-c", "user.email=t@example.invalid", "commit", "-q", "-m", "a")
	if err := g.Committed("a.json"); err != nil {
		t.Fatal(err)
	}
	if a, err := g.FirstAuthor("a.json"); err != nil || a != "Tester" {
		t.Fatalf("%q %v", a, err)
	}
	os.WriteFile(filepath.Join(dir, "b.json"), []byte("{}\n"), 0o644)
	if err := g.Committed("a.json", "b.json"); err == nil {
		t.Fatal("an untracked file is committed")
	}
	if err := g.Committed("a.json", "missing.json"); err == nil {
		t.Fatal("a file that does not exist is committed")
	}
	if _, err := g.FirstAuthor("b.json"); err == nil {
		t.Fatal("an untracked file has an author")
	}
	os.WriteFile(filepath.Join(dir, "a.json"), []byte("{\"x\": 1}\n"), 0o644)
	if err := g.Committed("a.json"); err == nil {
		t.Fatal("a changed file is committed")
	}
}
