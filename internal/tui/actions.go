package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// The actions screen (TUI_DESIGN §4.9): the three V0.1 actions. Every
// submission goes through the session's Apply, the same entry point an
// agent uses; the screen never changes the simulation itself. The row under
// the cursor is previewed with Validate (no effect) as it is edited.

const (
	extendStep    = 15 // minutes per ←/→ on an employee row
	extendDefault = 60
)

var walkInChoices = []string{api.WalkInAcceptAlways, api.WalkInAcceptIfGap, api.WalkInAcceptUntil, api.WalkInScheduleLater}

var worklistChoices = []string{api.WorklistUrgentThenFIFO, api.WorklistFIFO, api.WorklistOldestFirst, api.WorklistShortestFirst}

var policyLabels = map[string]string{
	api.WalkInAcceptAlways: "accept always", api.WalkInAcceptIfGap: "accept if gap",
	api.WalkInAcceptUntil: "accept until", api.WalkInScheduleLater: "decline (offer a later slot)",
	api.WorklistUrgentThenFIFO: "urgent & overdue, then FIFO", api.WorklistFIFO: "FIFO",
	api.WorklistOldestFirst: "oldest first", api.WorklistShortestFirst: "shortest first (expected read time)",
}

// maxActionDay is how many days ahead the screen offers extensions.
const maxActionDay = 13

// actionStaff lists the employees rostered on the selected day, in staff
// order, with any extensions already approved.
func (m *Model) actionStaff() []api.RosterEntry { return m.roster }

// statusOf is an employee's status today ("" on other days).
func (m *Model) statusOf(id string) string {
	if m.actDay != 0 {
		return ""
	}
	for _, e := range m.obs.Staff {
		if e.ID == id {
			return statusLabel(e)
		}
	}
	return ""
}

// actDate is the selected day's date ("" for today).
func (m *Model) actDate() string {
	if m.actDay == 0 || len(m.obs.Date) < 10 {
		return ""
	}
	d, err := simtime.ParseDate(m.obs.Date)
	if err != nil {
		return ""
	}
	return d.AddDays(m.actDay).String()
}

// walkInCutoff is the accept_until cutoff offered: an hour before today's
// scheduled close (18:00 on a closed day).
func (m *Model) walkInCutoff() string {
	c, err := simtime.ParseClock(m.obs.Practice.ScheduledClose)
	if err != nil {
		return "18:00"
	}
	return simtime.FormatClock(max(simtime.Hour, c-simtime.Hour))
}

// selectedAction is the action the cursor row would submit.
func (m *Model) selectedAction() (api.Action, bool) {
	staff := m.actionStaff()
	switch k := m.actSel - len(staff); {
	case m.actSel < len(staff):
		e := staff[m.actSel]
		return api.Action{Type: api.ActExtendShift, Employee: e.ID, Minutes: m.extendMinutes(e.ID), Date: m.actDate()}, true
	case k == 0:
		a := api.Action{Type: api.ActSetWalkInPolicy, Policy: walkInChoices[m.actWalk]}
		if a.Policy == api.WalkInAcceptUntil {
			a.Cutoff = m.walkInCutoff()
		}
		return a, true
	case k == 1:
		return api.Action{Type: api.ActSetWorklistPolicy, Policy: worklistChoices[m.actWork]}, true
	}
	return api.Action{}, false
}

func (m *Model) extendMinutes(id string) int64 {
	if v, ok := m.actMinutes[id]; ok {
		return v
	}
	return extendDefault
}

// actionKey handles the actions screen's keys; it reports whether k was one.
func (m *Model) actionKey(k string) bool {
	staff := m.actionStaff()
	rows := len(staff) + 2
	switch k {
	case "up", "k":
		m.actSel = max(0, m.actSel-1)
	case "down", "j":
		m.actSel = min(rows-1, m.actSel+1)
	case "left", "right":
		step := 1
		if k == "left" {
			step = -1
		}
		switch i := m.actSel - len(staff); {
		case m.actSel < len(staff):
			id := staff[m.actSel].ID
			m.actMinutes[id] = min(240, max(extendStep, m.extendMinutes(id)+int64(step*extendStep)))
		case i == 0:
			m.actWalk = (m.actWalk + step + len(walkInChoices)) % len(walkInChoices)
		case i == 1:
			m.actWork = (m.actWork + step + len(worklistChoices)) % len(worklistChoices)
		}
	case "[", "]":
		if k == "[" {
			m.actDay = max(0, m.actDay-1)
		} else {
			m.actDay = min(maxActionDay, m.actDay+1)
		}
		m.actSel = 0
	case "enter":
		if a, ok := m.selectedAction(); ok {
			r := m.sess.Apply(a)[0]
			m.actResult = &r
			m.status = resultText(r)
			if r.OK && a.Type == api.ActExtendShift {
				delete(m.actMinutes, a.Employee) // the next preview is a fresh +60m
			}
		}
	default:
		return false
	}
	return true
}

func resultText(r api.ActionResult) string {
	if r.OK {
		return "✓ " + r.Message
	}
	return "✗ " + r.Message
}

// syncPolicyChoices points the policy rows at the policies in force when
// the screen opens.
func (m *Model) syncPolicyChoices() {
	for i, p := range walkInChoices {
		if p == m.obs.Policies.WalkIn {
			m.actWalk = i
		}
	}
	for i, p := range worklistChoices {
		if p == m.obs.Policies.Worklist {
			m.actWork = i
		}
	}
}

// actionsBody renders the actions screen.
func (m *Model) actionsBody(w, n int) []string {
	o := m.obs
	staff := m.actionStaff()
	lines := []string{titleStyle.Render(section(fmt.Sprintf("ACTIONS · %s %s %s", o.Weekday, shortDate(o.Date), o.Clock), w))}
	cursor := func(i int) string {
		if i == m.actSel {
			return titleStyle.Render(">")
		}
		return " "
	}
	day := "today"
	if d := m.actDate(); d != "" {
		day = shortDate(d)
		if dd, err := simtime.ParseDate(d); err == nil {
			day = dd.Weekday().String() + " " + day
		}
	}
	lines = append(lines, titleStyle.Render(" Extend a shift · "+day)+dimStyle.Render(" ([ ] day) — approved overtime: they stay on regular duty, adding capacity; ahead, the extra slots open for booking"))
	lines = append(lines, dimStyle.Render("   employee    role         shift        12h cap now       add   approved"))
	for i, e := range staff {
		ext := ""
		if e.ApprovedMin > 0 {
			ext = openStyle.Render(fmt.Sprintf("  +%dm (rostered to %s)", e.ApprovedMin, e.RosterEnd))
		}
		lines = append(lines, fmt.Sprintf(" %s %-11s %-12s %-11s  %-5s   %-9s +%3dm%s", cursor(i), trim(e.Name, 11),
			trim(strings.ReplaceAll(e.Role, "_", " "), 12), e.Shift, e.CapAt, trim(m.statusOf(e.ID), 9), m.extendMinutes(e.ID), ext))
	}
	if len(staff) == 0 {
		lines = append(lines, dimStyle.Render("   (nobody rostered)"))
	}
	k := len(staff)
	cutoff := ""
	if walkInChoices[m.actWalk] == api.WalkInAcceptUntil {
		cutoff = " " + m.walkInCutoff()
	}
	inForce := policyLabels[o.Policies.WalkIn]
	if o.Policies.WalkInCutoff != "" {
		inForce += " " + o.Policies.WalkInCutoff
	}
	lines = append(lines, "", fmt.Sprintf(" %s %s ‹ %s%s ›  %s", cursor(k), titleStyle.Render("Walk-in policy "),
		policyLabels[walkInChoices[m.actWalk]], cutoff, dimStyle.Render("in force: "+inForce)))
	lines = append(lines, fmt.Sprintf(" %s %s ‹ %s ›  %s", cursor(k+1), titleStyle.Render("Worklist policy"),
		policyLabels[worklistChoices[m.actWork]], dimStyle.Render("in force: "+policyLabels[o.Policies.Worklist])))
	lines = append(lines, "")
	if r := m.actResult; r != nil {
		mark := openStyle.Render(" Done ✓ ")
		if !r.OK {
			mark = alertStyle.Render(" Refused ✗ ")
		}
		lines = append(lines, wrapAfter(mark, r.Message, w, 3)...)
	}
	if a, ok := m.selectedAction(); ok {
		p := m.sess.Validate(a)
		mark := dimStyle.Render(" If submitted: ") + openStyle.Render("✓ ")
		if !p.OK {
			mark = dimStyle.Render(" If submitted: ") + alertStyle.Render("✗ ")
		}
		lines = append(lines, wrapAfter(mark, p.Message, w, 3)...)
	}
	al := o.Actions
	lines = append(lines, dimStyle.Render(fmt.Sprintf(" Actions so far: %d attempted · %d accepted · %d rejected", al.Attempted, al.Accepted, al.Rejected)))
	for _, r := range al.Recent {
		if len(lines) >= n {
			break
		}
		mark := "✓"
		if !r.OK {
			mark = "✗"
		}
		lines = append(lines, dimStyle.Render(fmt.Sprintf("   %s %s %s %s", r.ID, clockOf(r.Time), mark, trim(r.Message, w-24))))
	}
	if len(lines) > n {
		lines = lines[:n]
	}
	return lines
}

// statusLabel is a short status for the actions list.
// (Today only; other days show the roster.)
func statusLabel(e api.Employee) string {
	switch e.Status {
	case api.StaffOnDuty:
		return "on duty"
	case api.StaffBusy:
		return "busy"
	case api.StaffOnBreak:
		return "on break"
	case api.StaffReading:
		return "reading"
	case api.StaffReadingFree:
		return "can read"
	case api.StaffOtherDuties:
		return "other"
	}
	if e.StartsAt != "" {
		return "starts " + e.StartsAt
	}
	return "gone home"
}

// selected is the cursor row's action (zero when none).
func (m *Model) selected() api.Action {
	a, _ := m.selectedAction()
	return a
}

// wrapAfter lays out text after a prefix, wrapped at word boundaries to
// width w in at most n lines (continuation lines indented).
func wrapAfter(prefix, text string, w, n int) []string {
	first := w - ansi.StringWidth(prefix) - 1
	var out []string
	line := prefix
	room := first
	for _, word := range strings.Fields(text) {
		cur := ansi.StringWidth(line) - (w - 1 - room)
		if cur > 0 && cur+1+ansi.StringWidth(word) > room {
			out = append(out, line)
			if len(out) == n {
				return out
			}
			line, room = "   ", w-4
		}
		if line == "   " || line == prefix {
			line += word
		} else {
			line += " " + word
		}
	}
	return append(out, line)
}
