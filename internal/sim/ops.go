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
	st.scheduleDeadline(e)
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
	// Desk-capable staff block self-registration only while they can still
	// register someone: on regular duty (desk staff are never held over)
	// and with room before their cap for a registration.
	deskOnDuty := false
	for i := range st.emps {
		if st.deskCanRegister(i) {
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
				if pick < 0 && emp.status == available && !emp.over && deskLevel(emp.cfg) == lv.level &&
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

// deskCanRegister reports whether desk-capable employee i is on regular
// duty and could still fit a registration before their cap (now or after
// their current task or break).
func (st *state) deskCanRegister(i int) bool {
	emp := &st.emps[i]
	b := &st.cfg.Behavior
	if emp.status == offDuty || emp.over || deskLevel(emp.cfg) == DeskClosed {
		return false
	}
	longest := max(b.RegScheduled.Max(), b.RegWalkIn.Max())
	if emp.cfg.Role == OpsManager {
		longest = longest * b.RegOpsBP / 10_000
	}
	return st.now.Add(simtime.Duration(longest)) <= emp.capAt
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
	if x.Contrast {
		v.supervisor = st.supervisor(x)
	}
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

// patientTask is the longest an outstanding patient can still occupy a
// technologist: registration (if not done yet, at the self-registration
// rate) plus the exam and documentation at p100.
func (st *state) patientTask(vi int) simtime.Duration {
	v := &st.dem.visits[vi]
	o := &st.dem.orders[v.order]
	x := &st.cfg.Catalog[o.exam]
	d := x.MaxTask()
	if v.state == visitArrived || v.state == visitRegistering {
		q := st.cfg.Behavior.RegScheduled
		if o.walkIn {
			q = st.cfg.Behavior.RegWalkIn
		}
		d += simtime.Duration(q.Max() * st.cfg.Behavior.RegSelfBP / 10_000)
	}
	return d
}

// earliestStart is the earliest an outstanding patient's exam could start:
// now, or when they become due if early starts are off, and not before
// opening (§5.9).
func (st *state) earliestStart(vi int) simtime.Time {
	v := &st.dem.visits[vi]
	at := st.now
	if v.appt >= 0 && !v.afterGrace && !st.cfg.Policies.StartEarly && st.dem.orders[v.order].priority != Urgent {
		at = max(at, st.apptStart(v.appt).Add(-st.cfg.Policies.EarlyStart))
	}
	if h := st.plan.Hours; h != nil && !st.open && !st.closing {
		at = max(at, st.cal.At(st.day, h.From))
	}
	return at
}

// lastStart is the latest an outstanding patient's work could start with
// employee i and still end by i's cap; ok is false if it never fits.
func (st *state) lastStart(i, vi int) (simtime.Time, bool) {
	emp := &st.emps[i]
	last := emp.capAt.Add(-st.patientTask(vi))
	return last, max(st.earliestStart(vi), emp.startAt) <= last
}

// outstanding reports whether visit vi still needs a technologist:
// arrived, registering or waiting (not yet in the room).
func (st *state) outstanding(vi int) bool {
	s := st.dem.visits[vi].state
	return s == visitArrived || s == visitRegistering || s == visitWaiting
}

// neededFor says why employee i, past shift end, must stay ("" if not):
// a technologist while an outstanding patient for a room they kept can
// still be done by them in time and no technologist on regular duty covers
// that room; a radiologist while supervising a contrast exam, or while a
// waiting contrast patient needs them and no on-site colleague on regular
// duty could supervise instead (§6.9).
func (st *state) neededFor(i int) string {
	emp := &st.emps[i]
	clock := simtime.ClockOf(st.now)
	switch emp.cfg.Role {
	case Technologist:
		for _, r := range emp.heldRooms {
			m := st.cfg.Rooms[r].Modality
			n := 0
			for _, vi := range st.dem.waiting {
				x := &st.cfg.Catalog[st.dem.orders[st.dem.visits[vi].order].exam]
				if _, ok := st.lastStart(i, vi); ok && st.outstanding(vi) && x.Modality == m {
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
		for _, vi := range st.dem.waiting {
			if v := &st.dem.visits[vi]; v.state == visitInExam && v.supervisor == i {
				return "supervising a contrast exam"
			}
		}
		n := 0
		for _, vi := range st.dem.waiting {
			x := &st.cfg.Catalog[st.dem.orders[st.dem.visits[vi].order].exam]
			if !x.Contrast || !st.outstanding(vi) {
				continue
			}
			start := st.earliestStart(vi)
			if start.Add(simtime.Duration(x.RoomTime.Max())) > emp.capAt {
				continue // she couldn't supervise it anyway
			}
			relieved := false
			for j := range st.emps {
				o := &st.emps[j]
				if j != i && o.cfg.Role == Radiologist && o.status != offDuty && !o.over &&
					!st.plan.Shifts[o.shift].Remote && start.Add(simtime.Duration(x.RoomTime.Max())) <= o.capAt {
					relieved = true
				}
			}
			if !relieved {
				n++
			}
		}
		if n > 0 {
			return fmt.Sprintf("%d contrast patient%s need an on-site radiologist", n, plural(n))
		}
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

// servable reports whether exam xi can still be done safely today for a
// patient whose exam could start at the earliest at `from` and who needs
// `task` of a technologist's time: a qualified technologist (on duty, held
// over, or starting later today) could finish within their cap and, for
// contrast, an on-site radiologist could supervise within theirs. It
// returns a reason if not.
func (st *state) servable(xi int, from simtime.Time, task simtime.Duration) (bool, string) {
	x := &st.cfg.Catalog[xi]
	qualified, fits := false, false
	for i := range st.emps {
		emp := &st.emps[i]
		if emp.cfg.Role != Technologist || !emp.cfg.HasSkill(x.Modality) || emp.shift < 0 || emp.done ||
			!st.mayStaff(i, x.Modality, st.now) {
			continue
		}
		qualified = true
		fits = fits || max(from, emp.startAt).Add(task) <= emp.capAt
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
			max(from, emp.startAt).Add(simtime.Duration(x.RoomTime.Max())) <= emp.capAt {
			return true, ""
		}
	}
	return false, "no radiologist on site to supervise contrast for the rest of today"
}

// visitServable applies servable to an admitted patient.
func (st *state) visitServable(vi int) (bool, string) {
	return st.servable(st.dem.orders[st.dem.visits[vi].order].exam, st.earliestStart(vi), st.patientTask(vi))
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
		if ok, why := st.visitServable(vi); !ok {
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

// --- deadline wake (§3.4 rule 4, §6.9) -------------------------------------

// scheduleDeadline queues one wake at the earliest future moment a
// time-based condition changes with no event of its own:
//   - a held employee stops being useful (an outstanding patient's last
//     possible start with them passes);
//   - a desk-capable employee can no longer fit a registration while
//     patients wait to register (self-registration may then begin);
//   - an outstanding patient stops being servable by anyone today.
//
// Without it, those changes would wait for an unrelated event, so outcomes
// would depend on what else happens to occur (review finding 2).
func (st *state) scheduleDeadline(e *Engine) {
	next := simtime.Time(-1)
	consider := func(t simtime.Time) {
		if t > e.Now() && (next < 0 || t < next) {
			next = t
		}
	}
	waitingToRegister := false
	for _, vi := range st.dem.waiting {
		if !st.outstanding(vi) {
			continue
		}
		if st.dem.visits[vi].state == visitArrived {
			waitingToRegister = true
		}
		// Latest start over every qualified technologist who may still
		// staff the modality: one second later the patient is unservable.
		o := &st.dem.orders[st.dem.visits[vi].order]
		m := st.cfg.Catalog[o.exam].Modality
		latest := simtime.Time(-1)
		for i := range st.emps {
			emp := &st.emps[i]
			if emp.cfg.Role != Technologist || !emp.cfg.HasSkill(m) || emp.shift < 0 || emp.done || !st.mayStaff(i, m, st.now) {
				continue
			}
			if last, ok := st.lastStart(i, vi); ok && last > latest {
				latest = last
			}
		}
		if x := &st.cfg.Catalog[o.exam]; x.Contrast && latest >= 0 {
			// …and for contrast, the latest start an on-site radiologist
			// could still supervise.
			radLatest := simtime.Time(-1)
			for i := range st.emps {
				emp := &st.emps[i]
				if emp.cfg.Role == Radiologist && emp.shift >= 0 && !emp.done && !st.plan.Shifts[emp.shift].Remote {
					radLatest = max(radLatest, emp.capAt.Add(-simtime.Duration(x.RoomTime.Max())))
				}
			}
			latest = min(latest, radLatest)
		}
		if latest >= 0 {
			consider(latest.Add(1))
		}
	}
	for i := range st.emps {
		emp := &st.emps[i]
		if emp.held {
			for _, vi := range st.dem.waiting {
				if last, ok := st.lastStart(i, vi); ok && st.outstanding(vi) {
					consider(last.Add(1))
				}
			}
		}
		if waitingToRegister && st.deskCanRegister(i) {
			b := &st.cfg.Behavior
			longest := max(b.RegScheduled.Max(), b.RegWalkIn.Max())
			if emp.cfg.Role == OpsManager {
				longest = longest * b.RegOpsBP / 10_000
			}
			consider(emp.capAt.Add(-simtime.Duration(longest)).Add(1))
		}
	}
	if next >= 0 && next != st.deadlineAt {
		st.deadlineGen++
		st.deadlineAt = next
		e.Push(Event{At: next, Class: ClassDeadline, Kind: kindDeadline, Gen: st.deadlineGen})
	}
}
