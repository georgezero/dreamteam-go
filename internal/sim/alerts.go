package sim

import (
	"fmt"
	"slices"
	"strings"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Alerts (TUI_DESIGN §4.1, ROADMAP step 12): conditions a manager would
// want to know about, each computed from observable state only — patients
// on site, rooms, the roster, the worklist, today's demand counters and the
// books. Never from a hidden draw: no future arrival, read time or payer
// outcome is used.
//
// Rules are evaluated after every transaction from the takeover on. An
// alert is raised once per episode (logged with its severity), escalates
// at most once (warn → alert), and resolves when its condition has been
// clear for its clear delay (logged as info); day-scoped alerts expire at
// midnight. Conditions that change with time alone (a wait or a TAT
// crossing its threshold, a shift cap approaching, a clear delay ending)
// get an exact wake: one pending passive event at the earliest such time,
// so alerts never depend on unrelated events and never poll.

// AlertConfig holds the alert thresholds.
type AlertConfig struct {
	WaitWarn, WaitAlert       simtime.Duration // a patient on site waiting this long (past the appointment, or since arrival)
	CapWarn                   simtime.Duration // a held-over employee this close to the shift cap
	ClosingWarn, ClosingAlert simtime.Duration // operations still running this long after scheduled close
	BacklogWarn               int              // unread + reading studies
	LostWarn                  int              // orders lost today, per modality
	LowCashCents              int64
	ClearAfter                simtime.Duration // a cleared wait, room or backlog alert resolves after this long
}

// DefaultAlerts are the default thresholds.
func DefaultAlerts() AlertConfig {
	return AlertConfig{WaitWarn: 30 * simtime.Minute, WaitAlert: 60 * simtime.Minute, CapWarn: 30 * simtime.Minute,
		ClosingWarn: 30 * simtime.Minute, ClosingAlert: 60 * simtime.Minute, BacklogWarn: 40, LostWarn: 3,
		LowCashCents: 2_500_000, ClearAfter: 10 * simtime.Minute}
}

type alertRule uint8

const (
	ruleCashNegative alertRule = iota
	ruleCashLow
	ruleTAT
	ruleWait
	ruleUnstaffed
	ruleCap
	ruleHeld
	ruleClosing
	ruleBacklog
	ruleLost
)

var ruleInfo = [...]struct {
	key, category, screen string
	clears                bool // resolves only after the clear delay
	expires               bool // ends with its day (not cleared)
	silent                bool // its end is already logged in other words (off duty, operations closed)
}{
	ruleCashNegative: {key: "cash.negative", category: "finance", screen: "f"},
	ruleCashLow:      {key: "cash.low", category: "finance", screen: "f"},
	ruleTAT:          {key: "tat", category: "reading", screen: "w"},
	ruleWait:         {key: "wait", category: "patients", screen: "p", clears: true},
	ruleUnstaffed:    {key: "unstaffed", category: "rooms", screen: "s", clears: true},
	ruleCap:          {key: "shift-cap", category: "staff", screen: "d", silent: true},
	ruleHeld:         {key: "overtime", category: "staff", screen: "d", silent: true},
	ruleClosing:      {key: "closing", category: "practice", screen: "d", silent: true},
	ruleBacklog:      {key: "backlog", category: "reading", screen: "w", clears: true},
	ruleLost:         {key: "lost", category: "demand", screen: "s", expires: true},
}

// alertKey identifies one condition: a rule and its subject (modality,
// room, employee or priority index; 0 when the rule has none).
type alertKey struct {
	rule alertRule
	subj int
}

func (k alertKey) less(o alertKey) int {
	if k.rule != o.rule {
		return int(k.rule) - int(o.rule)
	}
	return k.subj - o.subj
}

// cond is a rule's condition now: its level (1 warn, 2 alert), the level
// up to which it is not logged (the event is already in the log in its own
// words), and its text and explanation.
type cond struct {
	key       alertKey
	level     int8
	quiet     int8
	text, why string
}

type activeAlert struct {
	cond
	raisedText string // the text as last logged (texts drift with time; logs must not)
	fired      int8   // highest level raised this episode
	since      simtime.Time
	falseSince simtime.Time // when the condition last cleared; −1 while it holds
}

type resolvedAlert struct {
	activeAlert
	at      simtime.Time
	expired bool
}

// raisedAlert is one raise or escalation, for period counts.
type raisedAlert struct {
	at    simtime.Time
	level int8
	text  string
}

type alertState struct {
	on       bool // from the takeover
	cfg      AlertConfig
	active   []activeAlert // in key order
	resolved []resolvedAlert
	raised   []raisedAlert
	wakeGen  uint32
	wakeAt   simtime.Time
}

const resolvedCap = 20

func severityOf(level int8) api.Severity {
	if level >= 2 {
		return api.Alert
	}
	return api.Warn
}

// evaluateAlerts reconciles the active alerts with the conditions now and
// arms the next time-based wake.
func (st *state) evaluateAlerts(e *Engine) {
	now := e.Now()
	al := &st.alerts
	conds, next := st.alertConditions(now)
	consider := func(t simtime.Time) {
		if t > now && (next < 0 || t < next) {
			next = t
		}
	}
	var kept []activeAlert
	i := 0
	for _, c := range conds {
		for i < len(al.active) && al.active[i].key.less(c.key) < 0 {
			if a, keep := st.clearing(e, al.active[i], consider); keep {
				kept = append(kept, a)
			}
			i++
		}
		if i < len(al.active) && al.active[i].key == c.key {
			a := al.active[i]
			i++
			a.falseSince = -1
			// The stored text is the text as raised (or escalated): texts
			// drift with time ("longest 42m") and what is kept must not
			// depend on when evaluations happen. Observations recompute it.
			if c.level > a.fired {
				a.level, a.text, a.why = c.level, c.text, c.why
				st.raise(now, &a.cond, c.level)
				a.fired, a.raisedText = c.level, c.text
			}
			kept = append(kept, a)
			continue
		}
		a := activeAlert{cond: c, since: now, falseSince: -1, fired: c.level, raisedText: c.text}
		st.raise(now, &a.cond, c.level)
		kept = append(kept, a)
	}
	for ; i < len(al.active); i++ {
		if a, keep := st.clearing(e, al.active[i], consider); keep {
			kept = append(kept, a)
		}
	}
	al.active = kept
	if next >= 0 && next != al.wakeAt {
		al.wakeGen++
		al.wakeAt = next
		e.Push(Event{At: next, Class: ClassDeadline, Kind: kindAlertWake, Gen: al.wakeGen})
	}
}

// raise logs a raise or escalation (unless quiet at that level) and
// records it.
func (st *state) raise(now simtime.Time, c *cond, level int8) {
	st.alerts.raised = append(st.alerts.raised, raisedAlert{at: now, level: level, text: c.text})
	if level > c.quiet {
		st.emit(now, severityOf(level), ruleInfo[c.key.rule].category, c.text)
	}
}

// clearing handles an active alert whose condition no longer holds: it
// resolves now, or after the clear delay (arming a wake), or expires.
func (st *state) clearing(e *Engine, a activeAlert, consider func(simtime.Time)) (activeAlert, bool) {
	now := e.Now()
	info := ruleInfo[a.key.rule]
	if a.falseSince < 0 {
		a.falseSince = now
	}
	if info.clears && now < a.falseSince.Add(st.alerts.cfg.ClearAfter) {
		consider(a.falseSince.Add(st.alerts.cfg.ClearAfter))
		return a, true
	}
	r := resolvedAlert{activeAlert: a, at: now, expired: info.expires}
	if !info.expires && !info.silent && a.fired > a.quiet {
		text := fmt.Sprintf("Resolved: %s (raised %s)", a.raisedText, st.cal.Clock(a.since))
		if a.key.rule == ruleCashNegative {
			text = "Cash back above zero"
		}
		st.emit(now, api.Info, info.category, text)
	}
	al := &st.alerts
	al.resolved = append(al.resolved, r)
	if len(al.resolved) > resolvedCap {
		al.resolved = al.resolved[len(al.resolved)-resolvedCap:]
	}
	return a, false
}

// alertConditions computes every rule's condition now, in key order, and
// the earliest later time at which one changes with no event.
func (st *state) alertConditions(now simtime.Time) ([]cond, simtime.Time) {
	cfg := &st.alerts.cfg
	next := simtime.Time(-1)
	consider := func(t simtime.Time) {
		if t > now && (next < 0 || t < next) {
			next = t
		}
	}
	var out []cond
	add := func(r alertRule, subj int, level, quiet int8, text, why string) {
		out = append(out, cond{key: alertKey{r, subj}, level: level, quiet: quiet, text: text, why: why})
	}
	active := func(r alertRule, subj int) bool {
		for _, a := range st.alerts.active {
			if a.key == (alertKey{r, subj}) {
				return true
			}
		}
		return false
	}

	// Cash (from the takeover; no credit line, §8.8).
	if st.fin.opened {
		c := st.fin.led.bal[acctCash]
		if c < 0 {
			add(ruleCashNegative, 0, 2, 0, fmt.Sprintf("Cash below zero: financing shortfall %s (no credit line is modelled; the shortfall is tracked)", money(-c)), st.cashWhy())
		}
		low := cfg.LowCashCents
		if active(ruleCashLow, 0) {
			low += low / 10 // hysteresis: clears 10% above the threshold
		}
		if c >= 0 && c < low {
			add(ruleCashLow, 0, 1, 0, fmt.Sprintf("Cash low: %s (alert below %s)", money(c), money(cfg.LowCashCents)), st.cashWhy())
		}
	}

	// Report turnaround: outstanding studies past their target, by
	// priority. Urgent is an alert; soon breaches are already logged one
	// by one (quiet); routine ones are logged here.
	var overdue [numPriorities]int
	var oldest [numPriorities]simtime.Duration
	for _, si := range st.dem.worklist {
		s := &st.dem.studies[si]
		tgt := st.cfg.Policies.TATTarget[s.priority]
		age := now.Sub(s.acquiredAt)
		if age > tgt {
			overdue[s.priority]++
			oldest[s.priority] = max(oldest[s.priority], age)
		} else {
			consider(s.acquiredAt.Add(tgt + 1))
		}
	}
	for p := Routine; p < numPriorities; p++ {
		if overdue[p] == 0 {
			continue
		}
		level, quiet := int8(1), int8(0)
		switch p {
		case Urgent:
			level = 2
		case Soon:
			quiet = 1
		}
		add(ruleTAT, int(p), level, quiet, fmt.Sprintf("%d %s report%s overdue (target %s) · oldest %s",
			overdue[p], p, plural(overdue[p]), st.cfg.Policies.TATTarget[p], oldest[p]), st.readingWhy())
	}

	// Waits: patients on site not yet in an exam, measured past their
	// appointment time (scheduled) or since arrival (walk-ins, patients
	// fitted in after being marked missed).
	type waitAgg struct {
		n, over int
		longest simtime.Duration
	}
	waits := make([]waitAgg, len(st.mods))
	for _, vi := range st.dem.waiting {
		v := &st.dem.visits[vi]
		if v.state == visitInExam {
			continue
		}
		k := st.modIndex(st.cfg.Catalog[st.dem.orders[v.order].exam].Modality)
		if k < 0 {
			continue
		}
		from := v.arrived
		if v.appt >= 0 && !v.afterGrace {
			from = max(from, st.apptStart(v.appt))
		}
		w := &waits[k]
		w.n++
		d := now.Sub(from)
		w.longest = max(w.longest, d)
		if d >= cfg.WaitWarn {
			w.over++
		}
		consider(from.Add(cfg.WaitWarn))
		consider(from.Add(cfg.WaitAlert))
	}
	for k, w := range waits {
		if w.over == 0 {
			continue
		}
		level := int8(1)
		if w.longest >= cfg.WaitAlert {
			level = 2
		}
		m := st.mods[k]
		add(ruleWait, k, level, 0, fmt.Sprintf("%s: %d patient%s waiting over %s · longest %s",
			m, w.over, plural(w.over), cfg.WaitWarn, w.longest), st.waitWhy(m, w.n))
	}

	// Rooms open but unstaffed with patients for them on site.
	if st.open {
		for r := range st.rooms {
			rm := &st.rooms[r]
			if rm.state != roomUnstaffed {
				continue
			}
			if n := st.onSiteFor(rm.cfg.Modality); n > 0 {
				add(ruleUnstaffed, r, 1, 0, fmt.Sprintf("%s room unstaffed with %d patient%s on site",
					rm.cfg.Modality, n, plural(n)), st.unstaffedWhy(r))
			}
		}
	}

	// Overtime: held past shift end (already logged when it happened), and
	// the shift cap approaching while held.
	for i := range st.emps {
		emp := &st.emps[i]
		if !emp.held {
			continue
		}
		if warnAt := emp.capAt.Add(-cfg.CapWarn); now >= warnAt {
			add(ruleCap, i, 2, 0, fmt.Sprintf("%s reaches %s at %s", emp.cfg.Name,
				st.capName(emp.startAt, emp.capAt, "shift limit"), st.cal.Clock(emp.capAt)),
				"work that cannot be finished by then is rescheduled; nobody works past the limit")
		} else {
			consider(warnAt)
		}
	}
	for i := range st.emps {
		emp := &st.emps[i]
		if emp.held {
			add(ruleHeld, i, 1, 1, fmt.Sprintf("%s held past shift end (%s): overtime", emp.cfg.Name,
				st.cal.Clock(emp.endAt)), st.neededFor(i))
		}
	}

	// Closing running late.
	if st.closing && st.plan.Hours != nil {
		closeAt := st.cal.At(st.day, st.plan.Hours.To)
		late := now.Sub(closeAt)
		consider(closeAt.Add(cfg.ClosingWarn))
		consider(closeAt.Add(cfg.ClosingAlert))
		if late >= cfg.ClosingWarn {
			level := int8(1)
			if late >= cfg.ClosingAlert {
				level = 2
			}
			n := len(st.dem.waiting)
			add(ruleClosing, 0, level, 0, fmt.Sprintf("Closing running late: %d patient%s still on site %s after %s",
				n, plural(n), late, simtime.FormatClock(st.plan.Hours.To)), st.closingWhy())
		}
	}

	// Reporting backlog (clears below 80% of the threshold).
	n := len(st.dem.worklist)
	limit := cfg.BacklogWarn
	if active(ruleBacklog, 0) {
		limit = limit * 4 / 5
	}
	if cfg.BacklogWarn > 0 && n >= limit {
		add(ruleBacklog, 0, 1, 0, fmt.Sprintf("Reporting backlog: %d studies unread or being read", n), st.readingWhy())
	}

	// Demand lost today, per modality (expires at midnight).
	for k, m := range st.mods {
		if lost := st.dem.today[m].lost; cfg.LostWarn > 0 && lost >= cfg.LostWarn {
			add(ruleLost, k, 1, 0, fmt.Sprintf("%s: %d order%s lost today", m, lost, plural(lost)), st.lostWhy(m, now))
		}
	}
	slices.SortStableFunc(out, func(a, b cond) int { return a.key.less(b.key) })
	return out, next
}

// --- explanations, from observed facts -------------------------------------

func (st *state) cashWhy() string {
	led := &st.fin.led
	day := st.day
	date := st.cal.Date(st.cal.StartOfDay(day))
	payday := st.cal.Date(st.cal.StartOfDay(st.nextPayday(day)))
	bills := date.AddDays(int64(date.DaysInMonth() - date.Day + 1))
	return fmt.Sprintf("payroll %s accrued, paid %s · bills %s accrued, paid %s · receivables %s still to collect",
		money(-led.bal[acctAccruedPayroll]), payday.String()[5:], money(-led.bal[acctAP]), bills.String()[5:], money(led.bal[acctAR]))
}

// readingWhy says who is reading and when site reading next happens.
func (st *state) readingWhy() string {
	var reading []string
	inBlock := 0
	next := simtime.Time(-1)
	for i := range st.emps {
		emp := &st.emps[i]
		if emp.cfg.Role != Radiologist {
			continue
		}
		if emp.task == taskRead {
			reading = append(reading, emp.cfg.Name)
		}
		if emp.reading && emp.status != offDuty {
			inBlock++
		}
		if emp.shift >= 0 && !emp.done {
			for _, b := range st.plan.Reading[i] {
				if t := st.cal.At(st.day, b.From); t > st.now && (next < 0 || t < next) {
					next = t
				}
			}
		}
	}
	parts := []string{}
	switch {
	case len(reading) > 0:
		parts = append(parts, "reading now: "+strings.Join(reading, ", "))
	case inBlock > 0:
		parts = append(parts, fmt.Sprintf("%d radiologist%s in a site-reading block", inBlock, plural(inBlock)))
	default:
		parts = append(parts, "no radiologist in a site-reading block now")
	}
	if next >= 0 {
		parts = append(parts, "next block "+st.cal.Clock(next))
	}
	return strings.Join(parts, " · ")
}

// waitWhy explains a modality's waits: what its rooms are doing.
func (st *state) waitWhy(m string, waiting int) string {
	var parts []string
	for r := range st.rooms {
		rm := &st.rooms[r]
		if rm.cfg.Modality != m {
			continue
		}
		switch rm.state {
		case roomOccupied:
			v := &st.dem.visits[rm.visit]
			x := &st.cfg.Catalog[st.dem.orders[v.order].exam]
			parts = append(parts, fmt.Sprintf("room in use (%s, %s of %s expected)", x.Name,
				st.now.Sub(v.examStart), x.Expected()))
		case roomUnstaffed:
			parts = append(parts, "room unstaffed: "+st.unstaffedWhy(r))
		case roomIdle:
			var away []string
			for _, i := range rm.staff {
				emp := &st.emps[i]
				if emp.status == busy && emp.task == taskExam {
					if v := &st.dem.visits[emp.taskVisit]; v.room != r {
						away = append(away, fmt.Sprintf("%s busy in %s", emp.cfg.Name, st.cfg.Rooms[v.room].Modality))
					}
				} else if emp.status == busy {
					away = append(away, emp.cfg.Name+" busy ("+st.taskText(i)+")")
				}
			}
			if len(away) == len(rm.staff) && len(away) > 0 {
				parts = append(parts, "shared technologist "+strings.Join(away, ", "))
			} else {
				parts = append(parts, "room free")
			}
		}
	}
	reg := 0
	for _, vi := range st.dem.waiting {
		v := &st.dem.visits[vi]
		if (v.state == visitArrived || v.state == visitRegistering) &&
			st.cfg.Catalog[st.dem.orders[v.order].exam].Modality == m {
			reg++
		}
	}
	if reg > 0 {
		if lvl, _ := st.desk(); lvl == DeskSelf {
			parts = append(parts, fmt.Sprintf("%d still to register, no desk staff on duty", reg))
		} else {
			parts = append(parts, fmt.Sprintf("%d still to register", reg))
		}
	}
	return fmt.Sprintf("%d on site for %s · %s", waiting, m, strings.Join(parts, " · "))
}

// unstaffedWhy says why room r has nobody: the rostered technologist on
// break, or the next time the roster covers it.
func (st *state) unstaffedWhy(r int) string {
	clock := simtime.ClockOf(st.now)
	room := st.cfg.Rooms[r]
	for i := range st.emps {
		emp := &st.emps[i]
		if emp.status == onBreak && emp.shift >= 0 && staffs(emp.cfg, &st.plan.Shifts[emp.shift], room, clock) {
			return fmt.Sprintf("%s on break until %s", emp.cfg.Name, st.cal.Clock(emp.breakUntil))
		}
	}
	for _, seg := range st.cov.Segments {
		if seg.From > clock && seg.Open && len(seg.Rooms[r]) > 0 {
			return fmt.Sprintf("no %s technologist rostered until %s", room.Modality, simtime.FormatClock(seg.From))
		}
	}
	return fmt.Sprintf("no %s technologist rostered for the rest of today", room.Modality)
}

func (st *state) closingWhy() string {
	var held []string
	for i := range st.emps {
		if st.emps[i].held {
			held = append(held, st.emps[i].cfg.Name)
		}
	}
	if len(held) == 0 {
		return "staff on duty are finishing the last patients"
	}
	return "held over: " + strings.Join(held, ", ")
}

func (st *state) lostWhy(m string, now simtime.Time) string {
	today := st.cal.DayIndex(now)
	if day, ok := st.nextOpen(m, today, now.Add(st.cfg.Policies.BookingNotice)); ok {
		return fmt.Sprintf("next open %s slot %s (%d days)", m, st.cal.Date(st.cal.StartOfDay(day)).String()[5:], day-today)
	}
	return fmt.Sprintf("no open %s slot within the %d-day booking horizon", m, st.cfg.Policies.BookingHorizonDays)
}

// --- observation -----------------------------------------------------------

func (st *state) alertItem(a *activeAlert, sev api.Severity) api.AlertItem {
	info := ruleInfo[a.key.rule]
	key := info.key
	switch a.key.rule {
	case ruleCashNegative, ruleCashLow, ruleClosing, ruleBacklog:
	default: // rules with a subject: modality, room, employee or priority index
		key = fmt.Sprintf("%s.%d", key, a.key.subj)
	}
	return api.AlertItem{Key: key, Severity: sev, Since: st.cal.ISO(a.since), Text: a.text, Why: a.why, Screen: info.screen}
}

// observeAlerts lists the active alerts (most severe, then oldest first)
// with their texts as of now, and the latest resolved ones.
func (st *state) observeAlerts(o *api.Observation, now simtime.Time) {
	al := &st.alerts
	o.Alerts.Active, o.Alerts.Resolved = nil, nil
	if !al.on {
		return
	}
	conds, _ := st.alertConditions(now)
	for i := range al.active {
		a := al.active[i]
		for _, c := range conds {
			if c.key == a.key {
				a.text, a.why = c.text, c.why
			}
		}
		o.Alerts.Active = append(o.Alerts.Active, st.alertItem(&a, severityOf(a.fired)))
	}
	slices.SortStableFunc(o.Alerts.Active, func(x, y api.AlertItem) int {
		if x.Severity != y.Severity {
			if x.Severity == api.Alert {
				return -1
			}
			return 1
		}
		return strings.Compare(x.Since, y.Since)
	})
	for i := len(al.resolved) - 1; i >= 0; i-- {
		r := &al.resolved[i]
		it := st.alertItem(&r.activeAlert, severityOf(r.fired))
		it.Resolved, it.Expired = st.cal.ISO(r.at), r.expired
		o.Alerts.Resolved = append(o.Alerts.Resolved, it)
	}
	start := st.cal.StartOfDay(st.cal.DayIndex(now))
	for k := len(al.raised) - 1; k >= 0 && al.raised[k].at >= start; k-- {
		o.Alerts.RaisedToday++
	}
}

// alertCounts summarizes the alerts raised in [from, to) of the raised list.
func (st *state) alertCounts(from, to int) api.AlertCounts {
	var c api.AlertCounts
	rs := st.alerts.raised[from:to]
	for _, r := range rs {
		if r.level >= 2 {
			c.Alert++
		} else {
			c.Warn++
		}
	}
	for _, lvl := range []int8{2, 1} {
		for _, r := range rs {
			if r.level == lvl && len(c.Top) < 5 {
				c.Top = append(c.Top, r.text)
			}
		}
	}
	return c
}
