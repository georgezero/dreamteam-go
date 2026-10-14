package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/georgezero/dreamteam-go/internal/api"
)

const (
	minWidth  = 60
	minHeight = 16
)

var (
	barStyle   = lipgloss.NewStyle().Reverse(true)
	titleStyle = lipgloss.NewStyle().Bold(true)
	dimStyle   = lipgloss.NewStyle().Faint(true)
	warnStyle  = lipgloss.NewStyle().Foreground(lipgloss.Yellow)
	alertStyle = lipgloss.NewStyle().Foreground(lipgloss.Red).Bold(true)
	openStyle  = lipgloss.NewStyle().Foreground(lipgloss.Green).Bold(true)
)

func (m *Model) render() string {
	w, h := m.width, m.height
	if w < minWidth || h < minHeight {
		return fmt.Sprintf("Terminal too small (need %d×%d, have %d×%d).", minWidth, minHeight, w, h)
	}
	o := m.obs
	var lines []string

	// Top bar.
	run := "❚❚ PAUSED"
	if m.running {
		run = "▶ RUNNING " + speeds[m.speed].label
	} else {
		run += " · " + speeds[m.speed].label
	}
	left := " DREAMTEAM · " + o.Practice.Name
	right := fmt.Sprintf("%s %s · Day %d · %s ── %s ", o.Weekday, shortDate(o.Date), o.Day, o.Clock, run)
	if ansi.StringWidth(left)+ansi.StringWidth(right) >= w {
		left = " DREAMTEAM" // the clock and run state matter more than the name
	}
	lines = append(lines, barStyle.Render(spread(left, right, w)))

	// Practice line.
	state := dimStyle.Render("○ CLOSED")
	if o.Practice.Open {
		state = openStyle.Render("● OPEN")
	}
	info := " " + state
	if o.Practice.Hours != "" {
		info += " " + o.Practice.Hours
	} else {
		info += " today"
	}
	info += "   " + deskText(o) + "   Scenario: " + o.Practice.Scenario
	lines = append(lines, fit(info, w))

	// Rooms and staff: side by side when wide, stacked otherwise; the staff
	// list collapses to one summary line when the terminal is short.
	rooms := roomLines(o)
	staff := staffLines(o)
	if w >= 100 {
		lw := 44
		lines = append(lines, titleStyle.Render(fit(section("ROOMS", lw)+" "+section("STAFF", w-lw-1), w)))
		for i := 0; i < max(len(rooms), len(staff)); i++ {
			var l, r string
			if i < len(rooms) {
				l = rooms[i]
			}
			if i < len(staff) {
				r = staff[i]
			}
			lines = append(lines, fit(fit(l, lw)+" "+r, w))
		}
	} else {
		lines = append(lines, titleStyle.Render(section("ROOMS", w)))
		for _, l := range rooms {
			lines = append(lines, fit(l, w))
		}
		lines = append(lines, titleStyle.Render(section("STAFF", w)))
		if h-len(lines)-len(staff)-4 >= 4 {
			for _, l := range staff {
				lines = append(lines, fit(l, w))
			}
		} else {
			lines = append(lines, fit(staffSummary(o), w))
		}
	}

	// Event log, newest first, filling the space between header and footer.
	lines = append(lines, titleStyle.Render(section("EVENT LOG", w)))
	room := h - len(lines) - 2
	n := 0
	for i := len(o.RecentLog) - 1; i >= 0 && n < room; i-- {
		lines = append(lines, fit(" "+logLine(o.RecentLog[i]), w))
		n++
	}
	if n == 0 && room > 0 {
		lines = append(lines, dimStyle.Render(fit(" (nothing yet)", w)))
		n++
	}
	for ; n < room; n++ {
		lines = append(lines, "")
	}

	// Status and key hints.
	lines = append(lines, fit(" "+m.status, w))
	auto := "a auto-pause:off"
	if m.autoPause {
		auto = "a auto-pause:on"
	}
	hints := " space run/pause  1-5 speed  n next event  " + auto + "  q quit"
	lines = append(lines, barStyle.Render(fit(hints, w)))
	return strings.Join(lines, "\n")
}

// section renders a section rule: "─ ROOMS ─────".
func section(title string, w int) string {
	return fit("─ "+title+" "+strings.Repeat("─", max(0, w-len(title)-3)), w)
}

func deskText(o api.Observation) string {
	switch o.Desk.Coverage {
	case api.DeskStaffed:
		return "Desk " + openStyle.Render("●") + " " + staffName(o, o.Desk.By)
	case api.DeskOps:
		return "Desk " + warnStyle.Render("▒") + " " + staffName(o, o.Desk.By) + " (ops)"
	case api.DeskSelf:
		return "Desk " + warnStyle.Render("─ techs self-register")
	}
	return dimStyle.Render("Desk closed")
}

func staffName(o api.Observation, id string) string {
	for _, e := range o.Staff {
		if e.ID == id {
			return e.Name
		}
	}
	return id
}

func roomLines(o api.Observation) []string {
	var out []string
	for _, r := range o.Rooms {
		var st string
		switch r.State {
		case api.RoomIdle:
			st = openStyle.Render("○") + " Idle     "
		case api.RoomUnstaffed:
			st = warnStyle.Render("─ Unstaffed")
		default:
			st = dimStyle.Render("· Closed   ")
		}
		var who []string
		for _, id := range r.StaffBy {
			who = append(who, staffName(o, id))
		}
		line := fmt.Sprintf(" %-3s%s  %s", r.Modality, st, strings.Join(who, ", "))
		if r.Shared {
			line += dimStyle.Render(" (shared)")
		}
		out = append(out, line)
	}
	return out
}

func roleLabel(e api.Employee) string {
	switch e.Role {
	case "front_desk":
		return "Desk"
	case "ops_manager":
		return "Ops"
	case "radiologist":
		return "Rad"
	}
	return strings.Join(e.Skills, " ")
}

func staffLines(o api.Observation) []string {
	var out []string
	for _, e := range o.Staff {
		var st string
		switch e.Status {
		case api.StaffOnDuty:
			st = openStyle.Render("●") + " on duty until " + e.EndsAt
			if len(e.Rooms) > 0 {
				var mods []string
				for _, id := range e.Rooms {
					for _, r := range o.Rooms {
						if r.ID == id {
							mods = append(mods, r.Modality)
						}
					}
				}
				st += dimStyle.Render(" · " + strings.Join(mods, "+"))
			}
		case api.StaffOnBreak:
			st = warnStyle.Render("◐ break until " + e.EndsAt)
		case api.StaffReading:
			st = openStyle.Render("●") + " reading"
		case api.StaffOtherDuties:
			st = "○ other duties"
		default:
			switch {
			case e.Shift == "":
				st = dimStyle.Render("· off today")
			case e.StartsAt != "":
				st = dimStyle.Render("○ starts " + e.StartsAt)
			default:
				st = dimStyle.Render("· off (" + e.Shift + ")")
			}
		}
		if e.Role == "radiologist" && e.Shift != "" {
			note := fmt.Sprintf(" · %d%% site", e.SiteBP/100)
			if e.Remote {
				note += ", remote"
			}
			if e.EndsAt != "" {
				note += " · until " + e.EndsAt
			}
			st += dimStyle.Render(note)
		}
		out = append(out, fmt.Sprintf(" %-10s %-6s %s", e.Name, roleLabel(e), st))
	}
	return out
}

// staffSummary is the one-line staff panel for short terminals.
func staffSummary(o api.Observation) string {
	var techs, techsOn, rads, radsOn int
	for _, e := range o.Staff {
		on := e.Status == api.StaffOnDuty || e.Status == api.StaffReading || e.Status == api.StaffOtherDuties
		switch e.Role {
		case "technologist":
			techs++
			if on {
				techsOn++
			}
		case "radiologist":
			rads++
			if on {
				radsOn++
			}
		}
	}
	return fmt.Sprintf(" Techs on duty %d/%d   Rads %d/%d", techsOn, techs, radsOn, rads)
}

func logLine(e api.LogEntry) string {
	clock := e.Time
	if len(clock) >= 16 {
		clock = clock[11:16]
	}
	text := clock + "  " + e.Text
	switch e.Severity {
	case api.Warn:
		return warnStyle.Render("⚠ " + text)
	case api.Alert:
		return alertStyle.Render("‼ " + text)
	}
	return text
}

// shortDate turns "2026-10-13" into "Oct 13".
func shortDate(iso string) string {
	months := []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
	var y, mo, d int
	if _, err := fmt.Sscanf(iso, "%d-%d-%d", &y, &mo, &d); err != nil || mo < 1 || mo > 12 {
		return iso
	}
	return fmt.Sprintf("%s %d", months[mo-1], d)
}

// fit truncates s to width w (ANSI-aware) and pads it to exactly w cells.
func fit(s string, w int) string {
	s = ansi.Truncate(s, w, "…")
	if pad := w - ansi.StringWidth(s); pad > 0 {
		s += strings.Repeat(" ", pad)
	}
	return s
}

// spread places left and right on one line of width w.
func spread(left, right string, w int) string {
	gap := w - ansi.StringWidth(left) - ansi.StringWidth(right)
	if gap < 1 {
		return fit(left+" "+right, w)
	}
	return left + strings.Repeat(" ", gap) + right
}
