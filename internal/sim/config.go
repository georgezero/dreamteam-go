package sim

import (
	"slices"

	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Config is a compiled, validated scenario: everything the engine needs to
// build a practice. The scenario package produces it from JSON; the engine
// never reads files.
type Config struct {
	ID, Name   string
	Start      simtime.Date     // first observed day (day 1)
	StartClock simtime.Duration // when play begins on Start
	WarmupDays int64
	// RotaWeekA is a date in rota week A; weeks run Monday to Sunday and
	// alternate A, B (SIMULATION_DESIGN §6.3).
	RotaWeekA        simtime.Date
	OpeningCashCents int64

	Hours     [7]*Span // by simtime.Weekday; nil means closed
	Rooms     []RoomConfig
	Employees []EmployeeConfig
	Policies  Policies
}

// Span is a half-open interval [From, To) of time of day.
type Span struct{ From, To simtime.Duration }

// Contains reports whether c lies in [From, To).
func (s Span) Contains(c simtime.Duration) bool { return s.From <= c && c < s.To }

// RoomConfig describes one room (one scanner in V0.1).
type RoomConfig struct {
	ID, Name, Modality string
}

// Role is an employee's job.
type Role string

const (
	FrontDesk    Role = "front_desk"
	Technologist Role = "technologist"
	Radiologist  Role = "radiologist"
	OpsManager   Role = "ops_manager"
)

// PayType says whether overtime rules apply.
type PayType string

const (
	Hourly   PayType = "hourly"   // non-exempt
	Salaried PayType = "salaried" // exempt
)

// EmployeeConfig is one employee and their roster.
type EmployeeConfig struct {
	ID, Name string
	Role     Role
	Skills   []string // modalities for technologists; "registration" for desk-capable staff
	Pay      PayType
	// RateCents is the hourly rate (Hourly) or annual salary (Salaried).
	RateCents int64
	// SiteAllocationBP is a radiologist's share of paid time committed to
	// this site, in basis points.
	SiteAllocationBP int64
	Roster           [7]*ShiftTemplate // by simtime.Weekday; nil means off
}

// HasSkill reports whether the employee holds skill s.
func (c *EmployeeConfig) HasSkill(s string) bool { return slices.Contains(c.Skills, s) }

// Rota weeks.
const (
	WeekBoth = 0
	WeekA    = 1
	WeekB    = 2
)

// ShiftTemplate is a roster entry for one weekday.
type ShiftTemplate struct {
	Span
	Weeks      int    // WeekBoth, WeekA or WeekB
	Breaks     []Span // unpaid, inside the shift
	Assignment string // room ID, "desk", "reading" or "admin"
	Covers     []Cover
	Remote     bool // radiologists: reads but cannot supervise contrast
}

// Cover is a window in which a technologist also staffs another room.
type Cover struct {
	Room string
	Span
}

// Policies are the operating rules (SIMULATION_DESIGN §5.5, §5.9, §6.4, §6.8).
type Policies struct {
	WalkIn              string
	Worklist            string
	Late                string
	Grace               simtime.Duration
	OvertimeAllowance   simtime.Duration
	MaxShift            simtime.Duration
	ContrastSupervision string
}

// Assignment values other than room IDs.
const (
	AssignDesk    = "desk"
	AssignReading = "reading"
	AssignAdmin   = "admin"
)

// RotaWeek returns WeekA or WeekB for date d.
func (c *Config) RotaWeek(d simtime.Date) int {
	monday := func(x simtime.Date) int64 {
		return x.Ordinal() - int64((x.Weekday()+6)%7)
	}
	weeks := (monday(d) - monday(c.RotaWeekA)) / 7
	if weeks%2 == 0 {
		return WeekA
	}
	return WeekB
}

// Shift is a roster entry materialized for one day. Times are seconds since
// that day's midnight so a plan can be built without a calendar.
type Shift struct {
	Employee int // index into Config.Employees
	ShiftTemplate
}

// OnDutyAt reports whether the planned shift has the employee working at c
// (inside the shift and not on a planned break).
func (s *Shift) OnDutyAt(c simtime.Duration) bool {
	if !s.Contains(c) {
		return false
	}
	for _, b := range s.Breaks {
		if b.Contains(c) {
			return false
		}
	}
	return true
}

// CoversRoom reports whether the shift assigns the employee to room at c,
// either as primary assignment or through a cover window. It does not check
// duty status.
func (s *Shift) CoversRoom(room string, c simtime.Duration) bool {
	if s.Assignment == room {
		return true
	}
	for _, cv := range s.Covers {
		if cv.Room == room && cv.Contains(c) {
			return true
		}
	}
	return false
}

// DayPlan is the roster materialized for one date.
type DayPlan struct {
	Date    simtime.Date
	Week    int
	Hours   *Span // nil when closed
	Shifts  []Shift
	ByEmp   []int // index into Shifts per employee, -1 when off
	cfg     *Config
	Reading [][]Span // per employee: site reading blocks (radiologists)
}

// Plan materializes the roster for date d.
func (c *Config) Plan(d simtime.Date) DayPlan {
	wd := d.Weekday()
	p := DayPlan{Date: d, Week: c.RotaWeek(d), Hours: c.Hours[wd], cfg: c,
		ByEmp: make([]int, len(c.Employees)), Reading: make([][]Span, len(c.Employees))}
	for i := range c.Employees {
		p.ByEmp[i] = -1
		t := c.Employees[i].Roster[wd]
		if t == nil || (t.Weeks != WeekBoth && t.Weeks != p.Week) {
			continue
		}
		p.ByEmp[i] = len(p.Shifts)
		p.Shifts = append(p.Shifts, Shift{Employee: i, ShiftTemplate: *t})
		if c.Employees[i].Role == Radiologist {
			p.Reading[i] = ReadingBlocks(t.Span, c.Employees[i].SiteAllocationBP)
		}
	}
	return p
}

// ReadingCycle is the length of one site/other-duties cycle (§6.7): for a
// 60% allocation, 60 min reading for this site then 40 min other duties.
const ReadingCycle = 100 * simtime.Minute

// ReadingBlocks splits a shift into site reading blocks for allocation bp.
func ReadingBlocks(shift Span, bp int64) []Span {
	site := simtime.Duration(int64(ReadingCycle) * bp / 10_000)
	if site <= 0 {
		return nil
	}
	var out []Span
	for at := shift.From; at < shift.To; at += ReadingCycle {
		out = append(out, Span{at, min(at+site, shift.To)})
	}
	return out
}

// Coverage is the planned staffing of one day, as segments between
// consecutive roster boundaries.
type Coverage struct {
	Plan     *DayPlan
	Segments []Segment
}

// Desk coverage levels.
const (
	DeskClosed  = iota // practice closed
	DeskSelf           // technologists self-register
	DeskOps            // operations manager covers
	DeskStaffed        // front desk on duty
)

// Segment is an interval with constant planned staffing.
type Segment struct {
	Span
	Open  bool
	Rooms [][]int // per room: employees staffing it
	Desk  int
	// Supervision is true when a radiologist is on duty and on site.
	Supervision bool
	// Reading counts radiologists in a site reading block.
	Reading int
}

// Coverage evaluates the plan between all of its boundaries.
func (p *DayPlan) Coverage() Coverage {
	cuts := []simtime.Duration{0, simtime.Day}
	if p.Hours != nil {
		cuts = append(cuts, p.Hours.From, p.Hours.To)
	}
	for _, s := range p.Shifts {
		cuts = append(cuts, s.From, s.To)
		for _, b := range s.Breaks {
			cuts = append(cuts, b.From, b.To)
		}
		for _, cv := range s.Covers {
			cuts = append(cuts, cv.From, cv.To)
		}
	}
	for _, blocks := range p.Reading {
		for _, b := range blocks {
			cuts = append(cuts, b.From, b.To)
		}
	}
	slices.Sort(cuts)
	cuts = slices.Compact(cuts)
	cov := Coverage{Plan: p}
	for i := 0; i+1 < len(cuts); i++ {
		at := cuts[i]
		seg := Segment{Span: Span{at, cuts[i+1]}, Rooms: make([][]int, len(p.cfg.Rooms))}
		seg.Open = p.Hours != nil && p.Hours.Contains(at)
		for si := range p.Shifts {
			s := &p.Shifts[si]
			if !s.OnDutyAt(at) {
				continue
			}
			emp := &p.cfg.Employees[s.Employee]
			if seg.Open {
				for r, room := range p.cfg.Rooms {
					if staffs(emp, s, room, at) {
						seg.Rooms[r] = append(seg.Rooms[r], s.Employee)
					}
				}
				seg.Desk = max(seg.Desk, deskLevel(emp))
			}
			if emp.Role == Radiologist {
				if !s.Remote {
					seg.Supervision = true
				}
				for _, b := range p.Reading[s.Employee] {
					if b.Contains(at) {
						seg.Reading++
					}
				}
			}
		}
		if seg.Open && seg.Desk == DeskClosed {
			seg.Desk = DeskSelf
		}
		// Merge with the previous segment when nothing changed.
		if n := len(cov.Segments); n > 0 && cov.Segments[n-1].sameAs(&seg) {
			cov.Segments[n-1].To = seg.To
			continue
		}
		cov.Segments = append(cov.Segments, seg)
	}
	return cov
}

// staffs is the single definition of "this employee staffs this room now",
// given that they are on duty and the practice is open. The runtime uses it
// too, so the coverage report and the simulation cannot disagree.
func staffs(emp *EmployeeConfig, s *Shift, room RoomConfig, at simtime.Duration) bool {
	return emp.Role == Technologist && emp.HasSkill(room.Modality) && s.CoversRoom(room.ID, at)
}

func deskLevel(emp *EmployeeConfig) int {
	switch {
	case emp.Role == FrontDesk:
		return DeskStaffed
	case emp.Role == OpsManager && emp.HasSkill("registration"):
		return DeskOps
	}
	return DeskClosed
}

func (s *Segment) sameAs(t *Segment) bool {
	if s.Open != t.Open || s.Desk != t.Desk || s.Supervision != t.Supervision || s.Reading != t.Reading {
		return false
	}
	for r := range s.Rooms {
		if !slices.Equal(s.Rooms[r], t.Rooms[r]) {
			return false
		}
	}
	return true
}

// StaffedSeconds returns, per room, the planned staffed time within open
// hours, plus the open time itself.
func (c *Coverage) StaffedSeconds() (rooms []simtime.Duration, open simtime.Duration) {
	rooms = make([]simtime.Duration, len(c.Plan.cfg.Rooms))
	for _, s := range c.Segments {
		if !s.Open {
			continue
		}
		d := s.To - s.From
		open += d
		for r := range s.Rooms {
			if len(s.Rooms[r]) > 0 {
				rooms[r] += d
			}
		}
	}
	return rooms, open
}
