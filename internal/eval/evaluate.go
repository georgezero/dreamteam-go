package eval

import (
	"fmt"
	"slices"
	"sort"
)

const ReportSchema = "dreamteam.evalreport/v1"

// Verdicts.
const (
	GO          = "GO"
	CONDITIONAL = "CONDITIONAL GO"
	NOGO        = "NO GO"
	INVALID     = "INVALID"
)

// Gate statuses.
const (
	Pass         = "pass"
	Fail         = "fail"
	Insufficient = "insufficient"
)

// SeedGate is one gate on one seed.
type SeedGate struct {
	Gate      string  `json:"gate"`
	Window    string  `json:"window"`
	Stratum   string  `json:"stratum,omitempty"`
	Status    string  `json:"status"`
	Value     float64 `json:"value"`
	Threshold float64 `json:"threshold"`
	Num       int64   `json:"num"`
	Den       int64   `json:"den"`
	Reason    string  `json:"reason,omitempty"`
	Undecided bool    `json:"undecided,omitempty"` // missing or unresolved: the pool is undefined
}

// SeedResult is one seed of a plan.
type SeedResult struct {
	Seed     int64         `json:"seed"`
	Service  string        `json:"service"` // pass, fail, insufficient over the gated windows
	Backlog  string        `json:"backlog"`
	Solvent  bool          `json:"solvent"`
	Gates    []SeedGate    `json:"gates"`
	Measures *SeedMeasures `json:"measures"`
}

// GateSummary is one gate across seeds: how many failed or lacked
// evidence, the pooled value, and the worst seed.
type GateSummary struct {
	Gate              string   `json:"gate"`
	Window            string   `json:"window"`
	Stratum           string   `json:"stratum,omitempty"`
	Role              string   `json:"role"` // gated, validation, info
	Status            string   `json:"status"`
	Threshold         float64  `json:"threshold"`
	Seeds             int      `json:"seeds"`
	Passing           int      `json:"passing"`
	Failing           int      `json:"failing"`
	InsufficientSeeds int      `json:"insufficientSeeds"`
	FailLimit         int      `json:"failLimit"`
	FailingFraction   Interval `json:"failingFraction"` // Wilson 95%
	Pooled            Ratio    `json:"pooled"`
	Worst             float64  `json:"worst"`
	WorstSeed         int64    `json:"worstSeed"`
	Mean              float64  `json:"mean"`
}

// PlanReport is one plan alone: its own gates, no comparison.
type PlanReport struct {
	Plan            string        `json:"plan"`
	Seeds           []int64       `json:"seeds"`
	Valid           bool          `json:"valid"`
	Integrity       []string      `json:"integrity"`
	ServiceGates    []GateSummary `json:"serviceGates"`
	ServiceStatus   string        `json:"serviceStatus"` // over gated windows
	ValidationPass  bool          `json:"validationServicePass"`
	BacklogStable   int           `json:"backlogStableSeeds"`
	BacklogStatus   string        `json:"backlogStatus"`
	LegacyBacklog   string        `json:"legacyBacklogRule"` // the historical rule on the seed means, for comparison only
	InsolventSeeds  []int64       `json:"insolventSeeds"`
	WorstMinCash    int64         `json:"worstMinCashCents"`
	WorstMinCashAt  int64         `json:"worstMinCashSeed"`
	SolvencyStatus  string        `json:"solvencyStatus"`
	FundingGapSeeds []int64       `json:"fundingGapSeedsAtTailEnd"`
	Eligible        bool          `json:"eligible"` // service, backlog and solvency, before comparison
	// A plan that differs from another only by teleradiology must book
	// exactly the same appointments on every seed (remote reading cannot
	// change technologist booking feasibility).
	BookingInvariantWith string       `json:"bookingInvariantWith,omitempty"`
	BookingInvariant     bool         `json:"bookingInvariant,omitempty"`
	PerSeed              []SeedResult `json:"perSeed"`
}

// Payback is the sustained break-even of a paired increment: the first
// day from which it stays at or above zero through the horizon. A seed
// still below zero at the horizon is right-censored.
type Payback struct {
	Horizon       int      `json:"horizonDays"`
	Days          []int    `json:"days"` // per seed; 0 = censored
	Repaid        int      `json:"repaid"`
	Censored      int      `json:"censored"`
	RepaidFrac    Interval `json:"repaidFraction"`
	Median        int      `json:"median"`        // 0 unless at least half repaid
	MedianDefined bool     `json:"medianDefined"` // false: report "not reached by the horizon"
	Earliest      int      `json:"earliest"`
	Latest        int      `json:"latest"`
}

// Comparison is a candidate against its paired baseline (or another plan).
type Comparison struct {
	Candidate string   `json:"candidate"`
	Against   string   `json:"against"`
	Valid     bool     `json:"valid"`
	Integrity []string `json:"integrity"`
	// Paired differences, candidate − against, per window.
	Profit              map[string]Interval              `json:"incrementalProfitCents"`
	Revenue             map[string]Interval              `json:"incrementalNetRevenueCents"`
	Reports             map[string]Interval              `json:"incrementalReports"`
	CashAtEnd           map[string]Interval              `json:"incrementalCashCents"`
	BreachDelta         map[string]map[string]Interval   `json:"breachRateDelta"` // window → priority
	TATP90Delta         map[string]map[string]Interval   `json:"completedTatP90DeltaMin"`
	WorstBreach         map[string]map[string][2]float64 `json:"worstSeedBreachRate"` // window → priority → [candidate, against]
	MinCashDelta        Interval                         `json:"minCashDeltaCents"`
	RemoteCost          map[string]Interval              `json:"remoteCostCents"`
	UnbilledDelta       map[string]Interval              `json:"unbilledCompletedDeltaCents"` // listed apart: not revenue
	AccrualPayback      []Payback                        `json:"accrualPayback"`
	CashPayback         []Payback                        `json:"cashPayback"`
	CaseSelection       []GateSummary                    `json:"caseSelection"`
	CaseSelectionStatus string                           `json:"caseSelectionStatus"`
	Verdict             string                           `json:"verdict,omitempty"`
	Reasons             []string                         `json:"reasons,omitempty"`
}

// Report is a whole evaluation.
type Report struct {
	Schema       string       `json:"schema"`
	PolicyID     string       `json:"policyId"`
	PolicyDigest string       `json:"policyDigest"`
	Baseline     string       `json:"baseline"`
	Plans        []PlanReport `json:"plans"`
	Comparisons  []Comparison `json:"comparisons"`
}

// Input is a set of plans' records on the same seeds. Baseline is the
// no-action plan every candidate is judged against; Pairs adds
// informational candidate-to-candidate comparisons (no verdict).
type Input struct {
	Baseline string
	Plans    map[string][]Record
	Pairs    [][2]string
}

// Evaluate applies the policy. The output depends only on its arguments.
func Evaluate(p *Policy, digest string, in Input) *Report {
	rep := &Report{Schema: ReportSchema, PolicyID: p.ID, PolicyDigest: digest, Baseline: in.Baseline}
	names := make([]string, 0, len(in.Plans))
	for n := range in.Plans {
		names = append(names, n)
	}
	sort.Strings(names)
	plans := map[string]*PlanReport{}
	for _, n := range names {
		pr := planReport(p, n, in.Plans[n])
		rep.Plans = append(rep.Plans, *pr)
		plans[n] = pr
	}
	for _, n := range names {
		bookingInvariance(plans[n], in.Plans[n], in.Plans)
	}
	rep.Plans = rep.Plans[:0]
	for _, n := range names {
		rep.Plans = append(rep.Plans, *plans[n])
	}
	base := plans[in.Baseline]
	for _, n := range names {
		if n == in.Baseline || base == nil {
			continue
		}
		c := compare(p, plans[n], base, in.Plans[n], in.Plans[in.Baseline])
		verdict(p, &c, plans[n], base)
		rep.Comparisons = append(rep.Comparisons, c)
	}
	for _, pair := range in.Pairs {
		a, b := plans[pair[0]], plans[pair[1]]
		if a == nil || b == nil {
			continue
		}
		rep.Comparisons = append(rep.Comparisons, compare(p, a, b, in.Plans[pair[0]], in.Plans[pair[1]]))
	}
	return rep
}

// bookingInvariance checks a teleradiology-only variant's bookings
// against its plan's, seed by seed; a difference invalidates the variant.
func bookingInvariance(pr *PlanReport, recs []Record, all map[string][]Record) {
	var of string
	for i := range recs {
		if recs[i].TeleOnlyOf != "" {
			of = recs[i].TeleOnlyOf
		}
	}
	if of == "" {
		return
	}
	pr.BookingInvariantWith, pr.BookingInvariant = of, true
	other := map[int64]string{}
	for _, r := range all[of] {
		other[r.Seed] = r.AppointmentsSHA256
	}
	for _, r := range recs {
		d, ok := other[r.Seed]
		if r.TeleOnlyOf != of || !ok || d != r.AppointmentsSHA256 || d == "" {
			pr.BookingInvariant = false
			pr.Integrity = append(pr.Integrity, fmt.Sprintf("seed %d: bookings differ from %s's", r.Seed, of))
		}
	}
	if !pr.BookingInvariant {
		pr.Valid, pr.Eligible = false, false
	}
}

func role(p *Policy, w string) string {
	switch {
	case slices.Contains(p.Service.Gated, w):
		return "gated"
	case slices.Contains(p.Service.Validation, w):
		return "validation"
	}
	return "info"
}

func planReport(p *Policy, name string, recs []Record) *PlanReport {
	pr := &PlanReport{Plan: name, Valid: true, Integrity: []string{}, InsolventSeeds: []int64{}, FundingGapSeeds: []int64{}}
	recs = slices.Clone(recs)
	sort.SliceStable(recs, func(i, j int) bool { return recs[i].Seed < recs[j].Seed })
	seen := map[int64]bool{}
	for i := range recs {
		r := &recs[i]
		if r.Plan != name {
			pr.Integrity = append(pr.Integrity, fmt.Sprintf("seed %d: record of plan %q", r.Seed, r.Plan))
		}
		if seen[r.Seed] {
			pr.Integrity = append(pr.Integrity, fmt.Sprintf("seed %d twice", r.Seed))
		}
		seen[r.Seed] = true
		m := Measure(p, r)
		for _, f := range m.Integrity {
			pr.Integrity = append(pr.Integrity, fmt.Sprintf("seed %d: %s", r.Seed, f))
		}
		sr := SeedResult{Seed: r.Seed, Measures: m, Backlog: m.Backlog.Status}
		sr.Solvent = m.MinCash >= p.Finance.MinCashFloorCents && m.PastDue <= p.Finance.MaxPastDueCents
		sr.Gates = serviceGates(p, m)
		sr.Service = Pass
		for _, g := range sr.Gates {
			if role(p, g.Window) != "gated" {
				continue
			}
			switch {
			case g.Status == Fail:
				sr.Service = Fail
			case g.Status == Insufficient && sr.Service == Pass:
				sr.Service = Insufficient
			}
		}
		if !sr.Solvent {
			pr.InsolventSeeds = append(pr.InsolventSeeds, r.Seed)
		}
		if w := m.Windows[WinTail]; w != nil && w.Money.FundingGapCents > 0 {
			pr.FundingGapSeeds = append(pr.FundingGapSeeds, r.Seed)
		}
		if len(pr.PerSeed) == 0 || m.MinCash < pr.WorstMinCash {
			pr.WorstMinCash, pr.WorstMinCashAt = m.MinCash, r.Seed
		}
		if m.Backlog.Status == "stable" {
			pr.BacklogStable++
		}
		pr.PerSeed = append(pr.PerSeed, sr)
		pr.Seeds = append(pr.Seeds, r.Seed)
	}
	sort.Slice(pr.PerSeed, func(i, j int) bool { return pr.PerSeed[i].Seed < pr.PerSeed[j].Seed })
	slices.Sort(pr.Seeds)
	slices.Sort(pr.Integrity)
	pr.Valid = len(pr.Integrity) == 0 && len(recs) > 0
	n := len(pr.PerSeed)
	pr.ServiceGates = summarize(p, pr.PerSeed, func(s SeedResult) []SeedGate { return s.Gates }, pooledBreach)
	pr.ServiceStatus, pr.ValidationPass = Pass, true
	for _, g := range pr.ServiceGates {
		switch {
		case g.Role == "gated" && g.Status == Fail:
			pr.ServiceStatus = Fail
		case g.Role == "gated" && g.Status == Insufficient && pr.ServiceStatus == Pass:
			pr.ServiceStatus = Insufficient
		case g.Role == "validation" && g.Status != Pass:
			pr.ValidationPass = false
		}
	}
	pr.BacklogStatus = Pass
	if n-pr.BacklogStable > failLimit(p.Backlog.MaxUnstableSeedFrac, n) {
		pr.BacklogStatus = Fail
	}
	var m90, m180 float64
	var legacyN int
	for _, s := range pr.PerSeed {
		if s.Measures.Backlog.LegacyDefined {
			m90 += float64(s.Measures.Backlog.LegacyDay90)
			m180 += float64(s.Measures.Backlog.LegacyDay180)
			legacyN++
		}
	}
	pr.LegacyBacklog = "undefined (no day-180 sample)"
	if legacyN > 0 {
		pr.LegacyBacklog = fmt.Sprintf("day 180 mean %.1f vs 1.1 × day 90 mean %.1f: holds %v", m180/float64(legacyN), 1.1*m90/float64(legacyN), m180 <= 1.1*m90)
	}
	pr.SolvencyStatus = Pass
	if len(pr.InsolventSeeds) > p.Finance.MaxInsolventSeeds {
		pr.SolvencyStatus = Fail
	}
	pr.Eligible = pr.Valid && pr.ServiceStatus == Pass && pr.BacklogStatus == Pass && pr.SolvencyStatus == Pass
	return pr
}

// serviceGates are a seed's breach gates in every window it has.
func serviceGates(p *Policy, m *SeedMeasures) []SeedGate {
	var out []SeedGate
	for _, w := range windowOrder(p) {
		wm := m.Windows[w]
		for _, q := range Priorities {
			g := SeedGate{Gate: "breach_rate", Window: w, Stratum: q, Threshold: p.Service.BreachGate[q]}
			if wm == nil {
				g.Status, g.Reason, g.Undecided = Insufficient, "window not measured", true
				out = append(out, g)
				continue
			}
			c := wm.Cohorts[q]
			g.Num, g.Den, g.Value = c.Breaches, c.Acquired, c.BreachRate.Value
			switch {
			case c.Unresolved > 0:
				g.Status, g.Reason, g.Undecided = Insufficient, fmt.Sprintf("%d studies undecided when observation ended", c.Unresolved), true
			case c.Acquired < p.Service.MinCohort[q]:
				g.Status, g.Reason = Insufficient, fmt.Sprintf("%d acquired, fewer than %d", c.Acquired, p.Service.MinCohort[q])
			case !c.BreachRate.Defined:
				g.Status, g.Reason = Insufficient, "empty"
			case c.BreachRate.Value <= g.Threshold:
				g.Status = Pass
			default:
				g.Status = Fail
			}
			out = append(out, g)
		}
	}
	return out
}

func windowOrder(p *Policy) []string {
	var out []string
	for _, w := range []string{WinT30, WinT90, WinOps, WinTail, WinT180} {
		if role(p, w) != "info" || slices.Contains(p.Service.Info, w) {
			out = append(out, w)
		}
	}
	return out
}

// pooledBreach pools a breach gate's counts over seeds: undefined if any
// seed's window is missing or its cohort undecided, or all are empty.
func pooledBreach(gs []SeedGate) Ratio {
	var num, den int64
	for _, g := range gs {
		if g.Undecided {
			return Ratio{}
		}
		num += g.Num
		den += g.Den
	}
	return ratio(num, den)
}

// summarize groups seed gates by (gate, window, stratum) in first-seen
// order and judges each across seeds: at most failLimit seeds failing or
// lacking evidence, and the pooled value within the threshold.
func summarize(p *Policy, seeds []SeedResult, gates func(SeedResult) []SeedGate, pool func([]SeedGate) Ratio) []GateSummary {
	type key struct{ g, w, s string }
	var order []key
	by := map[key][]SeedGate{}
	seedOf := map[key][]int64{}
	for _, sr := range seeds {
		for _, g := range gates(sr) {
			k := key{g.Gate, g.Window, g.Stratum}
			if _, ok := by[k]; !ok {
				order = append(order, k)
			}
			by[k] = append(by[k], g)
			seedOf[k] = append(seedOf[k], sr.Seed)
		}
	}
	var out []GateSummary
	for _, k := range order {
		gs := by[k]
		s := GateSummary{Gate: k.g, Window: k.w, Stratum: k.s, Role: role(p, k.w), Seeds: len(gs), Threshold: gs[0].Threshold}
		s.FailLimit = failLimit(p.Service.MaxFailingSeedFraction, len(gs))
		var sum float64
		for i, g := range gs {
			switch g.Status {
			case Pass:
				s.Passing++
			case Fail:
				s.Failing++
			default:
				s.InsufficientSeeds++
			}
			sum += g.Value
			if i == 0 || g.Value > s.Worst {
				s.Worst, s.WorstSeed = g.Value, seedOf[k][i]
			}
		}
		s.Mean = sum / float64(len(gs))
		s.FailingFraction = wilson(s.Failing+s.InsufficientSeeds, len(gs))
		s.Pooled = pool(gs)
		switch {
		case !s.Pooled.Defined || s.InsufficientSeeds > s.FailLimit:
			s.Status = Insufficient
		case s.Failing+s.InsufficientSeeds > s.FailLimit || s.Pooled.Value > s.Threshold:
			s.Status = Fail
		default:
			s.Status = Pass
		}
		out = append(out, s)
	}
	return out
}
