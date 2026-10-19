package eval

import (
	"maps"
	"slices"
	"strings"
	"testing"
)

// Slice 5C.4c's adversarial cases for the two access gates (timeliness
// over served, open and late-cancelled orders; lost demand against the
// paired baseline) and the volume-normalized weekly severity, under the
// candidate policy (candidate-policy-v5.json), on the synthetic records:
// 40 seeds, one urgent order a day.

// clinicalCaseAB is clinicalCase with the baseline's records changed too.
func clinicalCaseAB(t *testing.T, p *Policy, d string, changeA, changeB func(b synth, r *Record)) (*PlanReport, Comparison) {
	t.Helper()
	in := Input{Baseline: "A", Plans: map[string][]Record{}}
	for seed := int64(1); seed <= 40; seed++ {
		in.Seeds = append(in.Seeds, seed)
		for _, x := range []struct {
			plan   string
			change func(b synth, r *Record)
		}{{"A", changeA}, {"B", changeB}} {
			b := base(x.plan, seed)
			b.breachPM = map[string]int{}
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

func lostDemandOf(c Comparison, q string) LostDemandGate {
	for _, g := range c.LostDemand {
		if g.Priority == q {
			return g
		}
	}
	return LostDemandGate{Status: "missing"}
}

func hasReason(c Comparison, sub string) bool {
	return slices.ContainsFunc(c.Reasons, func(s string) bool { return strings.Contains(s, sub) })
}

// C1: lost urgent orders. Thirty urgent orders a seed lost where the
// baseline images them: counted apart from timeliness (timely access
// passes), the unimaged fraction rises 0.20: lost demand fails, NO GO.
// Three lost a seed rise exactly the margin (0.02): every discordant order
// one way, the bound is the increase itself: pass (the margin is
// inclusive); four: fail. Four lost a seed where the baseline loses one
// other: the increase is the margin again, but with discordance both ways
// its one-sided upper bound exceeds it: fail. None lost: pass.
func TestGateLostUrgentOrders(t *testing.T) {
	p, d := candidatePolicy(t)
	lose := func(n int) func(b synth, r *Record) {
		return func(b synth, r *Record) {
			for _, i := range urgentOrders(r, 20, 50)[:n] {
				resolveAs(r, i, OutLost, r.Orders[i].ReceivedSec)
			}
		}
	}
	pr, c := clinicalCaseAB(t, p, d, nil, lose(30))
	g := lostDemandOf(c, "urgent")
	if gateStatus(pr, 1, GateTimelyAccess, WinGated, "urgent") != Pass || g.Status != Fail || g.Discordant != [2]int64{1200, 0} ||
		g.Increase < 0.199 || c.Verdict != NOGO || !hasReason(c, "lost demand urgent: fail") {
		t.Fatalf("30 lost: %+v, verdict %s %v", g, c.Verdict, c.Reasons)
	}
	if lostDemandOf(c, "soon").Status != Pass || lostDemandOf(c, "routine").Status != Pass {
		t.Fatal("other priorities")
	}
	_, c = clinicalCaseAB(t, p, d, nil, lose(3))
	if g := lostDemandOf(c, "urgent"); g.Status != Pass || g.Increase != 0.02 || g.IncreaseUpper != 0.02 {
		t.Fatalf("3 lost: %+v", g)
	}
	_, c = clinicalCaseAB(t, p, d, nil, lose(4))
	if g := lostDemandOf(c, "urgent"); g.Status != Fail {
		t.Fatalf("4 lost: %+v", g)
	}
	_, c = clinicalCaseAB(t, p, d, func(b synth, r *Record) {
		i := urgentOrders(r, 30, 31)[0]
		resolveAs(r, i, OutLost, r.Orders[i].ReceivedSec)
	}, lose(4))
	if g := lostDemandOf(c, "urgent"); g.Status != Fail || g.Discordant != [2]int64{160, 40} || g.Increase > 0.0201 || g.IncreaseUpper <= 0.021 {
		t.Fatalf("4 against 1: %+v", g)
	}
	_, c = clinicalCaseAB(t, p, d, nil, nil)
	if g := lostDemandOf(c, "urgent"); g.Status != Pass || g.IncreaseUpper != 0 || g.MatchedSeeds != 40 || g.Orders != 40*150 {
		t.Fatalf("none lost: %+v", g)
	}
}

// C2: delayed urgent acquisition. Thirty urgent orders a seed imaged ten
// days late: timely 0.8 < 0.95 fails timeliness on every seed (NO GO,
// clinical seed reliability); nothing is unimaged, so lost demand passes.
func TestGateDelayedUrgentAcquisition(t *testing.T) {
	p, d := candidatePolicy(t)
	pr, c := clinicalCaseAB(t, p, d, nil, delayOrders("urgent", 0, 30, 10*daySec, 30))
	if gateStatus(pr, 1, GateTimelyAccess, WinGated, "urgent") != Fail || lostDemandOf(c, "urgent").Status != Pass ||
		!hasReason(c, "clinical seed reliability") {
		t.Fatalf("%s %v", c.Verdict, c.Reasons)
	}
}

// C3: more loss bought with better timeliness. The baseline images 15
// urgent orders a seed three days late; the candidate images every one it
// keeps on time but loses 15 others: its timeliness (1.0) is better than
// the baseline's (0.90) and passes, the lost-demand gate fails: the gain
// does not offset the loss.
func TestGateLossNotOffsetByTimeliness(t *testing.T) {
	p, d := candidatePolicy(t)
	pr, c := clinicalCaseAB(t, p, d, delayOrders("urgent", 60, 75, 3*daySec, 15), func(b synth, r *Record) {
		for _, i := range urgentOrders(r, 80, 95) {
			resolveAs(r, i, OutLost, r.Orders[i].ReceivedSec)
		}
	})
	a := urgentAccess(pr)
	if a.Untimely != 0 || a.Lost != 15 || gateStatus(pr, 1, GateTimelyAccess, WinGated, "urgent") != Pass {
		t.Fatalf("timeliness: %+v", a)
	}
	if g := lostDemandOf(c, "urgent"); g.Status != Fail || c.Verdict != NOGO {
		t.Fatalf("%+v %s", g, c.Verdict)
	}
}

// C4: cancellation manipulation. Thirty urgent orders a seed ended by the
// patient within an hour of their wanted time (withdrawn from timeliness)
// where the baseline images them: still unimaged demand, lost demand
// fails. A patient outcome is not a way out of the denominator.
func TestGateCancellationManipulation(t *testing.T) {
	p, d := candidatePolicy(t)
	pr, c := clinicalCaseAB(t, p, d, nil, func(b synth, r *Record) {
		for _, i := range urgentOrders(r, 20, 50) {
			resolveAs(r, i, OutPatient, r.Orders[i].WantedSec+hour)
		}
	})
	g := lostDemandOf(c, "urgent")
	if urgentAccess(pr).Withdrawn != 30 || gateStatus(pr, 1, GateTimelyAccess, WinGated, "urgent") != Pass || g.Status != Fail || g.Candidate.Patient != 1200 {
		t.Fatalf("%+v", g)
	}
}

// C5: rebooking. Thirty urgent orders a seed displaced twice: imaged
// within 40 hours of the original wanted time, timely (both gates pass);
// imaged five days after it, untimely (a rebooking never resets the
// clock), timeliness fails.
func TestGateRebooking(t *testing.T) {
	p, d := candidatePolicy(t)
	rebook := func(served int64) func(b synth, r *Record) {
		return func(b synth, r *Record) {
			for _, i := range urgentOrders(r, 20, 50) {
				o := &r.Orders[i]
				o.DisplacementSec = []int64{o.WantedSec + hour, o.WantedSec + 2*hour}
				resolveAs(r, i, OutServed, o.WantedSec+served)
			}
		}
	}
	pr, c := clinicalCaseAB(t, p, d, nil, rebook(40*hour))
	if gateStatus(pr, 1, GateTimelyAccess, WinGated, "urgent") != Pass || lostDemandOf(c, "urgent").Status != Pass {
		t.Fatalf("within 40 h: %v", c.Reasons)
	}
	pr, _ = clinicalCaseAB(t, p, d, nil, rebook(5*daySec))
	if a := urgentAccess(pr); a.Untimely != 30 || gateStatus(pr, 1, GateTimelyAccess, WinGated, "urgent") != Fail {
		t.Fatalf("five days: %+v", a)
	}
}

// C6: horizon censoring. An urgent order wanted an hour before the
// horizon and still open at the end of observation (30 days after it) is
// untimely, never dropped; one wanted after the horizon, added to the
// candidate only, is outside both gates' demand: the demand stays paired.
// Ten urgent orders a seed deferred past the horizon are untimely.
func TestGateHorizonCensoring(t *testing.T) {
	p, d := candidatePolicy(t)
	add := func(r *Record, o Order, out string) {
		r.Orders = append(r.Orders, o)
		for i := range r.Captures {
			r.Captures[i].OrderOutcomes = append(r.Captures[i].OrderOutcomes, out)
		}
	}
	both := func(b synth, r *Record) {
		H := r.HorizonEndSec
		add(r, Order{ID: "H1", Priority: "urgent", Modality: "CT", ReceivedSec: H - 2*hour, WantedSec: H - hour}, OutOpen)
	}
	pr, c := clinicalCaseAB(t, p, d, both, func(b synth, r *Record) {
		both(b, r)
		H := r.HorizonEndSec
		add(r, Order{ID: "H2", Priority: "urgent", Modality: "CT", ReceivedSec: H - hour, WantedSec: H + hour}, OutOpen)
	})
	if a := urgentAccess(pr); a.Untimely != 1 || a.Undecided != 0 {
		t.Fatalf("open at the end: %+v", a)
	}
	if g := lostDemandOf(c, "urgent"); g.Status != Pass || g.MatchedSeeds != 40 {
		t.Fatalf("after the horizon: %+v", g)
	}
	pr, _ = clinicalCaseAB(t, p, d, nil, func(b synth, r *Record) {
		for _, i := range urgentOrders(r, 140, 150) {
			resolveAs(r, i, OutServed, r.HorizonEndSec+5*daySec)
		}
	})
	if a := urgentAccess(pr); a.Untimely != 10 {
		t.Fatalf("deferred: %+v", a)
	}
}

// C7: low-volume weekly cohorts. Urgent work is seven acquisitions a
// Monday week here, below the minimum of 20: weeks join into blocks of at
// least 20. Two urgent studies 13 hours past due (severe at 12 hours,
// within the 24-hour absolute limit) in one week: 2 of about 21 in their
// block, above 5%: the weekly gate fails. One: 1 of 21, within 5%:
// passes. With a minimum above the whole period's acquisitions the gate
// lacks evidence: insufficient, not a pass.
func TestGateLowVolumeWeeks(t *testing.T) {
	p, d := candidatePolicy(t)
	late := func(n int) func(b synth, r *Record) {
		return func(b synth, r *Record) {
			for _, i := range studiesOf(r, "urgent", 60, 61+int64(n)-1) {
				finishAt(r, i, r.Studies[i].AcquiredSec+testTargets["urgent"]*60+13*hour)
			}
		}
	}
	pr, c := clinicalCaseAB(t, p, d, nil, late(2))
	ws := pr.PerSeed[0].Measures.Clinical.WeeklySevere["urgent"]
	if gateStatus(pr, 1, GateSevereWeek, WinGated, "urgent") != Fail || ws.WorstSevere != 2 || ws.WorstAcquired < 20 || ws.WorstAcquired > 27 ||
		pr.ClinicalSafety.Status != Pass {
		t.Fatalf("two: %+v, safety %s, %v", ws, pr.ClinicalSafety.Status, c.Reasons)
	}
	pr, _ = clinicalCaseAB(t, p, d, nil, late(1))
	if gateStatus(pr, 1, GateSevereWeek, WinGated, "urgent") != Pass {
		t.Fatalf("one: %+v", pr.PerSeed[0].Measures.Clinical.WeeklySevere["urgent"])
	}
	q := *p
	cl := *p.Clinical
	cl.MinWeeklyCohort = maps.Clone(cl.MinWeeklyCohort)
	cl.MinWeeklyCohort["urgent"] = 100_000
	q.Clinical = &cl
	pr, c = clinicalCaseAB(t, &q, d, nil, nil)
	if gateStatus(pr, 1, GateSevereWeek, WinGated, "urgent") != Insufficient || c.Verdict != NOGO {
		t.Fatalf("too few: %s %v", gateStatus(pr, 1, GateSevereWeek, WinGated, "urgent"), c.Reasons)
	}
}

// C8: overlapping clinical failures count once. Four seeds each failing
// timeliness, the weekly severity and backlog: four failing seeds, within
// the allowance of 4 of 40. A fifth seed failing timeliness alone: five,
// NO GO.
func TestGateOverlappingFailures(t *testing.T) {
	p, d := candidatePolicy(t)
	bad := func(seeds ...int64) func(b synth, r *Record) {
		return func(b synth, r *Record) {
			if !slices.Contains(seeds, r.Seed) {
				return
			}
			delayOrders("urgent", 0, 30, 10*daySec, 30)(b, r)
			if r.Seed <= 4 {
				for _, i := range studiesOf(r, "urgent", 60, 62) {
					finishAt(r, i, r.Studies[i].AcquiredSec+testTargets["urgent"]*60+13*hour)
				}
				staleBlock(b, r)
			}
		}
	}
	pr, _ := clinicalCaseAB(t, p, d, nil, bad(1, 2, 3, 4))
	for _, s := range pr.PerSeed[:4] {
		if s.Backlog != "unstable" || gateStatus(pr, s.Seed, GateSevereWeek, WinGated, "urgent") != Fail || gateStatus(pr, s.Seed, GateTimelyAccess, WinGated, "urgent") != Fail {
			t.Fatalf("seed %d does not fail all three", s.Seed)
		}
	}
	if j := pr.ClinicalJoint; j.Status != Pass || len(j.FailingSeeds) != 4 {
		t.Fatalf("four overlapping: %+v", j)
	}
	pr, c := clinicalCaseAB(t, p, d, nil, bad(1, 2, 3, 4, 5))
	if j := pr.ClinicalJoint; j.Status != Fail || len(j.FailingSeeds) != 5 || c.Verdict != NOGO {
		t.Fatalf("five: %+v", j)
	}
}

// C9: missing baseline data. The baseline lacking a seed: the comparison
// is invalid (INVALID, never a verdict). The baseline's records with no
// captures for a seed: that seed's demand cannot be read: the lost-demand
// gates lack evidence, NO GO.
func TestGateMissingBaseline(t *testing.T) {
	p, d := candidatePolicy(t)
	in := Input{Baseline: "A", Plans: map[string][]Record{}}
	for seed := int64(1); seed <= 40; seed++ {
		in.Seeds = append(in.Seeds, seed)
		for _, x := range []string{"A", "B"} {
			b := base(x, seed)
			b.breachPM = map[string]int{}
			if x == "A" && seed == 7 {
				continue
			}
			in.Plans[x] = append(in.Plans[x], b.record())
		}
	}
	rep := Evaluate(p, d, in)
	if c := comparison(t, rep, "B"); c.Verdict != INVALID {
		t.Fatalf("missing seed: %s", c.Verdict)
	}
	a := base("A", 7).record()
	a.Captures = nil
	gs := lostDemandGates(p, []Record{base("B", 7).record()}, []Record{a})
	for _, g := range gs {
		if g.Status != Insufficient || !slices.Equal(g.UnmatchedSeeds, []int64{7}) {
			t.Fatalf("no captures: %+v", g)
		}
	}
}

// C10: noncomparable demand. Extra urgent orders on one seed of the
// candidate, or receipt times moved: that seed's demand is not the
// baseline's, so the comparison is not paired: lost demand insufficient,
// NO GO, rather than pairing orders that are not the same patients.
// Relabelling thirty urgent orders as routine changes both priorities'
// demand: both insufficient.
func TestGateNoncomparableDemand(t *testing.T) {
	p, d := candidatePolicy(t)
	_, c := clinicalCaseAB(t, p, d, nil, func(b synth, r *Record) {
		if r.Seed == 9 {
			for _, i := range urgentOrders(r, 10, 12) {
				r.Orders[i].ReceivedSec -= 60
				r.Orders[i].WantedSec -= 60
			}
		}
	})
	if g := lostDemandOf(c, "urgent"); g.Status != Insufficient || !slices.Equal(g.UnmatchedSeeds, []int64{9}) || c.Verdict != NOGO ||
		!hasReason(c, "not paired") {
		t.Fatalf("moved receipts: %+v %v", g, c.Reasons)
	}
	_, c = clinicalCaseAB(t, p, d, nil, func(b synth, r *Record) {
		for _, i := range urgentOrders(r, 20, 50) {
			r.Orders[i].Priority = "routine"
		}
	})
	if lostDemandOf(c, "urgent").Status != Insufficient || lostDemandOf(c, "routine").Status != Insufficient || c.Verdict != NOGO {
		t.Fatalf("relabelled: %+v", c.LostDemand)
	}
}

// The policy's new fields validate: the weekly rate replaces the weekly
// count; a severe delay must be defined before the absolute limit with
// it; a report-only maximum wait has no limit.
func TestCandidateValidation(t *testing.T) {
	p, _ := candidatePolicy(t)
	for name, change := range map[string]func(c *Clinical){
		"both weekly rules":     func(c *Clinical) { c.MaxSeverePerWeek = map[string]int64{"urgent": 5, "soon": 5, "routine": 5} },
		"severe at the limit":   func(c *Clinical) { c.SevereAfterDueMin = maps.Clone(c.AbsoluteMaxPastDueMin) },
		"report-only with wait": func(c *Clinical) { c.MaxWaitMin = map[string]int64{"urgent": 10080, "soon": 30240, "routine": 44640} },
		"no weekly minimum":     func(c *Clinical) { c.MinWeeklyCohort = map[string]int64{} },
		"margin out of range": func(c *Clinical) {
			c.LostDemand = &LostDemand{MaxIncrease: map[string]float64{"urgent": 1, "soon": 0, "routine": 0}, Confidence: 0.95, MinCohort: map[string]int64{"urgent": 1, "soon": 1, "routine": 1}}
		},
		"low confidence": func(c *Clinical) {
			c.LostDemand = &LostDemand{MaxIncrease: map[string]float64{"urgent": 0.02, "soon": 0.02, "routine": 0.02}, Confidence: 0.5, MinCohort: map[string]int64{"urgent": 1, "soon": 1, "routine": 1}}
		},
	} {
		q := *p
		c := *p.Clinical
		change(&c)
		q.Clinical = &c
		if err := q.validate(); err == nil {
			t.Errorf("%s: valid", name)
		}
	}
}

// C7a: the short remainder of weeks after the last full block joins it,
// so severe delays at the end of the period are judged. About seventy
// urgent studies, one a day, in blocks of at least 60: the last days'
// remainder joins the one block. Four severe studies in the last two days:
// 4 of 72, above 5%: fail; three: 3 of 71, within: pass.
func TestGateWeeklyRemainder(t *testing.T) {
	p, _ := candidatePolicy(t)
	c := *p.Clinical
	c.MinWeeklyCohort = map[string]int64{"urgent": 60, "soon": 40, "routine": 100}
	mk := func(severe int) *WeeklySevere {
		r := &Record{TakeoverSec: t0, HorizonEndSec: t0 + 70*daySec, TATTargetMin: testTargets}
		for d := range 70 {
			acq := t0 + int64(d)*daySec + hour
			fin := acq + 30*60
			if d >= 70-2 && severe > 0 {
				for range min(severe, 2) {
					r.Studies = append(r.Studies, Study{Priority: "urgent", AcquiredSec: acq, FinalSec: acq + hour + 13*hour})
					severe--
					acq += 60
				}
				continue
			}
			r.Studies = append(r.Studies, Study{Priority: "urgent", AcquiredSec: acq, FinalSec: fin})
		}
		return weeklySevere(&c, r, t0+200*daySec)["urgent"]
	}
	if ws := mk(4); ws.Failing != 1 || ws.Blocks != 1 || ws.WorstSevere != 4 {
		t.Fatalf("four at the end: %+v", ws)
	}
	if ws := mk(3); ws.Failing != 0 || ws.WorstSevere != 3 {
		t.Fatalf("three at the end: %+v", ws)
	}
}
