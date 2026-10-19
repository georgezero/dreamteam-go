package eval

import (
	"fmt"
	"slices"
	"sort"
)

const ReportSchema = "dreamteam.evalreport/v4"

// ReportSchemaV5 is a report under a v0.2-eval-5 (clinical) policy.
const ReportSchemaV5 = "dreamteam.evalreport/v5"

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
	ClassFinancial        = "financial superiority"      // profitable, and better than the baseline with statistical support
	ClassRelative         = "relative improvement"       // better than the baseline, without every GO condition
	ClassTransition       = "transition, not yet repaid" // eligible and solvent, ahead after the transition, not yet at T90
	ClassServicePurchase  = "service purchase"           // restores service the baseline lacks, at a cost
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
	// The cohort's day and week design effects (0: undefined).
	DayDesignEffect  float64 `json:"dayDesignEffect"`
	WeekDesignEffect float64 `json:"weekDesignEffect"`
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
	// v0.2-eval-5: the clinical seed-level gates, and the seed's shared
	// clinical status (service, those gates and backlog together).
	ClinicalGates []SeedGate `json:"clinicalGates,omitempty"`
	Clinical      string     `json:"clinical,omitempty"`
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
	// ClusterRobust: the pooled rate with a 95% interval that treats the
	// seeds as the independent units (ratio estimator): valid whatever the
	// dependence inside a seed. DayDesignEffect and WeekDesignEffect are
	// the defined seeds' means. Reported, never gated.
	ClusterRobust    Interval `json:"clusterRobustPooled"`
	DayDesignEffect  float64  `json:"meanDayDesignEffect"`
	WeekDesignEffect float64  `json:"meanWeekDesignEffect"`
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
	// The prospective reading of the allowance (README §2): PassFraction
	// is the empirical share of seeds passing every per-seed gate; Upper,
	// the exact one-sided upper bound (at the policy's reliability
	// confidence) on the population share of failing seeds. Passing the
	// rule is an empirical statement: it does not establish that the
	// population share is within the allowance unless Upper is.
	PassFraction float64 `json:"passFraction"`
	Upper        float64 `json:"failingFractionUpperBound"`
	Confidence   float64 `json:"confidence"`
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
	ValidationPass bool    `json:"validationServicePass"`
	BacklogStable  int     `json:"backlogStableSeeds"`
	BacklogStatus  string  `json:"backlogStatus"`
	LegacyBacklog  string  `json:"legacyBacklogRule"` // the historical rule on the seed means, for comparison only
	InsolventSeeds []int64 `json:"insolventSeeds"`
	WorstMinCash   int64   `json:"worstMinCashCents"`
	WorstMinCashAt int64   `json:"worstMinCashSeed"`
	SolvencyStatus string  `json:"solvencyStatus"`
	// FundingGapSeeds: the dated settlement test fails (committed
	// liabilities outside the payment cycle above cash at the end of
	// observation). The paid-when-due part is solvency's.
	FundingGapSeeds []int64 `json:"fundingGapSeeds"`
	FundingStatus   string  `json:"fundingStatus"`
	// The 14-day commitment coverage, reported only: the seeds whose full
	// bridge (future wages included) exceeded cash at some capture or
	// sample, the largest such gap, and the share of instants in gap.
	CommitmentGapSeeds    []int64 `json:"commitmentGapSeeds"`
	WorstCommitmentGap    int64   `json:"worstCommitmentGapCents"`
	CommitmentGapFraction Ratio   `json:"commitmentGapInstantFraction"`
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
	// v0.2-eval-5: the clinical gates' summaries, the shared seed
	// allowance over every seed-level clinical gate, its status with every
	// gated pooled rate, and the absolute limits. Eligibility then rests on
	// these, solvency and funding; the separate service and backlog
	// statuses above stay reported.
	ClinicalGates  []GateSummary `json:"clinicalGates,omitempty"`
	ClinicalJoint  *JointGate    `json:"clinicalJoint,omitempty"`
	ClinicalStatus string        `json:"clinicalStatus,omitempty"`
	ClinicalSafety *Safety       `json:"clinicalSafety,omitempty"`
	// ClinicalInsufficient: by seed-level gate of the shared allowance,
	// every seed lacking evidence on it, whatever its other gates' outcomes
	// (Slice 5C.4h, audit G1: a seed failing elsewhere is FAIL in the
	// joint rule, so the joint lists cannot give these counts).
	ClinicalInsufficient map[string][]int64 `json:"clinicalInsufficientSeedsByGate,omitempty"`
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
	// LostDemand: v0.2-eval-5's lost-demand gates against the paired plan
	// (policies with clinical.lostDemand only).
	LostDemand []LostDemandGate `json:"lostDemand,omitempty"`
	// Access: the candidate's access by priority, always shown with the
	// lost-demand gates: timeliness, absolute unimaged demand, the
	// baseline's, and the comparison's uncertainty. AccessLimitation is
	// set on every favourable verdict: the access gates establish only
	// non-deterioration relative to the baseline.
	Access           []AccessStatement `json:"access,omitempty"`
	AccessLimitation string            `json:"accessLimitation,omitempty"`
	// AccessClaims (the empirical rule): the empirical acceptance apart
	// from population confidence and absolute adequacy. ClinicalJoint:
	// with the per-seed safeguard, the candidate's shared clinical
	// allowance with each seed's safeguard counted in; the verdict uses
	// it in place of the plan's.
	AccessClaims  *AccessClaims `json:"accessClaims,omitempty"`
	ClinicalJoint *JointGate    `json:"clinicalJointWithSafeguard,omitempty"`
	// Financial: the classification's evidence, kept apart (README §10).
	Financial Financial `json:"financial"`
	// ServiceRestored: the baseline fails the service gates the candidate
	// passes (whatever the verdict); a CONDITIONAL GO restores it at the
	// T90 increment's cost.
	ServiceRestored bool     `json:"serviceRestored"`
	Role            string   `json:"role,omitempty"` // primary or exploratory, when the evaluation names a primary
	Verdict         string   `json:"verdict,omitempty"`
	Classification  string   `json:"classification,omitempty"`
	Reasons         []string `json:"reasons,omitempty"`
}

// Financial is a comparison's money, kept apart: the candidate's own
// (absolute) operating profit and its paired increment at T90 (primary),
// over the operating window (days 31–90) and at T180 (validation), and
// the transition obligations still outstanding at T90. Payback is in the
// comparison's accrual and cash paybacks.
type Financial struct {
	T90Absolute        Interval `json:"t90AbsoluteCents"`
	T90Increment       Interval `json:"t90IncrementCents"`
	OperatingAbsolute  Interval `json:"operatingWindowAbsoluteCents"`
	OperatingIncrement Interval `json:"operatingWindowIncrementCents"`
	T180Absolute       Interval `json:"t180AbsoluteCents"`
	T180Increment      Interval `json:"t180IncrementCents"`
	// TransitionObligationsT90: severance payable and contract minimum
	// retainers owed at T90 (the candidate's, mean over seeds).
	TransitionObligationsT90 Interval `json:"transitionObligationsT90Cents"`
}

// Report is a whole evaluation.
type Report struct {
	Schema        string       `json:"schema"`
	PolicyID      string       `json:"policyId"`
	PolicyDigest  string       `json:"policyDigest"`
	Baseline      string       `json:"baseline"`
	Primary       string       `json:"primary,omitempty"`
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
	// Seeds, when set, is the complete seed set every plan must have (a
	// holdout run's matrix); PlanSHA, when set, the strategy digest every
	// record of each plan must carry; Primary, when set, the one
	// prespecified strategy the evaluation confirms: every other
	// candidate's comparison is exploratory.
	Seeds   []int64
	PlanSHA map[string]string
	Primary string
}

// Evaluate applies the policy. The output depends only on its arguments.
func Evaluate(p *Policy, digest string, in Input) *Report {
	rep := &Report{Schema: ReportSchema, PolicyID: p.ID, PolicyDigest: digest, Baseline: in.Baseline, Primary: in.Primary, Integrity: []string{}}
	if p.Clinical != nil {
		rep.Schema = ReportSchemaV5
	}
	names := make([]string, 0, len(in.Plans))
	for n := range in.Plans {
		names = append(names, n)
	}
	sort.Strings(names)
	want := slices.Sorted(slices.Values(in.Seeds))
	for _, n := range names {
		var seeds []int64
		shas := map[string]bool{}
		for _, r := range in.Plans[n] {
			seeds = append(seeds, r.Seed)
			shas[r.PlanSHA256] = true
			if s, ok := in.PlanSHA[n]; ok && r.PlanSHA256 != s {
				rep.Integrity = append(rep.Integrity, fmt.Sprintf("plan %s seed %d: strategy digest %q, the set pins %q", n, r.Seed, r.PlanSHA256, s))
			}
		}
		if len(shas) > 1 {
			rep.Integrity = append(rep.Integrity, fmt.Sprintf("plan %s: its records carry %d strategy digests", n, len(shas)))
		}
		if in.PlanSHA != nil {
			if _, ok := in.PlanSHA[n]; !ok {
				rep.Integrity = append(rep.Integrity, fmt.Sprintf("plan %s is not in the strategy set", n))
			}
		}
		if len(want) > 0 && !slices.Equal(slices.Sorted(slices.Values(seeds)), want) {
			rep.Integrity = append(rep.Integrity, fmt.Sprintf("plan %s: %d seeds, not the required %d", n, len(seeds), len(want)))
		}
	}
	for n := range in.PlanSHA {
		if _, ok := in.Plans[n]; !ok {
			rep.Integrity = append(rep.Integrity, fmt.Sprintf("strategy %s has no records", n))
		}
	}
	if in.Primary != "" && (in.Primary == in.Baseline || in.Plans[in.Primary] == nil) {
		rep.Integrity = append(rep.Integrity, fmt.Sprintf("primary %q is not a candidate", in.Primary))
	}
	slices.Sort(rep.Integrity)
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
		switch {
		case in.Primary == "":
		case n == in.Primary:
			c.Role = "primary"
		default:
			c.Role = "exploratory"
		}
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
	pr := &PlanReport{Plan: name, Valid: true, Integrity: []string{}, InsolventSeeds: []int64{}, FundingGapSeeds: []int64{},
		CommitmentGapSeeds: []int64{}, OperatingProfit: map[string]Interval{}}
	var gapInstants, instants int64
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
		if p.Clinical != nil {
			sr.ClinicalGates = clinicalGates(p, m)
			sr.Clinical = clinicalSeed(&sr)
		}
		if !sr.Solvent {
			pr.InsolventSeeds = append(pr.InsolventSeeds, r.Seed)
		}
		if !sr.Funded {
			pr.FundingGapSeeds = append(pr.FundingGapSeeds, r.Seed)
		}
		if m.CommitmentGap > 0 {
			pr.CommitmentGapSeeds = append(pr.CommitmentGapSeeds, r.Seed)
			pr.WorstCommitmentGap = max(pr.WorstCommitmentGap, m.CommitmentGap)
		}
		gapInstants, instants = gapInstants+int64(m.CommitmentGapInstant), instants+int64(m.CommitmentInstants)
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
	pr.CommitmentGapFraction = ratio(gapInstants, instants)
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
	if p.Clinical != nil {
		clinicalPlan(p, pr)
	}
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
			g.DayDesignEffect, g.WeekDesignEffect = c.DayDesignEffect, c.WeekDesignEffect
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
	j := jointOf(p, seeds, rl, bp(p.Service.MaxFailingSeedFraction), func(s *SeedResult) string {
		if rl == "validation" {
			return s.Validation
		}
		return s.Service
	})
	if len(seeds) > 0 {
		for _, g := range seeds[0].Gates {
			if g.PerSeed && role(p, g.Window) == rl {
				j.Gates++
			}
		}
	}
	if len(seeds) > 0 && j.Gates == 0 {
		j.Status = Insufficient
	}
	return j
}

// jointOf is the seed rule over each seed's status (status): the seeds
// failing or lacking evidence, at most floor(fraction × seeds) of them.
// The caller counts the gates per seed.
func jointOf(p *Policy, seeds []SeedResult, rl string, fracBP int64, status func(*SeedResult) string) JointGate {
	j := JointGate{Role: rl, Seeds: len(seeds), FailingSeeds: []int64{}, InsufficientSeeds: []int64{}}
	j.FailLimit = failLimit(fracBP, len(seeds))
	for i := range seeds {
		s := &seeds[i]
		switch status(s) {
		case Fail:
			j.FailingSeeds = append(j.FailingSeeds, s.Seed)
		case Insufficient:
			j.InsufficientSeeds = append(j.InsufficientSeeds, s.Seed)
		}
	}
	bad := len(j.FailingSeeds) + len(j.InsufficientSeeds)
	j.SeedFraction = wilson(bad, len(seeds))
	if len(seeds) > 0 {
		j.PassFraction = float64(len(seeds)-bad) / float64(len(seeds))
		j.Confidence = p.Service.ReliabilityConfidence
		j.Upper = binomUpper(bad, len(seeds), j.Confidence)
	}
	switch {
	case len(seeds) == 0 || len(j.InsufficientSeeds) > j.FailLimit:
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
	return summarizeGates(p, seeds, func(s *SeedResult) []SeedGate { return s.Gates })
}

// summarizeGates is summarize over the gates gates returns of each seed.
// A minimum fraction (timely access) fails below its threshold, and its
// worst seed is the lowest; a gate of counts or maxima (severe delays in
// a week, the maximum wait) or one of the whole gated period is judged
// per seed only.
func summarizeGates(p *Policy, seeds []SeedResult, gates func(*SeedResult) []SeedGate) []GateSummary {
	type key struct{ g, w, s string }
	var order []key
	by := map[key][]SeedGate{}
	seedOf := map[key][]int64{}
	for i := range seeds {
		sr := &seeds[i]
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
		s := GateSummary{Gate: k.g, Window: k.w, Stratum: k.s, Role: role(p, k.w), PerSeed: gs[0].PerSeed, Seeds: len(gs), Threshold: gs[0].Threshold}
		var sum, dde, wde float64
		var num, den int64
		var nd, nw int
		var ones, ns []int64
		pooled := true
		for i, g := range gs {
			ones, ns = append(ones, g.Num), append(ns, g.Den)
			if g.DayDesignEffect > 0 {
				dde, nd = dde+g.DayDesignEffect, nd+1
			}
			if g.WeekDesignEffect > 0 {
				wde, nw = wde+g.WeekDesignEffect, nw+1
			}
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
				s.Worst.note(g.Value, seedOf[k][i], k.g != GateTimelyAccess)
			}
		}
		if s.DefinedSeeds > 0 {
			s.Mean = sum / float64(s.DefinedSeeds)
		}
		s.Dispersion = dispersion(gs)
		if pooled {
			s.Pooled = ratio(num, den)
			s.ClusterRobust = clusterRobustRate(ones, ns)
		}
		if nd > 0 {
			s.DayDesignEffect = dde / float64(nd)
		}
		if nw > 0 {
			s.WeekDesignEffect = wde / float64(nw)
		}
		switch {
		case k.w == WinGated || k.g == GateSevereRate || k.g == GateSevereWeek || k.g == GateMaxWait:
			s.Status = "per-seed" // judged in the joint seed rule only
		case !s.Pooled.Defined:
			s.Status = Insufficient
		case k.g == GateTimelyAccess && rateBelow(s.Pooled.Num, s.Pooled.Den, bp(s.Threshold)):
			s.Status = Fail
		case k.g != GateTimelyAccess && rateAbove(s.Pooled.Num, s.Pooled.Den, bp(s.Threshold)):
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
