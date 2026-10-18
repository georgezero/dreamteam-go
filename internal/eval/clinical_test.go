package eval

import (
	"slices"
	"strings"
	"testing"
)

// v0.2-eval-5's clinical acceptance on synthetic records: the acceptance
// test plan of docs/evaluation/proposals/v5-clinical-acceptance.md §G
// (G1–G6, G10–G14; the synthetic queues G7–G9 and G15 are the combined
// operating characteristics). The limits here are test values chosen
// only to sit on either side of each case, not proposals: the owner sets
// the frozen ones.

// v5Policy is the policy in force made a v0.2-eval-5 candidate with test
// limits.
func v5Policy(t *testing.T) (*Policy, string) {
	t.Helper()
	p, d := testPolicy(t)
	p.Schema = PolicyV5Schema
	p.Backlog.InformationalChecks = append(slices.Clone(p.Backlog.InformationalChecks), "persistent")
	day := int64(1440)
	p.Clinical = &Clinical{
		AbsoluteMaxPastDueMin:  map[string]int64{"urgent": 5 * day, "soon": 7 * day, "routine": 14 * day},
		SevereAfterDueMin:      map[string]int64{"urgent": day, "soon": day, "routine": day},
		MaxSevereRate:          map[string]float64{"urgent": 0.1, "soon": 0.1, "routine": 0.05},
		MaxSeverePerWeek:       map[string]int64{"urgent": 3, "soon": 5, "routine": 5},
		AccessLimitMin:         map[string]int64{"urgent": 2 * day, "soon": 10 * day, "routine": 30 * day},
		MinTimelyFraction:      map[string]float64{"urgent": 0.9, "soon": 0.9, "routine": 0.9},
		MinAccessCohort:        map[string]int64{"urgent": 43, "soon": 88, "routine": 23},
		MaxWaitMin:             map[string]int64{"urgent": 7 * day, "soon": 20 * day, "routine": 31 * day},
		InheritedFrom:          "takeover",
		MaxFailingSeedFraction: 0.1,
	}
	if err := p.validate(); err != nil {
		t.Fatal(err)
	}
	return p, d
}

// resync recomputes a modified record's daily samples and capture
// outcomes from its studies and orders, as a recorder would.
func resync(b synth, r *Record) {
	for i := range r.Samples {
		s := r.Samples[i]
		r.Samples[i] = b.sample(r, s.Day, s.AtSec, b.money(s.AtSec))
	}
	for i := range r.Captures {
		c := &r.Captures[i]
		end := r.Captures[len(r.Captures)-1].OrderOutcomes
		for j, o := range r.Orders {
			switch {
			case o.ReceivedSec > c.AtSec:
				c.OrderOutcomes[j] = OutNotDisclosed
			case o.ResolvedSec == 0 || o.ResolvedSec > c.AtSec:
				c.OrderOutcomes[j] = OutOpen
			case end[j] != OutOpen && end[j] != OutNotDisclosed:
				c.OrderOutcomes[j] = end[j]
			default:
				c.OrderOutcomes[j] = OutServed
			}
		}
	}
}

// clinicalCase evaluates a no-action baseline A and a candidate B on 40
// seeds; change edits B's record of each seed (after it is built).
func clinicalCase(t *testing.T, p *Policy, d string, mkB func(seed int64) synth, change func(b synth, r *Record)) (*Report, *PlanReport, Comparison) {
	t.Helper()
	in := Input{Baseline: "A", Plans: map[string][]Record{}}
	for seed := int64(1); seed <= 40; seed++ {
		in.Seeds = append(in.Seeds, seed)
		a := base("A", seed)
		a.breachPM = map[string]int{}
		in.Plans["A"] = append(in.Plans["A"], a.record())
		b := base("B", seed)
		b.breachPM, b.profitPD = map[string]int{}, 30_000
		if mkB != nil {
			b = mkB(seed)
		}
		r := b.record()
		if change != nil {
			change(b, &r)
			resync(b, &r)
		}
		in.Plans["B"] = append(in.Plans["B"], r)
	}
	rep := Evaluate(p, d, in)
	return rep, plan(rep, "B"), comparison(t, rep, "B")
}

// studiesOf are a record's study indices of priority q acquired on days
// [from, to) after the takeover.
func studiesOf(r *Record, q string, from, to int64) []int {
	var out []int
	for i, s := range r.Studies {
		if d := (s.AcquiredSec - t0) / daySec; s.Priority == q && s.AcquiredSec > t0 && d >= from && d < to {
			out = append(out, i)
		}
	}
	return out
}

// finishAt finalizes study i at fin.
func finishAt(r *Record, i int, fin int64) {
	s := &r.Studies[i]
	s.FinalSec, s.ClaimedSec, s.ReadStartSec = fin, fin-60, fin-60
}

func gateStatus(pr *PlanReport, seed int64, gate, w, q string) string {
	for _, s := range pr.PerSeed {
		if s.Seed != seed {
			continue
		}
		for _, g := range s.ClinicalGates {
			if g.Gate == gate && g.Window == w && g.Stratum == q {
				return g.Status
			}
		}
	}
	return "missing"
}

func wantVerdict(t *testing.T, pr *PlanReport, c Comparison, verdict string, reason string) {
	t.Helper()
	if !pr.Valid || c.Verdict != verdict || (reason != "" && !slices.ContainsFunc(c.Reasons, func(s string) bool { return strings.Contains(s, reason) })) {
		t.Fatalf("verdict %s (want %s with %q): %v %v", c.Verdict, verdict, reason, pr.Integrity, c.Reasons)
	}
}

// G1, G1a: six urgent reports eight days late in one week (the audit's
// witness) fail the absolute urgent limit and the week's concentration;
// six two hours past due are ordinary lateness.
func TestClinicalSevereReportDelay(t *testing.T) {
	p, d := v5Policy(t)
	late := func(by int64) func(b synth, r *Record) {
		return func(b synth, r *Record) {
			for _, i := range studiesOf(r, "urgent", 112, 118) {
				finishAt(r, i, r.Studies[i].AcquiredSec+testTargets["urgent"]*60+by)
			}
		}
	}
	_, pr, c := clinicalCase(t, p, d, nil, late(8*daySec))
	wantVerdict(t, pr, c, NOGO, "clinical safety")
	if gateStatus(pr, 1, GateSevereWeek, WinGated, "urgent") != Fail || pr.ClinicalSafety.Status != Fail || len(pr.ClinicalSafety.Seeds) != 40 {
		t.Fatalf("week %s, safety %+v", gateStatus(pr, 1, GateSevereWeek, WinGated, "urgent"), pr.ClinicalSafety.Status)
	}
	// Under v0.2-eval-4 the same records pass (the audit's H1).
	q, qd := testPolicy(t)
	_, pr4, c4 := clinicalCase(t, q, qd, nil, late(8*daySec))
	wantVerdict(t, pr4, c4, GO, "")
	_, pr, c = clinicalCase(t, p, d, nil, late(2*hour))
	wantVerdict(t, pr, c, GO, "")
}

// G2, G2a, G2b: the age limit from exact timestamps. A report 348 h past
// due is seen at 334 h by the daily samples, and passes v0.2-eval-4; v5
// rejects it. 335 h passes. An unfinished study is never dropped: due 20
// days before the horizon and unfinished at the end, it violates.
func TestClinicalAgeLimitExact(t *testing.T) {
	p, d := v5Policy(t)
	routine := func(by int64) func(b synth, r *Record) {
		return func(b synth, r *Record) {
			i := studiesOf(r, "routine", 140, 141)[0]
			finishAt(r, i, r.Studies[i].AcquiredSec+testTargets["routine"]*60+by)
		}
	}
	_, pr, c := clinicalCase(t, p, d, nil, routine(348*hour))
	wantVerdict(t, pr, c, NOGO, "absolute pastDue limit of 20160 min (worst 20880)")
	if v := pr.ClinicalSafety.Violations[0]; v.Priority != "routine" || v.Violations != 1 {
		t.Fatalf("%+v", v)
	}
	q, qd := testPolicy(t)
	_, pr4, _ := clinicalCase(t, q, qd, nil, routine(348*hour))
	if bm := pr4.PerSeed[0].Measures.Backlog; bm.Persistent != 0 || pr4.BacklogStatus != Pass {
		t.Fatalf("v0.2-eval-4 sees it: %+v", bm.Persistent)
	}
	_, pr, c = clinicalCase(t, p, d, nil, routine(335*hour))
	wantVerdict(t, pr, c, GO, "")
	// Exactly at the limit passes; a minute over fails.
	_, pr, c = clinicalCase(t, p, d, nil, routine(14*daySec))
	wantVerdict(t, pr, c, GO, "")
	_, pr, c = clinicalCase(t, p, d, nil, routine(14*daySec+60))
	wantVerdict(t, pr, c, NOGO, "clinical safety")
	_, pr, c = clinicalCase(t, p, d, nil, func(b synth, r *Record) {
		i := studiesOf(r, "urgent", 130, 131)[0]
		s := &r.Studies[i]
		s.FinalSec = 0
	})
	wantVerdict(t, pr, c, NOGO, "clinical safety")
	if v := pr.ClinicalSafety.Violations[0]; v.Priority != "urgent" || v.WorstMin < 50*1440 {
		t.Fatalf("censored: %+v", v)
	}
}

// delayOrders moves the urgent (or q) orders wanted on days [from, to)
// and their studies by later (their wanted time stays).
func delayOrders(q string, from, to, later int64, limit int) func(b synth, r *Record) {
	return func(b synth, r *Record) {
		n := 0
		for i := range r.Studies {
			s := &r.Studies[i]
			if dd := (s.AcquiredSec - t0) / daySec; s.Priority != q || s.AcquiredSec <= t0 || dd < from || dd >= to || n >= limit {
				continue
			}
			for j := range r.Orders {
				o := &r.Orders[j]
				if o.Priority == q && o.ResolvedSec == s.AcquiredSec {
					o.FirstApptSec += later
					o.ResolvedSec += later
				}
			}
			s.AcquiredSec += later
			s.EnteredSec += later
			s.ClaimedSec += later
			s.ReadStartSec += later
			if s.FinalSec != 0 {
				s.FinalSec += later
			}
			n++
		}
		slices.SortStableFunc(r.Studies, func(x, y Study) int { return int(x.AcquiredSec - y.AcquiredSec) })
	}
}

// G3, G3a: thirty urgent acquisitions thirty days after their wanted time
// (the audit's witness) fail timely access on every seed; two hours later
// they are timely.
func TestClinicalDelayedAccess(t *testing.T) {
	p, d := v5Policy(t)
	_, pr, c := clinicalCase(t, p, d, nil, delayOrders("urgent", 0, 30, 30*daySec, 30))
	wantVerdict(t, pr, c, NOGO, "clinical seed reliability: fail (40 failing")
	if gateStatus(pr, 1, GateTimelyAccess, WinGated, "urgent") != Fail || gateStatus(pr, 1, GateMaxWait, WinGated, "urgent") != Fail {
		t.Fatal("access gates")
	}
	q, qd := testPolicy(t)
	_, pr4, c4 := clinicalCase(t, q, qd, nil, delayOrders("urgent", 0, 30, 30*daySec, 30))
	wantVerdict(t, pr4, c4, GO, "")
	_, pr, c = clinicalCase(t, p, d, nil, delayOrders("urgent", 0, 30, 2*hour, 30))
	wantVerdict(t, pr, c, GO, "")
}

// G4, G4a: no survivorship. Urgent orders never booked stay open and keep
// waiting: untimely, and beyond the maximum wait. Lost urgent orders are
// untimely.
func TestClinicalUnresolvedAndLostOrders(t *testing.T) {
	p, d := v5Policy(t)
	_, pr, c := clinicalCase(t, p, d, nil, func(b synth, r *Record) {
		for k := range 5 {
			at := t0 + int64(20+k)*daySec + 5*hour
			r.Orders = append(r.Orders, Order{ID: "U", Priority: "urgent", Modality: "MR", ReceivedSec: at, WantedSec: at})
			for i := range r.Captures {
				r.Captures[i].OrderOutcomes = append(r.Captures[i].OrderOutcomes, OutOpen)
			}
		}
	})
	wantVerdict(t, pr, c, NOGO, "clinical seed reliability")
	a := pr.PerSeed[0].Measures.Clinical.Access[WinGated]["urgent"]
	if a.Untimely != 5 || a.Undecided != 0 || gateStatus(pr, 1, GateMaxWait, WinGated, "urgent") != Fail || gateStatus(pr, 1, GateTimelyAccess, WinGated, "urgent") != Pass {
		t.Fatalf("open orders: %+v", a)
	}
	_, pr, _ = clinicalCase(t, p, d, func(seed int64) synth {
		b := base("B", seed)
		b.breachPM, b.profitPD, b.declinePM = map[string]int{}, 30_000, map[string]int{"urgent": 150}
		return b
	}, nil)
	if a := pr.PerSeed[0].Measures.Clinical.Access[WinGated]["urgent"]; a.Untimely == 0 || gateStatus(pr, 1, GateTimelyAccess, WinGated, "urgent") != Fail {
		t.Fatalf("lost orders: %+v", a)
	}
}

// G4b: a patient who cancels before the limit leaves the denominator; one
// who cancels after it was already failed.
func TestClinicalPatientCensoring(t *testing.T) {
	var a AccessCohort
	for _, at := range []int64{1800, 3600, 3 * 3600} {
		o := Order{WantedSec: 0, ResolvedSec: at}
		classifyAccess(&a, &o, OutPatient, 0, 2*3600, 100*3600)
	}
	o := Order{}
	classifyAccess(&a, &o, OutOpen, 0, 2*3600, 1*3600)
	classifyAccess(&a, &o, OutLeft, 0, 2*3600, 100*3600)
	if a.Withdrawn != 2 || a.Untimely != 2 || a.Undecided != 1 || a.Wanted != 5 {
		t.Fatalf("%+v", a)
	}
}

// G5, G5a: six severe soon delays in one week fail the week's
// concentration although their rate is low; spread one a week, they pass.
func TestClinicalConcentratedDelays(t *testing.T) {
	p, d := v5Policy(t)
	severe := func(days []int64) func(b synth, r *Record) {
		return func(b synth, r *Record) {
			used := map[int64]int{}
			for _, dd := range days {
				i := studiesOf(r, "soon", dd, dd+1)[used[dd]]
				used[dd]++
				finishAt(r, i, r.Studies[i].AcquiredSec+testTargets["soon"]*60+2*daySec)
			}
		}
	}
	_, pr, c := clinicalCase(t, p, d, nil, severe([]int64{50, 50, 51, 51, 52, 52}))
	wantVerdict(t, pr, c, NOGO, "clinical seed reliability")
	if gateStatus(pr, 1, GateSevereWeek, WinGated, "soon") != Fail || gateStatus(pr, 1, GateSevereRate, WinT90, "soon") != Pass {
		t.Fatal("concentration")
	}
	_, pr, c = clinicalCase(t, p, d, nil, severe([]int64{8, 22, 36, 50, 64, 78}))
	wantVerdict(t, pr, c, GO, "")
	// Exactly a day past due is not severe.
	_, pr, c = clinicalCase(t, p, d, nil, func(b synth, r *Record) {
		for _, i := range studiesOf(r, "soon", 50, 53) {
			finishAt(r, i, r.Studies[i].AcquiredSec+testTargets["soon"]*60+daySec)
		}
	})
	wantVerdict(t, pr, c, GO, "")
}

// G6, G6a: access deteriorating in the tail fails the tail window; a dip
// in the first month that recovers within T90 passes.
func TestClinicalAccessDeterioration(t *testing.T) {
	p, d := v5Policy(t)
	_, pr, c := clinicalCase(t, p, d, nil, delayOrders("soon", 100, 150, 11*daySec, 30))
	wantVerdict(t, pr, c, NOGO, "clinical seed reliability")
	if gateStatus(pr, 1, GateTimelyAccess, WinTail, "soon") != Fail || gateStatus(pr, 1, GateTimelyAccess, WinT90, "soon") != Pass {
		t.Fatal("tail")
	}
	_, pr, c = clinicalCase(t, p, d, nil, delayOrders("soon", 5, 15, 11*daySec, 15))
	wantVerdict(t, pr, c, GO, "")
}

// G10: an order imaged before its wanted time makes the record invalid
// (v5 only; never a pass or a fail).
func TestClinicalContradictoryTimestamps(t *testing.T) {
	p, d := v5Policy(t)
	_, pr, c := clinicalCase(t, p, d, nil, func(b synth, r *Record) {
		for i := range r.Orders {
			if r.Orders[i].ResolvedSec > t0+daySec && r.Orders[i].ResolvedSec != 0 {
				r.Orders[i].WantedSec = r.Orders[i].ResolvedSec + hour
				r.Orders[i].ReceivedSec = r.Orders[i].ResolvedSec - hour
				break
			}
		}
	})
	if pr.Valid || c.Verdict != INVALID || !slices.ContainsFunc(pr.Integrity, func(s string) bool { return strings.Contains(s, "before its wanted time") }) {
		t.Fatalf("%v %s", pr.Integrity, c.Verdict)
	}
}

// staleBlock leaves eight routine studies of the block from day 111 more
// than a day but less than fourteen past due, four at a time, so the
// block's median stale count is four (above the cap of three), while
// their severe rate (8 of the tail's routine studies) stays within its
// limit: backlog alone fails.
func staleBlock(b synth, r *Record) {
	for _, from := range []int64{111, 124} {
		for _, i := range studiesOf(r, "routine", from, from+2)[:4] {
			finishAt(r, i, r.Studies[i].AcquiredSec+testTargets["routine"]*60+13*daySec)
		}
	}
}

// G11, G11a: one allowance. Four seeds failing service and four others
// failing backlog pass v0.2-eval-4's separate allowances and fail v5's
// shared one (8 > 4); the same four seeds failing both pass (each counts
// once).
func TestClinicalSharedAllowance(t *testing.T) {
	p, d := v5Policy(t)
	q, qd := testPolicy(t)
	mk := func(svc []int64) func(seed int64) synth {
		return func(seed int64) synth {
			b := base("B", seed)
			b.breachPM, b.profitPD = map[string]int{}, 30_000
			if slices.Contains(svc, seed) {
				b.breachPM = map[string]int{"urgent": 200}
			}
			return b
		}
	}
	stale := func(seeds []int64) func(b synth, r *Record) {
		return func(b synth, r *Record) {
			if slices.Contains(seeds, r.Seed) {
				staleBlock(b, r)
			}
		}
	}
	svc, blg := []int64{1, 2, 3, 4}, []int64{5, 6, 7, 8}
	_, pr4, _ := clinicalCase(t, q, qd, mk(svc), stale(blg))
	if pr4.ServiceJoint.Status != Pass || pr4.BacklogStatus != Pass || pr4.BacklogStable != 36 {
		t.Fatalf("v4: service %s, backlog %s (%d stable)", pr4.ServiceJoint.Status, pr4.BacklogStatus, pr4.BacklogStable)
	}
	_, pr, c := clinicalCase(t, p, d, mk(svc), stale(blg))
	wantVerdict(t, pr, c, NOGO, "clinical seed reliability: fail (8 failing")
	for _, s := range pr.PerSeed {
		if slices.Contains(blg, s.Seed) && (s.Backlog != "unstable" || gateStatus(pr, s.Seed, GateSevereRate, WinTail, "routine") != Pass) {
			t.Fatalf("seed %d: backlog %s, severe %s: not backlog alone", s.Seed, s.Backlog, gateStatus(pr, s.Seed, GateSevereRate, WinTail, "routine"))
		}
	}
	_, pr, c = clinicalCase(t, p, d, mk(svc), stale(svc))
	if pr.ClinicalJoint.Status != Pass || len(pr.ClinicalJoint.FailingSeeds) != 4 {
		t.Fatalf("same seeds: %+v %v", pr.ClinicalJoint, c.Reasons)
	}
}

// G12: one absolute violation on one seed, everything else passing: NO GO
// for clinical safety, the seed allowance untouched.
func TestClinicalAbsoluteApartFromAllowance(t *testing.T) {
	p, d := v5Policy(t)
	_, pr, c := clinicalCase(t, p, d, nil, func(b synth, r *Record) {
		if r.Seed == 7 {
			i := studiesOf(r, "routine", 60, 61)[0]
			finishAt(r, i, r.Studies[i].AcquiredSec+testTargets["routine"]*60+15*daySec)
		}
	})
	wantVerdict(t, pr, c, NOGO, "clinical safety: seed 7")
	if pr.Eligible || pr.ClinicalJoint.Status != Pass || len(pr.ClinicalJoint.FailingSeeds) != 0 || !slices.Equal(pr.ClinicalSafety.Seeds, []int64{7}) {
		t.Fatalf("%+v %+v", pr.ClinicalJoint, pr.ClinicalSafety)
	}
}

// G13: inherited work timed from the takeover (the default): a study 20
// days past due at the takeover and finished on day 2 passes; one left
// fifteen more days fails. Timed from its own due time, the first fails.
func TestClinicalInheritedTiming(t *testing.T) {
	inherit := func(finDay int64) func(b synth, r *Record) {
		return func(b synth, r *Record) {
			acq := t0 - 21*daySec
			r.Studies = append([]Study{{ID: "I", Priority: "routine", Modality: "XR", AcquiredSec: acq, EnteredSec: acq,
				ClaimedSec: t0 + finDay*daySec - 60, ReadStartSec: t0 + finDay*daySec - 60, FinalSec: t0 + finDay*daySec}}, r.Studies...)
		}
	}
	p, d := v5Policy(t)
	_, pr, c := clinicalCase(t, p, d, nil, inherit(2))
	wantVerdict(t, pr, c, GO, "")
	_, pr, c = clinicalCase(t, p, d, nil, inherit(15))
	wantVerdict(t, pr, c, NOGO, "clinical safety")
	p.Clinical.InheritedFrom = "own"
	_, pr, c = clinicalCase(t, p, d, nil, inherit(2))
	wantVerdict(t, pr, c, NOGO, "clinical safety")
}

// G14 (audit M3): under v5 a baseline whose service only lacks evidence
// is not restored; v0.2-eval-4's classification is kept as frozen.
func TestClinicalInsufficientBaselineNotRestored(t *testing.T) {
	p, _ := v5Policy(t)
	safe := &Safety{Status: Pass}
	cand := &PlanReport{Valid: true, ServiceStatus: Pass, ServiceJoint: JointGate{Status: Pass}, BacklogStatus: Pass, SolvencyStatus: Pass,
		FundingStatus: Pass, ClinicalStatus: Pass, ClinicalJoint: &JointGate{Status: Pass}, ClinicalSafety: safe}
	baseline := &PlanReport{Valid: true, ServiceStatus: Insufficient}
	c := Comparison{Valid: true, Profit: map[string]Interval{WinT90: {N: 40, Mean: -100}, WinOps: {N: 40, Mean: -10}}}
	verdict(p, &c, cand, baseline)
	if c.ServiceRestored || c.Classification == ClassServicePurchase || c.Verdict != NOGO {
		t.Fatalf("%+v", c)
	}
	baseline.ServiceStatus = Fail
	c = Comparison{Valid: true, Profit: map[string]Interval{WinT90: {N: 40, Mean: -100}, WinOps: {N: 40, Mean: -10}}}
	verdict(p, &c, cand, baseline)
	if !c.ServiceRestored || c.Classification != ClassServicePurchase {
		t.Fatalf("%+v", c)
	}
}

// The v5 policy's own checks.
func TestClinicalPolicyValidation(t *testing.T) {
	for name, change := range map[string]func(p *Policy){
		"v5 without clinical rules":     func(p *Policy) { p.Clinical = nil },
		"v4 with clinical rules":        func(p *Policy) { p.Schema = PolicySchema },
		"sampled age limit still gated": func(p *Policy) { p.Backlog.InformationalChecks = []string{"mean", "max"} },
		"urgent allowed longer":         func(p *Policy) { p.Clinical.AbsoluteMaxPastDueMin["urgent"] = 8 * 1440 },
		"limit beyond the observation":  func(p *Policy) { p.Clinical.MaxWaitMin["routine"] = 32 * 1440 },
		"severe after the absolute":     func(p *Policy) { p.Clinical.SevereAfterDueMin["urgent"] = 6 * 1440 },
		"timely after the maximum wait": func(p *Policy) { p.Clinical.AccessLimitMin["soon"] = 21 * 1440 },
		"no inherited timing":           func(p *Policy) { p.Clinical.InheritedFrom = "" },
		"fraction not whole bp":         func(p *Policy) { p.Clinical.MinTimelyFraction["soon"] = 0.12345 },
		"no access cohort":              func(p *Policy) { p.Clinical.MinAccessCohort["routine"] = 0 },
	} {
		p, _ := v5Policy(t)
		change(p)
		if err := p.validate(); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	p, _ := testPolicy(t)
	p.Backlog.InformationalChecks = append(p.Backlog.InformationalChecks, "persistent")
	if err := p.validate(); err == nil {
		t.Error("v4 with the age limit informational: accepted")
	}
}
