// Package eval is the frozen V0.2 evaluation (docs/evaluation, V0.2_DESIGN
// §13.4, §13.5): it turns per-seed measurement records of a plan and its paired
// no-action baseline into gate results, paired comparisons and a GO /
// CONDITIONAL GO / NO GO verdict under a machine-readable policy. It is
// pure: it reads records, never a simulation, and the same policy and
// records always give byte-identical output.
package eval

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"slices"
)

const (
	PolicySchema = "dreamteam.evalpolicy/v4"
	// The superseded schemas, kept byte for byte and named by digest in the
	// next policy's supersedes: v0.2-eval-1 (Slice 5C.0), under which no
	// evaluation was run, v0.2-eval-2 (Slice 5C.0a), under which only the
	// Slice 5C.1 development runs were evaluated, and v0.2-eval-3 (Slice
	// 5C.2), under which only the Slice 5C.2 development run was.
	policyV1Schema = "dreamteam.evalpolicy/v1"
	policyV2Schema = "dreamteam.evalpolicy/v2"
	policyV3Schema = "dreamteam.evalpolicy/v3"
	// PolicyV5Schema: a v0.2-eval-5 candidate (Slice 5C.4b), with the
	// clinical acceptance rules (docs/evaluation/proposals/
	// v5-clinical-acceptance.md). Evaluated only for the combined synthetic
	// validation until one is frozen; the policy in force stays v4.
	PolicyV5Schema = "dreamteam.evalpolicy/v5"
)

// Priorities in report order.
var Priorities = []string{"urgent", "soon", "routine"}

// Policy is the frozen evaluation policy. The file is decoded strictly, so
// a misspelt or extra field is an error; every rule field is used.
type Policy struct {
	Schema    string   `json:"schema"`
	ID        string   `json:"id"`
	Status    string   `json:"status"`
	Effective string   `json:"effective"`
	Notes     []string `json:"notes"`
	// Supersedes: the earlier policy, preserved unchanged at Path.
	Supersedes struct {
		ID     string `json:"id"`
		Path   string `json:"path"`
		SHA256 string `json:"sha256"`
	} `json:"supersedes"`
	Scenario struct {
		Fingerprint  string           `json:"fingerprint"`
		TATTargetMin map[string]int64 `json:"tatTargetMin"`
		RotaWeekA    string           `json:"rotaWeekA"`
	} `json:"scenario"`
	Horizons struct {
		FeedbackDays   int `json:"feedbackDays"`
		PrimaryDays    int `json:"primaryDays"`
		OperatingFrom  int `json:"operatingFromDay"` // days (from, primary]
		ValidationDays int `json:"validationDays"`
		// Tail: each record's required tail end is max(T90 + MinDaysAfter-
		// Primary, latest effect + DaysAfterEffect, latest commitment end);
		// the evaluation's common horizon covers every record's, and every
		// record of an evaluation is observed to the same horizon.
		Tail struct {
			MinDaysAfterPrimary int `json:"minDaysAfterPrimary"`
			DaysAfterEffect     int `json:"daysAfterLatestEffect"`
		} `json:"tail"`
		// ClosureMin: how long after a window's end a cohort is observed so
		// every study's breach is decided (the longest TAT target).
		ClosureMin int64 `json:"closureMin"`
	} `json:"horizons"`
	Service struct {
		BreachGate map[string]float64 `json:"breachGate"`
		// Gated windows: each priority's pooled rate must pass in each.
		// Validation windows: pooled and per seed, part of the GO condition.
		Gated      []string `json:"gatedWindows"`
		Validation []string `json:"validationWindows"`
		Info       []string `json:"infoWindows"`
		// PerSeedWindows: the windows in which each priority's breach rate is
		// decided per seed (urgent over the whole gated period, which is
		// what its volume supports; docs/evaluation/README.md §2).
		PerSeedWindows map[string][]string `json:"perSeedWindows"`
		// MinCohort: acquisitions a seed's cohort needs before its breach
		// rate is evidence (fewer: insufficient, never a pass). An evidence
		// floor, not a precision guarantee.
		MinCohort map[string]int64 `json:"minCohort"`
		// MaxFailingSeedFraction: seeds that fail or lack evidence on any
		// per-seed service gate, jointly, at most floor(fraction × seeds).
		MaxFailingSeedFraction float64 `json:"maxFailingSeedFraction"`
		// PlanningRate: the underlying breach rates the evidence design was
		// chosen for (operating characteristics), not gates.
		PlanningRate map[string]float64 `json:"planningRate"`
		// OperatingCharacteristics: the analysis behind the design.
		OperatingCharacteristics string `json:"operatingCharacteristics"`
		// ClusterLevels: the time clusters whose design effects every gate
		// reports (day, weekday-adjusted; rota week). Reported, never gated.
		ClusterLevels []string `json:"clusterLevels"`
		// ReliabilityConfidence: the one-sided confidence of the reported
		// upper bound on the population failing-seed fraction.
		ReliabilityConfidence float64 `json:"reliabilityConfidence"`
	} `json:"service"`
	CaseSelection struct {
		MinStratumDue        int64   `json:"minStratumDue"`
		PerSeedMaxServedDrop float64 `json:"perSeedMaxServedDrop"`
		PooledMaxServedDrop  float64 `json:"pooledMaxServedDrop"`
		MaxAcquiredDrop      float64 `json:"maxAcquiredDrop"`
		MaxLossRateIncrease  float64 `json:"maxLossRateIncrease"`
		// BlockingStrata: kinds of stratum whose lack of evidence blocks
		// eligibility; a rare stratum of another kind is reported
		// insufficient and fails only if it is dropped.
		BlockingStrata []string `json:"blockingStrata"`
	} `json:"caseSelection"`
	Backlog struct {
		BlockDays     int     `json:"blockDays"`
		RotaCycleDays int     `json:"rotaCycleDays"`
		Growth        float64 `json:"growthFraction"`
		GrowthStudies int64   `json:"growthStudies"`
		// ExposureFloorMin: the floor of the overdue-exposure tolerance, in
		// study-minutes past due per sample.
		ExposureFloorMin int64 `json:"exposureFloorStudyMin"`
		// StaleAfterDueMin: an unfinished study this long past its due time
		// is stale; PersistentAfterDueDays: this long, persistent (a seed
		// with one is unstable).
		StaleAfterDueMin       int64   `json:"staleAfterDueMin"`
		PersistentAfterDueDays int     `json:"persistentAfterDueDays"`
		MaxUnstableSeedFrac    float64 `json:"maxUnstableSeedFraction"`
		LegacyRuleDescription  string  `json:"legacyRule"`
		// WorkloadAdjustedChecks compare a block with the reference scaled by
		// the block's workload (its acquisitions over the reference's);
		// AbsoluteChecks never scale. Together they are every check.
		WorkloadAdjustedChecks []string `json:"workloadAdjustedChecks"`
		AbsoluteChecks         []string `json:"absoluteChecks"`
		// InformationalChecks are measured and reported like the others but
		// never make a seed unstable.
		InformationalChecks []string `json:"informationalChecks"`
		// LevelStatistic: the block statistic of the daily backlog the level
		// check compares; LateStatistic: that of the daily overdue, stale
		// and exposure values (and the stale cap). Each "mean" or "median".
		LevelStatistic string `json:"levelStatistic"`
		LateStatistic  string `json:"lateWorkStatistic"`
		// The level check's own tolerance: max(LevelGrowth of the scaled
		// reference, LevelGrowthStudies); the other later-block checks use
		// growthFraction and growthStudies.
		LevelGrowth        float64 `json:"levelGrowthFraction"`
		LevelGrowthStudies int64   `json:"levelGrowthStudies"`
		// StaleCapStudies: no block from the reference through the one
		// covering the horizon may average more stale studies a sample
		// than this, whatever the reference's.
		StaleCapStudies int64 `json:"staleCapStudies"`
	} `json:"backlog"`
	Finance struct {
		MinCashFloorCents int64 `json:"minCashFloorCents"`
		MaxInsolventSeeds int   `json:"maxInsolventSeeds"`
		MaxPastDueCents   int64 `json:"maxPastDueCents"`
		// SettlementDays: observation runs at least this long after the
		// horizon, so every obligation recognized by then falls due and is
		// paid inside the observed run (the longest recurring payment cycle).
		SettlementDays int `json:"settlementDays"`
		// EmployeeCommitmentDays: the forward contracted pay reported as a
		// liquidity-risk indicator (the notice an employer must give); not
		// a gate.
		EmployeeCommitmentDays int `json:"employeeCommitmentDays"`
		// MaxFundingGapSeeds: seeds whose cash at the end of observation is
		// below their committed liabilities outside the payment cycle.
		MaxFundingGapSeeds int `json:"maxFundingGapSeeds"`
	} `json:"finance"`
	Payback struct {
		Horizons            []int   `json:"horizonDays"`
		MedianMinRepaidFrac float64 `json:"medianMinRepaidFraction"`
	} `json:"payback"`
	Seeds struct {
		Development string `json:"development"`
		Reference   string `json:"reference"`
		Holdout     string `json:"holdoutManifest"`
	} `json:"seeds"`
	Verdict []string `json:"verdict"`
	// Changes: each rule changed from the superseded policy.
	Changes []struct {
		Rule      string `json:"rule"`
		Finding   string `json:"finding"`
		Was       string `json:"was"`
		Now       string `json:"now"`
		Rationale string `json:"rationale"`
		Impact    string `json:"expectedImpact"`
	} `json:"changes"`
	// Deferred: design measures not measured by this version.
	Deferred []string `json:"deferredMeasures"`
	// Historical: files the evaluation must leave byte-identical.
	Historical []struct {
		Path   string `json:"path"`
		SHA256 string `json:"sha256"`
	} `json:"historicalReferences"`
	// Clinical: the v0.2-eval-5 clinical acceptance rules; present exactly
	// when the schema is v5. A v4 evaluation never reads it.
	Clinical *Clinical `json:"clinical,omitempty"`
}

// Clinical is v0.2-eval-5's clinical acceptance (proposal §A–§C). Every
// time is minutes; every rate is exact in basis points. Each map has one
// entry per priority.
type Clinical struct {
	// Absolute limits (zero tolerance): no study acquired by the horizon
	// may finish (or, unfinished, be observed) more than this past its due
	// time (A.1, A.2). Urgent ≤ soon ≤ routine.
	AbsoluteMaxPastDueMin map[string]int64 `json:"absoluteMaxPastDueMin"`
	// A study is severely delayed when it is more than this past due (A.3).
	SevereAfterDueMin map[string]int64 `json:"severeAfterDueMin"`
	// Per seed, in the service per-seed windows: severely delayed ÷
	// acquired at most this (A.3) …
	MaxSevereRate map[string]float64 `json:"maxSevereRate"`
	// … and at most this many severely delayed studies acquired in any one
	// Monday-based week of the gated period (A.4).
	MaxSeverePerWeek map[string]int64 `json:"maxSeverePerWeek"`
	// Access (B): an order is timely when imaged within AccessLimitMin of
	// its wanted time. Per seed, in the service per-seed windows, timely ÷
	// decided at least MinTimelyFraction, with at least MinAccessCohort
	// decided orders; pooled the same in every gated window.
	AccessLimitMin    map[string]int64   `json:"accessLimitMin"`
	MinTimelyFraction map[string]float64 `json:"minTimelyFraction"`
	MinAccessCohort   map[string]int64   `json:"minAccessCohort"`
	// MaxWaitMin: no served or open order of the gated period may wait
	// longer from its wanted time (B.2): a seed-level gate, or absolute
	// when MaxWaitAbsolute.
	MaxWaitMin      map[string]int64 `json:"maxWaitMin"`
	MaxWaitAbsolute bool             `json:"maxWaitAbsolute"`
	// InheritedFrom: how inherited work's absolute limits and waits are
	// timed (A.5, B.3): "takeover", from max(due or wanted, takeover); or
	// "own", from its own due or wanted time.
	InheritedFrom string `json:"inheritedFrom"`
	// MaxFailingSeedFraction: the shared allowance (C.1): seeds failing or
	// lacking evidence on any seed-level clinical gate (service, severe
	// delay, access, backlog), each counted once.
	MaxFailingSeedFraction float64 `json:"maxFailingSeedFraction"`
	// MaxWaitReportOnly: the maximum wait is reported, not gated (no
	// maxWaitMin).
	MaxWaitReportOnly bool `json:"maxWaitReportOnly,omitempty"`
	// MaxWeeklySevereRate and MinWeeklyCohort: the volume-normalized
	// weekly severity, in place of maxSeverePerWeek. The gated period's
	// Monday weeks are taken in order and joined into blocks of at least
	// MinWeeklyCohort acquisitions (a short remainder joins the last
	// block); no block may have a severe-delay rate above
	// MaxWeeklySevereRate. Fewer acquisitions in the whole period than
	// the minimum: insufficient. A severe delay must then come strictly
	// before the absolute limit.
	MaxWeeklySevereRate map[string]float64 `json:"maxWeeklySevereRate,omitempty"`
	MinWeeklyCohort     map[string]int64   `json:"minWeeklyCohort,omitempty"`
	// LostDemand (modified option B): when set, lost and left orders leave
	// the timeliness cohorts (counted apart as lost) and the demand the
	// candidate leaves unimaged is gated against the paired baseline's
	// (lostDemandGates).
	LostDemand *LostDemand `json:"lostDemand,omitempty"`
}

// LostDemand is the paired non-deterioration gate on unimaged demand: of
// the eligible orders (received after the takeover, wanted in the gated
// period), those lost, left or ended by the patient. Seeds are paired only
// where the candidate's eligible demand is the baseline's, order for order
// (receipt and wanted times, priority, modality, walk-in); then the
// one-sided upper confidence bound of the pooled increase must not exceed
// MaxIncrease. Unmatched demand on any seed, or fewer than MinCohort
// matched orders, is insufficient.
type LostDemand struct {
	MaxIncrease map[string]float64 `json:"maxIncrease"`
	Confidence  float64            `json:"confidence"`
	MinCohort   map[string]int64   `json:"minCohort"`
}

// LoadPolicy reads and validates a policy file; the digest is the SHA-256
// of its exact bytes.
func LoadPolicy(path string) (*Policy, string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	p, err := DecodePolicy(raw)
	if err != nil {
		return nil, "", fmt.Errorf("%s: %w", path, err)
	}
	sum := sha256.Sum256(raw)
	return p, hex.EncodeToString(sum[:]), nil
}

// DecodePolicy decodes strictly and checks the policy is complete.
func DecodePolicy(raw []byte) (*Policy, error) {
	var head struct {
		Schema string `json:"schema"`
		ID     string `json:"id"`
	}
	if err := json.Unmarshal(raw, &head); err == nil && slices.Contains([]string{policyV1Schema, policyV2Schema, policyV3Schema}, head.Schema) {
		return nil, fmt.Errorf("policy %s is superseded (schema %s): evaluate under %s", head.ID, head.Schema, PolicyPath)
	}
	var p Policy
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&p); err != nil {
		return nil, err
	}
	return &p, p.validate()
}

func (p *Policy) validate() error {
	switch {
	case p.Schema != PolicySchema && p.Schema != PolicyV5Schema:
		return fmt.Errorf("schema %q, want %q", p.Schema, PolicySchema)
	case (p.Schema == PolicyV5Schema) != (p.Clinical != nil):
		return fmt.Errorf("the clinical rules are v0.2-eval-5's: schema %q with clinical %v", p.Schema, p.Clinical != nil)
	case p.Horizons.PrimaryDays <= p.Horizons.FeedbackDays || p.Horizons.OperatingFrom >= p.Horizons.PrimaryDays:
		return fmt.Errorf("horizons out of order")
	case p.Horizons.ClosureMin <= 0:
		return fmt.Errorf("closureMin must be positive")
	case p.Backlog.BlockDays <= 0 || p.Backlog.RotaCycleDays <= 0 || p.Backlog.BlockDays%p.Backlog.RotaCycleDays != 0:
		return fmt.Errorf("backlog blocks must be whole rota cycles")
	case p.Backlog.StaleAfterDueMin <= 0 || p.Backlog.PersistentAfterDueDays <= 0:
		return fmt.Errorf("backlog stale and persistent limits must be positive")
	case p.Finance.EmployeeCommitmentDays <= 0:
		return fmt.Errorf("employeeCommitmentDays must be positive")
	case p.Finance.SettlementDays <= 0:
		return fmt.Errorf("settlementDays must be positive")
	case p.Service.ReliabilityConfidence <= 0 || p.Service.ReliabilityConfidence >= 1:
		return fmt.Errorf("reliabilityConfidence must be in (0, 1)")
	}
	checks := append(slices.Clone(p.Backlog.WorkloadAdjustedChecks), p.Backlog.AbsoluteChecks...)
	slices.Sort(checks)
	if !slices.Equal(checks, backlogChecks) {
		return fmt.Errorf("backlog checks %v: every one of %v exactly once, adjusted or absolute", checks, backlogChecks)
	}
	// Old and persistent work is never normalized, and the age limit is
	// never only reported.
	for _, c := range []string{"stale", "staleCap", "persistent"} {
		if p.adjusted(c) {
			return fmt.Errorf("backlog check %s must be absolute", c)
		}
	}
	// v0.2-eval-5 moves the age limit out of the backlog blocks into the
	// clinical absolute limits (exact timestamps, zero tolerance): there
	// the sampled check is reported only, and must be.
	if v5 := p.Clinical != nil; v5 != p.informational("persistent") {
		return fmt.Errorf("backlog check persistent: informational exactly when the clinical age limit holds (v5 %v)", v5)
	}
	for _, c := range p.Backlog.InformationalChecks {
		if !slices.Contains(backlogChecks, c) {
			return fmt.Errorf("backlog check %q cannot be informational", c)
		}
	}
	switch {
	case !slices.Contains([]string{"mean", "median"}, p.Backlog.LevelStatistic) || !slices.Contains([]string{"mean", "median"}, p.Backlog.LateStatistic):
		return fmt.Errorf("backlog statistics %q, %q: mean or median", p.Backlog.LevelStatistic, p.Backlog.LateStatistic)
	case p.Backlog.LevelGrowth < 0 || p.Backlog.LevelGrowthStudies < 0 || p.Backlog.GrowthStudies < 0 || p.Backlog.StaleCapStudies < 0:
		return fmt.Errorf("backlog tolerances must not be negative")
	}
	for _, l := range p.Service.ClusterLevels {
		if l != "day" && l != "week" {
			return fmt.Errorf("unknown cluster level %q", l)
		}
	}
	fracs := map[string]float64{"maxFailingSeedFraction": p.Service.MaxFailingSeedFraction,
		"perSeedMaxServedDrop": p.CaseSelection.PerSeedMaxServedDrop, "pooledMaxServedDrop": p.CaseSelection.PooledMaxServedDrop,
		"maxAcquiredDrop": p.CaseSelection.MaxAcquiredDrop, "maxLossRateIncrease": p.CaseSelection.MaxLossRateIncrease,
		"growthFraction": p.Backlog.Growth, "levelGrowthFraction": p.Backlog.LevelGrowth, "maxUnstableSeedFraction": p.Backlog.MaxUnstableSeedFrac,
		"reliabilityConfidence": p.Service.ReliabilityConfidence}
	for _, q := range Priorities {
		fracs["breachGate."+q] = p.Service.BreachGate[q]
	}
	for name, f := range fracs {
		if err := wholeBP(name, f); err != nil {
			return err
		}
	}
	for _, q := range Priorities {
		if _, ok := p.Service.BreachGate[q]; !ok {
			return fmt.Errorf("no breach gate for %s", q)
		}
		if p.Service.MinCohort[q] <= 0 {
			return fmt.Errorf("no minimum cohort for %s", q)
		}
		if p.Scenario.TATTargetMin[q] <= 0 {
			return fmt.Errorf("no TAT target for %s", q)
		}
		if p.Scenario.TATTargetMin[q] > p.Horizons.ClosureMin {
			return fmt.Errorf("closure %d min is shorter than the %s target", p.Horizons.ClosureMin, q)
		}
		if len(p.Service.PerSeedWindows[q]) == 0 {
			return fmt.Errorf("no per-seed window for %s", q)
		}
		for _, w := range p.Service.PerSeedWindows[q] {
			if w != WinGated && !slices.Contains(p.Service.Gated, w) {
				return fmt.Errorf("%s per-seed window %q is not gated", q, w)
			}
		}
	}
	for _, k := range p.CaseSelection.BlockingStrata {
		if !slices.Contains(strataKinds, k) {
			return fmt.Errorf("unknown stratum kind %q", k)
		}
	}
	for _, w := range append(append(append([]string{}, p.Service.Gated...), p.Service.Validation...), p.Service.Info...) {
		if !knownWindow(w) || w == WinGated {
			return fmt.Errorf("unknown window %q", w)
		}
	}
	if p.Clinical != nil {
		return p.Clinical.validate(p)
	}
	return nil
}

func (c *Clinical) validate(p *Policy) error {
	// Every limit must be decided inside the observation: it runs at least
	// settlementDays past the horizon, so a study due or an order wanted by
	// the horizon is decided against any limit no longer than that.
	decided := int64(p.Finance.SettlementDays) * 1440
	weekly := c.MaxWeeklySevereRate != nil
	switch {
	case c.MaxWaitReportOnly && (len(c.MaxWaitMin) > 0 || c.MaxWaitAbsolute):
		return fmt.Errorf("clinical maxWaitReportOnly: no maxWaitMin and no absolute maximum wait")
	case weekly && len(c.MaxSeverePerWeek) > 0:
		return fmt.Errorf("clinical maxWeeklySevereRate replaces maxSeverePerWeek: give one")
	}
	for _, q := range Priorities {
		limits := map[string]int64{"absoluteMaxPastDueMin": c.AbsoluteMaxPastDueMin[q], "severeAfterDueMin": c.SevereAfterDueMin[q],
			"accessLimitMin": c.AccessLimitMin[q]}
		if !c.MaxWaitReportOnly {
			limits["maxWaitMin"] = c.MaxWaitMin[q]
		}
		for name, v := range limits {
			if v <= 0 || v > decided {
				return fmt.Errorf("clinical %s.%s %d: positive and at most the %d minutes observed past the horizon", name, q, v, decided)
			}
		}
		if weekly {
			r := c.MaxWeeklySevereRate[q]
			if err := wholeBP("clinical maxWeeklySevereRate."+q, r); err != nil {
				return err
			}
			if r <= 0 || r >= 1 || c.MinWeeklyCohort[q] <= 0 {
				return fmt.Errorf("clinical %s: maxWeeklySevereRate in (0, 1) and a positive minWeeklyCohort", q)
			}
			if c.SevereAfterDueMin[q] >= c.AbsoluteMaxPastDueMin[q] {
				return fmt.Errorf("clinical %s: a severe delay (%d min) must be defined apart from, and before, the absolute limit (%d min)", q, c.SevereAfterDueMin[q], c.AbsoluteMaxPastDueMin[q])
			}
		}
		if ld := c.LostDemand; ld != nil {
			if err := wholeBP("clinical lostDemand.maxIncrease."+q, ld.MaxIncrease[q]); err != nil {
				return err
			}
			if ld.MaxIncrease[q] < 0 || ld.MaxIncrease[q] >= 1 || ld.MinCohort[q] <= 0 {
				return fmt.Errorf("clinical lostDemand %s: maxIncrease in [0, 1) and a positive minCohort", q)
			}
		}
		if c.MaxSeverePerWeek[q] < 0 || c.MinAccessCohort[q] <= 0 {
			return fmt.Errorf("clinical %s: maxSeverePerWeek must not be negative, minAccessCohort must be positive", q)
		}
		for name, f := range map[string]float64{"maxSevereRate": c.MaxSevereRate[q], "minTimelyFraction": c.MinTimelyFraction[q]} {
			if err := wholeBP("clinical "+name+"."+q, f); err != nil {
				return err
			}
			if f < 0 || f > 1 {
				return fmt.Errorf("clinical %s.%s %v: a fraction", name, q, f)
			}
		}
		if c.SevereAfterDueMin[q] > c.AbsoluteMaxPastDueMin[q] || (!c.MaxWaitReportOnly && c.AccessLimitMin[q] > c.MaxWaitMin[q]) {
			return fmt.Errorf("clinical %s: a severe delay must come before the absolute limit, timely access before the maximum wait", q)
		}
	}
	a := c.AbsoluteMaxPastDueMin
	if a["urgent"] > a["soon"] || a["soon"] > a["routine"] {
		return fmt.Errorf("clinical absolute limits must not allow a more urgent priority a longer delay: %v", a)
	}
	if ld := c.LostDemand; ld != nil {
		if err := wholeBP("clinical lostDemand.confidence", ld.Confidence); err != nil {
			return err
		}
		if ld.Confidence <= 0.5 || ld.Confidence >= 1 {
			return fmt.Errorf("clinical lostDemand.confidence %v: in (0.5, 1)", ld.Confidence)
		}
	}
	if c.InheritedFrom != "takeover" && c.InheritedFrom != "own" {
		return fmt.Errorf("clinical inheritedFrom %q: takeover or own", c.InheritedFrom)
	}
	return wholeBP("clinical maxFailingSeedFraction", c.MaxFailingSeedFraction)
}

// backlogChecks are the backlog rule's checks, sorted.
var backlogChecks = []string{"exposure", "level", "max", "mean", "overdue", "persistent", "stale", "staleCap"}

// adjusted reports whether backlog check name scales with workload.
func (p *Policy) adjusted(name string) bool {
	return slices.Contains(p.Backlog.WorkloadAdjustedChecks, name)
}

// informational reports whether backlog check name is reported only.
func (p *Policy) informational(name string) bool {
	return slices.Contains(p.Backlog.InformationalChecks, name)
}

// Window names.
const (
	WinT30  = "T30"
	WinT90  = "T90"
	WinOps  = "D31-90"
	WinTail = "tail"
	WinT180 = "T180"
	// WinGated is the whole gated period (t0, tailEnd]: T90 and the tail.
	WinGated = "gated"
)

func knownWindow(w string) bool {
	switch w {
	case WinT30, WinT90, WinOps, WinTail, WinT180, WinGated:
		return true
	}
	return false
}
