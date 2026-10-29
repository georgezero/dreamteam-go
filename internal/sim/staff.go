package sim

import (
	"fmt"
	"slices"
	"strings"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

type empStatus uint8

const (
	offDuty empStatus = iota
	available
	onBreak
	busy
)

type taskKind uint8

const (
	taskNone taskKind = iota
	taskReg           // registering a visit
	taskExam          // running an exam
	taskDoc           // documentation after an exam
	taskRead          // a radiologist reading a study
)

type employee struct {
	// cfg is the employee as working today (a trainee's training view in
	// onboarding); terms are the employee's own terms (employment.go).
	cfg     *EmployeeConfig
	terms   *EmployeeConfig
	shift   int // index into state.plan.Shifts; -1 when off today
	status  empStatus
	done    bool // today's shift has ended and they went home
	reading bool // radiologist inside a site reading block
	gen     uint32

	task      taskKind
	taskVisit int
	taskStudy int // study being read (taskRead)

	// Reading blocks (§6.7): claims start at readFrom (block start plus any
	// overrun owed); readDebt is overrun still owed to later blocks.
	readFrom, blockEndAt simtime.Time
	readDebt             simtime.Duration

	// Today's shift in absolute time, and its hard cap (start +
	// MaxShiftHours, never past accounting close: Policies.ShiftCap,
	// SIMULATION_DESIGN §6.4). rosterEnd is the
	// rostered end; endAt is the scheduled end, later by any approved
	// extension (actions.go). Time past rosterEnd is overtime: approved up
	// to endAt, required (held over) after it.
	startAt, endAt, capAt simtime.Time
	rosterEnd             simtime.Time
	over                  bool  // past the scheduled shift end, still on duty
	held                  bool  // over and kept on because patients need them (§6.9)
	heldRooms             []int // rooms a held technologist staffs

	breakDue   int          // index of a break deferred because they were busy; -1 if none
	breakDueAt simtime.Time // when the deferred break was due
	breakFrom  simtime.Time
	breakUntil simtime.Time
	breakTaken simtime.Duration // unpaid break time taken today

	otToday  simtime.Duration // overtime worked today (past scheduled end)
	weekPaid simtime.Duration // paid time this week, for weekly overtime pay
}

type roomState uint8

const (
	roomClosed roomState = iota
	roomUnstaffed
	roomIdle
	roomOccupied
)

type room struct {
	cfg   *RoomConfig
	state roomState
	staff []int // employees staffing the room, in employee order
	visit int   // visit in the room, -1 when empty
	// Eligibility wake (§3.4 rule 4): the latest pending wake's generation
	// and time.
	wakeGen uint32
	wakeAt  simtime.Time
}

// materialize turns employee i's roster entry for today into events. The
// generation bump makes any event from an earlier version stale.
func (st *state) materialize(e *Engine, i int) {
	emp := &st.emps[i]
	emp.gen++
	emp.shift = st.plan.ByEmp[i]
	emp.cfg = emp.terms
	if c := st.plan.emps[i]; c != nil {
		emp.cfg = c
	}
	emp.status, emp.done, emp.reading = offDuty, false, false
	emp.task, emp.taskVisit, emp.taskStudy, emp.over, emp.held, emp.heldRooms = taskNone, -1, -1, false, false, nil
	emp.readFrom, emp.blockEndAt, emp.readDebt = 0, 0, 0
	emp.breakDue, emp.breakTaken, emp.otToday = -1, 0, 0
	if emp.shift < 0 {
		return
	}
	s := &st.plan.Shifts[emp.shift]
	emp.startAt, emp.endAt = st.cal.At(st.day, s.From), st.cal.At(st.day, s.To)
	emp.rosterEnd = emp.endAt.Add(-st.ext[st.day][i])
	emp.capAt = st.cal.At(st.day, st.cfg.Policies.ShiftCap(s.From))
	push := func(at simtime.Duration, c Class, k Kind, b int64) {
		e.Push(Event{At: st.cal.At(st.day, at), Class: c, Kind: k, Gen: emp.gen, A: int64(i), B: b})
	}
	push(s.From, ClassCapacityUp, kindShiftStart, 0)
	push(s.To, ClassCapacityDown, kindShiftEnd, 0)
	for k, b := range s.Breaks {
		push(b.From, ClassCapacityDown, kindBreakStart, int64(k))
	}
	for k, b := range st.plan.Reading[i] {
		push(b.From, ClassCapacityUp, kindBlockStart, int64(k))
		push(b.To, ClassCapacityDown, kindBlockEnd, int64(k))
	}
	for _, cv := range s.Covers {
		push(cv.From, ClassCapacityUp, kindRosterEdge, 0)
		push(cv.To, ClassCapacityDown, kindRosterEdge, 0)
	}
}

func (st *state) staffEvent(e *Engine, ev Event) {
	i := int(ev.A)
	emp := &st.emps[i]
	s := &st.plan.Shifts[emp.shift]
	name := emp.cfg.Name
	switch ev.Kind {
	case kindShiftStart:
		emp.status = available
		st.emit(e.Now(), api.Info, "staff", name+" on duty · "+st.describe(emp, s))
	case kindShiftEnd:
		emp.over = true
		emp.heldRooms = st.roomsAtShiftEnd(i)
		if emp.breakDue >= 0 {
			st.skipBreak(e, i)
		}
		// Busy: finish-in-progress. Either way, dispatch decides after this
		// instant's coverage checks whether they stay or go home.
	case kindBreakStart:
		b := s.Breaks[ev.B]
		if emp.status == busy || emp.status == onBreak {
			// Busy, or still on an earlier (deferred) break: this one waits
			// its turn. An older break still waiting is skipped, counted.
			if emp.breakDue >= 0 {
				st.skipBreak(e, i)
			}
			emp.breakDue, emp.breakDueAt = int(ev.B), e.Now()
			return
		}
		st.startBreak(e, i, b.To-b.From)
	case kindBreakEnd:
		emp.status = available
		emp.breakTaken += e.Now().Sub(emp.breakFrom)
		st.emit(e.Now(), api.Info, "staff", name+" back from break")
		st.takeDueBreak(e, i)
	case kindBlockStart:
		st.blockStart(i, int(ev.B))
	case kindBlockEnd:
		st.blockEnd(i, int(ev.B))
	}
}

func (st *state) startBreak(e *Engine, i int, length simtime.Duration) {
	emp := &st.emps[i]
	emp.status, emp.breakFrom, emp.breakUntil = onBreak, e.Now(), e.Now().Add(length)
	text := fmt.Sprintf("%s on break until %s", emp.cfg.Name, st.cal.Clock(emp.breakUntil))
	if emp.breakDue >= 0 {
		text += fmt.Sprintf(" (delayed %s)", e.Now().Sub(emp.breakDueAt))
	}
	emp.breakDue = -1
	st.emit(e.Now(), api.Info, "staff", text)
	e.Push(Event{At: emp.breakUntil, Class: ClassCapacityUp, Kind: kindBreakEnd, Gen: emp.gen, A: int64(i)})
}

func (st *state) skipBreak(e *Engine, i int) {
	emp := &st.emps[i]
	emp.breakDue = -1
	st.dem.flowToday.skippedBreaks++
	st.dem.flowTotal.skippedBreaks++
	st.emit(e.Now(), api.Warn, "staff", emp.cfg.Name+" skipped a break (busy)")
}

// taskDone frees employee i after a task: a deferred break starts if it is
// still within the deferral limit and ends before the shift does. Past the
// shift end, dispatch decides whether they stay (§6.9).
func (st *state) taskDone(e *Engine, i int) {
	emp := &st.emps[i]
	emp.status, emp.task, emp.taskVisit = available, taskNone, -1
	st.takeDueBreak(e, i)
}

// takeDueBreak starts a deferred break for a newly free employee if it is
// still within the deferral limit and ends before the shift does, and
// skips it (counted) otherwise.
func (st *state) takeDueBreak(e *Engine, i int) {
	emp := &st.emps[i]
	if emp.breakDue >= 0 {
		b := st.plan.Shifts[emp.shift].Breaks[emp.breakDue]
		length := b.To - b.From
		if e.Now() <= emp.breakDueAt.Add(st.cfg.Policies.BreakDeferMax) && e.Now().Add(length) <= emp.endAt {
			st.startBreak(e, i, length)
		} else {
			st.skipBreak(e, i)
		}
	}
}

// endOrHold decides, for an idle employee past their scheduled shift end,
// whether they stay (held over) or go home.
func (st *state) endOrHold(e *Engine, i int) {
	emp := &st.emps[i]
	if why := st.neededFor(i); why != "" {
		if !emp.held {
			st.hold(e, i, why)
		}
		return
	}
	st.goOff(e, i)
}

func (st *state) hold(e *Engine, i int, why string) {
	emp := &st.emps[i]
	emp.held = true
	st.dem.flowToday.heldOver++
	st.dem.flowTotal.heldOver++
	st.emit(e.Now(), api.Warn, "staff", fmt.Sprintf("%s stays past shift end: %s (overtime)", emp.cfg.Name, why))
}

// goOff sends employee i home and books their overtime and paid time.
func (st *state) goOff(e *Engine, i int) {
	emp := &st.emps[i]
	now := e.Now()
	ot := max(0, now.Sub(emp.rosterEnd))
	paid := now.Sub(emp.startAt) - emp.breakTaken
	cost := st.overtimeCost(emp, paid, ot)
	// The approved part comes first (right after the rostered end); its
	// cost is the overtime cost of paid time up to the end of it.
	appr := min(ot, max(0, emp.endAt.Sub(emp.rosterEnd)))
	apprCost := st.overtimeCost(emp, paid-(ot-appr), appr)
	st.payWages(i, paid)
	if cost > 0 && emp.cfg.Pay == Salaried {
		st.accruePay(now, entryKey{kind: entRadOvertime, a: int64(i), b: st.day}, cost)
	}
	emp.weekPaid += paid
	if emp.cfg.Role == Trainee {
		st.staff[i].Trained += paid
	}
	emp.otToday += ot
	emp.status, emp.done, emp.reading, emp.held, emp.over = offDuty, true, false, false, false
	text := emp.cfg.Name + " off duty"
	if ot > 0 {
		st.dem.flowToday.otSec += ot
		st.dem.flowTotal.otSec += ot
		st.dem.flowToday.otCost += cost
		st.dem.flowTotal.otCost += cost
		for _, f := range []*flowCounts{&st.dem.flowToday, &st.dem.flowTotal} {
			f.otApprovedSec += appr
			f.otApprovedCost += apprCost
		}
		text += fmt.Sprintf(" after %s overtime", ot)
		if cost > 0 {
			text += fmt.Sprintf(" ($%d)", (cost+50)/100)
		}
	}
	st.emit(now, api.Info, "staff", text)
}

// overtimeCost prices the overtime part of a day's paid time for hourly
// staff: at the base rate while the week is under WeeklyHours, then at
// ×OvertimeBP (§6.4). Salaried staff cost nothing extra, except on-site
// radiologists under RadOvertimePaid (radOvertimeCost).
func (st *state) overtimeCost(emp *employee, paid, ot simtime.Duration) int64 {
	if ot <= 0 {
		return 0
	}
	if emp.cfg.Pay != Hourly {
		return st.radOvertimeCost(emp, ot)
	}
	before := emp.weekPaid + paid - ot // paid time this week before the overtime began
	regular := min(ot, max(0, st.cfg.Policies.WeeklyHours-before))
	premium := ot - regular
	rate := emp.cfg.RateCents
	return (rate*int64(regular) + rate*int64(premium)*st.cfg.Policies.OvertimeBP/10_000) / 3600
}

// radOvertimeCost prices a salaried radiologist's time on duty past the
// rostered end under RadOvertimePaid: the annual salary's hourly
// equivalent over a year of 52 weeks of WeeklyHours, straight time, with
// no site-allocation factor (the time is on site), rounded down to the
// cent (V0.7_OVERTIME_DESIGN.md §3). Zero otherwise, and for remote
// shifts.
func (st *state) radOvertimeCost(emp *employee, ot simtime.Duration) int64 {
	p := &st.cfg.Policies
	if !st.cfg.Finance.RadOvertimePaid || ot <= 0 || emp.cfg.Role != Radiologist || emp.cfg.Pay != Salaried || p.WeeklyHours <= 0 ||
		(emp.shift >= 0 && st.plan.Shifts[emp.shift].Remote) {
		return 0
	}
	return emp.cfg.RateCents * int64(ot) / (52 * int64(p.WeeklyHours))
}

// roomsAtShiftEnd lists the rooms technologist i staffs at the last second
// of the shift: the rooms they keep if held over.
func (st *state) roomsAtShiftEnd(i int) []int {
	emp := &st.emps[i]
	if emp.cfg.Role != Technologist {
		return nil
	}
	s := &st.plan.Shifts[emp.shift]
	var out []int
	for r := range st.cfg.Rooms {
		if staffs(emp.cfg, s, st.cfg.Rooms[r], s.To-1) {
			out = append(out, r)
		}
	}
	return out
}

// techStaffs reports whether technologist i, on duty, staffs room r now:
// by roster (assignment or cover window) during the shift, by the rooms
// kept at shift end when held over.
func (st *state) techStaffs(i, r int, clock simtime.Duration) bool {
	emp := &st.emps[i]
	if emp.cfg.Role != Technologist || (emp.status != available && emp.status != busy) {
		return false
	}
	if emp.over {
		return slices.Contains(emp.heldRooms, r)
	}
	return staffs(emp.cfg, &st.plan.Shifts[emp.shift], st.cfg.Rooms[r], clock)
}

// describe summarizes a shift for the log: "XR, covers CT until 11:30".
func (st *state) describe(emp *employee, s *Shift) string {
	if emp.cfg.Role == Trainee {
		return "training (paid, no patient capacity) · until " + simtime.FormatClock(s.To)
	}
	var parts []string
	switch s.Assignment {
	case AssignDesk:
		parts = append(parts, "front desk")
	case AssignAdmin:
		parts = append(parts, "admin")
	case AssignReading:
		where := "on site"
		if s.Remote {
			where = "remote"
		}
		parts = append(parts, fmt.Sprintf("reading %s, %d%% allocation", where, emp.cfg.SiteAllocationBP/100))
	default:
		parts = append(parts, st.roomByID(s.Assignment).Modality)
	}
	for _, cv := range s.Covers {
		m := st.roomByID(cv.Room).Modality
		switch {
		case cv.From <= s.From && cv.To >= s.To:
			parts = append(parts, "covers "+m)
		case cv.From <= s.From:
			parts = append(parts, "covers "+m+" until "+simtime.FormatClock(cv.To))
		case cv.To >= s.To:
			parts = append(parts, "covers "+m+" from "+simtime.FormatClock(cv.From))
		default:
			parts = append(parts, fmt.Sprintf("covers %s %s–%s", m, simtime.FormatClock(cv.From), simtime.FormatClock(cv.To)))
		}
	}
	return strings.Join(parts, ", ") + " · until " + simtime.FormatClock(s.To)
}

func (st *state) roomByID(id string) *RoomConfig {
	for i := range st.cfg.Rooms {
		if st.cfg.Rooms[i].ID == id {
			return &st.cfg.Rooms[i]
		}
	}
	panic("sim: unknown room " + id)
}

// restaff recomputes every room's state at the end of a transaction
// (SIMULATION_DESIGN §2.5, §6.9) and logs staffing changes while open.
// After scheduled close a room stays in service only while it has an exam
// or admitted patients waiting for its modality.
func (st *state) restaff(now simtime.Time) {
	clock := simtime.ClockOf(now)
	for r := range st.rooms {
		rm := &st.rooms[r]
		buf := st.staffBuf[:0]
		for i := range st.emps {
			if st.techStaffs(i, r, clock) {
				buf = append(buf, i)
			}
		}
		st.staffBuf = buf
		// Keep the room's list when unchanged (no allocation per dispatch);
		// a changed list is a fresh copy, never the scratch buffer.
		staff := rm.staff
		switch {
		case len(buf) == 0:
			staff = nil
		case !slices.Equal(buf, staff):
			staff = slices.Clone(buf)
		}
		next := roomClosed
		switch {
		case rm.visit >= 0:
			next = roomOccupied
		case st.open || (st.closing && st.onSiteFor(rm.cfg.Modality) > 0):
			next = roomUnstaffed
			if len(staff) > 0 {
				next = roomIdle
			}
		}
		if next == roomClosed {
			staff = nil
		}
		prev := rm.state
		rm.state, rm.staff = next, staff
		if !st.open {
			continue
		}
		label := rm.cfg.Modality
		staffed := func(s roomState) bool { return s == roomIdle || s == roomOccupied }
		switch {
		case staffed(prev) && next == roomUnstaffed:
			st.emit(now, api.Info, "rooms", label+" unstaffed")
		case prev == roomClosed && next == roomUnstaffed:
			st.emit(now, api.Info, "rooms", label+" unstaffed at opening")
		case prev == roomUnstaffed && staffed(next):
			st.emit(now, api.Info, "rooms", label+" staffed · "+st.names(staff))
		}
	}
}

func (st *state) names(idx []int) string {
	var ns []string
	for _, i := range idx {
		ns = append(ns, st.emps[i].cfg.Name)
	}
	return strings.Join(ns, ", ")
}

// desk returns who registers patients now.
func (st *state) desk() (level int, by int) {
	if !st.open && !st.closing {
		return DeskClosed, -1
	}
	level, by = DeskSelf, -1
	for i := range st.emps {
		emp := &st.emps[i]
		if emp.status == offDuty || emp.status == onBreak {
			continue
		}
		if l := deskLevel(emp.cfg); l > level {
			level, by = l, i
		}
	}
	return level, by
}

// capName names what sets a shift's cap: "the 12-hour <what>", or accounting
// close for a shift starting late enough that it binds first.
func (st *state) capName(startAt, capAt simtime.Time, what string) string {
	if capAt == startAt.Add(st.cfg.Policies.MaxShift) {
		return fmt.Sprintf("the %d-hour %s", st.cfg.Policies.MaxShift/simtime.Hour, what)
	}
	return "the end of the working day (accounting close)"
}

func (st *state) checkStaffing() error {
	for i := range st.emps {
		emp := &st.emps[i]
		if emp.status == offDuty {
			if emp.task != taskNone || emp.held {
				return fmt.Errorf("%s off duty with a task or held", emp.cfg.Name)
			}
			if emp.reading {
				return fmt.Errorf("%s reading while off duty", emp.cfg.Name)
			}
			continue
		}
		if emp.shift < 0 {
			return fmt.Errorf("%s on duty without a shift", emp.cfg.Name)
		}
		switch {
		case st.now < emp.startAt:
			return fmt.Errorf("%s on duty before the shift starts", emp.cfg.Name)
		case st.now > emp.capAt:
			return fmt.Errorf("%s on duty past the shift cap %s", emp.cfg.Name, st.cal.Clock(emp.capAt))
		case !emp.over && st.now >= emp.endAt:
			return fmt.Errorf("%s past shift end but not marked over", emp.cfg.Name)
		case (emp.status == busy) != (emp.task != taskNone):
			return fmt.Errorf("%s status %d with task %d", emp.cfg.Name, emp.status, emp.task)
		case emp.held && !emp.over:
			return fmt.Errorf("%s held but not past shift end", emp.cfg.Name)
		}
	}
	for r, rm := range st.rooms {
		switch {
		case rm.state == roomOccupied:
			if rm.visit < 0 {
				return fmt.Errorf("room %s occupied with no visit", rm.cfg.ID)
			}
			v := &st.dem.visits[rm.visit]
			if v.state != visitInExam || v.room != r || st.emps[v.examBy].task != taskExam || st.emps[v.examBy].taskVisit != rm.visit {
				return fmt.Errorf("room %s occupancy inconsistent with %s", rm.cfg.ID, visitID(rm.visit))
			}
			if st.cfg.Catalog[st.dem.orders[v.order].exam].Contrast {
				if v.supervisor < 0 {
					return fmt.Errorf("%s contrast exam without a supervisor", visitID(rm.visit))
				}
				rad := &st.emps[v.supervisor]
				if rad.status == offDuty || st.plan.Shifts[rad.shift].Remote || st.now > rad.capAt {
					return fmt.Errorf("%s contrast exam: %s not on duty on site within cap", visitID(rm.visit), rad.cfg.Name)
				}
			}
		case rm.visit >= 0:
			return fmt.Errorf("room %s has a visit but is %d", rm.cfg.ID, rm.state)
		case rm.state == roomIdle && len(rm.staff) == 0, rm.state == roomUnstaffed && len(rm.staff) > 0:
			return fmt.Errorf("room %s state %d with %d staff", rm.cfg.ID, rm.state, len(rm.staff))
		case rm.state != roomClosed && !st.open && !st.closing:
			return fmt.Errorf("room %s in service while the practice is closed", rm.cfg.ID)
		}
		for _, i := range rm.staff {
			emp := &st.emps[i]
			if emp.cfg.Role != Technologist || !emp.cfg.HasSkill(rm.cfg.Modality) {
				return fmt.Errorf("room %s staffed by an unqualified employee", rm.cfg.ID)
			}
		}
	}
	return nil
}

var roomStateNames = [...]string{roomClosed: api.RoomClosed, roomUnstaffed: api.RoomUnstaffed,
	roomIdle: api.RoomIdle, roomOccupied: api.RoomOccupied}

var deskNames = [...]string{DeskClosed: api.DeskClosed, DeskSelf: api.DeskSelf, DeskOps: api.DeskOps, DeskStaffed: api.DeskStaffed}

func span(s Span) string { return simtime.FormatClock(s.From) + "–" + simtime.FormatClock(s.To) }

// overtimeNow is employee i's overtime today including time still running.
func (st *state) overtimeNow(i int, now simtime.Time) simtime.Duration {
	emp := &st.emps[i]
	ot := emp.otToday
	if emp.status != offDuty {
		ot += max(0, now.Sub(emp.rosterEnd))
	}
	return ot
}

// observeStaff fills the room, staff and desk parts of an observation.
func (st *state) observeStaff(o *api.Observation, now simtime.Time) {
	if h := st.plan.Hours; h != nil {
		o.Practice.Hours = span(*h)
	}
	staffing := make([][]string, len(st.emps))
	for _, rm := range st.rooms {
		r := api.Room{ID: rm.cfg.ID, Name: rm.cfg.Name, Modality: rm.cfg.Modality, State: roomStateNames[rm.state]}
		for _, i := range rm.staff {
			r.StaffBy = append(r.StaffBy, st.emps[i].cfg.ID)
			staffing[i] = append(staffing[i], rm.cfg.ID)
		}
		if rm.visit >= 0 {
			r.Current = st.examView(rm.visit, now)
		}
		o.Rooms = append(o.Rooms, r)
	}
	for k := range o.Rooms {
		for _, i := range st.rooms[k].staff {
			if len(staffing[i]) > 1 {
				o.Rooms[k].Shared = true
			}
		}
	}
	for i := range st.emps {
		emp := &st.emps[i]
		t := emp.terms
		x := api.Employee{ID: t.ID, Name: t.Name, Role: string(t.Role),
			Skills: slices.Clone(t.Skills), Status: api.StaffOff, SiteBP: st.siteBP(i, st.day),
			Rooms: staffing[i], Held: emp.held, OvertimeMin: int64(st.overtimeNow(i, now) / simtime.Minute)}
		x.WeekPaidMin = int64(emp.weekPaid / simtime.Minute)
		st.observeEmployment(&x, i)
		if t.Pay == Hourly {
			x.HourlyCents = t.RateCents
		}
		if emp.shift >= 0 {
			s := &st.plan.Shifts[emp.shift]
			x.Shift, x.Remote = span(s.Span), s.Remote
			x.RosterEnd, x.CapAt = st.cal.Clock(emp.rosterEnd), st.cal.Clock(emp.capAt)
			x.ApprovedMin = int64(emp.endAt.Sub(emp.rosterEnd) / simtime.Minute)
			if emp.status != offDuty {
				x.ApprovedOTMin = int64(max(0, min(now, emp.endAt).Sub(emp.rosterEnd)) / simtime.Minute)
			}
			switch emp.status {
			case offDuty:
				if !emp.done {
					x.StartsAt = simtime.FormatClock(s.From)
				}
			case onBreak:
				x.Status, x.EndsAt = api.StaffOnBreak, st.cal.Clock(emp.breakUntil)
			case busy:
				x.Status, x.EndsAt, x.Task = api.StaffBusy, simtime.FormatClock(s.To), st.taskText(i)
				if emp.task == taskRead {
					x.Status = api.StaffReading
				}
			case available:
				x.Status, x.EndsAt = api.StaffOnDuty, simtime.FormatClock(s.To)
				if emp.cfg.Role == Trainee {
					x.Status = api.StaffTraining
				}
				if emp.cfg.Role == Radiologist {
					x.Status = api.StaffOtherDuties
					if emp.reading && !emp.over {
						x.Status = api.StaffReadingFree
					}
				}
			}
			for _, vi := range st.dem.waiting {
				if v := &st.dem.visits[vi]; v.state == visitInExam && v.supervisor == i {
					x.Supervising = patientID(st.dem.orders[v.order].patient)
				}
			}
		}
		o.Staff = append(o.Staff, x)
	}
	level, by := st.desk()
	o.Desk.Coverage = deskNames[level]
	if by >= 0 {
		o.Desk.By = st.emps[by].cfg.ID
	}
}

func (st *state) taskText(i int) string {
	emp := &st.emps[i]
	if emp.taskVisit < 0 {
		return ""
	}
	v := &st.dem.visits[emp.taskVisit]
	o := &st.dem.orders[v.order]
	p := patientID(o.patient)
	switch emp.task {
	case taskReg:
		return "registering " + p
	case taskExam:
		return st.cfg.Rooms[v.room].Modality + " exam " + p
	case taskDoc:
		return "documenting " + p
	case taskRead:
		s := &st.dem.studies[emp.taskStudy]
		x := &st.cfg.Catalog[s.exam]
		return "reading " + studyID(emp.taskStudy) + " " + x.Modality + " " + x.Name
	}
	return ""
}
