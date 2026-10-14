package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/version"
)

const (
	minWidth  = 60
	minHeight = 16
)

var (
	barStyle   = lipgloss.NewStyle().Reverse(true)
	plainStyle = lipgloss.NewStyle()
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
	var body []string
	hints := ""
	switch m.screen {
	case screenSchedule:
		body = m.scheduleBody(w, h-3)
		hints = " ↑↓ room  ←→ day  j/k scroll  t today  d dash  p patients  l log  q quit"
	case screenLog:
		body = m.logBody(w, h-3)
		hints = " ↑↓ scroll  pgup/pgdn page  g/G oldest/newest  d dash  p patients  q quit"
	case screenPatients:
		body = m.patientsBody(w, h-3)
		hints = " ↑↓ select  enter open  tab filter  ←→ day  t today  esc back  q quit"
		if m.journeyID != "" {
			hints = " esc back to list  d dashboard  space run  q quit"
		}
	default:
		body = m.dashboardBody(w, h-3)
		auto := "off"
		if m.autoPause {
			auto = "on"
		}
		hints = " space run  1-5 speed  n next  p patients  s sched  l log  a auto:" + auto + "  q quit"
	}
	lines := []string{m.topBar(w)}
	for i := 0; i < h-3; i++ {
		l := ""
		if i < len(body) {
			l = body[i]
		}
		lines = append(lines, fit(l, w))
	}
	lines = append(lines, fit(" "+m.status, w), barStyle.Render(fit(hints, w)))
	return strings.Join(lines, "\n")
}

func (m *Model) topBar(w int) string {
	o := m.obs
	run := "❚❚ PAUSED"
	if m.running {
		run = "▶ RUNNING " + speeds[m.speed].label
	} else {
		run += " · " + speeds[m.speed].label
	}
	brand := " DREAMTEAM v" + version.Number
	left := brand + " · " + o.Practice.Name
	right := fmt.Sprintf("%s %s · Day %d · %s ── %s ", o.Weekday, shortDate(o.Date), o.Day, o.Clock, run)
	if ansi.StringWidth(left)+ansi.StringWidth(right) >= w {
		left = brand // the clock and run state matter more than the name
	}
	return barStyle.Render(spread(left, right, w))
}

// dashboardBody lays out the dashboard in n lines. Sections are added in
// priority order; the event log takes what is left, up to dashboardLog.
func (m *Model) dashboardBody(w, n int) []string {
	o := m.obs
	info := " " + practiceStatus(o)
	if o.Practice.Phase != api.PhaseClosing {
		info += "   " + deskText(o)
	}
	if w < 100 {
		info += "   " + staffSummary(o)
	}
	info += "   Scenario: " + o.Practice.Scenario
	lines := []string{info}

	rooms := roomLines(o)
	staff := staffLines(o)
	if w >= 100 {
		lw := 44
		lines = append(lines, titleStyle.Render(section("ROOMS", lw)+" "+section("STAFF", w-lw-1)))
		for i := 0; i < max(len(rooms), len(staff)); i++ {
			var l, r string
			if i < len(rooms) {
				l = rooms[i]
			}
			if i < len(staff) {
				r = staff[i]
			}
			lines = append(lines, fit(l, lw)+" "+r)
		}
	} else {
		lines = append(lines, titleStyle.Render(section("ROOMS", w)))
		lines = append(lines, rooms...)
		if n >= 34 {
			lines = append(lines, titleStyle.Render(section("STAFF", w)))
			lines = append(lines, staff...)
		}
	}
	lines = append(lines, flowLines(o, w)...)
	demand := demandLines(o, w)
	if n < 28 {
		demand = demand[:len(demand)-1] // drop the since-day-1 line on short screens
	}
	lines = append(lines, demand...)

	// Event log: newest first, compact; the full log is on its own screen.
	room := min(dashboardLog, n-len(lines)-1)
	if room > 0 {
		lines = append(lines, titleStyle.Render(section("EVENT LOG (l: full log)", w)))
		k := 0
		for i := len(o.RecentLog) - 1; i >= 0 && k < room; i-- {
			lines = append(lines, " "+logLine(o.RecentLog[i]))
			k++
		}
		if k == 0 {
			lines = append(lines, dimStyle.Render(" (nothing yet)"))
		}
	}
	if len(lines) > n {
		lines = lines[:n]
	}
	return lines
}

// demandLines is the demand panel: today's orders and what became of them,
// the day's schedule and the earliest open slot per modality.
func demandLines(o api.Observation, w int) []string {
	out := []string{titleStyle.Render(section("DEMAND today", w)),
		dimStyle.Render("      orders booked recall  lost  cxl │ appts today  open │ next open slot")}
	for _, d := range o.Demand.Modalities {
		t := d.Today
		next := warnStyle.Render("none within horizon")
		switch {
		case d.NextOpenDays == 0:
			next = "today"
		case d.NextOpenDays == 1:
			next = "tomorrow"
		case d.NextOpenDays > 1:
			next = fmt.Sprintf("%s (%d days)", shortDate(d.NextOpen), d.NextOpenDays)
			if d.NextOpenDays > 14 {
				next = warnStyle.Render(next)
			}
		}
		lost := fmt.Sprintf("%5d", t.Lost)
		if t.Lost > 0 {
			lost = warnStyle.Render(lost)
		}
		out = append(out, fmt.Sprintf(" %-3s %8d %6d %6d %s %4d │ %11d %5d │ %s",
			d.Modality, t.Received, t.Booked, t.Recall, lost, t.Cancelled, d.ApptsToday, d.OpenToday, next))
	}
	var recv, lost int
	for _, d := range o.Demand.Modalities {
		recv += d.SinceStart.Received
		lost += d.SinceStart.Lost
	}
	out = append(out, dimStyle.Render(fmt.Sprintf(" Since day 1: %d orders, %d lost · recall list %d", recv, lost, o.Demand.RecallList)))
	return out
}

// logBody is the full event log, newest at the bottom, scrollable.
func (m *Model) logBody(w, n int) []string {
	o := m.obs
	lines := []string{titleStyle.Render(section(fmt.Sprintf("EVENT LOG · %d entries (last %d kept)", o.LogTotal, len(o.RecentLog)), w))}
	rows := n - 1
	m.logScroll = min(m.logScroll, max(0, len(o.RecentLog)-rows))
	end := len(o.RecentLog) - m.logScroll
	start := max(0, end-rows)
	for i := start; i < end; i++ {
		e := o.RecentLog[i]
		day := ""
		if len(e.Time) >= 10 {
			day = shortDate(e.Time[:10]) + " "
		}
		lines = append(lines, " "+dimStyle.Render(day)+logLine(e))
	}
	return lines
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

// practiceStatus is the day's phase (TUI_DESIGN §4.1, SIMULATION_DESIGN
// §6.9): open with hours, closing with patients remaining and overtime, or
// closed with when operations actually ended.
func practiceStatus(o api.Observation) string {
	p := o.Practice
	ot := ""
	if p.OvertimeMin > 0 {
		ot = " · OT +" + minutes(p.OvertimeMin)
		if p.OvertimeCostCents > 0 {
			ot += fmt.Sprintf(" ($%d)", (p.OvertimeCostCents+50)/100)
		}
	}
	switch p.Phase {
	case api.PhaseOpen:
		return openStyle.Render("● OPEN") + " " + p.Hours
	case api.PhaseClosing:
		s := warnStyle.Render(fmt.Sprintf("◐ CLOSING — %d patient%s remaining", p.OnSite, plural(p.OnSite))) +
			ot + " after " + p.ScheduledClose
		if len(p.HeldOver) > 0 {
			s += dimStyle.Render(" · held over: " + strings.Join(p.HeldOver, ", "))
		}
		return s
	}
	switch {
	case p.Hours == "":
		return dimStyle.Render("○ CLOSED today")
	case p.OperationalClose != "" && p.OperationalClose != p.ScheduledClose:
		return dimStyle.Render("○ CLOSED") + fmt.Sprintf(" · operations closed %s (scheduled %s)", p.OperationalClose, p.ScheduledClose) + ot
	case p.OperationalClose != "":
		return dimStyle.Render("○ CLOSED") + " · closed " + p.OperationalClose + ot
	}
	return dimStyle.Render("○ CLOSED") + " · opens " + strings.SplitN(p.Hours, "–", 2)[0]
}

func roomLines(o api.Observation) []string {
	var out []string
	for _, r := range o.Rooms {
		var st string
		switch r.State {
		case api.RoomOccupied:
			st = alertStyle.Render("●") + " Scanning "
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
		staff := strings.Join(who, ", ")
		if r.Shared {
			staff += " (sh)"
		}
		line := fmt.Sprintf(" %-3s%s  %-14s", r.Modality, st, staff)
		if c := r.Current; c != nil {
			el := fmt.Sprintf("%d/%dm", c.ElapsedMin, c.ExpectedMin)
			if c.ElapsedMin > c.ExpectedMin {
				el = warnStyle.Render(el)
			}
			line = fmt.Sprintf(" %-3s%s  %s %s %s %s", r.Modality, st, c.Patient, trim(c.ExamName, 14),
				el, dimStyle.Render(c.Technologist))
		}
		if n := queuedFor(o, r.Modality); n > 0 {
			line += warnStyle.Render(fmt.Sprintf(" · %d waiting", n))
		}
		out = append(out, line)
	}
	return out
}

// queuedFor counts patients on site for a modality who are not in an exam.
func queuedFor(o api.Observation, modality string) int {
	n := 0
	for _, w := range o.Flow.Waiting {
		if w.Modality == modality && w.State != api.VisitInExam {
			n++
		}
	}
	return n
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
		case api.StaffBusy:
			st = alertStyle.Render("●") + " " + e.Task
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
		if e.OvertimeMin > 0 && e.Status != api.StaffOff {
			st += warnStyle.Render(" · OT +" + minutes(e.OvertimeMin))
		}
		out = append(out, fmt.Sprintf(" %-10s %-6s %s", e.Name, roleLabel(e), st))
	}
	return out
}

// flowLines is the patient-flow panel: who is on site now and in which
// stage, and what happened to today's patients so far.
func flowLines(o api.Observation, w int) []string {
	f := o.Flow
	t := f.Today
	var toReg, waiting, inExam int
	longest := int64(0)
	for _, p := range f.Waiting {
		switch p.State {
		case api.VisitArrived, api.VisitRegistering:
			toReg++
		case api.VisitWaiting:
			waiting++
		case api.VisitInExam:
			inExam++
		}
		if p.State != api.VisitInExam {
			longest = max(longest, p.WaitedMin)
		}
	}
	onSite := fmt.Sprintf(" On site %d", len(f.Waiting))
	if waiting+toReg > 0 {
		onSite = warnStyle.Render(onSite)
	}
	var by []string
	for _, d := range o.Demand.Modalities {
		by = append(by, fmt.Sprintf("%s %d", d.Modality, queuedFor(o, d.Modality)))
	}
	l1 := onSite + fmt.Sprintf(" · registering %d · waiting %d (%s) · in exam %d · longest wait %s",
		toReg, waiting, strings.Join(by, " "), inExam, minutes(longest))
	missed := fmt.Sprintf("missed %d", f.MissedNow)
	if f.MissedNow > 0 {
		missed = warnStyle.Render(missed)
	}
	left := fmt.Sprintf("left %d", t.Left)
	if t.Left > 0 {
		left = warnStyle.Render(left)
	}
	l2 := fmt.Sprintf(" Done %d · arrived %d (late %d) · walk-ins %d · %s · %s · no-shows %d",
		t.ExamsDone, t.Arrived, t.Late, t.WalkIns, left, missed, f.NoShowsLastClose)
	if t.NoCoverage > 0 {
		l2 += warnStyle.Render(fmt.Sprintf(" · rescheduled %d", t.NoCoverage))
	}
	if t.AfterHours > 0 && w >= 100 {
		l2 += fmt.Sprintf(" · after hours %d", t.AfterHours)
	}
	return []string{titleStyle.Render(section("PATIENT FLOW today", w)), l1, l2}
}

// minutes formats a minute count: "34m", "2h 05m".
func minutes(m int64) string {
	if m < 60 {
		return fmt.Sprintf("%dm", m)
	}
	return fmt.Sprintf("%dh %02dm", m/60, m%60)
}

// staffSummary is the one-line staff panel for short terminals.
func staffSummary(o api.Observation) string {
	var techs, techsOn, rads, radsOn int
	for _, e := range o.Staff {
		on := e.Status == api.StaffOnDuty || e.Status == api.StaffBusy || e.Status == api.StaffReading || e.Status == api.StaffOtherDuties
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
	return fmt.Sprintf("Techs %d/%d  Rads %d/%d", techsOn, techs, radsOn, rads)
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
