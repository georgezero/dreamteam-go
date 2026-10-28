package bench

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/georgezero/dreamteam-go/internal/agent"
)

// ConservativeV2 is the financially conservative baseline, version 2
// (V0.6c; docs/BENCHMARK_SUITE.md §7.2). Version 1 (Conservative) is kept
// unchanged for reproduction: its signal, the oldest unfinished study at
// the decision point, never showed a sustained problem in the suite.
//
// Version 2 watches reporting breaches instead: the public log's "report
// overdue" warnings, which the simulator emits when an urgent or soon
// study passes its target. It keeps, per decision point, the urgent and
// soon breaches logged since the previous one, and at each decision point,
// in order:
//
//   - history: nothing until it has 7 daily records (a week);
//   - wait: nothing while any allocation change is pending, or within 14
//     days of the effective date of its own last change (a raise is
//     reviewed when that period ends, and no other raise is made before);
//   - review: 14 days after a raise took effect, it compares the breaches
//     per day of those 14 days with the (up to) 14 days before the raise
//     was decided. If they did not fall by at least a quarter, the raise
//     did not help: it steps that raise back and raises no more in the run;
//   - step back: an allocation is above its scenario level and either the
//     trailing-30-day operating profit is below zero or the last 7 days
//     had no urgent breach and soon breaches on at most one day;
//   - raise: a sustained problem (urgent breaches on at least 3 of the
//     last 7 days, or soon breaches on at least 5 of them), no ineffective
//     raise yet, cash at least $50,000, trailing-30-day operating profit
//     at least zero, and less than 10 points in effect above the scenario
//     levels in all. The radiologist with the lowest allocation in effect
//     and room to grow gains 5 points from the earliest lawful date, but
//     only if the trailing-30-day profit less the preview's added salary
//     per 30 days stays at or above zero;
//   - otherwise nothing.
//
// Every change is checked with validate and the staffing preview first and
// submitted only if valid, accepted and displacing no booked patient. At
// most one change per decision point. The thresholds were fixed before the
// policy ran on any development seed (§7.2 gives the reasons).
type ConservativeV2 struct {
	days     []breachDay
	lastSeq  uint64
	started  bool
	settle   string // no change before this date
	review   *raiseReview
	futile   bool   // a raise did not help: no more raises
	undoTo   *undo  // the raise to step back after a failed review
	lastDate string // the last decision point's date
}

// breachDay is one decision point's record: the urgent and soon breaches
// logged since the previous one; Partial when the log returned may not
// reach back that far (a lower bound).
type breachDay struct {
	Date         string
	Urgent, Soon int
	Partial      bool
}

// raiseReview is a raise awaiting its review: when it was decided, when it
// took effect, and the breaches and recorded days in the 14 days before the
// decision.
type raiseReview struct {
	Decided, Effective string
	Before, BeforeDays int
	Employee           string
	FromBP, ToBP       int64
}

type undo struct {
	Employee string
	BP       int64
}

func (*ConservativeV2) Name() string { return "conservative-v2" }

const (
	c2History      = 7
	c2UrgentDays   = 3
	c2SoonDays     = 5
	c2ClearSoon    = 1
	c2ReviewDays   = 14
	c2MinReduction = 25 // percent
	c2StepBP       = 500
	c2CapBP        = 1000
	c2SettleDays   = 14
	c2CashFloor    = 5_000_000 // cents
	c2LogEntries   = 2000
)

// logEntry is a public log entry as observe returns it.
type logEntry struct {
	Seq      uint64 `json:"seq"`
	Time     string `json:"time"`
	Severity string `json:"severity"`
	Category string `json:"category"`
	Text     string `json:"text"`
}

// breach reports whether a log entry is an urgent or soon breach, and
// which: "S01234 soon report overdue: …" (sim.tatBreach).
func breach(e logEntry) (urgent, soon bool) {
	if e.Category != "reading" || !strings.Contains(e.Text, " report overdue: ") {
		return false, false
	}
	f := strings.Fields(e.Text)
	if len(f) < 2 {
		return false, false
	}
	return f[1] == "urgent", f[1] == "soon"
}

// record counts the breaches since the previous decision point (at the
// first, those of the last 24 hours) and keeps the day.
func (p *ConservativeV2) record(date, now string, log []logEntry) {
	d := breachDay{Date: date}
	from := p.lastSeq
	if !p.started && len(log) > 0 {
		from = 0
	}
	since := ""
	if !p.started {
		if t, err := time.Parse("2006-01-02T15:04:05", now); err == nil {
			since = t.Add(-24 * time.Hour).Format("2006-01-02T15:04:05")
		}
	}
	for _, e := range log {
		if e.Seq <= from || (since != "" && e.Time < since) {
			continue
		}
		u, s := breach(e)
		if u {
			d.Urgent++
		}
		if s {
			d.Soon++
		}
	}
	if len(log) > 0 {
		if p.started && len(log) >= c2LogEntries && log[0].Seq > p.lastSeq+1 {
			d.Partial = true
		}
		p.lastSeq = log[len(log)-1].Seq
	}
	p.started = true
	p.days = append(p.days, d)
	p.lastDate = date
}

// week summarizes the last 7 records: days with an urgent breach, days
// with a soon breach, and all breaches.
func (p *ConservativeV2) week() (urgentDays, soonDays, total int) {
	for _, d := range p.days[max(0, len(p.days)-c2History):] {
		if d.Urgent > 0 {
			urgentDays++
		}
		if d.Soon > 0 {
			soonDays++
		}
		total += d.Urgent + d.Soon
	}
	return
}

// sustained: urgent breaches on at least 3 of the last 7 days, or soon
// breaches on at least 5.
func (p *ConservativeV2) sustained() bool {
	u, s, _ := p.week()
	return len(p.days) >= c2History && (u >= c2UrgentDays || s >= c2SoonDays)
}

// clear: the last 7 days had no urgent breach and soon breaches on at
// most one day.
func (p *ConservativeV2) clear() bool {
	u, s, _ := p.week()
	return len(p.days) >= c2History && u == 0 && s <= c2ClearSoon
}

// breachesBetween counts urgent and soon breaches, and the records, dated
// from (inclusive) to to (exclusive).
func (p *ConservativeV2) breachesBetween(from, to string) (n, days int) {
	for _, d := range p.days {
		if d.Date >= from && d.Date < to {
			n += d.Urgent + d.Soon
			days++
		}
	}
	return n, days
}

func addDays(date string, n int) string {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date
	}
	return t.AddDate(0, 0, n).Format("2006-01-02")
}

// reviewDue reports whether the pending review can be made today and, if
// so, whether the raise helped: breaches per recorded day in the 14 days
// from its effective date fell by at least a quarter against the (up to)
// 14 days before its decision.
func (p *ConservativeV2) reviewDue(today string) (due, helped bool) {
	r := p.review
	if r == nil || today < addDays(r.Effective, c2ReviewDays) {
		return false, false
	}
	after, days := p.breachesBetween(r.Effective, addDays(r.Effective, c2ReviewDays))
	if days == 0 || r.BeforeDays == 0 {
		return true, false
	}
	return true, after*r.BeforeDays*100 <= r.Before*days*(100-c2MinReduction)
}

func (p *ConservativeV2) Decide(c *agent.Client) ([][]json.RawMessage, error) {
	var obs struct {
		Observation struct {
			Date      string     `json:"date"`
			Time      string     `json:"time"`
			RecentLog []logEntry `json:"recentLog"`
			Finance   struct {
				CashCents  int64 `json:"cashCents"`
				Trailing30 struct {
					OperatingProfitCents int64 `json:"operatingProfitCents"`
				} `json:"trailing30"`
			} `json:"finance"`
		} `json:"observation"`
	}
	if err := c.Call("observe", map[string]any{"sections": []string{"finance", "recentLog"}, "compact": true,
		"recentLog": c2LogEntries}, &obs); err != nil {
		return nil, err
	}
	o := obs.Observation
	p.record(o.Date, o.Time, o.RecentLog)
	profit, cash := o.Finance.Trailing30.OperatingProfitCents, o.Finance.CashCents

	if due, helped := p.reviewDue(o.Date); due {
		if !helped {
			p.futile = true
			p.undoTo = &undo{p.review.Employee, p.review.FromBP}
		}
		p.review = nil
	}
	if len(p.days) < c2History || (p.settle != "" && o.Date < p.settle) {
		return nil, nil
	}
	retreat := profit < 0 || p.clear() || p.undoTo != nil
	raise := !p.futile && p.review == nil && p.sustained() && cash >= c2CashFloor && profit >= 0
	if !retreat && !raise {
		return nil, nil
	}

	ts, err := allocationTargets(c)
	if err != nil {
		return nil, err
	}
	above := int64(0)
	for _, t := range ts {
		if t.Pending {
			return nil, nil
		}
		above += max(0, t.InEffect-t.Scenario)
	}
	best, bp := -1, int64(0)
	switch {
	case p.undoTo != nil:
		for i, t := range ts {
			if t.Employee == p.undoTo.Employee && t.Editable && t.InEffect > p.undoTo.BP {
				best, bp = i, max(t.Scenario, p.undoTo.BP)
			}
		}
		if best < 0 {
			p.undoTo = nil // nothing left to undo
		}
	case retreat && above > 0:
		for i, t := range ts {
			if t.Editable && t.InEffect > t.Scenario && (best < 0 || t.InEffect-t.Scenario > ts[best].InEffect-ts[best].Scenario) {
				best = i
			}
		}
		if best >= 0 {
			bp = max(ts[best].Scenario, ts[best].InEffect-c2StepBP)
		}
	case raise && !retreat && above+c2StepBP <= c2CapBP:
		for i, t := range ts {
			if t.Editable && t.InEffect < t.Max && (best < 0 || t.InEffect < ts[best].InEffect) {
				best = i
			}
		}
		if best >= 0 {
			bp = min(ts[best].Max, ts[best].InEffect+c2StepBP)
		}
	}
	if best < 0 {
		return nil, nil
	}
	t := ts[best]
	batch, added, ok, err := checkAllocation(c, t, bp)
	if err != nil || !ok {
		return nil, err
	}
	if bp > t.InEffect {
		if !affordable(profit, added) {
			return nil, nil
		}
		before, days := p.breachesBetween(addDays(o.Date, -c2ReviewDays), o.Date)
		p.review = &raiseReview{Decided: o.Date, Effective: t.Earliest, Employee: t.Employee, FromBP: t.InEffect, ToBP: bp,
			Before: before, BeforeDays: days}
	} else {
		if p.undoTo != nil && t.Employee == p.undoTo.Employee {
			p.undoTo = nil
		}
		if p.review != nil && t.Employee == p.review.Employee {
			p.review = nil // the raise under review is being stepped back
		}
	}
	p.settle = addDays(t.Earliest, c2SettleDays)
	return [][]json.RawMessage{batch}, nil
}

// allocationTargets reads the catalog's set_reading_allocation targets.
func allocationTargets(c *agent.Client) ([]consTarget, error) {
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
	return ts, nil
}

// checkAllocation builds a change of t to bp and checks it: valid, the
// preview accepting it without displacing a booked patient. It returns
// the batch and the preview's added salary per 30 days.
func checkAllocation(c *agent.Client, t consTarget, bp int64) ([]json.RawMessage, int64, bool, error) {
	act, err := json.Marshal(map[string]any{"type": "set_reading_allocation", "op": "set", "employee": t.Employee,
		"effective": t.Earliest, "siteAllocationBP": bp})
	if err != nil {
		return nil, 0, false, err
	}
	batch := []json.RawMessage{act}
	var v struct {
		Valid bool `json:"valid"`
	}
	if err := c.Call("validate", map[string]any{"actions": batch}, &v); err != nil {
		return nil, 0, false, err
	}
	if !v.Valid {
		return nil, 0, false, nil
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
		return nil, 0, false, err
	}
	if !pv.Preview.OK || len(pv.Preview.Displacements) > 0 || len(pv.Preview.Allocations) == 0 {
		return nil, 0, false, nil
	}
	added := int64(0)
	for _, x := range pv.Preview.Allocations {
		added += x.AfterSalary30Cents - x.BeforeSalary30Cents
	}
	return batch, added, true, nil
}
