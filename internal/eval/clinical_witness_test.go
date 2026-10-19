package eval

import (
	"encoding/json"
	"maps"
	"os"
	"slices"
	"testing"
)

// The acceptance-design witnesses of Slice 5C.4b's closure
// (docs/calibration/clinical-5c4b/acceptance-design.md §6): each under the
// candidate policy's clinical values (docs/calibration/clinical-5c4b/
// candidate-policy-v5.json) on the synthetic records, with its expected
// classification, denominator, numerator and gate outcome. Since Slice
// 5C.4c the candidate is the owner's modified option B: timeliness over
// served, open and late-cancelled orders (urgent 48 calendar hours at
// 95%), and lost, left and patient-ended demand against the paired
// baseline (the comparison's lostDemand). The synthetic practice has one
// urgent order a day, wanted 30 minutes before it is imaged.

const candidatePath = "../../docs/calibration/clinical-5c4b/candidate-policy-v5.json"

// candidatePolicy is the synthetic fixture's v5 policy with the
// candidate's clinical section.
func candidatePolicy(t *testing.T) (*Policy, string) {
	t.Helper()
	p, d := v5Policy(t)
	raw, err := os.ReadFile(candidatePath)
	if err != nil {
		t.Fatal(err)
	}
	var c struct {
		Clinical *Clinical `json:"clinical"`
	}
	if err := json.Unmarshal(raw, &c); err != nil {
		t.Fatal(err)
	}
	p.Clinical = c.Clinical
	if err := p.validate(); err != nil {
		t.Fatal(err)
	}
	return p, d
}

// The candidate file itself loads as a v5 policy and keeps every rule of
// v0.2-eval-4 apart from its clinical section and the persistent check.
func TestCandidatePolicyFile(t *testing.T) {
	c, _, err := LoadPolicy(candidatePath)
	if err != nil {
		t.Fatal(err)
	}
	v4, _, err := LoadPolicy("../../" + PolicyPath)
	if err != nil {
		t.Fatal(err)
	}
	if c.Schema != PolicyV5Schema || c.Clinical == nil || c.ID == v4.ID {
		t.Fatalf("schema %s id %s", c.Schema, c.ID)
	}
	a, b := *c, *v4
	a.Schema, a.ID, a.Status, a.Effective, a.Notes, a.Supersedes, a.Changes, a.Clinical = "", "", "", "", nil, b.Supersedes, nil, nil
	b.Schema, b.ID, b.Status, b.Effective, b.Notes, b.Changes = "", "", "", "", nil, nil
	a.Backlog.InformationalChecks = slices.DeleteFunc(slices.Clone(a.Backlog.InformationalChecks), func(s string) bool { return s == "persistent" })
	ja, _ := json.Marshal(a)
	jb, _ := json.Marshal(b)
	if string(ja) != string(jb) {
		t.Errorf("the candidate changes a v0.2-eval-4 rule beyond its clinical section:\n%s\n%s", ja, jb)
	}
	cl := c.Clinical
	day := int64(1440)
	if cl.AbsoluteMaxPastDueMin["urgent"] != day || cl.AbsoluteMaxPastDueMin["soon"] != 2*day || cl.AbsoluteMaxPastDueMin["routine"] != 14*day ||
		cl.AccessLimitMin["urgent"] != 2*day || cl.MaxFailingSeedFraction != 0.1 {
		t.Errorf("candidate values: %+v", cl)
	}
}

// urgentOrders are the record's urgent order indices received on days
// [from, to) after the takeover.
func urgentOrders(r *Record, from, to int64) []int {
	var out []int
	for i, o := range r.Orders {
		if d := (o.ReceivedSec - t0) / daySec; o.Priority == "urgent" && o.ReceivedSec > t0 && d >= from && d < to {
			out = append(out, i)
		}
	}
	return out
}

// dropStudy removes the study acquired when order i was resolved (its
// order no longer imaged).
func dropStudy(r *Record, i int) {
	at := r.Orders[i].ResolvedSec
	r.Studies = slices.DeleteFunc(r.Studies, func(s Study) bool { return s.Priority == r.Orders[i].Priority && s.AcquiredSec == at })
}

// resolveAs makes order i end as out at `at` (0: unresolved), its study
// dropped.
func resolveAs(r *Record, i int, out string, at int64) {
	dropStudy(r, i)
	o := &r.Orders[i]
	o.ResolvedSec = at
	if out == OutOpen || out == OutLost || out == OutPatient {
		o.FirstApptSec = 0
	}
	r.Captures[len(r.Captures)-1].OrderOutcomes[i] = out
}

func urgentAccess(pr *PlanReport) *AccessCohort {
	return pr.PerSeed[0].Measures.Clinical.Access[WinGated]["urgent"]
}

// W1: thirty urgent acquisitions 30 days after their wanted time (the
// audit's witness). Denominator every urgent order wanted in the gated
// period; the thirty are served late (untimely); timely 0.8 < 0.95:
// timely access fails on every seed: NO GO (clinical seed reliability);
// the lost-demand gate passes (nothing unimaged). v0.2-eval-4 returns GO.
func TestWitnessDelayedUrgentAccess(t *testing.T) {
	p, d := candidatePolicy(t)
	_, pr, c := clinicalCase(t, p, d, nil, delayOrders("urgent", 0, 30, 30*daySec, 30))
	wantVerdict(t, pr, c, NOGO, "clinical seed reliability: fail (40 failing")
	if a := urgentAccess(pr); a.Untimely != 30 || a.Timely+a.Untimely != a.Wanted || a.Withdrawn != 0 {
		t.Fatalf("%+v", a)
	}
	if gateStatus(pr, 1, GateTimelyAccess, WinGated, "urgent") != Fail || gateStatus(pr, 1, GateMaxWait, WinGated, "urgent") != "missing" ||
		pr.PerSeed[0].Measures.Clinical.MaxWaitGatedMin["urgent"] < 30*1440 || c.LostDemand[0].Status != Pass {
		t.Fatalf("gates: lost demand %+v", c.LostDemand[0])
	}
	q, qd := testPolicy(t)
	_, pr4, c4 := clinicalCase(t, q, qd, nil, delayOrders("urgent", 0, 30, 30*daySec, 30))
	wantVerdict(t, pr4, c4, GO, "")
}

// W2: thirty urgent orders never imaged, the baseline imaging them.
// Lost: counted apart from timeliness (timely access passes) and unimaged
// against the baseline: the lost-demand gate fails, NO GO. Cancelled by
// the patient after the 48-hour limit: untimely, and unimaged: both fail.
// Cancelled by the patient within the limit: withdrawn from timeliness
// (the record does not say why a patient cancelled) but still unimaged
// against the baseline: the lost-demand gate fails. No outcome lets
// demand disappear.
func TestWitnessUrgentOrdersNotImaged(t *testing.T) {
	p, d := candidatePolicy(t)
	for _, c := range []struct {
		name     string
		out      string
		after    int64 // resolution after the wanted time
		untimely int64
		withdraw int64
		lost     int64
		gate     string
	}{
		{"lost", OutLost, 0, 0, 0, 30, Pass},
		{"patient after the limit", OutPatient, 3 * daySec, 30, 0, 0, Fail},
		{"patient within the limit", OutPatient, 2 * hour, 0, 30, 0, Pass},
	} {
		_, pr, cmp := clinicalCase(t, p, d, nil, func(b synth, r *Record) {
			for _, i := range urgentOrders(r, 20, 50) {
				at := r.Orders[i].WantedSec + c.after
				if c.out == OutLost {
					at = r.Orders[i].ReceivedSec
				}
				resolveAs(r, i, c.out, at)
			}
		})
		a := urgentAccess(pr)
		ld := cmp.LostDemand[0]
		if a.Untimely != c.untimely || a.Withdrawn != c.withdraw || a.Lost != c.lost || a.Timely+a.Untimely+a.Withdrawn+a.Lost != a.Wanted ||
			gateStatus(pr, 1, GateTimelyAccess, WinGated, "urgent") != c.gate {
			t.Errorf("%s: %+v, gate %s", c.name, a, gateStatus(pr, 1, GateTimelyAccess, WinGated, "urgent"))
		}
		if ld.Priority != "urgent" || ld.Status != Fail || ld.Candidate.Total != 40*30 || ld.Baseline.Total != 0 || cmp.Verdict != NOGO {
			t.Errorf("%s: lost demand %+v, verdict %s", c.name, ld, cmp.Verdict)
		}
	}
}

// W3: thirty urgent orders still unresolved at the end of observation,
// long past the limit: untimely (never dropped), timely access fails;
// the maximum wait is reported (over 100 days). Within the limit at the end they would be
// undecided, and the gate lacks evidence (not a pass).
func TestWitnessUnresolvedUrgentOrders(t *testing.T) {
	p, d := candidatePolicy(t)
	_, pr, c := clinicalCase(t, p, d, nil, func(b synth, r *Record) {
		for _, i := range urgentOrders(r, 20, 50) {
			resolveAs(r, i, OutOpen, 0)
		}
	})
	wantVerdict(t, pr, c, NOGO, "clinical seed reliability")
	if a := urgentAccess(pr); a.Untimely != 30 || a.Undecided != 0 || pr.PerSeed[0].Measures.Clinical.MaxWaitGatedMin["urgent"] < 100*1440 {
		t.Fatalf("%+v", a)
	}
	var a AccessCohort
	o := Order{WantedSec: 0}
	classifyAccess(&a, &o, OutOpen, 0, 48*hour, 47*hour)
	if a.Undecided != 1 || a.Untimely != 0 {
		t.Fatalf("within the limit at the end: %+v", a)
	}
}

// W4: urgent orders rescheduled repeatedly. Displaced three times and
// never imaged: open, untimely. Displaced twice and imaged five days
// after the wanted time: served late, untimely (a rebooking never moves
// the wanted time). Displaced twice and imaged within 40 hours: timely
// (rescheduling is not itself a failure).
func TestWitnessRepeatedRescheduling(t *testing.T) {
	p, d := candidatePolicy(t)
	_, pr, c := clinicalCase(t, p, d, nil, func(b synth, r *Record) {
		os := urgentOrders(r, 20, 50)
		for k, i := range os {
			o := &r.Orders[i]
			w := o.WantedSec
			switch {
			case k < 10:
				o.DisplacementSec = []int64{w + 2*hour, w + daySec, w + 3*daySec}
				resolveAs(r, i, OutOpen, 0)
			case k < 20:
				o.DisplacementSec = []int64{w + 2*hour, w + daySec}
				resolveAs(r, i, OutServed, w+5*daySec)
			default:
				o.DisplacementSec = []int64{w + hour, w + 2*hour}
				resolveAs(r, i, OutServed, w+40*hour)
			}
		}
	})
	wantVerdict(t, pr, c, NOGO, "clinical seed reliability")
	if a := urgentAccess(pr); a.Untimely != 20 || a.Timely+a.Untimely != a.Wanted {
		t.Fatalf("%+v", a)
	}
}

// openSec is the time in [a, b) when the synthetic practice images
// (Mon–Fri 08:00–20:00, Sat 10:00–18:00, the scenario's hours), seconds.
func openSec(a, b int64) int64 {
	hours := [7][2]int64{{8, 20}, {8, 20}, {8, 20}, {8, 20}, {8, 20}, {10, 18}, {0, 0}}
	mid := t0 - 6*hour // Tuesday 2026-10-13 00:00
	s := int64(0)
	for d := floorDiv(a-mid, daySec); mid+d*daySec < b; d++ {
		wd := int((d + 1) % 7) // day 0 is a Tuesday (Monday 0)
		if wd < 0 {
			wd += 7
		}
		o, c := mid+d*daySec+hours[wd][0]*hour, mid+d*daySec+hours[wd][1]*hour
		s += max(0, min(b, c)-max(a, o))
	}
	return s
}

// W5: an urgent order wanted on Saturday 17:00 with no Sunday imaging,
// imaged Monday 08:30: 39.5 calendar hours, 1.5 open hours. Calendar
// clock: timely at 48 hours, untimely at 24. The open-hours clock would
// call it timely at any limit of two hours or more: it hides the weekend.
// Lost instead (no Saturday slot), it is untimely under every clock.
func TestWitnessSaturdayUrgent(t *testing.T) {
	sat := t0 + 4*daySec + 11*hour // Sat 2026-10-17 17:00
	mon := sat + 39*hour + 30*60   // Mon 08:30
	if got := openSec(sat, mon); got != 90*60 {
		t.Fatalf("open time %d s", got)
	}
	E := t0 + 200*daySec
	o := Order{WantedSec: sat, ReceivedSec: sat, ResolvedSec: mon}
	for _, c := range []struct {
		limit          int64
		timely, untime int64
	}{{48 * hour, 1, 0}, {24 * hour, 0, 1}} {
		var a AccessCohort
		classifyAccess(&a, &o, OutServed, sat, c.limit, E)
		if a.Timely != c.timely || a.Untimely != c.untime || a.Wanted != 1 {
			t.Errorf("limit %d h: %+v", c.limit/hour, a)
		}
	}
	var a AccessCohort
	lost := Order{WantedSec: sat, ReceivedSec: sat, ResolvedSec: sat}
	classifyAccess(&a, &lost, OutLost, sat, 48*hour, E)
	if a.Untimely != 1 {
		t.Errorf("lost: %+v", a)
	}
}

// W6: a cancellation the record shows as the patient's ("patient":
// cancellation or no-show; the cause is not recorded). Within the limit:
// withdrawn, outside the denominator. After the limit: untimely, the
// practice had already failed the patient. A no-show the simulator
// rebooks and leaves open, with no resolution time, counts to the end of
// observation: untimely once past the limit.
func TestWitnessPatientCancellation(t *testing.T) {
	E := int64(200 * daySec)
	var a AccessCohort
	for _, at := range []int64{2 * hour, 47 * hour} {
		o := Order{WantedSec: 0, ResolvedSec: at}
		classifyAccess(&a, &o, OutPatient, 0, 48*hour, E)
	}
	late := Order{WantedSec: 0, ResolvedSec: 60 * hour}
	classifyAccess(&a, &late, OutPatient, 0, 48*hour, E)
	rebooked := Order{WantedSec: 0}
	classifyAccess(&a, &rebooked, OutPatient, 0, 48*hour, E)
	if a.Withdrawn != 2 || a.Untimely != 2 || a.Wanted != 4 {
		t.Fatalf("%+v", a)
	}
}

// W7: the observation boundaries. An urgent order wanted an hour before
// the horizon and imaged ten hours after it: in the gated cohort, timely.
// One wanted an hour after the horizon: outside it. One wanted an hour
// before the horizon and never imaged: untimely at the end of observation
// (30 days on). One received before the takeover and imaged an hour after
// it: in the inherited cohort, timed from the takeover, timely.
func TestWitnessObservationBoundaries(t *testing.T) {
	p, _ := candidatePolicy(t)
	b := base("B", 1)
	b.breachPM = map[string]int{}
	r0 := b.record()
	m0 := Measure(p, &r0)
	r := b.record()
	H := r.HorizonEndSec
	add := func(o Order, out string) {
		r.Orders = append(r.Orders, o)
		for i := range r.Captures {
			r.Captures[i].OrderOutcomes = append(r.Captures[i].OrderOutcomes, out)
		}
	}
	add(Order{ID: "B1", Priority: "urgent", Modality: "CT", ReceivedSec: H - 2*hour, WantedSec: H - hour, FirstApptSec: H + 10*hour, ResolvedSec: H + 10*hour}, OutServed)
	add(Order{ID: "B2", Priority: "urgent", Modality: "CT", ReceivedSec: H, WantedSec: H + hour, FirstApptSec: H + 3*hour, ResolvedSec: H + 3*hour}, OutServed)
	add(Order{ID: "B3", Priority: "urgent", Modality: "CT", ReceivedSec: H - 2*hour, WantedSec: H - hour}, OutOpen)
	add(Order{ID: "B4", Priority: "urgent", Modality: "CT", ReceivedSec: t0 - daySec, WantedSec: t0 - 12*hour, FirstApptSec: t0 + hour, ResolvedSec: t0 + hour}, OutServed)
	resync(b, &r)
	m := Measure(p, &r)
	if len(m.Integrity) > 0 {
		t.Fatal(m.Integrity)
	}
	g, g0 := m.Clinical.Access[WinGated]["urgent"], m0.Clinical.Access[WinGated]["urgent"]
	inh, inh0 := m.Clinical.InheritedAccess["urgent"], m0.Clinical.InheritedAccess["urgent"]
	if g.Wanted-g0.Wanted != 2 || g.Timely-g0.Timely != 1 || g.Untimely-g0.Untimely != 1 || inh.Wanted-inh0.Wanted != 1 || inh.Timely-inh0.Timely != 1 {
		t.Fatalf("gated %+v (was %+v), inherited %+v (was %+v)", g, g0, inh, inh0)
	}
}

// W8–W10, the audit's other witnesses under the candidate's values: a
// report 348 h past due (routine limit 14 days: a violation, NO GO for
// clinical safety); six urgent reports eight days late in one week (the
// urgent 24-hour limit and the weekly count of 5: NO GO); four
// service-failing seeds and four other backlog-failing seeds (8 > 4 of
// the shared allowance: NO GO, where v0.2-eval-4 passes).
func TestWitnessAuditUnderCandidate(t *testing.T) {
	p, d := candidatePolicy(t)
	_, pr, c := clinicalCase(t, p, d, nil, func(b synth, r *Record) {
		i := studiesOf(r, "routine", 140, 141)[0]
		finishAt(r, i, r.Studies[i].AcquiredSec+testTargets["routine"]*60+348*hour)
	})
	wantVerdict(t, pr, c, NOGO, "absolute pastDue limit of 20160 min (worst 20880)")
	_, pr, c = clinicalCase(t, p, d, nil, func(b synth, r *Record) {
		// Six urgent studies of one evaluator week (Monday-based, as
		// measureClinical counts them).
		byWeek := map[int64][]int{}
		for _, i := range studiesOf(r, "urgent", 105, 126) {
			w := floorDiv(r.Studies[i].AcquiredSec/daySec+3, 7)
			byWeek[w] = append(byWeek[w], i)
		}
		for _, w := range slices.Sorted(maps.Keys(byWeek)) {
			if len(byWeek[w]) >= 6 {
				for _, i := range byWeek[w][:6] {
					finishAt(r, i, r.Studies[i].AcquiredSec+testTargets["urgent"]*60+8*daySec)
				}
				return
			}
		}
		t.Fatal("no full week")
	})
	wantVerdict(t, pr, c, NOGO, "clinical safety")
	if gateStatus(pr, 1, GateSevereWeek, WinGated, "urgent") != Fail || len(pr.ClinicalSafety.Seeds) != 40 {
		t.Fatal("six urgent reports eight days late")
	}
	mk := func(seed int64) synth {
		b := base("B", seed)
		b.breachPM, b.profitPD = map[string]int{}, 30_000
		if seed <= 4 {
			b.breachPM = map[string]int{"urgent": 200}
		}
		return b
	}
	_, pr, c = clinicalCase(t, p, d, mk, func(b synth, r *Record) {
		if r.Seed >= 5 && r.Seed <= 8 {
			staleBlock(b, r)
		}
	})
	wantVerdict(t, pr, c, NOGO, "clinical seed reliability: fail (8 failing")
}

// W11: missing or contradictory lifecycle data makes the record invalid,
// never a pass or a fail: an order imaged before its wanted time; a
// served outcome with no resolution time; an outcome at an earlier
// capture contradicting the final one after the order was resolved.
func TestWitnessContradictoryLifecycle(t *testing.T) {
	p, _ := candidatePolicy(t)
	for name, change := range map[string]func(r *Record, i int){
		"imaged before wanted": func(r *Record, i int) { r.Orders[i].WantedSec = r.Orders[i].ResolvedSec + hour },
		"served without a resolution": func(r *Record, i int) {
			r.Orders[i].ResolvedSec = 0
		},
		"outcome changed after resolution": func(r *Record, i int) {
			for k := range r.Captures {
				if r.Captures[k].Name == "T90" {
					r.Captures[k].OrderOutcomes[i] = OutLost
				}
			}
		},
	} {
		b := base("B", 3)
		r := b.record()
		change(&r, urgentOrders(&r, 30, 31)[0])
		if m := Measure(p, &r); len(m.Integrity) == 0 {
			t.Errorf("%s: the record is valid", name)
		}
	}
}

// W12: manipulation. Relabelling thirty urgent orders and studies as
// routine moves them out of the urgent cohort, which the clinical gates
// cannot see; v0.2-eval-4's case selection against the paired baseline
// catches the fall in urgent acquisitions: NO GO. Deferring twenty urgent
// appointments past the horizon leaves the orders wanted in the gated
// period served late: untimely, never out of the denominator (timely
// 0.87 < 0.90: NO GO).
func TestWitnessManipulation(t *testing.T) {
	p, d := candidatePolicy(t)
	_, pr, c := clinicalCase(t, p, d, nil, func(b synth, r *Record) {
		for _, i := range urgentOrders(r, 20, 50) {
			at := r.Orders[i].ResolvedSec
			r.Orders[i].Priority = "routine"
			for k := range r.Studies {
				if r.Studies[k].Priority == "urgent" && r.Studies[k].AcquiredSec == at {
					r.Studies[k].Priority = "routine"
				}
			}
		}
	})
	wantVerdict(t, pr, c, NOGO, "case selection")
	if a := urgentAccess(pr); a.Untimely != 0 {
		t.Fatalf("relabelled: %+v", a)
	}
	_, pr, c = clinicalCase(t, p, d, nil, func(b synth, r *Record) {
		for _, i := range urgentOrders(r, 130, 150) {
			resolveAs(r, i, OutServed, r.HorizonEndSec+5*daySec)
		}
	})
	if a := urgentAccess(pr); a.Untimely != 20 {
		t.Fatalf("deferred past the horizon: %+v", a)
	}
	wantVerdict(t, pr, c, NOGO, "clinical seed reliability")
}
