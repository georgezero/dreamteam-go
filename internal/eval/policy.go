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
	PolicySchema = "dreamteam.evalpolicy/v2"
	// The superseded schema: v0.2-eval-1 (Slice 5C.0) is kept byte for byte
	// and named by digest in every later policy's supersedes; no evaluation
	// was ever run under it, so nothing was decided by it.
	policyV1Schema = "dreamteam.evalpolicy/v1"
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
	} `json:"backlog"`
	Finance struct {
		MinCashFloorCents int64 `json:"minCashFloorCents"`
		MaxInsolventSeeds int   `json:"maxInsolventSeeds"`
		MaxPastDueCents   int64 `json:"maxPastDueCents"`
		// EmployeeCommitmentDays: the forward contracted pay counted as an
		// unavoidable commitment (the notice an employer must give).
		EmployeeCommitmentDays int `json:"employeeCommitmentDays"`
		MaxFundingGapSeeds     int `json:"maxFundingGapSeeds"`
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
	case p.Schema == policyV1Schema:
		return fmt.Errorf("policy %s is superseded (schema %s): evaluate under %s", p.ID, p.Schema, PolicyPath)
	case p.Schema != PolicySchema:
		return fmt.Errorf("schema %q, want %q", p.Schema, PolicySchema)
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
	}
	fracs := map[string]float64{"maxFailingSeedFraction": p.Service.MaxFailingSeedFraction,
		"perSeedMaxServedDrop": p.CaseSelection.PerSeedMaxServedDrop, "pooledMaxServedDrop": p.CaseSelection.PooledMaxServedDrop,
		"maxAcquiredDrop": p.CaseSelection.MaxAcquiredDrop, "maxLossRateIncrease": p.CaseSelection.MaxLossRateIncrease,
		"growthFraction": p.Backlog.Growth, "maxUnstableSeedFraction": p.Backlog.MaxUnstableSeedFrac}
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
	return nil
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
