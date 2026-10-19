package eval

import (
	"fmt"
	"maps"
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

// LostDemandGate is one priority's lost-demand gate.
type LostDemandGate struct {
	Priority string `json:"priority"`
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
	// Absolute unimaged fractions, reported always: seed-averaged and
	// pooled, candidate then baseline.
	SeedFraction   [2]float64 `json:"unimagedFractionSeedMean"`
	PooledFraction [2]float64 `json:"unimagedFractionPooled"`
	// The seed-averaged increase, its standard deviation, the bound, and
	// the bound the design gives when every seed's difference is zero.
	Increase          float64          `json:"increase"`
	IncreaseSD        float64          `json:"increaseSD"`
	IncreaseUpper     float64          `json:"increaseUpper"`
	ZeroIncreaseUpper float64          `json:"zeroIncreaseUpper"`
	Threshold         float64          `json:"threshold"`
	WorstSeed         WorstSeed        `json:"worstSeedIncrease"`
	Seeds             []SeedDifference `json:"seeds"`
}

// SeedDifference is one matched seed's unimaged fractions.
type SeedDifference struct {
	Seed       int64   `json:"seed"`
	Orders     int64   `json:"orders"`
	Candidate  int64   `json:"candidateUnimaged"`
	Baseline   int64   `json:"baselineUnimaged"`
	Difference float64 `json:"difference"`
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
		g := LostDemandGate{Priority: q, Method: ld.Method, UnmatchedSeeds: []int64{}, Threshold: ld.MaxIncrease[q], Seeds: []SeedDifference{}}
		var diffs, fa, fb []float64
		thin := false
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
				thin = true
				continue
			}
			d := float64(sa.Total-sb.Total) / float64(n)
			g.Seeds = append(g.Seeds, SeedDifference{Seed: a.Seed, Orders: n, Candidate: sa.Total, Baseline: sb.Total, Difference: d})
			g.WorstSeed.note(d, a.Seed, true)
			diffs = append(diffs, d)
			fa, fb = append(fa, float64(sa.Total)/float64(n)), append(fb, float64(sb.Total)/float64(n))
		}
		if g.Orders > 0 {
			g.PooledFraction = [2]float64{float64(g.Candidate.Total) / float64(g.Orders), float64(g.Baseline.Total) / float64(g.Orders)}
		}
		if len(diffs) > 0 {
			ma, _ := meanSD(fa)
			mb, _ := meanSD(fb)
			g.SeedFraction = [2]float64{ma, mb}
			g.Increase, g.IncreaseSD = meanSD(diffs)
			g.IncreaseUpper = upperBound(ld.Method, diffs, ld.Confidence, -1, 1)
		}
		g.ZeroIncreaseUpper = zeroFloor(ld.Method, len(cr), ld.Confidence, -1, 1)
		switch {
		case len(g.UnmatchedSeeds) > 0:
			g.Status = Insufficient
			g.Reason = fmt.Sprintf("eligible demand not the baseline's, or its identity evidence missing, on seeds %v: the comparison is not paired", g.UnmatchedSeeds)
		case thin:
			g.Status = Insufficient
			g.Reason = fmt.Sprintf("a seed has fewer than %d matched orders", ld.MinSeedOrders[q])
		case len(diffs) < ld.MinSeeds:
			g.Status = Insufficient
			g.Reason = fmt.Sprintf("%d seeds, fewer than %d", len(diffs), ld.MinSeeds)
		case exceeds(g.ZeroIncreaseUpper, ld.MaxIncrease[q]):
			g.Status = Insufficient
			g.Reason = fmt.Sprintf("with %d seeds the %s bound cannot establish an increase of at most %.4f (%.4f with no difference on any seed)",
				len(diffs), ld.Method, ld.MaxIncrease[q], g.ZeroIncreaseUpper)
		case exceeds(g.IncreaseUpper, ld.MaxIncrease[q]):
			g.Status = Fail
			g.Reason = fmt.Sprintf("unimaged %.4f against the baseline's %.4f (seed means): increase %.4f, upper bound %.4f above %.4f",
				g.SeedFraction[0], g.SeedFraction[1], g.Increase, g.IncreaseUpper, ld.MaxIncrease[q])
		default:
			g.Status = Pass
		}
		out = append(out, g)
	}
	return out
}

// exceeds compares a computed bound with a policy margin, the margin's
// last basis point inclusive (the bound is a float of a bisection).
func exceeds(v, limit float64) bool { return v > limit+1e-12 }

// AccessLimitation is stated on every favourable v5 verdict with
// lost-demand gates (owner decision, Slice 5C.4d).
const AccessLimitation = "Access non-deterioration only; absolute access adequacy not established."

// AccessStatement is one priority's access, as a verdict shows it:
// timeliness of the served, open and late-cancelled orders (Gate A,
// pooled over the gated period), the absolute unimaged fractions of the
// candidate and the baseline, and Gate B's increase with its bound.
type AccessStatement struct {
	Priority            string     `json:"priority"`
	Timeliness          Ratio      `json:"timeliness"`
	TimelinessStatus    string     `json:"timelinessStatus"`
	Unimaged            [2]float64 `json:"unimagedFractionSeedMean"` // candidate, baseline
	PooledUnimaged      [2]float64 `json:"unimagedFractionPooled"`
	Increase            float64    `json:"increase"`
	IncreaseUpper       float64    `json:"increaseUpper"`
	Method              string     `json:"method"`
	LostDemandStatus    string     `json:"lostDemandStatus"`
	AdequacyEstablished bool       `json:"absoluteAdequacyEstablished"` // always false: no gate establishes it
}

// accessStatements are a candidate's access statements.
func accessStatements(cand *PlanReport, gates []LostDemandGate) []AccessStatement {
	var out []AccessStatement
	for _, g := range gates {
		a := AccessStatement{Priority: g.Priority, Unimaged: g.SeedFraction, PooledUnimaged: g.PooledFraction, Increase: g.Increase,
			IncreaseUpper: g.IncreaseUpper, Method: g.Method, LostDemandStatus: g.Status, TimelinessStatus: "missing"}
		for _, s := range cand.ClinicalGates {
			if s.Gate == GateTimelyAccess && s.Window == WinGated && s.Stratum == g.Priority {
				a.Timeliness, a.TimelinessStatus = s.Pooled, s.Status
			}
		}
		out = append(out, a)
	}
	return out
}
