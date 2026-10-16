package sim

import (
	"fmt"
	"slices"
	"strings"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Practice-caused displacement (V0.2_DESIGN §7). A committed staffing
// change that takes coverage away (a termination's end, a roster version,
// a trainee's later productive date) is reconciled once per batch, at the
// decision instant, against the final timeline:
//
//  1. Every materialized day from the earliest change on is re-planned
//     from the registry, and its booked appointments are re-inserted in
//     survivor order — priority (urgent first), original start,
//     confirmation time, appointment — each against only the retained
//     ones, with the booking rule's own predicates (SlotBookable for the
//     whole slot, sharedFree for the shared-technologist cluster, with
//     backtracking). An appointment that cannot be retained is displaced.
//  2. All displaced appointments release their slots (state displaced;
//     the history stays), then every day's coverage and slot staffing
//     move to the new plan, so the retained set is exactly what the new
//     coverage supports.
//  3. Each displaced order is offered a replacement in rebooking order —
//     original start, priority, receipt, order, appointment — in a
//     same-category slot no earlier than the original start, the
//     original wanted date and the booking notice: the first in the
//     patient's preferred window over the horizon, else the first at any
//     time. It is accepted when its day is within the patient's tolerance
//     of the tolerance anchor (the earliest date of the patient's current
//     voluntary booking attempt, never renewed by a displacement), and
//     otherwise the order is lost to displacement. No random draw is
//     made: the replacement keeps the original behavioral attempt, so the
//     same no-show, cancellation, arrival and exam draws apply to the new
//     start, with the original no-show lead basis and confirmation.
//
// Displacement is neither a patient cancellation nor a no-show: it has its
// own appointment state, order state, counters and log, and is never a
// booking loss. Earlier days and patients on site are untouched: every
// change takes effect on a later day.

// Displacement causes.
const (
	causeEmploymentEnded = "employment_ended"
)

// Displacement counters (flowCounts.disp).
const (
	dispEvents = iota
	dispOrders
	dispRepeated
	dispRebooked
	dispNoSlot
	dispDeclined
	dispWalkIn
	numDisp
)

// displacement is one displaced appointment's immutable record.
type displacement struct {
	Appt, Order int
	Actions     []int // the batch's contributing actions (action record indices), in input order
	Cause       string
	Reason      string // the first failed condition: unstaffed, no_supervision, shared_capacity
	At          simtime.Time
	Replacement int    // the replacement appointment, -1 if none
	Resolution  string // rebooked, no_feasible_slot, replacement_declined
}

const (
	resRebooked = "rebooked"
	resNoSlot   = "no_feasible_slot"
	resDeclined = "replacement_declined"
)

// survivors re-inserts day's booked appointments under plan in survivor
// order and returns those that cannot be retained, with the first
// condition each fails. It reads state only.
func (st *state) survivors(day int64, plan DayPlan) []StaffingConflict {
	ds := st.dem.days[day]
	type cand struct{ room, k, appt int }
	var cands []cand
	for r := range ds.rooms {
		for k := range ds.rooms[r] {
			if a := ds.rooms[r][k].appt; a >= 0 {
				cands = append(cands, cand{r, k, a})
			}
		}
	}
	if len(cands) == 0 {
		return nil
	}
	cov := plan.Coverage()
	tmp := &daySlots{segs: cov.Segments, rooms: make([][]slot, len(ds.rooms))}
	for r := range ds.rooms {
		tmp.rooms[r] = slices.Clone(ds.rooms[r])
		for k := range tmp.rooms[r] {
			tmp.rooms[r][k].appt = -1
			st.coverSlot(&cov, r, &tmp.rooms[r][k])
		}
	}
	slices.SortFunc(cands, func(a, b cand) int {
		pa, pb := st.dem.orders[st.dem.appts[a.appt].order].priority, st.dem.orders[st.dem.appts[b.appt].order].priority
		switch {
		case pa != pb:
			return int(pb) - int(pa)
		case ds.rooms[a.room][a.k].From != ds.rooms[b.room][b.k].From:
			return int(ds.rooms[a.room][a.k].From - ds.rooms[b.room][b.k].From)
		case st.dem.appts[a.appt].bookedAt != st.dem.appts[b.appt].bookedAt:
			return int(st.dem.appts[a.appt].bookedAt - st.dem.appts[b.appt].bookedAt)
		}
		return a.appt - b.appt
	})
	var out []StaffingConflict
	for _, c := range cands {
		s := &tmp.rooms[c.room][c.k]
		why := ""
		switch {
		case s.why == slotUnstaffed:
			why = ConflictUnstaffed
		case s.why == slotNoSupervision:
			why = ConflictNoSupervision
		case !sharedFree(tmp, c.room, s):
			why = ConflictShared
		default:
			s.appt = c.appt
			continue
		}
		out = append(out, StaffingConflict{Appointment: apptID(c.appt), Day: day, Room: st.cfg.Rooms[c.room].ID,
			Start: s.From, Priority: st.dem.orders[st.dem.appts[c.appt].order].priority, Reason: why, appt: c.appt})
	}
	return out
}

// reconcile displaces and re-offers every booking that the committed
// registry no longer covers, from day from on (step 1–3 above).
func (st *state) reconcile(e *Engine, from int64, cause string, actions []int) {
	n := int64(len(st.dem.days))
	var conf []StaffingConflict
	plans := make([]DayPlan, max(0, n-from))
	for d := from; d < n; d++ {
		plans[d-from] = st.planFor(d)
		conf = append(conf, st.survivors(d, plans[d-from])...)
	}
	// Release every displaced reservation before any re-offer.
	type held struct {
		ai     int
		reason string
		start  simtime.Time
	}
	var out []held
	for _, c := range conf {
		a := &st.dem.appts[c.appt]
		ds := st.dem.days[a.day]
		ds.rooms[a.room][a.slot].appt = -1
		ds.booked--
		out = append(out, held{c.appt, c.Reason, st.apptStart(c.appt)})
		a.state, a.endedAt = apptDisplaced, st.now
	}
	for d := from; d < n; d++ {
		st.applyCoverage(d, plans[d-from])
	}
	slices.SortFunc(out, func(x, y held) int {
		ox, oy := &st.dem.orders[st.dem.appts[x.ai].order], &st.dem.orders[st.dem.appts[y.ai].order]
		switch {
		case x.start != y.start:
			return int(x.start - y.start)
		case ox.priority != oy.priority:
			return int(oy.priority) - int(ox.priority)
		case ox.received != oy.received:
			return int(ox.received - oy.received)
		case st.dem.appts[x.ai].order != st.dem.appts[y.ai].order:
			return st.dem.appts[x.ai].order - st.dem.appts[y.ai].order
		}
		return x.ai - y.ai
	})
	for _, h := range out {
		st.rebookDisplaced(e, h.ai, h.reason, h.start, cause, actions)
	}
}

// rebookDisplaced resolves one displaced appointment: a replacement the
// patient accepts, or the order lost to displacement.
func (st *state) rebookDisplaced(e *Engine, ai int, reason string, start simtime.Time, cause string, actions []int) {
	old := st.dem.appts[ai]
	oi := old.order
	o := &st.dem.orders[oi]
	rec := displacement{Appt: ai, Order: oi, Actions: slices.Clone(actions), Cause: cause, Reason: reason, At: st.now, Replacement: -1}
	for _, f := range []*flowCounts{&st.dem.flowToday, &st.dem.flowTotal} {
		f.disp[dispEvents]++
		if o.displaced == 0 {
			f.disp[dispOrders]++
		} else {
			f.disp[dispRepeated]++
		}
		if o.walkIn {
			f.disp[dispWalkIn]++
		}
	}
	o.displaced++
	notBefore := max(start, st.cal.StartOfDay(o.earliest), st.now.Add(st.cfg.Policies.BookingNotice))
	pref, any, hasPref, hasAny := st.displaceFit(o, notBefore)
	head := fmt.Sprintf("%s (%s %s %s, %s %s) displaced by the practice: %s (%s)", apptID(ai), patientID(o.patient), orderID(oi),
		st.examLabel(o.exam), st.dayLabel(old.day), simtime.FormatClock(st.dem.days[old.day].rooms[old.room][old.slot].From),
		st.causeText(cause, actions), conflictText(reason))
	pick, res := acceptReplacement(o.anchor, o.tolerance, pref, any, hasPref, hasAny)
	if res != resRebooked {
		rec.Resolution = res
		why := fmt.Sprintf("no %s slot within the %d-day horizon", st.cfg.Catalog[o.exam].Modality, st.cfg.Policies.BookingHorizonDays)
		if res == resDeclined {
			why = "declined the replacement, earliest " + st.dayLabel(any.day)
		}
		k := dispNoSlot
		if rec.Resolution == resDeclined {
			k = dispDeclined
		}
		st.dem.flowToday.disp[k]++
		st.dem.flowTotal.disp[k]++
		o.state, o.endedAt, o.lostWhy = orderDisplacedLost, st.now, "displaced by the practice; "+why
		st.dem.displaced = append(st.dem.displaced, rec)
		st.emit(st.now, api.Warn, "orders", head+" → lost: "+why)
		return
	}
	n := st.book(e, oi, pick, old.confirmed)
	a := &st.dem.appts[n]
	a.noShowLead = old.noShowLead
	o.placements++
	a.placement = o.placements
	rec.Replacement, rec.Resolution = n, resRebooked
	st.dem.flowToday.disp[dispRebooked]++
	st.dem.flowTotal.disp[dispRebooked]++
	st.dem.displaced = append(st.dem.displaced, rec)
	st.emit(st.now, api.Info, "orders", fmt.Sprintf("%s → rebooked %s %s (%s)", head, st.dayLabel(pick.day),
		simtime.FormatClock(st.slotAt(pick).From), apptID(n)))
}

// acceptReplacement is the patient's answer to a replacement offer
// (V0.2_DESIGN §7.2): the preferred-window slot if its day is within the
// tolerance of the anchor, else the first any-window slot if its is;
// otherwise declined (a slot exists, too late) or no feasible slot. The
// anchor is the earliest date of the current voluntary booking attempt,
// never the displaced appointment's day: displacement renews no patience.
func acceptReplacement(anchor, tolerance int64, pref, any slotRef, hasPref, hasAny bool) (slotRef, string) {
	switch {
	case hasPref && pref.day-anchor <= tolerance:
		return pref, resRebooked
	case hasAny && any.day-anchor <= tolerance:
		return any, resRebooked
	case hasAny:
		return slotRef{}, resDeclined
	}
	return slotRef{}, resNoSlot
}

// displaceFit finds replacement slots for o no earlier than notBefore:
// the first in the patient's preferred window over the horizon and the
// first at any time, in day, time, room, slot order. The search stops
// once a later day could not be accepted (past the anchor's tolerance).
func (st *state) displaceFit(o *order, notBefore simtime.Time) (pref, any slotRef, hasPref, hasAny bool) {
	cat := st.cfg.Catalog[o.exam].Category
	last := min(st.day+st.cfg.Policies.BookingHorizonDays, int64(len(st.dem.days))-1)
	for d := st.cal.DayIndex(notBefore); d <= last; d++ {
		if hasAny && d-o.anchor > o.tolerance {
			break
		}
		ds := st.dem.days[d]
		dayPref, dayAny := slotRef{idx: -1}, slotRef{idx: -1}
		var prefAt, anyAt simtime.Duration
		for r := range ds.rooms {
			for i := range ds.rooms[r] {
				s := &ds.rooms[r][i]
				if s.Category != cat || s.appt >= 0 || s.why != slotOK || st.cal.At(d, s.From) < notBefore {
					continue
				}
				canAny := dayAny.idx < 0 || s.From < anyAt
				canPref := inWindow(o.window, s.From) && (dayPref.idx < 0 || s.From < prefAt)
				if (!canAny && !canPref) || !sharedFree(ds, r, s) {
					continue
				}
				if canAny {
					dayAny, anyAt = slotRef{d, r, i}, s.From
				}
				if canPref {
					dayPref, prefAt = slotRef{d, r, i}, s.From
				}
			}
		}
		if !hasAny && dayAny.idx >= 0 {
			any, hasAny = dayAny, true
		}
		if dayPref.idx >= 0 {
			return dayPref, any, true, hasAny
		}
	}
	return pref, any, false, hasAny
}

func conflictText(reason string) string {
	switch reason {
	case ConflictUnstaffed:
		return "no qualified technologist for the slot"
	case ConflictNoSupervision:
		return "no on-site radiologist for contrast"
	case ConflictShared:
		return "the remaining technologists are needed in other rooms"
	}
	return reason
}

// causeText names what displaced: "Sam (E04) leaves after Sat Nov 21",
// "Riley (E03)'s roster changes from Mon Nov 16". A batch's cause lists
// its kinds of change joined by "+" (employment_ended, roster_changed,
// training_delayed).
func (st *state) causeText(cause string, actions []int) string {
	var who []string
	for _, c := range strings.Split(cause, "+") {
		switch c {
		case causeEmploymentEnded:
			for i := range st.staff {
				if x := st.staff[i].Exit; x != nil && slices.Contains(actions, x.Action) {
					who = append(who, fmt.Sprintf("%s leaves after %s", st.staffLabel(i), st.dayLabel(x.LastDay)))
				}
			}
		case causeRosterChanged:
			for i := range st.staff {
				for _, e := range st.staff[i].RosterLog {
					if e.Op != rosterVoided && slices.Contains(actions, e.Action) {
						verb := "changes"
						if e.Op == rosterRemove {
							verb = "change is withdrawn"
						}
						who = append(who, fmt.Sprintf("%s's roster %s from %s", st.staffLabel(i), verb, st.dayLabel(e.From)))
					}
				}
			}
		case causeTrainingDelayed:
			who = append(who, "training completes later")
		default:
			who = append(who, c)
		}
	}
	return strings.Join(who, "; ")
}

// displacementWhy is the observable account of appointment ai's
// displacement (for the patient journey).
func (st *state) displacementWhy(ai int) string {
	for k := len(st.dem.displaced) - 1; k >= 0; k-- {
		if x := &st.dem.displaced[k]; x.Appt == ai {
			text := conflictText(x.Reason)
			if x.Replacement >= 0 {
				return text + "; rebooked as " + apptID(x.Replacement)
			}
			return text + "; no replacement accepted"
		}
	}
	return ""
}

// dispMetrics reports a period's displacement counters (nil when none).
func dispMetrics(f *flowCounts) *api.DisplacementMetrics {
	if f.disp == [numDisp]int{} {
		return nil
	}
	return &api.DisplacementMetrics{Events: f.disp[dispEvents], Orders: f.disp[dispOrders], Repeated: f.disp[dispRepeated],
		Rebooked: f.disp[dispRebooked], NoFeasibleSlot: f.disp[dispNoSlot], Declined: f.disp[dispDeclined],
		WalkInOrigin: f.disp[dispWalkIn]}
}

// checkDisplaced verifies the displacement records against appointments
// and orders.
func (st *state) checkDisplaced() error {
	for k := range st.dem.displaced {
		x := &st.dem.displaced[k]
		a := &st.dem.appts[x.Appt]
		switch {
		case a.state != apptDisplaced || a.order != x.Order:
			return fmt.Errorf("displacement %d: %s is %s", k, apptID(x.Appt), apptStateNames[a.state])
		case x.Replacement >= 0 && (st.dem.appts[x.Replacement].order != x.Order || st.dem.appts[x.Replacement].attempt != a.attempt):
			return fmt.Errorf("displacement %d: replacement %s is not the same order and attempt", k, apptID(x.Replacement))
		case x.Replacement < 0 && x.Resolution == resRebooked:
			return fmt.Errorf("displacement %d: rebooked without a replacement", k)
		}
	}
	return nil
}
