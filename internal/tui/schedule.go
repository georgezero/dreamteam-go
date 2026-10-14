package tui

import (
	"fmt"
	"strings"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// cell is the timeline resolution: one character per 15 minutes.
const cell = 15 * simtime.Minute

// scheduleBody renders the schedule screen in n lines: one timeline per
// room for the selected day, then the selected room's slot list.
func (m *Model) scheduleBody(w, n int) []string {
	d := m.sched
	o := m.obs
	when := fmt.Sprintf("%s %s", d.Weekday, shortDate(d.Date))
	switch {
	case d.Offset == 0:
		when += " (today)"
	case d.Offset == 1:
		when += " (tomorrow)"
	case d.Offset > 1:
		when += fmt.Sprintf(" (in %d days)", d.Offset)
	default:
		when += fmt.Sprintf(" (%d days ago)", -d.Offset)
	}
	lines := []string{titleStyle.Render(section("SCHEDULE · "+when, w))}
	if d.Hours == nil {
		return append(lines, "", dimStyle.Render(" Closed all day."))
	}
	m.schedRoom = min(m.schedRoom, len(d.Rooms)-1)

	// Hour ruler.
	ruler := []rune(strings.Repeat(" ", int((d.Hours.To-d.Hours.From)/cell)+2))
	for t := d.Hours.From; t <= d.Hours.To; t += simtime.Hour {
		if t%simtime.Hour != 0 {
			continue
		}
		i := int((t - d.Hours.From) / cell)
		for k, c := range simtime.FormatClock(t)[:2] {
			if i+k < len(ruler) {
				ruler[i+k] = c
			}
		}
	}
	lines = append(lines, "       "+string(ruler))
	for r, rs := range d.Rooms {
		mark := "  "
		if r == m.schedRoom {
			mark = "▶ "
		}
		booked, open := 0, 0
		for _, s := range rs.Slots {
			switch s.Status {
			case api.SlotBooked:
				booked++
			case api.SlotOpen:
				open++
			}
		}
		lines = append(lines, fmt.Sprintf(" %-3s%s %s  %s", rs.Modality, mark, timeline(d, rs),
			dimStyle.Render(fmt.Sprintf("%d booked · %d open", booked, open))))
	}
	if d.Offset == 0 {
		now := simtime.ClockOf(o.Now)
		if now >= d.Hours.From && now < d.Hours.To {
			lines = append(lines, "       "+strings.Repeat(" ", int((now-d.Hours.From)/cell))+"^ now")
		} else {
			lines = append(lines, "")
		}
	}
	lines = append(lines, dimStyle.Render(" ░ booked  · open  ˙ too late to book  ─ unstaffed  ◐ tech break  ⊘ contrast, no radiologist on site"))

	// Selected room: demand against capacity, then its slots.
	rs := d.Rooms[m.schedRoom]
	head := fmt.Sprintf("%s · %s", rs.Modality, rs.Name)
	for _, md := range o.Demand.Modalities {
		if md.Modality != rs.Modality {
			continue
		}
		head += fmt.Sprintf(" · %s demand today: %d orders, %d booked, %d lost", md.Modality, md.Today.Received, md.Today.Booked, md.Today.Lost)
		switch {
		case md.NextOpenDays < 0:
			head += " · no open slot within horizon"
		case md.NextOpenDays > 1:
			head += fmt.Sprintf(" · next open slot in %d days", md.NextOpenDays)
		}
	}
	lines = append(lines, titleStyle.Render(section(head, w)))
	lines = append(lines, dimStyle.Render("  TIME   APPT    PATIENT  EXAM                     PRI      REFERRED BY"))
	rows := slotRows(rs)
	if len(rows) == 0 {
		rows = []string{dimStyle.Render("  (no template slots this day)")}
	}
	room := n - len(lines)
	m.schedScroll = min(m.schedScroll, max(0, len(rows)-room))
	for i := m.schedScroll; i < len(rows) && len(lines) < n; i++ {
		lines = append(lines, rows[i])
	}
	return lines
}

// timeline draws one room's day, one character per 15 minutes.
func timeline(d api.DaySchedule, rs api.RoomSchedule) string {
	var b strings.Builder
	for t := d.Hours.From; t < d.Hours.To; t += cell {
		staffed := inSpans(rs.Staffed, t)
		brk := inSpans(rs.Breaks, t)
		var s *api.Slot
		for i := range rs.Slots {
			if rs.Slots[i].From <= t && t < rs.Slots[i].To {
				s = &rs.Slots[i]
			}
		}
		switch {
		case s != nil && s.Status == api.SlotBooked:
			b.WriteString(openStyle.Render("░"))
		case brk:
			b.WriteString(warnStyle.Render("◐"))
		case !staffed:
			b.WriteString(dimStyle.Render("─"))
		case s == nil:
			b.WriteString(" ")
		case s.Status == api.SlotOpen:
			b.WriteString("·")
		case s.Status == api.SlotPassed:
			b.WriteString(dimStyle.Render("˙"))
		case s.Status == api.SlotNoSupervision:
			b.WriteString(warnStyle.Render("⊘"))
		default:
			b.WriteString(dimStyle.Render("─"))
		}
	}
	return b.String()
}

func inSpans(xs []api.Span, t simtime.Duration) bool {
	for _, x := range xs {
		if x.From <= t && t < x.To {
			return true
		}
	}
	return false
}

// slotRows lists a room's slots: appointments, open slots and slots that
// cannot be booked, with consecutive unbookable slots collapsed.
func slotRows(rs api.RoomSchedule) []string {
	var out []string
	for i := 0; i < len(rs.Slots); i++ {
		s := rs.Slots[i]
		at := simtime.FormatClock(s.From)
		switch s.Status {
		case api.SlotBooked:
			a := s.Appt
			pri := a.Priority
			if pri == "urgent" {
				pri = alertStyle.Render("‼ urgent")
			}
			out = append(out, fmt.Sprintf("  %s  %s  %s   %-24s %-8s %s", at, a.ID, a.Patient, trim(a.ExamName, 24), pri, a.Source))
		case api.SlotOpen:
			out = append(out, fmt.Sprintf("  %s  %s", at, openStyle.Render("open")+dimStyle.Render(" · "+s.Category)))
		case api.SlotPassed:
			out = append(out, dimStyle.Render(fmt.Sprintf("  %s  open, too late to book · %s", at, s.Category)))
		default:
			j := i
			for j+1 < len(rs.Slots) && rs.Slots[j+1].Status == s.Status {
				j++
			}
			why := "no technologist rostered"
			if s.Status == api.SlotNoSupervision {
				why = "contrast slot, no radiologist on site"
			}
			n := j - i + 1
			span := at
			if n > 1 {
				span = at + "–" + simtime.FormatClock(rs.Slots[j].To)
			}
			out = append(out, warnStyle.Render(fmt.Sprintf("  %s  not bookable: %s (%d slot%s)", span, why, n, plural(n))))
			i = j
		}
	}
	return out
}

func trim(s string, n int) string {
	if len([]rune(s)) <= n {
		return s
	}
	return string([]rune(s)[:n-1]) + "…"
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
