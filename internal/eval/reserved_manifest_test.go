package eval

import (
	"bufio"
	"bytes"
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/version"
)

// The reserved protocol's executable scenario manifest (Slices 5C.4h and 5C.4j,
// docs/calibration/clinical-5c4l/scenario-manifest.json): every
// scenario's practices, paired demand, booking and injection streams,
// injection, seeds and groups, built here exactly as the manifest says
// and judged by the production evaluator. The reserved family is refused
// in this build: the manifest runs only on a non-reserved family (the
// development checks, PCG(20261021, ·)).

const manifestPath = "../../docs/calibration/clinical-5c4l/scenario-manifest.json"

// reservedFamily is the confirmatory calibration's family, never run by
// this build.
const reservedFamily = 20261201

// devManifestFamily is the construction checks' family.
const devManifestFamily = 20261021

// The stream roles: seed i of scenario k draws its demand from PCG(F, 2i)
// (every scenario: paired demand), and each role from PCG(F, role +
// 100 000 k + i), k < 100, i < 100 000, so that no two roles, scenarios
// or seeds share a stream.
const (
	roleBaselineBooking   = 10_000_000
	roleCandidateBooking  = 20_000_000
	roleCandidateInject   = 30_000_000
	roleBaselineInjection = 40_000_000
)

func streamIndex(role uint64, k, i int) uint64 { return role + 100_000*uint64(k) + uint64(i) }

type manInject struct {
	// Kind: quota (⌈effect × N⌉ of the eligible orders of the priority
	// made lost, uniformly among those served), bernoulli (each served
	// eligible order lost with probability effect × N ÷ served: an
	// expected increase of exactly effect), all (every served one), count
	// (Count of them, as Outcome), or a witness operation.
	Kind     string  `json:"kind"`
	Priority string  `json:"priority,omitempty"`
	Effect   float64 `json:"effect,omitempty"`
	Count    int     `json:"count,omitempty"`
	Outcome  string  `json:"outcome,omitempty"` // count: lost (default) or patient
	// HarmProbability: the seed is harmed (injected) when its injection
	// stream's first draw is below it; 0: every seed.
	HarmProbability float64 `json:"harmProbability,omitempty"`
	// OnVolume: only seeds of this demand volume.
	OnVolume float64 `json:"onVolume,omitempty"`
	// GroupFirstOnly: only the first seed of each group of 40.
	GroupFirstOnly bool `json:"groupFirstSeedOnly,omitempty"`
}

type manSide struct {
	Practice    string    `json:"practice"`
	VolumeCycle []float64 `json:"volumeCycle,omitempty"` // seed i's demand volume: [i mod len]
	// Booking (candidate): "independent" (its own stream), or
	// "baseline" (the baseline's: the records identical before injection).
	Booking string     `json:"booking,omitempty"`
	Inject  *manInject `json:"inject,omitempty"`
}

type manScenario struct {
	K          int      `json:"k"`
	ID         string   `json:"id"`
	Group      string   `json:"group"`
	Criteria   []string `json:"criteria"`
	Mechanism  string   `json:"mechanism"`
	Seeds      int      `json:"seeds"`
	Base       *int     `json:"baseScenario,omitempty"` // reuse that scenario's pairs (seeds 0 … Seeds−1)
	Replicates int      `json:"injectionReplicates,omitempty"`
	Baseline   manSide  `json:"baseline"`
	Candidate  manSide  `json:"candidate"`
	Effect     string   `json:"expectedEffect"`
	Metric     string   `json:"primaryMetric"`
	Acceptance string   `json:"acceptanceRule"`
	Failure    string   `json:"failureConsequence"`
}

type manifestDoc struct {
	Schema     string            `json:"schema"`
	Version    int               `json:"version"`
	Status     string            `json:"status"`
	Policy     map[string]string `json:"policy"`
	Streams    map[string]string `json:"streams"`
	GroupSize  int               `json:"groupSize"`
	Practices  map[string]string `json:"practices"`
	Scenarios  []manScenario     `json:"scenarios"`
	Criteria   json.RawMessage   `json:"criteria"`
	Tolerances json.RawMessage   `json:"t5Tolerances"`
}

func loadManifest(t testing.TB) *manifestDoc {
	t.Helper()
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var m manifestDoc
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&m); err != nil {
		t.Fatal(err)
	}
	return &m
}

// manPractices are the manifest's practices: the fidelity scenarios'
// (ocFidelityScenarios) and IL40-85, at the aligned takeover, with
// exogenous modalities (each order's modality part of its demand, as in
// the simulator: case selection's modality strata then pair).
func manPractices() map[string]ocFidelity {
	out := map[string]ocFidelity{}
	for _, d := range ocFidelityScenarios() {
		q := d.q
		q.Takeover, q.ExogenousModality, q.PrefixStable, q.SharedHistory = fidAlignedTakeover, true, true, true
		out[d.name] = q
	}
	q := out["A85"]
	q.ImgCap = []ocCap{{From: 30, Factor: 0.6}}
	out["IL40-85"] = q
	return out
}

// demandModel is what a practice's demand draws depend on: two practices
// pair only when it is the same.
func demandModel(q ocFidelity) [5]float64 {
	s, m := 0.0, 0.0
	if q.Seasonal {
		s = 1
	}
	if q.ExogenousModality {
		m = 1
	}
	return [5]float64{s, q.DayCV, cmp.Or(q.Volume, 1), float64(q.Takeover), m}
}

// pairInfo is what the construction did on one seed.
type pairInfo struct {
	Seed     int     `json:"seed"`
	Volume   float64 `json:"volume,omitempty"`
	Harmed   bool    `json:"harmed,omitempty"`
	Injected int     `json:"injected,omitempty"` // candidate orders or studies changed
	BaseInj  int     `json:"baselineInjected,omitempty"`
	// BaseEligible: the baseline injection's eligible orders.
	BaseEligible int    `json:"baselineEligible,omitempty"`
	Eligible     int    `json:"eligible,omitempty"` // the injected priority's eligible orders
	Served       int    `json:"served,omitempty"`
	Error        string `json:"constructionError,omitempty"`
}

// manPair builds scenario s's pair on seed i of family F.
func manPair(s *manScenario, F uint64, i int, pol *Policy, practices map[string]ocFidelity) (base, cand *Record, info pairInfo) {
	info.Seed = i
	qb, qc := practices[s.Baseline.Practice], practices[s.Candidate.Practice]
	if vc := s.Baseline.VolumeCycle; len(vc) > 0 {
		qb.Volume = vc[i%len(vc)]
		info.Volume = qb.Volume
	}
	if vc := s.Candidate.VolumeCycle; len(vc) > 0 {
		qc.Volume = vc[i%len(vc)]
	}
	if demandModel(qb) != demandModel(qc) {
		info.Error = "the two practices' demand models differ: not paired"
		return nil, nil, info
	}
	pcg := func(idx uint64) ocRNG { return ocRNG{rand.New(rand.NewPCG(F, idx))} }
	k := s.K
	if s.Base != nil {
		k = *s.Base
	}
	bb := streamIndex(roleBaselineBooking, k, i)
	cb := streamIndex(roleCandidateBooking, k, i)
	if s.Candidate.Booking == "baseline" {
		cb = bb
	}
	base, _ = qb.recordPaired(pcg(uint64(2*i)), pcg(bb), pol)
	if qc.SharedHistory && cb != bb {
		// The takeover state is the baseline's (Slice 5C.4l): its practice
		// and booking stream before the takeover, the candidate's after.
		cand, _ = qc.recordTakeover(pcg(uint64(2*i)), pcg(bb), pcg(cb), &qb, pol)
	} else {
		cand, _ = qc.recordPaired(pcg(uint64(2*i)), pcg(cb), pol)
	}
	base.Seed, cand.Seed = int64(i), int64(i)
	base.Plan, cand.Plan = "B", "C"
	if in := s.Baseline.Inject; in != nil {
		if in.Priority != "" {
			info.BaseEligible = len(manEligible(base, pol, in.Priority, false))
		}
		n, _, err := manApply(base, in, pol, pcg(streamIndex(roleBaselineInjection, s.K, i)), i, info.Volume)
		info.BaseInj = n
		if err != "" {
			info.Error = "baseline: " + err
		}
	}
	if in := s.Candidate.Inject; in != nil && s.Replicates == 0 {
		info = manInjectCandidate(cand, in, pol, pcg(streamIndex(roleCandidateInject, s.K, i)), info)
	}
	return base, cand, info
}

func manInjectCandidate(cand *Record, in *manInject, pol *Policy, r ocRNG, info pairInfo) pairInfo {
	if in.Priority != "" {
		// Before the injection: its denominator and the orders it draws from.
		info.Eligible, info.Served = len(manEligible(cand, pol, in.Priority, false)), len(manEligible(cand, pol, in.Priority, true))
	}
	n, harmed, err := manApply(cand, in, pol, r, info.Seed, info.Volume)
	info.Injected, info.Harmed = n, harmed
	if err != "" && info.Error == "" {
		info.Error = err
	}
	return info
}

// manEligible: record r's eligible orders of priority q (Gate B's: received
// after the takeover, wanted in the gated period), served only or all, in
// record order.
func manEligible(r *Record, pol *Policy, q string, served bool) []int {
	end := endCapture(r)
	b := windows(pol, r)[WinGated]
	var out []int
	for i, o := range r.Orders {
		if o.Priority != q || o.ReceivedSec <= r.TakeoverSec || o.WantedSec <= b[0] || o.WantedSec > b[1] {
			continue
		}
		if !served || end.OrderOutcomes[i] == OutServed {
			out = append(out, i)
		}
	}
	return out
}

// manApply applies one injection to r (r's own randomness from rng):
// the number of orders or studies changed, whether the seed is harmed,
// and a construction error (an infeasible injection: never silently
// reduced).
func manApply(r *Record, in *manInject, pol *Policy, rng ocRNG, seed int, volume float64) (int, bool, string) {
	harmed := true
	if in.HarmProbability > 0 {
		harmed = rng.Float64() < in.HarmProbability
	}
	if !harmed || (in.OnVolume != 0 && volume != in.OnVolume) || (in.GroupFirstOnly && seed%40 != 0) {
		return 0, false, ""
	}
	outs := slices.Clone(endCapture(r).OrderOutcomes)
	n := 0
	errs := ""
	switch in.Kind {
	case "quota", "bernoulli", "all", "count":
		all := manEligible(r, pol, in.Priority, false)
		served := manEligible(r, pol, in.Priority, true)
		var pick []int
		switch in.Kind {
		case "quota":
			k := int(math.Ceil(in.Effect*float64(len(all)) - 1e-9))
			pick, errs = manSample(rng, served, k)
		case "count":
			pick, errs = manSample(rng, served, in.Count)
		case "all":
			pick = served
		case "bernoulli":
			if len(served) == 0 {
				return 0, harmed, "no served eligible order"
			}
			p := in.Effect * float64(len(all)) / float64(len(served))
			if p > 1 {
				return 0, harmed, fmt.Sprintf("bernoulli probability %.4f above 1", p)
			}
			for _, j := range served {
				if rng.Float64() < p {
					pick = append(pick, j)
				}
			}
		}
		out := cmp.Or(in.Outcome, OutLost)
		for _, j := range pick {
			manUnimage(r, j, out, outs)
		}
		n = len(pick)
	case "marginFloor", "marginAbove":
		all := manEligible(r, pol, "urgent", false)
		served := manEligible(r, pol, "urgent", true)
		k := len(all) * 2 / 100 // ⌊0.02 N⌋: at most 2 points exactly
		if in.Kind == "marginAbove" {
			k++
		}
		var pick []int
		pick, errs = manSample(rng, served, k)
		for _, j := range pick {
			manUnimage(r, j, OutLost, outs)
		}
		n = len(pick)
	case "urgentLate8d":
		// From day 60, the first six urgent studies acquired each week
		// reported 8 days past due.
		for w := int64(0); 60+7*w < 150; w++ {
			from := r.TakeoverSec + (60+7*w)*daySec
			k := 0
			for j := range r.Studies {
				s := &r.Studies[j]
				if s.Priority == "urgent" && s.AcquiredSec > from && s.AcquiredSec <= from+7*daySec && k < 6 {
					s.FinalSec = s.AcquiredSec + target(r, "urgent") + 8*daySec
					k++
					n++
				}
			}
		}
	case "routine348":
		for j := range r.Studies {
			s := &r.Studies[j]
			if s.Priority == "routine" && s.AcquiredSec > r.TakeoverSec+30*daySec {
				s.FinalSec = s.AcquiredSec + target(r, "routine") + 348*hour
				n = 1
				break
			}
		}
	case "accessLate30d":
		// 30 served eligible urgent orders wanted on days 10–100 imaged 30
		// days later (their studies moved with them).
		for _, j := range manEligible(r, pol, "urgent", true) {
			o := &r.Orders[j]
			if d := (o.WantedSec - r.TakeoverSec) / daySec; d < 10 || d >= 100 || n >= 30 {
				continue
			}
			manMove(r, j, o.ResolvedSec+30*daySec, -2)
			n++
		}
		if n < 30 {
			errs = fmt.Sprintf("%d urgent orders to delay, fewer than 30", n)
		}
	case "idRemove", "idRepeat", "idChange":
		es := manEligible(r, pol, "urgent", false)
		if len(es) < 2 {
			return 0, harmed, "fewer than two eligible urgent orders"
		}
		switch in.Kind {
		case "idRemove":
			r.Orders[es[0]].Origin = ""
		case "idRepeat":
			r.Orders[es[1]].Origin = r.Orders[es[0]].Origin
		case "idChange":
			// The request's wanted time one minute later under the same
			// origin, on the first eligible urgent order imaged (if at all)
			// at least a minute after it, so that the record stays valid.
			k := slices.IndexFunc(es, func(j int) bool {
				o := &r.Orders[j]
				return endCapture(r).OrderOutcomes[j] != OutServed || o.ResolvedSec >= o.WantedSec+60
			})
			if k < 0 {
				return 0, harmed, "no eligible urgent order whose wanted time can move a minute"
			}
			r.Orders[es[k]].WantedSec += 60
		}
		n = 1
	case "partialSevere":
		// Every soon study acquired in the leading partial week reported
		// 30 hours past due.
		mon := (floorDiv(r.TakeoverSec/daySec+3, 7)*7 + 4) * daySec
		for j := range r.Studies {
			s := &r.Studies[j]
			if s.Priority == "soon" && s.AcquiredSec > r.TakeoverSec && s.AcquiredSec < mon {
				s.FinalSec = s.AcquiredSec + target(r, "soon") + 30*hour
				n++
			}
		}
		if n < 6 {
			errs = fmt.Sprintf("%d soon studies in the leading partial week, fewer than the 6 the witness needs", n)
		}
	case "horizonCross", "pending":
		// The last soon order wanted by the horizon and imaged by it:
		// acquired one second after the horizon and reported 60 h past due
		// (horizonCross), or acquired 10 h before the end of observation
		// and unfinished (pending).
		k := -1
		for j, o := range r.Orders {
			if o.Priority == "soon" && o.WantedSec <= r.HorizonEndSec && outs[j] == OutServed && o.ResolvedSec <= r.HorizonEndSec {
				k = j
			}
		}
		if k < 0 {
			return 0, harmed, "no soon order imaged by the horizon"
		}
		if in.Kind == "horizonCross" {
			manMove(r, k, r.HorizonEndSec+1, target(r, "soon")+60*hour)
		} else {
			manMove(r, k, r.ObservedUntilSec-10*hour, -1)
		}
		n = 1
	default:
		return 0, harmed, "unknown injection " + in.Kind
	}
	manRefresh(r, outs)
	return n, harmed, errs
}

// manSample draws k of xs uniformly without replacement (a partial
// Fisher–Yates on a copy), in draw order; k above len(xs) is infeasible.
func manSample(rng ocRNG, xs []int, k int) ([]int, string) {
	if k > len(xs) {
		return nil, fmt.Sprintf("%d to inject, only %d served", k, len(xs))
	}
	ys := slices.Clone(xs)
	for j := range k {
		m := j + rng.IntN(len(ys)-j)
		ys[j], ys[m] = ys[m], ys[j]
	}
	return ys[:k], ""
}

// manUnimage makes served order j unimaged (lost at its receipt, or
// ended by the patient 30 minutes after it), its study removed.
func manUnimage(r *Record, j int, out string, outs []string) {
	o := &r.Orders[j]
	r.Studies = slices.DeleteFunc(r.Studies, func(s Study) bool { return s.Order == o.ID })
	o.FirstApptSec = 0
	o.ResolvedSec = o.ReceivedSec
	if out == OutPatient {
		o.ResolvedSec = o.ReceivedSec + 30*60
	}
	outs[j] = out
}

// manMove re-times order j's imaging to `at`, its study finished `final`
// seconds after acquisition (−1: unfinished; −2: shifted with it).
func manMove(r *Record, j int, at, final int64) {
	o := &r.Orders[j]
	for k := range r.Studies {
		s := &r.Studies[k]
		if s.Order != o.ID {
			continue
		}
		dt := at - s.AcquiredSec
		s.AcquiredSec, s.EnteredSec = at, s.EnteredSec+dt // the documentation time kept
		switch {
		case final == -1:
			s.ClaimedSec, s.ReadStartSec, s.FinalSec = 0, 0, 0
		case final == -2:
			if s.FinalSec != 0 {
				s.ClaimedSec, s.ReadStartSec, s.FinalSec = s.ClaimedSec+dt, s.ReadStartSec+dt, s.FinalSec+dt
			}
		default:
			s.ClaimedSec, s.ReadStartSec, s.FinalSec = s.EnteredSec, s.EnteredSec, max(s.EnteredSec, at+final)
		}
		if s.FinalSec > r.ObservedUntilSec {
			s.ClaimedSec, s.ReadStartSec, s.FinalSec = 0, 0, 0
		}
	}
	o.FirstApptSec, o.ResolvedSec = at, at
}

// manRefresh re-derives a changed record's samples and captures from its
// studies and final outcomes (fidSamples), so that it stays reconciled.
func manRefresh(r *Record, outs []string) {
	slices.SortStableFunc(r.Studies, func(x, y Study) int { return cmp.Compare(x.AcquiredSec, y.AcquiredSec) })
	r.Samples, r.Captures = nil, nil
	fidSamples(r, outs, r.TakeoverSec, int((r.ObservedUntilSec-r.TakeoverSec)/daySec))
}

// cloneRecord is a deep copy of the parts an injection changes.
func cloneRecord(r *Record) *Record {
	c := *r
	c.Orders, c.Studies, c.Samples = slices.Clone(r.Orders), slices.Clone(r.Studies), slices.Clone(r.Samples)
	c.Captures = slices.Clone(r.Captures)
	for i := range c.Captures {
		c.Captures[i].OrderOutcomes = slices.Clone(r.Captures[i].OrderOutcomes)
	}
	return &c
}

// groupResult is one 40-seed evaluation's protocol decision and the
// per-seed facts the criteria count.
type groupResult struct {
	Scenario  string `json:"scenario"`
	Group     int    `json:"group"`
	Replicate int    `json:"replicate,omitempty"`
	Seeds     [2]int `json:"seeds"` // first, last
	// Remainder: the scenario's last seeds, fewer than a group: judged for
	// the per-seed criteria only, never as a group (audit 5C.4i I5).
	Remainder bool `json:"remainder,omitempty"`
	Valid     bool `json:"valid"`
	Rejected  bool `json:"rejected"`
	// Rules: the rules rejecting it (integrity, safety, pending,
	// allowance, pooled, gateB:<q>, caseSelection).
	Rules        []string           `json:"rules"`
	GateB        map[string]string  `json:"gateB"`
	Increase     map[string]float64 `json:"increase"`
	Matched      map[string]int     `json:"matchedSeeds"`
	Deteriorated map[string][]int64 `json:"deterioratedSeeds"`
	Failing      []int64            `json:"allowanceFailingSeeds"`
	Insufficient []int64            `json:"allowanceInsufficientSeeds"`
	RawInsuff    map[string][]int64 `json:"rawInsufficientSeedsByGate,omitempty"`
	// FailingByGate: every seed failing each seed-level gate of the
	// allowance (reported: which rules the allowance spends on), the
	// partial-window ceiling's failures apart.
	FailingByGate map[string][]int64    `json:"failingSeedsByGate,omitempty"`
	Violations    map[string][]int64    `json:"violatingSeedsByLimit,omitempty"`
	Pending       []int64               `json:"pendingSeeds,omitempty"`
	Pairs         []pairInfo            `json:"pairs"`
	Errors        []string              `json:"constructionErrors,omitempty"`
	Claims        *AccessClaims         `json:"accessClaims,omitempty"`
	Fractions     map[string][2]float64 `json:"unimagedSeedMean,omitempty"`
	// UnmatchedUrgent: Gate B urgent's first unmatched seed (−1: none),
	// for the identity witnesses' attribution.
	UnmatchedUrgent int `json:"unmatchedUrgentSeed"`
	// Provenance: the seeds whose candidate orders do not reconcile with
	// the baseline's by origin (orderProvenance, Slice 5C.4l): the
	// comparison is INVALID there.
	Provenance []int64 `json:"provenanceSeeds,omitempty"`
	// UrgentSeeds: Gate B's urgent seed differences (T5's prediction).
	UrgentSeeds []SeedDifference `json:"urgentSeeds,omitempty"`
	// T2G's materiality inputs: the candidate's severe and acquired
	// studies over the gated period by priority (summed over the seeds),
	// and each seed's worst soon delay past due (minutes).
	Severe        map[string][2]int64 `json:"severeAcquired,omitempty"`
	WorstSoonPast []int64             `json:"worstSoonPastDueMin,omitempty"`
}

// protocolDecision judges a group as the whole candidate policy does,
// apart from finance: rejected when a record is invalid, Gate B is not
// PASS in a priority, the shared allowance with the safeguard, a gated
// pooled clinical rate or clinical safety is not PASS, or v4 case
// selection fails (compare.go verdict's non-financial rules).
func protocolDecision(p *Policy, base, cand []Record) groupResult {
	g := groupResult{GateB: map[string]string{}, Increase: map[string]float64{}, Matched: map[string]int{}, Deteriorated: map[string][]int64{}, UnmatchedUrgent: -1}
	pc, pb := planReport(p, "C", cand), planReport(p, "B", base)
	g.Valid = pc.Valid && pb.Valid
	if !g.Valid {
		g.Rejected, g.Rules = true, []string{"integrity"}
		return g
	}
	if g.Provenance = provenanceSeeds(p, base, cand); len(g.Provenance) > 0 {
		g.Rules = append(g.Rules, "provenance")
	}
	gates := lostDemandGates(p, cand, base)
	j := cmp.Or(seedSafeguardJoint(p, pc, gates), pc.ClinicalJoint)
	g.Claims = accessClaims(gates)
	g.Fractions = map[string][2]float64{}
	for _, x := range gates {
		g.GateB[x.Priority], g.Increase[x.Priority], g.Matched[x.Priority] = x.Status, x.Increase, x.MatchedSeeds
		g.Deteriorated[x.Priority] = x.Deteriorated
		if x.SeedFraction != nil {
			g.Fractions[x.Priority] = *x.SeedFraction
		}
		if x.Status != Pass {
			g.Rules = append(g.Rules, "gateB:"+x.Priority)
		}
		if x.Priority == "urgent" {
			g.UrgentSeeds, g.UnmatchedUrgent = x.Seeds, firstOr(x.UnmatchedSeeds, -1)
		}
	}
	g.Failing, g.Insufficient = j.FailingSeeds, j.InsufficientSeeds
	if j.Status != Pass {
		g.Rules = append(g.Rules, "allowance")
	}
	for _, gs := range [][]GateSummary{pc.ServiceGates, pc.ClinicalGates} {
		for _, x := range gs {
			if x.Role == "gated" && (x.Status == Fail || x.Status == Insufficient) && !slices.Contains(g.Rules, "pooled") {
				g.Rules = append(g.Rules, "pooled")
			}
		}
	}
	sf := pc.ClinicalSafety
	switch sf.Status {
	case Fail:
		g.Rules = append(g.Rules, "safety")
	case Insufficient:
		g.Rules = append(g.Rules, "pending")
	}
	for _, v := range sf.Violations {
		key := v.Limit + " " + v.Priority
		if g.Violations == nil {
			g.Violations = map[string][]int64{}
		}
		g.Violations[key] = append(g.Violations[key], v.Seed)
	}
	for _, v := range sf.Pending {
		g.Pending = append(g.Pending, v.Seed)
	}
	g.RawInsuff = pc.ClinicalInsufficient
	g.FailingByGate = map[string][]int64{}
	for _, s := range pc.PerSeed {
		for _, x := range append(slices.Clone(s.Gates), s.ClinicalGates...) {
			if x.PerSeed && role(p, x.Window) == "gated" && x.Status == Fail {
				key := x.Gate + " " + x.Window + " " + x.Stratum
				g.FailingByGate[key] = append(g.FailingByGate[key], s.Seed)
			}
		}
		if s.Backlog == "unstable" {
			g.FailingByGate["backlog_stability"] = append(g.FailingByGate["backlog_stability"], s.Seed)
		}
		if c := s.Measures.Clinical; c != nil {
			for _, q := range Priorities {
				if ws := c.WeeklySevere[q]; ws != nil && ws.PartialFailing > 0 {
					g.FailingByGate["partial_window_ceiling "+q] = append(g.FailingByGate["partial_window_ceiling "+q], s.Seed)
				}
			}
		}
	}
	g.Severe = map[string][2]int64{}
	for _, s := range pc.PerSeed {
		if s.Measures.Clinical == nil {
			continue
		}
		for _, q := range Priorities {
			d := s.Measures.Clinical.Delay[WinGated][q]
			x := g.Severe[q]
			g.Severe[q] = [2]int64{x[0] + d.Severe, x[1] + d.Acquired}
		}
		g.WorstSoonPast = append(g.WorstSoonPast, s.Measures.Clinical.Delay[WinGated]["soon"].MaxPastDueMin)
	}
	if _, st := caseSelection(p, pc, pb); st != Pass {
		g.Rules = append(g.Rules, "caseSelection")
	}
	g.Rejected = len(g.Rules) > 0
	return g
}

// provenanceSeeds: the seeds whose candidate record does not reconcile
// with the baseline's by origin, under a policy that reconciles (as
// compare does: the comparison is INVALID).
func provenanceSeeds(p *Policy, base, cand []Record) []int64 {
	if !provenanceApplies(p) {
		return nil
	}
	by := map[int64]*Record{}
	for i := range base {
		by[base[i].Seed] = &base[i]
	}
	var out []int64
	for i := range cand {
		if b := by[cand[i].Seed]; b == nil || len(orderProvenance(&cand[i], b)) > 0 {
			out = append(out, cand[i].Seed)
		}
	}
	return out
}

// familyAllowed refuses the reserved family: this build is not the lock
// commit.
func familyAllowed(F uint64) error {
	if F == reservedFamily {
		return fmt.Errorf("the reserved family PCG(20261201, ·) is not authorized in this build")
	}
	return nil
}

// gateBDecision judges a near-margin replicate as T5 does: Gate B alone
// (its status, increase and seed differences by priority); rejected when
// a priority's Gate B is not PASS. The rest of the policy is judged on the
// same pairs without injection (the base scenario).
func gateBDecision(p *Policy, base, cand []Record) groupResult {
	g := groupResult{Valid: true, GateB: map[string]string{}, Increase: map[string]float64{}, Matched: map[string]int{}, Deteriorated: map[string][]int64{}, UnmatchedUrgent: -1}
	if g.Provenance = provenanceSeeds(p, base, cand); len(g.Provenance) > 0 {
		g.Rules = append(g.Rules, "provenance")
	}
	for _, x := range lostDemandGates(p, cand, base) {
		g.GateB[x.Priority], g.Increase[x.Priority], g.Matched[x.Priority] = x.Status, x.Increase, x.MatchedSeeds
		g.Deteriorated[x.Priority] = x.Deteriorated
		if x.Status != Pass {
			g.Rules = append(g.Rules, "gateB:"+x.Priority)
		}
		if x.Priority == "urgent" {
			g.UrgentSeeds, g.UnmatchedUrgent = x.Seeds, firstOr(x.UnmatchedSeeds, -1)
		}
	}
	g.Rejected = len(g.Rules) > 0
	return g
}

// firstOr is xs's first element as an int, or d.
func firstOr(xs []int64, d int) int {
	if len(xs) == 0 {
		return d
	}
	return int(xs[0])
}

// The certified execution (Slice 5C.4n, audit 5C.4m M1). A confirmatory
// verdict is computed only by the process that constructs the records:
// TestReservedManifestRun builds every pair, judges every group and
// adjudicates every criterion in memory, and decides the certification
// before it writes anything. Its files report that decision; they are
// never its input. A reader of result files (TestReservedCriteria) is an
// offline analysis whose output is non-certifying, whatever the file
// says.

// protocolPath is the confirmatory protocol the execution certifies
// under.
const protocolPath = "../../docs/calibration/clinical-5c4n/reserved-validation-protocol.md"

// certifiedInputs are the files the execution reads at run time, with
// their SHA-256 in this build. Each is verified before anything runs; the
// lock commit re-pins them (TestCertifiedInputsPinned).
var certifiedInputs = []struct{ Role, Path, SHA256 string }{
	{"candidate", candidatePath, "83d481b7040b69d78a58de68afc938a588b5b37ba396438509f38c56ca163948"},
	{"manifest", manifestPath, "250bc7fc835d99140212505e0e79f74f2da1484dbe856fea6a5b93eec9153ef4"},
	{"protocol", protocolPath, "094bdab86eca795223f748be72e862ea4815c3156cfdd6d6a577ca104a658bad"},
	{"policyInForce", "../../" + PolicyPath, "4e277c7263dcff7ddabe63741beb44abfc9a46d3bc23f6b6418715b58ffb6fb3"},
	{"simProfile", simProfilePath, "ac8621889813a2d582b4b19674f04871c5711d27d0ac1620839ba01df4596100"},
}

// verifyInputs checks every runtime input against this build's digest,
// and the manifest's pin of the candidate, returning the digests.
func verifyInputs() (map[string]string, error) { return verifyInputList(certifiedInputs, manifestPath) }

func verifyInputList(inputs []struct{ Role, Path, SHA256 string }, manifestPath string) (map[string]string, error) {
	out := map[string]string{}
	for _, x := range inputs {
		got, err := fileDigest(x.Path)
		if err != nil {
			return nil, err
		}
		if got != x.SHA256 {
			return nil, fmt.Errorf("%s %s: SHA-256 %s; this build pins %s", x.Role, x.Path, got, x.SHA256)
		}
		out[x.Role] = x.SHA256
	}
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}
	var m struct {
		Policy map[string]string `json:"policy"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	if m.Policy["sha256"] != out["candidate"] {
		return nil, fmt.Errorf("the manifest pins candidate %q, not %s", m.Policy["sha256"], out["candidate"])
	}
	return out, nil
}

// fileDigest is a file's SHA-256, hex.
func fileDigest(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

// sourceDigest is the SHA-256 of this package's Go sources as found at
// run time (names and contents, in name order): recorded, so that the
// run can be checked against the lock commit's tree.
func sourceDigest() (string, error) {
	names, err := filepath.Glob("*.go")
	if err != nil {
		return "", err
	}
	slices.Sort(names)
	h := sha256.New()
	for _, n := range names {
		raw, err := os.ReadFile(n)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(h, "%s %d\n", n, len(raw))
		h.Write(raw)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// certContext is how a set of lines came to be judged.
type certContext struct {
	// Trusted: computed in this process from the records it constructed
	// (TestReservedManifestRun), never read from a file.
	Trusted bool
	Family  uint64
	Inputs  map[string]string
	Source  string
}

// mandatoryCriteria are those every one of which must PASS (protocol §6).
var mandatoryCriteria = []string{"T1", "T2", "T2-conformance", "T2G", "T3", "T4", "T5a", "T5b", "T6", "T7", "T8"}

// certify decides what the criteria certify. The invocation is
// confirmatory only for a trusted execution on the reserved family at
// every scenario's manifest seeds; a development execution never
// certifies; lines read from a file are a non-certifying analysis,
// whatever they contain. A confirmatory invocation certifies GO only
// when every mandatory criterion PASSes, and NO GO otherwise (§6).
func certify(r map[string]criterion, ctx certContext) {
	inv := r["invocation"]
	full := inv.Status == "manifest seeds"
	switch {
	case !ctx.Trusted:
		inv.Status = "non-certifying analysis"
	case full && ctx.Family == reservedFamily:
		inv.Status = "confirmatory"
	default:
		inv.Status = "development"
	}
	if inv.Detail == nil {
		inv.Detail = map[string]any{}
	}
	inv.Detail["trustedExecution"] = ctx.Trusted
	if ctx.Trusted {
		inv.Detail["family"] = ctx.Family
		inv.Detail["inputs"] = ctx.Inputs
		inv.Detail["sourceSHA256"] = ctx.Source
		inv.Detail["engine"] = version.Number
	}
	r["invocation"] = inv
	var fail, open []string
	for _, c := range mandatoryCriteria {
		switch r[c].Status {
		case Pass:
		case Fail:
			fail = append(fail, c)
		default:
			open = append(open, c)
		}
	}
	d := map[string]any{"failing": fail, "notPassing": open}
	cert := criterion{Detail: d}
	switch {
	case inv.Status != "confirmatory":
		cert.Status = "not certified"
		d["reason"] = map[string]string{
			"non-certifying analysis": "lines read from a file: an offline analysis never certifies (audit 5C.4m M1)",
			"development":             "a development execution: not the reserved family at every scenario's manifest seeds",
		}[inv.Status]
	case len(fail) > 0:
		cert.Status = "certified NO GO: confirmation failed"
	case len(open) > 0:
		cert.Status = "certified NO GO: confirmation not achieved"
	default:
		cert.Status = "certified GO: confirmation succeeded"
	}
	r["certification"] = cert
}

// manInvocation reads a run's settings strictly (audit 5C.4m M4): a
// malformed family, a malformed or nonpositive seed count or an unknown
// or repeated scenario is an error, never a default. Unset: the
// development family, every scenario at its manifest seeds.
func manInvocation(m *manifestDoc) (F uint64, seeds func(*manScenario) int, err error) {
	F = devManifestFamily
	if v := os.Getenv("DT_MANIFEST_FAMILY"); v != "" {
		if F, err = strconv.ParseUint(v, 10, 64); err != nil {
			return 0, nil, fmt.Errorf("DT_MANIFEST_FAMILY %q: %v", v, err)
		}
	}
	n := 0
	if v := os.Getenv("DT_MANIFEST_SEEDS"); v != "" {
		if n, err = strconv.Atoi(v); err != nil || n <= 0 {
			return 0, nil, fmt.Errorf("DT_MANIFEST_SEEDS %q: not a positive count", v)
		}
	}
	var only []string
	if v := os.Getenv("DT_MANIFEST_SCENARIOS"); v != "" {
		for _, id := range strings.Split(v, ",") {
			if manScenarioOf(m, id) == nil || slices.Contains(only, id) {
				return 0, nil, fmt.Errorf("DT_MANIFEST_SCENARIOS: %q unknown or repeated", id)
			}
			only = append(only, id)
		}
	}
	return F, func(s *manScenario) int {
		switch {
		case len(only) > 0 && !slices.Contains(only, s.ID):
			return 0 // not run under this invocation: missing, reported
		case n > 0:
			return min(n, s.Seeds)
		}
		return s.Seeds
	}, nil
}

// writeCriteria writes the adjudication; it never writes a blank or
// partial file (audit 5C.4m M4).
func writeCriteria(path string, r map[string]criterion) error {
	raw, err := json.MarshalIndent(r, "", " ")
	if err != nil {
		return fmt.Errorf("the adjudication cannot be serialized: %v", err)
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}

// coverageProblems lists every scenario's coverage problems, but the
// identity witnesses' expected provenance (none of them is complete
// coverage).
func coverageProblems(m *manifestDoc, lines []groupResult, seeds func(*manScenario) int) []string {
	var out []string
	cov := manCoverage(m, lines, seeds)
	for _, id := range slices.Sorted(maps.Keys(cov)) {
		for _, p := range cov[id] {
			out = append(out, id+": "+p)
		}
	}
	return out
}

// manRunScenario runs scenario s on family F, n seeds (whole groups
// only; the remainder is per-seed only), returning one line a group (a
// replicate). Records are dropped after each group.
func manRunScenario(t *testing.T, m *manifestDoc, s *manScenario, F uint64, n int, pol *Policy) []groupResult {
	if err := familyAllowed(F); err != nil {
		t.Fatal(err)
	}
	var lines []groupResult
	practices := manPractices()
	size := m.GroupSize
	for g := 0; g*size < n; g++ {
		lo, hi := g*size, min(n, (g+1)*size)
		bs, cs, infos := make([]Record, hi-lo), make([]Record, hi-lo), make([]pairInfo, hi-lo)
		parallelFor(hi-lo, func(x int) {
			b, c, info := manPair(s, F, lo+x, pol, practices)
			infos[x] = info
			if b != nil {
				bs[x], cs[x] = *b, *c
			}
		})
		emit := func(r int, cand []Record, infos []pairInfo) {
			res := groupResult{Scenario: s.ID, Group: g, Replicate: r, Seeds: [2]int{lo, hi - 1}, UnmatchedUrgent: -1}
			var errs []string
			for _, x := range infos {
				if x.Error != "" {
					errs = append(errs, fmt.Sprintf("seed %d: %s", x.Seed, x.Error))
				}
			}
			switch {
			case len(errs) == 0 && s.Replicates > 0:
				res = gateBDecision(pol, bs, cand)
				res.Scenario, res.Group, res.Replicate, res.Seeds = s.ID, g, r, [2]int{lo, hi - 1}
			case len(errs) == 0:
				res = protocolDecision(pol, bs, cand)
				res.Scenario, res.Group, res.Replicate, res.Seeds = s.ID, g, r, [2]int{lo, hi - 1}
			}
			res.Pairs, res.Errors, res.Remainder = infos, errs, hi-lo < size
			lines = append(lines, res)
		}
		if s.Replicates == 0 {
			emit(0, cs, infos)
			continue
		}
		// Injection replicates on the same pairs: replicate r continues
		// each seed's injection stream where replicate r − 1 stopped.
		rngs := make([]ocRNG, hi-lo)
		for x := range rngs {
			rngs[x] = ocRNG{rand.New(rand.NewPCG(F, streamIndex(roleCandidateInject, s.K, lo+x)))}
		}
		for r := range s.Replicates {
			cr, ri := make([]Record, hi-lo), slices.Clone(infos)
			parallelFor(hi-lo, func(x int) {
				c := cloneRecord(&cs[x])
				if s.Candidate.Inject != nil {
					ri[x] = manInjectCandidate(c, s.Candidate.Inject, pol, rngs[x], ri[x])
				}
				cr[x] = *c
			})
			emit(r, cr, ri)
		}
	}
	return lines
}

// manifestPolicy is the fixture policy with the candidate's clinical
// section, as every protocol evaluation uses it.
func manifestPolicy(t *testing.T) *Policy {
	p, _ := candidatePolicy(t)
	return p
}

// TestReservedManifestRun is the certified execution (protocol §8): it
// runs the manifest, judges every group and adjudicates every criterion
// in this process, decides the certification, and only then writes the
// lines (DT_MANIFEST_OUT) and the adjudication (DT_CRITERIA_OUT), neither
// of which may exist. DT_MANIFEST_FAMILY (default the development checks'
// 20261021; this build refuses the reserved family), DT_MANIFEST_SEEDS
// (seeds a scenario, at most the manifest's; unset: the manifest's) and
// DT_MANIFEST_SCENARIOS (IDs) are read strictly. Every runtime input is
// verified against this build's digests first. Incomplete coverage fails
// the test after the files are written.
func TestReservedManifestRun(t *testing.T) {
	out, crit := os.Getenv("DT_MANIFEST_OUT"), os.Getenv("DT_CRITERIA_OUT")
	if out == "" || crit == "" {
		t.Skip("set DT_MANIFEST_OUT=<lines> and DT_CRITERIA_OUT=<adjudication> to run the scenario manifest on a non-reserved family")
	}
	m := loadManifest(t)
	F, seeds, err := manInvocation(m)
	if err != nil {
		t.Fatal(err)
	}
	if err := familyAllowed(F); err != nil {
		t.Fatal(err)
	}
	inputs, err := verifyInputs()
	if err != nil {
		t.Fatal(err)
	}
	src, err := sourceDigest()
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{out, crit} {
		if _, err := os.Stat(p); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("%s exists: the execution never overwrites", p)
		}
	}
	lines, r := runCertified(t, m, F, seeds, certContext{Trusted: true, Family: F, Inputs: inputs, Source: src})
	t.Logf("invocation %s; certification: %s", r["invocation"].Status, r["certification"].Status)
	// The report of that decision.
	var buf bytes.Buffer
	for _, g := range lines {
		raw, err := json.Marshal(g)
		if err != nil {
			t.Fatal(err)
		}
		buf.Write(append(raw, '\n'))
	}
	if err := os.WriteFile(out, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeCriteria(crit, r); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{out, crit} {
		raw, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("SHA-256 %x  %s", sha256.Sum256(raw), p)
	}
	if bad := coverageProblems(m, lines, seeds); len(bad) > 0 {
		t.Fatalf("incomplete execution (%d problems, first %q): not certified", len(bad), bad[0])
	}
}

// runCertified runs the manifest's scenarios on family F, judges every
// group and adjudicates every criterion, all in this process, and
// decides the certification (ctx: the execution's identity). Nothing is
// read from or written to a file.
func runCertified(t *testing.T, m *manifestDoc, F uint64, seeds func(*manScenario) int, ctx certContext) ([]groupResult, map[string]criterion) {
	tol, err := manifestTolerances(m)
	if err != nil {
		t.Fatal(err)
	}
	pol := manifestPolicy(t)
	var lines []groupResult
	for i := range m.Scenarios {
		s := &m.Scenarios[i]
		if n := seeds(s); n > 0 {
			lines = append(lines, manRunScenario(t, m, s, F, n, pol)...)
			t.Logf("%s done", s.ID)
		}
	}
	r := reservedCriteria(m, lines, tol, seeds)
	certify(r, ctx)
	return lines, r
}

// The manifest is complete and its streams disjoint: every scenario has
// its fields, a unique k below 100 and ID, seeds within a stream's range,
// known practices, paired demand models, known injections and a known
// criterion; a reused base exists, is earlier and has as many seeds.
func TestReservedManifestWellFormed(t *testing.T) {
	m := loadManifest(t)
	practices := manPractices()
	if m.Schema != "dreamteam.reserved-scenario-manifest/v1" || m.GroupSize != 40 {
		t.Fatalf("%s %d", m.Schema, m.GroupSize)
	}
	kinds := []string{"quota", "bernoulli", "all", "count", "marginFloor", "marginAbove", "urgentLate8d", "routine348", "accessLate30d",
		"idRemove", "idRepeat", "idChange", "partialSevere", "horizonCross", "pending"}
	crit := []string{"T1", "T2", "T2G", "T3", "T4", "T5", "T6", "T7", "T8", "report"}
	ks, ids := map[int]bool{}, map[string]bool{}
	for _, s := range m.Scenarios {
		if ks[s.K] || ids[s.ID] || s.K < 0 || s.K >= 100 || s.Seeds < 40 || s.Seeds >= 100_000 || s.Mechanism == "" || s.Effect == "" ||
			s.Metric == "" || s.Acceptance == "" || s.Failure == "" || len(s.Criteria) == 0 {
			t.Errorf("%s: incomplete or repeated", s.ID)
		}
		ks[s.K], ids[s.ID] = true, true
		for _, c := range s.Criteria {
			if !slices.Contains(crit, c) {
				t.Errorf("%s: criterion %s", s.ID, c)
			}
		}
		for _, side := range []manSide{s.Baseline, s.Candidate} {
			q, ok := practices[side.Practice]
			if !ok {
				t.Errorf("%s: practice %s", s.ID, side.Practice)
			}
			if side.Inject != nil && !slices.Contains(kinds, side.Inject.Kind) {
				t.Errorf("%s: injection %s", s.ID, side.Inject.Kind)
			}
			_ = q
		}
		qb, qc := practices[s.Baseline.Practice], practices[s.Candidate.Practice]
		if demandModel(qb) != demandModel(qc) || !slices.Equal(s.Baseline.VolumeCycle, s.Candidate.VolumeCycle) {
			t.Errorf("%s: the two practices' demand differs: not a paired construction", s.ID)
		}
		if s.Base != nil {
			b := slices.IndexFunc(m.Scenarios, func(x manScenario) bool { return x.K == *s.Base })
			if b < 0 || m.Scenarios[b].K >= s.K || m.Scenarios[b].Seeds < s.Seeds || m.Scenarios[b].Baseline.Practice != s.Baseline.Practice {
				t.Errorf("%s: base %d", s.ID, *s.Base)
			}
		}
	}
	// The roles' ranges: [role, role + 100 000 × 100) are disjoint from
	// each other and from the demand streams (even, below 200 000).
	roles := []uint64{roleBaselineBooking, roleCandidateBooking, roleCandidateInject, roleBaselineInjection}
	for a := range roles {
		if roles[a] < 200_000 {
			t.Fatal("a role overlaps demand")
		}
		for b := a + 1; b < len(roles); b++ {
			if roles[b]-roles[a] < 100*100_000 {
				t.Fatal("roles overlap")
			}
		}
	}
}

// Every scenario is constructible: on two seeds of the debug family
// PCG(1, ·) its pairs match on every priority (paired demand), its
// injections are feasible and change what they say, and the protocol
// decision is reached.
func TestReservedManifestConstruction(t *testing.T) {
	if testing.Short() {
		t.Skip("short")
	}
	m := loadManifest(t)
	pol := manifestPolicy(t)
	m.GroupSize = 2
	for i := range m.Scenarios {
		s := m.Scenarios[i]
		s.Replicates = min(s.Replicates, 1)
		g := manRunScenario(t, m, &s, 1, 2, pol)[0]
		if bad := lineProblems(&s, &g); len(bad) > 0 {
			t.Errorf("%s: measurements %v", s.ID, bad)
		}
		if len(g.Errors) > 0 {
			t.Errorf("%s: %v", s.ID, g.Errors)
			continue
		}
		for _, q := range Priorities {
			want := 2
			if c := s.Candidate.Inject; c != nil && strings.HasPrefix(c.Kind, "id") {
				want = 1 // the identity witness's own seed
			}
			if g.Matched[q] < want {
				t.Errorf("%s %s: %d matched seeds", s.ID, q, g.Matched[q])
			}
		}
	}
}

// criterion is one protocol criterion's result.
type criterion struct {
	Status string         `json:"status"` // pass, fail, inconclusive, or not locked
	Detail map[string]any `json:"detail"`
}

// T5 tolerances: nil until the owner approves them (protocol §5).
type t5Tolerance struct{ At21, At3 *float64 }

// manifestTolerances are the manifest's T5 tolerances, when approved.
func manifestTolerances(m *manifestDoc) (t5Tolerance, error) {
	var x struct {
		Status string   `json:"status"`
		At21   *float64 `json:"at2.1"`
		At3    *float64 `json:"at3"`
	}
	if err := json.Unmarshal(m.Tolerances, &x); err != nil {
		return t5Tolerance{}, err
	}
	if !strings.HasPrefix(x.Status, "APPROVED") {
		return t5Tolerance{}, nil
	}
	return t5Tolerance{x.At21, x.At3}, nil
}

// manCoverage checks a run's lines against the manifest before anything
// is judged (audit 5C.4i I5): for every scenario, exactly its complete
// groups of 40 (seeds 40g … 40g + 39) and, when the seeds do not divide,
// one remainder (marked, the last seeds), each replicate exactly once,
// every pair's seed present once, every record valid, no construction
// error. seeds gives a scenario's seeds (the manifest's, or fewer for a
// development check). Its result is each scenario's problems.
func manCoverage(m *manifestDoc, lines []groupResult, seeds func(*manScenario) int) map[string][]string {
	out := map[string][]string{}
	note := func(id, f string, a ...any) { out[id] = append(out[id], fmt.Sprintf(f, a...)) }
	type key struct{ g, r int }
	seen := map[string]map[key]bool{}
	scen := map[string]*manScenario{}
	for i := range m.Scenarios {
		scen[m.Scenarios[i].ID] = &m.Scenarios[i]
		seen[m.Scenarios[i].ID] = map[key]bool{}
	}
	size := m.GroupSize
	for _, g := range lines {
		s := scen[g.Scenario]
		if s == nil {
			note("(unknown)", "a line of unknown scenario %q", g.Scenario)
			continue
		}
		n, reps := seeds(s), max(1, s.Replicates)
		k := key{g.Group, g.Replicate}
		switch {
		case seen[s.ID][k]:
			note(s.ID, "group %d replicate %d twice", g.Group, g.Replicate)
			continue
		case g.Replicate < 0 || g.Replicate >= reps || g.Group < 0 || g.Group*size >= n:
			note(s.ID, "group %d replicate %d outside the manifest", g.Group, g.Replicate)
			continue
		}
		seen[s.ID][k] = true
		lo, hi := g.Group*size, min(n, (g.Group+1)*size)
		if g.Seeds != [2]int{lo, hi - 1} || len(g.Pairs) != hi-lo || g.Remainder != (hi-lo < size) {
			note(s.ID, "group %d: seeds %v, %d pairs, remainder %v; want %d–%d", g.Group, g.Seeds, len(g.Pairs), g.Remainder, lo, hi-1)
			continue
		}
		for x, p := range g.Pairs {
			if p.Seed != lo+x {
				note(s.ID, "group %d: pair %d is seed %d", g.Group, x, p.Seed)
				break
			}
		}
		if len(g.Errors) > 0 {
			note(s.ID, "group %d: construction errors %v", g.Group, g.Errors)
		}
		if !g.Valid {
			note(s.ID, "group %d: an invalid record", g.Group)
		}
		if len(g.Errors) == 0 && g.Valid {
			for _, p := range lineProblems(s, &g) {
				note(s.ID, "group %d replicate %d: %s", g.Group, g.Replicate, p)
			}
		}
		if len(g.Provenance) > 0 && !slices.Contains([]string{"W-ID-REMOVED", "W-ID-REPEATED", "W-ID-CHANGED"}, s.ID) {
			note(s.ID, "group %d: order provenance fails on seeds %v", g.Group, g.Provenance)
		}
	}
	for _, s := range m.Scenarios {
		n, reps := seeds(&s), max(1, s.Replicates)
		if n <= 0 {
			// Not run under this invocation: never complete coverage
			// (audit 5C.4k K4).
			note(s.ID, "no seeds under this invocation: not run")
		}
		for g := 0; g*size < n; g++ {
			for r := range reps {
				if !seen[s.ID][key{g, r}] {
					note(s.ID, "group %d replicate %d missing", g, r)
				}
			}
		}
	}
	return out
}

// lineGateB is the candidate's Gate B rule, which a line's urgent
// measurements are reconciled with (nil when it cannot be read).
var lineGateB = sync.OnceValue(func() *LostDemand {
	p, _, err := LoadPolicy(candidatePath)
	if err != nil || p.Clinical == nil {
		return nil
	}
	return p.Clinical.LostDemand
})

// gateBReconciles: a line's Gate B is the rule's computation on its own
// measurements (audit 5C.4o O1). A PASS or FAIL is decided on every pair,
// matched; urgent's on every pair's difference, at least the rule's
// seeds. Each difference is its counts', exactly; urgent's increase is
// their mean; its deteriorated seeds are those above the per-seed
// safeguard; its status is the mean's against the margin.
func gateBReconciles(g *groupResult, bad func(string, ...any)) {
	ld := lineGateB()
	if ld == nil || ld.Rule != "empirical" {
		bad("Gate B's rule unavailable: the line cannot be reconciled")
		return
	}
	n := len(g.Pairs)
	for _, q := range Priorities {
		if st := g.GateB[q]; (st == Pass || st == Fail) && g.Matched[q] != n {
			bad("Gate B %s %s with %d of %d pairs matched", q, st, g.Matched[q], n)
		}
	}
	var diffs []float64
	var det []int64
	for _, d := range g.UrgentSeeds {
		if d.Orders <= 0 {
			continue // refused above
		}
		if d.Difference != float64(d.Candidate-d.Baseline)/float64(d.Orders) {
			bad("urgent seed %d: difference %v is not its counts' (%d − %d) / %d", d.Seed, d.Difference, d.Candidate, d.Baseline, d.Orders)
		}
		over := mul(d.Candidate-d.Baseline, 10_000).Cmp(mul(bp(ld.MaxSeedIncrease["urgent"]), d.Orders)) > 0
		if d.Deteriorated != over {
			bad("urgent seed %d: deteriorated %v, its counts say %v", d.Seed, d.Deteriorated, over)
		}
		if over {
			det = append(det, d.Seed)
		}
		diffs = append(diffs, d.Difference)
	}
	want := 0.0
	if len(diffs) > 0 {
		want, _ = meanSD(diffs)
	}
	if g.Increase["urgent"] != want {
		bad("Gate B urgent: increase %v, its seed differences' mean %v", g.Increase["urgent"], want)
	}
	if got := slices.Sorted(slices.Values(g.Deteriorated["urgent"])); !slices.Equal(got, slices.Sorted(slices.Values(det))) {
		bad("Gate B urgent: deteriorated seeds %v, its seed differences' %v", got, det)
	}
	if st := g.GateB["urgent"]; st == Pass || st == Fail {
		if len(g.UrgentSeeds) != n || n < ld.MinSeeds {
			bad("Gate B urgent %s on %d seed differences of %d pairs", st, len(g.UrgentSeeds), n)
		} else if (st == Pass) != meanAtMost(g.UrgentSeeds, bp(ld.MaxIncrease["urgent"])) {
			bad("Gate B urgent %s, its seed differences' mean says otherwise", st)
		}
	}
}

// lineRules are the rules a group can be rejected by (protocol §3).
var lineRules = []string{"integrity", "provenance", "gateB:urgent", "gateB:soon", "gateB:routine", "allowance", "pooled", "safety", "pending", "caseSelection"}

// lineProblems: a judged line's measurements are complete and coherent
// (audit 5C.4m M2). Every measurement a criterion reads is present, of
// its exact cardinality, finite and in its range; every seed it names is
// one of the group's pairs, named once; the decision agrees with its
// rules. A missing measurement is never read as zero. A near-margin
// replicate carries Gate B alone (gateBDecision).
func lineProblems(s *manScenario, g *groupResult) []string {
	var out []string
	bad := func(f string, a ...any) { out = append(out, fmt.Sprintf(f, a...)) }
	finite := func(x float64) bool { return !math.IsNaN(x) && !math.IsInf(x, 0) }
	n := len(g.Pairs)
	pairs := map[int64]bool{}
	for _, p := range g.Pairs {
		pairs[int64(p.Seed)] = true
		if p.Injected < 0 || p.BaseInj < 0 || p.Eligible < 0 || p.BaseEligible < 0 || p.Served < 0 || p.Served > p.Eligible || !finite(p.Volume) {
			bad("seed %d: construction counts out of range", p.Seed)
		}
		if p.Error != "" {
			bad("seed %d: a construction error not among the line's", p.Seed) // audit 5C.4o
		}
	}
	// seeds: every seed one of the group's pairs, and named once unless
	// repeats are its form (a seed pending in two priorities).
	seedsRep := func(name string, ss []int64, repeats bool) {
		seen := map[int64]bool{}
		for _, x := range ss {
			switch {
			case !pairs[x]:
				bad("%s: seed %d is not one of the group's pairs", name, x)
			case seen[x] && !repeats:
				bad("%s: seed %d twice", name, x)
			}
			seen[x] = true
		}
	}
	seeds := func(name string, ss []int64) { seedsRep(name, ss, false) }
	if g.Rejected != (len(g.Rules) > 0) {
		bad("rejected %v with rules %v", g.Rejected, g.Rules)
	}
	for i, r := range g.Rules {
		if !slices.Contains(lineRules, r) || slices.Contains(g.Rules[:i], r) {
			bad("rule %q unknown or repeated", r)
		}
	}
	if len(g.GateB) != len(Priorities) || len(g.Increase) != len(Priorities) || len(g.Matched) != len(Priorities) || len(g.Deteriorated) != len(Priorities) {
		bad("Gate B's measurements are not exactly one a priority")
	}
	for _, q := range Priorities {
		if st, ok := g.GateB[q]; !ok || (st != Pass && st != Fail && st != Insufficient) {
			bad("Gate B %s: status %q", q, st)
		}
		if x, ok := g.Increase[q]; !ok || !finite(x) {
			bad("Gate B %s: increase missing or not finite", q)
		}
		if x, ok := g.Matched[q]; !ok || x < 0 || x > n {
			bad("Gate B %s: %d matched seeds of %d pairs", q, x, n)
		}
		if _, ok := g.Deteriorated[q]; !ok {
			bad("Gate B %s: deteriorated seeds missing", q)
		}
		seeds("deteriorated "+q, g.Deteriorated[q])
	}
	if g.UnmatchedUrgent != -1 && !pairs[int64(g.UnmatchedUrgent)] {
		bad("unmatched urgent seed %d is not one of the group's pairs", g.UnmatchedUrgent)
	}
	seeds("provenance", g.Provenance)
	var us []int64
	for _, d := range g.UrgentSeeds {
		us = append(us, d.Seed)
		if d.Orders <= 0 || d.Baseline < 0 || d.Candidate < 0 || d.Baseline > d.Orders || d.Candidate > d.Orders || !finite(d.Difference) {
			bad("urgent seed %d: counts out of range", d.Seed)
		}
	}
	seeds("urgent seeds", us)
	if len(us) > g.Matched["urgent"] {
		bad("%d urgent seed differences, %d matched seeds", len(us), g.Matched["urgent"])
	}
	gateBReconciles(g, bad)
	if s.Replicates > 0 {
		// T5's prediction reads every seed's difference and its served
		// orders: the group's own seeds, all of them (protocol §5).
		if len(us) != n {
			bad("%d urgent seed differences of %d pairs: T5's prediction needs every pair", len(us), n)
		}
		diff := map[int64]SeedDifference{}
		for _, d := range g.UrgentSeeds {
			diff[d.Seed] = d
		}
		for _, p := range g.Pairs {
			if s.Candidate.Inject != nil && p.Served == 0 {
				bad("seed %d: no served order recorded for the injection", p.Seed)
			}
			if d, ok := diff[int64(p.Seed)]; ok && (p.Injected > p.Served || int64(p.Injected) > d.Candidate) {
				bad("seed %d: %d injected orders, %d served before it, %d unimaged after it", p.Seed, p.Injected, p.Served, d.Candidate)
			}
		}
		return out
	}
	if g.Claims == nil {
		bad("access claims missing")
	}
	seeds("allowance failing", g.Failing)
	seeds("allowance insufficient", g.Insufficient)
	seedsRep("pending", g.Pending, true)
	for k, ss := range g.Violations {
		seeds("violations "+k, ss)
	}
	for k, ss := range g.FailingByGate {
		seeds("failing "+k, ss)
	}
	for k, ss := range g.RawInsuff {
		seeds("raw insufficient "+k, ss)
	}
	if len(g.Severe) != len(Priorities) {
		bad("severe and acquired studies: %d priorities", len(g.Severe))
	}
	for _, q := range Priorities {
		if x, ok := g.Severe[q]; !ok || x[0] < 0 || x[1] < x[0] {
			bad("severe and acquired %s: %v", q, x)
		}
	}
	if len(g.WorstSoonPast) != n {
		bad("%d worst soon delays for %d pairs", len(g.WorstSoonPast), n)
	}
	for _, x := range g.WorstSoonPast {
		if x < 0 {
			bad("worst soon delay %d", x)
		}
	}
	for q, f := range g.Fractions {
		if !finite(f[0]) || !finite(f[1]) || f[0] < 0 || f[0] > 1 || f[1] < 0 || f[1] > 1 {
			bad("unimaged fraction %s: %v", q, f)
		}
	}
	return out
}

// reservedCriteria judges a manifest run's group lines by the protocol's
// §4, exactly: one result a criterion. Coverage first (manCoverage): a
// criterion any of whose scenarios is incomplete, duplicated, malformed,
// invalid or wrongly constructed is INCONCLUSIVE unless it already FAILs.
// Group-level criteria (T1, T2-conformance, T2G, T4, T5, T7) use the
// complete groups only; per-seed criteria (T2's detection, T3, T6) every
// seed, the remainder's included. Without approved T5 tolerances T5b is
// "not locked" and the run cannot confirm.
func reservedCriteria(m *manifestDoc, lines []groupResult, tol t5Tolerance, seeds func(*manScenario) int) map[string]criterion {
	cov := manCoverage(m, lines, seeds)
	by, all := map[string][]groupResult{}, map[string][]groupResult{}
	for _, g := range lines {
		all[g.Scenario] = append(all[g.Scenario], g)
		if !g.Remainder {
			by[g.Scenario] = append(by[g.Scenario], g)
		}
	}
	scen := map[string]*manScenario{}
	for i := range m.Scenarios {
		scen[m.Scenarios[i].ID] = &m.Scenarios[i]
	}
	has := func(s *manScenario, c string) bool { return slices.Contains(s.Criteria, c) }
	out := map[string]criterion{}
	incomplete := func(ids ...string) bool {
		for _, id := range ids {
			if len(cov[id]) > 0 {
				return true
			}
		}
		return false
	}
	var stable, t2, t2g, wit, nm []string
	for _, s := range m.Scenarios {
		switch {
		case has(&s, "T1"):
			stable = append(stable, s.ID)
		case has(&s, "T2"):
			t2 = append(t2, s.ID)
		case has(&s, "T2G"):
			t2g = append(t2g, s.ID)
		case has(&s, "T4"):
			wit = append(wit, s.ID)
		case has(&s, "T5"):
			nm = append(nm, s.ID)
		}
	}
	verdict := func(fail, inconclusive bool) string {
		switch {
		case fail:
			return Fail
		case inconclusive:
			return "inconclusive"
		}
		return Pass
	}
	const dependence = "seeds share their demand across scenarios (paired demand): pooled per-seed counts are not independent binomial trials; intervals are reported for orientation only"
	// empty: a criterion with nothing to judge is never PASS (audit 5C.4k
	// K4): no scenario of its kind in the manifest, or one of them with no
	// complete group (or, for per-seed criteria, no seed).
	empty := func(ids []string, n func(id string) int) bool {
		if len(ids) == 0 {
			return true
		}
		for _, id := range ids {
			if n(id) == 0 {
				return true
			}
		}
		return false
	}
	groups := func(id string) int { return len(by[id]) }
	// partialSeeds: each scenario's seeds failing the partial-window
	// ceiling, by priority (reported: T1 and T7).
	partialSeeds := func(id string) map[string][]int64 {
		out := map[string][]int64{}
		for _, g := range all[id] {
			for _, q := range Priorities {
				out[q] = append(out[q], g.FailingByGate["partial_window_ceiling "+q]...)
			}
		}
		return out
	}
	// rates: rejected complete groups of report-only scenarios.
	rates := func(ids ...string) map[string]any {
		out := map[string]any{}
		for _, id := range ids {
			rej := 0
			for _, g := range by[id] {
				if g.Rejected {
					rej++
				}
			}
			out[id] = map[string]any{"groups": len(by[id]), "rejected": rej, "interval95": cpInterval(rej, len(by[id]), 0.05)}
		}
		return out
	}
	// T1 and T7: at most 5% of the complete groups (3 of 62).
	t1, t7 := map[string]any{}, map[string]any{}
	f1, f7 := false, false
	for _, id := range stable {
		gs := by[id]
		limit := len(gs) * 5 / 100
		rej, ins := 0, 0
		rules, gates := map[string]int{}, map[string]int{}
		for _, g := range gs {
			if g.Rejected {
				rej++
				for _, r := range g.Rules {
					rules[r]++
				}
			}
			window := len(g.Pending) > 0
			for k := range g.RawInsuff {
				if strings.HasPrefix(k, GateSevereWeek+" ") {
					window = true
					gates[k]++
				}
			}
			if window {
				ins++
			}
		}
		f1, f7 = f1 || rej > limit, f7 || ins > limit
		ps := partialSeeds(id)
		t1[id] = map[string]any{"groups": len(gs), "rejected": rej, "limit": limit, "byRule": rules,
			"interval95": cpInterval(rej, len(gs), 0.05), "bonferroniInterval": cpInterval(rej, len(gs), 0.05/float64(len(stable))),
			"partialCeilingSeeds": ps}
		t7[id] = map[string]any{"groups": len(gs), "insufficient": ins, "limit": limit, "groupsByGate": gates,
			"interval95": cpInterval(ins, len(gs), 0.05), "partialCeilingSeeds": ps}
	}
	// The stress practices: reported, never judged (protocol §4 T1).
	t1["stress (reported, never judged)"] = rates("S-A92", "S-B92", "S-C92", "S-Cs92")
	out["T1"] = criterion{verdict(f1, incomplete(stable...) || empty(stable, groups)), t1}
	out["T7"] = criterion{verdict(f7, incomplete(stable...) || empty(stable, groups)), t7}
	// T3 and T6: per seed, all seeds.
	t3, t6 := map[string]any{}, map[string]int{}
	f3, n6 := false, 0
	seedsOf := map[string]int{}
	for _, id := range stable {
		v := map[string]int{}
		n := 0
		for _, g := range all[id] {
			for k, ss := range g.Violations {
				v[k] += len(ss)
				f3 = true
			}
			for q, ss := range g.Deteriorated {
				t6[q] += len(ss)
			}
			n += len(g.Pairs)
		}
		n6 += n
		seedsOf[id] = n
		bounds := map[string]float64{}
		for _, q := range Priorities {
			k := v["pastDue "+q]
			bounds[q] = cpInterval(k, n, 0.10)[1] // one-sided 95%
		}
		t3[id] = map[string]any{"seeds": n, "violatingSeeds": v, "upper95": bounds,
			"bonferroniUpper": cpInterval(0, n, 0.10/float64(3*len(stable)))[1]}
	}
	perSeed := func(id string) int { return seedsOf[id] }
	out["T3"] = criterion{verdict(f3, incomplete(stable...) || empty(stable, perSeed)), t3}
	f6 := false
	ivs := map[string][2]float64{}
	for _, q := range Priorities {
		f6 = f6 || t6[q] > 19
		ivs[q] = cpInterval(t6[q], n6, 0.05)
	}
	out["T6"] = criterion{verdict(f6, incomplete(stable...) || empty(stable, perSeed)), map[string]any{"seeds": n6, "flagged": t6, "limit": 19, "interval95": ivs, "note": dependence}}
	// T2: detection of harmed seeds (every seed), conformance and
	// rejection (complete groups).
	harmed, flagged, conform := 0, 0, true
	t2rep := map[string]any{}
	for _, id := range t2 {
		sh, sf := 0, 0
		for _, g := range all[id] {
			for _, p := range g.Pairs {
				if !p.Harmed {
					continue
				}
				harmed++
				sh++
				hit := slices.Contains(g.Deteriorated["urgent"], int64(p.Seed))
				if hit {
					flagged++
					sf++
				}
				if hit && !g.Remainder && !slices.Contains(g.Failing, int64(p.Seed)) {
					conform = false
				}
			}
		}
		rej, predicted := 0, 0
		rules := map[string]int{}
		for _, g := range by[id] {
			if g.Rejected {
				rej++
				for _, r := range g.Rules {
					rules[r]++
				}
			}
			if len(g.Failing)+len(g.Insufficient) > 4 && !g.Rejected {
				conform = false
			}
			// The exact conditional prediction: every harmed seed flagged
			// and failing, beside the group's other failing or insufficient
			// seeds; rejected when more than four (the allowance).
			n := 0
			for _, p := range g.Pairs {
				x := int64(p.Seed)
				if p.Harmed || slices.Contains(g.Failing, x) || slices.Contains(g.Insufficient, x) {
					n++
				}
			}
			if n > 4 {
				predicted++
			}
		}
		det := 0.0
		if sh > 0 {
			det = float64(sf) / float64(sh)
		}
		t2rep[id] = map[string]any{"groups": len(by[id]), "rejected": rej, "byRule": rules, "harmed": sh, "flagged": sf, "detection": det,
			"lower95": cpInterval(sf, sh, 0.10)[0], "conditionalPrediction": float64(predicted) / float64(max(1, len(by[id])))}
	}
	t2rep["H10, OH10 (reported, never judged)"] = rates("H10", "OH10")
	det := 0.0
	if harmed > 0 {
		det = float64(flagged) / float64(harmed)
	}
	out["T2"] = criterion{verdict(harmed >= 200 && det < 0.95, harmed < 200 || incomplete(t2...) || empty(t2, groups)),
		map[string]any{"harmed": harmed, "flagged": flagged, "detection": det, "lower95": cpInterval(flagged, harmed, 0.10)[0],
			"rejection": t2rep, "allowancePrediction": 0.3710, "note": dependence}}
	out["T2-conformance"] = criterion{verdict(!conform, incomplete(t2...) || empty(t2, groups)), map[string]any{"holds": conform}}
	// T2G: materiality from the run, each trigger on its own (a severe
	// soon share, or a severe routine share, of at least 1%; a median
	// worst soon delay of at least 48 h; a Gate B seed-mean increase of
	// at least 3 points in a priority), then rejection of the material.
	t2gRep := map[string]any{}
	f2g, u2g := false, false
	for _, id := range t2g {
		share := map[string]float64{}
		var worst []int64
		rej := 0
		for _, q := range []string{"soon", "routine"} {
			var sev, acq float64
			for _, g := range by[id] {
				sev, acq = sev+float64(g.Severe[q][0]), acq+float64(g.Severe[q][1])
			}
			if acq > 0 {
				share[q] = sev / acq
			}
		}
		for _, g := range by[id] {
			worst = append(worst, g.WorstSoonPast...)
			if g.Rejected {
				rej++
			}
		}
		slices.Sort(worst)
		med := int64(0)
		if len(worst) > 0 {
			med = worst[len(worst)/2]
		}
		maxInc := 0.0
		// Materiality needs its observations (audit 5C.4m M2): a priority
		// with no acquired study, no worst delay or a missing increase
		// leaves it undetermined, never non-material.
		undetermined := len(worst) == 0 || len(share) < 2
		for _, q := range Priorities {
			var s float64
			for _, g := range by[id] {
				x, ok := g.Increase[q]
				undetermined = undetermined || !ok
				s += x
			}
			maxInc = max(maxInc, s/float64(max(1, len(by[id]))))
		}
		material := share["soon"] >= 0.01 || share["routine"] >= 0.01 || med >= 48*60 || maxInc >= 0.03
		if material && rej < len(by[id])-2 {
			f2g = true
		}
		u2g = u2g || (undetermined && !material)
		t2gRep[id] = map[string]any{"material": material, "materialityUndetermined": undetermined && !material, "severeShare": share,
			"worstSoonMedianMin": med, "maxIncrease": maxInc, "rejected": rej, "groups": len(by[id])}
	}
	out["T2G"] = criterion{verdict(f2g, u2g || incomplete(t2g...) || empty(t2g, groups)), t2gRep}
	// T4: every complete group of every witness as expected, by its
	// targeted rule.
	t4 := map[string]any{}
	f4 := false
	for _, id := range wit {
		met := 0
		for _, g := range by[id] {
			if witnessMet(id, g) {
				met++
			}
		}
		f4 = f4 || met != len(by[id])
		t4[id] = map[string]any{"groups": len(by[id]), "met": met}
	}
	out["T4"] = criterion{verdict(f4, incomplete(wit...) || empty(wit, groups)), t4}
	// T5: conformance with the prediction, and the tolerances.
	t5 := map[string]any{}
	f5a := false
	rate := map[string]float64{}
	few, unpredicted := false, false
	for _, id := range nm {
		means := map[int][2]float64{}
		pred := map[int]float64{}
		total, pass := 0, 0
		d := scen[id].Candidate.Inject
		for _, g := range by[id] {
			total++
			x := means[g.Group]
			if g.GateB["urgent"] == Pass {
				pass++
				x[0]++
			}
			x[1]++
			means[g.Group] = x
			if _, ok := pred[g.Group]; !ok {
				pred[g.Group] = predictAcceptance(g, d, 20_000)
			}
		}
		var gm []float64
		pm := 0.0
		// In group order, so that the sums, and the output's bytes, do not
		// depend on map iteration.
		for _, k := range slices.Sorted(maps.Keys(means)) {
			x := means[k]
			gm = append(gm, x[0]/x[1])
			pm += pred[k] / float64(len(means))
		}
		few = few || len(means) < 2 // a cluster standard error needs two groups
		obs := float64(pass) / float64(max(1, total))
		rate[id] = obs
		if len(gm) < 2 || math.IsNaN(pm) {
			// No prediction without seed differences to draw (audit 5C.4m
			// M2): undetermined, never a prediction of zero.
			unpredicted = unpredicted || math.IsNaN(pm)
			t5[id] = map[string]any{"evaluations": total, "accepted": pass, "rate": obs, "predicted": nil, "conforms": nil}
			continue
		}
		mu, sd := meanSD(gm)
		se := sd / math.Sqrt(float64(len(gm)))
		ok := math.Abs(obs-pm) <= 4*se+0.02
		f5a = f5a || !ok
		t5[id] = map[string]any{"evaluations": total, "accepted": pass, "rate": obs, "groupMean": mu, "clusterSE": se, "predicted": pm, "conforms": ok,
			"note": "500 evaluations are 25 groups × 20 replicates: the cluster standard error, not a binomial interval, measures them"}
	}
	out["T5a"] = criterion{verdict(f5a && !few && !unpredicted, few || unpredicted || incomplete(nm...) || empty(nm, groups)), t5}
	switch {
	case tol.At21 == nil || tol.At3 == nil:
		out["T5b"] = criterion{"not locked", map[string]any{"reason": "the owner has not approved the T5 tolerances", "rate2.1": rate["NM2.1"], "rate3": rate["NM3"]}}
	default:
		out["T5b"] = criterion{verdict(rate["NM2.1"] > *tol.At21 || rate["NM3"] > *tol.At3, incomplete("NM2.1", "NM3") || empty([]string{"NM2.1", "NM3"}, groups)),
			map[string]any{"rate2.1": rate["NM2.1"], "tolerance2.1": *tol.At21, "rate3": rate["NM3"], "tolerance3": *tol.At3}}
	}
	// T8: construction integrity everywhere: complete coverage, every pair
	// matched (but the identity witnesses' own seed), every quota or count
	// injection exactly as specified on both sides.
	t8 := map[string]any{}
	f8 := len(cov["(unknown)"]) > 0
	if f8 {
		t8["(unknown)"] = cov["(unknown)"]
	}
	for _, s := range m.Scenarios {
		bad := slices.Clone(cov[s.ID])
		for _, g := range all[s.ID] {
			for _, q := range Priorities {
				want := len(g.Pairs)
				if c := s.Candidate.Inject; c != nil && len(g.Pairs) > 0 && g.Pairs[0].Seed%40 == 0 && idUnmatched(c.Kind, q) {
					want-- // the identity witness's own seed
				}
				// Exactly the group's pairs (audit 5C.4m M2: never more).
				if g.Valid && g.Matched[q] != want {
					bad = append(bad, fmt.Sprintf("group %d: %s matched %d of %d", g.Group, q, g.Matched[q], want))
				}
			}
			for _, p := range g.Pairs {
				if w, ok := injectionExpected(s.Candidate.Inject, p.Harmed, p.Volume, p.Eligible); ok && p.Injected != w {
					bad = append(bad, fmt.Sprintf("seed %d: candidate injected %d, specified %d", p.Seed, p.Injected, w))
				}
				if w, ok := injectionExpected(s.Baseline.Inject, true, p.Volume, p.BaseEligible); ok && p.BaseInj != w {
					bad = append(bad, fmt.Sprintf("seed %d: baseline injected %d, specified %d", p.Seed, p.BaseInj, w))
				}
			}
		}
		if len(bad) > 0 {
			f8 = true
			t8[s.ID] = bad[:min(len(bad), 10)]
		}
	}
	out["T8"] = criterion{verdict(false, f8 || len(lines) == 0), t8}
	// What was judged: the seeds each scenario was judged on, against the
	// manifest's (audit 5C.4k K4): "manifest seeds" only when every
	// scenario was declared at its manifest seeds; otherwise a development
	// check.
	inv := map[string]any{}
	full := true
	for _, s := range m.Scenarios {
		n := seeds(&s)
		inv[s.ID] = map[string]int{"seeds": n, "manifestSeeds": s.Seeds, "lines": len(all[s.ID])}
		full = full && n == s.Seeds
	}
	// Whether the lines certify anything is decided by certify, from how
	// they were computed: never from the lines themselves.
	status := "development"
	if full {
		status = "manifest seeds"
	}
	out["invocation"] = criterion{status, inv}
	return out
}

// idUnmatched: whether an identity witness's own seed is unmatched in
// priority q. A removed or repeated origin leaves the record without
// identity evidence, so every priority is unmatched; a changed urgent
// request leaves only urgent unmatched.
func idUnmatched(kind, q string) bool {
	switch kind {
	case "idRemove", "idRepeat":
		return true
	case "idChange":
		return q == "urgent"
	}
	return false
}

// injectionExpected is a quota or count injection's specified number of
// orders on a seed (ok false for any other kind, or a seed it skips).
func injectionExpected(in *manInject, harmed bool, volume float64, eligible int) (int, bool) {
	if in == nil || !harmed || (in.OnVolume != 0 && volume != in.OnVolume) {
		return 0, false
	}
	switch in.Kind {
	case "quota":
		return int(math.Ceil(in.Effect*float64(eligible) - 1e-9)), true
	case "count":
		return in.Count, true
	}
	return 0, false
}

// witnessMet: a T4 group meets its scenario's expectation (protocol §4),
// by its targeted rule: the named gate on every seed, or the named limit.
func witnessMet(id string, g groupResult) bool {
	rule := func(r string) bool { return g.Rejected && slices.Contains(g.Rules, r) }
	// every: the seeds named are exactly the group's pairs (audit 5C.4m
	// M2: not merely as many).
	every := func(ss []int64) bool {
		var want []int64
		for _, p := range g.Pairs {
			want = append(want, int64(p.Seed))
		}
		got := slices.Clone(ss)
		slices.Sort(got)
		slices.Sort(want)
		return len(want) > 0 && slices.Equal(slices.Compact(got), want)
	}
	everySeed := func(gate string) bool {
		ss := g.FailingByGate[gate]
		if len(ss) != len(g.Pairs) || !every(ss) {
			return false
		}
		for _, p := range g.Pairs {
			if !slices.Contains(g.Failing, int64(p.Seed)) {
				return false
			}
		}
		return true
	}
	switch id {
	case "W-URG-LATE":
		return rule("safety") && every(g.Violations["pastDue urgent"]) && len(g.Violations["pastDue urgent"]) == len(g.Pairs)
	case "W-ROUT-348":
		return rule("safety") && every(g.Violations["pastDue routine"]) && len(g.Violations["pastDue routine"]) == len(g.Pairs)
	case "W-HORIZON":
		return rule("safety") && every(g.Violations["pastDue soon"]) && len(g.Violations["pastDue soon"]) == len(g.Pairs)
	case "W-PENDING":
		return rule("pending") && every(g.Pending)
	case "W-PARTIAL":
		return rule("allowance") && everySeed("partial_window_ceiling soon")
	case "W-ACC-LATE":
		return rule("allowance") && everySeed(GateTimelyAccess+" "+WinGated+" urgent")
	case "W-LOST30", "W-PAT30", "W-MARGIN-ABOVE":
		return rule("gateB:urgent") && g.GateB["urgent"] == Fail
	case "W-ID-REMOVED", "W-ID-REPEATED", "W-ID-CHANGED":
		// Order provenance on exactly the stated seed (Slice 5C.4l: the
		// comparison INVALID), and Gate B unmatched there.
		return rule("provenance") && len(g.Pairs) > 0 && slices.Equal(g.Provenance, []int64{int64(g.Pairs[0].Seed)}) &&
			rule("gateB:urgent") && g.GateB["urgent"] == Insufficient && g.UnmatchedUrgent == g.Pairs[0].Seed
	case "W-MARGIN-AT":
		return g.GateB["urgent"] == Pass
	case "W-ACCESS65":
		// A favourable Gate B on both plans losing 65% of their urgent
		// demand (non-deterioration only): Gate B PASS, and the comparison
		// discloses the absolute loss with it (each plan's urgent unimaged
		// fraction at least 0.65) and claims neither adequacy nor
		// population confidence (audit 5C.4k K6: not the constants alone).
		fr, ok := g.Fractions["urgent"]
		return g.GateB["urgent"] == Pass && ok && fr[0] >= 0.65-1e-9 && fr[1] >= 0.65-1e-9 &&
			g.Claims != nil && g.Claims.EmpiricalAcceptance == Pass && g.Claims.AbsoluteAdequacy == AdequacyNotEstablished &&
			g.Claims.PopulationConfidence == ConfidenceNotClaimed
	}
	return false
}

// binomialCDF is Binomial(n, p)'s cumulative distribution, exactly (the
// probabilities by the recurrence of their ratios, in logs).
func binomialCDF(n int, p float64) []float64 {
	cdf := make([]float64, n+1)
	switch {
	case p <= 0:
		for k := range cdf {
			cdf[k] = 1
		}
		return cdf
	case p >= 1:
		cdf[n] = 1
		return cdf
	}
	lp, lq := math.Log(p), math.Log1p(-p)
	acc := 0.0
	for k := 0; k <= n; k++ {
		lg, _ := math.Lgamma(float64(n + 1))
		lk, _ := math.Lgamma(float64(k + 1))
		lnk, _ := math.Lgamma(float64(n - k + 1))
		acc += math.Exp(lg - lk - lnk + float64(k)*lp + float64(n-k)*lq)
		cdf[k] = min(acc, 1)
	}
	return cdf
}

// predictAcceptance is the probability that Gate B urgent passes on a
// group's pairs under the Bernoulli injection d: each seed's increase is
// its natural difference plus Binomial(served, δN/served)/N, drawn
// exactly (inverse cumulative distribution) for any number of served
// orders (audit 5C.4i I7). The natural counts are the replicate's less
// what it injected.
func predictAcceptance(g groupResult, d *manInject, draws int) float64 {
	if d == nil {
		if g.GateB["urgent"] == Pass {
			return 1
		}
		return 0
	}
	type seed struct {
		n, base, cand0 int64
		cdf            []float64
	}
	var ss []seed
	inj := map[int64]pairInfo{}
	for _, p := range g.Pairs {
		inj[int64(p.Seed)] = p
	}
	for _, s := range g.UrgentSeeds {
		p := inj[s.Seed]
		ss = append(ss, seed{s.Orders, s.Baseline, s.Candidate - int64(p.Injected), binomialCDF(p.Served, d.Effect*float64(s.Orders)/float64(p.Served))})
	}
	if len(ss) == 0 {
		return math.NaN() // nothing to draw: no prediction (audit 5C.4m M2)
	}
	r := rand.New(rand.NewPCG(7, 7)) // the prediction's own Monte Carlo
	ok := 0
	for range draws {
		var num float64
		for _, s := range ss {
			u := r.Float64()
			k, _ := slices.BinarySearchFunc(s.cdf, u, func(c, u float64) int { return cmp.Compare(c, u) })
			k = min(k, len(s.cdf)-1)
			num += float64(s.cand0+int64(k)-s.base) / float64(s.n)
		}
		if num/float64(len(ss)) <= 0.02+1e-12 {
			ok++
		}
	}
	return float64(ok) / float64(draws)
}

// cpInterval is the two-sided Clopper–Pearson interval of k of n at
// level 1 − alpha (betaQuantile, oc_clinical_test.go).
func cpInterval(k, n int, alpha float64) [2]float64 {
	if n == 0 {
		return [2]float64{0, 1}
	}
	lo, hi := 0.0, 1.0
	if k > 0 {
		lo = betaQuantile(float64(k), float64(n-k+1), alpha/2)
	}
	if k < n {
		hi = betaQuantile(float64(k+1), float64(n-k), 1-alpha/2)
	}
	return [2]float64{lo, hi}
}

// TestCPInterval: Clopper–Pearson against known values (0 of 2,500: the
// one-sided 95% upper bound 0.1197%, the two-sided interval at alpha
// 0.10).
func TestCPInterval(t *testing.T) {
	if iv := cpInterval(0, 2500, 0.10); math.Abs(iv[1]-0.0011976) > 2e-6 || iv[0] != 0 {
		t.Fatalf("%v", iv)
	}
	if iv := cpInterval(5, 10, 0.05); math.Abs(iv[0]-0.187086) > 1e-5 || math.Abs(iv[1]-0.812914) > 1e-5 {
		t.Fatalf("%v", iv)
	}
}

// analyzeFile judges a file of lines offline: never certifying, whatever
// the lines claim (audit 5C.4m M1). The settings are read as the run's
// (manInvocation). Its error is an unreadable, truncated or malformed
// file, or malformed settings; incomplete coverage is in the criteria.
func analyzeFile(m *manifestDoc, path string) (map[string]criterion, []string, error) {
	_, seeds, err := manInvocation(m)
	if err != nil {
		return nil, nil, err
	}
	tol, err := manifestTolerances(m)
	if err != nil {
		return nil, nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	var lines []groupResult
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<26)
	for sc.Scan() {
		var g groupResult
		dec := json.NewDecoder(bytes.NewReader(sc.Bytes()))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&g); err != nil {
			return nil, nil, fmt.Errorf("line %d: %v", len(lines)+1, err)
		}
		// One value a line, and nothing after it (audit 5C.4o O2).
		if _, err := dec.Token(); err != io.EOF {
			return nil, nil, fmt.Errorf("line %d: data after the line's value", len(lines)+1)
		}
		lines = append(lines, g)
	}
	if err := sc.Err(); err != nil {
		return nil, nil, err // an unreadable or truncated input is never judged
	}
	r := reservedCriteria(m, lines, tol, seeds)
	certify(r, certContext{Trusted: false})
	return r, coverageProblems(m, lines, seeds), nil
}

// TestReservedCriteria is an offline analysis of a file of lines:
// DT_MANIFEST_IN=<lines>, DT_CRITERIA_OUT=<file>, with the run's
// DT_MANIFEST_SEEDS and DT_MANIFEST_SCENARIOS. Its output is always
// non-certifying: a file's lines are assertions, not trusted
// measurements (audit 5C.4m M1); only the execution certifies
// (TestReservedManifestRun). Malformed settings or input fail before
// anything is written; incomplete coverage fails after the analysis is
// written.
func TestReservedCriteria(t *testing.T) {
	in, out := os.Getenv("DT_MANIFEST_IN"), os.Getenv("DT_CRITERIA_OUT")
	if in == "" || out == "" {
		t.Skip("set DT_MANIFEST_IN and DT_CRITERIA_OUT to analyze a manifest run's lines (non-certifying)")
	}
	m := loadManifest(t)
	r, bad, err := analyzeFile(m, in)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeCriteria(out, r); err != nil {
		t.Fatal(err)
	}
	t.Logf("non-certifying analysis: %s", r["certification"].Status)
	if len(bad) > 0 {
		t.Fatalf("incomplete input (%d problems, first %q)", len(bad), bad[0])
	}
}

// fakeRun is a complete, well-formed, uneventful run of the manifest:
// every scenario's complete groups and remainder, every replicate, every
// pair present and matched, Gate B passing, nothing rejected.
func fakeRun(m *manifestDoc) []groupResult {
	var lines []groupResult
	for _, s := range m.Scenarios {
		for g := 0; g*40 < s.Seeds; g++ {
			lo, hi := 40*g, min(s.Seeds, 40*g+40)
			for r := range max(1, s.Replicates) {
				x := groupResult{Scenario: s.ID, Group: g, Replicate: r, Seeds: [2]int{lo, hi - 1}, Remainder: hi-lo < 40, Valid: true,
					GateB:        map[string]string{"urgent": Pass, "soon": Pass, "routine": Pass},
					Deteriorated: map[string][]int64{"urgent": nil, "soon": nil, "routine": nil}, UnmatchedUrgent: -1,
					Matched:  map[string]int{"urgent": hi - lo, "soon": hi - lo, "routine": hi - lo},
					Increase: map[string]float64{"urgent": 0, "soon": 0, "routine": 0}}
				if s.Replicates == 0 {
					// The whole decision's measurements (audit 5C.4m M2).
					x.Claims = &AccessClaims{EmpiricalAcceptance: Pass, AbsoluteAdequacy: AdequacyNotEstablished, PopulationConfidence: ConfidenceNotClaimed}
					x.Severe = map[string][2]int64{"urgent": {0, 400}, "soon": {0, 1200}, "routine": {0, 2000}}
					x.WorstSoonPast = make([]int64, hi-lo)
				}
				for i := lo; i < hi; i++ {
					p := pairInfo{Seed: i}
					if s.Replicates > 0 {
						p.Eligible, p.Served = 30, 25
					}
					x.UrgentSeeds = append(x.UrgentSeeds, SeedDifference{Seed: int64(i), Orders: 30, Candidate: 3, Baseline: 3})
					if in := s.Candidate.Inject; in != nil && in.HarmProbability == 0 && (!in.GroupFirstOnly || i%40 == 0) {
						p.Harmed = true
						if in.Kind == "count" {
							p.Injected = in.Count
						}
					}
					x.Pairs = append(x.Pairs, p)
				}
				if c := s.Candidate.Inject; c != nil && lo%40 == 0 {
					// What the development runs measured, written out
					// rather than taken from idUnmatched, so that the
					// adjudicator's expectation is tested, not repeated.
					unmatched := map[string][]string{"idRemove": Priorities, "idRepeat": Priorities, "idChange": {"urgent"}}[c.Kind]
					for _, q := range unmatched {
						x.Matched[q]--
						x.GateB[q] = Insufficient // not paired: no decision
					}
					if slices.Contains(unmatched, "urgent") {
						x.UrgentSeeds = x.UrgentSeeds[1:] // no difference on an unmatched seed
					}
				}
				if x.Remainder {
					// Fewer seeds than the rule's: no Gate B decision.
					for _, q := range Priorities {
						x.GateB[q] = Insufficient
					}
				}
				gateBUrgent(&x)
				lines = append(lines, x)
			}
		}
	}
	return lines
}

// gateBUrgent sets a fixture line's urgent Gate B measurements to the
// rule's computation on its seed differences, its status kept (audit
// 5C.4o O1): each difference, its safeguard flag, the deteriorated seeds
// and the increase.
func gateBUrgent(x *groupResult) {
	ld := lineGateB()
	var diffs []float64
	x.Deteriorated["urgent"] = nil
	for i := range x.UrgentSeeds {
		d := &x.UrgentSeeds[i]
		d.Difference = float64(d.Candidate-d.Baseline) / float64(d.Orders)
		d.Deteriorated = mul(d.Candidate-d.Baseline, 10_000).Cmp(mul(bp(ld.MaxSeedIncrease["urgent"]), d.Orders)) > 0
		if d.Deteriorated {
			x.Deteriorated["urgent"] = append(x.Deteriorated["urgent"], d.Seed)
		}
		diffs = append(diffs, d.Difference)
	}
	x.Increase["urgent"] = 0
	if len(diffs) > 0 {
		x.Increase["urgent"], _ = meanSD(diffs)
	}
}

// flagUrgent makes seed deteriorate in a fixture line's urgent Gate B: a
// difference of 0.20, above the per-seed safeguard, with the measurements
// and a decided status following from it (gateBUrgent).
func flagUrgent(x *groupResult, seed int64) {
	for i := range x.UrgentSeeds {
		if x.UrgentSeeds[i].Seed == seed {
			x.UrgentSeeds[i].Candidate = x.UrgentSeeds[i].Baseline + 6
		}
	}
	gateBUrgent(x)
	if st := x.GateB["urgent"]; (st == Pass || st == Fail) && !meanAtMost(x.UrgentSeeds, bp(lineGateB().MaxIncrease["urgent"])) {
		x.GateB["urgent"] = Fail
		if !slices.Contains(x.Rules, "gateB:urgent") {
			x.Rejected, x.Rules = true, append(x.Rules, "gateB:urgent")
		}
	}
}

// failUrgent makes a fixture line's urgent Gate B FAIL as the rule does:
// 0.10 a seed, above the margin and at the per-seed safeguard.
func failUrgent(x *groupResult) {
	for j := range x.UrgentSeeds {
		x.UrgentSeeds[j].Candidate = x.UrgentSeeds[j].Baseline + 3
	}
	gateBUrgent(x)
	x.GateB["urgent"] = Fail
	if !slices.Contains(x.Rules, "gateB:urgent") {
		x.Rejected, x.Rules = true, append(x.Rules, "gateB:urgent")
	}
}

// witnessesMet makes a fake run's witnesses meet their expectations.
func witnessesMet(ls []groupResult) {
	all := func(x *groupResult) []int64 {
		var ss []int64
		for _, p := range x.Pairs {
			ss = append(ss, int64(p.Seed))
		}
		return ss
	}
	for i := range ls {
		x := &ls[i]
		switch x.Scenario {
		case "W-URG-LATE", "W-ROUT-348", "W-HORIZON":
			q := map[string]string{"W-URG-LATE": "urgent", "W-ROUT-348": "routine", "W-HORIZON": "soon"}[x.Scenario]
			x.Rejected, x.Rules, x.Violations = true, []string{"safety"}, map[string][]int64{"pastDue " + q: all(x)}
		case "W-PENDING":
			x.Rejected, x.Rules, x.Pending = true, []string{"pending"}, all(x)
		case "W-PARTIAL":
			x.Rejected, x.Rules, x.Failing, x.FailingByGate = true, []string{"allowance"}, all(x), map[string][]int64{"partial_window_ceiling soon": all(x)}
		case "W-ACC-LATE":
			x.Rejected, x.Rules, x.Failing, x.FailingByGate = true, []string{"allowance"}, all(x), map[string][]int64{GateTimelyAccess + " gated urgent": all(x)}
		case "W-LOST30", "W-PAT30", "W-MARGIN-ABOVE":
			failUrgent(x)
		case "W-ID-REMOVED", "W-ID-REPEATED", "W-ID-CHANGED":
			x.Rejected, x.Rules, x.GateB["urgent"], x.UnmatchedUrgent = true, []string{"provenance", "gateB:urgent"}, Insufficient, x.Pairs[0].Seed
			x.Provenance = []int64{int64(x.Pairs[0].Seed)}
		case "W-ACCESS65":
			x.Claims = &AccessClaims{EmpiricalAcceptance: Pass, AbsoluteAdequacy: AdequacyNotEstablished, PopulationConfidence: ConfidenceNotClaimed}
			x.Fractions = map[string][2]float64{"urgent": {0.66, 0.66}}
		}
	}
}

// The criteria's exact boundaries, on a well-formed fake run changed one
// way at a time (audit 5C.4i I5–I7): positive, negative, missing,
// malformed and boundary cases.
func TestReservedCriteriaRules(t *testing.T) {
	m := loadManifest(t)
	tol, err := manifestTolerances(m)
	if err != nil || tol.At21 == nil || *tol.At21 != 0.40 || *tol.At3 != 0.01 {
		t.Fatalf("the manifest's approved T5 tolerances: %v %+v", err, tol)
	}
	full := func(s *manScenario) int { return s.Seeds }
	judge := func(lines []groupResult, c string) string { return reservedCriteria(m, lines, tol, full)[c].Status }
	run := func() []groupResult { ls := fakeRun(m); witnessesMet(ls); return ls }
	edit := func(f func(ls []groupResult)) []groupResult { ls := run(); f(ls); return ls }
	find := func(ls []groupResult, id string, g, r int) *groupResult {
		for i := range ls {
			if ls[i].Scenario == id && ls[i].Group == g && ls[i].Replicate == r {
				return &ls[i]
			}
		}
		panic(id)
	}
	reject := func(ls []groupResult, id string, groups ...int) {
		for _, g := range groups {
			x := find(ls, id, g, 0)
			x.Rejected, x.Rules = true, []string{"allowance"}
		}
	}
	// The uneventful run: every judged criterion passes (T2 inconclusive:
	// no harmed seed), T5a conforms on its trivial predictions.
	base := reservedCriteria(m, run(), tol, full)
	for _, c := range []string{"T1", "T2-conformance", "T2G", "T3", "T4", "T6", "T7", "T8"} {
		if base[c].Status != Pass {
			t.Errorf("uneventful %s: %s %v", c, base[c].Status, base[c].Detail)
		}
	}
	for _, x := range []struct {
		name, c, want string
		ls            []groupResult
	}{
		{"T1 3 of 62", "T1", Pass, edit(func(ls []groupResult) { reject(ls, "S-A85", 0, 1, 2) })},
		{"T1 4 of 62", "T1", Fail, edit(func(ls []groupResult) { reject(ls, "S-A85", 0, 1, 2, 3) })},
		{"T1 3 of 62 and a rejected remainder", "T1", Pass, edit(func(ls []groupResult) { reject(ls, "S-A85", 0, 1, 2, 62) })},
		{"T7 4 of 62 windows", "T7", Fail, edit(func(ls []groupResult) {
			for g := range 4 {
				find(ls, "S-B85", g, 0).RawInsuff = map[string][]int64{GateSevereWeek + " gated soon": {int64(40 * g)}}
			}
		})},
		{"T7 3 pending", "T7", Pass, edit(func(ls []groupResult) {
			for g := range 3 {
				find(ls, "S-B85", g, 0).Pending = []int64{int64(40 * g)}
			}
		})},
		{"T6 19 flags", "T6", Pass, edit(func(ls []groupResult) {
			for g := range 19 {
				flagUrgent(find(ls, "S-C85", g, 0), int64(40*g))
			}
		})},
		{"T6 20 flags, one in a remainder", "T6", Fail, edit(func(ls []groupResult) {
			for g := range 19 {
				flagUrgent(find(ls, "S-C85", g, 0), int64(40*g))
			}
			flagUrgent(find(ls, "S-C85", 62, 0), 2480)
		})},
		{"T3 one violation in a remainder", "T3", Fail, edit(func(ls []groupResult) {
			find(ls, "S-Cs85", 62, 0).Violations = map[string][]int64{"pastDue soon": {2490}}
		})},
		{"T2G soon 1.5%, routine 0%, not rejected", "T2G", Fail, edit(func(ls []groupResult) {
			for g := range 25 {
				find(ls, "L10-85", g, 0).Severe = map[string][2]int64{"urgent": {0, 400}, "soon": {3, 200}, "routine": {0, 10000}}
			}
		})},
		{"T2G soon 1.5%, rejected in 23 of 25", "T2G", Pass, edit(func(ls []groupResult) {
			for g := range 25 {
				find(ls, "L10-85", g, 0).Severe = map[string][2]int64{"urgent": {0, 400}, "soon": {3, 200}, "routine": {0, 10000}}
			}
			reject(ls, "L10-85", 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22)
		})},
		{"T8 a missing group", "T8", "inconclusive", edit(func(ls []groupResult) { find(ls, "W-ACCESS65", 24, 0).Scenario = "W-ACCESS65x" })},
		{"T8 an invalid record", "T8", "inconclusive", edit(func(ls []groupResult) { find(ls, "HV", 3, 0).Valid = false })},
		{"T8 a wrong seed", "T8", "inconclusive", edit(func(ls []groupResult) { find(ls, "H15", 3, 0).Pairs[5].Seed = 7 })},
		{"T8 a count injection short", "T8", "inconclusive", edit(func(ls []groupResult) { find(ls, "W-LOST30", 3, 0).Pairs[5].Injected = 29 })},
		{"T8 an unmatched pair", "T8", "inconclusive", edit(func(ls []groupResult) { find(ls, "S-A92", 3, 0).Matched["soon"] = 39 })},
	} {
		if got := judge(x.ls, x.c); got != x.want {
			t.Errorf("%s: %s %s, want %s: %v", x.name, x.c, got, x.want, reservedCriteria(m, x.ls, tol, full)[x.c].Detail)
		}
	}
	// Duplicated, missing and truncated output never confirms.
	dup := append(run(), run()[0])
	if r := reservedCriteria(m, dup, tol, full); r["T1"].Status != "inconclusive" || r["T8"].Status != "inconclusive" {
		t.Errorf("a duplicated group: T1 %s T8 %s", r["T1"].Status, r["T8"].Status)
	}
	ls := run()
	var cut []groupResult
	for _, g := range ls {
		if g.Scenario != "W-ACCESS65" || g.Group == 0 {
			cut = append(cut, g)
		}
	}
	if r := reservedCriteria(m, cut, tol, full); r["T4"].Status != "inconclusive" || r["T8"].Status != "inconclusive" {
		t.Errorf("a truncated witness (1 of 25 groups): T4 %s T8 %s", r["T4"].Status, r["T8"].Status)
	}
	if r := reservedCriteria(m, append(run(), groupResult{Scenario: "nowhere"}), tol, full); r["T8"].Status != "inconclusive" {
		t.Errorf("an unknown scenario: T8 %s", r["T8"].Status)
	}
	if r := reservedCriteria(m, nil, tol, full); r["T1"].Status != "inconclusive" || r["T8"].Status != "inconclusive" || r["T4"].Status != "inconclusive" {
		t.Errorf("no output: %s %s %s", r["T1"].Status, r["T8"].Status, r["T4"].Status)
	}
	// T2: harmed-seed detection, at 0.95 exactly, below it, and too few.
	harm := func(n, flagged int) []groupResult {
		ls := fakeRun(m)
		k := 0
		for i := range ls {
			if ls[i].Scenario != "H15" {
				continue
			}
			for j := range ls[i].Pairs {
				if k < n {
					ls[i].Pairs[j].Harmed = true
					if k < flagged {
						s := int64(ls[i].Pairs[j].Seed)
						flagUrgent(&ls[i], s)
						ls[i].Failing = append(ls[i].Failing, s)
					}
				}
				k++
			}
			if len(ls[i].Failing) > 4 {
				ls[i].Rejected, ls[i].Rules = true, append(ls[i].Rules, "allowance")
			}
		}
		return ls
	}
	for _, x := range []struct {
		n, f int
		want string
	}{{200, 190, Pass}, {200, 189, Fail}, {199, 199, "inconclusive"}} {
		if got := judge(harm(x.n, x.f), "T2"); got != x.want {
			t.Errorf("T2 %d of %d: %s", x.f, x.n, got)
		}
	}
	bad := harm(200, 190)
	for i := range bad {
		if bad[i].Scenario == "H15" && bad[i].Rejected {
			bad[i].Rejected, bad[i].Rules = false, nil
			break
		}
	}
	if judge(bad, "T2-conformance") != Fail {
		t.Error("T2 conformance: an unrejected group over the allowance passes")
	}
	// T4: the targeted rule, not any rejection.
	all := func(x *groupResult) []int64 {
		var ss []int64
		for _, p := range x.Pairs {
			ss = append(ss, int64(p.Seed))
		}
		return ss
	}
	ls = run()
	if judge(ls, "T4") != Pass {
		t.Fatalf("every witness as specified: %v", reservedCriteria(m, ls, tol, full)["T4"])
	}
	for name, f := range map[string]func(x *groupResult){
		"W-PARTIAL by an unrelated gate":  func(x *groupResult) { x.FailingByGate = map[string][]int64{"unrelated": all(x)} },
		"W-ACC-LATE by an unrelated gate": func(x *groupResult) { x.FailingByGate = map[string][]int64{"unrelated": all(x)} },
		"W-HORIZON by another limit":      func(x *groupResult) { x.Violations = map[string][]int64{"pastDue urgent": all(x)} },
		"W-ID-CHANGED on another seed":    func(x *groupResult) { x.UnmatchedUrgent = x.Pairs[1].Seed },
		"W-ID-REMOVED by Gate B alone (no provenance failure)": func(x *groupResult) {
			x.Provenance, x.Rules = nil, []string{"gateB:urgent"}
		},
		"W-ID-REPEATED with provenance failing on two seeds": func(x *groupResult) {
			x.Provenance = []int64{int64(x.Pairs[0].Seed), int64(x.Pairs[1].Seed)}
		},
		"W-ACCESS65 with Gate B failing":                failUrgent,
		"W-ACCESS65 without the disclosed loss":         func(x *groupResult) { x.Fractions = nil },
		"W-ACCESS65 disclosing a loss below 65 percent": func(x *groupResult) { x.Fractions = map[string][2]float64{"urgent": {0.3, 0.66}} },
	} {
		id := strings.Fields(name)[0]
		ls := run()
		for i := range ls {
			if ls[i].Scenario == id && ls[i].Group == 3 {
				f(&ls[i])
			}
		}
		if got := judge(ls, "T4"); got != Fail {
			t.Errorf("%s: T4 %s", name, got)
		}
	}
	// T5b at the approved boundaries: 200 of 500 at +2.1 and 5 at +3 pass;
	// 201, or 6, fail; without approved tolerances it is not locked.
	t5 := func(n21, n3 int) []groupResult {
		ls := fakeRun(m)
		k := map[string]int{}
		for i := range ls {
			lim := map[string]int{"NM2.1": n21, "NM3": n3}[ls[i].Scenario]
			if ls[i].Scenario == "NM2.1" || ls[i].Scenario == "NM3" {
				if k[ls[i].Scenario] >= lim {
					failUrgent(&ls[i])
				}
				k[ls[i].Scenario]++
			}
		}
		return ls
	}
	for _, x := range []struct {
		n21, n3 int
		want    string
	}{{200, 5, Pass}, {201, 5, Fail}, {200, 6, Fail}} {
		if got := judge(t5(x.n21, x.n3), "T5b"); got != x.want {
			t.Errorf("T5b %d/500, %d/500: %s", x.n21, x.n3, got)
		}
	}
	if r := reservedCriteria(m, fakeRun(m), t5Tolerance{}, full)["T5b"]; r.Status != "not locked" {
		t.Errorf("T5b without tolerances: %s", r.Status)
	}
}

// The exact binomial prediction (audit 5C.4i I7): its distribution
// matches the exact probabilities above 200 orders (201 at 0.021: no
// injection with probability 0.01404, where a normal approximation gave
// 0.03359), and its cumulative distribution ends at 1.
func TestBinomialCDF(t *testing.T) {
	c := binomialCDF(201, 0.021)
	if math.Abs(c[0]-0.014042) > 2e-5 || math.Abs(c[len(c)-1]-1) > 1e-9 {
		t.Fatalf("P(0) %.6f, P(≤201) %.9f", c[0], c[len(c)-1])
	}
	// Equal seeds, no natural difference, no baseline loss: the
	// prediction is P{Binomial(40N, δ) ≤ 0.02 · 40N}, here N = 150 at 2.1
	// points: 0.314.
	g := groupResult{GateB: map[string]string{}}
	for i := range 40 {
		g.UrgentSeeds = append(g.UrgentSeeds, SeedDifference{Seed: int64(i), Orders: 150})
		g.Pairs = append(g.Pairs, pairInfo{Seed: i, Served: 150})
	}
	if p := predictAcceptance(g, &manInject{Kind: "bernoulli", Priority: "urgent", Effect: 0.021}, 20_000); math.Abs(p-0.314088) > 0.012 {
		t.Fatalf("prediction %.4f, exact 0.3141", p)
	}
}

// The harness refuses the reserved family, and any other runs.
func TestReservedManifestRefusesReserved(t *testing.T) {
	if familyAllowed(reservedFamily) == nil || familyAllowed(devManifestFamily) != nil || familyAllowed(1) != nil {
		t.Fatal("family check")
	}
}

// Practices whose demand differs (the audit's A85 against seasonal
// B85) are never constructed as a pair: a construction error, never an
// unmatched comparison treated as paired.
func TestReservedManifestUnpaired(t *testing.T) {
	pol := manifestPolicy(t)
	s := &manScenario{K: 99, ID: "A85-B85", Seeds: 40, Baseline: manSide{Practice: "A85"}, Candidate: manSide{Practice: "B85"}}
	if b, c, info := manPair(s, 1, 0, pol, manPractices()); b != nil || c != nil || !strings.Contains(info.Error, "not paired") {
		t.Fatalf("%+v", info)
	}
}

// Extending the observation from 31 to 45 days past the horizon leaves the
// manifest's practices unchanged up to the earlier end (audit 5C.4i I4):
// demand, modalities, acquisitions, documentation, reading and final
// times of every study acquired by then, and every order's request and
// resolution by then. The committed model (booking-stream study draws,
// kept for its pinned digests) is shown changing, the defect repaired.
func TestFidelityFollowUpPrefixStable(t *testing.T) {
	if testing.Short() {
		t.Skip("short")
	}
	p := manifestPolicy(t)
	p31 := *p
	c := *p.Clinical
	c.FollowUpDays = 0
	p31.Clinical = &c
	compare := func(q ocFidelity) (studies, changed, orders int) {
		gen := func(pol *Policy) *Record {
			r, _ := q.recordPaired(ocRNG{rand.New(rand.NewPCG(3, 0))}, ocRNG{rand.New(rand.NewPCG(3, 10000000))}, pol)
			return r
		}
		a, b := gen(&p31), gen(p)
		if b.ObservedUntilSec-b.HorizonEndSec != 45*daySec || a.ObservedUntilSec-a.HorizonEndSec != 31*daySec {
			t.Fatalf("observation %d / %d days", (a.ObservedUntilSec-a.HorizonEndSec)/daySec, (b.ObservedUntilSec-b.HorizonEndSec)/daySec)
		}
		E := a.ObservedUntilSec
		by := map[string]Study{}
		for _, s := range b.Studies {
			by[s.Order] = s
		}
		cut := func(x int64) int64 {
			if x > E {
				return 0
			}
			return x
		}
		for _, s := range a.Studies {
			x, ok := by[s.Order]
			studies++
			// A study unfinished at the earlier end has its claim and read
			// cleared there (the model's convention): compare them when final.
			read := s.FinalSec == 0 || x.ReadStartSec == s.ReadStartSec
			if !ok || x.Modality != s.Modality || x.AcquiredSec != s.AcquiredSec || cut(x.EnteredSec) != cut(s.EnteredSec) ||
				!read || cut(x.FinalSec) != s.FinalSec {
				changed++
			}
		}
		ob := map[string]Order{}
		for _, o := range b.Orders {
			ob[o.ID] = o
		}
		for _, o := range a.Orders {
			x := ob[o.ID]
			orders++
			if x.Origin != o.Origin || x.Modality != o.Modality || x.ReceivedSec != o.ReceivedSec || x.WantedSec != o.WantedSec ||
				(o.ResolvedSec != 0 && x.ResolvedSec != o.ResolvedSec) {
				t.Fatalf("order %s differs: %+v / %+v", o.ID, o, x)
			}
		}
		return
	}
	q := manPractices()["A85"]
	if n, changed, orders := compare(q); changed != 0 || n < 5000 || orders < 5000 {
		t.Fatalf("prefix-stable: %d of %d studies changed (%d orders)", changed, n, orders)
	}
	q.PrefixStable = false
	if n, changed, _ := compare(q); changed < n/2 {
		t.Fatalf("the committed model's draws: only %d of %d studies changed", changed, n)
	}
}
