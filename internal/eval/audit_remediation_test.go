package eval

import (
	"math"
	"slices"
	"strings"
	"testing"
)

// Slice 5C.4d: remediation of the 5C.4c audit's implementation findings
// (docs/audits/slice-5c4c-access-gates-audit.md F2, F3, F8, F9) and the
// weekly rule's partition sensitivity. F8 is in TestCandidateValidation.

// F2: Gate B reads the validated end capture by name. Thirty urgent
// orders a seed received after T30 lost by the candidate only; the
// capture arrays permuted so that T30 is last: the gates are the same as
// with the arrays in order (before the fix, the last element, T30, showed
// those orders not yet disclosed, and the loss vanished).
func TestAuditF2CaptureOrder(t *testing.T) {
	p, d := candidatePolicy(t)
	pt := withMethod(p, BoundT)
	lose := func(b synth, r *Record) {
		for _, i := range urgentOrders(r, 40, 70) {
			resolveAs(r, i, OutLost, r.Orders[i].ReceivedSec)
		}
	}
	_, inOrder := clinicalCaseAB(t, pt, d, nil, lose)
	permute := func(r *Record) {
		k := slices.IndexFunc(r.Captures, func(c Capture) bool { return c.Name == "T30" })
		c := r.Captures[k]
		r.Captures = append(slices.Delete(r.Captures, k, k+1), c)
	}
	in := Input{Baseline: "A", Plans: map[string][]Record{}}
	for seed := int64(1); seed <= 40; seed++ {
		in.Seeds = append(in.Seeds, seed)
		for _, x := range []string{"A", "B"} {
			b := base(x, seed)
			b.breachPM = map[string]int{}
			if x == "B" {
				b.profitPD = 30_000
			}
			r := b.record()
			if x == "B" {
				lose(b, &r)
				resync(b, &r)
			}
			permute(&r)
			in.Plans[x] = append(in.Plans[x], r)
		}
	}
	permuted := comparison(t, Evaluate(pt, d, in), "B")
	a, b := lostDemandOf(inOrder, "urgent"), lostDemandOf(permuted, "urgent")
	if a.Status != Fail || b.Status != a.Status || b.Increase != a.Increase || b.Candidate != a.Candidate {
		t.Fatalf("in order %+v; permuted %+v", a, b)
	}
	if endCapture(&in.Plans["B"][0]) == nil || in.Plans["B"][0].Captures[len(in.Plans["B"][0].Captures)-1].Name != "T30" {
		t.Fatal("the witness is not permuted")
	}
}

// F3: missing or unknown evidence is INVALID: an empty or unknown
// outcome for a resolved order; an unknown outcome at an earlier
// capture; an unknown order or study priority (before the fix: measured
// as untimely or ignored, or a panic indexing the cohort).
func TestAuditF3UnknownEvidence(t *testing.T) {
	p, _ := candidatePolicy(t)
	for name, change := range map[string]func(r *Record){
		"empty terminal outcome": func(r *Record) {
			i := urgentOrders(r, 60, 61)[0]
			for k := range r.Captures {
				if r.Captures[k].AtSec >= r.Orders[i].ResolvedSec {
					r.Captures[k].OrderOutcomes[i] = ""
				}
			}
		},
		"unknown terminal outcome": func(r *Record) {
			i := urgentOrders(r, 60, 61)[0]
			for k := range r.Captures {
				if r.Captures[k].AtSec >= r.Orders[i].ResolvedSec {
					r.Captures[k].OrderOutcomes[i] = "imaged"
				}
			}
		},
		"unknown order priority": func(r *Record) { r.Orders[urgentOrders(r, 60, 61)[0]].Priority = "stat" },
		"unknown study priority": func(r *Record) { r.Studies[studiesOf(r, "urgent", 60, 61)[0]].Priority = "stat" },
	} {
		b := base("B", 1)
		b.breachPM = map[string]int{}
		r := b.record()
		change(&r)
		var m *SeedMeasures
		func() {
			defer func() {
				if x := recover(); x != nil {
					t.Fatalf("%s: panic %v", name, x)
				}
			}()
			m = Measure(p, &r)
		}()
		if len(m.Integrity) == 0 || m.Clinical != nil {
			t.Errorf("%s: valid (%v), clinical measured %v", name, m.Integrity, m.Clinical != nil)
		}
	}
}

// weeklyRecord is a record whose soon studies fall in consecutive
// evaluator weeks from the first Monday after the takeover, n[w] in week
// w, the first sev[w] of them severe (24 h past due), the rest read on
// time.
func weeklyRecord(n, sev []int) *Record {
	mon := int64(11575) * daySec // an evaluator Monday: (11575 + 3) mod 7 = 0
	r := &Record{TakeoverSec: mon - hour, TakeoverDate: "2001-09-09", HorizonEndSec: mon + int64(len(n))*7*daySec, TATTargetMin: testTargets}
	for w := range n {
		for j := range n[w] {
			acq := mon + int64(w)*7*daySec + int64(j)*hour + hour
			fin := acq + hour
			if j < sev[w] {
				fin = acq + testTargets["soon"]*60 + 25*hour
			}
			r.Studies = append(r.Studies, Study{Priority: "soon", AcquiredSec: acq, FinalSec: fin})
		}
	}
	return r
}

func weeklySoon(t *testing.T, n, sev []int) *WeeklySevere {
	t.Helper()
	p, _ := candidatePolicy(t)
	r := weeklyRecord(n, sev)
	return weeklySevere(p.Clinical, r, r.HorizonEndSec+40*daySec)["soon"]
}

// F9: the worst block's span is reported in calendar weeks and dates. A
// block over five calendar weeks, three with acquisitions: five weeks,
// three occupied, Monday to Sunday dates. (Before the fix the block was
// "three weeks".)
func TestAuditF9WeeklySpans(t *testing.T) {
	ws := weeklySoon(t, []int{15, 0, 15, 0, 15}, []int{3, 0, 0, 0, 0})
	if ws.Blocks != 1 || ws.WorstWeeks != 5 || ws.WorstOccupiedWeeks != 3 || ws.WorstFrom != "2001-09-10" || ws.WorstTo != "2001-10-14" {
		t.Fatalf("%+v", ws)
	}
}

// The weekly rule at its boundaries (Phase 6): the exact 5% (2 of 40)
// passes and 3 of 40 fails; a study on Sunday 23:00 and one on Monday
// 01:00 are in different weeks; acquisitions after the horizon are not
// counted; an unfinished study is severe once observation shows it past
// the threshold.
func TestWeeklyBoundaries(t *testing.T) {
	if ws := weeklySoon(t, []int{40}, []int{2}); ws.Failing != 0 {
		t.Fatalf("2 of 40: %+v", ws)
	}
	if ws := weeklySoon(t, []int{40}, []int{3}); ws.Failing != 1 {
		t.Fatalf("3 of 40: %+v", ws)
	}
	p, _ := candidatePolicy(t)
	r := weeklyRecord([]int{40, 40}, []int{0, 0})
	mon2 := int64(11575)*daySec + 7*daySec
	r.Studies = append(r.Studies, Study{Priority: "soon", AcquiredSec: mon2 - hour, FinalSec: mon2 - hour + 30*hour},
		Study{Priority: "soon", AcquiredSec: mon2 + hour, FinalSec: mon2 + hour + 30*hour})
	ws := weeklySevere(p.Clinical, r, r.HorizonEndSec+40*daySec)["soon"]
	if ws.Acquired != 82 || ws.Blocks != 2 || ws.WorstSevere != 1 {
		t.Fatalf("Monday boundary: %+v", ws)
	}
	r = weeklyRecord([]int{40}, []int{0})
	r.Studies = append(r.Studies, Study{Priority: "soon", AcquiredSec: r.HorizonEndSec + hour})
	if ws := weeklySevere(p.Clinical, r, r.HorizonEndSec+40*daySec)["soon"]; ws.Acquired != 40 {
		t.Fatalf("after the horizon counted: %+v", ws)
	}
	r = weeklyRecord([]int{40}, []int{0})
	for i := range 3 {
		r.Studies[i].FinalSec = 0
	}
	if ws := weeklySevere(p.Clinical, r, r.HorizonEndSec+40*daySec)["soon"]; ws.Severe != 3 || ws.Failing != 1 {
		t.Fatalf("unfinished: %+v", ws)
	}
}

// Partition sensitivity (Phase 6): the same clinical sequence, five weeks
// of 20 soon acquisitions with 3 severe in the second, fails under
// forward pooling (weeks 1–2: 3 of 40) and passes pooled from the end
// (weeks 5–4, then 3–2 with week 1's remainder: 3 of 60): its verdict
// depends on the partition alone, and is flagged. Adding non-severe work
// changes the forward verdict (the audit's F7.2). A sequence both
// partitions fail is not flagged.
func TestWeeklyPartitionSensitivity(t *testing.T) {
	ws := weeklySoon(t, []int{20, 20, 20, 20, 20}, []int{0, 3, 0, 0, 0})
	if ws.Failing == 0 || ws.BackwardFailing != 0 || !ws.PartitionSensitive {
		t.Fatalf("sensitive sequence: %+v", ws)
	}
	if a, b := weeklySoon(t, []int{39, 100}, []int{3, 0}), weeklySoon(t, []int{40, 100}, []int{3, 0}); a.Failing != 0 || b.Failing != 1 {
		t.Fatalf("non-severe volume: %+v / %+v", a, b)
	}
	if ws := weeklySoon(t, []int{40, 40}, []int{3, 3}); ws.PartitionSensitive {
		t.Fatalf("both fail: %+v", ws)
	}
}

// The verdict limitation (Phase 5): a candidate and a baseline both
// leaving 65% of urgent demand unimaged (the simulator's no-action loss),
// everything else passing, with the t bound (so that Gate B can pass):
// the favourable verdict states "Access non-deterioration only; absolute
// access adequacy not established.", and the access statement shows the
// 65% unimaged for both, the timeliness and the bound, adequacy never
// established. Under the candidate's betting bound Gate B is insufficient
// and the verdict NO GO, with the access statement still shown.
func TestAccessVerdictLimitation(t *testing.T) {
	p, d := candidatePolicy(t)
	lose := func(b synth, r *Record) {
		for k, i := range urgentOrders(r, 0, 150) {
			if k%20 < 13 { // 65%, spread over the period
				resolveAs(r, i, OutLost, r.Orders[i].ReceivedSec)
			}
		}
	}
	_, c := clinicalCaseAB(t, withMethod(p, BoundT), d, lose, lose)
	if c.Verdict != GO && c.Verdict != CONDITIONAL {
		t.Fatalf("not favourable: %s %v", c.Verdict, c.Reasons)
	}
	if c.AccessLimitation != AccessLimitation || len(c.Reasons) == 0 || c.Reasons[0] != AccessLimitation {
		t.Fatalf("limitation missing: %q %v", c.AccessLimitation, c.Reasons)
	}
	if len(c.Access) != 3 {
		t.Fatalf("access statements: %+v", c.Access)
	}
	for _, a := range c.Access {
		if a.AdequacyEstablished || a.LostDemandStatus != Pass || a.TimelinessStatus == "missing" {
			t.Fatalf("%+v", a)
		}
		if a.Priority == "urgent" && (a.Unimaged[0] < 0.64 || a.Unimaged[1] < 0.64 || a.Timeliness.Value != 1) {
			t.Fatalf("urgent: %+v", a)
		}
	}
	_, c = clinicalCaseAB(t, p, d, lose, lose)
	if c.Verdict != NOGO || c.AccessLimitation != "" || len(c.Access) != 3 || c.Access[0].LostDemandStatus != Insufficient {
		t.Fatalf("betting: %s %q %+v", c.Verdict, c.AccessLimitation, c.Access)
	}
}

// Identity (Phase 2): an eligible order with no origin, or an origin
// repeated, on one seed of the candidate: that seed is unmatched and
// Gate B insufficient, whatever the method; never a favourable match.
func TestGateBMissingIdentity(t *testing.T) {
	p, d := candidatePolicy(t)
	pt := withMethod(p, BoundT)
	for name, change := range map[string]func(r *Record){
		"missing":  func(r *Record) { r.Orders[urgentOrders(r, 30, 31)[0]].Origin = "" },
		"repeated": func(r *Record) { os := urgentOrders(r, 30, 32); r.Orders[os[1]].Origin = r.Orders[os[0]].Origin },
	} {
		_, c := clinicalCaseAB(t, pt, d, nil, func(b synth, r *Record) {
			if r.Seed == 5 {
				change(r)
			}
		})
		if g := lostDemandOf(c, "urgent"); g.Status != Insufficient || !slices.Equal(g.UnmatchedSeeds, []int64{5}) || c.Verdict != NOGO {
			t.Errorf("%s: %+v %s", name, g, c.Verdict)
		}
	}
}

// Records written before 0.2.0-dev.18 carry no origins: no seed can be
// paired, Gate B is insufficient for the missing identity (never the
// orders collapsed onto one empty key). A seed with fewer matched orders
// than the policy's minimum makes Gate B insufficient too.
func TestGateBOldRecordsAndThinSeeds(t *testing.T) {
	p, d := candidatePolicy(t)
	pt := withMethod(p, BoundT)
	strip := func(b synth, r *Record) {
		for i := range r.Orders {
			r.Orders[i].Origin = ""
		}
	}
	_, c := clinicalCaseAB(t, pt, d, strip, strip)
	if g := lostDemandOf(c, "urgent"); g.Status != Insufficient || len(g.UnmatchedSeeds) != 40 || !strings.Contains(g.Reason, "identity evidence missing") {
		t.Fatalf("no origins: %+v", g)
	}
	q := withMethod(p, BoundT)
	q.Clinical.LostDemand.MinSeedOrders = map[string]int64{"urgent": 1000, "soon": 40, "routine": 40}
	_, c = clinicalCaseAB(t, q, d, nil, nil)
	if g := lostDemandOf(c, "urgent"); g.Status != Insufficient || !strings.Contains(g.Reason, "fewer than 1000 matched orders") {
		t.Fatalf("thin: %+v", g)
	}
}

// The t bound's value: x̄ + t(0.95, n − 1) s / √n, one-sided.
func TestTBoundValue(t *testing.T) {
	xs := []float64{0.01, 0.03, 0.02, 0.00, 0.04}
	m, sd := meanSD(xs)
	want := m + 2.131847*sd/math.Sqrt(5)
	if u := tUpper(xs, 0.95); math.Abs(u-want) > 1e-6 {
		t.Fatalf("t bound %v, want %v", u, want)
	}
}
