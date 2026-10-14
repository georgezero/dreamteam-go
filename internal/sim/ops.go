package sim

import (
	"fmt"
	"slices"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/rng"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Operations: registration, imaging dispatch, exam timing, hold-over and
// coverage (SIMULATION_DESIGN §3.5, §5.7, §5.9–5.10, §6.4–6.5, §6.9).
//
// Dispatch runs once per transaction, after every event at that instant,
// visiting employees, rooms and visits in fixed order, so the outcome does
// not depend on how the run is chunked or how fast it is shown.

// Dispatch matches people to work, then settles hold-overs, rooms, the
// closing phase and eligibility wakes.
func (st *state) Dispatch(e *Engine) {
	st.now = e.Now()
	st.holdNeeded(e)
	st.matchRegistration(e)
	st.matchImaging(e)
	st.checkCoverage(e)
	st.releaseHeld(e)
	st.restaff(e.Now())
	st.operationalClose(e)
	st.scheduleWakes(e)
}

// --- registration ---------------------------------------------------------

// regOrder ranks visits waiting to register: urgent first, then scheduled
// patients by appointment time, then walk-ins by arrival (§3.5).
func (st *state) regOrder(vs []int) []int {
	out := slices.Clone(vs)
	key := func(vi int) (int, simtime.Time) {
		v := &st.dem.visits[vi]
		o := &st.dem.orders[v.order]
		switch {
		case o.priority == Urgent:
			return 0, v.arrived
		case v.appt >= 0:
			return 1, st.apptStart(v.appt)
		}
		return 2, v.arrived
	}
	slices.SortStableFunc(out, func(a, b int) int {
		ra, ta := key(a)
		rb, tb := key(b)
		if ra != rb {
			return ra - rb
		}
		if ta != tb {
			return int(ta - tb)
		}
		return a - b
	})
	return out
}

func (st *state) apptStart(ai int) simtime.Time {
	a := &st.dem.appts[ai]
	return st.cal.At(a.day, st.dem.days[a.day].rooms[a.room][a.slot].From)
}

// matchRegistration gives each arrived visit, in queue order, the first
// free registrar: front desk, then the operations manager, then (only when
// no desk-capable person is on duty) a technologist of the visit's modality.
func (st *state) matchRegistration(e *Engine) {
	var queue []int
	for _, vi := range st.dem.waiting {
		if st.dem.visits[vi].state == visitArrived {
			queue = append(queue, vi)
		}
	}
	if len(queue) == 0 {
		return
	}
	deskOnDuty := false
	for i := range st.emps {
		if st.emps[i].status != offDuty && deskLevel(st.emps[i].cfg) > DeskClosed {
			deskOnDuty = true
		}
	}
	for _, vi := range st.regOrder(queue) {
		v := &st.dem.visits[vi]
		o := &st.dem.orders[v.order]
		b := &st.cfg.Behavior
		q := b.RegScheduled
		if o.walkIn {
			q = b.RegWalkIn
		}
		pick, factor := -1, int64(10_000)
		for _, lv := range []struct {
			level  int
			factor int64
		}{{DeskStaffed, 10_000}, {DeskOps, b.RegOpsBP}} {
			for i := range st.emps {
				emp := &st.emps[i]
				if pick < 0 && emp.status == available && deskLevel(emp.cfg) == lv.level &&
					st.fits(i, simtime.Duration(q.Max()*lv.factor/10_000)) {
					pick, factor = i, lv.factor
				}
			}
		}
		if pick < 0 && !deskOnDuty {
			m := st.cfg.Catalog[o.exam].Modality
			for i := range st.emps {
				if pick < 0 && st.emps[i].status == available && st.staffsModality(i, m) &&
					st.fits(i, simtime.Duration(q.Max()*b.RegSelfBP/10_000)) {
					pick, factor = i, b.RegSelfBP
				}
			}
		}
		if pick < 0 {
			continue
		}
		draw := rng.Hash(st.seed, "registration", o.key.keys(v.attempt)...)
		dur := simtime.Duration(q.Sample(draw.Lane(0)) * factor / 10_000)
		emp := &st.emps[pick]
		emp.status, emp.task, emp.taskVisit = busy, taskReg, vi
		v.state, v.regBy, v.regStart = visitRegistering, pick, e.Now()
		st.dem.flowToday.registered++
		st.dem.flowTotal.registered++
		if factor == b.RegSelfBP && emp.cfg.Role == Technologist {
			st.dem.flowToday.selfRegistered++
			st.dem.flowTotal.selfRegistered++
		}
		e.Push(Event{At: e.Now().Add(max(1, dur)), Class: ClassCompletion, Kind: kindRegEnd, A: int64(vi)})
	}
}

// fits applies the admission rule (§6.4): a task may start only if it can
// end by the employee's shift cap.
func (st *state) fits(i int, d simtime.Duration) bool {
	return st.now.Add(d) <= st.emps[i].capAt
}

// staffsModality reports whether on-duty technologist i staffs some room
// of modality m now.
func (st *state) staffsModality(i int, m string) bool {
	clock := simtime.ClockOf(st.now)
	for r, room := range st.cfg.Rooms {
		if room.Modality == m && st.techStaffs(i, r, clock) {
			return true
		}
	}
	return false
}

func (st *state) regEnd(e *Engine, vi int) {
	v := &st.dem.visits[vi]
	v.state, v.regEnd = visitWaiting, e.Now()
	st.taskDone(e, v.regBy)
}

// --- imaging --------------------------------------------------------------

// imagingCand is a registered patient ranked for a free room by the
// imaging queue policy (§5.9).
type imagingCand struct {
	room, vi int
	rank     int          // 0 urgent, 1 due scheduled, 2 late fitted in, 3 walk-in, 4 early scheduled
	at       simtime.Time // appointment start (ranks 1, 4) or arrival
}

// imagingOrder ranks the registered patients for every free room: urgent;
// scheduled patients who are due (from EarlyStart before the appointment)
// by appointment; late patients fitted in; walk-ins by arrival (subject to
// the walk-in policy); then, if StartEarly, early patients by appointment.
// Ranking across rooms (rather than room by room) lets a technologist
// shared between rooms serve the most overdue patient first; room order
// only breaks ties.
func (st *state) imagingOrder() []imagingCand {
	pol := &st.cfg.Policies
	var cs []imagingCand
	for r := range st.rooms {
		if st.rooms[r].visit >= 0 {
			continue
		}
		m := st.cfg.Rooms[r].Modality
		var room []imagingCand
		nextDue := simtime.Time(-1) // earliest appointment of a waiting patient not yet due
		for _, vi := range st.dem.waiting {
			v := &st.dem.visits[vi]
			o := &st.dem.orders[v.order]
			if v.state != visitWaiting || st.cfg.Catalog[o.exam].Modality != m {
				continue
			}
			switch {
			case o.priority == Urgent:
				room = append(room, imagingCand{r, vi, 0, v.arrived})
			case v.afterGrace:
				room = append(room, imagingCand{r, vi, 2, v.arrived})
			case v.appt >= 0:
				start := st.apptStart(v.appt)
				if start <= st.now.Add(pol.EarlyStart) {
					room = append(room, imagingCand{r, vi, 1, start})
				} else {
					if nextDue < 0 || start < nextDue {
						nextDue = start
					}
					if pol.StartEarly {
						room = append(room, imagingCand{r, vi, 4, start})
					}
				}
			default:
				room = append(room, imagingCand{r, vi, 3, v.arrived})
			}
		}
		// accept_if_gap: a walk-in goes only if the room can image them
		// before the next arrived scheduled patient is due.
		if pol.WalkIn == "accept_if_gap" && nextDue >= 0 {
			room = slices.DeleteFunc(room, func(c imagingCand) bool {
				if c.rank != 3 {
					return false
				}
				x := &st.cfg.Catalog[st.dem.orders[st.dem.visits[c.vi].order].exam]
				return st.now.Add(x.Expected()) > nextDue.Add(-pol.EarlyStart)
			})
		}
		cs = append(cs, room...)
	}
	slices.SortStableFunc(cs, func(a, b imagingCand) int {
		switch {
		case a.rank != b.rank:
			return a.rank - b.rank
		case a.at != b.at:
			return int(a.at - b.at)
		case a.room != b.room:
			return a.room - b.room
		}
		return a.vi - b.vi
	})
	return cs
}

// matchImaging starts exams: in queue order across free rooms, each
// patient whose room is still free and who has a qualified technologist
// free for that room within their cap and, for contrast, an on-site
// radiologist (§6.5). Exams start from opening and, after scheduled close,
// only for admitted patients (everyone on site was admitted).
func (st *state) matchImaging(e *Engine) {
	if !st.open && !st.closing {
		return
	}
	for _, c := range st.imagingOrder() {
		if st.rooms[c.room].visit >= 0 || st.dem.visits[c.vi].state != visitWaiting {
			continue
		}
		x := &st.cfg.Catalog[st.dem.orders[st.dem.visits[c.vi].order].exam]
		if x.Contrast && st.supervisor(x) < 0 {
			continue
		}
		if t := st.pickTech(c.room, x); t >= 0 {
			st.startExam(e, c.room, c.vi, t)
		}
	}
}

// pickTech chooses a technologist for an exam in room r: free, staffing the
// room, able to finish (p100) by their cap; regular-duty staff before
// held-over staff. A technologist near shift end hands over to a colleague
// on regular duty who covers the room (§6.4 handover rule).
func (st *state) pickTech(r int, x *ExamConfig) int {
	clock := simtime.ClockOf(st.now)
	expectedEnd := st.now.Add(x.Expected())
	for _, wantOver := range []bool{false, true} {
		for i := range st.emps {
			emp := &st.emps[i]
			if emp.over != wantOver || emp.status != available || !st.techStaffs(i, r, clock) || !st.fits(i, x.MaxTask()) {
				continue
			}
			if !emp.over && expectedEnd > emp.endAt.Add(st.cfg.Policies.OvertimeAllowance) && st.relief(i, r, expectedEnd) {
				continue
			}
			return i
		}
	}
	return -1
}

// relief reports whether another technologist on regular duty staffs room
// r and is rostered past t.
func (st *state) relief(i, r int, t simtime.Time) bool {
	clock := simtime.ClockOf(st.now)
	for j := range st.emps {
		if j != i && !st.emps[j].over && st.emps[j].endAt >= t && st.techStaffs(j, r, clock) {
			return true
		}
	}
	return false
}

// supervisor returns an on-site radiologist on duty (or held over) whose
// cap covers the exam's longest room time, or -1 (§6.5 rule 2).
func (st *state) supervisor(x *ExamConfig) int {
	for i := range st.emps {
		emp := &st.emps[i]
		if emp.cfg.Role == Radiologist && emp.status != offDuty && !st.plan.Shifts[emp.shift].Remote &&
			st.now.Add(simtime.Duration(x.RoomTime.Max())) <= emp.capAt {
			return i
		}
	}
	return -1
}

func (st *state) startExam(e *Engine, r, vi, t int) {
	v := &st.dem.visits[vi]
	o := &st.dem.orders[v.order]
	x := &st.cfg.Catalog[o.exam]
	emp := &st.emps[t]
	emp.status, emp.task, emp.taskVisit = busy, taskExam, vi
	v.state, v.examBy, v.room, v.examStart = visitInExam, t, r, e.Now()
	v.afterHours = !st.open
	st.rooms[r].visit = vi
	draw := rng.Hash(st.seed, "attempt.exam", o.key.keys(v.attempt)...)
	dur := simtime.Duration(x.RoomTime.Sample(draw.Lane(0)))
	for _, f := range []*flowCounts{&st.dem.flowToday, &st.dem.flowTotal} {
		f.examsStarted++
		if v.afterHours {
			f.afterHours++
		}
	}
	text := fmt.Sprintf("%s exam started: %s %s (%s, expected %s)", st.cfg.Rooms[r].Modality,
		patientID(o.patient), x.Name, emp.cfg.Name, x.Expected())
	if v.afterHours {
		text += " · after hours"
	}
	st.emit(e.Now(), api.Info, "imaging", text)
	e.Push(Event{At: e.Now().Add(max(1, dur)), Class: ClassCompletion, Kind: kindExamEnd, A: int64(vi)})
}

// examEnd: the patient departs and the room is free; the technologist
// documents the study (busy, off the room) before taking the next task.
func (st *state) examEnd(e *Engine, vi int) {
	v := &st.dem.visits[vi]
	o := &st.dem.orders[v.order]
	x := &st.cfg.Catalog[o.exam]
	v.state, v.examEnd, v.endedAt = visitDone, e.Now(), e.Now()
	st.rooms[v.room].visit = -1
	st.removeWaiting(vi)
	o.state, o.endedAt = orderImaged, e.Now()
	st.dem.flowToday.examsDone++
	st.dem.flowTotal.examsDone++
	draw := rng.Hash(st.seed, "attempt.exam", o.key.keys(v.attempt)...)
	doc := simtime.Duration(x.TechExtra.Sample(draw.Lane(1)))
	emp := &st.emps[v.examBy]
	emp.task = taskDoc
	st.emit(e.Now(), api.Info, "imaging", fmt.Sprintf("%s %s %s done in %s, departs (waited %s)",
		st.cfg.Rooms[v.room].Modality, patientID(o.patient), x.Name, e.Now().Sub(v.examStart), v.examStart.Sub(v.arrived)))
	e.Push(Event{At: e.Now().Add(max(1, doc)), Class: ClassCompletion, Kind: kindDocEnd, Gen: emp.gen, A: int64(v.examBy)})
}

// --- hold-over and coverage (§6.9) ----------------------------------------

// neededFor says why employee i, past shift end, must stay ("" if not).
func (st *state) neededFor(i int) string {
	emp := &st.emps[i]
	clock := simtime.ClockOf(st.now)
	switch emp.cfg.Role {
	case Technologist:
		for _, r := range emp.heldRooms {
			m := st.cfg.Rooms[r].Modality
			n := 0
			for _, vi := range st.dem.waiting {
				v := &st.dem.visits[vi]
				x := &st.cfg.Catalog[st.dem.orders[v.order].exam]
				if (v.state == visitArrived || v.state == visitWaiting) && x.Modality == m && st.fits(i, x.MaxTask()) {
					n++
				}
			}
			if n == 0 {
				continue
			}
			covered := false
			for j := range st.emps {
				if j != i && !st.emps[j].over && st.techStaffs(j, r, clock) {
					covered = true
				}
			}
			if !covered {
				return fmt.Sprintf("%d patient%s still need %s", n, plural(n), m)
			}
		}
	case Radiologist:
		if st.plan.Shifts[emp.shift].Remote {
			return ""
		}
		n := 0
		for _, vi := range st.dem.waiting {
			if st.cfg.Catalog[st.dem.orders[st.dem.visits[vi].order].exam].Contrast {
				n++
			}
		}
		if n == 0 {
			return ""
		}
		for j := range st.emps {
			o := &st.emps[j]
			if j != i && o.cfg.Role == Radiologist && o.status != offDuty && !o.over && !st.plan.Shifts[o.shift].Remote {
				return ""
			}
		}
		return fmt.Sprintf("%d contrast patient%s need an on-site radiologist", n, plural(n))
	}
	return ""
}

// holdNeeded marks idle employees past shift end who are still needed as
// held over, before any matching, so required overtime is recorded as such.
func (st *state) holdNeeded(e *Engine) {
	for i := range st.emps {
		emp := &st.emps[i]
		if emp.over && !emp.held && emp.status == available {
			if why := st.neededFor(i); why != "" {
				st.hold(e, i, why)
			}
		}
	}
}

// releaseHeld sends home employees past shift end who are idle and no
// longer needed.
func (st *state) releaseHeld(e *Engine) {
	for i := range st.emps {
		if emp := &st.emps[i]; emp.over && emp.status == available {
			st.endOrHold(e, i)
		}
	}
}

// servable reports whether exam xi can still be done safely today: a
// qualified technologist (on duty, held over, or starting later today)
// could finish it within their cap and, for contrast, an on-site
// radiologist could supervise within theirs. It returns a reason if not.
func (st *state) servable(xi int, now simtime.Time) (bool, string) {
	x := &st.cfg.Catalog[xi]
	qualified, fits := false, false
	for i := range st.emps {
		emp := &st.emps[i]
		if emp.cfg.Role != Technologist || !emp.cfg.HasSkill(x.Modality) || emp.shift < 0 || emp.done ||
			!st.mayStaff(i, x.Modality, now) {
			continue
		}
		qualified = true
		fits = fits || max(now, emp.startAt).Add(x.MaxTask()) <= emp.capAt
	}
	switch {
	case !qualified:
		return false, fmt.Sprintf("no %s technologist for the rest of today", x.Modality)
	case !fits:
		return false, fmt.Sprintf("no %s technologist can finish within the %d-hour shift limit",
			x.Modality, st.cfg.Policies.MaxShift/simtime.Hour)
	case !x.Contrast:
		return true, ""
	}
	for i := range st.emps {
		emp := &st.emps[i]
		if emp.cfg.Role == Radiologist && emp.shift >= 0 && !emp.done && !st.plan.Shifts[emp.shift].Remote &&
			max(now, emp.startAt).Add(simtime.Duration(x.RoomTime.Max())) <= emp.capAt {
			return true, ""
		}
	}
	return false, "no radiologist on site to supervise contrast for the rest of today"
}

// mayStaff reports whether technologist i (not gone home) staffs, or will
// staff later today, a room of modality m.
func (st *state) mayStaff(i int, m string, now simtime.Time) bool {
	emp := &st.emps[i]
	if emp.over {
		for _, r := range emp.heldRooms {
			if st.cfg.Rooms[r].Modality == m {
				return true
			}
		}
		return false
	}
	s := &st.plan.Shifts[emp.shift]
	clock := simtime.ClockOf(now)
	for _, room := range st.cfg.Rooms {
		if room.Modality != m {
			continue
		}
		if s.Assignment == room.ID {
			return true
		}
		for _, cv := range s.Covers {
			if cv.Room == room.ID && cv.To > clock {
				return true
			}
		}
	}
	return false
}

// checkCoverage reschedules admitted patients nobody can serve safely
// today, with the reason, instead of letting them wait for nothing.
func (st *state) checkCoverage(e *Engine) {
	for _, vi := range slices.Clone(st.dem.waiting) {
		v := &st.dem.visits[vi]
		if v.state != visitArrived && v.state != visitWaiting {
			continue
		}
		o := &st.dem.orders[v.order]
		if ok, why := st.servable(o.exam, e.Now()); !ok {
			st.rescheduleVisit(e, vi, why)
		}
	}
}

func (st *state) rescheduleVisit(e *Engine, vi int, why string) {
	v := &st.dem.visits[vi]
	v.state, v.endedAt, v.reason = visitRescheduled, e.Now(), why
	st.removeWaiting(vi)
	st.dem.flowToday.noCoverage++
	st.dem.flowTotal.noCoverage++
	o := &st.dem.orders[v.order]
	o.attempt++
	st.offer(e, v.order, st.day+1, fmt.Sprintf("%s %s %s rescheduled after waiting %s: %s",
		visitID(vi), patientID(o.patient), st.examLabel(o.exam), e.Now().Sub(v.arrived), why), "")
}

// --- closing --------------------------------------------------------------

// onSiteFor counts admitted patients for modality m still in the building.
func (st *state) onSiteFor(m string) int {
	n := 0
	for _, vi := range st.dem.waiting {
		if st.cfg.Catalog[st.dem.orders[st.dem.visits[vi].order].exam].Modality == m {
			n++
		}
	}
	return n
}

// anyBusy reports whether any employee is mid-task.
func (st *state) anyBusy() bool {
	for i := range st.emps {
		if st.emps[i].status == busy {
			return true
		}
	}
	return false
}

// scheduledClose handles hours.To: admissions stop; patients on site are
// still served (closing phase) and operations close when they are done.
func (st *state) scheduledClose(e *Engine) {
	st.open = false
	if len(st.dem.waiting) == 0 && !st.anyBusy() {
		st.emit(e.Now(), api.Info, "practice", "Practice closes")
		st.opClosedAt = e.Now()
		return
	}
	st.closing = true
	n := len(st.dem.waiting)
	st.emit(e.Now(), api.Warn, "practice", fmt.Sprintf("Scheduled close: no new patients; %d patient%s still to be seen", n, plural(n)))
}

// operationalClose ends the closing phase once nobody is on site and no
// on-site task remains (§3.4.2).
func (st *state) operationalClose(e *Engine) {
	if !st.closing || len(st.dem.waiting) > 0 || st.anyBusy() {
		return
	}
	st.closing = false
	st.opClosedAt = e.Now()
	f := &st.dem.flowToday
	text := fmt.Sprintf("Operations closed %s (scheduled %s) · %d exam%s after hours", st.cal.Clock(e.Now()),
		simtime.FormatClock(st.plan.Hours.To), f.afterHours, plural(f.afterHours))
	if f.otSec > 0 {
		text += fmt.Sprintf(" · overtime %s ($%d)", f.otSec, (f.otCost+50)/100)
	}
	st.emit(e.Now(), api.Info, "practice", text)
}

// --- eligibility wakes (§3.4 rule 4) ---------------------------------------

// scheduleWakes queues, for each staffed free room with patients waiting
// who are not yet due (when early starts are off), a wake at the earliest
// time one becomes due, so nothing depends on unrelated events.
func (st *state) scheduleWakes(e *Engine) {
	if st.cfg.Policies.StartEarly || (!st.open && !st.closing) {
		return
	}
	for r := range st.rooms {
		rm := &st.rooms[r]
		if rm.visit >= 0 || rm.state != roomIdle {
			continue
		}
		m := rm.cfg.Modality
		next := simtime.Time(-1)
		for _, vi := range st.dem.waiting {
			v := &st.dem.visits[vi]
			if v.state != visitWaiting || v.appt < 0 || st.cfg.Catalog[st.dem.orders[v.order].exam].Modality != m {
				continue
			}
			due := st.apptStart(v.appt).Add(-st.cfg.Policies.EarlyStart)
			if due > e.Now() && (next < 0 || due < next) {
				next = due
			}
		}
		if next >= 0 && next != rm.wakeAt {
			rm.wakeGen++
			rm.wakeAt = next
			e.Push(Event{At: next, Class: ClassDeadline, Kind: kindWake, Gen: rm.wakeGen, A: int64(r)})
		}
	}
}
