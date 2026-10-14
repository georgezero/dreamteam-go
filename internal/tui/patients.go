package tui

import (
	"fmt"
	"strings"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// patientsBody renders the patient list for a day, or one patient's
// journey when one is open.
func (m *Model) patientsBody(w, n int) []string {
	if m.journeyID != "" {
		return m.journeyBody(w, n)
	}
	when := dayLabel(m.patDay, m.obs)
	lines := []string{titleStyle.Render(section(fmt.Sprintf("PATIENTS · %s · %s (tab) · %d", when,
		patientFilters[m.patFilter], len(m.patients)), w))}
	lines = append(lines, dimStyle.Render("  PATIENT  MOD EXAM                 APPT   ARRIVED       WAITED  STATUS"))
	rows := n - len(lines)
	if len(m.patients) == 0 {
		return append(lines, dimStyle.Render("  (none)"))
	}
	if m.patSel < m.patScroll {
		m.patScroll = m.patSel
	}
	if m.patSel >= m.patScroll+rows {
		m.patScroll = m.patSel - rows + 1
	}
	for i := m.patScroll; i < len(m.patients) && len(lines) < n; i++ {
		r := m.patients[i]
		mark := "  "
		if i == m.patSel {
			mark = "▶ "
		}
		appt := r.ApptTime
		if r.WalkIn {
			appt = "walk"
		}
		arr := r.Arrived
		if r.Punctuality != "" && r.Punctuality != "on time" {
			arr += " " + shortPunct(r.Punctuality)
		}
		waited := ""
		if r.Visit != "" {
			waited = minutes(r.WaitedMin)
		}
		line := fmt.Sprintf("%s%s   %-3s %-20s %-6s %-13s %6s  %s", mark, r.Patient, r.Modality,
			trim(r.ExamName, 20), appt, arr, waited, statusStyle(r.Status).Render(r.StatusText))
		if r.Priority == "urgent" {
			line += alertStyle.Render(" ‼")
		}
		lines = append(lines, line)
	}
	return lines
}

// shortPunct compresses "12m early" to "-12m" and "9m late" to "+9m".
func shortPunct(p string) string {
	switch {
	case strings.HasSuffix(p, " early"):
		return "-" + strings.TrimSuffix(p, " early")
	case strings.HasSuffix(p, " late"):
		return "+" + strings.TrimSuffix(p, " late")
	}
	return p
}

func statusStyle(s string) interface{ Render(...string) string } {
	switch s {
	case api.PatientWaiting:
		return openStyle
	case api.PatientMissed, api.PatientLeft, api.PatientUnseen, api.PatientLost:
		return warnStyle
	case api.PatientNoShow:
		return alertStyle
	case api.PatientCancelled, api.PatientRescheduled, api.PatientSentToBook:
		return dimStyle
	}
	return plainStyle
}

func (m *Model) journeyBody(w, n int) []string {
	j := m.journey
	head := fmt.Sprintf("PATIENT %s · %s %s · %s · %s", j.Patient, j.Modality, j.ExamName, j.Priority, j.Source)
	lines := []string{titleStyle.Render(section(head, w)),
		fmt.Sprintf(" Order %s · status: %s", j.Order, j.Status), ""}
	for _, s := range j.Steps {
		mark := openStyle.Render("✓")
		if !s.Done {
			mark = dimStyle.Render("○")
		}
		when := isoShort(s.Time)
		lines = append(lines, fmt.Sprintf(" %s %-20s %-12s %s", mark, s.Label, when, s.Detail))
	}
	lines = append(lines, "", dimStyle.Render(" Only what the practice has observed is shown."))
	if len(lines) > n {
		lines = lines[:n]
	}
	return lines
}

// isoShort turns "2026-10-13T09:18:05" into "Oct 13 09:18".
func isoShort(iso string) string {
	if len(iso) < 16 {
		return iso
	}
	return shortDate(iso[:10]) + " " + iso[11:16]
}

// dayLabel names a day offset: "Tue Oct 13 (today)".
func dayLabel(off int64, o api.Observation) string {
	rel := fmt.Sprintf("%d days ago", -off)
	switch {
	case off == 0:
		rel = "today"
	case off == 1:
		rel = "tomorrow"
	case off == -1:
		rel = "yesterday"
	case off > 1:
		rel = fmt.Sprintf("in %d days", off)
	}
	d, err := simtime.ParseDate(o.Date)
	if err != nil {
		return rel
	}
	d = d.AddDays(off)
	return fmt.Sprintf("%s %s (%s)", d.Weekday(), shortDate(d.String()), rel)
}
