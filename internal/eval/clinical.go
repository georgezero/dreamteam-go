package eval

import (
	"fmt"
	"github.com/georgezero/dreamteam-go/internal/simtime"
	"maps"
	"math/bits"
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
	// Under the safetyCohort ordersByHorizon (Slice 5C.4h), a past-due
	// limit's studies by state at the end of observation, each from its
	// own due time (acquisition plus the target): Observed, finished
	// beyond the limit; Overdue, unfinished and already beyond it (an
	// unresolved overdue obligation, a violation whatever follows);
	// Pending, unfinished and not yet beyond it (not yet observable:
	// never a pass). Violations = Observed + Overdue. PostHorizon: the
	// cohort's studies acquired after the horizon, of orders wanted by it
	// (judged in the follow-up). Unacquired: orders wanted by the horizon
	// still open without a study (no report is yet due; Gate A judges
	// their access).
	SafetyCohort string `json:"safetyCohort,omitempty"`
	Observed     int64  `json:"observed,omitempty"`
	Overdue      int64  `json:"overdue,omitempty"`
	Pending      int64  `json:"pending,omitempty"`
	PostHorizon  int64  `json:"postHorizon,omitempty"`
	Unacquired   int64  `json:"unacquired,omitempty"`
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
	WindowWeeks       int64        `json:"windowWeeks,omitempty"` // calendar weeks a window, when more than one
	Weeks             []WeekSevere `json:"weeks,omitempty"`
	PartialWeeks      int          `json:"partialWeeks,omitempty"`
	InsufficientWeeks int          `json:"insufficientWeeks,omitempty"`
	NotJudgedWeeks    int          `json:"notJudgedWeeks,omitempty"`
	// Under partialWindows countCeiling (Slice 5C.4h): PartialCeiling, the
	// most severe reports a partial window below the minimum may hold
	// (⌊rate × minimum⌋, a compliant minimum window's); PartialFailing,
	// the partial windows above it (failing, counted in Failing); and
	// NoJudgedWindow, when no window was judged and none is insufficient
	// (the gate is then insufficient, never a vacuous pass).
	PartialCeiling *int64 `json:"partialSevereCeiling,omitempty"`
	// PartialRule: under partialWindows rateCeiling (Slice 5C.4j), every
	// nonempty partial window below the minimum fails when its own severe
	// fraction exceeds the rate (an observed safety ceiling, not an
	// estimate); PartialFailing and NoJudgedWindow as above.
	PartialRule    string `json:"partialRule,omitempty"`
	PartialFailing int    `json:"partialFailingWeeks,omitempty"`
	NoJudgedWindow bool   `json:"noJudgedWindow,omitempty"`
}

// WeekSevere is one calendar week of a priority's weekly severity.
// Complete: the whole week, Monday 00:00 to Sunday 24:00, lies within the
// gated period. Status: pass or fail against the rate when the week has
// at least the minimum cohort; below it, insufficient when complete (the
// seed then lacks evidence) and not_judged when partial (its shortfall is
// the evaluation window's edge, the same for every plan; reported, never
// a pass).
type WeekSevere struct {
	From     string `json:"from"` // Monday
	To       string `json:"to"`   // Sunday
	FirstDay int64  `json:"firstDay"`
	Weeks    int64  `json:"weeks,omitempty"` // calendar weeks, for a multi-week window
	// Occupied: of a multi-week window's calendar weeks, those with an
	// acquisition (audit G6.1).
	Occupied int64   `json:"occupiedWeeks,omitempty"`
	Complete bool    `json:"complete"`
	Acquired int64   `json:"acquired"`
	Severe   int64   `json:"severe"`
	Rate     float64 `json:"rate"`
	Status   string  `json:"status"`

	first int64  // its first week (index)
	occ   uint16 // its occupied weeks, by offset from first
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
	// included) and, under the safetyCohort ordersByHorizon, every study
	// of an order wanted by the horizon acquired in the follow-up after
	// it; and, when absolute, every served or open order's wait.
	byHorizon := c.SafetyCohort == SafetyOrdersByHorizon
	var wanted map[string]int64 // order ID → wanted time
	var imaged map[string]bool  // orders with a study
	if byHorizon {
		wanted, imaged = map[string]int64{}, map[string]bool{}
		for i := range r.Orders {
			wanted[r.Orders[i].ID] = r.Orders[i].WantedSec
		}
		for i := range r.Studies {
			imaged[r.Studies[i].Order] = true
		}
	}
	for _, q := range Priorities {
		a := AbsoluteCount{Limit: "pastDue", Priority: q, LimitMin: c.AbsoluteMaxPastDueMin[q]}
		if byHorizon {
			a.SafetyCohort = SafetyOrdersByHorizon
		}
		for i := range r.Studies {
			s := &r.Studies[i]
			if s.Priority != q {
				continue
			}
			if s.AcquiredSec > H {
				// checkClinicalRecord has verified the link.
				if w, ok := wanted[s.Order]; !byHorizon || !ok || w > H {
					continue
				}
				a.PostHorizon++
			}
			from := s.AcquiredSec + target(r, q)
			if takeover && s.AcquiredSec <= t0 {
				from = max(from, t0)
			}
			l, cens := delayAt(s, from, E)
			a.WorstMin = max(a.WorstMin, l/60)
			switch {
			case l > a.LimitMin*60:
				a.Violations++
				if byHorizon && cens {
					a.Overdue++
				} else if byHorizon {
					a.Observed++
				}
			case byHorizon && cens:
				a.Pending++
			}
		}
		if byHorizon {
			for i := range r.Orders {
				o := &r.Orders[i]
				if o.Priority == q && o.WantedSec <= H && end.OrderOutcomes[i] == OutOpen && !imaged[o.ID] {
					a.Unacquired++
				}
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
// over fixed calendar windows of the gated period (Slices 5C.4e and
// 5C.4f), each judged on its own acquisitions, never pooled. A
// priority's window is weeklyWeeks(q) calendar weeks: the windows are
// anchored at the first Monday 00:00 at or after the takeover and run
// consecutively from it; the takeover's own week before the anchor is a
// leading partial window; the window the horizon cuts is a trailing
// partial one. A priority without a window (0) is not measured.
func weeklySevereCalendar(c *Clinical, r *Record, E int64) map[string]*WeeklySevere {
	t0, H := r.TakeoverSec, r.HorizonEndSec
	date, _ := simtime.ParseDate(r.TakeoverDate)
	// Week k runs from local day 7k − 3 (a Monday) for seven days.
	week := func(sec int64) int64 { return floorDiv(sec/daySec+3, 7) }
	monday := func(k int64) int64 { return (7*k - 3) * daySec }
	first, last := week(t0+1), week(H)
	anchor := first
	if monday(first) < t0 {
		anchor++
	}
	out := map[string]*WeeklySevere{}
	// index is the window of week k for a priority with L-week windows.
	index := func(k, L int64) int64 {
		if k < anchor {
			return 0
		}
		return anchor - first + (k-anchor)/L
	}
	for _, q := range Priorities {
		L := c.weeklyWeeks(q)
		if L == 0 {
			continue
		}
		ws := &WeeklySevere{Partition: PartitionCalendar}
		if L > 1 {
			ws.WindowWeeks = L
		}
		out[q] = ws
		add := func(k, n int64) {
			start := monday(k)
			w := WeekSevere{FirstDay: 7*k - 3 - t0/daySec, Complete: start >= t0 && start+7*n*daySec <= H, first: k}
			w.From, w.To = date.AddDays(w.FirstDay).String(), date.AddDays(w.FirstDay+7*n-1).String()
			if L > 1 {
				w.Weeks = n
			}
			ws.Weeks = append(ws.Weeks, w)
		}
		if anchor > first {
			add(first, 1)
		}
		for k := anchor; k <= last; k += L {
			add(k, L)
		}
	}
	for i := range r.Studies {
		s := &r.Studies[i]
		ws := out[s.Priority]
		if ws == nil || s.AcquiredSec <= t0 || s.AcquiredSec > H {
			continue
		}
		k := week(s.AcquiredSec)
		w := &ws.Weeks[index(k, c.weeklyWeeks(s.Priority))]
		w.Acquired++
		w.occ |= 1 << (k - w.first)
		if l, _ := delayAt(s, s.AcquiredSec+target(r, s.Priority), E); c.severe(s.Priority, l) {
			w.Severe++
		}
	}
	for _, q := range Priorities {
		ws := out[q]
		if ws == nil {
			continue
		}
		bpRate, minCohort := bp(c.MaxWeeklySevereRate[q]), c.MinWeeklyCohort[q]
		ceiling, rated := c.PartialWindows == PartialCountCeiling, c.PartialWindows == PartialRateCeiling
		if rated {
			ws.PartialRule = PartialRateCeiling
		}
		if ceiling {
			// The most severe a window of the minimum size may hold.
			n := bpRate * minCohort / 10_000
			ws.PartialCeiling = &n
		}
		worst := -1
		for i := range ws.Weeks {
			w := &ws.Weeks[i]
			ws.Acquired += w.Acquired
			ws.Severe += w.Severe
			if w.Acquired > 0 {
				w.Rate = float64(w.Severe) / float64(w.Acquired)
			}
			if ws.WindowWeeks > 1 {
				w.Occupied = int64(bits.OnesCount16(w.occ))
			}
			if !w.Complete {
				ws.PartialWeeks++
			}
			switch {
			case w.Acquired < minCohort && w.Complete:
				w.Status = Insufficient
				ws.InsufficientWeeks++
				continue
			case w.Acquired < minCohort && (ceiling && rateAbove(w.Severe, minCohort, bpRate) || rated && w.Acquired > 0 && rateAbove(w.Severe, w.Acquired, bpRate)):
				// More severe reports than a compliant window of the
				// minimum size could hold: a failure whatever the volume
				// missing at the boundary (audit G4.1).
				w.Status = Fail
				ws.Failing++
				ws.PartialFailing++
			case w.Acquired < minCohort:
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
			// The worst judged window: the highest rate (exact, by
			// cross-products), the earliest of equals.
			if worst < 0 || w.Severe*ws.Weeks[worst].Acquired > ws.Weeks[worst].Severe*w.Acquired {
				worst = i
			}
		}
		ws.Insufficient = ws.InsufficientWeeks > 0
		if (ceiling || rated) && ws.Blocks == 0 && !ws.Insufficient {
			ws.NoJudgedWindow, ws.Insufficient = true, true
		}
		if worst >= 0 {
			w := ws.Weeks[worst]
			ws.WorstFirstDay, ws.WorstFrom, ws.WorstTo = w.FirstDay, w.From, w.To
			ws.WorstWeeks, ws.WorstOccupiedWeeks = int(max(1, w.Weeks)), bits.OnesCount16(w.occ)
			ws.WorstAcquired, ws.WorstSevere = w.Acquired, w.Severe
		}
	}
	return out
}

// partialWithin states, in a reason, that the not-judged partial windows
// were within the partial-window rule: the count ceiling (5C.4h) or the
// rate (5C.4j: which never establishes that the window was safe).
func partialWithin(ws *WeeklySevere) string {
	if ws.PartialCeiling != nil {
		return fmt.Sprintf("each within the count ceiling of %d severe", *ws.PartialCeiling)
	}
	return "each at or below the rate (too few to establish it)"
}

// calendarWindows names a calendar partition's windows in a reason.
func calendarWindows(ws *WeeklySevere) string {
	if ws.WindowWeeks > 1 {
		return fmt.Sprintf("%d-week calendar blocks", ws.WindowWeeks)
	}
	return "calendar weeks"
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
// default.
//
// Under the safetyCohort ordersByHorizon every study acquired after the
// horizon names exactly one order of the record, of its priority (Slice
// 5C.4h); with orderStudyIntegrity "contract" the whole order–study
// contract for new work holds (Slice 5C.4j, audit 5C.4i I2), so that no
// reporting obligation can be dropped, duplicated or moved between
// cohorts (checkOrderStudies). It reports whether the record can be
// measured clinically.
func checkClinicalRecord(c *Clinical, r *Record, caps map[string]*Capture, bad func(string, ...any)) bool {
	ok := true
	switch end := caps["end"]; {
	case c.OrderStudyIntegrity == IntegrityContract && end != nil && len(end.OrderOutcomes) == len(r.Orders):
		ok = checkOrderStudies(r, end, bad)
	case c.SafetyCohort == SafetyOrdersByHorizon:
		ids, pri := map[string]int{}, map[string]string{}
		for i := range r.Orders {
			ids[r.Orders[i].ID]++
			pri[r.Orders[i].ID] = r.Orders[i].Priority
		}
		for i := range r.Studies {
			s := &r.Studies[i]
			if s.AcquiredSec > r.HorizonEndSec && (ids[s.Order] != 1 || pri[s.Order] != s.Priority) {
				bad("study %s: acquired after the horizon without exactly one order of its priority (%q)", s.ID, s.Order)
				ok = false
				break
			}
		}
	}
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
		case c.MaxWeeklySevereRate != nil && c.WeeklyPartition == PartitionCalendar && c.weeklyWeeks(q) == 0:
			// No weekly gate for this priority (Slice 5C.4f).
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
				g.Status, g.Reason = Fail, fmt.Sprintf("%d of %d judged %s above the rate; worst %d of %d, %s to %s",
					ws.Failing, ws.Blocks, calendarWindows(ws), ws.WorstSevere, ws.WorstAcquired, ws.WorstFrom, ws.WorstTo)
				switch {
				case ws.PartialFailing > 0 && ws.PartialCeiling != nil:
					g.Reason += fmt.Sprintf("; %d partial windows below the minimum with more than the %d severe a minimum window may hold",
						ws.PartialFailing, *ws.PartialCeiling)
				case ws.PartialFailing > 0:
					g.Reason += fmt.Sprintf("; %d partial windows below the minimum with an observed severe fraction above the rate", ws.PartialFailing)
				}
			case ws.Partition == PartitionCalendar && ws.NoJudgedWindow:
				g.Status, g.Reason, g.Undecided = Insufficient, fmt.Sprintf("no window judged: %d partial %s below the minimum cohort of %d, %s",
					ws.NotJudgedWeeks, calendarWindows(ws), c.MinWeeklyCohort[q], partialWithin(ws)), true
			case ws.Partition == PartitionCalendar && ws.Insufficient:
				g.Status, g.Reason = Insufficient, fmt.Sprintf("%d of %d complete %s below the minimum cohort of %d",
					ws.InsufficientWeeks, len(ws.Weeks)-ws.PartialWeeks, calendarWindows(ws), c.MinWeeklyCohort[q])
			case ws.Insufficient:
				g.Status, g.Reason = Insufficient, fmt.Sprintf("%d acquired in the gated period, fewer than %d", ws.Acquired, c.MinWeeklyCohort[q])
			case ws.Failing > 0:
				g.Status, g.Reason = Fail, fmt.Sprintf("%d of %d weekly blocks above the rate; worst %d of %d in %d weeks from day %d",
					ws.Failing, ws.Blocks, ws.WorstSevere, ws.WorstAcquired, ws.WorstWeeks, ws.WorstFirstDay)
				if ws.WorstFrom != "" {
					g.Reason += fmt.Sprintf(" (%s to %s)", ws.WorstFrom, ws.WorstTo)
				}
			case (ws.PartialCeiling != nil || ws.PartialRule != "") && ws.NotJudgedWeeks > 0:
				// Disclosed, never counted as passing windows.
				g.Reason = fmt.Sprintf("%d judged %s pass; %d partial windows below the minimum not judged, %s",
					ws.Blocks, calendarWindows(ws), ws.NotJudgedWeeks, partialWithin(ws))
			}
		default:
			g.Value, g.Num = float64(cm.WorstWeekSevere[q]), cm.WorstWeekSevere[q]
			if cm.WorstWeekSevere[q] > c.MaxSeverePerWeek[q] {
				g.Status, g.Reason = Fail, fmt.Sprintf("%d in the week from day %d", cm.WorstWeekSevere[q], cm.WorstWeekFirstDay[q])
			}
		}
		if !(c.MaxWeeklySevereRate != nil && c.WeeklyPartition == PartitionCalendar && c.weeklyWeeks(q) == 0) {
			out = append(out, g)
		}
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
// the plan NO GO (clinical safety), whatever the seed allowance. Under
// the safetyCohort ordersByHorizon, a study not yet observable against
// its limit when observation ends (Pending) leaves the safety
// insufficient when nothing is violated: never a pass.
type Safety struct {
	Status     string            `json:"status"`
	Seeds      []int64           `json:"seeds"`
	Violations []SafetyViolation `json:"violations"`
	Pending    []SafetyViolation `json:"pending,omitempty"`
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
	for _, s := range pr.PerSeed {
		note := func(key string) {
			if pr.ClinicalInsufficient == nil {
				pr.ClinicalInsufficient = map[string][]int64{}
			}
			pr.ClinicalInsufficient[key] = append(pr.ClinicalInsufficient[key], s.Seed)
		}
		for _, g := range append(slices.Clone(s.Gates), s.ClinicalGates...) {
			// A severity-window gate's evaluability is read from its windows
			// (below): a failing window must not hide another's missing
			// evidence (audit 5C.4i I6).
			if g.PerSeed && role(p, g.Window) == "gated" && g.Status == Insufficient && g.Gate != GateSevereWeek {
				note(g.Gate + " " + g.Window + " " + g.Stratum)
			}
		}
		if c := s.Measures.Clinical; c != nil {
			for _, q := range Priorities {
				if ws := c.WeeklySevere[q]; ws != nil && ws.Insufficient {
					note(GateSevereWeek + " " + WinGated + " " + q)
				}
			}
		}
		for _, g := range s.ClinicalGates {
			if g.Gate == GateSevereWeek && g.Status == Insufficient && (s.Measures.Clinical == nil || s.Measures.Clinical.WeeklySevere[g.Stratum] == nil) {
				note(g.Gate + " " + g.Window + " " + g.Stratum)
			}
		}
		if s.Backlog != "stable" && s.Backlog != "unstable" {
			note("backlog_stability")
		}
	}
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
				if a.Pending > 0 {
					sf.Pending = append(sf.Pending, SafetyViolation{Seed: s.Seed, AbsoluteCount: a})
					sf.Status = Insufficient
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
		for _, v := range s.Pending {
			why = append(why, fmt.Sprintf("clinical safety: seed %d, %d %s studies unfinished and not yet observable against the absolute %s limit of %d min when observation ended",
				v.Seed, v.Pending, v.Priority, v.Limit, v.LimitMin))
		}
		if len(s.Violations) == 0 && len(s.Pending) == 0 {
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

// checkOrderStudies is the order–study contract for new work:
//   - every study acquired after the takeover has a unique ID and names
//     exactly one order of the record, of its priority and modality,
//     whose final outcome is served and whose resolution is that
//     acquisition;
//   - no two studies name the same order;
//   - every order received after the takeover and served has exactly one
//     such study.
//
// Inherited work is exempt where the records cannot reconcile it: a
// study acquired at or before the takeover (its order may have been
// resolved before it and be absent), and an order received at or before
// the takeover and served after it without a post-takeover study.
func checkOrderStudies(r *Record, end *Capture, bad func(string, ...any)) bool {
	t0 := r.TakeoverSec
	byID := map[string][]int{}
	for i := range r.Orders {
		byID[r.Orders[i].ID] = append(byID[r.Orders[i].ID], i)
	}
	linked := map[int]int{} // order index → its post-takeover studies
	ids := map[string]bool{}
	for i := range r.Studies {
		s := &r.Studies[i]
		if s.AcquiredSec <= t0 {
			continue
		}
		if ids[s.ID] {
			bad("study %s: its ID repeats", s.ID)
			return false
		}
		ids[s.ID] = true
		os := byID[s.Order]
		if len(os) != 1 {
			bad("study %s: names %d orders (%q), not exactly one", s.ID, len(os), s.Order)
			return false
		}
		o := &r.Orders[os[0]]
		switch {
		case o.Priority != s.Priority || o.Modality != s.Modality:
			bad("study %s: its order %s asked for %s %s, not %s %s", s.ID, o.ID, o.Priority, o.Modality, s.Priority, s.Modality)
			return false
		case end.OrderOutcomes[os[0]] != OutServed:
			bad("study %s: its order %s ended %q, not served", s.ID, o.ID, end.OrderOutcomes[os[0]])
			return false
		case o.ResolvedSec != s.AcquiredSec:
			bad("study %s: acquired at %d, its order %s imaged at %d", s.ID, s.AcquiredSec, o.ID, o.ResolvedSec)
			return false
		}
		if linked[os[0]]++; linked[os[0]] > 1 {
			bad("order %s: more than one study", o.ID)
			return false
		}
	}
	for i := range r.Orders {
		o := &r.Orders[i]
		if o.ReceivedSec > t0 && end.OrderOutcomes[i] == OutServed && linked[i] != 1 {
			bad("order %s: served with no study", o.ID)
			return false
		}
	}
	return true
}
