package eval

import (
	"crypto/sha256"
	"fmt"
	"math/rand/v2"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// Slice 5C.4l: order provenance, inherited safety and reserved protocol
// closure, the remediation of the independent 5C.4k audit
// (docs/audits/slice-5c4k-clinical-safety-protocol-lock-audit.md) under
// the owner's decisions of 2026-10-20 (docs/calibration/clinical-5c4l/).
// The audit's witnesses (kept outside the repository) are reproduced here
// as regressions. Synthetic fixtures only.

// candidate5c4jPath is the 5C.4j candidate (the contract without
// inherited work, no provenance reconciliation), kept byte for byte.
const candidate5c4jPath = "testdata/candidate-policy-v5-5c4j.json"

func candidate5c4jPolicy(t *testing.T) (*Policy, string) {
	t.Helper()
	return clinicalFrom(t, candidate5c4jPath)
}

// violation is the 5C.4i I2 situation the audit started from on every
// candidate seed: the last soon order wanted by the horizon, imaged one
// second after it and reported 60 hours past due (an absolute violation
// the follow-up cohort keeps). It returns the order's and its study's
// indexes.
func violation(r *Record) (int, int) {
	i := lastSoonBefore(r)
	id := r.Orders[i].ID
	moveAcquisition(r, i, r.HorizonEndSec+1, 60*hour)
	return i, slices.IndexFunc(r.Studies, func(s Study) bool { return s.Order == id })
}

// dropStudyOf deletes order id's studies; it returns si adjusted for the
// deletions before it.
func dropStudyOf(r *Record, id string, si int) int {
	for k := len(r.Studies) - 1; k >= 0; k-- {
		if r.Studies[k].Order == id {
			r.Studies = slices.Delete(r.Studies, k, k+1)
			if k < si {
				si--
			}
		}
	}
	return si
}

// afterH is the index of the first soon order of modality mod wanted
// after the horizon, other than i, with a study (-1: none).
func afterH(r *Record, i int, mod string) int {
	for j, o := range r.Orders {
		if j != i && o.Priority == "soon" && o.Modality == mod && o.WantedSec > r.HorizonEndSec &&
			slices.ContainsFunc(r.Studies, func(s Study) bool { return s.Order == o.ID }) {
			return j
		}
	}
	return -1
}

func endOf(r *Record) []string { return r.Captures[len(r.Captures)-1].OrderOutcomes }

// K1: the audit's relink-and-retime bypass and every provenance attack
// of the brief. Under the 5C.4l candidate every manipulated record is
// INVALID (order provenance against the paired baseline), every
// legitimate difference of outcome is not; under the 5C.4j candidate the
// audit's bypass is reproduced (GO).
func TestOrderProvenance(t *testing.T) {
	cand, dc := candidatePolicy(t)
	old, do := candidate5c4jPolicy(t)
	type attack struct {
		name    string
		edit    func(r *Record, i, si int)
		want    string // under the 5C.4l candidate
		want5j  string // under the 5C.4j candidate ("": not checked)
		because string // the reason expected among the comparison's
	}
	relinkRetime := func(r *Record, i, si int) {
		j := afterH(r, i, r.Studies[si].Modality)
		o := &r.Orders[j]
		si = dropStudyOf(r, o.ID, si)
		at := r.Studies[si].AcquiredSec
		o.ReceivedSec, o.WantedSec, o.ResolvedSec, o.FirstApptSec = at, at, at, at
		endOf(r)[j] = OutServed
		r.Studies[si].Order = o.ID
		r.Orders[i].ResolvedSec = r.Orders[i].ReceivedSec
		endOf(r)[i] = OutLost
	}
	for _, x := range []attack{
		{"control: the violation kept", func(r *Record, i, si int) {}, NOGO, NOGO, ""},
		{"K1: relinked to a same-request order wanted after H, re-timed consistently, the original lost", relinkRetime, INVALID, GO, "order provenance"},
		{"the original order's wanted time moved to H + 1 s", func(r *Record, i, si int) { r.Orders[i].WantedSec = r.HorizonEndSec + 1 }, INVALID, NOGO, "order provenance"},
		{"the original order's origin changed", func(r *Record, i, si int) { r.Orders[i].Origin += "x" }, INVALID, "", "no trusted provenance"},
		{"order and study relabelled routine", func(r *Record, i, si int) { r.Studies[si].Priority, r.Orders[i].Priority = "routine", "routine" }, INVALID, NOGO, "order provenance"},
		{"order and study relabelled to another modality", func(r *Record, i, si int) { r.Studies[si].Modality, r.Orders[i].Modality = "US", "US" }, INVALID, "", "order provenance"},
		{"two same-request orders' IDs swapped with their studies' links", func(r *Record, i, si int) {
			j := afterH(r, i, r.Studies[si].Modality)
			a, b := r.Orders[i].ID, r.Orders[j].ID
			r.Orders[i].ID, r.Orders[j].ID = b, a
			for k := range r.Studies {
				switch r.Studies[k].Order {
				case a:
					r.Studies[k].Order = b
				case b:
					r.Studies[k].Order = a
				}
			}
		}, INVALID, "", "order provenance"},
		{"an order missing (another soon order and its study deleted)", func(r *Record, i, si int) {
			j := afterH(r, i, r.Studies[si].Modality)
			dropStudyOf(r, r.Orders[j].ID, si)
			r.Orders = slices.Delete(r.Orders, j, j+1)
			for k := range r.Captures {
				r.Captures[k].OrderOutcomes = slices.Delete(r.Captures[k].OrderOutcomes, j, j+1)
			}
		}, INVALID, "", "missing from plan"},
		{"an order duplicated under a new ID (same origin)", func(r *Record, i, si int) {
			o := r.Orders[i]
			o.ID += "-dup"
			o.ResolvedSec = o.ReceivedSec
			r.Orders = append(r.Orders, o)
			for k := range r.Captures {
				r.Captures[k].OrderOutcomes = append(r.Captures[k].OrderOutcomes, OutLost)
			}
		}, INVALID, "", "repeats"},
		{"a candidate-only order (an origin the baseline lacks)", func(r *Record, i, si int) {
			at := r.HorizonEndSec - daySec
			r.Orders = append(r.Orders, Order{ID: "NEW", Origin: "new/1", Priority: "routine", Modality: "XR", ReceivedSec: at, WantedSec: at, ResolvedSec: at})
			for k := range r.Captures {
				out := OutNotDisclosed
				if r.Captures[k].AtSec >= at {
					out = OutLost
				}
				r.Captures[k].OrderOutcomes = append(r.Captures[k].OrderOutcomes, out)
			}
		}, INVALID, "", "no trusted provenance"},
		{"an order outside Gate A/B eligibility (routine, wanted after H) re-timed", func(r *Record, i, si int) {
			for j := range r.Orders {
				if o := &r.Orders[j]; o.Priority == "routine" && o.WantedSec > r.HorizonEndSec {
					o.WantedSec += daySec
					return
				}
			}
			panic("no routine order wanted after H")
		}, INVALID, "", "order provenance"},
		{"a legitimately different outcome: the violating order's report on time instead", func(r *Record, i, si int) {
			s := &r.Studies[si]
			s.FinalSec = s.AcquiredSec + testTargets["soon"]*60
		}, GO, GO, ""},
		{"a legitimately different outcome: another order lost instead of served", func(r *Record, i, si int) {
			j := afterH(r, i, r.Studies[si].Modality)
			dropStudyOf(r, r.Orders[j].ID, si)
			r.Orders[j].ResolvedSec = r.Orders[j].ReceivedSec
			endOf(r)[j] = OutLost
		}, NOGO, NOGO, ""},
	} {
		for _, pol := range []struct {
			name, d, want string
			p             *Policy
		}{{"5C.4l", dc, x.want, cand}, {"5C.4j", do, x.want5j, old}} {
			if pol.want == "" {
				continue
			}
			_, c := boundaryCase(t, pol.p, pol.d, nil, func(b synth, r *Record) {
				i, si := violation(r)
				x.edit(r, i, si)
			})
			t.Logf("%s %-95s %-8s %s", pol.name, x.name, c.Verdict, strings.Join(firstReasons(c.Reasons, 1), ""))
			if c.Verdict != pol.want {
				t.Errorf("%s, %s: %s, want %s; %v", pol.name, x.name, c.Verdict, pol.want, firstReasons(c.Reasons, 3))
			}
			if pol.name == "5C.4l" && x.because != "" && !slices.ContainsFunc(c.Reasons, func(s string) bool { return strings.Contains(s, x.because) }) {
				t.Errorf("%s: no reason with %q: %v", x.name, x.because, firstReasons(c.Reasons, 3))
			}
		}
	}
}

func firstReasons(xs []string, n int) []string { return xs[:min(n, len(xs))] }

// The limitation the provenance contract does not close, shown rather
// than assumed: a study's link to its order is an outcome of the
// candidate's record. Two same-request orders, one wanted by the
// horizon (the violation) and one wanted after it, both imaged after the
// later wanted time, can exchange their studies and resolutions. Every
// immutable request still reconciles, both records are internally
// consistent, and the violation leaves the follow-up cohort: GO. Closing
// it needs a decision (protocol §7, the maintainer's response to 5C.4k):
// this test fails when that changes.
func TestStudySwapLimitation(t *testing.T) {
	p, d := candidatePolicy(t)
	_, c := boundaryCase(t, p, d, nil, func(b synth, r *Record) {
		i, si := violation(r)
		j := afterH(r, i, r.Studies[si].Modality)
		sj := slices.IndexFunc(r.Studies, func(s Study) bool { return s.Order == r.Orders[j].ID })
		// The violation acquired after j's wanted time: a day after H.
		at := r.Orders[j].WantedSec + hour
		moveAcquisition(r, i, at, 60*hour)
		si = slices.IndexFunc(r.Studies, func(s Study) bool { return s.Order == r.Orders[i].ID })
		sj = slices.IndexFunc(r.Studies, func(s Study) bool { return s.Order == r.Orders[j].ID })
		r.Studies[si].Order, r.Studies[sj].Order = r.Orders[j].ID, r.Orders[i].ID
		r.Orders[i].ResolvedSec, r.Orders[j].ResolvedSec = r.Studies[sj].AcquiredSec, r.Studies[si].AcquiredSec
	})
	t.Logf("study swap: %s %v", c.Verdict, firstReasons(c.Reasons, 2))
	if c.Verdict != GO {
		t.Fatalf("the swap limitation changed (%s): update the protocol's limitations and this test", c.Verdict)
	}
	// The unswapped control is NO GO: the swap is what removes it.
	_, c = boundaryCase(t, p, d, nil, func(b synth, r *Record) {
		i, si := violation(r)
		j := afterH(r, i, r.Studies[si].Modality)
		moveAcquisition(r, i, r.Orders[j].WantedSec+hour, 60*hour)
	})
	if c.Verdict != NOGO {
		t.Fatalf("control: %s", c.Verdict)
	}
}

// inheritedSet adds n inherited soon CT orders, received the day before
// the takeover and open at it, each imaged on day 1 + k minutes with its
// study, reported on time; an inherited order's request is the same in
// both plans (one takeover state).
func inheritedSet(n int) func(b synth, r *Record) {
	return func(b synth, r *Record) {
		for k := range n {
			id := fmt.Sprintf("INH%d", k)
			at := t0 + daySec + int64(k)*60
			r.Orders = append(r.Orders, Order{ID: id, Origin: "inh/" + id, Priority: "soon", Modality: "CT", ReceivedSec: t0 - daySec, WantedSec: t0 - daySec,
				FirstApptSec: at - 600, ResolvedSec: at})
			for c := range r.Captures {
				out := OutServed
				if r.Captures[c].AtSec < at {
					out = OutOpen
				}
				r.Captures[c].OrderOutcomes = append(r.Captures[c].OrderOutcomes, out)
			}
			r.Studies = append(r.Studies, Study{ID: "S" + id, Order: id, Priority: "soon", Modality: "CT", AcquiredSec: at, EnteredSec: at,
				ClaimedSec: at + 60, ReadStartSec: at + 60, FinalSec: at + hour})
		}
		slices.SortStableFunc(r.Studies, func(x, y Study) int { return int(x.AcquiredSec - y.AcquiredSec) })
	}
}

// K2: inherited orders imaged after the takeover stay under clinical
// safety. The audit's bypasses (study deleted, relabelled routine, wanted
// time moved past the horizon) and further cases, on 40 inherited orders
// in both plans; the 5C.4j candidate (inherited exemption, no
// provenance) is shown accepting the bypasses.
func TestInheritedSafety(t *testing.T) {
	cand, dc := candidatePolicy(t)
	old, do := candidate5c4jPolicy(t)
	// target re-times inherited order INH0 to acquisition at(r), finished
	// `past` after due (negative: unfinished), in the candidate only.
	target := func(at func(r *Record) int64, past int64) func(r *Record) {
		return func(r *Record) {
			i := slices.IndexFunc(r.Orders, func(o Order) bool { return o.ID == "INH0" })
			moveAcquisition(r, i, at(r), past)
			for c := range r.Captures {
				out := OutServed
				if r.Captures[c].AtSec < at(r) {
					out = OutOpen
				}
				r.Captures[c].OrderOutcomes[i] = out
			}
		}
	}
	day1 := func(r *Record) int64 { return t0 + daySec }
	afterHorizon := func(r *Record) int64 { return r.HorizonEndSec + 1 }
	inh := func(r *Record) (int, int) {
		i := slices.IndexFunc(r.Orders, func(o Order) bool { return o.ID == "INH0" })
		return i, slices.IndexFunc(r.Studies, func(s Study) bool { return s.Order == "INH0" })
	}
	del := func(r *Record) {
		_, si := inh(r)
		r.Studies = slices.Delete(r.Studies, si, si+1)
	}
	for _, x := range []struct {
		name         string
		setup, edit  func(r *Record)
		want, want5j string
	}{
		{"inherited orders open at the takeover, served after it, on time", nil, nil, GO, GO},
		{"control: imaged day 1, reported 60 h past due", target(day1, 60*hour), nil, NOGO, NOGO},
		{"its study deleted, the order still served", target(day1, 60*hour), del, INVALID, GO},
		{"order and study relabelled routine", target(day1, 60*hour), func(r *Record) {
			i, si := inh(r)
			r.Orders[i].Priority, r.Studies[si].Priority = "routine", "routine"
		}, INVALID, GO},
		{"control: imaged H + 1 s, reported 60 h past due (after the horizon)", target(afterHorizon, 60*hour), nil, NOGO, NOGO},
		{"the same, study deleted", target(afterHorizon, 60*hour), del, INVALID, GO},
		{"the same, wanted time moved to H + 1 s", target(afterHorizon, 60*hour), func(r *Record) {
			i, _ := inh(r)
			r.Orders[i].WantedSec = r.HorizonEndSec + 1
		}, INVALID, GO},
		{"control: imaged day 1, unfinished at E (overdue)", target(day1, -1), nil, NOGO, NOGO},
		{"the same, study deleted", target(day1, -1), del, INVALID, GO},
		{"imaged H + 1 s, still awaiting its report at E (not yet observable)", target(func(r *Record) int64 { return r.ObservedUntilSec - 10*hour }, -1), nil, NOGO, NOGO},
		{"its study duplicated (a second row, new ID)", target(day1, 60*hour), func(r *Record) {
			_, si := inh(r)
			s := r.Studies[si]
			s.ID += "-dup"
			r.Studies = append(r.Studies, s)
		}, INVALID, INVALID},
		{"its study relinked to another inherited order", target(day1, 60*hour), func(r *Record) {
			_, si := inh(r)
			r.Studies[si].Order = "INH1"
		}, INVALID, INVALID},
		{"an inherited order deleted from the candidate's record", nil, func(r *Record) {
			i, si := inh(r)
			r.Studies = slices.Delete(r.Studies, si, si+1)
			r.Orders = slices.Delete(r.Orders, i, i+1)
			for c := range r.Captures {
				r.Captures[c].OrderOutcomes = slices.Delete(r.Captures[c].OrderOutcomes, i, i+1)
			}
		}, INVALID, GO},
	} {
		for _, pol := range []struct {
			name, d, want string
			p             *Policy
		}{{"5C.4l", dc, x.want, cand}, {"5C.4j", do, x.want5j, old}} {
			pr, c := boundaryCase(t, pol.p, pol.d, inheritedSet(40), func(b synth, r *Record) {
				inheritedSet(40)(b, r)
				if x.setup != nil {
					x.setup(r)
				}
				if x.edit != nil {
					x.edit(r)
				}
			})
			t.Logf("%s %-75s valid %-5v %-8s %s", pol.name, x.name, pr.Valid, c.Verdict, strings.Join(firstReasons(c.Reasons, 1), ""))
			if c.Verdict != pol.want {
				t.Errorf("%s, %s: %s, want %s; %v %v", pol.name, x.name, c.Verdict, pol.want, firstReasons(pr.Integrity, 2), firstReasons(c.Reasons, 2))
			}
		}
	}
}

// The contract with inherited work, on one record: a served inherited
// order needs its post-takeover study; a study acquired before the
// takeover stays exempt (its order may be absent).
func TestContractInheritedUnit(t *testing.T) {
	r := base("B", 1).record()
	inheritedSet(3)(base("B", 1), &r)
	end := endCapture(&r)
	var msgs []string
	bad := func(f string, a ...any) { msgs = append(msgs, fmt.Sprintf(f, a...)) }
	if !checkOrderStudies(&r, end, true, bad) || !checkOrderStudies(&r, end, false, bad) {
		t.Fatalf("consistent record refused: %v", msgs)
	}
	r.Studies = slices.DeleteFunc(r.Studies, func(s Study) bool { return s.Order == "INH1" })
	if checkOrderStudies(&r, end, true, bad) || !checkOrderStudies(&r, end, false, func(string, ...any) {}) {
		t.Fatalf("a served inherited order without its study: strict accepts or 5C.4j refuses (%v)", msgs)
	}
	r = base("B", 1).record()
	acq := t0 - 3*hour
	r.Studies = append([]Study{{ID: "PRE", Priority: "soon", Modality: "CT", AcquiredSec: acq, EnteredSec: acq, ClaimedSec: acq + 60, ReadStartSec: acq + 60, FinalSec: t0 + hour}}, r.Studies...)
	if !checkOrderStudies(&r, endCapture(&r), true, bad) {
		t.Fatalf("a study before the takeover without its order refused: %v", msgs)
	}
	// A repeated ID of two orders without studies (lost at receipt):
	// refused as a repeated ID, not through a study's link.
	r = base("B", 1).record()
	for k := range 2 {
		at := t0 + daySec + int64(k)
		r.Orders = append(r.Orders, Order{ID: "LOST", Origin: fmt.Sprintf("lost/%d", k), Priority: "routine", Modality: "XR", ReceivedSec: at, WantedSec: at, ResolvedSec: at})
		for c := range r.Captures {
			r.Captures[c].OrderOutcomes = append(r.Captures[c].OrderOutcomes, OutLost)
		}
	}
	msgs = nil
	if checkOrderStudies(&r, endCapture(&r), true, bad) || !strings.Contains(strings.Join(msgs, " "), "its ID repeats") {
		t.Fatalf("a repeated order ID: %v", msgs)
	}
}

// orderProvenance on its own: every failure kind, and outcomes ignored.
func TestOrderProvenanceUnit(t *testing.T) {
	a, b := base("B", 1).record(), base("A", 1).record()
	if f := orderProvenance(&a, &b); len(f) != 0 {
		t.Fatalf("paired records: %v", f)
	}
	a.Orders[3].ResolvedSec += hour
	a.Orders[3].FirstApptSec += hour
	a.Captures[len(a.Captures)-1].OrderOutcomes[3] = OutLost
	if f := orderProvenance(&a, &b); len(f) != 0 {
		t.Fatalf("outcomes compared: %v", f)
	}
	for name, edit := range map[string]func(r *Record){
		"ID":       func(r *Record) { r.Orders[2].ID = "other" },
		"origin":   func(r *Record) { r.Orders[2].Origin = "other" },
		"received": func(r *Record) { r.Orders[2].ReceivedSec-- },
		"wanted":   func(r *Record) { r.Orders[2].WantedSec++ },
		"priority": func(r *Record) { r.Orders[2].Priority = "routine" },
		"modality": func(r *Record) { r.Orders[2].Modality = "PET" },
		"walk-in":  func(r *Record) { r.Orders[2].WalkIn = !r.Orders[2].WalkIn },
		"missing":  func(r *Record) { r.Orders = r.Orders[1:] },
		"extra":    func(r *Record) { r.Orders = append(r.Orders, Order{ID: "X", Origin: "x/1"}) },
		"repeated": func(r *Record) { r.Orders = append(r.Orders, r.Orders[0]) },
		"empty":    func(r *Record) { r.Orders[0].Origin = "" },
	} {
		c := base("B", 1).record()
		edit(&c)
		if f := orderProvenance(&c, &b); len(f) == 0 {
			t.Errorf("%s: reconciled", name)
		}
		if f := orderProvenance(&b, &c); len(f) == 0 {
			t.Errorf("%s (in the paired plan): reconciled", name)
		}
	}
	// Records without origins (written before 0.2.0-dev.18): never
	// reconciled, even alike.
	x, y := base("B", 1).record(), base("A", 1).record()
	for i := range x.Orders {
		x.Orders[i].Origin, y.Orders[i].Origin = "", ""
	}
	if f := orderProvenance(&x, &y); len(f) == 0 {
		t.Error("records without origins reconciled")
	}
	// One order without an origin in both records alike: no identity,
	// never matched on the empty key.
	x, y = base("B", 1).record(), base("A", 1).record()
	x.Orders[0].Origin, y.Orders[0].Origin = "", ""
	if f := orderProvenance(&x, &y); len(f) == 0 {
		t.Error("an order without an origin in both records reconciled")
	}
}

// K4: a criterion with nothing judged is never PASS, and the output
// records what it judged.
func TestReservedCriteriaZeroEvidence(t *testing.T) {
	m := loadManifest(t)
	tol, err := manifestTolerances(m)
	if err != nil {
		t.Fatal(err)
	}
	full := func(s *manScenario) int { return s.Seeds }
	ls := fakeRun(m)
	witnessesMet(ls)
	r := reservedCriteria(m, ls, tol, full)
	if r["invocation"].Status != "confirmatory" {
		t.Fatalf("full run: %v", r["invocation"])
	}
	criteria := []string{"T1", "T2-conformance", "T2G", "T3", "T4", "T5a", "T5b", "T6", "T7", "T8"}
	for _, c := range []string{"T1", "T2-conformance", "T2G", "T3", "T4", "T6", "T7", "T8"} {
		if r[c].Status != Pass {
			t.Fatalf("full run: %s %s", c, r[c].Status)
		}
	}
	// Nothing at all.
	r = reservedCriteria(m, nil, tol, full)
	for _, c := range append(criteria, "T2") {
		if r[c].Status == Pass {
			t.Errorf("no lines: %s PASS", c)
		}
	}
	// Every scenario declared absent (a filter selecting nothing): every
	// criterion INCONCLUSIVE, never PASS, and the invocation a development
	// check.
	r = reservedCriteria(m, nil, tol, func(*manScenario) int { return 0 })
	for _, c := range append(criteria, "T2") {
		if r[c].Status != "inconclusive" {
			t.Errorf("all declared absent: %s %s", c, r[c].Status)
		}
	}
	if r["invocation"].Status != "development" {
		t.Errorf("all declared absent: invocation %s", r["invocation"].Status)
	}
	// One kind of scenario declared absent: its criteria INCONCLUSIVE.
	for c, kind := range map[string]string{"T1": "T1", "T3": "T1", "T6": "T1", "T7": "T1", "T2G": "T2G", "T4": "T4", "T5a": "T5", "T5b": "T5", "T2-conformance": "T2"} {
		seeds := func(s *manScenario) int {
			if slices.Contains(s.Criteria, kind) {
				return 0
			}
			return s.Seeds
		}
		var keep []groupResult
		for _, g := range ls {
			if s := manScenarioOf(m, g.Scenario); s != nil && !slices.Contains(s.Criteria, kind) {
				keep = append(keep, g)
			}
		}
		r := reservedCriteria(m, keep, tol, seeds)
		if r[c].Status != "inconclusive" || r["T8"].Status != "inconclusive" {
			t.Errorf("%s scenarios declared absent: %s %s, T8 %s", kind, c, r[c].Status, r["T8"].Status)
		}
	}
	// Coverage complete, yet nothing to judge: a manifest without any
	// witness scenario (T4 has no member), and one without scenarios at
	// all (T8 has no line). Never PASS.
	noWit := *m
	noWit.Scenarios = nil
	for _, s := range m.Scenarios {
		if !slices.Contains(s.Criteria, "T4") {
			noWit.Scenarios = append(noWit.Scenarios, s)
		}
	}
	var lw []groupResult
	for _, g := range ls {
		if manScenarioOf(&noWit, g.Scenario) != nil {
			lw = append(lw, g)
		}
	}
	if r := reservedCriteria(&noWit, lw, tol, full); r["T4"].Status != "inconclusive" || r["T1"].Status != Pass {
		t.Errorf("no witness scenario: T4 %s, T1 %s", r["T4"].Status, r["T1"].Status)
	}
	none := *m
	none.Scenarios = nil
	if r := reservedCriteria(&none, nil, tol, full); r["T8"].Status != "inconclusive" {
		t.Errorf("no scenario and no line: T8 %s", r["T8"].Status)
	}
	// A reduced invocation is a development check, whatever it shows.
	r = reservedCriteria(m, ls, tol, func(s *manScenario) int { return min(s.Seeds, 40) })
	if r["invocation"].Status != "development" {
		t.Errorf("40 seeds: invocation %s", r["invocation"].Status)
	}
}

func manScenarioOf(m *manifestDoc, id string) *manScenario {
	for i := range m.Scenarios {
		if m.Scenarios[i].ID == id {
			return &m.Scenarios[i]
		}
	}
	return nil
}

// The manifest pins the candidate it is judged under (audit 5C.4k §12):
// its policy digest is the candidate file's, and the harness evaluates
// under the candidate's clinical rules.
func TestManifestPinsCandidate(t *testing.T) {
	m := loadManifest(t)
	raw, err := os.ReadFile("../../" + m.Policy["path"])
	if err != nil {
		t.Fatal(err)
	}
	if d := fmt.Sprintf("%x", sha256.Sum256(raw)); m.Policy["sha256"] != d {
		t.Fatalf("manifest pins %s, the candidate is %s", m.Policy["sha256"], d)
	}
	if m.Version != 3 {
		t.Fatalf("manifest version %d", m.Version)
	}
	cand, _ := candidatePolicy(t)
	if c := manifestPolicy(t).Clinical; c.OrderProvenance != ProvenancePairedOrigin || c.OrderStudyIntegrity != IntegrityContractInherited ||
		!reflect.DeepEqual(c, cand.Clinical) {
		t.Fatalf("the harness's clinical rules are not the candidate's")
	}
	for name, q := range manPractices() {
		if !q.SharedHistory || !q.PrefixStable || !q.ExogenousModality {
			t.Errorf("practice %s: shared history %v, prefix-stable %v, exogenous modality %v", name, q.SharedHistory, q.PrefixStable, q.ExogenousModality)
		}
	}
}

// The fidelity model's shared history (Slice 5C.4l): with it, every
// manifest pair's orders reconcile by origin (the same inherited orders,
// the same requests), and the studies acquired before the takeover are
// the baseline's; without it, a candidate on its own booking stream has
// another history before the takeover, and the reconciliation fails. On
// the suite's debug family PCG(1, ·), never the reserved one.
func TestSharedHistoryReconciles(t *testing.T) {
	if testing.Short() {
		t.Skip("short")
	}
	m := loadManifest(t)
	pol := manifestPolicy(t)
	practices := manPractices()
	preStudies := 0
	for _, id := range []string{"S-A85", "S-Cs92", "D-85", "IL10-85", "IK-85", "Wg85", "W-MARGIN-AT"} {
		s := manScenarioOf(m, id)
		for i := range 2 {
			base, cand, info := manPair(s, 1, i, pol, practices)
			if info.Error != "" {
				t.Fatalf("%s seed %d: %s", id, i, info.Error)
			}
			if f := orderProvenance(cand, base); len(f) > 0 {
				t.Fatalf("%s seed %d: %v", id, i, f)
			}
			pre := func(r *Record) []Study {
				var out []Study
				for _, x := range r.Studies {
					if x.AcquiredSec <= r.TakeoverSec {
						x.FinalSec, x.ClaimedSec, x.ReadStartSec, x.Remote = 0, 0, 0, false // reading is the practice's
						out = append(out, x)
					}
				}
				return out
			}
			pb, pc := pre(base), pre(cand)
			if !reflect.DeepEqual(pb, pc) {
				t.Fatalf("%s seed %d: the studies before the takeover differ (%d and %d)", id, i, len(pb), len(pc))
			}
			preStudies += len(pb)
			inh := 0
			for _, o := range cand.Orders {
				if o.ReceivedSec <= cand.TakeoverSec {
					inh++
				}
			}
			if inh == 0 {
				t.Fatalf("%s seed %d: no inherited order", id, i)
			}
			t.Logf("%s seed %d: %d orders reconciled, %d inherited", id, i, len(cand.Orders), inh)
		}
	}
	if preStudies == 0 {
		t.Fatal("no study before the takeover in any pair: nothing compared")
	}
	// A candidate practice whose imaging capacity differs from its
	// paired baseline's before the takeover too (half, from the warm-up's
	// first day): its bookings before the takeover are still the
	// baseline's.
	{
		s := manScenarioOf(m, "S-A85")
		qb, qc := practices["A85"], practices["A85"]
		qc.ImgCap = []ocCap{{From: -fidWarmup, Factor: 0.5}}
		pcg := func(idx uint64) ocRNG { return ocRNG{rand.New(rand.NewPCG(1, idx))} }
		bb, cb := streamIndex(roleBaselineBooking, s.K, 0), streamIndex(roleCandidateBooking, s.K, 0)
		base, _ := qb.recordPaired(pcg(0), pcg(bb), pol)
		cand, _ := qc.recordTakeover(pcg(0), pcg(bb), pcg(cb), &qb, pol)
		base.Plan, cand.Plan = "B", "C"
		if f := orderProvenance(cand, base); len(f) > 0 {
			t.Fatalf("half capacity before the takeover: %v", f)
		}
	}
	// Without the shared history the inherited orders differ.
	s := manScenarioOf(m, "S-A85")
	q := practices["A85"]
	q.SharedHistory = false
	pcg := func(idx uint64) ocRNG { return ocRNG{rand.New(rand.NewPCG(1, idx))} }
	base, _ := q.recordPaired(pcg(0), pcg(streamIndex(roleBaselineBooking, s.K, 0)), pol)
	cand, _ := q.recordPaired(pcg(0), pcg(streamIndex(roleCandidateBooking, s.K, 0)), pol)
	base.Plan, cand.Plan = "B", "C"
	if f := orderProvenance(cand, base); len(f) == 0 {
		t.Fatal("independent histories reconcile: the shared history is not needed")
	}
}

// The policy accepts orderProvenance only as pairedOrigin and only with
// the contract for inherited work; contractInherited only with the
// follow-up cohort.
func TestPolicyProvenanceValidation(t *testing.T) {
	p, _ := candidatePolicy(t)
	if err := p.validate(); err != nil {
		t.Fatal(err)
	}
	for name, f := range map[string]func(c *Clinical){
		"provenance unknown":                 func(c *Clinical) { c.OrderProvenance = "origin" },
		"provenance with the 5C.4j contract": func(c *Clinical) { c.OrderStudyIntegrity = IntegrityContract },
		"provenance without a contract":      func(c *Clinical) { c.OrderStudyIntegrity = "" },
		"contractInherited without the follow-up cohort": func(c *Clinical) {
			c.OrderProvenance, c.SafetyCohort, c.FollowUpDays = "", "", 0
		},
	} {
		q := *p
		c := *p.Clinical
		f(&c)
		q.Clinical = &c
		if err := q.validate(); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

// Coverage: a provenance failure outside the identity witnesses' first
// seeds is a construction failure (T8 and the scenario's criteria
// INCONCLUSIVE), never a judged rejection.
func TestProvenanceCoverage(t *testing.T) {
	m := loadManifest(t)
	tol, _ := manifestTolerances(m)
	full := func(s *manScenario) int { return s.Seeds }
	ls := fakeRun(m)
	witnessesMet(ls)
	for i := range ls {
		if ls[i].Scenario == "S-B85" && ls[i].Group == 7 {
			ls[i].Provenance = []int64{int64(ls[i].Pairs[3].Seed)}
			ls[i].Rejected, ls[i].Rules = true, []string{"provenance"}
		}
	}
	r := reservedCriteria(m, ls, tol, full)
	if r["T8"].Status != "inconclusive" || r["T3"].Status == Pass {
		t.Fatalf("a stable group's provenance failure: T8 %s, T3 %s", r["T8"].Status, r["T3"].Status)
	}
}
