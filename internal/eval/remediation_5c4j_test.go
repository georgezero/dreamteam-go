package eval

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

// Slice 5C.4j: the remediation of the independent 5C.4i audit
// (docs/audits/slice-5c4i-boundary-safety-lock-readiness-audit.md),
// under the owner's decisions of 2026-10-19 (docs/calibration/
// clinical-5c4j/). Synthetic fixtures only; the audit's own witnesses
// (kept outside the repository) are reproduced here as regressions.

// candidate5c4hPath is the 5C.4h candidate (the partial-window count
// ceiling, no clinical follow-up), kept byte for byte: SHA-256 e41d53ba…
const candidate5c4hPath = "testdata/candidate-policy-v5-5c4h.json"

func candidate5c4hPolicy(t *testing.T) (*Policy, string) {
	t.Helper()
	return clinicalFrom(t, candidate5c4hPath)
}

// leading makes the leading partial week hold n soon acquisitions (the
// fixture's six, and n − 6 more a minute apart, each with its order) and
// the first severe of them reported 30 hours past due (severe, within the
// 48 h absolute limit).
func leading(n, severe int) func(b synth, r *Record) {
	return func(b synth, r *Record) {
		for j := 6; j < n; j++ {
			at, id := t0+int64(j)*60, fmt.Sprintf("Y%d", j)
			r.Orders = append(r.Orders, Order{ID: id, Origin: "Y/" + id, Priority: "soon", Modality: "CT", ReceivedSec: at - 1, WantedSec: at - 1,
				FirstApptSec: at, ResolvedSec: at})
			for k := range r.Captures {
				r.Captures[k].OrderOutcomes = append(r.Captures[k].OrderOutcomes, OutServed)
			}
			r.Studies = append(r.Studies, Study{ID: "S" + id, Order: id, Priority: "soon", Modality: "CT", AcquiredSec: at, EnteredSec: at,
				ClaimedSec: at + 1, ReadStartSec: at + 1, FinalSec: at + hour})
		}
		k := 0
		for j := range r.Studies {
			if s := &r.Studies[j]; s.Priority == "soon" && s.AcquiredSec > t0 && s.AcquiredSec < firstMonday {
				if k < severe {
					s.FinalSec = s.AcquiredSec + testTargets["soon"]*60 + 30*hour
				}
				k++
			}
		}
		slices.SortStableFunc(r.Studies, func(a, b Study) int { return int(a.AcquiredSec - b.AcquiredSec) })
	}
}

// I1: the observed severe-rate ceiling on every nonempty partial window
// (partialWindows rateCeiling). The leading partial week of every
// candidate seed holds n soon acquisitions, s of them severe; the full
// 40-seed comparison against a baseline without the severe reports:
//
//	5/6, 5/10, 5/50, 5/99, 6/99, 1/6: FAIL locally (above 5%) on every
//	seed, NO GO through the shared allowance (40 failing seeds);
//	5/100: judged, PASS (exactly 5%); 6/100: FAIL;
//	0/6: NOT_JUDGED, GO.
//
// The 5C.4h count ceiling gave GO for 5/6, 5/10, 5/50 and 5/99.
func TestRateCeilingFullVerdicts(t *testing.T) {
	p, d := candidatePolicy(t)
	old, od := candidate5c4hPolicy(t)
	for _, x := range []struct {
		n, s     int
		local    string
		verdict  string
		was      string // under the 5C.4h candidate
		failSeed int
	}{
		{6, 5, Fail, NOGO, GO, 40}, {10, 5, Fail, NOGO, GO, 40}, {50, 5, Fail, NOGO, GO, 40}, {99, 5, Fail, NOGO, GO, 40},
		{99, 6, Fail, NOGO, NOGO, 40}, {100, 5, Pass, GO, GO, 0}, {100, 6, Fail, NOGO, NOGO, 40},
		{6, 0, NotJudged, GO, GO, 0}, {6, 1, Fail, NOGO, GO, 40},
	} {
		pr, c := boundaryCase(t, p, d, leading(x.n, 0), leading(x.n, x.s))
		w := pr.PerSeed[0].Measures.Clinical.WeeklySevere["soon"]
		if !pr.Valid || w.Weeks[0].Acquired != int64(x.n) || w.Weeks[0].Severe != int64(x.s) || w.Weeks[0].Status != x.local ||
			c.Verdict != x.verdict || len(c.ClinicalJoint.FailingSeeds) != x.failSeed {
			t.Errorf("%d/%d: local %s, verdict %s, failing %d; want %s %s %d (%v %v)", x.s, x.n, w.Weeks[0].Status, c.Verdict,
				len(c.ClinicalJoint.FailingSeeds), x.local, x.verdict, x.failSeed, pr.Integrity, c.Reasons)
		}
		if x.local == Fail && x.n < 100 && (w.PartialRule != PartialRateCeiling || w.PartialFailing != 1) {
			t.Errorf("%d/%d: %+v", x.s, x.n, w)
		}
		if _, c := boundaryCase(t, old, od, leading(x.n, 0), leading(x.n, x.s)); c.Verdict != x.was {
			t.Errorf("%d/%d under 5C.4h: %s, want %s", x.s, x.n, c.Verdict, x.was)
		}
	}
	// The original 640-report exploit (16 of 24 on every seed): NO GO,
	// as under 5C.4h; GO under 5C.4f.
	if _, c := boundaryCase(t, p, d, leading(24, 0), leading(24, 16)); c.Verdict != NOGO {
		t.Errorf("640-report exploit: %s", c.Verdict)
	}
	o, od4f := candidate5c4fPolicy(t)
	if _, c := boundaryCase(t, o, od4f, leading(24, 0), leading(24, 16)); c.Verdict != GO {
		t.Errorf("640-report exploit under 5C.4f: %s", c.Verdict)
	}
}

// The rate ceiling's window cases: empty partial windows are NOT_JUDGED
// (never failed, never passed); several partial windows are each judged
// on their own counts, never pooled; a window reaching 100 is judged on
// its rate; complete windows below 100 stay INSUFFICIENT; with no window
// judged, the priority is INSUFFICIENT; the gate's reasons say so.
func TestRateCeilingWindows(t *testing.T) {
	p, _ := candidatePolicy(t)
	for _, x := range []struct {
		name   string
		n, sev []int
		want   string
	}{
		{"empty partial windows", []int{0, 100, 100, 0}, []int{0, 0, 0, 0}, "not_judged pass pass not_judged"},
		{"two partial windows, each 1 of 20 (5%)", []int{20, 100, 100, 20}, []int{1, 0, 0, 1}, "not_judged pass pass not_judged"},
		{"two partial windows, 1 of 19 and 0 of 19", []int{19, 100, 100, 19}, []int{1, 0, 0, 0}, "fail pass pass not_judged"},
		{"two partial windows, 5 of 6 each", []int{6, 100, 100, 6}, []int{5, 0, 0, 5}, "fail pass pass fail"},
		{"not pooled: 1 of 19 then 0 of 81 elsewhere", []int{19, 100, 100, 81}, []int{1, 0, 0, 0}, "fail pass pass not_judged"},
		{"complete 99 insufficient", []int{0, 99, 100, 0}, []int{0, 0, 0, 0}, "not_judged insufficient pass not_judged"},
		{"partial 100 judged, 6%", []int{100, 100, 100, 0}, []int{6, 0, 0, 0}, "fail pass pass not_judged"},
	} {
		ws := weeklySevereCalendar(p.Clinical, blockRecord("soon", soonStarts, x.n, x.sev), bwHorz+45*daySec)["soon"]
		if got := statuses(ws); got != x.want {
			t.Errorf("%s: %s, want %s", x.name, got, x.want)
		}
	}
	r := blockRecord("soon", []int64{bwTake}, []int{20}, []int{1})
	r.HorizonEndSec = bwTake + 20*daySec
	all := weeklySevereCalendar(p.Clinical, r, r.HorizonEndSec+45*daySec)
	ws := all["soon"]
	if !ws.NoJudgedWindow || !ws.Insufficient {
		t.Fatalf("no window judged: %+v", ws)
	}
	m := &SeedMeasures{Clinical: &ClinicalMeasures{WeeklySevere: all}}
	for _, g := range clinicalGates(p, m) {
		if g.Gate == GateSevereWeek && g.Stratum == "soon" && (g.Status != Insufficient || !strings.Contains(g.Reason, "at or below the rate (too few to establish it)")) {
			t.Fatalf("%+v", g)
		}
	}
}

// I2: the order–study contract. The audit's manipulations of a
// post-horizon violation (a soon study acquired one second after the
// horizon, reported 60 hours past due) are each INVALID under the
// candidate: a missing, unknown or ambiguous link (as in 5C.4h), and now
// a duplicated study row, a study relinked to an order wanted after the
// horizon, and a deleted study whose order stays served (each GO under
// 5C.4h). Further breaches of the contract are INVALID too. Under the
// 5C.4f candidate, without the follow-up cohort, none is checked.
func TestOrderStudyContract(t *testing.T) {
	p, d := candidatePolicy(t)
	h, hd := candidate5c4hPolicy(t)
	f, fd := candidate5c4fPolicy(t)
	for _, x := range []struct {
		kind string
		edit func(r *Record, i, si int)
		was  string // under 5C.4h
	}{
		{"missing link", func(r *Record, i, si int) { r.Studies[si].Order = "" }, INVALID},
		{"unknown order", func(r *Record, i, si int) { r.Studies[si].Order = "absent" }, INVALID},
		{"ambiguous order", func(r *Record, i, si int) {
			for j := range r.Orders {
				if j != i && r.Orders[j].Priority == "soon" {
					r.Orders[j].ID = r.Orders[i].ID
					break
				}
			}
		}, INVALID},
		{"duplicated study row", func(r *Record, i, si int) {
			s := r.Studies[si]
			s.ID += "-copy"
			r.Studies = append(r.Studies, s)
		}, NOGO},
		{"relinked to an order wanted after the horizon", func(r *Record, i, si int) {
			for _, o := range r.Orders {
				if o.Priority == "soon" && o.WantedSec > r.HorizonEndSec {
					r.Studies[si].Order = o.ID
					break
				}
			}
		}, GO},
		{"deleted study, order still served", func(r *Record, i, si int) { r.Studies = slices.Delete(r.Studies, si, si+1) }, GO},
		{"repeated study ID", func(r *Record, i, si int) {
			for j := range r.Studies {
				if j != si && r.Studies[j].AcquiredSec > t0 {
					r.Studies[si].ID = r.Studies[j].ID
					break
				}
			}
		}, NOGO},
		{"study of another modality", func(r *Record, i, si int) { r.Studies[si].Modality = "XR" }, NOGO},
		{"study acquired other than its order's imaging", func(r *Record, i, si int) { r.Orders[i].ResolvedSec++ }, NOGO},
		{"study of an order not served", func(r *Record, i, si int) {
			r.Captures[len(r.Captures)-1].OrderOutcomes[i] = OutLost
		}, NOGO},
	} {
		edit := func(b synth, r *Record) {
			i := lastSoonBefore(r)
			id := r.Orders[i].ID
			moveAcquisition(r, i, r.HorizonEndSec+1, 60*hour)
			si := slices.IndexFunc(r.Studies, func(s Study) bool { return s.Order == id })
			x.edit(r, i, si)
		}
		if _, c := boundaryCase(t, p, d, nil, edit); c.Verdict != INVALID {
			t.Errorf("%s: %s %v", x.kind, c.Verdict, c.Reasons)
		}
		if _, c := boundaryCase(t, h, hd, nil, edit); c.Verdict != x.was {
			t.Errorf("%s under 5C.4h: %s, want %s", x.kind, c.Verdict, x.was)
		}
		if _, c := boundaryCase(t, f, fd, nil, edit); c.Verdict == INVALID && !strings.Contains(x.kind, "not served") && !strings.Contains(x.kind, "imaging") {
			t.Errorf("%s under 5C.4f: %s %v", x.kind, c.Verdict, c.Reasons)
		}
	}
	// A served order received after the takeover with no study at all.
	noStudy := func(b synth, r *Record) {
		i := lastSoonBefore(r)
		id := r.Orders[i].ID
		r.Studies = slices.DeleteFunc(r.Studies, func(s Study) bool { return s.Order == id })
	}
	if _, c := boundaryCase(t, p, d, nil, noStudy); c.Verdict != INVALID {
		t.Errorf("served with no study: %s", c.Verdict)
	}
}

// The contract's exemptions: inherited work (studies acquired before the
// takeover, their orders served before or after it) is valid as the
// fixture records it, and the candidate's ordinary records are valid.
func TestOrderStudyInheritedExempt(t *testing.T) {
	p, d := candidatePolicy(t)
	in := Input{Baseline: "A", Plans: map[string][]Record{}}
	for seed := int64(1); seed <= 40; seed++ {
		in.Seeds = append(in.Seeds, seed)
		for _, plan := range []string{"A", "B"} {
			b := base(plan, seed)
			b.breachPM, b.perDay, b.inherited = map[string]int{}, boundaryVolume, 40
			in.Plans[plan] = append(in.Plans[plan], b.record())
		}
	}
	rep := Evaluate(p, d, in)
	for _, pr := range rep.Plans {
		if !pr.Valid {
			t.Fatalf("%s: %v", pr.Plan, pr.Integrity)
		}
	}
}

// I3: the 45-day follow-up (followUpDays). The candidate's records are
// observed 45 days past the horizon; the policy refuses a follow-up
// shorter than access limit + reporting target + absolute limit for any
// priority (45 days for routine: 30 + 1 + 14), or without the follow-up
// cohort. The states at the boundaries (the audit's I3 table), each
// soon or routine order wanted by the horizon, E = H + 45 days:
//
//	acquired H + 1 s, reported 60 h past due          FAIL (observed)
//	acquired before H, reported H + 1 s, 60 h past due FAIL (observed)
//	acquired H + 1 s, reported exactly 48 h past due   PASS
//	the same, one second later                         FAIL
//	soon, acquired E − 53 h, unfinished                FAIL (overdue)
//	soon, acquired E − 52 h, unfinished                INSUFFICIENT (due exactly at E)
//	soon, acquired E − 10 h, unfinished                INSUFFICIENT (not yet observable)
//	routine wanted at H, acquired at its 30-day access
//	deadline, finished exactly at E                    PASS
//	the same, unfinished at E                          INSUFFICIENT (due exactly at E)
//	routine acquired H + 30 d + 1 s, unfinished        INSUFFICIENT
//	routine acquired H + 29 d, unfinished              FAIL (overdue)
//	a report claimed one second after E                INVALID (outside the observation)
func TestFollowUp45(t *testing.T) {
	p, d := candidatePolicy(t)
	if p.Clinical.FollowUpDays != 45 {
		t.Fatalf("followUpDays %d", p.Clinical.FollowUpDays)
	}
	r := base("B", 1).record()
	if observationNeeded(p, &r) != r.HorizonEndSec+45*daySec || r.ObservedUntilSec != r.HorizonEndSec+45*daySec {
		t.Fatalf("observation %d days past the horizon", (observationNeeded(p, &r)-r.HorizonEndSec)/daySec)
	}
	for name, edit := range map[string]func(c *Clinical){
		"44 days":            func(c *Clinical) { c.FollowUpDays = 44 },
		"without the cohort": func(c *Clinical) { c.SafetyCohort = "" },
		"negative":           func(c *Clinical) { c.FollowUpDays = -1 },
		"routine access 31 d": func(c *Clinical) {
			c.AccessLimitMin = map[string]int64{"urgent": 2880, "soon": 10080, "routine": 31 * 1440}
		},
	} {
		q, _ := candidatePolicy(t)
		c := *q.Clinical
		edit(&c)
		q.Clinical = &c
		if err := q.validate(); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	// last is the last order of priority q wanted by the bound (and by
	// the horizon), imaged by the horizon.
	last := func(r *Record, q string, bound int64) int {
		k := -1
		for i, o := range r.Orders {
			if o.Priority == q && o.WantedSec <= min(bound, r.HorizonEndSec) && o.ResolvedSec != 0 && o.ResolvedSec <= r.HorizonEndSec {
				k = i
			}
		}
		return k
	}
	for _, x := range []struct {
		name      string
		q         string
		at        func(r *Record) int64
		past      int64 // −1: unfinished; reported this long after acquisition otherwise
		wantedAtH bool
		safety    string
	}{
		{"acquired H + 1 s", "soon", func(r *Record) int64 { return r.HorizonEndSec + 1 }, 4*hour + 60*hour, false, Fail},
		{"reported H + 1 s", "soon", func(r *Record) int64 { return r.HorizonEndSec - 64*hour + 1 }, 64 * hour, false, Fail},
		{"exactly at the limit", "soon", func(r *Record) int64 { return r.HorizonEndSec + 1 }, 4*hour + 48*hour, false, Pass},
		{"a second beyond", "soon", func(r *Record) int64 { return r.HorizonEndSec + 1 }, 4*hour + 48*hour + 1, false, Fail},
		{"overdue", "soon", func(r *Record) int64 { return r.ObservedUntilSec - 53*hour }, -1, false, Fail},
		{"due exactly at E", "soon", func(r *Record) int64 { return r.ObservedUntilSec - 52*hour }, -1, false, Insufficient},
		{"not yet observable", "soon", func(r *Record) int64 { return r.ObservedUntilSec - 10*hour }, -1, false, Insufficient},
		{"routine at its access deadline, finished at E", "routine", func(r *Record) int64 { return r.HorizonEndSec + 30*daySec }, 15 * daySec, true, Pass},
		{"routine at its access deadline, unfinished", "routine", func(r *Record) int64 { return r.HorizonEndSec + 30*daySec }, -1, true, Insufficient},
		{"routine a second later, unfinished", "routine", func(r *Record) int64 { return r.HorizonEndSec + 30*daySec + 1 }, -1, true, Insufficient},
		{"routine H + 29 d, unfinished", "routine", func(r *Record) int64 { return r.HorizonEndSec + 29*daySec }, -1, true, Fail},
	} {
		// The baseline asks for the same (its order wanted at H too,
		// imaged then): the demand stays paired.
		var same func(b synth, r *Record)
		if x.wantedAtH {
			same = func(b synth, r *Record) {
				i := last(r, x.q, x.at(r))
				r.Orders[i].WantedSec = r.HorizonEndSec
				moveAcquisition(r, i, r.HorizonEndSec, 0)
			}
		}
		pr, c := boundaryCase(t, p, d, same, func(b synth, r *Record) {
			i := last(r, x.q, x.at(r))
			if x.wantedAtH {
				r.Orders[i].WantedSec = r.HorizonEndSec
			}
			past := x.past
			if past >= 0 {
				past -= testTargets[x.q] * 60 // moveAcquisition counts from the due time
			}
			moveAcquisition(r, i, x.at(r), past)
		})
		if !pr.Valid || pr.ClinicalSafety.Status != x.safety || favourable(c) != (x.safety == Pass) {
			t.Errorf("%s: valid %v, safety %s, %s %v %v", x.name, pr.Valid, pr.ClinicalSafety.Status, c.Verdict, pr.Integrity, c.Reasons)
		}
	}
	pr, _ := boundaryCase(t, p, d, nil, func(b synth, r *Record) {
		i := last(r, "routine", r.HorizonEndSec)
		moveAcquisition(r, i, r.HorizonEndSec+30*daySec, 15*daySec+1-testTargets["routine"]*60)
	})
	if pr.Valid {
		t.Error("a report after the end of observation accepted")
	}
}

// The order–study contract on one record, each breach alone: the
// candidate's Measure reports it (the record not measured clinically),
// and the unbroken record passes.
func TestOrderStudyContractUnit(t *testing.T) {
	p, _ := candidatePolicy(t)
	h, _ := candidate5c4hPolicy(t)
	mk := func() *Record {
		b := base("B", 3)
		b.perDay = boundaryVolume
		r := b.record()
		return &r
	}
	soonAfterT0 := func(r *Record) (int, int) {
		for si, s := range r.Studies {
			if s.Priority == "soon" && s.AcquiredSec > t0+daySec {
				i := slices.IndexFunc(r.Orders, func(o Order) bool { return o.ID == s.Order })
				return i, si
			}
		}
		panic("no soon study")
	}
	if m := Measure(p, mk()); len(m.Integrity) != 0 || m.Clinical == nil {
		t.Fatalf("unbroken: %v", m.Integrity)
	}
	for want, edit := range map[string]func(r *Record){
		"its ID repeats": func(r *Record) {
			_, si := soonAfterT0(r)
			r.Studies[si].ID = r.Studies[si+1].ID
		},
		"not exactly one": func(r *Record) { _, si := soonAfterT0(r); r.Studies[si].Order = "nowhere" },
		"asked for":       func(r *Record) { _, si := soonAfterT0(r); r.Studies[si].Modality = "XR" },
		"not served": func(r *Record) {
			i, _ := soonAfterT0(r)
			r.Captures[len(r.Captures)-1].OrderOutcomes[i] = OutLost
		},
		"its order": func(r *Record) { i, _ := soonAfterT0(r); r.Orders[i].ResolvedSec++ },
		"more than one study": func(r *Record) {
			_, si := soonAfterT0(r)
			s := r.Studies[si]
			s.ID += "-copy"
			r.Studies = append(r.Studies, s)
		},
		"served with no study": func(r *Record) { _, si := soonAfterT0(r); r.Studies = slices.Delete(r.Studies, si, si+1) },
	} {
		r := mk()
		edit(r)
		m := Measure(p, r)
		if m.Clinical != nil || !strings.Contains(strings.Join(m.Integrity, "\n"), want) {
			t.Errorf("%s: %v", want, m.Integrity)
		}
		// The 5C.4h candidate checks only post-horizon links: these pass it.
		if want != "not exactly one" {
			r := mk()
			edit(r)
			if m := Measure(h, r); m.Clinical == nil && !strings.Contains(strings.Join(m.Integrity, "\n"), "imaged at") {
				t.Errorf("%s under 5C.4h: %v", want, m.Integrity)
			}
		}
	}
}
