package sim

import (
	"fmt"
	"slices"
	"strings"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Site reading allocation (V0.2 Slice 4: set_reading_allocation;
// V0.2_DESIGN §5.10, §6.3, §8.1).
//
// An employed salaried radiologist's site allocation — the share of paid
// time committed to this site — is a timeline: the scenario's allocation
// (immutable) and dated versions, each in force from its date (any
// weekday at least NoticeDays ahead) until the next. Versions are state in
// the employment record and follow the roster version rules: op set
// inserts or replaces a pending version, op remove deletes one (its
// predecessor resumes), a version in effect or past never changes, and
// every operation is appended to the allocation log.
//
// The allocation in force on a day sets that day's site reading blocks
// (ReadingBlocks, the existing 100-minute cycle inside the immutable
// shifts) and that day's salary accrual (finance: annual salary ×
// allocation, daily rounding, benefits). Nothing else moves: shifts,
// contrast supervision (presence on site) and technologist booking
// coverage (planOf keeps the scenario allocation's block boundaries in the
// booking segments) are the same whatever the allocation, so a change
// displaces no booking.

// Allocation bounds (§17.2 [proposed]): the published 30% floor to the
// radiologist's whole paid time.
const (
	MinSiteBP = 3_000
	MaxSiteBP = 10_000
)

// allocVersion is one dated allocation in force from From.
type allocVersion struct {
	From   int64
	BP     int64
	Action int // the accepting action's index in the action record
	Batch  int64
	At     simtime.Time
	// view is the radiologist's terms at BP: derived, not state.
	view *EmployeeConfig
}

// allocEntry is one operation on an allocation timeline (op set, remove).
type allocEntry struct {
	From   int64
	Op     string
	BP     int64 // op set
	Action int
	Batch  int64
	At     simtime.Time
}

// allocSpec is a validated set_reading_allocation item, not yet committed.
type allocSpec struct {
	emp  int
	from int64
	op   string
	bp   int64 // op set
	// was is the allocation in force from the date before this item; then
	// the one in force after it (op remove: the predecessor resuming).
	was, then int64
}

// allocAt is the index of the version in force on day (-1: the
// scenario's).
func allocAt(vs []allocVersion, day int64) int {
	for k := len(vs) - 1; k >= 0; k-- {
		if vs[k].From <= day {
			return k
		}
	}
	return -1
}

// siteBP is radiologist i's allocation in force on day.
func (st *state) siteBP(i int, day int64) int64 {
	return st.bpIn(i, st.staff[i].Allocs, day)
}

// bpIn is employee i's allocation on day under timeline vs.
func (st *state) bpIn(i int, vs []allocVersion, day int64) int64 {
	if k := allocAt(vs, day); k >= 0 {
		return vs[k].BP
	}
	return st.emps[i].terms.SiteAllocationBP
}

// allocated is working terms t of employee i on day with the allocation
// of timeline vs in force then.
func (st *state) allocated(i int, day int64, vs []allocVersion, t *EmployeeConfig) *EmployeeConfig {
	k := allocAt(vs, day)
	if k < 0 || t == nil {
		return t
	}
	v := &vs[k]
	if v.view == nil {
		c := *t
		c.SiteAllocationBP = v.BP
		v.view = &c
	}
	return v.view
}

// allocsOf is employee i's allocation timeline as the batch so far
// proposes it.
func (p *proposal) allocsOf(st *state, i int) []allocVersion {
	if vs, ok := p.aversions[i]; ok {
		return vs
	}
	return st.staff[i].Allocs
}

// withAlloc is vs with a set (bp > 0) or removal at from, in date order,
// as a new slice.
func withAlloc(vs []allocVersion, from, bp int64) []allocVersion {
	out := make([]allocVersion, 0, len(vs)+1)
	done := false
	for _, v := range vs {
		switch {
		case v.From == from:
			continue
		case v.From > from && !done && bp > 0:
			out = append(out, allocVersion{From: from, BP: bp})
			done = true
		}
		out = append(out, allocVersion{From: v.From, BP: v.BP, Action: v.Action, Batch: v.Batch, At: v.At})
	}
	if !done && bp > 0 {
		out = append(out, allocVersion{From: from, BP: bp})
	}
	return out
}

// isReader reports whether terms t have a site reading allocation: an
// employed salaried radiologist.
func isReader(t *EmployeeConfig) bool { return t.Role == Radiologist && t.Pay == Salaried }

// checkSetAllocation validates one set_reading_allocation item against the
// state and the batch's earlier items.
func (st *state) checkSetAllocation(a api.Action, p *proposal) staffOutcome {
	bad := func(code, format string, args ...any) staffOutcome {
		return staffOutcome{code: code, msg: fmt.Sprintf(format, args...)}
	}
	switch {
	case a.Minutes != 0 || a.Date != "" || a.Policy != "" || a.Cutoff != "" || a.RequestID != "" || a.Role != "" ||
		a.Skills != nil || a.RateCents != 0 || a.ContractedWeeklyMinutes != 0 || a.Roster != nil || a.StartDate != "" || a.LastDay != "":
		return bad("invalid_schema", "set_reading_allocation takes op, employee, effective and (op set) siteAllocationBP only.")
	case a.Op != api.RosterOpSet && a.Op != api.RosterOpRemove:
		return bad("invalid_schema", "set_reading_allocation needs op %q or %q.", api.RosterOpSet, api.RosterOpRemove)
	case a.Employee == "" || a.Effective == "":
		return bad("invalid_schema", "set_reading_allocation needs employee and effective.")
	case a.Op == api.RosterOpSet && a.SiteAllocationBP == 0:
		return bad("invalid_schema", "set_reading_allocation op set needs siteAllocationBP.")
	case a.Op == api.RosterOpRemove && a.SiteAllocationBP != 0:
		return bad("invalid_schema", "set_reading_allocation op remove takes no siteAllocationBP.")
	}
	i := st.findEmployee(a.Employee)
	if i < 0 {
		return bad("unknown_employee", "No employee %q.", a.Employee)
	}
	r := &st.staff[i]
	t := st.emps[i].terms
	label := st.staffLabel(i)
	if !isReader(t) {
		return bad("wrong_role", "Only employed salaried radiologists have a site reading allocation (%s is %s).", label,
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
	switch {
	case from <= st.day:
		return bad("past_date", "effective %s is not after today: an allocation in force or past cannot change.", a.Effective)
	case from < st.day+NoticeDays:
		return bad("notice_too_short", "An allocation change needs %d days' notice: the earliest effective date is %s.", NoticeDays,
			st.cal.Date(st.cal.StartOfDay(st.day+NoticeDays)))
	case from > st.day+st.cfg.Policies.BookingHorizonDays:
		return bad("beyond_horizon", "effective %s is beyond the %d-day booking horizon.", a.Effective, st.cfg.Policies.BookingHorizonDays)
	case a.Op == api.RosterOpSet && (a.SiteAllocationBP < MinSiteBP || a.SiteAllocationBP > MaxSiteBP):
		return bad("allocation_out_of_band", "siteAllocationBP must be %d–%d (%s–%s of paid time), not %d.", MinSiteBP, MaxSiteBP,
			pct(MinSiteBP), pct(MaxSiteBP), a.SiteAllocationBP)
	}
	key := rosterKey{i, from}
	if p.akeys[key] {
		return bad("invalid_parameters", "%s has two allocation changes effective %s in this batch.", label, st.dayLabel(from))
	}
	p.akeys[key] = true // a no-op still claims its date
	vs := p.allocsOf(st, i)
	k := slices.IndexFunc(vs, func(v allocVersion) bool { return v.From == from })
	if a.Op == api.RosterOpRemove {
		if k < 0 {
			return staffOutcome{ok: true, code: api.ActionUnchanged,
				msg: fmt.Sprintf("%s has no allocation change effective %s: nothing changed.", label, st.dayLabel(from))}
		}
		as := &allocSpec{emp: i, from: from, op: rosterRemove, was: vs[k].BP}
		p.addAlloc(st, as)
		as.then = st.bpIn(i, p.aversions[i], from)
		return staffOutcome{ok: true, code: api.ActionAccepted, alloc: as, effective: date.String(), msg: st.allocText(as, false)}
	}
	cur := st.bpIn(i, vs, from)
	if cur == a.SiteAllocationBP {
		return staffOutcome{ok: true, code: api.ActionUnchanged,
			msg: fmt.Sprintf("%s's site allocation from %s is already %s: nothing changed.", label, st.dayLabel(from), pct(cur))}
	}
	as := &allocSpec{emp: i, from: from, op: rosterSet, bp: a.SiteAllocationBP, was: cur, then: a.SiteAllocationBP}
	p.addAlloc(st, as)
	return staffOutcome{ok: true, code: api.ActionAccepted, alloc: as, effective: date.String(), msg: st.allocText(as, false)}
}

func (p *proposal) addAlloc(st *state, as *allocSpec) {
	p.aversions[as.emp] = withAlloc(p.allocsOf(st, as.emp), as.from, as.bp)
	p.allocs = append(p.allocs, as)
}

// commitAlloc records the version (or removal) in the timeline and the
// allocation log. Nothing is reconciled: no booking depends on reading.
func (st *state) commitAlloc(as *allocSpec, action int) {
	r := &st.staff[as.emp]
	st.emit(st.now, api.Info, "actions", st.allocText(as, true))
	r.AllocLog = append(r.AllocLog, allocEntry{From: as.from, Op: as.op, BP: as.bp, Action: action, Batch: st.actions.batches, At: st.now})
	vs := withAlloc(r.Allocs, as.from, as.bp)
	for k := range vs {
		if vs[k].From == as.from {
			vs[k].Action, vs[k].Batch, vs[k].At = action, st.actions.batches, st.now
		}
	}
	r.Allocs = vs
}

// pct formats basis points as a percentage: "62.5%".
func pct(bp int64) string {
	s := fmt.Sprintf("%d", bp/100)
	if f := bp % 100; f != 0 {
		s += strings.TrimRight(fmt.Sprintf(".%02d", f), "0")
	}
	return s + "%"
}

// siteWeek is the site reading time of terms t at allocation bp in rota
// weeks A and B.
func siteWeek(t *EmployeeConfig, bp int64) (a, b simtime.Duration) {
	for wd := range 7 {
		for k, wk := range []int{WeekA, WeekB} {
			x := t.ShiftOn(simtime.Weekday(wd), wk)
			if x == nil {
				continue
			}
			for _, blk := range ReadingBlocks(x.Span, bp) {
				if k == 0 {
					a += blk.To - blk.From
				} else {
					b += blk.To - blk.From
				}
			}
		}
	}
	return a, b
}

// salary30 is radiologist i's salary with benefits per 30 days at bp.
func (st *state) salary30(i int, bp int64) int64 {
	w := st.emps[i].terms.RateCents * bp / 10_000 * 30 / 365
	return w + roundDiv(w*st.cfg.Finance.BenefitsBP, 10_000)
}

// allocText describes an allocation change (done: once committed).
func (st *state) allocText(as *allocSpec, done bool) string {
	i := as.emp
	t := st.emps[i].terms
	label := st.staffLabel(i)
	week := func(bp int64) string {
		a, b := siteWeek(t, bp)
		if a == b {
			return hours(a) + " a week"
		}
		return fmt.Sprintf("%s (week A), %s (week B)", hours(a), hours(b))
	}
	money30 := func(bp int64) string { return money(st.salary30(i, bp)) }
	if as.op == rosterRemove {
		head := "Allocation change withdrawn"
		if !done {
			head = "Allocation change withdrawal"
		}
		return fmt.Sprintf("%s: %s's change to %s effective %s · the allocation before it resumes (%s: site reading %s; salary with benefits %s per 30 days) · shifts and contrast supervision unchanged.",
			head, label, pct(as.was), st.dayLabel(as.from), pct(as.then), week(as.then), money30(as.then))
	}
	head := "Allocation change"
	if done {
		head = "Allocation changed"
	}
	d := st.salary30(i, as.then) - st.salary30(i, as.was)
	sign := "+"
	if d < 0 {
		sign, d = "−", -d
	}
	return fmt.Sprintf("%s: %s from %s: site share %s (was %s) · site reading %s (was %s) · salary with benefits %s per 30 days (%s%s) · shifts and contrast supervision unchanged.",
		head, label, st.dayLabel(as.from), pct(as.then), pct(as.was), week(as.then), week(as.was), money30(as.then), sign, money(d))
}

// ReadingAllocations is every employed salaried radiologist's allocation
// timeline (V0.2 Slice 4): the scenario allocation and every version,
// with site reading time and salary per version, today's marked.
func (w *World) ReadingAllocations() []api.ReadingAllocation {
	st := w.st
	today := st.cal.DayIndex(w.eng.Now())
	date := func(d int64) string { return st.cal.Date(st.cal.StartOfDay(d)).String() }
	var out []api.ReadingAllocation
	for i := range st.staff {
		t := st.emps[i].terms
		if !isReader(t) {
			continue
		}
		r := &st.staff[i]
		x := api.ReadingAllocation{Employee: t.ID, Name: t.Name, AnnualSalaryCents: t.RateCents, InEffectBP: st.siteBP(i, today),
			MinBP: MinSiteBP, MaxBP: MaxSiteBP}
		if r.Until > today+NoticeDays {
			x.Editable, x.EarliestEffective = true, date(today+NoticeDays)
		}
		cur := allocAt(r.Allocs, today)
		add := func(k int, bp int64) {
			a, b := siteWeek(t, bp)
			v := api.AllocationVersion{SiteBP: bp, SiteMinA: int64(a / simtime.Minute), SiteMinB: int64(b / simtime.Minute),
				Salary30Cents: st.salary30(i, bp), InEffect: k == cur}
			if k >= 0 {
				av := &r.Allocs[k]
				v.Effective, v.Pending, v.Action = date(av.From), av.From > today, actionID(av.Action)
			}
			x.Versions = append(x.Versions, v)
		}
		add(-1, t.SiteAllocationBP)
		for k := range r.Allocs {
			add(k, r.Allocs[k].BP)
		}
		out = append(out, x)
	}
	return out
}

// observeAllocations lists employee i's pending allocation versions.
func (st *state) observeAllocations(x *api.Employee, i int, today int64) {
	for _, v := range st.staff[i].Allocs {
		if v.From > today {
			x.AllocationChanges = append(x.AllocationChanges, api.AllocationChange{Effective: st.cal.Date(st.cal.StartOfDay(v.From)).String(),
				SiteBP: v.BP, Action: actionID(v.Action)})
		}
	}
}

// checkAllocations verifies the allocation timelines: versions of
// employed salaried radiologists only, strictly in date order, within the
// bounds, each recorded by a set in the log.
func (st *state) checkAllocations() error {
	for i := range st.staff {
		r := &st.staff[i]
		for k, v := range r.Allocs {
			switch {
			case !isReader(st.emps[i].terms):
				return fmt.Errorf("%s: an allocation version for a %s", r.ID, st.emps[i].terms.Role)
			case k > 0 && v.From <= r.Allocs[k-1].From:
				return fmt.Errorf("%s: allocation versions out of order", r.ID)
			case v.BP < MinSiteBP || v.BP > MaxSiteBP:
				return fmt.Errorf("%s: allocation %d out of bounds", r.ID, v.BP)
			case !slices.ContainsFunc(r.AllocLog, func(e allocEntry) bool {
				return e.Op == rosterSet && e.From == v.From && e.BP == v.BP && e.Action == v.Action
			}):
				return fmt.Errorf("%s: allocation version on day %d is not in the log", r.ID, v.From)
			}
		}
	}
	return nil
}
