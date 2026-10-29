package sim

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Staffing previews (V0.2_DESIGN §6.5): read-only. A preview validates the
// batch as Apply would, then plans the proposed registry — hires added,
// terminations' ends set — without committing anything: no ID, posting,
// draw, event, log line or cache that a digest covers changes, so
// previewing any number of times leaves the run exactly as it was.

// hypo is a batch's proposed registry changes.
type hypo struct {
	p       *proposal
	until   map[int]int64     // proposed ends (first day no longer employed)
	touched []int             // existing employees whose working terms change
	views   []*EmployeeConfig // the hires' training views
}

func newHypo(st *state, p *proposal) *hypo {
	h := &hypo{p: p, until: map[int]int64{}}
	seen := map[int]bool{}
	touch := func(i int) {
		if !seen[i] {
			seen[i] = true
			h.touched = append(h.touched, i)
		}
	}
	for _, t := range p.terms {
		h.until[t.emp] = t.lastDay + 1
		touch(t.emp)
	}
	for _, rs := range p.rosters {
		touch(rs.emp)
	}
	for _, as := range p.allocs {
		touch(as.emp)
	}
	for _, sp := range p.hires {
		v := sp.terms
		v.Role, v.Skills = Trainee, nil
		h.views = append(h.views, &v)
	}
	slices.Sort(h.touched)
	return h
}

// planHypo is day's plan under the proposed registry, with approved
// extensions (those of employees who would have left drop out).
func (st *state) planHypo(day int64, h *hypo) DayPlan {
	emps := st.dayTerms(day)
	for _, i := range h.touched {
		r := &st.staff[i]
		until, prod := r.Until, r.Productive
		if u, ok := h.until[i]; ok {
			until = u
		}
		if np, ok := h.p.productive[i]; ok {
			prod = np
		}
		emps[i] = st.allocated(i, day, h.p.allocsOf(st, i), st.workingView(i, day, h.p.versionsOf(st, i), prod, until))
	}
	for k, sp := range h.p.hires {
		var t *EmployeeConfig
		switch {
		case day < sp.from:
		case day < sp.productive:
			t = h.views[k]
		default:
			t = &sp.terms
		}
		emps = append(emps, t)
	}
	p := st.cfg.planOf(st.cal.Date(st.cal.StartOfDay(day)), emps)
	for i, d := range st.ext[day] {
		if k := p.ByEmp[i]; k >= 0 {
			p.Shifts[k].To += d
		}
	}
	return p
}

// PreviewStaffing previews a V0.2 staffing batch now (see
// api.StaffingPreview). A batch without a V0.2 action previews nothing
// but its validation.
func (w *World) PreviewStaffing(actions []api.Action) api.StaffingPreview {
	st := w.st
	prev := st.now
	st.now = w.eng.Now()
	defer func() { st.now = prev }()
	out := api.StaffingPreview{AsOf: st.cal.ISO(st.now), OK: len(actions) > 0, Results: w.ValidateBatch(actions)}
	for _, r := range out.Results {
		out.OK = out.OK && r.OK
	}
	if sv := st.solvency(); sv != nil && slices.ContainsFunc(actions, func(a api.Action) bool { return isV2Action(a.Type) }) {
		// The funding test of the batch (V0.7b): known once every item is
		// otherwise valid.
		if _, p := st.staffBatch(w.eng, actions, false, v2Actions); p.fund != nil {
			out.Funding = &api.Funding{CommitmentCents: p.fund.commit, HeadroomCents: p.fund.headroom,
				Funded: p.fund.commit == 0 || p.fund.commit <= p.fund.headroom, Basis: st.headroomText(sv)}
		}
	}
	if !out.OK || !slices.ContainsFunc(actions, func(a api.Action) bool { return isV2Action(a.Type) }) {
		return out
	}
	res, p := st.staffBatch(w.eng, actions, false, v2Actions)
	h := newHypo(st, p)
	date := func(d int64) string { return st.cal.Date(st.cal.StartOfDay(d)).String() }
	n := int64(len(st.dem.days))

	// Plans from the first day anything changes.
	first := noEnd
	for _, u := range h.until {
		first = min(first, u)
	}
	for _, sp := range p.hires {
		first = min(first, sp.productive)
	}
	for _, rs := range p.rosters {
		first = min(first, rs.from)
	}
	for i, np := range p.productive {
		first = min(first, np, st.staff[i].Productive)
	}
	for _, as := range p.allocs {
		first = min(first, as.from)
	}
	plans := map[int64]DayPlan{}
	for d := max(first, st.day+1); d < n; d++ {
		plans[d] = st.planHypo(d, h)
	}

	// Hires: terms, timeline and the slots their capacity opens.
	for _, sp := range p.hires {
		t := &sp.terms
		var a, b simtime.Duration
		for wd := range 7 {
			if x := t.ShiftOn(simtime.Weekday(wd), WeekA); x != nil {
				a += netTime(x)
			}
			if x := t.ShiftOn(simtime.Weekday(wd), WeekB); x != nil {
				b += netTime(x)
			}
		}
		hp := api.HirePreview{RequestID: sp.req, Offer: offerLabel(t), Tier: sp.tier, RateCents: t.RateCents, LeadDays: sp.lead,
			ContractMin: int64(t.Contract.Weekly / simtime.Minute), RosterMinA: int64(a / simtime.Minute), RosterMinB: int64(b / simtime.Minute),
			StartDate: date(sp.from), ProductiveDate: date(sp.productive), TrainingMin: int64(sp.training / simtime.Minute),
			TrainingDays: sp.productive - sp.from}
		hp.OpensSlots = st.opensSlots(plans, sp.productive)
		out.Hires = append(out.Hires, hp)
		w := t.RateCents * int64(t.Contract.Weekly) / 3600
		g := w + roundDiv(w*st.cfg.Finance.BenefitsBP, 10_000)
		exit := st.hireExit(sp)
		who := offerLabel(t)
		out.Guaranteed = append(out.Guaranteed,
			api.MoneyLine{Label: "Recruiting fee · " + who, Cents: sp.fee, When: "recognized now, paid " + date(st.firstOfNextMonth()),
				Basis: "fixed offer table; not refundable"},
			api.MoneyLine{Label: "Contract guarantee per full week · " + who, Cents: g, When: "from " + date(sp.from),
				Basis: hours(t.Contract.Weekly) + " × " + rate(t.RateCents) + " with benefits; worked pay replaces it, training included"},
			api.MoneyLine{Label: "Earliest lawful exit · " + who, Cents: exit, When: "last day " + date(sp.from+NoticeDays),
				Basis: "contract guarantee from the start to the earliest last day, plus two contracted weeks' severance with benefits"})
	}

	// Terminations and the bookings they displace.
	var disp []StaffingConflict
	for d := max(first, st.day+1); d < n; d++ {
		if !slices.ContainsFunc(st.dem.days[d].appts, func(ai int) bool { return st.dem.appts[ai].state == apptBooked }) {
			continue
		}
		disp = append(disp, st.survivors(d, plans[d])...)
	}
	for _, o := range res {
		if o.term == nil || o.code != api.ActionAccepted {
			continue
		}
		i, last := o.term.emp, o.term.lastDay
		t := st.emps[i].terms
		tp := api.TerminationPreview{Employee: t.ID, Name: t.Name, LastDay: date(last), EndDate: date(last + 1),
			NoticeDays: last - st.day, SeveranceCents: st.severance(i), SeverancePayday: date(st.nextPayday(last)), Displaced: len(disp)}
		for _, v := range p.versionsOf(st, i) {
			if v.From > last {
				tp.VoidedRosters = append(tp.VoidedRosters, date(v.From))
			}
		}
		var days []int64
		for d := range st.ext {
			if d > last && st.ext[d][i] > 0 {
				days = append(days, d)
			}
		}
		slices.Sort(days)
		for _, d := range days {
			tp.VoidedExtensions = append(tp.VoidedExtensions, api.Extension{Date: date(d), Employee: t.ID, Name: t.Name,
				Minutes: int64(st.ext[d][i] / simtime.Minute)})
		}
		out.Terminations = append(out.Terminations, tp)
		var notice int64
		for d := st.day; d <= last; d++ {
			dd := st.cal.Date(st.cal.StartOfDay(d))
			notice += st.staff[i].Contract.DayCents(t.RateCents, st.cfg.RotaWeek(dd), dd.Weekday())
		}
		notice += roundDiv(notice*st.cfg.Finance.BenefitsBP, 10_000)
		who := st.staffLabel(i)
		out.Guaranteed = append(out.Guaranteed,
			api.MoneyLine{Label: "Severance · " + who, Cents: tp.SeveranceCents, When: "recognized now, paid " + tp.SeverancePayday,
				Basis: "two contracted weeks (" + hours(st.staff[i].Contract.Weekly) + ") at the base rate with benefits; independent of the roster"},
			api.MoneyLine{Label: "Contract pay to the last day · " + who, Cents: notice, When: "today to " + tp.LastDay,
				Basis: "minimum: the contract guarantee with benefits on the days still employed; worked pay replaces it"})
		out.Forecast = append(out.Forecast, api.MoneyLine{Label: "Contract cost no longer owed · " + who, Cents: st.weeklyGuarantee(i),
			When: "per full week from " + tp.EndDate, Basis: "estimate: the contract guarantee with benefits; overtime and hold-over not modeled"})
	}
	if len(p.rosters) > 0 {
		st.previewRosters(&out, p, h, plans)
	}
	if len(p.allocs) > 0 {
		st.previewAllocations(&out, p)
	}
	for _, ts := range p.teles {
		st.telePreview(ts, &out)
	}
	out.Displacements = st.provisional(plans, disp)
	if len(disp) > 0 {
		k := 0
		for _, x := range out.Displacements {
			if x.ProvisionalDate != "" {
				k++
			}
		}
		out.Forecast = append(out.Forecast, api.MoneyLine{Label: "Displaced bookings", Cents: 0,
			When: "at commit", Basis: strconv.Itoa(len(disp)) + " appointments lose coverage; " + strconv.Itoa(k) +
				" have a feasible replacement by capacity. Whether each patient accepts is decided only at commit; lost exams lose their revenue"})
	}
	return out
}

// hireExit is a hire's earliest lawful exit: the contract guarantee from
// the start to the earliest last day, plus two contracted weeks'
// severance, with benefits.
func (st *state) hireExit(sp *hireSpec) int64 {
	t := &sp.terms
	w := t.RateCents * int64(t.Contract.Weekly) / 3600
	var exit int64
	for d := sp.from; d <= sp.from+NoticeDays; d++ {
		dd := st.cal.Date(st.cal.StartOfDay(d))
		exit += t.Contract.DayCents(t.RateCents, st.cfg.RotaWeek(dd), dd.Weekday())
	}
	return exit + roundDiv(exit*st.cfg.Finance.BenefitsBP, 10_000) + 2*w + roundDiv(2*w*st.cfg.Finance.BenefitsBP, 10_000)
}

// firstOfNextMonth is the day index of the next 1st (A/P settlement).
func (st *state) firstOfNextMonth() int64 {
	d := st.cal.Date(st.cal.StartOfDay(st.day))
	return st.day + int64(d.DaysInMonth()-d.Day+1)
}

// opensSlots counts the template slots of materialized days from day on
// that are not bookable now and would be under plans.
func (st *state) opensSlots(plans map[int64]DayPlan, from int64) []api.ModalityCount {
	count := map[string]int{}
	for d := max(from, st.day+1); d < int64(len(st.dem.days)); d++ {
		p, ok := plans[d]
		if !ok || p.Hours == nil {
			continue
		}
		cov := p.Coverage()
		ds := st.dem.days[d]
		for r := range ds.rooms {
			for _, s := range ds.rooms[r] {
				if s.why == slotOK {
					continue
				}
				x := s
				st.coverSlot(&cov, r, &x)
				if x.why == slotOK {
					count[st.cfg.Rooms[r].Modality]++
				}
			}
		}
	}
	var out []api.ModalityCount
	for _, m := range st.modalities() {
		if count[m] > 0 {
			out = append(out, api.ModalityCount{Modality: m, Count: count[m]})
		}
	}
	return out
}

// provisional lays out the displaced appointments and, on copies of the
// affected days under plans, the earliest feasible replacement of each in
// rebooking order — by capacity alone, every patient accepting.
func (st *state) provisional(plans map[int64]DayPlan, disp []StaffingConflict) []api.DisplacementPreview {
	if len(disp) == 0 {
		return nil
	}
	days := map[int64]*daySlots{}
	for d, p := range plans {
		ds := st.dem.days[d]
		c := &daySlots{rooms: make([][]slot, len(ds.rooms))}
		cov := p.Coverage()
		if p.Hours != nil {
			c.segs = cov.Segments
		}
		for r := range ds.rooms {
			c.rooms[r] = slices.Clone(ds.rooms[r])
			if p.Hours != nil {
				for k := range c.rooms[r] {
					st.coverSlot(&cov, r, &c.rooms[r][k])
				}
			}
		}
		days[d] = c
	}
	for _, x := range disp {
		a := &st.dem.appts[x.appt]
		days[a.day].rooms[a.room][a.slot].appt = -1
	}
	order := slices.Clone(disp)
	slices.SortFunc(order, func(x, y StaffingConflict) int {
		sx, sy := st.apptStart(x.appt), st.apptStart(y.appt)
		ox, oy := &st.dem.orders[st.dem.appts[x.appt].order], &st.dem.orders[st.dem.appts[y.appt].order]
		switch {
		case sx != sy:
			return int(sx - sy)
		case ox.priority != oy.priority:
			return int(oy.priority) - int(ox.priority)
		case ox.received != oy.received:
			return int(ox.received - oy.received)
		case st.dem.appts[x.appt].order != st.dem.appts[y.appt].order:
			return st.dem.appts[x.appt].order - st.dem.appts[y.appt].order
		}
		return x.appt - y.appt
	})
	byAppt := map[int]api.DisplacementPreview{}
	last := min(st.day+st.cfg.Policies.BookingHorizonDays, int64(len(st.dem.days))-1)
	for _, x := range order {
		a := &st.dem.appts[x.appt]
		o := &st.dem.orders[a.order]
		start := st.apptStart(x.appt)
		p := api.DisplacementPreview{Appointment: x.Appointment, Patient: patientID(o.patient), Exam: st.examLabel(o.exam),
			Priority: x.Priority.String(), Date: st.cal.Date(start).String(), Start: simtime.FormatClock(x.Start), Room: x.Room, Reason: x.Reason}
		notBefore := max(start, st.cal.StartOfDay(o.earliest), st.now.Add(st.cfg.Policies.BookingNotice))
		cat := st.cfg.Catalog[o.exam].Category
	search:
		for d := st.cal.DayIndex(notBefore); d <= last; d++ {
			ds := days[d]
			if ds == nil {
				ds = st.dem.days[d]
			}
			best, bestAt := -1, simtime.Duration(0)
			var bestRoom int
			for r := range ds.rooms {
				for k := range ds.rooms[r] {
					s := &ds.rooms[r][k]
					if s.Category != cat || s.appt >= 0 || s.why != slotOK || st.cal.At(d, s.From) < notBefore ||
						(best >= 0 && s.From >= bestAt) || !sharedFree(ds, r, s) {
						continue
					}
					best, bestAt, bestRoom = k, s.From, r
				}
			}
			if best >= 0 {
				if days[d] != nil {
					days[d].rooms[bestRoom][best].appt = x.appt
				}
				p.ProvisionalDate, p.ProvisionalStart = st.cal.Date(st.cal.StartOfDay(d)).String(), simtime.FormatClock(bestAt)
				break search
			}
		}
		byAppt[x.appt] = p
	}
	var out []api.DisplacementPreview
	for _, x := range disp {
		out = append(out, byAppt[x.appt])
	}
	return out
}

// previewRosters describes each roster change of the batch: hours before
// and after against the unchanged contract, a trainee's productive date,
// the approved extensions kept, the batch's coverage differences, and the
// money — the guarantee (unchanged) apart from forecast idle contract time
// and overtime.
func (st *state) previewRosters(out *api.StaffingPreview, p *proposal, h *hypo, plans map[int64]DayPlan) {
	date := func(d int64) string { return st.cal.Date(st.cal.StartOfDay(d)).String() }
	mins := func(d simtime.Duration) int64 { return int64(d / simtime.Minute) }
	for _, rs := range p.rosters {
		i := rs.emp
		r := &st.staff[i]
		t := st.emps[i].terms
		vs := p.versionsOf(st, i)
		until := r.Until
		if u, ok := h.until[i]; ok {
			until = u
		}
		end := until
		for _, v := range vs {
			if v.From > rs.from {
				end = min(end, v.From)
				break
			}
		}
		ba, bb := rostered(rs.was)
		aa, ab := rostered(rs.then)
		rp := api.RosterPreview{Employee: t.ID, Name: t.Name, Op: rs.op, Effective: date(rs.from),
			ContractMin: mins(r.Contract.Weekly), BeforeMinA: mins(ba), BeforeMinB: mins(bb), AfterMinA: mins(aa), AfterMinB: mins(ab)}
		if end < noEnd {
			rp.Until = date(end)
		}
		if r.Hire != nil && r.Productive > st.day && r.Productive < noEnd {
			rp.ProductiveBefore = date(r.Productive)
			rp.ProductiveAfter = rp.ProductiveBefore
			if np, ok := p.productive[i]; ok {
				rp.ProductiveAfter = date(np)
			}
		}
		var days []int64
		for d, m := range st.ext {
			if d >= rs.from && d < end && m[i] > 0 {
				days = append(days, d)
			}
		}
		slices.Sort(days)
		var otMin int64
		for _, d := range days {
			x := api.Extension{Date: date(d), Employee: t.ID, Name: t.Name, Minutes: mins(st.ext[d][i])}
			if pl, ok := plans[d]; ok && pl.ByEmp[i] >= 0 {
				x.Until = simtime.FormatClock(pl.Shifts[pl.ByEmp[i]].To)
			}
			rp.Extensions = append(rp.Extensions, x)
			otMin += x.Minutes
		}
		out.Rosters = append(out.Rosters, rp)

		who := st.staffLabel(i)
		when := "from " + rp.Effective
		if rp.Until != "" {
			when += " until " + rp.Until
		}
		out.Guaranteed = append(out.Guaranteed, api.MoneyLine{Label: "Contract guarantee per full week · " + who, Cents: st.weeklyGuarantee(i),
			When: "unchanged by the roster", Basis: hours(r.Contract.Weekly) + " × " + rate(t.RateCents) + " with benefits, paid whatever is rostered; notice and severance unchanged"})
		c := r.Contract.Weekly
		ia, ib := max(0, c-aa), max(0, c-ab)
		if ia > 0 || ib > 0 {
			cost := func(d simtime.Duration) int64 {
				w := t.RateCents * int64(d) / 3600
				return w + roundDiv(w*st.cfg.Finance.BenefitsBP, 10_000)
			}
			basis := hours(ia) + " a week"
			if ia != ib {
				basis = fmt.Sprintf("%s (week A), %s (week B)", hours(ia), hours(ib))
			}
			out.Forecast = append(out.Forecast, api.MoneyLine{Label: "Contracted time not rostered · " + who, Cents: (cost(ia) + cost(ib)) / 2,
				When: "per week " + when, Basis: "estimate: " + basis + " paid as guarantee with benefits (part of the guarantee above, not extra), unless worked through extensions or hold-over"})
		}
		if otMin > 0 {
			out.Forecast = append(out.Forecast, api.MoneyLine{Label: "Approved extensions kept · " + who, Cents: t.RateCents * otMin / 60,
				When: when, Basis: fmt.Sprintf("estimate: %d min at the base rate; the overtime premium applies past %d h in a week", otMin,
					st.cfg.Policies.WeeklyHours/simtime.Hour)})
		}
	}
	// Coverage differences of the whole batch.
	opened, closed := map[string]int{}, map[string]int{}
	for d := st.day + 1; d < int64(len(st.dem.days)); d++ {
		pl, ok := plans[d]
		if !ok || pl.Hours == nil {
			continue
		}
		cov := pl.Coverage()
		ds := st.dem.days[d]
		for r := range ds.rooms {
			for _, s := range ds.rooms[r] {
				x := s
				st.coverSlot(&cov, r, &x)
				switch m := st.cfg.Rooms[r].Modality; {
				case s.why != slotOK && x.why == slotOK:
					opened[m]++
				case s.why == slotOK && x.why != slotOK:
					closed[m]++
				}
			}
		}
	}
	for _, m := range st.modalities() {
		if opened[m] > 0 {
			out.SlotsOpened = append(out.SlotsOpened, api.ModalityCount{Modality: m, Count: opened[m]})
		}
		if closed[m] > 0 {
			out.SlotsClosed = append(out.SlotsClosed, api.ModalityCount{Modality: m, Count: closed[m]})
		}
	}
}

// previewAllocations describes each allocation change of the batch: the
// allocation, site reading time and salary before and after, until the
// next version; the reporting backlog now; and the money — the salary
// run-rate change as a guaranteed obligation from the effective date.
// Shifts and contrast supervision are unchanged, and no booking moves.
func (st *state) previewAllocations(out *api.StaffingPreview, p *proposal) {
	date := func(d int64) string { return st.cal.Date(st.cal.StartOfDay(d)).String() }
	mins := func(d simtime.Duration) int64 { return int64(d / simtime.Minute) }
	var backlog, urgent int
	var oldest simtime.Duration
	for si := range st.dem.studies {
		s := &st.dem.studies[si]
		if s.state == studyFinal {
			continue
		}
		backlog++
		if s.priority == Urgent {
			urgent++
		}
		oldest = max(oldest, st.now.Sub(s.acquiredAt))
	}
	for _, as := range p.allocs {
		i := as.emp
		t := st.emps[i].terms
		ap := api.AllocationPreview{Employee: t.ID, Name: t.Name, Op: as.op, Effective: date(as.from), BeforeBP: as.was, AfterBP: as.then,
			WithdrawBy: date(as.from - NoticeDays), BacklogStudies: backlog, BacklogUrgent: urgent, BacklogOldestMin: mins(oldest)}
		for _, v := range p.allocsOf(st, i) {
			if v.From > as.from {
				ap.Until = date(v.From)
				break
			}
		}
		ba, bb := siteWeek(t, as.was)
		aa, ab := siteWeek(t, as.then)
		ap.BeforeSiteMinA, ap.BeforeSiteMinB, ap.AfterSiteMinA, ap.AfterSiteMinB = mins(ba), mins(bb), mins(aa), mins(ab)
		ap.BeforeSalary30Cents, ap.AfterSalary30Cents = st.salary30(i, as.was), st.salary30(i, as.then)
		out.Allocations = append(out.Allocations, ap)

		who := st.staffLabel(i)
		when := "per 30 days from " + ap.Effective
		if ap.Until != "" {
			when += " until " + ap.Until
		} else {
			when += " until changed (14 days' notice)"
		}
		out.Guaranteed = append(out.Guaranteed, api.MoneyLine{Label: "Salary allocation change · " + who,
			Cents: ap.AfterSalary30Cents - ap.BeforeSalary30Cents, When: when,
			Basis: fmt.Sprintf("%s salary × %s (was %s) with benefits, accrued daily and paid on paydays; withdrawable at no cost until %s",
				money(t.RateCents), pct(as.then), pct(as.was), ap.WithdrawBy)})
		d := (aa + ab - ba - bb) / 2
		out.Forecast = append(out.Forecast, api.MoneyLine{Label: "Site reading capacity · " + who, Cents: 0, When: "from " + ap.Effective,
			Basis: fmt.Sprintf("estimate: %+d min of site reading blocks a week (mean of rota weeks A and B) inside the same shifts; reports gained depend on the worklist and read times; supervision unchanged",
				mins(d))})
	}
}
