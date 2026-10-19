package eval

import (
	"bufio"
	"cmp"
	"encoding/json"
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// The reserved protocol's executable scenario manifest (Slice 5C.4h,
// docs/calibration/clinical-5c4h/scenario-manifest.json): every
// scenario's practices, paired demand, booking and injection streams,
// injection, seeds and groups, built here exactly as the manifest says
// and judged by the production evaluator. The reserved family is refused
// in this build: the manifest runs only on a non-reserved family (the
// development checks, PCG(20261021, ·)).

const manifestPath = "../../docs/calibration/clinical-5c4h/scenario-manifest.json"

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
		q.Takeover, q.ExogenousModality = fidAlignedTakeover, true
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
	Eligible int     `json:"eligible,omitempty"` // the injected priority's eligible orders
	Served   int     `json:"served,omitempty"`
	Error    string  `json:"constructionError,omitempty"`
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
	cand, _ = qc.recordPaired(pcg(uint64(2*i)), pcg(cb), pol)
	base.Seed, cand.Seed = int64(i), int64(i)
	base.Plan, cand.Plan = "B", "C"
	if in := s.Baseline.Inject; in != nil {
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
			r.Orders[es[0]].Modality = "US" + r.Orders[es[0]].Modality // a request changed under its origin
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
	Valid     bool   `json:"valid"`
	Rejected  bool   `json:"rejected"`
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
	g := groupResult{GateB: map[string]string{}, Increase: map[string]float64{}, Matched: map[string]int{}, Deteriorated: map[string][]int64{}}
	pc, pb := planReport(p, "C", cand), planReport(p, "B", base)
	g.Valid = pc.Valid && pb.Valid
	if !g.Valid {
		g.Rejected, g.Rules = true, []string{"integrity"}
		return g
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
			g.UrgentSeeds = x.Seeds
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
	g := groupResult{Valid: true, GateB: map[string]string{}, Increase: map[string]float64{}, Matched: map[string]int{}, Deteriorated: map[string][]int64{}}
	for _, x := range lostDemandGates(p, cand, base) {
		g.GateB[x.Priority], g.Increase[x.Priority], g.Matched[x.Priority] = x.Status, x.Increase, x.MatchedSeeds
		g.Deteriorated[x.Priority] = x.Deteriorated
		if x.Status != Pass {
			g.Rules = append(g.Rules, "gateB:"+x.Priority)
		}
		if x.Priority == "urgent" {
			g.UrgentSeeds = x.Seeds
		}
	}
	g.Rejected = len(g.Rules) > 0
	return g
}

// manRunScenario runs scenario s on family F, n seeds (whole groups
// only; the remainder is per-seed only), writing one JSON line a group
// (a replicate) to w. Records are dropped after each group.
func manRunScenario(t *testing.T, m *manifestDoc, s *manScenario, F uint64, n int, pol *Policy, w *bufio.Writer) {
	if err := familyAllowed(F); err != nil {
		t.Fatal(err)
	}
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
			res := groupResult{Scenario: s.ID, Group: g, Replicate: r, Seeds: [2]int{lo, hi - 1}}
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
			res.Pairs, res.Errors = infos, errs
			raw, _ := json.Marshal(res)
			w.Write(append(raw, '\n'))
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
		w.Flush()
	}
	w.Flush()
}

// manifestPolicy is the fixture policy with the candidate's clinical
// section, as every protocol evaluation uses it.
func manifestPolicy(t *testing.T) *Policy {
	p, _ := candidatePolicy(t)
	return p
}

// TestReservedManifestRun runs the manifest on a non-reserved family:
// DT_MANIFEST_OUT=<file>, DT_MANIFEST_FAMILY (default the development
// checks' 20261021; the reserved family is refused), DT_MANIFEST_SEEDS
// (seeds a scenario, default 40) and DT_MANIFEST_SCENARIOS (IDs).
func TestReservedManifestRun(t *testing.T) {
	out := os.Getenv("DT_MANIFEST_OUT")
	if out == "" {
		t.Skip("set DT_MANIFEST_OUT=<file> to run the scenario manifest on a non-reserved family")
	}
	F := uint64(devManifestFamily)
	if v := os.Getenv("DT_MANIFEST_FAMILY"); v != "" {
		x, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		F = x
	}
	n := 40
	if v := os.Getenv("DT_MANIFEST_SEEDS"); v != "" {
		n, _ = strconv.Atoi(v)
	}
	var only []string
	if v := os.Getenv("DT_MANIFEST_SCENARIOS"); v != "" {
		only = strings.Split(v, ",")
	}
	m := loadManifest(t)
	pol := manifestPolicy(t)
	f, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	for i := range m.Scenarios {
		s := &m.Scenarios[i]
		if len(only) > 0 && !slices.Contains(only, s.ID) {
			continue
		}
		manRunScenario(t, m, s, F, min(n, s.Seeds), pol, w)
		t.Logf("%s done", s.ID)
	}
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
		var buf strings.Builder
		w := bufio.NewWriter(&buf)
		manRunScenario(t, m, &s, 1, 2, pol, w)
		var g groupResult
		if err := json.Unmarshal([]byte(strings.SplitN(buf.String(), "\n", 2)[0]), &g); err != nil {
			t.Fatal(err)
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

// reservedCriteria judges a manifest run's group lines by the protocol's
// §4, exactly: one result a criterion. Without approved T5 tolerances
// T5b is "not locked" and the run cannot confirm.
func reservedCriteria(m *manifestDoc, lines []groupResult, tol t5Tolerance) map[string]criterion {
	by := map[string][]groupResult{}
	for _, g := range lines {
		by[g.Scenario] = append(by[g.Scenario], g)
	}
	scen := map[string]*manScenario{}
	for i := range m.Scenarios {
		scen[m.Scenarios[i].ID] = &m.Scenarios[i]
	}
	has := func(s *manScenario, c string) bool { return slices.Contains(s.Criteria, c) }
	out := map[string]criterion{}
	errs := func(ids ...string) int {
		n := 0
		for _, id := range ids {
			for _, g := range by[id] {
				n += len(g.Errors)
			}
		}
		return n
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
	// T1 and T7: at most 3 of 62 groups (5% of the groups, rounded down).
	t1, t7 := map[string]any{}, map[string]any{}
	f1, f7 := false, false
	for _, id := range stable {
		gs := by[id]
		limit := len(gs) * 5 / 100
		rej, ins := 0, 0
		rules := map[string]int{}
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
				}
			}
			if window {
				ins++
			}
		}
		f1, f7 = f1 || rej > limit, f7 || ins > limit
		t1[id] = map[string]any{"groups": len(gs), "rejected": rej, "limit": limit, "byRule": rules, "interval": cpInterval(rej, len(gs), 0.05)}
		t7[id] = map[string]any{"groups": len(gs), "insufficient": ins, "limit": limit}
	}
	out["T1"] = criterion{verdict(f1, errs(stable...) > 0), t1}
	out["T7"] = criterion{verdict(f7, errs(stable...) > 0), t7}
	// T3 and T6: per seed, all seeds.
	t3, t6 := map[string]any{}, map[string]int{}
	f3, n6 := false, 0
	for _, id := range stable {
		v := map[string]int{}
		for _, g := range by[id] {
			for k, seeds := range g.Violations {
				v[k] += len(seeds)
				f3 = true
			}
			for q, seeds := range g.Deteriorated {
				t6[q] += len(seeds)
			}
			n6 += len(g.Pairs)
		}
		t3[id] = v
	}
	out["T3"] = criterion{verdict(f3, errs(stable...) > 0), t3}
	f6 := false
	for _, q := range Priorities {
		f6 = f6 || t6[q] > 19
	}
	out["T6"] = criterion{verdict(f6, errs(stable...) > 0), map[string]any{"seeds": n6, "flagged": t6, "limit": 19}}
	// T2: detection of harmed seeds, pooled; conformance; rejection
	// reported against the allowance's prediction.
	harmed, flagged, conform := 0, 0, true
	t2rep := map[string]any{}
	for _, id := range t2 {
		rej := 0
		for _, g := range by[id] {
			if g.Rejected {
				rej++
			}
			for _, p := range g.Pairs {
				if !p.Harmed {
					continue
				}
				harmed++
				hit := slices.Contains(g.Deteriorated["urgent"], int64(p.Seed))
				if hit {
					flagged++
				}
				if hit && !slices.Contains(g.Failing, int64(p.Seed)) {
					conform = false
				}
			}
			if len(g.Failing)+len(g.Insufficient) > 4 && !g.Rejected {
				conform = false
			}
		}
		t2rep[id] = map[string]any{"groups": len(by[id]), "rejected": rej}
	}
	det := 0.0
	if harmed > 0 {
		det = float64(flagged) / float64(harmed)
	}
	out["T2"] = criterion{verdict(harmed >= 200 && det < 0.95, harmed < 200 || errs(t2...) > 0),
		map[string]any{"harmed": harmed, "flagged": flagged, "detection": det, "rejection": t2rep, "allowancePrediction": 0.3710}}
	out["T2-conformance"] = criterion{verdict(!conform, false), map[string]any{"holds": conform}}
	// T2G: materiality from the run, then rejection of the material.
	t2gRep := map[string]any{}
	f2g := false
	for _, id := range t2g {
		var sev, acq, inc, n float64
		var worst []int64
		rej := 0
		for _, g := range by[id] {
			for _, q := range []string{"soon", "routine"} {
				sev, acq = sev+float64(g.Severe[q][0]), acq+float64(g.Severe[q][1])
			}
			worst = append(worst, g.WorstSoonPast...)
			inc, n = inc+g.Increase["urgent"]+g.Increase["soon"]+g.Increase["routine"], n+3
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
		for _, q := range Priorities {
			var s float64
			for _, g := range by[id] {
				s += g.Increase[q]
			}
			maxInc = max(maxInc, s/float64(max(1, len(by[id]))))
		}
		material := (acq > 0 && sev/acq >= 0.01) || med >= 48*60 || maxInc >= 0.03
		if material && rej < len(by[id])-2 {
			f2g = true
		}
		t2gRep[id] = map[string]any{"material": material, "severeShare": sev / max(acq, 1), "worstSoonMedianMin": med, "maxIncrease": maxInc, "rejected": rej, "groups": len(by[id])}
	}
	out["T2G"] = criterion{verdict(f2g, errs(t2g...) > 0), t2gRep}
	// T4: every group of every witness as expected.
	t4 := map[string]any{}
	f4 := false
	for _, id := range wit {
		ok := 0
		for _, g := range by[id] {
			if witnessMet(id, g) {
				ok++
			}
		}
		f4 = f4 || ok != len(by[id])
		t4[id] = map[string]any{"groups": len(by[id]), "met": ok}
	}
	out["T4"] = criterion{verdict(f4, errs(wit...) > 0), t4}
	// T5: conformance with the prediction, and the tolerances.
	t5 := map[string]any{}
	f5a := false
	rate := map[string]float64{}
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
		for k, x := range means {
			gm = append(gm, x[0]/x[1])
			pm += pred[k] / float64(len(means))
		}
		mu, sd := meanSD(gm)
		se := sd / math.Sqrt(float64(max(1, len(gm))))
		obs := float64(pass) / float64(max(1, total))
		rate[id] = obs
		ok := math.Abs(obs-pm) <= 4*se+0.02
		f5a = f5a || !ok
		t5[id] = map[string]any{"evaluations": total, "accepted": pass, "rate": obs, "groupMean": mu, "clusterSE": se, "predicted": pm, "conforms": ok}
	}
	few := false // a cluster standard error needs two groups at least
	for _, id := range nm {
		groups := map[int]bool{}
		for _, g := range by[id] {
			groups[g.Group] = true
		}
		few = few || len(groups) < 2
	}
	out["T5a"] = criterion{verdict(f5a && !few, few || errs(nm...) > 0), t5}
	switch {
	case tol.At21 == nil || tol.At3 == nil:
		out["T5b"] = criterion{"not locked", map[string]any{"reason": "the owner has not approved the T5 tolerances", "rate2.1": rate["NM2.1"], "rate3": rate["NM3"]}}
	default:
		out["T5b"] = criterion{verdict(rate["NM2.1"] > *tol.At21 || rate["NM3"] > *tol.At3, false),
			map[string]any{"rate2.1": rate["NM2.1"], "tolerance2.1": *tol.At21, "rate3": rate["NM3"], "tolerance3": *tol.At3}}
	}
	// T8: construction integrity everywhere.
	t8 := map[string]any{}
	f8 := false
	for _, s := range m.Scenarios {
		e, unmatched := 0, 0
		for _, g := range by[s.ID] {
			e += len(g.Errors)
			want := len(g.Pairs)
			if c := s.Candidate.Inject; c != nil && strings.HasPrefix(c.Kind, "id") {
				want--
			}
			for _, q := range Priorities {
				if g.Valid && g.Matched[q] < want {
					unmatched++
				}
			}
			for _, p := range g.Pairs {
				in := s.Candidate.Inject
				if in != nil && in.Kind == "quota" && p.Harmed && (in.OnVolume == 0 || p.Volume == in.OnVolume) &&
					p.Injected != int(math.Ceil(in.Effect*float64(p.Eligible)-1e-9)) {
					e++
				}
				if in != nil && in.Kind == "count" && p.Injected != in.Count {
					e++
				}
			}
		}
		if e > 0 || unmatched > 0 || len(by[s.ID]) == 0 {
			f8 = true
			t8[s.ID] = map[string]any{"errors": e, "unmatched": unmatched, "groups": len(by[s.ID])}
		}
	}
	out["T8"] = criterion{verdict(false, f8), t8}
	return out
}

// witnessMet: a T4 group meets its scenario's expectation (protocol §4).
func witnessMet(id string, g groupResult) bool {
	rule := func(r string) bool { return g.Rejected && slices.Contains(g.Rules, r) }
	switch id {
	case "W-URG-LATE", "W-ROUT-348", "W-HORIZON":
		return rule("safety")
	case "W-PENDING":
		return rule("pending")
	case "W-ACC-LATE", "W-PARTIAL":
		return rule("allowance") && len(g.Failing) == len(g.Pairs)
	case "W-LOST30", "W-PAT30", "W-MARGIN-ABOVE":
		return rule("gateB:urgent") && g.GateB["urgent"] == Fail
	case "W-ID-REMOVED", "W-ID-REPEATED", "W-ID-CHANGED":
		return rule("gateB:urgent") && g.GateB["urgent"] == Insufficient && g.Matched["urgent"] == len(g.Pairs)-1
	case "W-MARGIN-AT":
		return g.GateB["urgent"] == Pass
	case "W-ACCESS65":
		return g.GateB["urgent"] != Pass || (g.Claims != nil && g.Claims.AbsoluteAdequacy == AdequacyNotEstablished && g.Claims.PopulationConfidence == ConfidenceNotClaimed)
	}
	return false
}

// predictAcceptance is the probability that Gate B urgent passes on a
// group's pairs under the Bernoulli injection d: each seed's increase is
// its natural difference plus Binomial(served, δN/served)/N. The
// natural counts are the replicate's less what it injected.
func predictAcceptance(g groupResult, d *manInject, draws int) float64 {
	if d == nil {
		if g.GateB["urgent"] == Pass {
			return 1
		}
		return 0
	}
	type seed struct {
		n, served, base, cand0 int64
	}
	var ss []seed
	inj := map[int64]pairInfo{}
	for _, p := range g.Pairs {
		inj[int64(p.Seed)] = p
	}
	for _, s := range g.UrgentSeeds {
		p := inj[s.Seed]
		ss = append(ss, seed{s.Orders, int64(p.Served), s.Baseline, s.Candidate - int64(p.Injected)})
	}
	r := rand.New(rand.NewPCG(7, 7)) // the prediction's own Monte Carlo
	ok := 0
	for range draws {
		var num float64
		for _, s := range ss {
			p := d.Effect * float64(s.n) / float64(s.served)
			k := (ocRNG{r}).binom(int(s.served), p)
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

// TestReservedCriteria judges a run's lines: DT_MANIFEST_IN=<lines>,
// DT_CRITERIA_OUT=<file>. The T5 tolerances are the approved ones (none
// in this build).
func TestReservedCriteria(t *testing.T) {
	in, out := os.Getenv("DT_MANIFEST_IN"), os.Getenv("DT_CRITERIA_OUT")
	if in == "" || out == "" {
		t.Skip("set DT_MANIFEST_IN and DT_CRITERIA_OUT to judge a manifest run")
	}
	f, err := os.Open(in)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var lines []groupResult
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<26)
	for sc.Scan() {
		var g groupResult
		if err := json.Unmarshal(sc.Bytes(), &g); err != nil {
			t.Fatal(err)
		}
		lines = append(lines, g)
	}
	res := reservedCriteria(loadManifest(t), lines, t5Tolerance{})
	raw, _ := json.MarshalIndent(res, "", " ")
	if err := os.WriteFile(out, append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

// The criteria's exact boundaries on fabricated group lines: T1 and T7
// at 3 of 62 groups (pass) and 4 (fail); T6 at 19 flags in 10,000 (pass)
// and 20 (fail); T2 at 190 of 200 harmed seeds flagged (0.95, pass), 189
// (fail), and fewer than 200 harmed (inconclusive); T3 at one violating
// seed (fail); T4 at one group missing its rule (fail); T5b not locked
// without approved tolerances, judged with them; T8 inconclusive on a
// construction error.
func TestReservedCriteriaRules(t *testing.T) {
	m := loadManifest(t)
	ok := func(id string, g int) groupResult {
		gr := groupResult{Scenario: id, Group: g, Valid: true, GateB: map[string]string{"urgent": Pass, "soon": Pass, "routine": Pass},
			Matched: map[string]int{"urgent": 40, "soon": 40, "routine": 40}, Deteriorated: map[string][]int64{}}
		for i := range 40 {
			gr.Pairs = append(gr.Pairs, pairInfo{Seed: 40*g + i})
		}
		return gr
	}
	base := func(rejected, insuff, flags int) []groupResult {
		var lines []groupResult
		for _, s := range m.Scenarios {
			if !slices.Contains(s.Criteria, "T1") {
				continue
			}
			for g := range 62 {
				x := ok(s.ID, g)
				if g < rejected {
					x.Rejected, x.Rules = true, []string{"allowance"}
				}
				if g < insuff {
					x.RawInsuff = map[string][]int64{GateSevereWeek + " gated soon": {int64(40 * g)}}
				}
				if s.ID == "S-A85" && g < flags {
					x.Deteriorated["urgent"] = []int64{int64(40 * g)}
				}
				lines = append(lines, x)
			}
		}
		return lines
	}
	st := func(lines []groupResult, c string) string { return reservedCriteria(m, lines, t5Tolerance{})[c].Status }
	for _, x := range []struct {
		lines []groupResult
		c     string
		want  string
	}{
		{base(3, 0, 0), "T1", Pass}, {base(4, 0, 0), "T1", Fail},
		{base(0, 3, 0), "T7", Pass}, {base(0, 4, 0), "T7", Fail},
		{base(0, 0, 19), "T6", Pass}, {base(0, 0, 20), "T6", Fail},
		{base(0, 0, 0), "T3", Pass},
	} {
		if got := st(x.lines, x.c); got != x.want {
			t.Errorf("%s: %s, want %s", x.c, got, x.want)
		}
	}
	lines := base(0, 0, 0)
	lines[0].Violations = map[string][]int64{"pastDue soon": {3}}
	if st(lines, "T3") != Fail {
		t.Error("T3: one violating seed passes")
	}
	harm := func(n, flagged int) []groupResult {
		var lines []groupResult
		for g := range (n + 39) / 40 {
			x := ok("H15", g)
			for i := range x.Pairs {
				if k := 40*g + i; k < n {
					x.Pairs[i].Harmed = true
					if k < flagged {
						x.Deteriorated["urgent"] = append(x.Deteriorated["urgent"], int64(x.Pairs[i].Seed))
						x.Failing = append(x.Failing, int64(x.Pairs[i].Seed))
					}
				}
			}
			x.Rejected = len(x.Failing) > 4
			lines = append(lines, x)
		}
		return lines
	}
	for _, x := range []struct {
		n, f int
		want string
	}{{200, 190, Pass}, {200, 189, Fail}, {199, 199, "inconclusive"}} {
		if got := st(harm(x.n, x.f), "T2"); got != x.want {
			t.Errorf("T2 %d of %d: %s", x.f, x.n, got)
		}
	}
	bad := harm(200, 190)
	bad[0].Rejected = false // 40 failing seeds, not rejected
	if st(bad, "T2-conformance") != Fail {
		t.Error("T2 conformance: an unrejected group over the allowance passes")
	}
	w := ok("W-HORIZON", 0)
	w.Rejected, w.Rules = true, []string{"allowance"}
	if st([]groupResult{w}, "T4") != Fail {
		t.Error("T4: a witness rejected by another rule passes")
	}
	w.Rules = []string{"safety"}
	if st([]groupResult{w}, "T4") != Pass {
		t.Error("T4: the witness's own rule fails")
	}
	nm := ok("NM2.1", 0)
	nm.UrgentSeeds = []SeedDifference{{Seed: 0, Orders: 100, Candidate: 2, Baseline: 0}}
	nm.Pairs = []pairInfo{{Seed: 0, Served: 100, Injected: 2, Eligible: 100}}
	if r := reservedCriteria(m, []groupResult{nm}, t5Tolerance{})["T5b"]; r.Status != "not locked" {
		t.Errorf("T5b without tolerances: %s", r.Status)
	}
	lo, hi := 0.45, 0.01
	if r := reservedCriteria(m, []groupResult{nm}, t5Tolerance{&lo, &hi})["T5b"]; r.Status != Fail {
		t.Errorf("T5b: an accepted 2.1 evaluation (rate 1) against 0.45: %s", r.Status)
	}
	e := base(0, 0, 0)
	e[0].Errors = []string{"seed 0: infeasible"}
	if st(e, "T8") != "inconclusive" || st(e, "T1") != "inconclusive" {
		t.Errorf("a construction error: T8 %s, T1 %s", st(e, "T8"), st(e, "T1"))
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
