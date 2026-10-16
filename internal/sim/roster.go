package sim

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Dated rosters (V0.2 Slice 3: set_roster; V0.2_DESIGN §5.8, §6.3–§6.4).
//
// An hourly technologist's or front desk employee's roster is a timeline:
// the original roster (the scenario's, or the hire's, immutable) and dated
// versions. A version is a complete A/B pattern effective from a Monday
// at least NoticeDays ahead; it replaces its predecessor from that day
// until the next version. Versions are state (the employment record):
// op set inserts a version or replaces a pending one on the same date, op
// remove deletes a pending one (its predecessor resumes). A version in
// effect, or past, is never changed. Every operation is also appended to
// the employee's roster log, so replaced and removed versions stay on
// record. A termination voids the versions effective on or after the end.
//
// The contract never changes with the roster: contracted weekly time is
// paid whatever is rostered (the guarantee, employment.go), so a cut or
// zero-hour roster saves nothing and leaves notice and severance as they
// were. A roster may not exceed the contract in either rota week.
//
// Rest rule (one rule for hiring, roster changes and overtime): at least
// MinRest between one day's shift cap — the latest permitted release,
// min(start + maxShift, 23:59:59), which bounds the scheduled end,
// approved extensions and hold-over alike — and the next day's start.
// Because no approved overtime or hold-over can pass the cap, no approval
// can break the rule and extend_shift needs no rest check of its own.

// rosterVersion is one dated roster in effect from From (a Monday).
type rosterVersion struct {
	From   int64
	Terms  *EmployeeConfig // the employee's terms with this roster
	Action int             // the accepting action's index in the action record
	Batch  int64
	At     simtime.Time
	// trainView is the training-day view of Terms (role Trainee): derived,
	// not state.
	trainView *EmployeeConfig
}

// Roster log operations.
const (
	rosterSet    = "set"
	rosterRemove = "remove"
	rosterVoided = "voided" // by a termination ending employment first
)

// rosterEntry is one operation on an employee's roster timeline.
type rosterEntry struct {
	From   int64
	Op     string
	Terms  *EmployeeConfig // op set: the version's terms
	Action int
	Batch  int64
	At     simtime.Time
}

// Displacement causes (displace.go) of a roster change.
const (
	causeRosterChanged   = "roster_changed"
	causeTrainingDelayed = "training_delayed"
)

// versionAt is the index of the version in effect on day (-1: none, the
// original roster).
func versionAt(vs []rosterVersion, day int64) int {
	for k := len(vs) - 1; k >= 0; k-- {
		if vs[k].From <= day {
			return k
		}
	}
	return -1
}

// termsOn is employee i's terms with the roster in effect on day.
func (st *state) termsOn(i int, day int64) *EmployeeConfig {
	vs := st.staff[i].Rosters
	if k := versionAt(vs, day); k >= 0 {
		return vs[k].Terms
	}
	return st.emps[i].terms
}

// workingView is employee i's working terms on day under a roster
// timeline vs, productive date and end: nil when not employed, the
// training view before the productive date.
func (st *state) workingView(i int, day int64, vs []rosterVersion, productive, until int64) *EmployeeConfig {
	r := &st.staff[i]
	if day < r.From || day >= until {
		return nil
	}
	k := versionAt(vs, day)
	if day >= productive {
		if k >= 0 {
			return vs[k].Terms
		}
		return st.emps[i].terms
	}
	view := &r.trainView
	t := st.emps[i].terms
	if k >= 0 {
		view, t = &vs[k].trainView, vs[k].Terms
	}
	if *view == nil {
		v := *t
		v.Role, v.Skills = Trainee, nil
		*view = &v
	}
	return *view
}

// rosterSpec is a validated set_roster item, not yet committed.
type rosterSpec struct {
	emp   int
	from  int64
	op    string
	terms *EmployeeConfig // op set
	item  int             // its index in the batch
	// was is the roster in force from the date before this item; then the
	// one in force after it (op remove: the predecessor resuming).
	was, then *EmployeeConfig
}

type rosterKey struct {
	emp  int
	from int64
}

// rostered is a roster's net time (breaks excluded) in rota weeks A, B.
func rostered(t *EmployeeConfig) (a, b simtime.Duration) {
	for wd := range 7 {
		if x := t.ShiftOn(simtime.Weekday(wd), WeekA); x != nil {
			a += netTime(x)
		}
		if x := t.ShiftOn(simtime.Weekday(wd), WeekB); x != nil {
			b += netTime(x)
		}
	}
	return a, b
}

// sameRoster reports whether two terms roster the same shifts in both
// rota weeks (covers compared as sets).
func sameRoster(a, b *EmployeeConfig) bool {
	for _, wk := range []int{WeekA, WeekB} {
		for wd := range 7 {
			x, y := a.ShiftOn(simtime.Weekday(wd), wk), b.ShiftOn(simtime.Weekday(wd), wk)
			if (x == nil) != (y == nil) {
				return false
			}
			if x != nil && !(x.Span == y.Span && x.Assignment == y.Assignment && x.Remote == y.Remote &&
				slices.Equal(x.Breaks, y.Breaks) && sameCovers(x.Covers, y.Covers)) {
				return false
			}
		}
	}
	return true
}

func sameCovers(a, b []Cover) bool {
	if len(a) != len(b) {
		return false
	}
	order := func(x, y Cover) int {
		if x.Room != y.Room {
			return strings.Compare(x.Room, y.Room)
		}
		return int(x.From - y.From)
	}
	a, b = slices.Clone(a), slices.Clone(b)
	slices.SortFunc(a, order)
	slices.SortFunc(b, order)
	return slices.Equal(a, b)
}

// versionsOf is employee i's timeline as the batch so far proposes it.
func (p *proposal) versionsOf(st *state, i int) []rosterVersion {
	if vs, ok := p.versions[i]; ok {
		return vs
	}
	return st.staff[i].Rosters
}

// untilOf is employee i's end as the batch so far proposes it.
func (p *proposal) untilOf(st *state, i int) int64 {
	for _, t := range p.terms {
		if t.emp == i {
			return t.lastDay + 1
		}
	}
	return st.staff[i].Until
}

// withVersion is vs with a set (terms non-nil) or removal at from, in
// date order, as a new slice.
func withVersion(vs []rosterVersion, from int64, terms *EmployeeConfig) []rosterVersion {
	out := make([]rosterVersion, 0, len(vs)+1)
	done := false
	for _, v := range vs {
		switch {
		case v.From == from:
			continue
		case v.From > from && !done && terms != nil:
			out = append(out, rosterVersion{From: from, Terms: terms})
			done = true
		}
		out = append(out, rosterVersion{From: v.From, Terms: v.Terms, Action: v.Action, Batch: v.Batch, At: v.At})
	}
	if !done && terms != nil {
		out = append(out, rosterVersion{From: from, Terms: terms})
	}
	return out
}

// checkSetRoster validates one set_roster item against the state and the
// batch's earlier items (the whole batch is checked again once composed:
// checkComposed).
func (st *state) checkSetRoster(a api.Action, p *proposal, item int) staffOutcome {
	bad := func(code, format string, args ...any) staffOutcome {
		return staffOutcome{code: code, msg: fmt.Sprintf(format, args...)}
	}
	switch {
	case a.Minutes != 0 || a.Date != "" || a.Policy != "" || a.Cutoff != "" || a.RequestID != "" || a.Role != "" ||
		a.Skills != nil || a.RateCents != 0 || a.ContractedWeeklyMinutes != 0 || a.StartDate != "" || a.LastDay != "":
		return bad("invalid_schema", "set_roster takes op, employee, effective and (op set) roster only.")
	case a.Op != api.RosterOpSet && a.Op != api.RosterOpRemove:
		return bad("invalid_schema", "set_roster needs op %q or %q.", api.RosterOpSet, api.RosterOpRemove)
	case a.Employee == "" || a.Effective == "":
		return bad("invalid_schema", "set_roster needs employee and effective.")
	case a.Op == api.RosterOpSet && a.Roster == nil:
		return bad("invalid_schema", "set_roster op set needs a roster.")
	case a.Op == api.RosterOpRemove && a.Roster != nil:
		return bad("invalid_schema", "set_roster op remove takes no roster.")
	}
	i := st.findEmployee(a.Employee)
	if i < 0 {
		return bad("unknown_employee", "No employee %q.", a.Employee)
	}
	r := &st.staff[i]
	t := st.emps[i].terms
	label := st.staffLabel(i)
	if t.Pay != Hourly || (t.Role != Technologist && t.Role != FrontDesk) {
		return bad("wrong_role", "Only hourly technologists and front desk staff have editable rosters (%s is %s).", label,
			strings.ReplaceAll(string(t.Role), "_", " "))
	}
	if r.Until <= st.day {
		return bad("not_employed", "%s left the practice on %s.", label, st.cal.Date(st.cal.StartOfDay(r.Until-1)))
	}
	date, err := simtime.ParseDate(a.Effective)
	if err != nil {
		return bad("invalid_date", "effective %q is not YYYY-MM-DD.", a.Effective)
	}
	from := st.cal.DayIndex(st.cal.StartOfDate(date))
	earliest := st.earliestEffective(st.day)
	switch {
	case from <= st.day:
		return bad("past_date", "effective %s is not after today: earlier roster history cannot change.", a.Effective)
	case date.Weekday() != simtime.Monday:
		return bad("not_monday", "A roster change takes effect on a Monday (%s is a %s).", a.Effective, date.Weekday())
	case from < st.day+NoticeDays:
		return bad("notice_too_short", "A roster change needs %d days' notice: the earliest effective Monday is %s.", NoticeDays,
			st.cal.Date(st.cal.StartOfDay(earliest)))
	case from > st.day+st.cfg.Policies.BookingHorizonDays:
		return bad("beyond_horizon", "effective %s is beyond the %d-day booking horizon.", a.Effective, st.cfg.Policies.BookingHorizonDays)
	}
	if u := p.untilOf(st, i); from >= u {
		return bad("not_employed", "%s's last day is %s: a roster from %s would never be worked.", label,
			st.dayLabel(u-1), st.dayLabel(from))
	}
	key := rosterKey{i, from}
	if p.rkeys[key] {
		return bad("invalid_parameters", "%s has two roster changes effective %s in this batch.", label, st.dayLabel(from))
	}
	p.rkeys[key] = true // a no-op still claims its date
	vs := p.versionsOf(st, i)
	k := slices.IndexFunc(vs, func(v rosterVersion) bool { return v.From == from })
	if a.Op == api.RosterOpRemove {
		if k < 0 {
			return staffOutcome{ok: true, code: api.ActionUnchanged,
				msg: fmt.Sprintf("%s has no roster change effective %s: nothing changed.", label, st.dayLabel(from))}
		}
		rs := &rosterSpec{emp: i, from: from, op: rosterRemove, item: item, was: vs[k].Terms}
		p.addRoster(st, rs)
		rs.then = t
		if j := versionAt(p.versions[i], from); j >= 0 {
			rs.then = p.versions[i][j].Terms
		}
		return staffOutcome{ok: true, code: api.ActionAccepted, roster: rs, effective: a.Effective, msg: st.rosterText(rs, false)}
	}
	nt := *t
	nt.Roster, nt.Alt = [7]*ShiftTemplate{}, [7]*ShiftTemplate{}
	nt.Contract = r.Contract
	if err := st.parseRoster(&nt, a.Roster); err != nil {
		return bad("invalid_roster", "%v", err)
	}
	cur := t
	if j := versionAt(vs, from); j >= 0 {
		cur = vs[j].Terms
	}
	// The roster already in force then is a no-op, before the whole-roster
	// rules: an unedited legacy roster need not meet them (§6.4).
	if sameRoster(cur, &nt) {
		return staffOutcome{ok: true, code: api.ActionUnchanged,
			msg: fmt.Sprintf("%s's roster from %s is already this one: nothing changed.", label, st.dayLabel(from))}
	}
	// The employee's own original roster may always be restored: an
	// unedited legacy roster need not meet the rules either (a scenario
	// roster above its inferred contract in one rota week stays lawful).
	if !sameRoster(t, &nt) {
		if err := st.rosterRules(&nt); err != nil {
			return bad("invalid_roster", "%v", err)
		}
	}
	rs := &rosterSpec{emp: i, from: from, op: rosterSet, terms: &nt, item: item, was: cur, then: &nt}
	p.addRoster(st, rs)
	return staffOutcome{ok: true, code: api.ActionAccepted, roster: rs, effective: a.Effective, msg: st.rosterText(rs, false)}
}

func (p *proposal) addRoster(st *state, rs *rosterSpec) {
	p.versions[rs.emp] = withVersion(p.versionsOf(st, rs.emp), rs.from, rs.terms)
	p.rosters = append(p.rosters, rs)
}

// earliestEffective is the first Monday a roster change decided on day
// today may take effect.
func (st *state) earliestEffective(today int64) int64 {
	d := today + NoticeDays
	wd := st.cal.Date(st.cal.StartOfDay(d)).Weekday()
	return d + int64((8-int(wd))%7)
}

// checkComposed checks the batch's composed roster timelines (V0.2_DESIGN
// §6.4), employee by employee in the order of their first item: no
// version after the end of employment (a termination in the same batch
// included); the rest rule across every future version boundary; a
// trainee's training still complete within its window (the productive
// date is recomputed into p); and every approved extension of a changed
// employee still valid on the new shift. It returns the first failing
// item and its outcome (-1: none).
func (st *state) checkComposed(p *proposal) (int, staffOutcome) {
	bad := func(rs *rosterSpec, code, format string, args ...any) (int, staffOutcome) {
		return rs.item, staffOutcome{code: code, msg: fmt.Sprintf(format, args...)}
	}
	var order []int
	first := map[int]*rosterSpec{}
	for _, rs := range p.rosters {
		if first[rs.emp] == nil {
			first[rs.emp] = rs
			order = append(order, rs.emp)
		}
	}
	for _, i := range order {
		r := &st.staff[i]
		label := st.staffLabel(i)
		vs := p.versions[i]
		until := p.untilOf(st, i)
		itemAt := func(day int64) *rosterSpec { // the batch's item governing day
			var best *rosterSpec
			for _, rs := range p.rosters {
				if rs.emp == i && rs.from <= day && (best == nil || rs.from > best.from) {
					best = rs
				}
			}
			if best == nil {
				best = first[i]
			}
			return best
		}
		for _, rs := range p.rosters {
			if rs.emp == i && rs.op == rosterSet && rs.from >= until {
				return bad(rs, "not_employed", "%s's last day is %s: a roster from %s would never be worked.", label, st.dayLabel(until-1), st.dayLabel(rs.from))
			}
		}
		// Rest across each future version boundary. (A removal leaves no
		// boundary of its own: the predecessor resumes, and the next
		// boundary is checked here.)
		for _, v := range vs {
			b := v.From
			if b <= st.day || b-1 < r.From || b >= until {
				continue
			}
			x, y := st.shiftIn(i, vs, b-1), st.shiftIn(i, vs, b)
			if x == nil || y == nil {
				continue
			}
			if rest := simtime.Day + y.From - st.cfg.Policies.ShiftCap(x.From); rest < MinRest {
				return bad(itemAt(b), "invalid_roster", "roster from %s: only %s between the latest permitted end of %s's shift (%s, the cap of %s–%s) and the next start (%s); at least %s",
					st.dayLabel(b), rest, st.dayLabel(b-1), simtime.FormatClock(st.cfg.Policies.ShiftCap(x.From)), simtime.FormatClock(x.From),
					simtime.FormatClock(x.To), simtime.FormatClock(y.From), MinRest)
			}
		}
		// Training: a hire not yet independent.
		if r.Hire != nil && r.Productive > st.day && r.Productive < noEnd {
			last, ok := st.projectTraining(i, vs)
			if !ok {
				return bad(first[i], "training_incomplete", "With this roster %s would attend fewer than %s of training within %d days of the start (%s): no independent work would ever begin.",
					label, hours(r.Hire.Training), TrainingWindowDays, st.dayLabel(r.From))
			}
			p.productive[i] = last + 1
		}
		// Approved extensions of the changed employee on the new shifts.
		prod := r.Productive
		if np, ok := p.productive[i]; ok {
			prod = np
		}
		days := make([]int64, 0)
		for d, m := range st.ext {
			if d > st.day && m[i] > 0 {
				days = append(days, d)
			}
		}
		slices.Sort(days)
		for _, d := range days {
			if d >= until {
				continue // voided by the termination
			}
			if why := st.extensionHolds(i, d, st.ext[d][i], st.workingView(i, d, vs, prod, until)); why != "" {
				ext := st.ext[d][i]
				return bad(itemAt(d), "extension_conflict", "%s has an approved extension on %s (+%s): %s. An approval is never cancelled by a roster change: keep a shift that carries it, or take effect after it.",
					label, st.dayLabel(d), ext, why)
			}
		}
	}
	return -1, staffOutcome{}
}

// shiftIn is employee i's rostered shift on day under timeline vs (nil:
// off), whatever the service stage.
func (st *state) shiftIn(i int, vs []rosterVersion, day int64) *ShiftTemplate {
	t := st.emps[i].terms
	if k := versionAt(vs, day); k >= 0 {
		t = vs[k].Terms
	}
	date := st.cal.Date(st.cal.StartOfDay(day))
	return t.ShiftOn(date.Weekday(), st.cfg.RotaWeek(date))
}

// projectTraining is a hire's last training day under timeline vs: the
// first day from the start by which the rostered time, net of breaks,
// reaches the requirement (attended exactly as rostered, so days already
// past are the attended ones). ok is false past the training window.
func (st *state) projectTraining(i int, vs []rosterVersion) (int64, bool) {
	r := &st.staff[i]
	var got simtime.Duration
	for d := r.From; d < r.From+TrainingWindowDays; d++ {
		if tpl := st.shiftIn(i, vs, d); tpl != nil {
			if got += netTime(tpl); got >= r.Hire.Training {
				return d, true
			}
		}
	}
	return 0, false
}

// extensionHolds re-checks an approved extension of employee i on day
// against working terms t (the composed roster): extend_shift's own
// conditions — a shift that day, the extension within its cap, before the
// close, covering a room, the desk or supervision — on the new shift. It
// returns why it no longer holds ("" when it does).
func (st *state) extensionHolds(i int, day int64, d simtime.Duration, t *EmployeeConfig) string {
	if t == nil {
		return "not employed that day"
	}
	emps := make([]*EmployeeConfig, len(st.staff))
	emps[i] = t
	plan := st.cfg.planOf(st.cal.Date(st.cal.StartOfDay(day)), emps)
	k := plan.ByEmp[i]
	if k < 0 {
		return "the new roster has no shift that day"
	}
	sh := plan.Shifts[k]
	switch capAt := st.cfg.Policies.ShiftCap(sh.From); {
	case sh.To+d > capAt:
		return fmt.Sprintf("the new shift %s would run past its cap %s", span(sh.Span), simtime.FormatClock(capAt))
	case plan.Hours == nil || sh.To >= plan.Hours.To:
		return fmt.Sprintf("the new shift %s already runs to the close", span(sh.Span))
	}
	if extensionCovers(plan, k, d, st.cfg.Rooms) == "" {
		return "in training then, or covering nothing in the extra time"
	}
	return ""
}

// commitRoster records the version (or removal) in the timeline and the
// roster log.
func (st *state) commitRoster(rs *rosterSpec, action int) {
	r := &st.staff[rs.emp]
	st.emit(st.now, api.Info, "actions", st.rosterText(rs, true))
	r.RosterLog = append(r.RosterLog, rosterEntry{From: rs.from, Op: rs.op, Terms: rs.terms, Action: action, Batch: st.actions.batches, At: st.now})
	vs := withVersion(r.Rosters, rs.from, rs.terms)
	for k := range vs {
		if vs[k].From == rs.from {
			vs[k].Action, vs[k].Batch, vs[k].At = action, st.actions.batches, st.now
		}
	}
	r.Rosters = vs
}

// rosterText describes a roster change (done: once committed).
func (st *state) rosterText(rs *rosterSpec, done bool) string {
	i := rs.emp
	r := &st.staff[i]
	head := "Roster change"
	if done {
		head = "Roster changed"
	}
	label := st.staffLabel(i)
	week := func(a, b simtime.Duration) string {
		if a == b {
			return hours(a) + " a week"
		}
		return fmt.Sprintf("%s (week A), %s (week B)", hours(a), hours(b))
	}
	na, nb := rostered(rs.then)
	idle := ""
	if c := r.Contract.Weekly; na < c || nb < c {
		idle = " · contracted time not rostered is still paid (" + week(max(0, c-na), max(0, c-nb)) + ")"
	}
	if rs.op == rosterRemove {
		return fmt.Sprintf("%s: %s's change effective %s withdrawn · the roster before it resumes (%s) · contract %s unchanged%s.",
			head, label, st.dayLabel(rs.from), week(na, nb), hours(r.Contract.Weekly), idle)
	}
	pa, pb := rostered(rs.was)
	return fmt.Sprintf("%s: %s from %s: rostered %s (was %s) · contract %s unchanged%s.", head, label, st.dayLabel(rs.from),
		week(na, nb), week(pa, pb), hours(r.Contract.Weekly), idle)
}

// voidRosters voids employee i's versions effective on or after its end
// (a termination), with a log line each.
func (st *state) voidRosters(i int, action int) {
	r := &st.staff[i]
	var keep []rosterVersion
	for _, v := range r.Rosters {
		if v.From < r.Until {
			keep = append(keep, v)
			continue
		}
		r.RosterLog = append(r.RosterLog, rosterEntry{From: v.From, Op: rosterVoided, Action: action, Batch: st.actions.batches, At: st.now})
		st.emit(st.now, api.Info, "actions", fmt.Sprintf("Roster change voided: %s from %s: after the last day", st.staffLabel(i), st.dayLabel(v.From)))
	}
	r.Rosters = keep
}

// trainingMoves recomputes the productive date of every hire in training
// whose roster the batch changed; it returns the earliest day whose
// capacity changes and whether any training was delayed.
func (st *state) trainingMoves(p *proposal) (from int64, delayed bool) {
	from = noEnd
	for _, i := range slices.Sorted(maps.Keys(p.productive)) {
		r := &st.staff[i]
		np := p.productive[i]
		if np == r.Productive {
			continue
		}
		old := r.Productive
		r.Productive = np
		r.Stage = r.stageOn(st.day)
		from = min(from, old, np)
		verb := "earlier"
		if np > old {
			verb, delayed = "later", true
		}
		st.emit(st.now, api.Info, "staff", fmt.Sprintf("%s's training now completes %s: independent from %s (was %s)", st.staffLabel(i), verb,
			st.dayLabel(np), st.dayLabel(old)))
	}
	return from, delayed
}

// rosterOf is a terms' roster in the action's A/B shape.
func rosterOf(t *EmployeeConfig) api.Roster {
	var r api.Roster
	for k, wk := range []int{WeekA, WeekB} {
		w := &r.A
		if k == 1 {
			w = &r.B
		}
		days := []**api.Shift{&w.Sun, &w.Mon, &w.Tue, &w.Wed, &w.Thu, &w.Fri, &w.Sat} // by simtime.Weekday
		for wd := range 7 {
			x := t.ShiftOn(simtime.Weekday(wd), wk)
			if x == nil {
				continue
			}
			s := &api.Shift{Start: simtime.FormatClock(x.From), End: simtime.FormatClock(x.To), Breaks: [][2]string{},
				Assignment: x.Assignment, Covers: []api.Cover{}}
			for _, b := range x.Breaks {
				s.Breaks = append(s.Breaks, [2]string{simtime.FormatClock(b.From), simtime.FormatClock(b.To)})
			}
			for _, c := range x.Covers {
				s.Covers = append(s.Covers, api.Cover{Room: c.Room, From: simtime.FormatClock(c.From), Until: simtime.FormatClock(c.To)})
			}
			*days[wd] = s
		}
	}
	return r
}

// RosterPlan is employee id's roster timeline (V0.2 Slice 3): the original
// roster and every version, with today's and the pending ones marked.
func (w *World) RosterPlan(id string) (api.RosterPlan, bool) {
	st := w.st
	i := st.findEmployee(id)
	if i < 0 {
		return api.RosterPlan{}, false
	}
	today := st.cal.DayIndex(w.eng.Now())
	r := &st.staff[i]
	t := st.emps[i].terms
	out := api.RosterPlan{Employee: t.ID, Name: t.Name, Role: string(t.Role), Skills: slices.Clone(t.Skills),
		ContractMin: int64(r.Contract.Weekly / simtime.Minute)}
	if t.Pay == Hourly && (t.Role == Technologist || t.Role == FrontDesk) && r.Until > today {
		if e := st.earliestEffective(today); e < r.Until {
			out.Editable, out.EarliestEffective = true, st.cal.Date(st.cal.StartOfDay(e)).String()
		}
		if t.Role == FrontDesk {
			out.Assignments = []string{AssignDesk}
		} else {
			for _, rc := range st.cfg.Rooms {
				if t.HasSkill(rc.Modality) {
					out.Assignments = append(out.Assignments, rc.ID)
				}
			}
		}
	}
	cur := versionAt(r.Rosters, today)
	add := func(k int, terms *EmployeeConfig) {
		a, b := rostered(terms)
		v := api.RosterVersion{Roster: rosterOf(terms), NetMinA: int64(a / simtime.Minute), NetMinB: int64(b / simtime.Minute), InEffect: k == cur}
		if k >= 0 {
			x := &r.Rosters[k]
			v.Effective = st.cal.Date(st.cal.StartOfDay(x.From)).String()
			v.Pending, v.Action = x.From > today, actionID(x.Action)
		}
		out.Versions = append(out.Versions, v)
	}
	add(-1, t)
	for k := range r.Rosters {
		add(k, r.Rosters[k].Terms)
	}
	return out, true
}

// observeRosters lists employee i's pending versions.
func (st *state) observeRosters(x *api.Employee, i int, today int64) {
	for _, v := range st.staff[i].Rosters {
		if v.From > today {
			a, b := rostered(v.Terms)
			x.RosterChanges = append(x.RosterChanges, api.RosterChange{Effective: st.cal.Date(st.cal.StartOfDay(v.From)).String(),
				NetMinA: int64(a / simtime.Minute), NetMinB: int64(b / simtime.Minute), Action: actionID(v.Action)})
		}
	}
}

// checkRosters verifies the roster timelines: versions of hourly
// technologists and front desk staff only, strictly in date order, on
// Mondays, within employment, each recorded by a set in the log.
func (st *state) checkRosters() error {
	for i := range st.staff {
		r := &st.staff[i]
		for k, v := range r.Rosters {
			t := st.emps[i].terms
			switch {
			case t.Pay != Hourly || (t.Role != Technologist && t.Role != FrontDesk):
				return fmt.Errorf("%s: a roster version for a %s", r.ID, t.Role)
			case k > 0 && v.From <= r.Rosters[k-1].From:
				return fmt.Errorf("%s: roster versions out of order", r.ID)
			case st.cal.Date(st.cal.StartOfDay(v.From)).Weekday() != simtime.Monday:
				return fmt.Errorf("%s: roster version on day %d is not a Monday", r.ID, v.From)
			case v.From >= r.Until:
				return fmt.Errorf("%s: roster version on day %d after the end (%d)", r.ID, v.From, r.Until)
			case v.Terms == nil || v.Terms.ID != t.ID || v.Terms.RateCents != t.RateCents || v.Terms.Contract != r.Contract:
				return fmt.Errorf("%s: roster version on day %d changes more than the roster", r.ID, v.From)
			case !slices.ContainsFunc(r.RosterLog, func(e rosterEntry) bool { return e.Op == rosterSet && e.From == v.From && e.Terms == v.Terms }):
				return fmt.Errorf("%s: roster version on day %d is not in the roster log", r.ID, v.From)
			}
		}
	}
	return nil
}
