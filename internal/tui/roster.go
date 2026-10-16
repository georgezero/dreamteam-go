package tui

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// The roster editor (V0.2 Slice 3): opened from a roster row of the staff
// screen, it edits one employee's A/B pattern — weekly (both rota weeks
// alike) or fortnightly — for a chosen effective Monday, previews it live
// with PreviewStaffing (read-only) and submits set_roster through the
// session's Apply. Only shifts, breaks, rooms and covers change: the
// contract does not, whatever is rostered.

// Editor fields of the cursor day.
const (
	edStart = iota
	edEnd
	edBreak
	edRoom
	edCover
	edFields
)

var edFieldNames = [edFields]string{"start", "end", "break", "room", "cover"}

var weekdayNames = [7]string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}

// rosterEditor is an open roster edit.
type rosterEditor struct {
	emp    string
	plan   api.RosterPlan
	r      api.Roster
	weekly bool  // edit both rota weeks alike
	cell   int   // 0–6 week A (or both), 7–13 week B
	field  int   // edStart …
	weeks  int64 // effective date: weeks after the earliest
	remove bool  // withdraw the pending change on that date
	note   string
}

// rosterable lists the employees whose roster can be edited.
func (m *Model) rosterable() []api.Employee {
	var out []api.Employee
	for _, e := range m.obs.Staff {
		if (e.Role == "technologist" || e.Role == "front_desk") && e.HourlyCents > 0 && e.Employment != api.EmploymentTerminated {
			if p, ok := m.sess.RosterPlan(e.ID); ok && p.Editable {
				out = append(out, e)
			}
		}
	}
	return out
}

// openRoster starts editing employee id from the roster in force on the
// earliest effective date.
func (m *Model) openRoster(id string) {
	p, ok := m.sess.RosterPlan(id)
	if !ok || !p.Editable {
		m.status = "That roster cannot be changed."
		return
	}
	ed := &rosterEditor{emp: id, plan: p}
	ed.r = ed.inForce(ed.effective())
	ed.weekly = reflect.DeepEqual(ed.r.A, ed.r.B)
	m.staff.edit = ed
}

// effective is the chosen effective date.
func (ed *rosterEditor) effective() string {
	d, _ := simtime.ParseDate(ed.plan.EarliestEffective)
	return d.AddDays(7 * ed.weeks).String()
}

// inForce is a deep copy of the roster version in force on date.
func (ed *rosterEditor) inForce(date string) api.Roster {
	var r api.Roster
	for _, v := range ed.plan.Versions {
		if v.Effective <= date {
			r = v.Roster
		}
	}
	c := r
	for _, w := range []*api.Week{&c.A, &c.B} {
		for _, d := range ed.days(w) {
			if *d != nil {
				s := **d
				s.Breaks, s.Covers = slices.Clone(s.Breaks), slices.Clone(s.Covers)
				*d = &s
			}
		}
	}
	return c
}

// pending is the version effective exactly on date, if any.
func (ed *rosterEditor) pending(date string) bool {
	return slices.ContainsFunc(ed.plan.Versions, func(v api.RosterVersion) bool { return v.Effective == date && v.Pending })
}

func (ed *rosterEditor) days(w *api.Week) [7]**api.Shift {
	return [7]**api.Shift{&w.Mon, &w.Tue, &w.Wed, &w.Thu, &w.Fri, &w.Sat, &w.Sun}
}

// at is the cursor cell's day slot (week A for a weekly edit).
func (ed *rosterEditor) at(cell int) **api.Shift {
	w := &ed.r.A
	if cell >= 7 {
		w = &ed.r.B
	}
	return ed.days(w)[cell%7]
}

// action is the set_roster the editor describes.
func (ed *rosterEditor) action() api.Action {
	if ed.remove {
		return api.Action{Type: api.ActSetRoster, Op: api.RosterOpRemove, Employee: ed.emp, Effective: ed.effective()}
	}
	r := ed.r
	if ed.weekly {
		r.B = r.A
	}
	return api.Action{Type: api.ActSetRoster, Op: api.RosterOpSet, Employee: ed.emp, Effective: ed.effective(), Roster: &r}
}

func clock(s string) simtime.Duration {
	c, _ := simtime.ParseClock(s)
	return c
}

func hhmm(c simtime.Duration) string {
	return simtime.FormatClock(max(0, min(c, simtime.Day-simtime.Minute)))
}

// edit applies a change to the cursor day (both weeks when weekly).
func (ed *rosterEditor) edit(f func(s **api.Shift)) {
	f(ed.at(ed.cell))
	if ed.weekly {
		s := *ed.at(ed.cell)
		if s != nil {
			c := *s
			c.Breaks, c.Covers = slices.Clone(c.Breaks), slices.Clone(c.Covers)
			s = &c
		}
		*ed.at(ed.cell + 7) = s
	}
}

// fitShift keeps breaks and covers inside the shift (dropping empty ones).
func fitShift(s *api.Shift) {
	from, to := clock(s.Start), clock(s.End)
	var brk [][2]string
	for _, b := range s.Breaks {
		if clock(b[0]) > from && clock(b[1]) < to {
			brk = append(brk, b)
		}
	}
	s.Breaks = brk
	if s.Breaks == nil {
		s.Breaks = [][2]string{}
	}
	covers := []api.Cover{}
	for _, c := range s.Covers {
		f, u := max(clock(c.From), from), min(clock(c.Until), to)
		if f < u {
			covers = append(covers, api.Cover{Room: c.Room, From: hhmm(f), Until: hhmm(u)})
		}
	}
	s.Covers = covers
}

// newShift is a day's default shift: a copy of the nearest rostered day,
// else 08:00–16:30 with a half-hour break.
func (ed *rosterEditor) newShift() *api.Shift {
	for k := 1; k < 14; k++ {
		for _, c := range []int{ed.cell - k, ed.cell + k} {
			if c >= 0 && c < 14 && *ed.at(c) != nil {
				s := **ed.at(c)
				s.Breaks, s.Covers = slices.Clone(s.Breaks), slices.Clone(s.Covers)
				return &s
			}
		}
	}
	room := "desk"
	if len(ed.plan.Assignments) > 0 {
		room = ed.plan.Assignments[0]
	}
	return &api.Shift{Start: "08:00", End: "16:30", Breaks: [][2]string{{"12:00", "12:30"}}, Assignment: room, Covers: []api.Cover{}}
}

// rosterKey handles the editor's keys; it reports whether k was one.
func (m *Model) rosterKey(k string) bool {
	ed := m.staff.edit
	cells := 14
	if ed.weekly {
		cells = 7
	}
	step := simtime.Duration(0)
	switch k {
	case "esc":
		m.staff.edit = nil
	case "left":
		ed.cell = (ed.cell + cells - 1) % cells
	case "right":
		ed.cell = (ed.cell + 1) % cells
	case "up", "k":
		ed.field = (ed.field + edFields - 1) % edFields
	case "down", "j":
		ed.field = (ed.field + 1) % edFields
	case ",", ".":
		if k == "," {
			ed.weeks = max(0, ed.weeks-1)
		} else {
			ed.weeks++
		}
		ed.remove = ed.remove && ed.pending(ed.effective())
	case "r":
		ed.r = ed.inForce(ed.effective())
		ed.note = "Reset to the roster in force on " + shortDate(ed.effective()) + "."
	case "w":
		ed.weekly = !ed.weekly
		ed.cell %= 7
		if ed.weekly && !reflect.DeepEqual(ed.r.A, ed.r.B) {
			ed.r.B = ed.r.A
			ed.note = "Weekly: week B now repeats week A."
		}
	case "x":
		if ed.pending(ed.effective()) {
			ed.remove = !ed.remove
		} else {
			ed.note = "No pending change on " + shortDate(ed.effective()) + " to withdraw."
		}
	case "o":
		ed.edit(func(s **api.Shift) {
			if *s != nil {
				*s = nil
			} else {
				*s = ed.newShift()
			}
		})
	case "b":
		ed.edit(func(s **api.Shift) {
			if *s == nil {
				return
			}
			if len((*s).Breaks) > 0 {
				(*s).Breaks = [][2]string{}
				return
			}
			mid := (clock((*s).Start) + clock((*s).End)) / 2 / (15 * simtime.Minute) * (15 * simtime.Minute)
			(*s).Breaks = [][2]string{{hhmm(mid), hhmm(mid + 30*simtime.Minute)}}
		})
	case "c":
		ed.edit(func(s **api.Shift) {
			if *s == nil {
				return
			}
			var rooms []string
			for _, a := range ed.plan.Assignments {
				if a != (*s).Assignment && a != "desk" {
					rooms = append(rooms, a)
				}
			}
			cur := ""
			if len((*s).Covers) > 0 {
				cur = (*s).Covers[0].Room
			}
			next := ""
			if i := slices.Index(rooms, cur); i+1 < len(rooms) {
				next = rooms[i+1]
			}
			(*s).Covers = []api.Cover{}
			if next != "" {
				(*s).Covers = []api.Cover{{Room: next, From: (*s).Start, Until: (*s).End}}
			}
		})
	case "+", "=", "]":
		step = 15 * simtime.Minute
	case "-", "_", "[":
		step = -15 * simtime.Minute
	case "enter":
		r := m.sess.Apply(ed.action())[0]
		m.staff.result = &r
		m.status = resultText(r)
		if r.OK {
			if p, ok := m.sess.RosterPlan(ed.emp); ok {
				ed.plan = p
			}
			ed.remove = false
		}
	case "q", "space", "1", "2", "3", "4", "5", "?":
		return false // time, quitting and help stay global
	default:
		return true // the editor holds the keyboard until esc
	}
	if step != 0 {
		ed.edit(func(s **api.Shift) {
			if *s == nil {
				return
			}
			x := *s
			from, to := clock(x.Start), clock(x.End)
			switch ed.field {
			case edStart:
				old := from
				from = max(0, min(to-15*simtime.Minute, from+step))
				x.Start = hhmm(from)
				for i := range x.Covers {
					if clock(x.Covers[i].From) == old {
						x.Covers[i].From = x.Start
					}
				}
			case edEnd:
				to = max(from+15*simtime.Minute, min(simtime.Day-simtime.Minute, to+step))
				for i := range x.Covers {
					if clock(x.Covers[i].Until) == clock(x.End) {
						x.Covers[i].Until = hhmm(to)
					}
				}
				x.End = hhmm(to)
			case edBreak:
				if len(x.Breaks) > 0 {
					b0, b1 := clock(x.Breaks[0][0])+step, clock(x.Breaks[0][1])+step
					if b0 > from && b1 < to {
						x.Breaks[0] = [2]string{hhmm(b0), hhmm(b1)}
					}
				}
			case edRoom:
				as := ed.plan.Assignments
				if i := slices.Index(as, x.Assignment); len(as) > 0 {
					d := 1
					if step < 0 {
						d = len(as) - 1
					}
					x.Assignment = as[(max(0, i)+d)%len(as)]
					x.Covers = slices.DeleteFunc(x.Covers, func(c api.Cover) bool { return c.Room == x.Assignment })
				}
			case edCover:
				if len(x.Covers) > 0 {
					c := &x.Covers[0]
					cf, cu := clock(c.From), clock(c.Until)
					// Move the cover's end; it starts with the shift.
					cu = max(cf+15*simtime.Minute, min(to, cu+step))
					c.Until = hhmm(cu)
				}
			}
			fitShift(x)
		})
	}
	return true
}

// cellText is a day of the grid: "0745–1615 R1" and its marks.
func cellText(s *api.Shift) (string, string) {
	if s == nil {
		return "off", ""
	}
	t := strings.ReplaceAll(s.Start, ":", "") + "–" + strings.ReplaceAll(s.End, ":", "")
	room := s.Assignment
	if room == "desk" {
		room = "dsk"
	}
	mark := room
	if len(s.Breaks) > 0 {
		mark += " b"
	}
	if len(s.Covers) > 0 {
		mark += " +" + s.Covers[0].Room
	}
	return t, mark
}

// netMin is a week's rostered minutes, breaks excluded.
func netMin(w api.Week) int64 {
	var n simtime.Duration
	for _, s := range w.Days() {
		if s == nil {
			continue
		}
		n += clock(s.End) - clock(s.Start)
		for _, b := range s.Breaks {
			n -= clock(b[1]) - clock(b[0])
		}
	}
	return int64(n / simtime.Minute)
}

// rosterBody renders the editor.
func (m *Model) rosterBody(w, n int) []string {
	ed := m.staff.edit
	p := ed.plan
	lines := []string{titleStyle.Render(section(fmt.Sprintf("ROSTER · %s (%s) · %s · contract %s", p.Name, p.Employee,
		strings.ReplaceAll(p.Role, "_", " "), minutes(p.ContractMin)), w))}
	mode := "weekly: both rota weeks alike"
	if !ed.weekly {
		mode = "fortnightly: weeks A and B differ"
	}
	op := "set"
	if ed.remove {
		op = "WITHDRAW the pending change"
	}
	lines = append(lines, fmt.Sprintf(" effective ‹ %s › (Mondays from %s) · %s · %s", shortDate(ed.effective()), shortDate(p.EarliestEffective), mode, op))
	var vers []string
	for _, v := range p.Versions {
		switch {
		case v.InEffect && v.Effective == "":
			vers = append(vers, fmt.Sprintf("in force: original (%s)", minutes(v.NetMinA)))
		case v.InEffect:
			vers = append(vers, fmt.Sprintf("in force: from %s (%s)", shortDate(v.Effective), minutes(v.NetMinA)))
		case v.Pending:
			vers = append(vers, fmt.Sprintf("pending %s (%s)", shortDate(v.Effective), minutes(v.NetMinA)))
		}
	}
	lines = append(lines, dimStyle.Render(" "+strings.Join(vers, " · ")))
	// The grid: rota weeks as rows, weekdays as columns.
	col := max(10, min(16, (w-4)/7))
	head := "    "
	for _, d := range weekdayNames {
		head += fmt.Sprintf("%-*s", col, d)
	}
	lines = append(lines, dimStyle.Render(head))
	rows := []int{0}
	if !ed.weekly {
		rows = append(rows, 7)
	}
	for _, base := range rows {
		label := " A  "
		if ed.weekly {
			label = " AB "
		} else if base == 7 {
			label = " B  "
		}
		top, bottom := label, "    "
		for d := range 7 {
			t, mark := cellText(*ed.at(base + d))
			cell := fmt.Sprintf("%-*s", col, t)
			if base+d == ed.cell {
				cell = titleStyle.Render(fmt.Sprintf("%-*s", col, "›"+t))
			}
			top += cell
			bottom += fmt.Sprintf("%-*s", col, mark)
		}
		lines = append(lines, top, dimStyle.Render(bottom))
	}
	// The cursor day's fields.
	if s := *ed.at(ed.cell); s != nil {
		brk, cov := "none", "none"
		if len(s.Breaks) > 0 {
			brk = s.Breaks[0][0] + "–" + s.Breaks[0][1]
		}
		if len(s.Covers) > 0 {
			cov = s.Covers[0].Room + " " + s.Covers[0].From + "–" + s.Covers[0].Until
		}
		vals := [edFields]string{s.Start, s.End, brk, s.Assignment, cov}
		var parts []string
		for f := range edFields {
			v := edFieldNames[f] + " " + vals[f]
			if f == ed.field {
				v = titleStyle.Render("‹" + v + "›")
			}
			parts = append(parts, v)
		}
		lines = append(lines, " "+weekdayNames[ed.cell%7]+": "+strings.Join(parts, " · "))
	} else {
		lines = append(lines, " "+weekdayNames[ed.cell%7]+": off (o: add a shift)")
	}
	a, b := netMin(ed.r.A), netMin(ed.r.B)
	if ed.weekly {
		b = a
	}
	lines = append(lines, fmt.Sprintf(" rostered %s (A) · %s (B) · contract %s, paid whatever is rostered", minutes(a), minutes(b), minutes(p.ContractMin)))
	lines = append(lines, dimStyle.Render(" ←→ day ↑↓ field +/- change · o on/off · b break · c cover · w weekly/fortnightly · , . effective · x withdraw · r reset · enter submit · esc back"))
	if ed.note != "" {
		lines = append(lines, dimStyle.Render(" "+ed.note))
		ed.note = ""
	}
	if r := m.staff.result; r != nil && r.Action.Type == api.ActSetRoster {
		mark := openStyle.Render(" Done ✓ ")
		if !r.OK {
			mark = alertStyle.Render(" Refused ✗ ")
		}
		lines = append(lines, wrapAfter(mark, r.Message, w, 2)...)
	}
	lines = append(lines, m.previewLines(m.sess.PreviewStaffing(ed.action()), w, n-len(lines))...)
	if len(lines) > n {
		lines = lines[:n]
	}
	return lines
}
