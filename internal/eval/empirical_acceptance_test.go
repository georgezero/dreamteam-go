package eval

import (
	"encoding/json"
	"math"
	"slices"
	"strings"
	"testing"
)

// Slice 5C.4e: Gate B as an empirical benchmark acceptance rule (no
// population-confidence claim), its per-seed safeguard in the shared
// allowance, and the weekly severity over fixed calendar weeks
// (docs/calibration/clinical-5c4e/). Synthetic fixtures only: 40 seeds,
// one urgent order a day, 150 eligible urgent orders a seed.
//
// The fixture's 7 urgent acquisitions a week can never meet the calendar
// rule's minimum of 20 (nor can the simulator's, about 6.5: the stop
// condition of empirical-acceptance.md §5), so the Gate B witnesses use
// the candidate with the pooled weekly blocks, to isolate Gate B;
// TestCalendarUrgentUnevaluable shows the candidate itself.

// empiricalPolicy is the 5C.4e candidate (the same Gate B and safeguard
// as the current one) with the pooled weekly blocks.
func empiricalPolicy(t *testing.T) (*Policy, string) {
	t.Helper()
	p, d := candidate5c4ePolicy(t)
	c := *p.Clinical
	c.WeeklyPartition = ""
	p.Clinical = &c
	return p, d
}

// loseUrgent makes lost k(seed) eligible urgent orders of a seed, spread
// over the period.
func loseUrgent(k func(seed int64) int) func(b synth, r *Record) {
	return func(b synth, r *Record) {
		os := urgentOrders(r, 0, 150)
		n := k(r.Seed)
		for j := range n {
			i := os[j*len(os)/max(n, 1)]
			resolveAs(r, i, OutLost, r.Orders[i].ReceivedSec)
		}
	}
}

func favourable(c Comparison) bool { return c.Verdict == GO || c.Verdict == CONDITIONAL }

// 1–3: the mean rule at its margin, exactly. Three lost of 150 on every
// seed is exactly 2 points: PASS (the rule compares the exact rational
// mean). One seed losing four: 121/6000, just above: FAIL. Two or three:
// PASS. (The v4 case-selection rules, served and acquired drop pooled at
// 2 points over the T90 due orders, bind on this fixture before Gate B
// does: the verdict is theirs; Gate B adds no reason when it passes.)
func TestEmpiricalMargin(t *testing.T) {
	p, d := empiricalPolicy(t)
	for _, x := range []struct {
		name string
		k    func(int64) int
		want string
	}{
		{"exactly 2 points", func(int64) int { return 3 }, Pass},
		{"just above", func(s int64) int { return 3 + int(max(0, 2-s)) }, Fail},
		{"below", func(s int64) int { return 2 + int(s%2) }, Pass},
	} {
		_, c := clinicalCaseAB(t, p, d, nil, loseUrgent(x.k))
		g := lostDemandOf(c, "urgent")
		if g.Status != x.want || g.Rule != RuleEmpirical || g.Orders != 40*150 || len(g.Deteriorated) != 0 {
			t.Errorf("%s: %s %s (increase %.6f)", x.name, g.Status, g.Reason, g.Increase)
		}
		if x.want == Pass && hasReason(c, "lost demand") {
			t.Errorf("%s: %s %v", x.name, c.Verdict, c.Reasons)
		}
		if x.want == Fail && (c.Verdict != NOGO || !hasReason(c, "lost demand urgent: fail")) {
			t.Errorf("%s: %s %v", x.name, c.Verdict, c.Reasons)
		}
	}
}

// 4: concentrated harm in one seed. Seed 7 loses 45 of its 150 urgent
// orders (30 points), no other seed loses any: the mean (0.75 points)
// passes; the safeguard flags seed 7, which fails in the comparison's
// shared allowance though its own plan gates pass. Within the allowance
// (1 of 4) the verdict stands; with four other seeds failing timeliness
// on their own (the plan's allowance exactly full) seed 7 is the fifth:
// NO GO on clinical seed reliability.
func TestSafeguardConcentratedHarm(t *testing.T) {
	p, d := empiricalPolicy(t)
	harm := loseUrgent(func(s int64) int { return map[bool]int{true: 45}[s == 7] })
	pr, c := clinicalCaseAB(t, p, d, nil, harm)
	g := lostDemandOf(c, "urgent")
	if g.Status != Pass || !slices.Equal(g.Deteriorated, []int64{7}) || g.SeedThreshold != 0.1 {
		t.Fatalf("one seed: %+v", g)
	}
	j := c.ClinicalJoint
	if j == nil || !slices.Equal(j.FailingSeeds, []int64{7}) || j.Status != Pass || slices.Contains(pr.ClinicalJoint.FailingSeeds, 7) ||
		j.Gates != pr.ClinicalJoint.Gates+1 || !favourable(c) {
		t.Fatalf("one seed: joint %+v, plan %+v, %s %v", j, pr.ClinicalJoint, c.Verdict, c.Reasons)
	}
	four := delayOrders("urgent", 0, 30, 10*daySec, 30)
	pr, c = clinicalCaseAB(t, p, d, nil, func(b synth, r *Record) {
		if r.Seed <= 4 {
			four(b, r)
		}
		harm(b, r)
	})
	if pr.ClinicalJoint.Status != Pass || len(pr.ClinicalJoint.FailingSeeds) != 4 || c.ClinicalJoint.Status != Fail ||
		!slices.Equal(c.ClinicalJoint.FailingSeeds, []int64{1, 2, 3, 4, 7}) || c.Verdict != NOGO || !hasReason(c, "clinical seed reliability: fail") {
		t.Fatalf("fifth seed: plan %+v, comparison %+v, %s %v", pr.ClinicalJoint, c.ClinicalJoint, c.Verdict, c.Reasons)
	}
}

// 5: harm offset by improvements elsewhere. The baseline loses 20 urgent
// orders a seed; the candidate none on 35 seeds and 65 on seeds 1–5
// (+30 points each). The mean increase is −7.9 points: Gate B passes. The
// safeguard flags the five seeds, beyond the allowance of four: NO GO.
func TestSafeguardHarmOffset(t *testing.T) {
	p, d := empiricalPolicy(t)
	_, c := clinicalCaseAB(t, p, d, loseUrgent(func(int64) int { return 20 }), loseUrgent(func(s int64) int {
		if s <= 5 {
			return 65
		}
		return 0
	}))
	g := lostDemandOf(c, "urgent")
	if g.Status != Pass || g.Increase > -0.07 || !slices.Equal(g.Deteriorated, []int64{1, 2, 3, 4, 5}) || g.Distribution.Better != 35 || g.Distribution.Worse != 5 {
		t.Fatalf("%+v", g)
	}
	if c.ClinicalJoint.Status != Fail || c.Verdict != NOGO || !hasReason(c, "clinical seed reliability: fail") {
		t.Fatalf("%+v %s %v", c.ClinicalJoint, c.Verdict, c.Reasons)
	}
	// Without the safeguard the clinical rule passes the offset: no
	// clinical reason. (The v4 case-selection per-seed served drop, 5
	// points with its own allowance, fails it here: an overlap reported in
	// seed-safeguard.md §2.)
	q := *p
	cl := *p.Clinical
	ld := *cl.LostDemand
	ld.MaxSeedIncrease = nil
	cl.LostDemand, q.Clinical = &ld, &cl
	_, c = clinicalCaseAB(t, &q, d, loseUrgent(func(int64) int { return 20 }), loseUrgent(func(s int64) int { return map[bool]int{true: 65}[s <= 5] }))
	if c.ClinicalJoint != nil || lostDemandOf(c, "urgent").Status != Pass || hasReason(c, "clinical") || !hasReason(c, "served_fraction_drop_per_seed") {
		t.Fatalf("no safeguard: %s %v", c.Verdict, c.Reasons)
	}
}

// 6–8: missing baseline evidence, mismatched or duplicated identities on
// one seed: INSUFFICIENT under the empirical rule, never PASS, and that
// seed lacks evidence in the safeguard's allowance.
func TestEmpiricalIdentity(t *testing.T) {
	p, d := empiricalPolicy(t)
	for name, change := range map[string]func(r *Record){
		"mismatched": func(r *Record) { r.Orders[urgentOrders(r, 30, 31)[0]].Origin = "X/1" },
		"duplicate":  func(r *Record) { os := urgentOrders(r, 30, 32); r.Orders[os[1]].Origin = r.Orders[os[0]].Origin },
		"missing":    func(r *Record) { r.Orders[urgentOrders(r, 30, 31)[0]].Origin = "" },
	} {
		_, c := clinicalCaseAB(t, p, d, nil, func(b synth, r *Record) {
			if r.Seed == 5 {
				change(r)
			}
		})
		g := lostDemandOf(c, "urgent")
		if g.Status != Insufficient || !slices.Equal(g.UnmatchedSeeds, []int64{5}) || c.Verdict != NOGO || !slices.Contains(c.ClinicalJoint.InsufficientSeeds, 5) {
			t.Errorf("%s: %+v %s", name, g, c.Verdict)
		}
	}
	// The baseline without a seed's captures: its demand cannot be read.
	a := base("A", 7).record()
	a.Captures = nil
	for _, g := range lostDemandGates(p, []Record{base("B", 7).record()}, []Record{a}) {
		if g.Status != Insufficient || !slices.Equal(g.UnmatchedSeeds, []int64{7}) {
			t.Fatalf("missing baseline: %+v", g)
		}
	}
	// No baseline record for a seed at all.
	for _, g := range lostDemandGates(p, []Record{base("B", 7).record()}, nil) {
		if g.Status != Insufficient || !slices.Equal(g.UnmatchedSeeds, []int64{7}) {
			t.Fatalf("no baseline: %+v", g)
		}
	}
}

// 9: cancellation relabelling. The same 30 urgent orders a seed lost, or
// relabelled as cancelled by the patient: both unimaged, so Gate B is the
// same (FAIL, 20 points); Gate A withdraws the patient cancellations and
// counts the lost apart: timeliness passes either way.
func TestEmpiricalCancellationRelabelling(t *testing.T) {
	p, d := empiricalPolicy(t)
	as := func(out string) func(b synth, r *Record) {
		return func(b synth, r *Record) {
			for _, i := range urgentOrders(r, 20, 50) {
				resolveAs(r, i, out, r.Orders[i].ReceivedSec)
			}
		}
	}
	prL, cl := clinicalCaseAB(t, p, d, nil, as(OutLost))
	prP, cp := clinicalCaseAB(t, p, d, nil, as(OutPatient))
	a, b := lostDemandOf(cl, "urgent"), lostDemandOf(cp, "urgent")
	if a.Status != Fail || b.Status != Fail || a.Increase != b.Increase || a.Candidate.Total != b.Candidate.Total || b.Candidate.Patient != 40*30 {
		t.Fatalf("lost %+v; patient %+v", a, b)
	}
	if gateStatus(prL, 1, GateTimelyAccess, WinGated, "urgent") != Pass || gateStatus(prP, 1, GateTimelyAccess, WinGated, "urgent") != Pass ||
		urgentAccess(prP).Withdrawn != 30 || urgentAccess(prL).Lost != 30 {
		t.Fatalf("Gate A: %+v / %+v", urgentAccess(prL), urgentAccess(prP))
	}
}

// 10: all outcomes unimaged. Both plans losing every urgent order: no
// increase, Gate B passes empirically, both absolute fractions 1.0 are
// shown, and Gate A has no decided orders: insufficient, NO GO. The
// candidate alone losing every one: +100 points on every seed, FAIL and
// every seed flagged.
func TestEmpiricalAllUnimaged(t *testing.T) {
	p, d := empiricalPolicy(t)
	all := loseUrgent(func(int64) int { return 150 })
	_, c := clinicalCaseAB(t, p, d, all, all)
	g := lostDemandOf(c, "urgent")
	if g.Status != Pass || g.Increase != 0 || g.SeedFraction != [2]float64{1, 1} || c.Verdict != NOGO || !hasReason(c, "timely_access") {
		t.Fatalf("both: %+v %s %v", g, c.Verdict, c.Reasons)
	}
	_, c = clinicalCaseAB(t, p, d, nil, all)
	if g := lostDemandOf(c, "urgent"); g.Status != Fail || g.Increase != 1 || len(g.Deteriorated) != 40 {
		t.Fatalf("candidate: %+v", g)
	}
}

// 11: zero observed deterioration passes (the 5C.4d betting rule could
// not), with every claim kept apart: the access limitation and the
// empirical scope lead the reasons; the claims say the acceptance is
// empirical, population confidence not claimed, adequacy not
// established; the betting bound is reported only as a labelled research
// diagnostic (it cannot reach the margin at 40 seeds); no upper bound or
// confidence appears in the gate's decision.
func TestEmpiricalNoPopulationClaim(t *testing.T) {
	p, d := empiricalPolicy(t)
	_, c := clinicalCaseAB(t, p, d, nil, nil)
	if !favourable(c) || len(c.Reasons) < 2 || c.Reasons[0] != AccessLimitation || c.Reasons[1] != EmpiricalScope {
		t.Fatalf("%s %v", c.Verdict, c.Reasons)
	}
	if a := c.AccessClaims; a == nil || a.EmpiricalAcceptance != Pass || a.Seeds != 40 || a.PopulationConfidence != ConfidenceNotClaimed ||
		a.AbsoluteAdequacy != AdequacyNotEstablished {
		t.Fatalf("claims %+v", c.AccessClaims)
	}
	for _, g := range c.LostDemand {
		dg := g.Diagnostic
		if g.Status != Pass || g.Increase != 0 || g.IncreaseUpper != nil || g.ZeroIncreaseUpper != nil || dg == nil || dg.Purpose != DiagnosticPurpose ||
			dg.Method != BoundBetting || dg.Established || dg.ZeroUpper < 0.2 || g.Distribution.Equal != 40 {
			t.Fatalf("%+v %+v", g, dg)
		}
	}
	for _, a := range c.Access {
		if a.IncreaseUpper != nil || a.DiagnosticUpper == nil || a.AdequacyEstablished || a.Rule != RuleEmpirical {
			t.Fatalf("access %+v", a)
		}
	}
	// The decision's words: no bound, no confidence, no proof. The gate's
	// JSON has no increaseUpper; the bound appears only in the labelled
	// diagnostic.
	raw, _ := json.Marshal(c.LostDemand)
	if strings.Contains(string(raw), `"increaseUpper"`) || strings.Contains(string(raw), `"zeroIncreaseUpper"`) {
		t.Fatalf("an upper bound outside the diagnostic: %s", raw)
	}
	_, f := clinicalCaseAB(t, p, d, nil, loseUrgent(func(int64) int { return 30 }))
	for _, s := range append(slices.Clone(f.Reasons), lostDemandOf(f, "urgent").Reason) {
		low := strings.ToLower(s)
		for _, w := range []string{"upper bound", "95%", "confidence", "proof", "proven", "population non-deterioration"} {
			if strings.Contains(low, w) {
				t.Errorf("claim %q in %q", w, s)
			}
		}
	}
	for _, s := range c.Reasons {
		if low := strings.ToLower(s); strings.Contains(low, "proof") || strings.Contains(low, "95%") ||
			(strings.Contains(low, "confidence") && s != EmpiricalScope) {
			t.Errorf("favourable reason claims %q", s)
		}
	}
}

// The empirical rule's policy: the approximate t is refused even as its
// diagnostic; a safeguard at or below the mean margin repeats the mean
// rule and is refused; the safeguard belongs to the empirical rule; an
// unknown rule or partition is refused.
func TestEmpiricalValidation(t *testing.T) {
	p, _ := candidatePolicy(t)
	for name, change := range map[string]func(c *Clinical){
		"t diagnostic": func(c *Clinical) { c.LostDemand.Method = BoundT },
		"safeguard at margin": func(c *Clinical) {
			c.LostDemand.MaxSeedIncrease = map[string]float64{"urgent": 0.02, "soon": 0.06, "routine": 0.05}
		},
		"safeguard not fraction": func(c *Clinical) {
			c.LostDemand.MaxSeedIncrease = map[string]float64{"urgent": math.NaN(), "soon": 0.06, "routine": 0.05}
		},
		"safeguard, confidence": func(c *Clinical) { c.LostDemand.Rule = "" },
		"unknown rule":          func(c *Clinical) { c.LostDemand.Rule = "bayes" },
		"unknown partition":     func(c *Clinical) { c.WeeklyPartition = "rolling" },
		"partition, no rate":    func(c *Clinical) { c.MaxWeeklySevereRate, c.MinWeeklyCohort = nil, nil },
	} {
		q := *p
		c := *p.Clinical
		ld := *c.LostDemand
		c.LostDemand = &ld
		change(&c)
		q.Clinical = &c
		if err := q.validate(); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

// calendarSoon is weeklySevereCalendar's soon result on weeklyRecord's
// record (takeover Sunday 23:00, so a partial first week of one hour and
// a partial last week at the horizon's Monday 00:00).
func calendarSoon(t *testing.T, n, sev []int) *WeeklySevere {
	t.Helper()
	p, _ := candidate5c4ePolicy(t)
	r := weeklyRecord(n, sev)
	return weeklySevereCalendar(p.Clinical, r, r.HorizonEndSec+40*daySec)["soon"]
}

func statuses(ws *WeeklySevere) string {
	var s []string
	for _, w := range ws.Weeks {
		s = append(s, w.Status)
	}
	return strings.Join(s, " ")
}

// 12–15: fixed calendar weeks, soon (minimum 40, rate 5%). A complete
// week with no acquisitions or one below the minimum is INSUFFICIENT;
// exactly at the minimum it is judged; 2 of 40 (exactly 5%) passes and
// 3 of 40 fails; weeks are never pooled (39 and 100 would have pooled).
func TestCalendarWeeks(t *testing.T) {
	ws := calendarSoon(t, []int{40, 0, 40}, []int{0, 0, 0})
	if statuses(ws) != "not_judged pass insufficient pass not_judged" || ws.InsufficientWeeks != 1 || !ws.Insufficient || ws.PartialWeeks != 2 ||
		ws.Weeks[2].Acquired != 0 || !ws.Weeks[2].Complete || ws.Weeks[0].Complete {
		t.Fatalf("empty week: %s %+v", statuses(ws), ws)
	}
	if ws := calendarSoon(t, []int{40}, []int{2}); statuses(ws) != "not_judged pass not_judged" || ws.Failing != 0 || ws.Insufficient {
		t.Fatalf("at the minimum, exactly 5%%: %s", statuses(ws))
	}
	if ws := calendarSoon(t, []int{40}, []int{3}); ws.Weeks[1].Status != Fail || ws.Failing != 1 || ws.WorstSevere != 3 || ws.WorstFrom != "2001-09-10" || ws.WorstTo != "2001-09-16" {
		t.Fatalf("3 of 40: %+v", ws)
	}
	if ws := calendarSoon(t, []int{39}, []int{0}); ws.Weeks[1].Status != Insufficient || !ws.Insufficient {
		t.Fatalf("just below: %s", statuses(ws))
	}
	if ws := calendarSoon(t, []int{39, 100}, []int{3, 0}); statuses(ws) != "not_judged insufficient pass not_judged" || ws.Failing != 0 {
		t.Fatalf("not pooled: %s", statuses(ws))
	}
	if ws := calendarSoon(t, []int{20, 20, 20, 20, 20}, []int{0, 3, 0, 0, 0}); ws.InsufficientWeeks != 5 || ws.Failing != 0 || ws.BackwardFailing != 0 || ws.PartitionSensitive {
		t.Fatalf("the 5C.4d partition witness: %s %+v", statuses(ws), ws)
	}
}

// 16: Monday and Sunday boundaries. Studies on Sunday 23:59:59 and
// Monday 00:00:00 fall in different weeks; Sunday's is the first week's.
func TestCalendarMondaySunday(t *testing.T) {
	p, _ := candidate5c4ePolicy(t)
	r := weeklyRecord([]int{40, 40}, []int{0, 0})
	mon2 := int64(11575)*daySec + 7*daySec
	sev := testTargets["soon"]*60 + 25*hour
	r.Studies = append(r.Studies, Study{Priority: "soon", AcquiredSec: mon2 - 1, FinalSec: mon2 - 1 + sev},
		Study{Priority: "soon", AcquiredSec: mon2, FinalSec: mon2 + sev}, Study{Priority: "soon", AcquiredSec: mon2 + 1, FinalSec: mon2 + 1 + sev})
	ws := weeklySevereCalendar(p.Clinical, r, r.HorizonEndSec+40*daySec)["soon"]
	if a, b := ws.Weeks[1], ws.Weeks[2]; a.Acquired != 41 || a.Severe != 1 || b.Acquired != 42 || b.Severe != 2 || a.To != "2001-09-16" || b.From != "2001-09-17" {
		t.Fatalf("%+v / %+v", a, b)
	}
}

// 17: partial boundary weeks. Takeover Wednesday 06:00, horizon three
// weeks later: a partial first week (Wednesday to Sunday), two complete
// weeks, a partial last week (Monday to Wednesday 06:00). A partial week
// at or above the minimum is judged, and fails when above the rate; one
// below it is NOT_JUDGED, never a pass, and does not make the seed
// insufficient (its shortfall is the window's edge, the same for every
// plan); its severe delays stay in the per-seed severe rate and the
// absolute limits.
func TestCalendarPartialWeeks(t *testing.T) {
	p, _ := candidate5c4ePolicy(t)
	mon := int64(11575) * daySec
	wed := mon + 2*daySec + 6*hour
	rec := func(first, firstSev, last, lastSev int) *Record {
		r := &Record{TakeoverSec: wed, TakeoverDate: "2001-09-12", HorizonEndSec: wed + 21*daySec, TATTargetMin: testTargets}
		add := func(from int64, n, sev int) {
			for j := range n {
				acq := from + int64(j)*600
				fin := acq + hour
				if j < sev {
					fin = acq + testTargets["soon"]*60 + 25*hour
				}
				r.Studies = append(r.Studies, Study{Priority: "soon", AcquiredSec: acq, FinalSec: fin})
			}
		}
		add(wed+hour, first, firstSev)
		add(mon+7*daySec+hour, 40, 0)
		add(mon+14*daySec+hour, 40, 0)
		add(mon+21*daySec+hour, last, lastSev)
		return r
	}
	ws := weeklySevereCalendar(p.Clinical, rec(45, 3, 10, 5), wed+60*daySec)["soon"]
	if statuses(ws) != "fail pass pass not_judged" || ws.Weeks[0].Complete || !ws.Weeks[1].Complete || ws.Weeks[3].Complete || ws.PartialWeeks != 2 ||
		ws.NotJudgedWeeks != 1 || ws.Insufficient || ws.Weeks[0].From != "2001-09-10" || ws.Weeks[3].Severe != 5 {
		t.Fatalf("%s %+v", statuses(ws), ws)
	}
	if ws := weeklySevereCalendar(p.Clinical, rec(10, 0, 10, 0), wed+60*daySec)["soon"]; statuses(ws) != "not_judged pass pass not_judged" || ws.Insufficient || ws.Failing != 0 {
		t.Fatalf("both partial below: %s", statuses(ws))
	}
}

// The 5C.4e candidate on the synthetic fixture (7 urgent a week): every
// complete urgent week below the minimum of 20, every seed insufficient,
// NO GO; the gate reports the weeks.
func TestCalendarUrgentUnevaluable(t *testing.T) {
	p, d := candidate5c4ePolicy(t)
	pr, c := clinicalCaseAB(t, p, d, nil, nil)
	ws := pr.PerSeed[0].Measures.Clinical.WeeklySevere["urgent"]
	if ws.Partition != PartitionCalendar || ws.InsufficientWeeks != len(ws.Weeks)-ws.PartialWeeks || ws.InsufficientWeeks == 0 ||
		gateStatus(pr, 1, GateSevereWeek, WinGated, "urgent") != Insufficient || len(pr.ClinicalJoint.InsufficientSeeds) != 40 || c.Verdict != NOGO {
		t.Fatalf("%+v; %s %v", ws, c.Verdict, c.Reasons)
	}
}

// 18: old v5 policy compatibility. The 5C.4d candidate (its exact bytes)
// still loads and keeps its rules: the confidence rule (betting at 40
// seeds insufficient with no difference at all, the bound reported) and
// the pooled weekly blocks; no empirical field appears.
func TestOldV5PolicyCompatibility(t *testing.T) {
	_, digest, err := LoadPolicy(legacyCandidatePath)
	if err != nil || digest != "2bd8f152b75c91e97cfa8bb3437da8fcafb25bc6d19f4256d804e8652f48e2c6" {
		t.Fatalf("%v %s", err, digest)
	}
	p, d := legacyCandidatePolicy(t)
	pr, c := clinicalCaseAB(t, p, d, nil, nil)
	g := lostDemandOf(c, "urgent")
	if g.Status != Insufficient || g.Rule != "" || g.IncreaseUpper == nil || *g.ZeroIncreaseUpper < 0.2 || g.Diagnostic != nil || g.Distribution != nil ||
		c.AccessClaims != nil || c.ClinicalJoint != nil {
		t.Fatalf("%+v", g)
	}
	if ws := pr.PerSeed[0].Measures.Clinical.WeeklySevere["urgent"]; ws.Partition != "" || ws.Weeks != nil || ws.Blocks == 0 {
		t.Fatalf("weekly: %+v", ws)
	}
	raw, _ := json.Marshal(c)
	for _, k := range []string{`"rule"`, `"researchDiagnostic"`, `"accessClaims"`, `"clinicalJointWithSafeguard"`, `"partition"`} {
		if strings.Contains(string(raw), k) {
			t.Errorf("%s in an old v5 comparison", k)
		}
	}
}
