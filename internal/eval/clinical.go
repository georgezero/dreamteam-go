package eval

import (
	"fmt"
	"github.com/georgezero/dreamteam-go/internal/simtime"
	"maps"
	"slices"
)

// v0.2-eval-5's clinical acceptance (Slice 5C.4b; docs/evaluation/
// proposals/v5-clinical-acceptance.md): severe reporting delay and timely
// access from exact record timestamps, absolute limits at zero tolerance,
// and one shared seed allowance over every seed-level clinical gate. Only
// a policy with clinical rules (schema v5) reaches this file.

// DelayCohort is one window's studies of one priority (the breach
// cohort: acquired in the window, finished or not), by their exact delay
// past due. A study unfinished at the end of observation is censored
// there: its delay is a lower bound that kept growing, never dropped.
type DelayCohort struct {
	Acquired int64 `json:"acquired"`
	Severe   int64 `json:"severe"` // more than severeAfterDueMin past due
	// CensoredSevere: of the severe, unfinished at the end of observation.
	CensoredSevere int64 `json:"censoredSevere"`
	// Undecided: unfinished, and not yet severe when observation ended.
	Undecided     int64 `json:"undecided"`
	SevereRate    Ratio `json:"severeRate"` // undefined if empty or undecided
	MaxPastDueMin int64 `json:"maxPastDueMin"`
}

// AccessCohort is one window's orders of one priority wanted in it,
// classified against the access limit from each order's original wanted
// time (a rebooking never resets it): timely, imaged within the limit;
// untimely, not imaged by then for any reason (still open, lost, left, or
// imaged later); withdrawn, ended by the patient before the limit
// without being imaged (out of the denominator); undecided, open and
// within the limit when observation ended.
type AccessCohort struct {
	Wanted    int64 `json:"wanted"`
	Timely    int64 `json:"timely"`
	Untimely  int64 `json:"untimely"`
	Withdrawn int64 `json:"withdrawn"`
	Undecided int64 `json:"undecided"`
	// Lost: lost and left orders, apart from the timeliness count when
	// the policy gates lost demand against the baseline (lostDemand).
	Lost           int64 `json:"lost,omitempty"`
	TimelyFraction Ratio `json:"timelyFraction"` // timely ÷ (timely + untimely); undefined if undecided
	// MaxWaitMin: the longest wait of a served or open order (open: to the
	// end of observation); WaitMin, their distribution.
	MaxWaitMin int64 `json:"maxWaitMin"`
	WaitMin    Dist  `json:"waitMin"`

	waits []int64 // minutes, served and open
}

// AbsoluteCount is one absolute limit on one seed: its violations and the
// worst value, violated or not.
type AbsoluteCount struct {
	Limit      string `json:"limit"` // pastDue or wait
	Priority   string `json:"priority"`
	LimitMin   int64  `json:"limitMin"`
	Violations int64  `json:"violations"`
	WorstMin   int64  `json:"worstMin"`
}

// ClinicalMeasures are one seed's clinical measures.
type ClinicalMeasures struct {
	Delay  map[string]map[string]*DelayCohort  `json:"delay"`  // window → priority
	Access map[string]map[string]*AccessCohort `json:"access"` // window → priority, new orders
	// InheritedAccess: orders open at the takeover and wanted by the
	// horizon, timed by inheritedFrom; subject to the maximum wait, never
	// to a rate (they are not new demand).
	InheritedAccess map[string]*AccessCohort `json:"inheritedAccess"`
	// WorstWeekSevere: the most severely delayed studies of a priority
	// acquired in one Monday-based week of the gated period, and that
	// week's first day (days after the takeover's date; partial weeks
	// count as they are).
	WorstWeekSevere   map[string]int64 `json:"worstWeekSevere"`
	WorstWeekFirstDay map[string]int64 `json:"worstWeekFirstDay"`
	// MaxWaitGatedMin: the longest wait of a served or open order wanted in
	// the gated period, new or inherited (the maximum-wait limit's value).
	MaxWaitGatedMin map[string]int64 `json:"maxWaitGatedMin"`
	Absolute        []AbsoluteCount  `json:"absolute"`
	// WeeklySevere: the volume-normalized weekly severity
	// (maxWeeklySevereRate), by priority.
	WeeklySevere map[string]*WeeklySevere `json:"weeklySevere,omitempty"`
}

// WeeklySevere is a priority's weekly severity: the gated period's Monday
// weeks joined in order into blocks of at least minWeeklyCohort
// acquisitions (a short remainder joins the last block), each block's
// severe-delay rate, the worst block, and how many exceed the rate. The
// worst block's span is given in calendar weeks and dates (audit F9: a
// block over sparse weeks spans more calendar weeks than it has weeks
// with acquisitions). Backward is the same weeks pooled from the end (a
// short remainder joining the first block): another valid partition,
// reported only; PartitionSensitive when the two disagree on whether any
// block exceeds the rate.
type WeeklySevere struct {
	Acquired           int64  `json:"acquired"`
	Severe             int64  `json:"severe"`
	Blocks             int    `json:"blocks"`
	Failing            int    `json:"failingBlocks"`
	Insufficient       bool   `json:"insufficient"` // fewer acquired in the period than the minimum
	WorstFirstDay      int64  `json:"worstFirstDay"`
	WorstFrom          string `json:"worstFrom"` // the worst block's first Monday
	WorstTo            string `json:"worstTo"`   // its last Sunday
	WorstWeeks         int    `json:"worstWeeks"`
	WorstOccupiedWeeks int    `json:"worstOccupiedWeeks"`
	WorstAcquired      int64  `json:"worstAcquired"`
	WorstSevere        int64  `json:"worstSevere"`
	BackwardFailing    int    `json:"backwardFailingBlocks"`
	PartitionSensitive bool   `json:"partitionSensitive"`
	// The calendar partition (weeklyPartition "calendar"): every Monday–
	// Sunday week of the gated period, and how many are partial, below the
	// minimum (insufficient when complete, not judged when partial) and
	// failing. Blocks, Failing and the worst block are then weeks.
	Partition         string       `json:"partition,omitempty"`
	Weeks             []WeekSevere `json:"weeks,omitempty"`
	PartialWeeks      int          `json:"partialWeeks,omitempty"`
	InsufficientWeeks int          `json:"insufficientWeeks,omitempty"`
	NotJudgedWeeks    int          `json:"notJudgedWeeks,omitempty"`
}

// WeekSevere is one calendar week of a priority's weekly severity.
// Complete: the whole week, Monday 00:00 to Sunday 24:00, lies within the
// gated period. Status: pass or fail against the rate when the week has
// at least the minimum cohort; below it, insufficient when complete (the
// seed then lacks evidence) and not_judged when partial (its shortfall is
// the evaluation window's edge, the same for every plan; reported, never
// a pass).
type WeekSevere struct {
	From     string  `json:"from"` // Monday
	To       string  `json:"to"`   // Sunday
	FirstDay int64   `json:"firstDay"`
	Complete bool    `json:"complete"`
	Acquired int64   `json:"acquired"`
	Severe   int64   `json:"severe"`
	Rate     float64 `json:"rate"`
	Status   string  `json:"status"`
}

// NotJudged: a partial calendar week below the minimum cohort.
const NotJudged = "not_judged"

// delayAt is a study's delay past due at the end of observation end:
// finished, at its finish; unfinished, at end (a lower bound). from is
// when its delay starts to count (its due time, or the takeover for
// inherited work timed from it).
func delayAt(s *Study, from, end int64) (delay int64, censored bool) {
	if s.FinalSec != 0 {
		return max(0, s.FinalSec-from), false
	}
	return max(0, end-from), true
}

// severe reports whether a delay past due (seconds) is severe for
// priority q: more than severeAfterDueMin.
func (c *Clinical) severe(q string, delay int64) bool { return delay > c.SevereAfterDueMin[q]*60 }

func measureClinical(p *Policy, r *Record, ws map[string][2]int64, end *Capture) *ClinicalMeasures {
	c := p.Clinical
	m := &ClinicalMeasures{Delay: map[string]map[string]*DelayCohort{}, Access: map[string]map[string]*AccessCohort{},
		InheritedAccess: map[string]*AccessCohort{}, WorstWeekSevere: map[string]int64{}, WorstWeekFirstDay: map[string]int64{},
		MaxWaitGatedMin: map[string]int64{}}
	E, H, t0 := r.ObservedUntilSec, r.HorizonEndSec, r.TakeoverSec
	takeover := c.InheritedFrom == "takeover"
	for _, w := range windowOrder(p) {
		b := ws[w]
		m.Delay[w] = map[string]*DelayCohort{}
		m.Access[w] = map[string]*AccessCohort{}
		for _, q := range Priorities {
			m.Delay[w][q] = &DelayCohort{}
			m.Access[w][q] = &AccessCohort{}
		}
		for i := range r.Studies {
			s := &r.Studies[i]
			if s.AcquiredSec <= b[0] || s.AcquiredSec > b[1] {
				continue
			}
			d := m.Delay[w][s.Priority]
			d.Acquired++
			l, cens := delayAt(s, s.AcquiredSec+target(r, s.Priority), E)
			d.MaxPastDueMin = max(d.MaxPastDueMin, l/60)
			switch {
			case c.severe(s.Priority, l):
				d.Severe++
				if cens {
					d.CensoredSevere++
				}
			case cens:
				d.Undecided++
			}
		}
		for _, q := range Priorities {
			d := m.Delay[w][q]
			d.SevereRate = ratio(d.Severe, d.Acquired)
			if d.Undecided > 0 {
				d.SevereRate = Ratio{Num: d.Severe, Den: d.Acquired}
			}
		}
		for i := range r.Orders {
			o := &r.Orders[i]
			if o.ReceivedSec <= t0 || o.WantedSec <= b[0] || o.WantedSec > b[1] {
				continue
			}
			c.classify(m.Access[w][o.Priority], o, end.OrderOutcomes[i], o.WantedSec, E)
		}
		for _, q := range Priorities {
			a := m.Access[w][q]
			a.WaitMin = dist(a.waits)
			a.TimelyFraction = ratio(a.Timely, a.Timely+a.Untimely)
			if a.Undecided > 0 {
				a.TimelyFraction = Ratio{Num: a.Timely, Den: a.Timely + a.Untimely}
			}
		}
	}
	// Inherited orders, and the gated period's longest wait.
	for _, q := range Priorities {
		m.InheritedAccess[q] = &AccessCohort{}
		m.MaxWaitGatedMin[q] = m.Access[WinGated][q].MaxWaitMin
	}
	for i := range r.Orders {
		o := &r.Orders[i]
		if o.ReceivedSec > t0 || o.WantedSec > H {
			continue
		}
		from := o.WantedSec
		if takeover {
			from = max(from, t0)
		}
		a := m.InheritedAccess[o.Priority]
		c.classify(a, o, end.OrderOutcomes[i], from, E)
		m.MaxWaitGatedMin[o.Priority] = max(m.MaxWaitGatedMin[o.Priority], a.MaxWaitMin)
	}
	for _, q := range Priorities {
		a := m.InheritedAccess[q]
		a.WaitMin = dist(a.waits)
		a.TimelyFraction = ratio(a.Timely, a.Timely+a.Untimely)
	}
	// Severe delays concentrated in a week of the gated period.
	weeks := map[string]map[int64]int64{}
	for i := range r.Studies {
		s := &r.Studies[i]
		if s.AcquiredSec <= t0 || s.AcquiredSec > H {
			continue
		}
		if l, _ := delayAt(s, s.AcquiredSec+target(r, s.Priority), E); c.severe(s.Priority, l) {
			if weeks[s.Priority] == nil {
				weeks[s.Priority] = map[int64]int64{}
			}
			weeks[s.Priority][floorDiv(s.AcquiredSec/daySec+3, 7)]++
		}
	}
	switch {
	case c.MaxWeeklySevereRate != nil && c.WeeklyPartition == PartitionCalendar:
		m.WeeklySevere = weeklySevereCalendar(c, r, E)
	case c.MaxWeeklySevereRate != nil:
		m.WeeklySevere = weeklySevere(c, r, E)
	}
	for _, q := range Priorities {
		for _, k := range slices.Sorted(maps.Keys(weeks[q])) {
			if n := weeks[q][k]; n > m.WorstWeekSevere[q] {
				// Week k's Monday is local day 7k − 3 (stats.go clusterEffects).
				m.WorstWeekSevere[q], m.WorstWeekFirstDay[q] = n, 7*k-3-t0/daySec
			}
		}
	}
	// Absolute limits: every study acquired by the horizon (inherited
	// included), and, when absolute, every served or open order's wait.
	for _, q := range Priorities {
		a := AbsoluteCount{Limit: "pastDue", Priority: q, LimitMin: c.AbsoluteMaxPastDueMin[q]}
		for i := range r.Studies {
			s := &r.Studies[i]
			if s.Priority != q || s.AcquiredSec > H {
				continue
			}
			from := s.AcquiredSec + target(r, q)
			if takeover && s.AcquiredSec <= t0 {
				from = max(from, t0)
			}
			l, _ := delayAt(s, from, E)
			a.WorstMin = max(a.WorstMin, l/60)
			if l > a.LimitMin*60 {
				a.Violations++
			}
		}
		m.Absolute = append(m.Absolute, a)
	}
	if c.MaxWaitAbsolute {
		for _, q := range Priorities {
			a := AbsoluteCount{Limit: "wait", Priority: q, LimitMin: c.MaxWaitMin[q], WorstMin: m.MaxWaitGatedMin[q]}
			for _, xs := range [][]int64{m.Access[WinGated][q].waits, m.InheritedAccess[q].waits} {
				for _, x := range xs {
					if x > a.LimitMin {
						a.Violations++
					}
				}
			}
			m.Absolute = append(m.Absolute, a)
		}
	}
	return m
}

// classify adds order o to a under the policy: with lostDemand, a lost or
// left order is counted apart (gated against the baseline instead);
// otherwise classifyAccess.
func (c *Clinical) classify(a *AccessCohort, o *Order, out string, from, E int64) {
	if c.LostDemand != nil && (out == OutLost || out == OutLeft) {
		a.Wanted++
		a.Lost++
		return
	}
	classifyAccess(a, o, out, from, c.AccessLimitMin[o.Priority]*60, E)
}

// weeklySevere measures the volume-normalized weekly severity of every
// priority: studies acquired in the gated period by Monday week, joined
// into blocks of at least minWeeklyCohort acquisitions.
func weeklySevere(c *Clinical, r *Record, E int64) map[string]*WeeklySevere {
	t0, H := r.TakeoverSec, r.HorizonEndSec
	date, _ := simtime.ParseDate(r.TakeoverDate)
	type wk struct {
		k        int64
		acq, sev int64
	}
	weeks := map[string]map[int64]*wk{}
	for i := range r.Studies {
		s := &r.Studies[i]
		if s.AcquiredSec <= t0 || s.AcquiredSec > H {
			continue
		}
		k := floorDiv(s.AcquiredSec/daySec+3, 7)
		if weeks[s.Priority] == nil {
			weeks[s.Priority] = map[int64]*wk{}
		}
		w := weeks[s.Priority][k]
		if w == nil {
			w = &wk{k: k}
			weeks[s.Priority][k] = w
		}
		w.acq++
		if l, _ := delayAt(s, s.AcquiredSec+target(r, s.Priority), E); c.severe(s.Priority, l) {
			w.sev++
		}
	}
	type block struct {
		first, last int64 // weeks
		n           int   // weeks with acquisitions
		acq, sev    int64
	}
	// pool joins weeks (in order) into blocks of at least minCohort; a
	// short remainder joins the block before it.
	pool := func(ws []*wk, minCohort int64) []block {
		var blocks []block
		var cur block
		for _, w := range ws {
			if cur.n == 0 {
				cur.first = w.k
			}
			cur.last, cur.n, cur.acq, cur.sev = w.k, cur.n+1, cur.acq+w.acq, cur.sev+w.sev
			if cur.acq >= minCohort {
				blocks = append(blocks, cur)
				cur = block{}
			}
		}
		switch {
		case cur.n > 0 && len(blocks) > 0:
			last := &blocks[len(blocks)-1]
			last.n += cur.n
			last.acq += cur.acq
			last.sev += cur.sev
			if cur.first < last.first {
				last.first = cur.first
			}
			if cur.last > last.last {
				last.last = cur.last
			}
		case cur.n > 0:
			blocks = append(blocks, cur)
		}
		return blocks
	}
	out := map[string]*WeeklySevere{}
	for _, q := range Priorities {
		ws := &WeeklySevere{}
		out[q] = ws
		var ordered []*wk
		for _, k := range slices.Sorted(maps.Keys(weeks[q])) {
			ordered = append(ordered, weeks[q][k])
			ws.Acquired += weeks[q][k].acq
			ws.Severe += weeks[q][k].sev
		}
		minCohort := c.MinWeeklyCohort[q]
		bpRate := bp(c.MaxWeeklySevereRate[q])
		blocks := pool(ordered, minCohort)
		ws.Insufficient = ws.Acquired < minCohort
		ws.Blocks = len(blocks)
		worst := -1
		for i, b := range blocks {
			if rateAbove(b.sev, b.acq, bpRate) {
				ws.Failing++
			}
			// The worst block: the highest rate (exact, by cross-products).
			if worst < 0 || b.sev*blocks[worst].acq > blocks[worst].sev*b.acq {
				worst = i
			}
		}
		if worst >= 0 {
			b := blocks[worst]
			// Week k's Monday is local day 7k − 3 (clusterEffects).
			ws.WorstFirstDay = 7*b.first - 3 - t0/daySec
			ws.WorstFrom = date.AddDays(ws.WorstFirstDay).String()
			ws.WorstTo = date.AddDays(7*b.last - 3 - t0/daySec + 6).String()
			ws.WorstWeeks, ws.WorstOccupiedWeeks = int(b.last-b.first+1), b.n
			ws.WorstAcquired, ws.WorstSevere = b.acq, b.sev
		}
		// The same weeks pooled from the end.
		rev := slices.Clone(ordered)
		slices.Reverse(rev)
		for _, b := range pool(rev, minCohort) {
			if rateAbove(b.sev, b.acq, bpRate) {
				ws.BackwardFailing++
			}
		}
		ws.PartitionSensitive = !ws.Insufficient && (ws.Failing > 0) != (ws.BackwardFailing > 0)
	}
	return out
}

// weeklySevereCalendar measures the weekly severity of every priority
// over the fixed Monday–Sunday weeks of the gated period (Slice 5C.4e):
// each week judged on its own acquisitions, never pooled.
func weeklySevereCalendar(c *Clinical, r *Record, E int64) map[string]*WeeklySevere {
	t0, H := r.TakeoverSec, r.HorizonEndSec
	date, _ := simtime.ParseDate(r.TakeoverDate)
	// Week k runs from local day 7k − 3 (a Monday) for seven days.
	week := func(sec int64) int64 { return floorDiv(sec/daySec+3, 7) }
	first, last := week(t0+1), week(H)
	out := map[string]*WeeklySevere{}
	for _, q := range Priorities {
		ws := &WeeklySevere{Partition: PartitionCalendar}
		out[q] = ws
		for k := first; k <= last; k++ {
			mon := (7*k - 3) * daySec
			w := WeekSevere{FirstDay: 7*k - 3 - t0/daySec, Complete: mon >= t0 && mon+7*daySec <= H}
			w.From, w.To = date.AddDays(w.FirstDay).String(), date.AddDays(w.FirstDay+6).String()
			ws.Weeks = append(ws.Weeks, w)
		}
	}
	for i := range r.Studies {
		s := &r.Studies[i]
		if s.AcquiredSec <= t0 || s.AcquiredSec > H {
			continue
		}
		w := &out[s.Priority].Weeks[week(s.AcquiredSec)-first]
		w.Acquired++
		if l, _ := delayAt(s, s.AcquiredSec+target(r, s.Priority), E); c.severe(s.Priority, l) {
			w.Severe++
		}
	}
	for _, q := range Priorities {
		ws := out[q]
		bpRate := bp(c.MaxWeeklySevereRate[q])
		worst := -1
		for i := range ws.Weeks {
			w := &ws.Weeks[i]
			ws.Acquired += w.Acquired
			ws.Severe += w.Severe
			if w.Acquired > 0 {
				w.Rate = float64(w.Severe) / float64(w.Acquired)
			}
			if !w.Complete {
				ws.PartialWeeks++
			}
			switch {
			case w.Acquired < c.MinWeeklyCohort[q] && w.Complete:
				w.Status = Insufficient
				ws.InsufficientWeeks++
				continue
			case w.Acquired < c.MinWeeklyCohort[q]:
				w.Status = NotJudged
				ws.NotJudgedWeeks++
				continue
			case rateAbove(w.Severe, w.Acquired, bpRate):
				w.Status = Fail
				ws.Failing++
			default:
				w.Status = Pass
			}
			ws.Blocks++
			// The worst judged week: the highest rate (exact, by
			// cross-products), the earliest of equals.
			if worst < 0 || w.Severe*ws.Weeks[worst].Acquired > ws.Weeks[worst].Severe*w.Acquired {
				worst = i
			}
		}
		ws.Insufficient = ws.InsufficientWeeks > 0
		if worst >= 0 {
			w := ws.Weeks[worst]
			ws.WorstFirstDay, ws.WorstFrom, ws.WorstTo = w.FirstDay, w.From, w.To
			ws.WorstWeeks, ws.WorstOccupiedWeeks = 1, 1
			ws.WorstAcquired, ws.WorstSevere = w.Acquired, w.Severe
		}
	}
	return out
}

// classifyAccess adds order o (outcome out at the end of observation E)
// to a, its wait counted from from against limit (seconds).
func classifyAccess(a *AccessCohort, o *Order, out string, from, limit, E int64) {
	a.Wanted++
	switch out {
	case OutServed:
		wait := max(0, o.ResolvedSec-from)
		if wait <= limit {
			a.Timely++
		} else {
			a.Untimely++
		}
		a.MaxWaitMin = max(a.MaxWaitMin, wait/60)
		a.waits = append(a.waits, wait/60)
	case OutOpen:
		wait := max(0, E-from)
		if wait > limit {
			a.Untimely++
		} else {
			a.Undecided++
		}
		a.MaxWaitMin = max(a.MaxWaitMin, wait/60)
		a.waits = append(a.waits, wait/60)
	case OutPatient:
		at := o.ResolvedSec
		if at == 0 {
			at = E
		}
		if at < from+limit {
			a.Withdrawn++
		} else {
			a.Untimely++
		}
	default: // lost, left: never imaged, the practice's
		a.Untimely++
	}
}

// checkClinicalRecord is v0.2-eval-5's additional record integrity: an
// order cannot be imaged before its wanted time.
//
// Since 5C.4d (audit F3) it also requires every order's and study's
// priority to be a known one and every capture's order outcome to be a
// known outcome, a resolved order's final one terminal (served, lost,
// left or patient): missing or unknown evidence is INVALID, never a
// default. It reports whether the record can be measured clinically.
func checkClinicalRecord(r *Record, caps map[string]*Capture, bad func(string, ...any)) bool {
	ok := true
	for i := range r.Orders {
		if !slices.Contains(Priorities, r.Orders[i].Priority) {
			bad("order %s: unknown priority %q", r.Orders[i].ID, r.Orders[i].Priority)
			ok = false
			break
		}
	}
	for i := range r.Studies {
		if !slices.Contains(Priorities, r.Studies[i].Priority) {
			bad("study %s: unknown priority %q", r.Studies[i].ID, r.Studies[i].Priority)
			ok = false
			break
		}
	}
	known := []string{OutServed, OutLost, OutLeft, OutPatient, OutOpen, OutNotDisclosed}
	terminal := []string{OutServed, OutLost, OutLeft, OutPatient}
	for _, n := range captureNames {
		c := caps[n]
		if c == nil {
			continue
		}
		for i, out := range c.OrderOutcomes {
			if !slices.Contains(known, out) {
				bad("order %s: unknown outcome %q at %s", r.Orders[i].ID, out, n)
				ok = false
				break
			}
		}
	}
	end := caps["end"]
	if end == nil {
		return false
	}
	for i := range r.Orders {
		o := &r.Orders[i]
		out := end.OrderOutcomes[i]
		if o.ResolvedSec != 0 && !slices.Contains(terminal, out) {
			bad("order %s: resolved at %d with no terminal outcome (%q)", o.ID, o.ResolvedSec, out)
			return false
		}
		if out == OutServed && o.ResolvedSec < o.WantedSec {
			bad("order %s: imaged at %d, before its wanted time %d", o.ID, o.ResolvedSec, o.WantedSec)
			return false
		}
	}
	return ok
}

// Clinical gate names.
const (
	GateSevereRate   = "severe_delay_rate"
	GateSevereWeek   = "severe_delays_in_a_week"
	GateTimelyAccess = "timely_access"
	GateMaxWait      = "max_wait"
)

// clinicalGates are a seed's clinical seed-level gates in every window
// it has. Per seed (and so in the shared allowance) exactly where the
// priority's breach gate is decided per seed in a gated window.
func clinicalGates(p *Policy, m *SeedMeasures) []SeedGate {
	c := p.Clinical
	cm := m.Clinical
	var out []SeedGate
	for _, w := range windowOrder(p) {
		for _, q := range Priorities {
			per := slices.Contains(p.Service.PerSeedWindows[q], w)
			sg := SeedGate{Gate: GateSevereRate, Window: w, Stratum: q, Threshold: c.MaxSevereRate[q], PerSeed: per}
			ag := SeedGate{Gate: GateTimelyAccess, Window: w, Stratum: q, Threshold: c.MinTimelyFraction[q], PerSeed: per}
			if cm == nil || cm.Delay[w] == nil {
				sg.Status, sg.Reason, sg.Undecided = Insufficient, "window not measured", true
				ag.Status, ag.Reason, ag.Undecided = Insufficient, "window not measured", true
				out = append(out, sg, ag)
				continue
			}
			d := cm.Delay[w][q]
			sg.Num, sg.Den, sg.Value = d.Severe, d.Acquired, d.SevereRate.Value
			switch {
			case d.Undecided > 0:
				sg.Status, sg.Reason, sg.Undecided = Insufficient, fmt.Sprintf("%d studies not yet severe and unfinished when observation ended", d.Undecided), true
			case d.Acquired < p.Service.MinCohort[q]:
				sg.Status, sg.Reason = Insufficient, fmt.Sprintf("%d acquired, fewer than %d", d.Acquired, p.Service.MinCohort[q])
			case rateAbove(d.Severe, d.Acquired, bp(sg.Threshold)):
				sg.Status = Fail
			default:
				sg.Status = Pass
			}
			a := cm.Access[w][q]
			ag.Num, ag.Den, ag.Value = a.Timely, a.Timely+a.Untimely, a.TimelyFraction.Value
			switch {
			case a.Undecided > 0:
				ag.Status, ag.Reason, ag.Undecided = Insufficient, fmt.Sprintf("%d orders open and within the limit when observation ended", a.Undecided), true
			case ag.Den < c.MinAccessCohort[q]:
				ag.Status, ag.Reason = Insufficient, fmt.Sprintf("%d decided orders, fewer than %d", ag.Den, c.MinAccessCohort[q])
			case rateBelow(a.Timely, ag.Den, bp(ag.Threshold)):
				ag.Status = Fail
			default:
				ag.Status = Pass
			}
			out = append(out, sg, ag)
		}
	}
	for _, q := range Priorities {
		g := SeedGate{Gate: GateSevereWeek, Window: WinGated, Stratum: q, Threshold: float64(c.MaxSeverePerWeek[q]), PerSeed: true, Status: Pass}
		switch {
		case cm == nil:
			g.Status, g.Reason, g.Undecided = Insufficient, "not measured", true
		case c.MaxWeeklySevereRate != nil:
			ws := cm.WeeklySevere[q]
			g.Threshold = c.MaxWeeklySevereRate[q]
			g.Num, g.Den = ws.WorstSevere, ws.WorstAcquired
			if ws.WorstAcquired > 0 {
				g.Value = float64(ws.WorstSevere) / float64(ws.WorstAcquired)
			}
			switch {
			case ws.Partition == PartitionCalendar && ws.Failing > 0:
				g.Status, g.Reason = Fail, fmt.Sprintf("%d of %d judged calendar weeks above the rate; worst %d of %d, %s to %s",
					ws.Failing, ws.Blocks, ws.WorstSevere, ws.WorstAcquired, ws.WorstFrom, ws.WorstTo)
			case ws.Partition == PartitionCalendar && ws.Insufficient:
				g.Status, g.Reason = Insufficient, fmt.Sprintf("%d of %d complete calendar weeks below the minimum cohort of %d",
					ws.InsufficientWeeks, len(ws.Weeks)-ws.PartialWeeks, c.MinWeeklyCohort[q])
			case ws.Insufficient:
				g.Status, g.Reason = Insufficient, fmt.Sprintf("%d acquired in the gated period, fewer than %d", ws.Acquired, c.MinWeeklyCohort[q])
			case ws.Failing > 0:
				g.Status, g.Reason = Fail, fmt.Sprintf("%d of %d weekly blocks above the rate; worst %d of %d in %d weeks from day %d",
					ws.Failing, ws.Blocks, ws.WorstSevere, ws.WorstAcquired, ws.WorstWeeks, ws.WorstFirstDay)
				if ws.WorstFrom != "" {
					g.Reason += fmt.Sprintf(" (%s to %s)", ws.WorstFrom, ws.WorstTo)
				}
			}
		default:
			g.Value, g.Num = float64(cm.WorstWeekSevere[q]), cm.WorstWeekSevere[q]
			if cm.WorstWeekSevere[q] > c.MaxSeverePerWeek[q] {
				g.Status, g.Reason = Fail, fmt.Sprintf("%d in the week from day %d", cm.WorstWeekSevere[q], cm.WorstWeekFirstDay[q])
			}
		}
		out = append(out, g)
		if c.MaxWaitAbsolute || c.MaxWaitReportOnly {
			continue
		}
		g = SeedGate{Gate: GateMaxWait, Window: WinGated, Stratum: q, Threshold: float64(c.MaxWaitMin[q]), PerSeed: true, Status: Pass}
		if cm == nil {
			g.Status, g.Reason, g.Undecided = Insufficient, "not measured", true
		} else {
			g.Value, g.Num = float64(cm.MaxWaitGatedMin[q]), cm.MaxWaitGatedMin[q]
			if cm.MaxWaitGatedMin[q] > c.MaxWaitMin[q] {
				g.Status = Fail
			}
		}
		out = append(out, g)
	}
	return out
}

// clinicalSeed is a seed's shared clinical status: fail if its service,
// any per-seed clinical gate or its backlog fails; else insufficient if
// any of them lacks evidence.
func clinicalSeed(sr *SeedResult) string {
	st := Pass
	note := func(s string) {
		switch {
		case s == Fail:
			st = Fail
		case s == Insufficient && st == Pass:
			st = Insufficient
		}
	}
	note(sr.Service)
	for _, g := range sr.ClinicalGates {
		if g.PerSeed {
			note(g.Status)
		}
	}
	switch sr.Backlog {
	case "stable":
	case "unstable":
		note(Fail)
	default:
		note(Insufficient)
	}
	return st
}

// SafetyViolation is an absolute limit violated on a seed.
type SafetyViolation struct {
	Seed int64 `json:"seed"`
	AbsoluteCount
}

// Safety is the plan's absolute limits: any violation on any seed makes
// the plan NO GO (clinical safety), whatever the seed allowance.
type Safety struct {
	Status     string            `json:"status"`
	Seeds      []int64           `json:"seeds"`
	Violations []SafetyViolation `json:"violations"`
}

// clinicalPlan completes a v0.2-eval-5 plan report: the clinical gates'
// summaries, the shared seed allowance, the absolute limits, and
// eligibility from them in place of the separate service and backlog
// rules (which stay reported).
func clinicalPlan(p *Policy, pr *PlanReport) {
	pr.ClinicalGates = summarizeGates(p, pr.PerSeed, func(s *SeedResult) []SeedGate { return s.ClinicalGates })
	frac := bp(p.Clinical.MaxFailingSeedFraction)
	j := jointOf(p, pr.PerSeed, "clinical", frac, func(s *SeedResult) string { return s.Clinical })
	for _, s := range pr.PerSeed[:min(1, len(pr.PerSeed))] {
		j.Gates = 0
		for _, g := range append(slices.Clone(s.Gates), s.ClinicalGates...) {
			if g.PerSeed && role(p, g.Window) == "gated" {
				j.Gates++
			}
		}
		j.Gates++ // backlog stability
	}
	pr.ClinicalJoint = &j
	pr.ClinicalStatus = j.Status
	for _, gs := range [][]GateSummary{pr.ServiceGates, pr.ClinicalGates} {
		for _, g := range gs {
			switch {
			case g.Role != "gated" || g.Status == Pass || g.Status == "per-seed":
			case g.Status == Fail:
				pr.ClinicalStatus = Fail
			case pr.ClinicalStatus == Pass:
				pr.ClinicalStatus = Insufficient
			}
		}
	}
	sf := &Safety{Status: Pass, Seeds: []int64{}, Violations: []SafetyViolation{}}
	for _, s := range pr.PerSeed {
		hit := false
		if s.Measures.Clinical == nil {
			sf.Status, hit = Insufficient, true
		} else {
			for _, a := range s.Measures.Clinical.Absolute {
				if a.Violations > 0 {
					sf.Violations = append(sf.Violations, SafetyViolation{Seed: s.Seed, AbsoluteCount: a})
					hit = true
				}
			}
		}
		if hit {
			sf.Seeds = append(sf.Seeds, s.Seed)
		}
	}
	if len(sf.Violations) > 0 {
		sf.Status = Fail
	}
	pr.ClinicalSafety = sf
	pr.Eligible = pr.Valid && pr.ClinicalStatus == Pass && sf.Status == Pass && pr.SolvencyStatus == Pass && pr.FundingStatus == Pass
}

// clinicalReasons are a v0.2-eval-5 candidate's failed clinical rules,
// its shared allowance judged by j (the plan's, or the comparison's with
// the per-seed safeguard).
func clinicalReasons(cand *PlanReport, j *JointGate) []string {
	var why []string
	if s := cand.ClinicalSafety; s != nil && s.Status != Pass {
		for _, v := range s.Violations {
			why = append(why, fmt.Sprintf("clinical safety: seed %d, %d %s studies or orders beyond the absolute %s limit of %d min (worst %d)",
				v.Seed, v.Violations, v.Priority, v.Limit, v.LimitMin, v.WorstMin))
		}
		if len(s.Violations) == 0 {
			why = append(why, "clinical safety: not measured")
		}
	}
	if j != nil && j.Status != Pass {
		why = append(why, fmt.Sprintf("clinical seed reliability: %s (%d failing %v and %d insufficient %v of %d seeds, limit %d)",
			j.Status, len(j.FailingSeeds), j.FailingSeeds, len(j.InsufficientSeeds), j.InsufficientSeeds, j.Seeds, j.FailLimit))
	}
	for _, gs := range [][]GateSummary{cand.ServiceGates, cand.ClinicalGates} {
		for _, g := range gs {
			if g.Role == "gated" && (g.Status == Fail || g.Status == Insufficient) {
				why = append(why, fmt.Sprintf("clinical %s %s %s pooled: %s (%d/%d)", g.Gate, g.Window, g.Stratum, g.Status, g.Pooled.Num, g.Pooled.Den))
			}
		}
	}
	return why
}
