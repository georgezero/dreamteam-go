package sim

import (
	"fmt"
	"slices"
	"strings"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Actions (SIMULATION_DESIGN §3.4.1, §9.5): the only way anything outside
// the simulation changes it. The TUI, a future WebUI and agents all call
// World.Apply with a batch; validation here is the only gatekeeper.
//
// A batch applies at the current clock, between transactions. Its actions
// are validated and applied in order, each against the state the previous
// ones left; a rejected action changes nothing and does not stop the
// others. No dispatch runs between actions. If at least one action changed
// something, one dispatch (and the alert observer) runs at the end, so a
// policy change can start work at once; a batch of rejections and no-ops
// runs nothing. Every attempt is recorded with its result, in order, for
// the action log and replay.

const recentActions = 20

type actionRecord struct {
	at     simtime.Time
	batch  int64
	action api.Action
	ok     bool
	code   string
	msg    string
}

type actionState struct {
	log     []actionRecord
	batches int64
}

func actionID(i int) string { return pubID('X', int64(i+1), 5) }

// Apply applies one batch of actions now and returns their results.
func (w *World) Apply(actions []api.Action) []api.ActionResult {
	if len(actions) == 0 {
		return nil // no decision, no batch
	}
	st := w.st
	now := w.eng.Now()
	prev := st.now
	st.now = now
	st.actions.batches++
	changed := false
	var out []api.ActionResult
	w.eng.Batch(func() {
		for _, a := range actions {
			ok, code, msg := st.apply(w.eng, a, true)
			changed = changed || (ok && code == api.ActionAccepted)
			st.actions.log = append(st.actions.log, actionRecord{at: now, batch: st.actions.batches, action: a, ok: ok, code: code, msg: msg})
			out = append(out, st.actionResult(len(st.actions.log)-1))
		}
	})
	if changed {
		w.eng.Settle()
	} else {
		st.now = prev // nothing happened: not even the clock cache moves
	}
	return out
}

// beforeChange runs once an action is valid and about to change the
// state: the interval metrics are accrued to now first. (Rejections and
// no-ops leave even the accrual point untouched.)
func (st *state) beforeChange() { st.accrue(st.now) }

// Validate checks an action against the state now without applying or
// recording it (a preview for frontends).
func (w *World) Validate(a api.Action) api.ActionResult {
	st := w.st
	prev := st.now
	st.now = w.eng.Now()
	ok, code, msg := st.apply(w.eng, a, false)
	st.now = prev
	return api.ActionResult{Time: st.cal.ISO(st.now), Action: a, OK: ok, Code: code, Message: msg}
}

func (st *state) actionResult(i int) api.ActionResult {
	r := &st.actions.log[i]
	return api.ActionResult{ID: actionID(i), Batch: r.batch, Time: st.cal.ISO(r.at), Action: r.action,
		OK: r.ok, Code: r.code, Message: r.msg}
}

// apply validates a and, if do is set and it is valid, applies it.
func (st *state) apply(e *Engine, a api.Action, do bool) (ok bool, code, msg string) {
	switch a.Type {
	case api.ActExtendShift:
		return st.extendShift(e, a, do)
	case api.ActSetWalkInPolicy:
		return st.setWalkIn(e, a, do)
	case api.ActSetWorklistPolicy:
		return st.setWorklist(e, a, do)
	}
	return false, "unknown_action", fmt.Sprintf("Unknown action %q (V0.1: %s, %s, %s).", a.Type,
		api.ActExtendShift, api.ActSetWalkInPolicy, api.ActSetWorklistPolicy)
}

// MaxExtension bounds one extend_shift (the shift cap bounds the total).
const MaxExtension = 4 * simtime.Hour

// planFor is day's plan: the roster with the extensions approved for it.
func (st *state) planFor(day int64) DayPlan {
	p := st.cfg.Plan(st.cal.Date(st.cal.StartOfDay(day)))
	for i, d := range st.ext[day] {
		if k := p.ByEmp[i]; k >= 0 {
			p.Shifts[k].To += d
		}
	}
	return p
}

// extendShift approves extra time at the end of one employee's shift on
// a day (today by default): their scheduled end moves later, so they stay
// on regular duty staffing their rooms (or the desk, or supervision). It
// is new capacity: slots in the extra time become bookable at once (for a
// later day, bookings can fill them in advance), and today it serves
// waiting patients and walk-ins — not only the finishing of admitted
// patients, which hold-over already covers. Time past the rostered end is
// approved overtime, paid as worked.
func (st *state) extendShift(e *Engine, a api.Action, do bool) (bool, string, string) {
	i := -1
	for k := range st.emps {
		if st.emps[k].cfg.ID == a.Employee {
			i = k
		}
	}
	if i < 0 {
		return false, "unknown_employee", fmt.Sprintf("No employee %q.", a.Employee)
	}
	ec := &st.cfg.Employees[i]
	name := ec.Name
	d := simtime.Duration(a.Minutes) * simtime.Minute
	if a.Minutes < 1 || d > MaxExtension {
		return false, "invalid_minutes", fmt.Sprintf("Minutes must be 1–%d.", MaxExtension/simtime.Minute)
	}
	day := st.day
	if a.Date != "" {
		date, err := simtime.ParseDate(a.Date)
		if err != nil {
			return false, "invalid_date", fmt.Sprintf("Date %q is not YYYY-MM-DD.", a.Date)
		}
		day = st.cal.DayIndex(st.cal.StartOfDate(date))
	}
	switch {
	case day < st.day:
		return false, "past_date", fmt.Sprintf("%s is in the past: extensions are approved in advance, never retroactively.", a.Date)
	case day > st.day+st.cfg.Policies.BookingHorizonDays || day >= int64(len(st.dem.days)):
		return false, "beyond_horizon", fmt.Sprintf("%s is beyond the %d-day booking horizon.", a.Date, st.cfg.Policies.BookingHorizonDays)
	}
	plan := st.plan
	if day != st.day {
		plan = st.planFor(day)
	}
	k := plan.ByEmp[i]
	dateText := "today"
	if day != st.day {
		dd := st.cal.Date(st.cal.StartOfDay(day))
		dateText = fmt.Sprintf("%s %s %d", dd.Weekday(), dd.MonthName(), dd.Day)
	}
	if k < 0 {
		return false, "not_on_shift", fmt.Sprintf("%s is not rostered %s.", name, dateText)
	}
	sh := plan.Shifts[k]
	startAt, endAt := st.cal.At(day, sh.From), st.cal.At(day, sh.To)
	capAt := startAt.Add(st.cfg.Policies.MaxShift)
	if day == st.day {
		emp := &st.emps[i]
		switch {
		case emp.done:
			return false, "gone_home", name + " has gone home for today."
		case st.now >= emp.endAt:
			return false, "shift_ended", fmt.Sprintf("%s's shift ended at %s: extensions are approved in advance, never retroactively (staff past their end are held over only while patients need them).",
				name, st.cal.Clock(emp.endAt))
		}
	}
	end := endAt.Add(d)
	if end > capAt {
		return false, "exceeds_max_shift", fmt.Sprintf("%s would work until %s, past the %d-hour limit at %s (at most +%s more).",
			name, st.cal.Clock(end), st.cfg.Policies.MaxShift/simtime.Hour, st.cal.Clock(capAt), capAt.Sub(endAt))
	}
	if plan.Hours == nil || sh.To >= plan.Hours.To {
		return false, "after_close", fmt.Sprintf("%s's shift %s already runs to the scheduled close: an extension after close adds no capacity (staff stay automatically while admitted patients need them).", name, dateText)
	}
	covers, opens := st.extensionEffect(day, plan, k, d)
	if covers == "" {
		return false, "no_coverage", fmt.Sprintf("%s would cover no room, desk or contrast supervision in the extra time %s (a remote radiologist's reading blocks are not extended): nothing to gain.", name, dateText)
	}
	cost := ""
	if ec.Pay == Hourly {
		if day == st.day {
			// The shift as extended, leaving at the new end (breaks as rostered).
			emp := &st.emps[i]
			var brk simtime.Duration
			for _, b := range sh.Breaks {
				brk += b.To - b.From
			}
			paid := end.Sub(emp.startAt) - brk
			cost = " · ≈" + money(st.overtimeCost(emp, paid, end.Sub(emp.rosterEnd))-
				st.overtimeCost(emp, paid-d, emp.endAt.Sub(emp.rosterEnd)))
		} else {
			cost = fmt.Sprintf(" · ≈%s at the base rate, ×%d.%d past %d h in the week", money(ec.RateCents*int64(d)/3600),
				st.cfg.Policies.OvertimeBP/10_000, st.cfg.Policies.OvertimeBP%10_000/1000, st.cfg.Policies.WeeklyHours/simtime.Hour)
		}
	}
	msg := fmt.Sprintf("Shift extended: %s %s until %s (+%s approved overtime%s) · %s %s–%s%s.", name, dateText,
		st.cal.Clock(end), d, cost, covers, simtime.FormatClock(sh.To), simtime.FormatClock(sh.To+d), opens)
	if !do {
		return true, api.ActionAccepted, msg
	}
	st.beforeChange()
	if st.ext == nil {
		st.ext = map[int64]map[int]simtime.Duration{}
	}
	if st.ext[day] == nil {
		st.ext[day] = map[int]simtime.Duration{}
	}
	st.ext[day][i] += d
	if day == st.day {
		emp := &st.emps[i]
		emp.endAt = end
		st.plan.Shifts[k].To = simtime.ClockOf(end)
		e.Push(Event{At: end, Class: ClassCapacityDown, Kind: kindShiftEnd, Gen: emp.gen, A: int64(i)})
		st.cov = st.plan.Coverage()
		plan = st.plan
	} else {
		plan = st.planFor(day)
	}
	// The day's planned coverage and slots: the extra time is staffed, so
	// its slots become bookable for new orders, recalls and rebookings.
	if ds := st.dem.days[day]; ds != nil {
		cov := plan.Coverage()
		ds.segs = cov.Segments
		for r := range ds.rooms {
			for k := range ds.rooms[r] {
				st.coverSlot(&cov, r, &ds.rooms[r][k])
			}
		}
	}
	st.emit(st.now, api.Info, "actions", msg)
	return true, api.ActionAccepted, msg
}

// Roster returns day's shifts with any approved extensions.
func (w *World) Roster(day int64) []api.RosterEntry {
	st := w.st
	plan := st.planFor(day)
	var out []api.RosterEntry
	for i, ec := range st.cfg.Employees {
		k := plan.ByEmp[i]
		if k < 0 {
			continue
		}
		sh := plan.Shifts[k]
		ext := st.ext[day][i]
		x := api.RosterEntry{ID: ec.ID, Name: ec.Name, Role: string(ec.Role), Shift: span(sh.Span),
			RosterEnd: simtime.FormatClock(sh.To - ext), ApprovedMin: int64(ext / simtime.Minute),
			CapAt: simtime.FormatClock(sh.From + st.cfg.Policies.MaxShift)}
		if ec.Pay == Hourly {
			x.HourlyCents = ec.RateCents
		}
		out = append(out, x)
	}
	return out
}

// observeExtensions lists the extensions approved for today and later.
func (st *state) observeExtensions(o *api.Observation, today int64) {
	var days []int64
	for d := range st.ext {
		if d >= today {
			days = append(days, d)
		}
	}
	slices.Sort(days)
	for _, d := range days {
		plan := st.planFor(d)
		for i := range st.cfg.Employees {
			if x := st.ext[d][i]; x > 0 && plan.ByEmp[i] >= 0 {
				o.Extensions = append(o.Extensions, api.Extension{Date: st.cal.Date(st.cal.StartOfDay(d)).String(),
					Employee: st.cfg.Employees[i].ID, Name: st.cfg.Employees[i].Name, Minutes: int64(x / simtime.Minute),
					Until: simtime.FormatClock(plan.Shifts[plan.ByEmp[i]].To)})
			}
		}
	}
}

var walkInPolicies = []string{api.WalkInAcceptAlways, api.WalkInAcceptIfGap, api.WalkInAcceptUntil, api.WalkInScheduleLater}

var worklistPolicies = []string{api.WorklistUrgentThenFIFO, api.WorklistFIFO, api.WorklistOldestFirst, api.WorklistShortestFirst}

// setWalkIn changes how walk-ins arriving from now on are handled; those
// already waiting stay.
func (st *state) setWalkIn(e *Engine, a api.Action, do bool) (bool, string, string) {
	if !slices.Contains(walkInPolicies, a.Policy) {
		return false, "unknown_policy", fmt.Sprintf("Walk-in policy %q is not one of %v.", a.Policy, walkInPolicies)
	}
	var cutoff simtime.Duration
	text := a.Policy
	if a.Policy == api.WalkInAcceptUntil {
		c, err := simtime.ParseClock(a.Cutoff)
		if err != nil || c <= 0 || c >= simtime.Day {
			return false, "invalid_cutoff", fmt.Sprintf("accept_until needs a cutoff time \"HH:MM\" (got %q).", a.Cutoff)
		}
		cutoff = c
		text += " " + simtime.FormatClock(c)
	} else if a.Cutoff != "" {
		return false, "invalid_cutoff", "Only accept_until takes a cutoff."
	}
	if st.pol.WalkIn == a.Policy && st.pol.WalkInCutoff == cutoff {
		return true, api.ActionUnchanged, "Walk-in policy is already " + text + "."
	}
	msg := fmt.Sprintf("Walk-in policy: %s → %s (walk-ins already waiting stay).", st.walkInText(), text)
	if do {
		st.beforeChange()
		st.pol.WalkIn, st.pol.WalkInCutoff = a.Policy, cutoff
		st.emit(st.now, api.Info, "actions", msg)
	}
	return true, api.ActionAccepted, msg
}

func (st *state) walkInText() string {
	if st.pol.WalkIn == api.WalkInAcceptUntil {
		return st.pol.WalkIn + " " + simtime.FormatClock(st.pol.WalkInCutoff)
	}
	return st.pol.WalkIn
}

// setWorklist changes the order in which unread studies are offered to
// radiologists from now on. Reads in progress are never interrupted.
func (st *state) setWorklist(e *Engine, a api.Action, do bool) (bool, string, string) {
	if !slices.Contains(worklistPolicies, a.Policy) {
		return false, "unknown_policy", fmt.Sprintf("Worklist policy %q is not one of %v.", a.Policy, worklistPolicies)
	}
	if a.Cutoff != "" || a.Minutes != 0 || a.Employee != "" {
		return false, "invalid_parameters", "set_worklist_policy takes only a policy."
	}
	if st.pol.Worklist == a.Policy {
		return true, api.ActionUnchanged, "Worklist policy is already " + a.Policy + "."
	}
	msg := fmt.Sprintf("Worklist policy: %s → %s (reads in progress continue).", st.pol.Worklist, a.Policy)
	if do {
		st.beforeChange()
		st.pol.Worklist = a.Policy
		st.emit(st.now, api.Info, "actions", msg)
	}
	return true, api.ActionAccepted, msg
}

// observeActions fills the policies in force and the action log.
func (st *state) observeActions(o *api.Observation) {
	st.observeExtensions(o, st.cal.DayIndex(o.Now))
	o.Policies = api.Policies{WalkIn: st.pol.WalkIn, Worklist: st.pol.Worklist}
	if st.pol.WalkIn == api.WalkInAcceptUntil {
		o.Policies.WalkInCutoff = simtime.FormatClock(st.pol.WalkInCutoff)
	}
	l := &o.Actions
	for i := range st.actions.log {
		if st.actions.log[i].ok {
			l.Accepted++
		} else {
			l.Rejected++
		}
	}
	l.Attempted = len(st.actions.log)
	for i := len(st.actions.log) - 1; i >= 0 && len(l.Recent) < recentActions; i-- {
		l.Recent = append(l.Recent, st.actionResult(i))
	}
}

// actionCounts counts the actions recorded in [from, to).
func (st *state) actionCounts(from, to int) api.ActionCounts {
	c := api.ActionCounts{Attempted: to - from}
	for _, r := range st.actions.log[from:to] {
		if r.ok {
			c.Accepted++
		} else {
			c.Rejected++
		}
	}
	return c
}

// ActionLog returns every action attempted so far with its result, in
// order: the record a replay re-applies (batch by batch, at each Time).
func (w *World) ActionLog() []api.ActionResult {
	var out []api.ActionResult
	for i := range w.st.actions.log {
		out = append(out, w.st.actionResult(i))
	}
	return out
}

// extensionEffect says what extending shift k of plan by d on day would
// add: what the person covers in the extra time ("" if nothing) and the
// template slots that would become bookable.
func (st *state) extensionEffect(day int64, plan DayPlan, k int, d simtime.Duration) (covers, opens string) {
	sh := plan.Shifts[k]
	ec := &st.cfg.Employees[sh.Employee]
	last := sh.To + d - 1 // a moment inside the extra time
	ext := sh
	ext.To += d
	var rooms []string
	for _, room := range st.cfg.Rooms {
		if staffs(ec, &ext, room, last) {
			rooms = append(rooms, room.Modality)
		}
	}
	switch {
	case len(rooms) > 0:
		covers = "staffs " + strings.Join(rooms, "+")
	case deskLevel(ec) != DeskClosed:
		covers = "covers the desk"
	case ec.Role == Radiologist && !sh.Remote:
		covers = "on site for contrast supervision (reading blocks unchanged)"
	}
	if covers == "" {
		return "", ""
	}
	ds := st.dem.days[day]
	if ds == nil {
		return covers, ""
	}
	p := plan
	p.Shifts = slices.Clone(plan.Shifts)
	p.Shifts[k] = ext
	cov := p.Coverage()
	n := map[string]int{}
	var mods []string
	for r := range ds.rooms {
		for _, s := range ds.rooms[r] {
			if s.why == slotOK {
				continue
			}
			x := s
			st.coverSlot(&cov, r, &x)
			if x.why == slotOK {
				m := st.cfg.Rooms[r].Modality
				if n[m] == 0 {
					mods = append(mods, m)
				}
				n[m]++
			}
		}
	}
	var parts []string
	for _, m := range mods {
		parts = append(parts, fmt.Sprintf("%d %s", n[m], m))
	}
	switch {
	case len(parts) > 0:
		opens = " · opens " + strings.Join(parts, ", ") + " slots for booking"
	case len(rooms) > 0:
		opens = " · no slot newly bookable (already staffed, or no template slots then)"
	}
	return covers, opens
}
