package eval

import (
	"fmt"
	"maps"
	"math/big"
	"slices"
)

// The lost-demand gate of v0.2-eval-5's modified option B, at the seed
// level (Slice 5C.4d, docs/calibration/clinical-5c4d/gate-b-statistics.md;
// it replaces Slice 5C.4c's pooled discordant-pair bound, which the audit
// showed invalid).
//
// The eligible orders are a seed's orders received after the takeover and
// wanted in the gated period. An order is unimaged when it ends lost,
// left or by the patient (cancellation or no-show). A seed is matched
// when the candidate's eligible orders are the baseline's, origin for
// origin (the immutable demand identity), each with the same request.
// Each matched seed gives one difference, the candidate's unimaged
// fraction less the baseline's; orders within a seed are never treated
// as independent. The estimand is the mean of the seed differences; Gate
// B passes when its one-sided upper confidence bound (the policy's
// method) is at most the margin. Missing identity, unmatched demand, too
// few orders or seeds, or a design whose bound could not reach the margin
// even with no difference at all, is insufficient: never a pass.
//
// Under the empirical rule (Slice 5C.4e, docs/calibration/clinical-5c4e/
// empirical-acceptance.md) the claim is narrower: Gate B passes when the
// observed seed-averaged increase on the evaluated seeds is at most the
// margin, exactly. It is a benchmark acceptance on those seeds, not a
// confidence statement about the population of seeds; the betting bound
// is still computed and reported as a research diagnostic, never used to
// decide. The per-seed safeguard flags each matched seed whose own
// increase exceeds maxSeedIncrease; the comparison counts those seeds as
// failing in the shared clinical allowance (seedSafeguardJoint).

// LostDemandGate is one priority's lost-demand gate.
type LostDemandGate struct {
	Priority string `json:"priority"`
	Rule     string `json:"rule,omitempty"` // empirical; empty: the confidence rule
	Status   string `json:"status"`
	Reason   string `json:"reason,omitempty"`
	Method   string `json:"method"`
	// Seeds matched; seeds whose eligible demand is not the baseline's or
	// whose identity evidence is missing or duplicated.
	MatchedSeeds   int     `json:"matchedSeeds"`
	UnmatchedSeeds []int64 `json:"unmatchedSeeds"`
	// Orders: matched eligible orders, all seeds (reported only).
	Orders    int64    `json:"orders"`
	Candidate Unimaged `json:"candidate"`
	Baseline  Unimaged `json:"baseline"`
	// Absolute unimaged fractions of the matched demand, candidate then
	// baseline: seed-averaged over the judged seeds (matched, not thin),
	// and pooled over every matched seed's orders. Null when there is
	// nothing to average (audit G6.2: unavailable, never a measured zero);
	// they never describe unmatched demand.
	SeedFraction   *[2]float64 `json:"unimagedFractionSeedMean"`
	PooledFraction *[2]float64 `json:"unimagedFractionPooled"`
	// The seed-averaged increase, its standard deviation and, under the
	// confidence rule only, the bound and the bound the design gives when
	// every seed's difference is zero.
	Increase          float64          `json:"increase"`
	IncreaseSD        float64          `json:"increaseSD"`
	IncreaseUpper     *float64         `json:"increaseUpper,omitempty"`
	ZeroIncreaseUpper *float64         `json:"zeroIncreaseUpper,omitempty"`
	Threshold         float64          `json:"threshold"`
	WorstSeed         WorstSeed        `json:"worstSeedIncrease"`
	Seeds             []SeedDifference `json:"seeds"`
	// The empirical rule's report: the seed differences' distribution,
	// the research diagnostic, the per-seed safeguard's threshold and the
	// seeds it flags, and the matched seeds too thin to judge.
	Distribution  *SeedDistribution `json:"seedDifferenceDistribution,omitempty"`
	Diagnostic    *BoundDiagnostic  `json:"researchDiagnostic,omitempty"`
	SeedThreshold float64           `json:"maxSeedIncrease,omitempty"`
	Deteriorated  []int64           `json:"deterioratedSeeds,omitempty"`
	ThinSeeds     []int64           `json:"thinSeeds,omitempty"`
}

// SeedDistribution summarizes the seed differences (order statistics of
// the observed seeds; no inference).
type SeedDistribution struct {
	Seeds  int     `json:"seeds"`
	Min    float64 `json:"min"`
	P10    float64 `json:"p10"`
	Median float64 `json:"median"`
	P90    float64 `json:"p90"`
	Max    float64 `json:"max"`
	Worse  int     `json:"seedsWorse"` // difference above zero
	Equal  int     `json:"seedsEqual"`
	Better int     `json:"seedsBetter"`
}

// DiagnosticPurpose labels the empirical rule's bound.
const DiagnosticPurpose = "research diagnostic only: not used for acceptance"

// BoundDiagnostic is the finite-sample bound computed beside the
// empirical rule: what a population-confidence claim at its confidence
// could and could not establish from these seeds.
type BoundDiagnostic struct {
	Purpose    string  `json:"purpose"`
	Method     string  `json:"method"`
	Confidence float64 `json:"confidence"`
	Upper      float64 `json:"upper"`
	ZeroUpper  float64 `json:"zeroDifferenceUpper"`
	// Established: the bound is at most the margin (a claim the
	// empirical rule does not make).
	Established bool `json:"marginEstablishedAtConfidence"`
}

// SeedDifference is one matched seed's unimaged fractions.
type SeedDifference struct {
	Seed       int64   `json:"seed"`
	Orders     int64   `json:"orders"`
	Candidate  int64   `json:"candidateUnimaged"`
	Baseline   int64   `json:"baselineUnimaged"`
	Difference float64 `json:"difference"`
	// Deteriorated: above the per-seed safeguard (empirical rule).
	Deteriorated bool `json:"deteriorated,omitempty"`
}

// Unimaged counts a plan's unimaged eligible orders by outcome.
type Unimaged struct {
	Lost    int64 `json:"lost"`
	Left    int64 `json:"left"`
	Patient int64 `json:"patient"`
	Total   int64 `json:"total"`
}

func (u *Unimaged) add(out string) bool {
	switch out {
	case OutLost:
		u.Lost++
	case OutLeft:
		u.Left++
	case OutPatient:
		u.Patient++
	default:
		return false
	}
	u.Total++
	return true
}

func (u *Unimaged) plus(v Unimaged) {
	u.Lost, u.Left, u.Patient, u.Total = u.Lost+v.Lost, u.Left+v.Left, u.Patient+v.Patient, u.Total+v.Total
}

// demandRequest is what an eligible order's patient asked for.
type demandRequest struct {
	received, wanted int64
	modality         string
	walkIn           bool
}

type eligibleOrder struct {
	req demandRequest
	out string
}

// endCapture is a record's validated end capture: the one named "end",
// at the end of observation, with an outcome per order (audit F2: never
// the last element of the array).
func endCapture(r *Record) *Capture {
	var end *Capture
	for i := range r.Captures {
		c := &r.Captures[i]
		if c.Name != "end" {
			continue
		}
		if end != nil || c.AtSec != r.ObservedUntilSec || len(c.OrderOutcomes) != len(r.Orders) {
			return nil
		}
		end = c
	}
	return end
}

// eligibleDemand is a record's eligible orders of priority q by origin;
// ok is false when the end capture is missing or an order lacks an origin
// or repeats one.
func eligibleDemand(p *Policy, r *Record, q string) (map[string]eligibleOrder, bool) {
	end := endCapture(r)
	if end == nil {
		return nil, false
	}
	b := windows(p, r)[WinGated]
	out := map[string]eligibleOrder{}
	seen := map[string]bool{}
	for i, o := range r.Orders {
		if o.Origin == "" || seen[o.Origin] {
			return nil, false
		}
		seen[o.Origin] = true
		if o.Priority != q || o.ReceivedSec <= r.TakeoverSec || o.WantedSec <= b[0] || o.WantedSec > b[1] {
			continue
		}
		out[o.Origin] = eligibleOrder{demandRequest{o.ReceivedSec, o.WantedSec, o.Modality, o.WalkIn}, end.OrderOutcomes[i]}
	}
	return out, true
}

// lostDemandGates are a candidate's lost-demand gates against the paired
// plan, seed by seed in the candidate's seed order.
func lostDemandGates(p *Policy, cr, ar []Record) []LostDemandGate {
	ld := p.Clinical.LostDemand
	byseed := map[int64]*Record{}
	for i := range ar {
		byseed[ar[i].Seed] = &ar[i]
	}
	var out []LostDemandGate
	for _, q := range Priorities {
		empirical := ld.Rule == RuleEmpirical
		g := LostDemandGate{Priority: q, Rule: ld.Rule, Method: ld.Method, UnmatchedSeeds: []int64{}, Threshold: ld.MaxIncrease[q], Seeds: []SeedDifference{}}
		var diffs, fa, fb []float64
		var thin []int64
		for i := range cr {
			a, b := &cr[i], byseed[cr[i].Seed]
			if b == nil {
				g.UnmatchedSeeds = append(g.UnmatchedSeeds, a.Seed)
				continue
			}
			ea, okA := eligibleDemand(p, a, q)
			eb, okB := eligibleDemand(p, b, q)
			if !okA || !okB || len(ea) != len(eb) {
				g.UnmatchedSeeds = append(g.UnmatchedSeeds, a.Seed)
				continue
			}
			var sa, sb Unimaged
			matched := true
			for _, origin := range slices.Sorted(maps.Keys(ea)) {
				x, y := ea[origin], eb[origin]
				if _, ok := eb[origin]; !ok || x.req != y.req {
					matched = false
					break
				}
				sa.add(x.out)
				sb.add(y.out)
			}
			if !matched {
				g.UnmatchedSeeds = append(g.UnmatchedSeeds, a.Seed)
				continue
			}
			n := int64(len(ea))
			g.MatchedSeeds++
			g.Orders += n
			g.Candidate.plus(sa)
			g.Baseline.plus(sb)
			if n < ld.MinSeedOrders[q] || n == 0 {
				thin = append(thin, a.Seed)
				continue
			}
			d := float64(sa.Total-sb.Total) / float64(n)
			sd := SeedDifference{Seed: a.Seed, Orders: n, Candidate: sa.Total, Baseline: sb.Total, Difference: d}
			// The safeguard, exactly: (a − b) / n above the threshold.
			if empirical && ld.MaxSeedIncrease != nil && mul(sa.Total-sb.Total, 10_000).Cmp(mul(bp(ld.MaxSeedIncrease[q]), n)) > 0 {
				sd.Deteriorated = true
				g.Deteriorated = append(g.Deteriorated, a.Seed)
			}
			g.Seeds = append(g.Seeds, sd)
			g.WorstSeed.note(d, a.Seed, true)
			diffs = append(diffs, d)
			fa, fb = append(fa, float64(sa.Total)/float64(n)), append(fb, float64(sb.Total)/float64(n))
		}
		if g.Orders > 0 {
			g.PooledFraction = &[2]float64{float64(g.Candidate.Total) / float64(g.Orders), float64(g.Baseline.Total) / float64(g.Orders)}
		}
		if len(diffs) > 0 {
			ma, _ := meanSD(fa)
			mb, _ := meanSD(fb)
			g.SeedFraction = &[2]float64{ma, mb}
			g.Increase, g.IncreaseSD = meanSD(diffs)
		}
		if empirical {
			empiricalGate(&g, ld, q, diffs, thin, len(cr))
			out = append(out, g)
			continue
		}
		up, zero := upperBound(ld.Method, diffs, ld.Confidence, -1, 1), zeroFloor(ld.Method, len(cr), ld.Confidence, -1, 1)
		if len(diffs) == 0 {
			up = 0
		}
		g.IncreaseUpper, g.ZeroIncreaseUpper = &up, &zero
		switch {
		case len(g.UnmatchedSeeds) > 0:
			g.Status = Insufficient
			g.Reason = fmt.Sprintf("eligible demand not the baseline's, or its identity evidence missing, on seeds %v: the comparison is not paired", g.UnmatchedSeeds)
		case len(thin) > 0:
			g.Status = Insufficient
			g.Reason = fmt.Sprintf("a seed has fewer than %d matched orders", ld.MinSeedOrders[q])
		case len(diffs) < ld.MinSeeds:
			g.Status = Insufficient
			g.Reason = fmt.Sprintf("%d seeds, fewer than %d", len(diffs), ld.MinSeeds)
		case exceeds(zero, ld.MaxIncrease[q]):
			g.Status = Insufficient
			g.Reason = fmt.Sprintf("with %d seeds the %s bound cannot establish an increase of at most %.4f (%.4f with no difference on any seed)",
				len(diffs), ld.Method, ld.MaxIncrease[q], zero)
		case exceeds(up, ld.MaxIncrease[q]):
			g.Status = Fail
			g.Reason = fmt.Sprintf("unimaged %.4f against the baseline's %.4f (seed means): increase %.4f, upper bound %.4f above %.4f",
				g.SeedFraction[0], g.SeedFraction[1], g.Increase, up, ld.MaxIncrease[q])
		default:
			g.Status = Pass
		}
		out = append(out, g)
	}
	return out
}

// empiricalGate decides g under the empirical rule: unmatched or thin
// seeds, or fewer than minSeeds, insufficient; else FAIL when the
// observed seed-averaged increase exceeds the margin (exactly, as a
// rational), PASS otherwise. The betting bound is reported beside it as
// a research diagnostic. The reasons state observed values only.
func empiricalGate(g *LostDemandGate, ld *LostDemand, q string, diffs []float64, thin []int64, seeds int) {
	if ld.MaxSeedIncrease != nil {
		g.SeedThreshold = ld.MaxSeedIncrease[q]
	}
	g.ThinSeeds = thin
	if len(diffs) > 0 {
		g.Distribution = seedDistribution(diffs)
		d := &BoundDiagnostic{Purpose: DiagnosticPurpose, Method: ld.Method, Confidence: ld.Confidence,
			Upper: upperBound(ld.Method, diffs, ld.Confidence, -1, 1), ZeroUpper: zeroFloor(ld.Method, seeds, ld.Confidence, -1, 1)}
		d.Established = !exceeds(d.Upper, ld.MaxIncrease[q])
		g.Diagnostic = d
	}
	switch {
	case len(g.UnmatchedSeeds) > 0:
		g.Status = Insufficient
		g.Reason = fmt.Sprintf("eligible demand not the baseline's, or its identity evidence missing, on seeds %v: the comparison is not paired", g.UnmatchedSeeds)
	case len(thin) > 0:
		g.Status = Insufficient
		g.Reason = fmt.Sprintf("seeds %v have fewer than %d matched orders", thin, ld.MinSeedOrders[q])
	case len(diffs) < ld.MinSeeds:
		g.Status = Insufficient
		g.Reason = fmt.Sprintf("%d paired seeds, fewer than %d", len(diffs), ld.MinSeeds)
	case !meanAtMost(g.Seeds, bp(ld.MaxIncrease[q])):
		g.Status = Fail
		g.Reason = fmt.Sprintf("observed on %d paired seeds: unimaged %.4f against the baseline's %.4f (seed means), mean increase %.4f above the margin %.4f",
			len(diffs), g.SeedFraction[0], g.SeedFraction[1], g.Increase, ld.MaxIncrease[q])
	default:
		g.Status = Pass
	}
}

// meanAtMost reports whether the seeds' mean difference is at most limit
// basis points, exactly: Σ (cᵢ − bᵢ) / nᵢ ≤ limit × seeds / 10 000.
func meanAtMost(ds []SeedDifference, limit int64) bool {
	sum := new(big.Rat)
	for _, d := range ds {
		sum.Add(sum, big.NewRat(d.Candidate-d.Baseline, d.Orders))
	}
	return sum.Cmp(big.NewRat(limit*int64(len(ds)), 10_000)) <= 0
}

// seedDistribution is the order statistics of xs (nearest rank).
func seedDistribution(xs []float64) *SeedDistribution {
	s := slices.Clone(xs)
	slices.Sort(s)
	at := func(p int) float64 { return s[(p*(len(s)-1)+50)/100] }
	d := &SeedDistribution{Seeds: len(s), Min: s[0], P10: at(10), Median: at(50), P90: at(90), Max: s[len(s)-1]}
	for _, x := range s {
		switch {
		case x > 0:
			d.Worse++
		case x < 0:
			d.Better++
		default:
			d.Equal++
		}
	}
	return d
}

// exceeds compares a computed bound with a policy margin, the margin's
// last basis point inclusive (the bound is a float of a bisection).
func exceeds(v, limit float64) bool { return v > limit+1e-12 }

// AccessLimitation is stated on every favourable v5 verdict with
// lost-demand gates (owner decision, Slice 5C.4d).
const AccessLimitation = "Access non-deterioration only; absolute access adequacy not established."

// The empirical rule's claims, kept apart on every comparison under it
// (Slice 5C.4e): what the acceptance is, and the two claims it is not.
const (
	EmpiricalScope         = "Empirical benchmark acceptance on the evaluated paired seeds only; no population statistical confidence is claimed."
	ConfidenceNotClaimed   = "not claimed: the empirical rule has no population coverage; the finite-sample bound is a research diagnostic"
	AdequacyNotEstablished = "not established"
)

// AccessClaims separates, under the empirical rule, the empirical
// benchmark acceptance (Gate B on the evaluated seeds, every priority),
// population statistical confidence (never claimed) and absolute clinical
// adequacy (never established).
type AccessClaims struct {
	EmpiricalAcceptance  string `json:"empiricalBenchmarkAcceptance"`
	Seeds                int    `json:"pairedSeeds"`
	PopulationConfidence string `json:"populationStatisticalConfidence"`
	AbsoluteAdequacy     string `json:"absoluteClinicalAdequacy"`
}

// accessClaims are the comparison's claims under the empirical rule: the
// acceptance PASS only when every priority's Gate B passes, else the
// worst status.
func accessClaims(gates []LostDemandGate) *AccessClaims {
	a := &AccessClaims{EmpiricalAcceptance: Pass, PopulationConfidence: ConfidenceNotClaimed, AbsoluteAdequacy: AdequacyNotEstablished}
	for i, g := range gates {
		switch {
		case g.Status == Fail:
			a.EmpiricalAcceptance = Fail
		case g.Status != Pass && a.EmpiricalAcceptance == Pass:
			a.EmpiricalAcceptance = Insufficient
		}
		if i == 0 || g.MatchedSeeds < a.Seeds {
			a.Seeds = g.MatchedSeeds
		}
	}
	return a
}

// AccessStatement is one priority's access, as a verdict shows it:
// timeliness of the served, open and late-cancelled orders (Gate A,
// pooled over the gated period), the absolute unimaged fractions of the
// candidate and the baseline, and Gate B's increase with its bound.
type AccessStatement struct {
	Priority            string      `json:"priority"`
	Timeliness          Ratio       `json:"timeliness"`
	TimelinessStatus    string      `json:"timelinessStatus"`
	Unimaged            *[2]float64 `json:"unimagedFractionSeedMean"` // candidate, baseline; null: unavailable
	PooledUnimaged      *[2]float64 `json:"unimagedFractionPooled"`
	Increase            float64     `json:"increase"`
	IncreaseUpper       *float64    `json:"increaseUpper,omitempty"` // the confidence rule's bound
	Rule                string      `json:"rule,omitempty"`
	DiagnosticUpper     *float64    `json:"researchDiagnosticUpper,omitempty"` // the empirical rule's, never decisive
	Method              string      `json:"method"`
	LostDemandStatus    string      `json:"lostDemandStatus"`
	AdequacyEstablished bool        `json:"absoluteAdequacyEstablished"` // always false: no gate establishes it
}

// accessStatements are a candidate's access statements.
func accessStatements(cand *PlanReport, gates []LostDemandGate) []AccessStatement {
	var out []AccessStatement
	for _, g := range gates {
		a := AccessStatement{Priority: g.Priority, Unimaged: g.SeedFraction, PooledUnimaged: g.PooledFraction, Increase: g.Increase,
			IncreaseUpper: g.IncreaseUpper, Rule: g.Rule, Method: g.Method, LostDemandStatus: g.Status, TimelinessStatus: "missing"}
		if g.Diagnostic != nil {
			a.DiagnosticUpper = &g.Diagnostic.Upper
		}
		for _, s := range cand.ClinicalGates {
			if s.Gate == GateTimelyAccess && s.Window == WinGated && s.Stratum == g.Priority {
				a.Timeliness, a.TimelinessStatus = s.Pooled, s.Status
			}
		}
		out = append(out, a)
	}
	return out
}

// seedSafeguardJoint is the candidate's shared clinical allowance with
// the empirical rule's per-seed safeguard counted in: a seed flagged
// deteriorated in any priority fails; a seed without paired per-seed
// evidence (unmatched or thin) lacks it; each seed counted once with its
// own clinical status. Nil when the policy has no safeguard.
func seedSafeguardJoint(p *Policy, cand *PlanReport, gates []LostDemandGate) *JointGate {
	ld := p.Clinical.LostDemand
	if ld.Rule != RuleEmpirical || ld.MaxSeedIncrease == nil || cand.ClinicalJoint == nil {
		return nil
	}
	safe := map[int64]string{}
	for _, g := range gates {
		for _, s := range append(slices.Clone(g.UnmatchedSeeds), g.ThinSeeds...) {
			if safe[s] != Fail {
				safe[s] = Insufficient
			}
		}
		for _, s := range g.Deteriorated {
			safe[s] = Fail
		}
	}
	j := jointOf(p, cand.PerSeed, "clinical", bp(p.Clinical.MaxFailingSeedFraction), func(s *SeedResult) string {
		switch {
		case s.Clinical == Fail || safe[s.Seed] == Fail:
			return Fail
		case s.Clinical == Insufficient || safe[s.Seed] == Insufficient:
			return Insufficient
		}
		return Pass
	})
	j.Gates = cand.ClinicalJoint.Gates + 1 // the safeguard
	return &j
}
