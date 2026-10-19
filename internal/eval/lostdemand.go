package eval

import (
	"cmp"
	"fmt"
	"slices"
)

// The lost-demand gate of v0.2-eval-5's modified option B (Slice 5C.4c,
// docs/calibration/clinical-5c4c/access-gates.md): the demand a candidate
// leaves unimaged, against the paired baseline's, per priority.
//
// The eligible orders are a seed's orders received after the takeover and
// wanted in the gated period (the access cohorts' demand). An order is
// unimaged when it ends lost, left or by the patient (cancellation or
// no-show): a practice cannot hide loss as a patient outcome. Seeds are
// paired only where the candidate's eligible demand is the baseline's,
// order for order: the same receipt and wanted times, modality and
// walk-in flag, so that the same patients are compared. Within a group of
// identical orders (indistinguishable patients) only the counts are
// compared. The test is exact and one-sided: conditional on the
// discordant orders (unimaged under one plan only), the upper confidence
// bound of the pooled increase in the unimaged fraction must not exceed
// the policy's maxIncrease. Unmatched demand on any seed, or too few
// matched orders, is insufficient: never a pass.

// LostDemandGate is one priority's lost-demand gate.
type LostDemandGate struct {
	Priority string `json:"priority"`
	Status   string `json:"status"`
	Reason   string `json:"reason,omitempty"`
	// Seeds paired, and those whose eligible demand is not the baseline's.
	MatchedSeeds   int     `json:"matchedSeeds"`
	UnmatchedSeeds []int64 `json:"unmatchedSeeds"`
	// Orders: matched eligible orders, pooled.
	Orders    int64      `json:"orders"`
	Candidate Unimaged   `json:"candidate"`
	Baseline  Unimaged   `json:"baseline"`
	Fraction  [2]float64 `json:"unimagedFraction"` // candidate, baseline (absolute, reported always)
	// Discordant: unimaged under the candidate only, under the baseline
	// only.
	Discordant    [2]int64  `json:"discordant"`
	Increase      float64   `json:"increase"`      // pooled, candidate − baseline
	IncreaseUpper float64   `json:"increaseUpper"` // one-sided, at the policy's confidence
	Threshold     float64   `json:"threshold"`
	WorstSeed     WorstSeed `json:"worstSeedIncrease"`
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

// demandKey identifies an eligible order across plans: what the
// patient's request is, not what became of it.
type demandKey struct {
	received, wanted int64
	modality         string
	walkIn           bool
}

type eligibleOrder struct {
	key demandKey
	out string
}

// eligibleDemand is a record's eligible orders of priority q, sorted by
// key.
func eligibleDemand(p *Policy, r *Record, q string) ([]eligibleOrder, bool) {
	if len(r.Captures) == 0 {
		return nil, false
	}
	end := r.Captures[len(r.Captures)-1].OrderOutcomes
	b := windows(p, r)[WinGated]
	var out []eligibleOrder
	for i, o := range r.Orders {
		if o.Priority != q || o.ReceivedSec <= r.TakeoverSec || o.WantedSec <= b[0] || o.WantedSec > b[1] || i >= len(end) {
			continue
		}
		out = append(out, eligibleOrder{demandKey{o.ReceivedSec, o.WantedSec, o.Modality, o.WalkIn}, end[i]})
	}
	slices.SortFunc(out, func(x, y eligibleOrder) int {
		return cmp.Or(cmp.Compare(x.key.received, y.key.received), cmp.Compare(x.key.wanted, y.key.wanted),
			cmp.Compare(x.key.modality, y.key.modality), cmpBool(x.key.walkIn, y.key.walkIn))
	})
	return out, true
}

func cmpBool(a, b bool) int {
	switch {
	case a == b:
		return 0
	case !a:
		return -1
	}
	return 1
}

// lostDemandGates are a candidate's lost-demand gates against the paired
// plan, on the seeds both have.
func lostDemandGates(p *Policy, cr, ar []Record) []LostDemandGate {
	ld := p.Clinical.LostDemand
	byseed := map[int64]*Record{}
	for i := range ar {
		byseed[ar[i].Seed] = &ar[i]
	}
	var out []LostDemandGate
	for _, q := range Priorities {
		g := LostDemandGate{Priority: q, UnmatchedSeeds: []int64{}, Threshold: ld.MaxIncrease[q]}
		for i := range cr {
			a, b := &cr[i], byseed[cr[i].Seed]
			if b == nil {
				g.UnmatchedSeeds = append(g.UnmatchedSeeds, a.Seed)
				continue
			}
			ea, okA := eligibleDemand(p, a, q)
			eb, okB := eligibleDemand(p, b, q)
			if !okA || !okB || len(ea) != len(eb) || !slices.EqualFunc(ea, eb, func(x, y eligibleOrder) bool { return x.key == y.key }) {
				g.UnmatchedSeeds = append(g.UnmatchedSeeds, a.Seed)
				continue
			}
			g.MatchedSeeds++
			var sa, sb Unimaged
			for j := 0; j < len(ea); {
				k := j
				var na, nb int64
				for ; k < len(ea) && ea[k].key == ea[j].key; k++ {
					if sa.add(ea[k].out) {
						na++
					}
					if sb.add(eb[k].out) {
						nb++
					}
				}
				g.Discordant[0] += max(0, na-nb)
				g.Discordant[1] += max(0, nb-na)
				j = k
			}
			n := int64(len(ea))
			g.Orders += n
			for _, x := range []struct {
				u   *Unimaged
				add Unimaged
			}{{&g.Candidate, sa}, {&g.Baseline, sb}} {
				x.u.Lost += x.add.Lost
				x.u.Left += x.add.Left
				x.u.Patient += x.add.Patient
				x.u.Total += x.add.Total
			}
			if n > 0 {
				g.WorstSeed.note(float64(sa.Total-sb.Total)/float64(n), a.Seed, true)
			}
		}
		if g.Orders > 0 {
			g.Fraction = [2]float64{float64(g.Candidate.Total) / float64(g.Orders), float64(g.Baseline.Total) / float64(g.Orders)}
			g.Increase = g.Fraction[0] - g.Fraction[1]
			if d := g.Discordant[0] + g.Discordant[1]; d > 0 {
				u := binomUpper(int(g.Discordant[0]), int(d), ld.Confidence)
				g.IncreaseUpper = (2*u - 1) * float64(d) / float64(g.Orders)
			}
		}
		switch {
		case len(g.UnmatchedSeeds) > 0:
			g.Status = Insufficient
			g.Reason = fmt.Sprintf("eligible demand not the baseline's on seeds %v: the comparison is not paired", g.UnmatchedSeeds)
		case g.Orders < ld.MinCohort[q]:
			g.Status = Insufficient
			g.Reason = fmt.Sprintf("%d matched orders, fewer than %d", g.Orders, ld.MinCohort[q])
		case exceeds(g.IncreaseUpper, ld.MaxIncrease[q]):
			g.Status = Fail
			g.Reason = fmt.Sprintf("unimaged %.4f against the baseline's %.4f: increase %.4f, upper bound %.4f above %.4f",
				g.Fraction[0], g.Fraction[1], g.Increase, g.IncreaseUpper, ld.MaxIncrease[q])
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
