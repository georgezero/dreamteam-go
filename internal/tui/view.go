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
	info := " " + state + "   Scenario: " + o.Practice.Scenario
	if o.Practice.Scenario == "demo" {
		info += dimStyle.Render(" (synthetic visits until the real practice is built)")
	}
	lines = append(lines, fit(info, w), "")

	// Event log, newest first, filling the space between header and footer.
	lines = append(lines, titleStyle.Render(fit(" EVENT LOG "+strings.Repeat("─", max(0, w-11)), w)))
	room := h - len(lines) - 2
	n := 0
	for i := len(o.RecentLog) - 1; i >= 0 && n < room; i-- {
		lines = append(lines, fit(" "+logLine(o.RecentLog[i]), w))
		n++
	}
	if n == 0 {
		lines = append(lines, dimStyle.Render(" (nothing yet)"))
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
