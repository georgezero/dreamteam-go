package sim

import (
	"errors"
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
)

type employee struct {
	cfg     *EmployeeConfig
	shift   int // index into state.plan.Shifts; -1 when off today
	status  empStatus
	done    bool // today's shift has ended
	reading bool // radiologist inside a site reading block
	gen     uint32
}

type roomState uint8

const (
	roomClosed roomState = iota
	roomUnstaffed
	roomIdle
)

type room struct {
	cfg   *RoomConfig
	state roomState
	staff []int // employees staffing the room, in employee order
}

// materialize turns employee i's roster entry for today into events. The
// generation bump makes any event from an earlier version stale.
func (st *state) materialize(e *Engine, i int) {
	emp := &st.emps[i]
	emp.gen++
	emp.shift = st.plan.ByEmp[i]
	emp.status, emp.done, emp.reading = offDuty, false, false
	if emp.shift < 0 {
		return
	}
	s := &st.plan.Shifts[emp.shift]
	push := func(at simtime.Duration, c Class, k Kind, b int64) {
		e.Push(Event{At: st.cal.At(st.day, at), Class: c, Kind: k, Gen: emp.gen, A: int64(i), B: b})
	}
	push(s.From, ClassCapacityUp, kindShiftStart, 0)
	push(s.To, ClassCapacityDown, kindShiftEnd, 0)
	for k, b := range s.Breaks {
		push(b.From, ClassCapacityDown, kindBreakStart, int64(k))
		push(b.To, ClassCapacityUp, kindBreakEnd, int64(k))
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
	emp := &st.emps[ev.A]
	s := &st.plan.Shifts[emp.shift]
	name := emp.cfg.Name
	switch ev.Kind {
	case kindShiftStart:
		emp.status = available
		st.emit(e.Now(), api.Info, "staff", name+" on duty · "+st.describe(emp, s))
	case kindShiftEnd:
		emp.status, emp.done, emp.reading = offDuty, true, false
		st.emit(e.Now(), api.Info, "staff", name+" off duty")
	case kindBreakStart:
		emp.status = onBreak
		st.emit(e.Now(), api.Info, "staff", fmt.Sprintf("%s on break until %s", name, simtime.FormatClock(s.Breaks[ev.B].To)))
	case kindBreakEnd:
		emp.status = available
		st.emit(e.Now(), api.Info, "staff", name+" back from break")
	case kindBlockStart:
		emp.reading = true
	case kindBlockEnd:
		emp.reading = false
	}
}

// describe summarizes a shift for the log: "XR, covers CT until 11:30".
func (st *state) describe(emp *employee, s *Shift) string {
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

// restaff recomputes every room's Closed/Unstaffed/Idle state at the end of
// a transaction (SIMULATION_DESIGN §2.5) and logs changes while open.
func (st *state) restaff(now simtime.Time) {
	clock := simtime.ClockOf(now)
	for r := range st.rooms {
		rm := &st.rooms[r]
		var staff []int
		next := roomClosed
		if st.open {
			for i := range st.emps {
				emp := &st.emps[i]
				if emp.status == available && staffs(emp.cfg, &st.plan.Shifts[emp.shift], *rm.cfg, clock) {
					staff = append(staff, i)
				}
			}
			next = roomUnstaffed
			if len(staff) > 0 {
				next = roomIdle
			}
		}
		prev := rm.state
		rm.state, rm.staff = next, staff
		label := rm.cfg.Modality
		switch {
		case prev == roomIdle && next == roomUnstaffed:
			st.emit(now, api.Info, "rooms", label+" unstaffed")
		case prev == roomClosed && next == roomUnstaffed:
			st.emit(now, api.Info, "rooms", label+" unstaffed at opening")
		case prev == roomUnstaffed && next == roomIdle:
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
	if !st.open {
		return DeskClosed, -1
	}
	level, by = DeskSelf, -1
	for i := range st.emps {
		emp := &st.emps[i]
		if emp.status != available {
			continue
		}
		if l := deskLevel(emp.cfg); l > level {
			level, by = l, i
		}
	}
	return level, by
}

func (st *state) checkStaffing() error {
	clock := simtime.ClockOf(st.now)
	for i := range st.emps {
		emp := &st.emps[i]
		if emp.status != offDuty && emp.shift < 0 {
			return fmt.Errorf("%s on duty without a shift", emp.cfg.Name)
		}
		if emp.status == available && !st.plan.Shifts[emp.shift].OnDutyAt(clock) {
			return fmt.Errorf("%s available outside planned duty at %s", emp.cfg.Name, simtime.FormatClock(clock))
		}
		if emp.reading && emp.status == offDuty {
			return fmt.Errorf("%s reading while off duty", emp.cfg.Name)
		}
	}
	for _, rm := range st.rooms {
		switch {
		case rm.state == roomClosed && st.open, rm.state != roomClosed && !st.open:
			return fmt.Errorf("room %s state %d with practice open=%v", rm.cfg.ID, rm.state, st.open)
		case rm.state == roomIdle && len(rm.staff) == 0, rm.state == roomUnstaffed && len(rm.staff) > 0:
			return fmt.Errorf("room %s state %d with %d staff", rm.cfg.ID, rm.state, len(rm.staff))
		}
		for _, i := range rm.staff {
			emp := &st.emps[i]
			if emp.status != available || emp.cfg.Role != Technologist || !emp.cfg.HasSkill(rm.cfg.Modality) {
				return errors.New("room " + rm.cfg.ID + " staffed by an unavailable or unqualified employee")
			}
		}
	}
	return nil
}

var roomStateNames = [...]string{roomClosed: api.RoomClosed, roomUnstaffed: api.RoomUnstaffed, roomIdle: api.RoomIdle}

var deskNames = [...]string{DeskClosed: api.DeskClosed, DeskSelf: api.DeskSelf, DeskOps: api.DeskOps, DeskStaffed: api.DeskStaffed}

func span(s Span) string { return simtime.FormatClock(s.From) + "–" + simtime.FormatClock(s.To) }

// observeStaff fills the room, staff and desk parts of an observation.
func (st *state) observeStaff(o *api.Observation, now simtime.Time) {
	clock := simtime.ClockOf(now)
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
		x := api.Employee{ID: emp.cfg.ID, Name: emp.cfg.Name, Role: string(emp.cfg.Role),
			Skills: slices.Clone(emp.cfg.Skills), Status: api.StaffOff, SiteBP: emp.cfg.SiteAllocationBP,
			Rooms: staffing[i]}
		if emp.shift >= 0 {
			s := &st.plan.Shifts[emp.shift]
			x.Shift, x.Remote = span(s.Span), s.Remote
			switch emp.status {
			case offDuty:
				if !emp.done {
					x.StartsAt = simtime.FormatClock(s.From)
				}
			case onBreak:
				x.Status = api.StaffOnBreak
				for _, b := range s.Breaks {
					if b.Contains(clock) {
						x.EndsAt = simtime.FormatClock(b.To)
					}
				}
			case available:
				x.Status, x.EndsAt = api.StaffOnDuty, simtime.FormatClock(s.To)
				if emp.cfg.Role == Radiologist {
					x.Status = api.StaffOtherDuties
					if emp.reading {
						x.Status = api.StaffReading
					}
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
