package eval

import (
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"strings"
	"testing"
)

// Slice 5C.4h: clinical boundary safety (docs/calibration/clinical-5c4h/
// boundary-safety.md). The audit's partial-window and horizon witnesses
// (docs/audits/slice-5c4g-final-clinical-policy-audit.md G4), each
// favourable under the 5C.4f candidate and not under the revision; the
// allowance interaction (G5); the reporting repairs (G6). Synthetic
// fixtures only. The partial-window witnesses here are the 5C.4h count
// ceiling's, judged under the 5C.4h candidate; the 5C.4j rate ceiling's
// are in remediation_5c4j_test.go.

// candidate5c4fPath is the 5C.4f candidate (no count ceiling, the
// absolute limits over studies acquired by the horizon), kept byte for
// byte: SHA-256 d901fe58…
const candidate5c4fPath = "testdata/candidate-policy-v5-5c4f.json"

func candidate5c4fPolicy(t *testing.T) (*Policy, string) {
	t.Helper()
	return clinicalFrom(t, candidate5c4fPath)
}

// boundaryVolume: enough soon and routine studies for every complete
// window to reach the candidate's minimum of 100 (168 a soon block, 105 a
// routine week).
var boundaryVolume = map[string]int{"urgent": 1, "soon": 6, "routine": 15}

// boundaryCase evaluates a no-action baseline A and a candidate B on 40
// seeds at boundaryVolume, with no reading breach; changeA and changeB
// edit each seed's record.
func boundaryCase(t *testing.T, p *Policy, d string, changeA, changeB func(b synth, r *Record)) (*PlanReport, Comparison) {
	t.Helper()
	in := Input{Baseline: "A", Plans: map[string][]Record{}}
	for seed := int64(1); seed <= 40; seed++ {
		in.Seeds = append(in.Seeds, seed)
		for _, x := range []struct {
			plan   string
			change func(b synth, r *Record)
		}{{"A", changeA}, {"B", changeB}} {
			b := base(x.plan, seed)
			b.breachPM, b.perDay = map[string]int{}, boundaryVolume
			if x.plan == "B" {
				b.profitPD = 30_000
			}
			r := b.record()
			if x.change != nil {
				x.change(b, &r)
				resync(b, &r)
			}
			in.Plans[x.plan] = append(in.Plans[x.plan], r)
		}
	}
	rep := Evaluate(p, d, in)
	return plan(rep, "B"), comparison(t, rep, "B")
}

// lateReports makes the first n studies of priority q acquired in
// [from, to) finish `past` seconds after their due time.
func lateReports(q string, from, to int64, n int, past int64) func(b synth, r *Record) {
	return func(b synth, r *Record) {
		k := 0
		for i := range r.Studies {
			s := &r.Studies[i]
			if s.Priority == q && s.AcquiredSec >= from && s.AcquiredSec < to && k < n {
				s.FinalSec = s.AcquiredSec + testTargets[q]*60 + past
				k++
			}
		}
	}
}

// firstMonday is the synthetic takeover's anchor, the first evaluator
// Monday after t0 (the evaluator's weeks run on absolute time: the
// leading partial week is t0's last 22 hours).
var firstMonday = (floorDiv(t0/daySec+3, 7)*7 + 4) * daySec

// addSoon adds n soon orders to a record, received, wanted and imaged
// in the leading partial week (every 30 minutes from t0 + 10 minutes),
// each with its study, read within the hour.
func addSoon(n int) func(b synth, r *Record) {
	return func(b synth, r *Record) {
		for j := range n {
			at := t0 + 10*60 + int64(j)*1800
			id := fmt.Sprintf("X%d", j)
			r.Orders = append(r.Orders, Order{ID: id, Origin: "X/" + id, Priority: "soon", Modality: "CT", ReceivedSec: at - 60, WantedSec: at - 60,
				FirstApptSec: at, ResolvedSec: at})
			for i := range r.Captures {
				r.Captures[i].OrderOutcomes = append(r.Captures[i].OrderOutcomes, OutServed)
			}
			r.Studies = append(r.Studies, Study{ID: "X" + id, Order: id, Priority: "soon", Modality: "CT", AcquiredSec: at, EnteredSec: at,
				ClaimedSec: at + 60, ReadStartSec: at + 60, FinalSec: at + hour})
		}
		slices.SortStableFunc(r.Studies, func(x, y Study) int { return int(x.AcquiredSec - y.AcquiredSec) })
	}
}

// both applies changes in order.
func both(fs ...func(b synth, r *Record)) func(b synth, r *Record) {
	return func(b synth, r *Record) {
		for _, f := range fs {
			f(b, r)
		}
	}
}

// G4.1 reproduced: both plans have 24 soon acquisitions in the leading
// partial week (the fixture's 6 and 18 added); on every candidate seed 16
// of them are reported 30 hours past due (severe, within the 48 h
// absolute limit): 640 severe reports over 40 seeds, two thirds of the
// partial weeks. The per-seed soon rate is 16 of 558 in T90 (2.9%), every
// complete block passes. Under the 5C.4f candidate the partial week is NOT_JUDGED
// and the verdict GO; under the count ceiling (at most 5 severe, a
// compliant 100-study window's) it fails on every seed: NO GO.
func TestPartialWindowConcentration(t *testing.T) {
	harm := both(addSoon(18), lateReports("soon", t0, firstMonday, 16, 30*hour))
	old, od := candidate5c4fPolicy(t)
	pr, c := boundaryCase(t, old, od, addSoon(18), harm)
	ws := pr.PerSeed[0].Measures.Clinical.WeeklySevere["soon"]
	if w := ws.Weeks[0]; w.Complete || w.Acquired != 24 || w.Severe != 16 || w.Status != NotJudged || ws.Failing != 0 || c.Verdict != GO {
		t.Fatalf("5C.4f: %+v %s %v", w, c.Verdict, c.Reasons)
	}
	severe := 0
	for _, s := range pr.PerSeed {
		severe += int(s.Measures.Clinical.WeeklySevere["soon"].Weeks[0].Severe)
	}
	if severe != 640 {
		t.Fatalf("%d severe in the partial weeks", severe)
	}
	p, d := candidate5c4hPolicy(t)
	pr, c = boundaryCase(t, p, d, addSoon(18), harm)
	ws = pr.PerSeed[0].Measures.Clinical.WeeklySevere["soon"]
	if w := ws.Weeks[0]; w.Status != Fail || ws.PartialFailing != 1 || ws.PartialCeiling == nil || *ws.PartialCeiling != 5 || ws.Failing != 1 {
		t.Fatalf("5C.4h: %+v %+v", w, ws)
	}
	if len(c.ClinicalJoint.FailingSeeds) != 40 || favourable(c) || !hasReason(c, "clinical seed reliability: fail") {
		t.Fatalf("5C.4h: %s %v", c.Verdict, c.Reasons)
	}
	for _, g := range pr.PerSeed[0].ClinicalGates {
		if g.Gate == GateSevereWeek && g.Stratum == "soon" && (g.Status != Fail || !strings.Contains(g.Reason, "more than the 5 severe a minimum window may hold")) {
			t.Fatalf("gate: %+v", g)
		}
	}
}

// The count ceiling exactly: a partial window below the minimum holds at
// most ⌊5% × 100⌋ = 5 severe reports, whatever its size (5 of 99 not
// judged, 6 of 99 and 6 of 24 failing); a partial window reaching the
// minimum is judged on its rate (5 of 100 passes, 6 of 100 fails);
// complete windows are unchanged (99 insufficient, 6 of 100 failing).
// The 5C.4f candidate leaves every partial window below 100 not judged.
func TestPartialWindowCeiling(t *testing.T) {
	p, _ := candidate5c4hPolicy(t)
	old, _ := candidate5c4fPolicy(t)
	for _, x := range []struct {
		name     string
		n, sev   []int
		now, was string
	}{
		{"5 of 99 leading", []int{99, 100, 100, 0}, []int{5, 0, 0, 0}, "not_judged pass pass not_judged", "not_judged pass pass not_judged"},
		{"6 of 99 leading", []int{99, 100, 100, 0}, []int{6, 0, 0, 0}, "fail pass pass not_judged", "not_judged pass pass not_judged"},
		{"6 of 24 leading", []int{24, 100, 100, 0}, []int{6, 0, 0, 0}, "fail pass pass not_judged", "not_judged pass pass not_judged"},
		{"6 of 50 trailing", []int{0, 100, 100, 50}, []int{0, 0, 0, 6}, "not_judged pass pass fail", "not_judged pass pass not_judged"},
		{"5 of 100 partial, judged", []int{100, 100, 100, 0}, []int{5, 0, 0, 0}, "pass pass pass not_judged", "pass pass pass not_judged"},
		{"6 of 100 partial, judged", []int{100, 100, 100, 0}, []int{6, 0, 0, 0}, "fail pass pass not_judged", "fail pass pass not_judged"},
		{"complete 99, insufficient", []int{0, 99, 100, 0}, []int{0, 0, 0, 0}, "not_judged insufficient pass not_judged", "not_judged insufficient pass not_judged"},
		{"complete 6 of 100", []int{0, 100, 100, 0}, []int{0, 6, 0, 0}, "not_judged fail pass not_judged", "not_judged fail pass not_judged"},
	} {
		r := blockRecord("soon", soonStarts, x.n, x.sev)
		now := weeklySevereCalendar(p.Clinical, r, bwHorz+40*daySec)["soon"]
		was := weeklySevereCalendar(old.Clinical, r, bwHorz+40*daySec)["soon"]
		if statuses(now) != x.now || statuses(was) != x.was || was.PartialCeiling != nil {
			t.Errorf("%s: now %s, 5C.4f %s", x.name, statuses(now), statuses(was))
		}
	}
	// Routine weeks: the leading partial week (Wednesday to Sunday) holds
	// at most 5 severe too.
	r := blockRecord("routine", []int64{bwTake, bwAnchor}, []int{60, 100}, []int{6, 0})
	if ws := weeklySevereCalendar(p.Clinical, r, bwHorz+40*daySec)["routine"]; ws.Weeks[0].Status != Fail || ws.PartialFailing != 1 {
		t.Fatalf("routine: %s", statuses(ws))
	}
}

// No vacuous pass: a horizon with no complete soon block and partial
// windows within the ceiling judges nothing, and is insufficient under
// the revision (a pass under 5C.4f).
func TestPartialWindowNoJudgedWindow(t *testing.T) {
	r := blockRecord("soon", []int64{bwTake}, []int{30}, []int{2})
	r.HorizonEndSec = bwTake + 20*daySec // the leading week and part of one block
	p, _ := candidate5c4hPolicy(t)
	ws := weeklySevereCalendar(p.Clinical, r, r.HorizonEndSec+40*daySec)["soon"]
	if ws.Blocks != 0 || !ws.NoJudgedWindow || !ws.Insufficient || ws.InsufficientWeeks != 0 || statuses(ws) != "not_judged not_judged" {
		t.Fatalf("%s %+v", statuses(ws), ws)
	}
	old, _ := candidate5c4fPolicy(t)
	if ws := weeklySevereCalendar(old.Clinical, r, r.HorizonEndSec+40*daySec)["soon"]; ws.Insufficient || ws.NoJudgedWindow {
		t.Fatalf("5C.4f: %+v", ws)
	}
	// The seed gate states it.
	m := &SeedMeasures{Clinical: &ClinicalMeasures{WeeklySevere: map[string]*WeeklySevere{"soon": ws, "routine": ws}}}
	m.Clinical.WeeklySevere["soon"] = weeklySevereCalendar(p.Clinical, r, r.HorizonEndSec+40*daySec)["soon"]
	for _, g := range clinicalGates(p, m) {
		if g.Gate == GateSevereWeek && g.Stratum == "soon" && (g.Status != Insufficient || !strings.Contains(g.Reason, "no window judged")) {
			t.Fatalf("%+v", g)
		}
	}
}

// The passing gate discloses the partial windows it did not judge.
func TestPartialWindowDisclosed(t *testing.T) {
	p, d := candidate5c4hPolicy(t)
	pr, c := boundaryCase(t, p, d, nil, nil)
	if c.Verdict != GO {
		t.Fatalf("%s %v", c.Verdict, c.Reasons)
	}
	for _, g := range pr.PerSeed[0].ClinicalGates {
		if g.Gate == GateSevereWeek && (g.Status != Pass || !strings.Contains(g.Reason, "partial windows below the minimum not judged, each within the count ceiling of 5 severe")) {
			t.Fatalf("%+v", g)
		}
	}
}

// lastSoonBefore is the index of the last soon order wanted by the
// horizon, imaged by the horizon.
func lastSoonBefore(r *Record) int {
	k := -1
	for i, o := range r.Orders {
		if o.Priority == "soon" && o.WantedSec <= r.HorizonEndSec && o.ResolvedSec != 0 && o.ResolvedSec <= r.HorizonEndSec {
			k = i
		}
	}
	return k
}

// moveAcquisition re-times order i's study to acquisition at, finished
// `past` seconds after its due time (past < 0: unfinished).
func moveAcquisition(r *Record, i int, at, past int64) {
	o := &r.Orders[i]
	for j := range r.Studies {
		s := &r.Studies[j]
		if s.Order != o.ID {
			continue
		}
		s.AcquiredSec, s.EnteredSec, s.ClaimedSec, s.ReadStartSec = at, at, at+60, at+60
		s.FinalSec = 0
		if past >= 0 {
			s.FinalSec = at + testTargets[s.Priority]*60 + past
		}
	}
	o.ResolvedSec, o.FirstApptSec = at, at-10*60
	slices.SortStableFunc(r.Studies, func(x, y Study) int { return int(x.AcquiredSec - y.AcquiredSec) })
}

// G4.2 reproduced: on every seed one soon order wanted the day before the
// horizon is imaged within its seven-day access limit and reported 60
// hours past due (12 hours beyond the 48 h absolute limit). Acquired at
// the horizon H: clinical safety fails under both candidates. At H + 1
// second: the 5C.4f candidate's cohort (studies acquired by H) drops it,
// GO; the revision keeps it (its order was wanted by H): NO GO, with the
// violation observed in the follow-up.
func TestHorizonCrossingAcquisition(t *testing.T) {
	at := func(dt int64) func(b synth, r *Record) {
		return func(b synth, r *Record) { moveAcquisition(r, lastSoonBefore(r), r.HorizonEndSec+dt, 60*hour) }
	}
	old, od := candidate5c4fPolicy(t)
	p, d := candidatePolicy(t)
	for _, x := range []struct {
		name   string
		p      *Policy
		d      string
		dt     int64
		favour bool
	}{
		{"5C.4f, at H", old, od, 0, false},
		{"5C.4f, H + 1 s", old, od, 1, true},
		{"5C.4h, at H", p, d, 0, false},
		{"5C.4h, H + 1 s", p, d, 1, false},
	} {
		pr, c := boundaryCase(t, x.p, x.d, nil, at(x.dt))
		if favourable(c) != x.favour {
			t.Errorf("%s: %s %v", x.name, c.Verdict, c.Reasons)
		}
		if x.favour {
			continue
		}
		if pr.ClinicalSafety.Status != Fail || len(pr.ClinicalSafety.Seeds) != 40 || !hasReason(c, "clinical safety: seed 1, 1 soon studies or orders beyond the absolute pastDue limit of 2880 min (worst 3600)") {
			t.Errorf("%s: %+v %v", x.name, pr.ClinicalSafety.Status, c.Reasons)
		}
		if a := seedPastDue(pr, "soon"); x.p == p && (a.SafetyCohort != SafetyOrdersByHorizon || a.Observed != 1 || a.Overdue != 0 || a.PostHorizon != x.dt) {
			t.Errorf("%s: %+v", x.name, a)
		}
	}
}

// seedPastDue is seed 1's absolute past-due count of priority q.
func seedPastDue(pr *PlanReport, q string) AbsoluteCount {
	for _, a := range pr.PerSeed[0].Measures.Clinical.Absolute {
		if a.Limit == "pastDue" && a.Priority == q {
			return a
		}
	}
	return AbsoluteCount{}
}

// Reporting across the horizon: a study acquired an hour before H and
// reported after it, 60 hours past due, is in both candidates' cohorts
// (acquired by H) and fails both.
func TestHorizonCrossingReporting(t *testing.T) {
	cross := func(b synth, r *Record) { moveAcquisition(r, lastSoonBefore(r), r.HorizonEndSec-hour, 60*hour) }
	old, od := candidate5c4fPolicy(t)
	p, d := candidatePolicy(t)
	for _, x := range []struct {
		p *Policy
		d string
	}{{old, od}, {p, d}} {
		pr, c := boundaryCase(t, x.p, x.d, nil, cross)
		if favourable(c) || pr.ClinicalSafety.Status != Fail {
			t.Errorf("%s %v", c.Verdict, c.Reasons)
		}
	}
}

// The follow-up states, each from the study's own due time (acquisition
// plus the soon target, 4 h), against the 48 h limit at the end of
// observation E, for a soon order wanted by H and imaged after it:
//
//   - unfinished, acquired 60 h before E (56 h past due): an unresolved
//     overdue obligation, a violation: NO GO;
//   - unfinished, acquired 10 h before E (6 h past due): not yet
//     observable, safety insufficient: NO GO, never a pass;
//   - finished exactly at the limit: compliant; one second beyond:
//     observed violation.
//
// The 5C.4f candidate drops all four (acquired after H): GO.
func TestHorizonFollowUpStates(t *testing.T) {
	type state struct{ overdue, pending, observed int64 }
	old, od := candidate5c4fPolicy(t)
	p, d := candidatePolicy(t)
	for _, x := range []struct {
		name   string
		before int64 // acquisition, before E
		past   int64 // −1: unfinished
		safety string
		want   state
	}{
		{"unresolved overdue", 60 * hour, -1, Fail, state{1, 0, 0}},
		{"not yet observable", 10 * hour, -1, Insufficient, state{0, 1, 0}},
		{"at the limit", 60 * hour, 48 * hour, Pass, state{}},
		{"one second beyond", 60 * hour, 48*hour + 1, Fail, state{0, 0, 1}},
	} {
		change := func(b synth, r *Record) {
			moveAcquisition(r, lastSoonBefore(r), r.ObservedUntilSec-x.before, x.past)
		}
		pr, c := boundaryCase(t, p, d, nil, change)
		a := seedPastDue(pr, "soon")
		if pr.ClinicalSafety.Status != x.safety || (state{a.Overdue, a.Pending, a.Observed}) != x.want || a.PostHorizon != 1 || favourable(c) != (x.safety == Pass) {
			t.Errorf("%s: safety %s, %+v, %s %v", x.name, pr.ClinicalSafety.Status, a, c.Verdict, c.Reasons)
		}
		if x.safety == Insufficient && (len(pr.ClinicalSafety.Pending) != 40 || !hasReason(c, "not yet observable against the absolute pastDue limit")) {
			t.Errorf("%s: %+v %v", x.name, pr.ClinicalSafety, c.Reasons)
		}
		if _, c := boundaryCase(t, old, od, nil, change); !favourable(c) {
			t.Errorf("%s under 5C.4f: %s %v", x.name, c.Verdict, c.Reasons)
		}
	}
}

// An order wanted by H still open at E with no study: no report is due,
// so it is counted (unacquired) and judged by Gate A (untimely: E is 31
// days past H), never made a safety pass or failure. An order wanted
// after H, imaged after it, stays outside the cohort however late its
// report.
func TestHorizonUnacquiredAndLaterOrders(t *testing.T) {
	p, d := candidatePolicy(t)
	open := func(b synth, r *Record) {
		i := lastSoonBefore(r)
		resolveAs(r, i, OutOpen, 0)
	}
	pr, c := boundaryCase(t, p, d, nil, open)
	if a := seedPastDue(pr, "soon"); a.Unacquired != 1 || pr.ClinicalSafety.Status != Pass || c.Verdict != GO {
		t.Fatalf("unacquired: %+v %s %v", a, c.Verdict, c.Reasons)
	}
	acc := pr.PerSeed[0].Measures.Clinical.Access[WinGated]["soon"]
	pr0, _ := boundaryCase(t, p, d, nil, nil)
	if acc0 := pr0.PerSeed[0].Measures.Clinical.Access[WinGated]["soon"]; acc.Untimely != acc0.Untimely+1 {
		t.Fatalf("Gate A: %d untimely, %d without", acc.Untimely, acc0.Untimely)
	}
	later := func(b synth, r *Record) {
		for i := range r.Studies {
			s := &r.Studies[i]
			if s.Priority == "soon" && s.AcquiredSec > r.HorizonEndSec+2*daySec && s.FinalSec != 0 {
				s.FinalSec = s.AcquiredSec + testTargets["soon"]*60 + 60*hour
				break
			}
		}
	}
	if pr, c := boundaryCase(t, p, d, nil, later); pr.ClinicalSafety.Status != Pass || c.Verdict != GO {
		t.Fatalf("later order: %+v %s %v", pr.ClinicalSafety, c.Verdict, c.Reasons)
	}
}

// A study acquired after the horizon must name exactly one order of its
// priority: missing, unknown, ambiguous or of another priority, the
// record is INVALID (the cohort cannot be decided), never a pass. The
// 5C.4f candidate needs no link.
func TestHorizonLinkIntegrity(t *testing.T) {
	p, d := candidatePolicy(t)
	old, od := candidate5c4fPolicy(t)
	post := func(r *Record) *Study {
		for i := range r.Studies {
			if r.Studies[i].AcquiredSec > r.HorizonEndSec {
				return &r.Studies[i]
			}
		}
		return nil
	}
	for name, change := range map[string]func(r *Record){
		"missing": func(r *Record) { post(r).Order = "" },
		"unknown": func(r *Record) { post(r).Order = "nowhere" },
		"ambiguous": func(r *Record) {
			s := post(r)
			for i := range r.Orders {
				if r.Orders[i].ID != s.Order {
					r.Orders[i].ID = s.Order
					break
				}
			}
		},
		"other priority": func(r *Record) {
			s := post(r)
			for _, o := range r.Orders {
				if o.Priority != s.Priority {
					s.Order = o.ID
					break
				}
			}
		},
	} {
		edit := func(b synth, r *Record) {
			if r.Seed == 3 {
				change(r)
			}
		}
		if _, c := boundaryCase(t, p, d, nil, edit); c.Verdict != INVALID {
			t.Errorf("%s: %s %v", name, c.Verdict, c.Reasons)
		}
		if name == "missing" {
			if _, c := boundaryCase(t, old, od, nil, edit); c.Verdict != GO {
				t.Errorf("5C.4f %s: %s %v", name, c.Verdict, c.Reasons)
			}
		}
	}
}

// The absolute limits keep their override: one soon report beyond 48 h on
// one seed, in a complete block, is NO GO whatever the allowance.
func TestBoundaryAbsoluteOverride(t *testing.T) {
	p, d := candidatePolicy(t)
	one := lateReports("soon", firstMonday+7*daySec, firstMonday+8*daySec, 1, 48*hour+1)
	pr, c := boundaryCase(t, p, d, nil, func(b synth, r *Record) {
		if r.Seed == 9 {
			one(b, r)
		}
	})
	if pr.ClinicalSafety.Status != Fail || !slices.Equal(pr.ClinicalSafety.Seeds, []int64{9}) || pr.ClinicalJoint.Status != Pass || c.Verdict != NOGO {
		t.Fatalf("%+v %+v %s", pr.ClinicalSafety, pr.ClinicalJoint, c.Verdict)
	}
}

// G5: the allowances are separate. Seeds 1–4 lose 9 of their 150 urgent
// orders (6 points: under the 10-point safeguard, over the v4 per-seed
// served-drop limit of 5 points, which has its own allowance of four);
// seeds 5–8 fail a soon block (10 of 168 severe). The clinical allowance
// holds [5 6 7 8], case selection [1 2 3 4]: eight distinct seeds with
// adverse outcomes, and the verdict GO. On the same four seeds both
// failures take one place in each: GO. A fifth clinical seed: NO GO.
// Gate B's mean stays outside both: with 4 lost on every seed (2.67
// points) NO GO, every seed within the safeguard.
func TestAllowanceInteraction(t *testing.T) {
	p, d := candidatePolicy(t)
	lose9 := loseUrgent(func(int64) int { return 9 })
	block := lateReports("soon", firstMonday+28*daySec, firstMonday+56*daySec, 10, 30*hour)
	run := func(caseSel, clinical func(seed int64) bool) (*PlanReport, Comparison) {
		return boundaryCase(t, p, d, nil, func(b synth, r *Record) {
			if caseSel(r.Seed) {
				lose9(b, r)
			}
			if clinical(r.Seed) {
				block(b, r)
			}
		})
	}
	in := func(lo, hi int64) func(int64) bool { return func(s int64) bool { return s >= lo && s <= hi } }
	pr, c := run(in(1, 4), in(5, 8))
	g := lostDemandOf(c, "urgent")
	if !slices.Equal(c.ClinicalJoint.FailingSeeds, []int64{5, 6, 7, 8}) || len(g.Deteriorated) != 0 || g.Status != Pass || c.Verdict != GO {
		t.Fatalf("disjoint: %+v %+v %s %v", c.ClinicalJoint, g, c.Verdict, c.Reasons)
	}
	if n, st := caseSelectionFailing(c, "urgent"); n != 4 || st != Pass {
		t.Fatalf("disjoint: case selection %d failing, %s", n, st)
	}
	_ = pr
	if _, c := run(in(1, 4), in(1, 4)); !slices.Equal(c.ClinicalJoint.FailingSeeds, []int64{1, 2, 3, 4}) || c.Verdict != GO {
		t.Fatalf("overlapping: %+v %s %v", c.ClinicalJoint, c.Verdict, c.Reasons)
	}
	if _, c := run(in(1, 4), in(5, 9)); c.Verdict != NOGO || !hasReason(c, "clinical seed reliability: fail") {
		t.Fatalf("fifth: %s %v", c.Verdict, c.Reasons)
	}
	_, c = boundaryCase(t, p, d, nil, loseUrgent(func(int64) int { return 4 }))
	if g := lostDemandOf(c, "urgent"); g.Status != Fail || len(g.Deteriorated) != 0 || c.ClinicalJoint.Status != Pass || c.Verdict != NOGO {
		t.Fatalf("mean: %+v %s %v", g, c.Verdict, c.Reasons)
	}
}

// caseSelectionFailing is the comparison's seeds failing the v4 per-seed
// served-drop rule of priority q, and the case-selection status.
func caseSelectionFailing(c Comparison, q string) (int, string) {
	n := 0
	for _, g := range c.CaseSelection {
		if g.Gate == "served_fraction_drop_per_seed" && g.Stratum == KindPriority+":"+q {
			n = max(n, g.Failing)
		}
	}
	return n, c.CaseSelectionStatus
}

// G6.1: occupied weeks. 100 acquisitions in the first 50 hours of a
// complete four-week block occupy one calendar week of its four.
func TestOccupiedWeeks(t *testing.T) {
	p, _ := candidatePolicy(t)
	ws := weeklySevereCalendar(p.Clinical, blockRecord("soon", soonStarts, []int{0, 100, 100, 0}, []int{0, 6, 0, 0}), bwHorz+40*daySec)["soon"]
	if ws.WorstWeeks != 4 || ws.WorstOccupiedWeeks != 1 || ws.Weeks[1].Occupied != 1 || ws.Weeks[1].Weeks != 4 {
		t.Fatalf("%+v", ws)
	}
	// Spread over the block: every week occupied.
	r := blockRecord("soon", []int64{bwAnchor, bwAnchor + 7*daySec, bwAnchor + 14*daySec, bwAnchor + 21*daySec}, []int{25, 25, 25, 25}, []int{6, 0, 0, 0})
	if ws := weeklySevereCalendar(p.Clinical, r, bwHorz+40*daySec)["soon"]; ws.WorstOccupiedWeeks != 4 || ws.Weeks[1].Occupied != 4 {
		t.Fatalf("spread: %+v", ws.Weeks[1])
	}
}

// G6.2: with no seed paired, the absolute unimaged fractions are
// unavailable (null), never a measured zero.
func TestUnavailableFractions(t *testing.T) {
	p, d := candidatePolicy(t)
	_, c := clinicalCaseAB(t, p, d, nil, func(b synth, r *Record) {
		for i := range r.Orders {
			r.Orders[i].Origin = ""
		}
	})
	g := lostDemandOf(c, "urgent")
	raw, _ := json.Marshal(g)
	if g.Status != Insufficient || g.SeedFraction != nil || g.PooledFraction != nil || !strings.Contains(string(raw), `"unimagedFractionSeedMean":null`) {
		t.Fatalf("%s %s", g.Status, raw)
	}
	for _, a := range c.Access {
		if a.Unimaged != nil || a.PooledUnimaged != nil {
			t.Fatalf("%+v", a)
		}
	}
}

// T7's raw count: a seed failing timely access and lacking evidence on a
// soon block is FAIL in the joint rule, and still listed by gate as
// insufficient; so is a seed whose soon gate fails on another block
// (audit 5C.4i I6: the gate's own failure must not hide the window's
// missing evidence). The second soon block is thinned below 100 by moving
// 69 of its acquisitions, with their orders, into the third.
func TestRawInsufficiencyByGate(t *testing.T) {
	p, d := candidatePolicy(t)
	thin := func(r *Record) {
		var ids []string
		for _, s := range r.Studies {
			if s.Priority == "soon" && s.AcquiredSec >= firstMonday+28*daySec && s.AcquiredSec < firstMonday+56*daySec && len(ids) < 69 {
				ids = append(ids, s.Order)
			}
		}
		for _, id := range ids {
			j := slices.IndexFunc(r.Orders, func(o Order) bool { return o.ID == id })
			moveAcquisition(r, j, firstMonday+56*daySec+hour, 0)
		}
	}
	pr, _ := boundaryCase(t, p, d, nil, func(b synth, r *Record) {
		switch r.Seed {
		case 6:
			thin(r)
			delayOrders("urgent", 0, 30, 10*daySec, 30)(b, r)
		case 7:
			thin(r)
			lateReports("soon", firstMonday+56*daySec, firstMonday+84*daySec, 13, 30*hour)(b, r)
		}
	})
	key := GateSevereWeek + " " + WinGated + " soon"
	ws := pr.PerSeed[6].Measures.Clinical.WeeklySevere["soon"]
	if !pr.Valid || !slices.Contains(pr.ClinicalJoint.FailingSeeds, 6) || !slices.Contains(pr.ClinicalJoint.FailingSeeds, 7) ||
		ws.InsufficientWeeks != 1 || ws.Failing != 1 || gateStatus(pr, 7, GateSevereWeek, WinGated, "soon") != Fail ||
		!slices.Equal(pr.ClinicalInsufficient[key], []int64{6, 7}) {
		t.Fatalf("%v %+v %v", pr.Integrity, pr.ClinicalJoint, pr.ClinicalInsufficient)
	}
}

// Validation of the boundary rules.
func TestBoundaryValidation(t *testing.T) {
	for name, edit := range map[string]func(c *Clinical){
		"unknown partial rule": func(c *Clinical) { c.PartialWindows = "pool" },
		"ceiling without calendar": func(c *Clinical) {
			c.PartialWindows, c.WeeklyPartition, c.WeeklyWindowWeeks = PartialCountCeiling, "", nil
		},
		"unknown safety cohort":         func(c *Clinical) { c.SafetyCohort = "studies" },
		"safety cohort with wrong case": func(c *Clinical) { c.SafetyCohort = "OrdersByHorizon" },
	} {
		p, _ := candidatePolicy(t)
		c := *p.Clinical
		edit(&c)
		p.Clinical = &c
		if err := p.validate(); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

// G7: the all-zero bound is not the smallest bound over every sample.
// 40 seed differences of −0.5 give a betting bound below the 2-point
// margin (−0.2926); 40 zeros give 0.2074. The 5C.4d rule's design check
// (insufficient when the all-zero bound exceeds the margin) is kept.
func TestGateBStrongImprovementBound(t *testing.T) {
	neg := make([]float64, 40)
	for i := range neg {
		neg[i] = -0.5
	}
	up, zero := upperBound(BoundBetting, neg, 0.95, -1, 1), zeroFloor(BoundBetting, 40, 0.95, -1, 1)
	if math.Abs(up-(-0.292615588928)) > 1e-6 || math.Abs(zero-0.207384411072) > 1e-6 || exceeds(up, 0.02) || !exceeds(zero, 0.02) {
		t.Fatalf("bound %.12f, all-zero %.12f", up, zero)
	}
}

// The 5C.4f candidate's exact bytes still load and keep its rules: no
// count ceiling (partial windows below the minimum not judged, a vacuous
// pass possible), the absolute limits over studies acquired by the
// horizon, no follow-up states in its reports.
func TestOld5C4FCompatibility(t *testing.T) {
	_, digest, err := LoadPolicy(candidate5c4fPath)
	if err != nil || digest != "d901fe5877684567c6c2d19ad1c063e8cf36ce269ba6fd44ba54299f052e348b" {
		t.Fatalf("%v %s", err, digest)
	}
	p, d := candidate5c4fPolicy(t)
	if p.Clinical.PartialWindows != "" || p.Clinical.SafetyCohort != "" {
		t.Fatalf("%+v", p.Clinical)
	}
	pr, c := boundaryCase(t, p, d, nil, nil)
	raw, _ := json.Marshal(pr.PerSeed[0].Measures.Clinical)
	for _, f := range []string{`"partialSevereCeiling"`, `"safetyCohort"`, `"pending"`, `"postHorizon"`, `"observed"`} {
		if strings.Contains(string(raw), f) {
			t.Fatalf("%s in a 5C.4f result", f)
		}
	}
	if c.Verdict != GO {
		t.Fatalf("%s %v", c.Verdict, c.Reasons)
	}
}
