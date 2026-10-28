package bench

import (
	"encoding/json"
	"time"

	"github.com/georgezero/dreamteam-go/internal/agent"
)

// Conservative is the financially conservative baseline (version 1,
// V0.6b; docs/BENCHMARK_SUITE.md §7). It spends only on a sustained
// reporting problem, only when the practice can afford it, and steps back
// when money runs short or the problem has cleared. It reads the public
// observation (worklist and finance, compact), the action catalog and the
// staffing preview; its one lever is the site reading allocation. At each
// decision point, in order:
//
//   - wait: any radiologist's allocation change is pending, or the last
//     change this policy made took effect less than 14 days ago (the
//     allocation's own notice period, so a change is seen working before
//     the next);
//   - step back: an allocation is above its scenario level and either the
//     trailing-30-day operating profit is below zero or the oldest
//     unfinished study has been at most 8 hours old at 7 decision points
//     in a row. The radiologist furthest above it loses 5 points (never
//     below the scenario level);
//   - raise: the oldest unfinished study has been more than 24 hours old at
//     3 decision points in a row, no more than 5 points are in effect
//     above the scenario levels in all, cash is at least $50,000, and the
//     trailing-30-day operating profit stays at or above zero after the
//     change's added salary per 30 days (from the preview). The
//     radiologist with the lowest allocation in effect and room to grow
//     gains 5 points from the earliest lawful date;
//   - otherwise nothing.
//
// A change is submitted only if validate finds it valid and the preview
// accepts it without displacing a booked patient. At most one change per
// decision point. The thresholds were fixed before the policy was run on
// any scenario: 24 and 8 hours and 5 points are the rules policy's; 3
// decision points is three days in a row, so a weekend without reading
// alone is not enough; 7 is a week; 14 days is the allocation notice; the
// $50,000 floor and the 5-point cap are round numbers.
type Conservative struct {
	pressure, clear int    // consecutive decision points over 24 h, at most 8 h
	waitUntil       string // the last change's effective date + 14 days
}

func (*Conservative) Name() string { return "conservative" }

const (
	consPressureMin   = 24 * 60
	consClearMin      = 8 * 60
	consPressureTurns = 3
	consClearTurns    = 7
	consStepBP        = 500
	consCapBP         = 500 // in effect above the scenario levels, in all
	consSettleDays    = 14
	consCashFloor     = 5_000_000 // cents
)

// consTarget is one radiologist's allocation as the catalog shows it.
type consTarget struct {
	Employee                string
	Editable, Pending       bool
	InEffect, Scenario, Max int64 // basis points
	Earliest                string
}

// observe counts the consecutive decision points with the oldest
// unfinished study over 24 hours and at most 8 hours old.
func (p *Conservative) observe(oldestMin int64) {
	if oldestMin > consPressureMin {
		p.pressure++
	} else {
		p.pressure = 0
	}
	if oldestMin <= consClearMin {
		p.clear++
	} else {
		p.clear = 0
	}
}

// wanted reports whether, from the observation alone, the policy may step
// back or raise today (before it looks at the catalog).
func (p *Conservative) wanted(date string, profit, cash int64) (retreat, raise bool) {
	if p.waitUntil != "" && date < p.waitUntil {
		return false, false
	}
	return profit < 0 || p.clear >= consClearTurns, p.pressure >= consPressureTurns && cash >= consCashFloor && profit >= 0
}

// choose picks the change, if any: the target's index and its new
// allocation. A pending change anywhere means wait.
func choose(retreat, raise bool, ts []consTarget) (int, int64) {
	above := int64(0)
	for _, t := range ts {
		if t.Pending {
			return -1, 0
		}
		above += max(0, t.InEffect-t.Scenario)
	}
	best := -1
	if retreat && above > 0 {
		for i, t := range ts {
			if t.Editable && t.InEffect > t.Scenario && (best < 0 || t.InEffect-t.Scenario > ts[best].InEffect-ts[best].Scenario) {
				best = i
			}
		}
		if best >= 0 {
			return best, max(ts[best].Scenario, ts[best].InEffect-consStepBP)
		}
		return -1, 0
	}
	if raise && above+consStepBP <= consCapBP {
		for i, t := range ts {
			if t.Editable && t.InEffect < t.Max && (best < 0 || t.InEffect < ts[best].InEffect) {
				best = i
			}
		}
		if best >= 0 {
			return best, min(ts[best].Max, ts[best].InEffect+consStepBP)
		}
	}
	return -1, 0
}

// affordable is the guardrail on a raise's own cost: the trailing 30
// days' operating profit less the added salary per 30 days stays at or
// above zero.
func affordable(profit, added int64) bool { return profit-added >= 0 }

func (p *Conservative) Decide(c *agent.Client) ([][]json.RawMessage, error) {
	var obs struct {
		Observation struct {
			Date     string `json:"date"`
			Worklist struct {
				OldestMin int64 `json:"oldestMin"`
			} `json:"worklist"`
			Finance struct {
				CashCents  int64 `json:"cashCents"`
				Trailing30 struct {
					OperatingProfitCents int64 `json:"operatingProfitCents"`
				} `json:"trailing30"`
			} `json:"finance"`
		} `json:"observation"`
	}
	if err := c.Call("observe", map[string]any{"sections": []string{"worklist", "finance"}, "compact": true, "recentLog": 0}, &obs); err != nil {
		return nil, err
	}
	o := obs.Observation
	p.observe(o.Worklist.OldestMin)
	profit := o.Finance.Trailing30.OperatingProfitCents
	retreat, raise := p.wanted(o.Date, profit, o.Finance.CashCents)
	if !retreat && !raise {
		return nil, nil
	}

	var cat struct {
		Types []struct {
			Type    string          `json:"type"`
			Targets json.RawMessage `json:"targets"`
		} `json:"types"`
	}
	if err := c.Call("actions", nil, &cat); err != nil {
		return nil, err
	}
	var raw []struct {
		Employee          string `json:"employee"`
		InEffectBP        int64  `json:"inEffectBP"`
		Editable          bool   `json:"editable"`
		EarliestEffective string `json:"earliestEffective"`
		MaxBP             int64  `json:"maxBP"`
		Versions          []struct {
			Effective string `json:"effective"`
			SiteBP    int64  `json:"siteBP"`
			Pending   bool   `json:"pending"`
		} `json:"versions"`
	}
	for _, t := range cat.Types {
		if t.Type == "set_reading_allocation" {
			if err := json.Unmarshal(t.Targets, &raw); err != nil {
				return nil, err
			}
		}
	}
	ts := make([]consTarget, len(raw))
	for i, a := range raw {
		ts[i] = consTarget{Employee: a.Employee, Editable: a.Editable, InEffect: a.InEffectBP, Scenario: a.InEffectBP,
			Max: a.MaxBP, Earliest: a.EarliestEffective}
		for _, v := range a.Versions {
			ts[i].Pending = ts[i].Pending || v.Pending
			if v.Effective == "" {
				ts[i].Scenario = v.SiteBP
			}
		}
	}
	best, bp := choose(retreat, raise, ts)
	if best < 0 {
		return nil, nil
	}
	t := ts[best]
	act, err := json.Marshal(map[string]any{"type": "set_reading_allocation", "op": "set", "employee": t.Employee,
		"effective": t.Earliest, "siteAllocationBP": bp})
	if err != nil {
		return nil, err
	}
	batch := []json.RawMessage{act}
	var v struct {
		Valid bool `json:"valid"`
	}
	if err := c.Call("validate", map[string]any{"actions": batch}, &v); err != nil {
		return nil, err
	}
	var pv struct {
		Preview struct {
			OK            bool              `json:"ok"`
			Displacements []json.RawMessage `json:"displacements"`
			Allocations   []struct {
				BeforeSalary30Cents int64 `json:"beforeSalary30Cents"`
				AfterSalary30Cents  int64 `json:"afterSalary30Cents"`
			} `json:"allocations"`
		} `json:"preview"`
	}
	if err := c.Call("preview", map[string]any{"actions": batch}, &pv); err != nil {
		return nil, err
	}
	if !v.Valid || !pv.Preview.OK || len(pv.Preview.Displacements) > 0 {
		return nil, nil
	}
	if bp > t.InEffect {
		added := int64(0)
		for _, x := range pv.Preview.Allocations {
			added += x.AfterSalary30Cents - x.BeforeSalary30Cents
		}
		if len(pv.Preview.Allocations) == 0 || !affordable(profit, added) {
			return nil, nil
		}
		p.pressure = 0
	} else {
		p.clear = 0
	}
	eff, err := time.Parse("2006-01-02", t.Earliest)
	if err != nil {
		return nil, err
	}
	p.waitUntil = eff.AddDate(0, 0, consSettleDays).Format("2006-01-02")
	return [][]json.RawMessage{batch}, nil
}
