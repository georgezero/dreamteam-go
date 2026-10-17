package eval

import (
	"fmt"
	"slices"
	"sort"
)

const ReportSchema = "dreamteam.evalreport/v2"

// Verdicts.
const (
	GO          = "GO"
	CONDITIONAL = "CONDITIONAL GO"
	NOGO        = "NO GO"
	INVALID     = "INVALID"
)

// Financial classifications of a comparison that passes every hard gate
// (README §10): what kind of result it is, beside the verdict.
const (
	ClassFinancial        = "financial superiority" // profitable, and better than the baseline with statistical support
	ClassRelative         = "relative improvement"  // better than the baseline, without every GO condition
	ClassServicePurchase  = "service purchase"      // restores service the baseline lacks, at a cost
	ClassNoBenefit        = "no benefit"
	ClassNotJudged        = "not judged" // invalid, or a hard gate failed
	reasonServiceRestored = "restores service %s lacks (%s), at a T90 cost of %.0f [%.0f, %.0f] cents"
)

// Gate statuses.
const (
	Pass         = "pass"
	Fail         = "fail"
	Insufficient = "insufficient"
	// Empty: a case-selection stratum with no demand on either plan:
	// reported, never a pass, and nothing to protect.
	Empty = "empty"
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
	// PerSeed: this gate decides the seed's service (policy
	// perSeedWindows, or a validation window); otherwise it is pooled only.
	PerSeed   bool `json:"perSeed"`
	Undecided bool `json:"undecided,omitempty"` // missing or unresolved: the pool is undefined
}

func (g SeedGate) defined() bool { return g.Den > 0 && !g.Undecided }

// SeedResult is one seed of a plan.
type SeedResult struct {
	Seed       int64         `json:"seed"`
	Service    string        `json:"service"`    // pass, fail, insufficient: every per-seed gated gate
	Validation string        `json:"validation"` // the same over the validation windows
	Backlog    string        `json:"backlog"`
	Solvent    bool          `json:"solvent"`
	Funded     bool          `json:"funded"` // no funding gap through the end of observation
	Gates      []SeedGate    `json:"gates"`
	Measures   *SeedMeasures `json:"measures"`
}

// WorstSeed is an extreme value and the seed it was on.
type WorstSeed struct {
	Value   float64 `json:"value"`
	Seed    int64   `json:"seed"`
	Defined bool    `json:"defined"`
}

func (w *WorstSeed) note(v float64, seed int64, larger bool) {
	if !w.Defined || (larger && v > w.Value) || (!larger && v < w.Value) {
		*w = WorstSeed{Value: v, Seed: seed, Defined: true}
	}
}

// GateSummary is one gate across seeds: its pooled value, which decides
// it, and how its seeds fared (each seed's service is decided jointly, in
// JointGate). Mean and worst are over the seeds whose value is defined.
type GateSummary struct {
	Gate              string    `json:"gate"`
	Window            string    `json:"window"`
	Stratum           string    `json:"stratum,omitempty"`
	Role              string    `json:"role"` // gated, validation, info
	PerSeed           bool      `json:"perSeed"`
	Status            string    `json:"status"`
	Threshold         float64   `json:"threshold"`
	Seeds             int       `json:"seeds"`
	Passing           int       `json:"passing"`
	Failing           int       `json:"failing"`
	InsufficientSeeds int       `json:"insufficientSeeds"`
	DefinedSeeds      int       `json:"definedSeeds"`
	Pooled            Ratio     `json:"pooled"`
	Worst             WorstSeed `json:"worst"`
	Mean              float64   `json:"mean"`
	// Dispersion: the variance of the defined seeds' rates over the
	// binomial variance their pooled rate and sizes imply (1: as
	// independent studies; above 1: seed variation or clustered
	// breaches). Reported to check the operating characteristics'
	// independence assumption, never gated.
	Dispersion float64 `json:"dispersion"`
	// Case selection only: whether this gate's lack of evidence blocks
	// eligibility, and its per-seed failure limit.
	Blocking  bool `json:"blocking,omitempty"`
	FailLimit int  `json:"failLimit,omitempty"`
}

// JointGate is the plan's seed reliability over every per-seed service
// gate of a role together: a seed fails if any of them fails or lacks
// evidence, and at most FailLimit seeds may.
type JointGate struct {
	Role              string   `json:"role"`
	Gates             int      `json:"gatesPerSeed"`
	Seeds             int      `json:"seeds"`
	FailingSeeds      []int64  `json:"failingSeeds"`
	InsufficientSeeds []int64  `json:"insufficientSeeds"` // lacking evidence and failing none
	FailLimit         int      `json:"failLimit"`
	SeedFraction      Interval `json:"failingOrInsufficientSeedFraction"` // Wilson 95% of the seed fraction (seeds, not studies)
	Status            string   `json:"status"`
}

// PlanReport is one plan alone: its own gates, no comparison.
type PlanReport struct {
	Plan          string        `json:"plan"`
	Seeds         []int64       `json:"seeds"`
	HorizonEndSec int64         `json:"horizonEndSec"`
	Valid         bool          `json:"valid"`
	Integrity     []string      `json:"integrity"`
	ServiceGates  []GateSummary `json:"serviceGates"`
	ServiceJoint  JointGate     `json:"serviceJoint"`
	Validation    JointGate     `json:"validationJoint"`
	ServiceStatus string        `json:"serviceStatus"` // the joint seed rule and every gated pooled rate
	// ValidationPass: the validation windows' joint seed rule and pooled
	// rates both pass.
	ValidationPass  bool    `json:"validationServicePass"`
	BacklogStable   int     `json:"backlogStableSeeds"`
	BacklogStatus   string  `json:"backlogStatus"`
	LegacyBacklog   string  `json:"legacyBacklogRule"` // the historical rule on the seed means, for comparison only
	InsolventSeeds  []int64 `json:"insolventSeeds"`
	WorstMinCash    int64   `json:"worstMinCashCents"`
	WorstMinCashAt  int64   `json:"worstMinCashSeed"`
	SolvencyStatus  string  `json:"solvencyStatus"`
	FundingGapSeeds []int64 `json:"fundingGapSeeds"` // a gap at any capture or sample through the end of observation
	FundingStatus   string  `json:"fundingStatus"`
	// OperatingProfit: the plan's own (absolute) evaluation operating
	// profit per window, mean and t 95% interval over seeds.
	OperatingProfit map[string]Interval `json:"operatingProfitCents"`
	Eligible        bool                `json:"eligible"` // service, backlog, solvency and funding, before comparison
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
	Seeds         int      `json:"seeds"` // every pair: a pair without samples through the horizon is an integrity failure
	Days          []int    `json:"days"`  // per seed; 0 = censored
	Repaid        int      `json:"repaid"`
	Censored      int      `json:"censored"`
	RepaidFrac    Interval `json:"repaidFraction"`
	Median        int      `json:"median"`        // 0 unless at least half repaid
	MedianDefined bool     `json:"medianDefined"` // false: report "not reached by the horizon"
	Earliest      int      `json:"earliest"`
	Latest        int      `json:"latest"`
}

// BreachExtremes are a window's and priority's worst breach rates: each
// plan's own worst seed, and the largest paired increase.
type BreachExtremes struct {
	Candidate       WorstSeed `json:"candidate"`
	Against         WorstSeed `json:"against"`
	LargestIncrease WorstSeed `json:"largestPairedIncrease"`
}

// Comparison is a candidate against its paired baseline (or another plan).
type Comparison struct {
	Candidate string   `json:"candidate"`
	Against   string   `json:"against"`
	Valid     bool     `json:"valid"`
	Integrity []string `json:"integrity"`
	// Paired differences, candidate − against, per window.
	Profit              map[string]Interval                  `json:"incrementalProfitCents"`
	WorstProfit         map[string]WorstSeed                 `json:"worstIncrementalProfitSeed"`
	Revenue             map[string]Interval                  `json:"incrementalNetRevenueCents"`
	Reports             map[string]Interval                  `json:"incrementalReports"`
	CashAtEnd           map[string]Interval                  `json:"incrementalCashCents"`
	BreachDelta         map[string]map[string]Interval       `json:"breachRateDelta"` // window → priority
	TATP90Delta         map[string]map[string]Interval       `json:"completedTatP90DeltaMin"`
	WorstBreach         map[string]map[string]BreachExtremes `json:"worstSeedBreachRate"` // window → priority
	MinCashDelta        Interval                             `json:"minCashDeltaCents"`
	RemoteCost          map[string]Interval                  `json:"remoteCostCents"`
	UnbilledDelta       map[string]Interval                  `json:"unbilledCompletedDeltaCents"` // listed apart: not revenue
	AccrualPayback      []Payback                            `json:"accrualPayback"`
	CashPayback         []Payback                            `json:"cashPayback"`
	CaseSelection       []GateSummary                        `json:"caseSelection"`
	CaseSelectionStatus string                               `json:"caseSelectionStatus"`
	Verdict             string                               `json:"verdict,omitempty"`
	Classification      string                               `json:"classification,omitempty"`
	Reasons             []string                             `json:"reasons,omitempty"`
}

// Report is a whole evaluation.
type Report struct {
	Schema        string       `json:"schema"`
	PolicyID      string       `json:"policyId"`
	PolicyDigest  string       `json:"policyDigest"`
	Baseline      string       `json:"baseline"`
	HorizonEndSec int64        `json:"horizonEndSec"`
	Integrity     []string     `json:"integrity"` // evaluation-wide: every comparison is invalid if any
	Plans         []PlanReport `json:"plans"`
	Comparisons   []Comparison `json:"comparisons"`
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
	rep := &Report{Schema: ReportSchema, PolicyID: p.ID, PolicyDigest: digest, Baseline: in.Baseline, Integrity: []string{}}
	names := make([]string, 0, len(in.Plans))
	for n := range in.Plans {
		names = append(names, n)
	}
	sort.Strings(names)
	// One horizon and one observation end for every record (H2).
	type bounds struct{ h, obs int64 }
	seen := map[bounds][]string{}
	var order []bounds
	for _, n := range names {
		for _, r := range in.Plans[n] {
			b := bounds{r.HorizonEndSec, r.ObservedUntilSec}
			if _, ok := seen[b]; !ok {
				order = append(order, b)
			}
			seen[b] = append(seen[b], fmt.Sprintf("%s/%d", n, r.Seed))
		}
	}
	if len(order) > 0 {
		rep.HorizonEndSec = order[0].h
	}
	if len(order) > 1 {
		for _, b := range order {
			rep.Integrity = append(rep.Integrity, fmt.Sprintf("records do not share one horizon: horizon %d, observed to %d for %v", b.h, b.obs, seen[b]))
		}
	}
	plans := map[string]*PlanReport{}
	for _, n := range names {
		plans[n] = planReport(p, n, in.Plans[n])
	}
	for _, n := range names {
		bookingInvariance(plans[n], in.Plans[n], in.Plans)
		rep.Plans = append(rep.Plans, *plans[n])
	}
	base := plans[in.Baseline]
	for _, n := range names {
		if n == in.Baseline || base == nil {
			continue
		}
		c := compare(p, plans[n], base, in.Plans[n], in.Plans[in.Baseline], rep.Integrity)
		verdict(p, &c, plans[n], base)
		rep.Comparisons = append(rep.Comparisons, c)
	}
	for _, pair := range in.Pairs {
		a, b := plans[pair[0]], plans[pair[1]]
		if a == nil || b == nil {
			continue
		}
		rep.Comparisons = append(rep.Comparisons, compare(p, a, b, in.Plans[pair[0]], in.Plans[pair[1]], rep.Integrity))
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
	case w == WinGated:
		return "gated" // the whole gated period: per-seed only
	}
	return "info"
}

func planReport(p *Policy, name string, recs []Record) *PlanReport {
	pr := &PlanReport{Plan: name, Valid: true, Integrity: []string{}, InsolventSeeds: []int64{}, FundingGapSeeds: []int64{}, OperatingProfit: map[string]Interval{}}
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
		sr.Funded = m.FundingGap <= 0
		sr.Gates = serviceGates(p, m)
		sr.Service, sr.Validation = seedStatus(p, sr.Gates, "gated"), seedStatus(p, sr.Gates, "validation")
		if !sr.Solvent {
			pr.InsolventSeeds = append(pr.InsolventSeeds, r.Seed)
		}
		if !sr.Funded {
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
	if len(recs) > 0 {
		pr.HorizonEndSec = recs[0].HorizonEndSec
	}
	slices.Sort(pr.Integrity)
	pr.Valid = len(pr.Integrity) == 0 && len(recs) > 0
	n := len(pr.PerSeed)
	pr.ServiceGates = summarize(p, pr.PerSeed)
	pr.ServiceJoint, pr.Validation = joint(p, pr.PerSeed, "gated"), joint(p, pr.PerSeed, "validation")
	pr.ServiceStatus, pr.ValidationPass = pr.ServiceJoint.Status, pr.Validation.Status == Pass
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
	if n-pr.BacklogStable > failLimit(bp(p.Backlog.MaxUnstableSeedFrac), n) {
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
	pr.FundingStatus = Pass
	if len(pr.FundingGapSeeds) > p.Finance.MaxFundingGapSeeds {
		pr.FundingStatus = Fail
	}
	for _, w := range []string{WinT30, WinT90, WinOps, WinTail, WinT180} {
		var xs []float64
		for _, s := range pr.PerSeed {
			if wm := s.Measures.Windows[w]; wm != nil {
				xs = append(xs, float64(wm.Money.OperatingProfitCents))
			}
		}
		if len(xs) == n && n > 0 {
			pr.OperatingProfit[w] = pairedT(xs)
		}
	}
	pr.Eligible = pr.Valid && pr.ServiceStatus == Pass && pr.BacklogStatus == Pass && pr.SolvencyStatus == Pass && pr.FundingStatus == Pass
	return pr
}

// perSeedGate reports whether the breach gate of priority q in window w
// decides the seed: a per-seed window of the priority, or a validation
// window.
func perSeedGate(p *Policy, q, w string) bool {
	return slices.Contains(p.Service.PerSeedWindows[q], w) || slices.Contains(p.Service.Validation, w)
}

// serviceGates are a seed's breach gates in every window it has, decided
// exactly (a rate exactly at its gate passes).
func serviceGates(p *Policy, m *SeedMeasures) []SeedGate {
	var out []SeedGate
	for _, w := range windowOrder(p) {
		wm := m.Windows[w]
		for _, q := range Priorities {
			g := SeedGate{Gate: "breach_rate", Window: w, Stratum: q, Threshold: p.Service.BreachGate[q], PerSeed: perSeedGate(p, q, w)}
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
			case rateAbove(c.Breaches, c.Acquired, bp(g.Threshold)):
				g.Status = Fail
			default:
				g.Status = Pass
			}
			out = append(out, g)
		}
	}
	return out
}

// seedStatus combines a seed's per-seed gates of a role: fail if any
// fails, else insufficient if any lacks evidence.
func seedStatus(p *Policy, gs []SeedGate, rl string) string {
	st := Pass
	for _, g := range gs {
		if !g.PerSeed || role(p, g.Window) != rl {
			continue
		}
		switch {
		case g.Status == Fail:
			st = Fail
		case g.Status == Insufficient && st == Pass:
			st = Insufficient
		}
	}
	return st
}

// joint applies the seed-reliability rule to a role's per-seed gates
// together (H5): the seeds failing or lacking evidence on any of them, at
// most floor(fraction × seeds).
func joint(p *Policy, seeds []SeedResult, rl string) JointGate {
	j := JointGate{Role: rl, Seeds: len(seeds), FailingSeeds: []int64{}, InsufficientSeeds: []int64{}}
	j.FailLimit = failLimit(bp(p.Service.MaxFailingSeedFraction), len(seeds))
	if len(seeds) > 0 {
		for _, g := range seeds[0].Gates {
			if g.PerSeed && role(p, g.Window) == rl {
				j.Gates++
			}
		}
	}
	for _, s := range seeds {
		st := s.Service
		if rl == "validation" {
			st = s.Validation
		}
		switch st {
		case Fail:
			j.FailingSeeds = append(j.FailingSeeds, s.Seed)
		case Insufficient:
			j.InsufficientSeeds = append(j.InsufficientSeeds, s.Seed)
		}
	}
	bad := len(j.FailingSeeds) + len(j.InsufficientSeeds)
	j.SeedFraction = wilson(bad, len(seeds))
	switch {
	case len(seeds) == 0 || j.Gates == 0 || len(j.InsufficientSeeds) > j.FailLimit:
		j.Status = Insufficient
	case bad > j.FailLimit:
		j.Status = Fail
	default:
		j.Status = Pass
	}
	return j
}

func windowOrder(p *Policy) []string {
	var out []string
	for _, w := range []string{WinT30, WinT90, WinOps, WinTail, WinGated, WinT180} {
		if role(p, w) != "info" || slices.Contains(p.Service.Info, w) {
			out = append(out, w)
		}
	}
	return out
}

// summarize groups seed gates by (window, stratum) in first-seen order.
// A gated or validation window's gate is decided by its pooled rate (all
// seeds' breaches over all their acquisitions, exactly against the gate;
// undefined if any seed's cohort is missing or undecided, or all are
// empty); the whole-period urgent gate decides seeds only.
func summarize(p *Policy, seeds []SeedResult) []GateSummary {
	type key struct{ g, w, s string }
	var order []key
	by := map[key][]SeedGate{}
	seedOf := map[key][]int64{}
	for _, sr := range seeds {
		for _, g := range sr.Gates {
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
		s := GateSummary{Gate: k.g, Window: k.w, Stratum: k.s, Role: role(p, k.w), PerSeed: gs[0].PerSeed, Seeds: len(gs), Threshold: gs[0].Threshold}
		var sum float64
		var num, den int64
		pooled := true
		for i, g := range gs {
			switch g.Status {
			case Pass:
				s.Passing++
			case Fail:
				s.Failing++
			default:
				s.InsufficientSeeds++
			}
			if g.Undecided {
				pooled = false
			}
			num, den = num+g.Num, den+g.Den
			if g.defined() {
				s.DefinedSeeds++
				sum += g.Value
				s.Worst.note(g.Value, seedOf[k][i], true)
			}
		}
		if s.DefinedSeeds > 0 {
			s.Mean = sum / float64(s.DefinedSeeds)
		}
		s.Dispersion = dispersion(gs)
		if pooled {
			s.Pooled = ratio(num, den)
		}
		switch {
		case k.w == WinGated:
			s.Status = "per-seed" // judged in the joint seed rule only
		case !s.Pooled.Defined:
			s.Status = Insufficient
		case rateAbove(s.Pooled.Num, s.Pooled.Den, bp(s.Threshold)):
			s.Status = Fail
		default:
			s.Status = Pass
		}
		out = append(out, s)
	}
	return out
}

// dispersion is the observed variance of the defined seeds' rates over
// the mean binomial variance p(1 − p)/n at their pooled rate (0 when
// fewer than two seeds or the pooled rate is 0 or 1).
func dispersion(gs []SeedGate) float64 {
	var num, den, k int64
	var invN float64
	for _, g := range gs {
		if g.defined() {
			num, den, k = num+g.Num, den+g.Den, k+1
			invN += 1 / float64(g.Den)
		}
	}
	if k < 2 || num == 0 || num == den {
		return 0
	}
	p := float64(num) / float64(den)
	var ss float64
	for _, g := range gs {
		if g.defined() {
			x := float64(g.Num)/float64(g.Den) - p
			ss += x * x
		}
	}
	return (ss / float64(k-1)) / (p * (1 - p) * invN / float64(k))
}
