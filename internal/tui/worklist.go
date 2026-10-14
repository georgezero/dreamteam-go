package tui

import (
	"fmt"
	"strings"

	"github.com/georgezero/dreamteam-go/internal/api"
)

var policyNames = map[string]string{
	"urgent_then_fifo": "urgent then FIFO", "fifo": "FIFO", "oldest_first": "oldest first", "shortest_first": "shortest first",
}

// worklistBody renders the radiology worklist: unread and reading studies
// in the order the policy offers them, turnaround and backlog, and the
// latest reports.
func (m *Model) worklistBody(w, n int) []string {
	wl := m.obs.Worklist
	head := fmt.Sprintf("WORKLIST · %d unread · %d reading · %d urgent · policy: %s", wl.Unread, wl.Reading, wl.Urgent, policyNames[wl.Policy])
	lines := []string{titleStyle.Render(section(head, w))}
	lines = append(lines, reportingSummary(wl, w)...)
	var by []string
	for _, b := range wl.ByModality {
		by = append(by, fmt.Sprintf("%s %d", b.Key, b.Unread+b.Reading))
	}
	var bp []string
	for _, b := range wl.ByPriority {
		bp = append(bp, fmt.Sprintf("%s %d", b.Key, b.Unread+b.Reading))
	}
	lines = append(lines, dimStyle.Render(" Backlog by modality "+strings.Join(by, " · ")+"   by priority "+strings.Join(bp, " · ")+
		"   oldest "+minutes(wl.OldestMin)))
	lines = append(lines, dimStyle.Render("  PRI       STUDY   PATIENT  MOD EXAM                  SINCE IMAGING  STATUS"))

	recentRows := min(5, len(wl.Recent))
	room := n - len(lines) - recentRows - 1
	if len(wl.Items) == 0 {
		lines = append(lines, dimStyle.Render("  (worklist empty)"))
	}
	m.wlSel = max(0, min(m.wlSel, len(wl.Items)-1))
	if m.wlSel < m.wlScroll {
		m.wlScroll = m.wlSel
	}
	if room > 0 && m.wlSel >= m.wlScroll+room {
		m.wlScroll = m.wlSel - room + 1
	}
	for i := m.wlScroll; i < len(wl.Items) && i < m.wlScroll+room; i++ {
		lines = append(lines, worklistRow(wl.Items[i], i == m.wlSel))
	}
	for len(lines) < n-recentRows-1 {
		lines = append(lines, "")
	}
	if recentRows > 0 {
		lines = append(lines, titleStyle.Render(section("LATEST REPORTS", w)))
		for _, it := range wl.Recent[:recentRows] {
			tat := minutes(it.TATMin)
			if it.Overdue {
				tat = warnStyle.Render(tat + " (over target)")
			}
			lines = append(lines, fmt.Sprintf("  %s %-3s %-20s %-10s TAT %s", it.Study, it.Modality, trim(it.ExamName, 20), it.Radiologist, tat))
		}
	}
	if len(lines) > n {
		lines = lines[:n]
	}
	return lines
}

func worklistRow(it api.WorklistItem, selected bool) string {
	mark := "  "
	if selected {
		mark = "▶ "
	}
	pri := fmt.Sprintf("%-9s", it.Priority)
	switch it.Priority {
	case "urgent":
		pri = alertStyle.Render("‼ urgent  ")
	case "soon":
		pri = warnStyle.Render(pri) + " "
	default:
		pri += " "
	}
	age := fmt.Sprintf("%-13s", minutes(it.AgeMin))
	if it.Overdue {
		age = warnStyle.Render(age)
	}
	status := "unread"
	if it.State == api.StudyReading {
		status = openStyle.Render(fmt.Sprintf("reading · %s %dm", it.Radiologist, it.ReadingMin))
	}
	return fmt.Sprintf("%s%s%s  %s   %-3s %-21s %s  %s", mark, pri, it.Study, it.Patient, it.Modality, trim(it.ExamName, 21), age, status)
}

// reportingSummary is the turnaround line shared by the dashboard and the
// worklist screen.
func reportingSummary(wl api.Worklist, w int) []string {
	unread := fmt.Sprintf(" Unread %d", wl.Unread)
	if wl.Urgent > 0 {
		unread += alertStyle.Render(fmt.Sprintf(" ‼%d", wl.Urgent))
	}
	breach := ""
	if wl.BreachToday > 0 {
		breach = warnStyle.Render(fmt.Sprintf(" · TAT breaches %d", wl.BreachToday))
	}
	l := unread + fmt.Sprintf(" · reading %d · oldest %s · final today %d · TAT today median %s, p90 %s",
		wl.Reading, minutes(wl.OldestMin), wl.FinalToday, minutes(wl.TATMedianMin), minutes(wl.TATP90Min)) + breach
	if w >= 100 {
		l += dimStyle.Render(fmt.Sprintf(" · wait %s · read %s (medians)", minutes(wl.WaitMedianMin), minutes(wl.ReadMedianMin)))
	}
	return []string{l}
}
