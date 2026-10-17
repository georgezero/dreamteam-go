// Package eval is the frozen V0.2 evaluation (docs/evaluation, V0.2_DESIGN
// §13.4): it turns per-seed measurement records of a plan and its paired
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
)

const PolicySchema = "dreamteam.evalpolicy/v1"

// Priorities in report order.
var Priorities = []string{"urgent", "soon", "routine"}

// Policy is the frozen evaluation policy. Every field is used; the file
// is decoded strictly, so a misspelt or extra field is an error.
type Policy struct {
	Schema   string   `json:"schema"`
	ID       string   `json:"id"`
	Status   string   `json:"status"`
	Notes    []string `json:"notes"`
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
		Tail           struct {
			MinDaysAfterPrimary int `json:"minDaysAfterPrimary"`
			DaysAfterEffect     int `json:"daysAfterLatestEffect"`
		} `json:"tail"`
		// ClosureMin: how long after a window's end a cohort is observed so
		// every study's breach is decided (the longest TAT target).
		ClosureMin int64 `json:"closureMin"`
	} `json:"horizons"`
	Service struct {
		BreachGate map[string]float64 `json:"breachGate"`
		// Gated windows (hard), validation windows (reported, and part of
		// the GO condition) and informational windows.
		Gated      []string `json:"gatedWindows"`
		Validation []string `json:"validationWindows"`
		Info       []string `json:"infoWindows"`
		// MinCohort: acquisitions per priority a seed's window needs before
		// its breach rate is evidence (fewer: insufficient, never a pass).
		MinCohort map[string]int64 `json:"minCohort"`
		// MaxFailingSeedFraction: seeds that fail or lack evidence, at most
		// floor(fraction × seeds); the pooled rate must pass as well.
		MaxFailingSeedFraction float64 `json:"maxFailingSeedFraction"`
	} `json:"service"`
	CaseSelection struct {
		MinStratumDue        int64   `json:"minStratumDue"`
		PerSeedMaxServedDrop float64 `json:"perSeedMaxServedDrop"`
		PooledMaxServedDrop  float64 `json:"pooledMaxServedDrop"`
		MaxAcquiredDrop      float64 `json:"maxAcquiredDrop"`
		MaxLossRateIncrease  float64 `json:"maxLossRateIncrease"`
	} `json:"caseSelection"`
	Backlog struct {
		BlockDays             int     `json:"blockDays"`
		RotaCycleDays         int     `json:"rotaCycleDays"`
		GrowthFraction        float64 `json:"growthFraction"`
		GrowthStudies         float64 `json:"growthStudies"`
		MaxUnstableSeedFrac   float64 `json:"maxUnstableSeedFraction"`
		LegacyRuleDescription string  `json:"legacyRule"`
	} `json:"backlog"`
	Finance struct {
		MinCashFloorCents int64 `json:"minCashFloorCents"`
		MaxInsolventSeeds int   `json:"maxInsolventSeeds"`
		MaxPastDueCents   int64 `json:"maxPastDueCents"`
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
	case p.Schema != PolicySchema:
		return fmt.Errorf("schema %q, want %q", p.Schema, PolicySchema)
	case p.Horizons.PrimaryDays <= p.Horizons.FeedbackDays || p.Horizons.OperatingFrom >= p.Horizons.PrimaryDays:
		return fmt.Errorf("horizons out of order")
	case p.Horizons.ClosureMin <= 0:
		return fmt.Errorf("closureMin must be positive")
	case p.Backlog.BlockDays <= 0 || p.Backlog.RotaCycleDays <= 0 || p.Backlog.BlockDays%p.Backlog.RotaCycleDays != 0:
		return fmt.Errorf("backlog blocks must be whole rota cycles")
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
	}
	for _, w := range append(append(append([]string{}, p.Service.Gated...), p.Service.Validation...), p.Service.Info...) {
		if !knownWindow(w) {
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
)

func knownWindow(w string) bool {
	switch w {
	case WinT30, WinT90, WinOps, WinTail, WinT180:
		return true
	}
	return false
}

// failLimit is the most seeds of n allowed to fail at fraction f.
func failLimit(f float64, n int) int { return int(f*float64(n) + 1e-9) }
