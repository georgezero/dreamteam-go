package tui

import (
	"fmt"
	"slices"
	"strings"

	"github.com/georgezero/dreamteam-go/internal/api"
)

// historyDays is how many completed days the briefing and history screens
// read.
const historyDays = 30

// briefingBody is the opening briefing: the practice as it stands when the
// player takes over, built from the warm-up history (shown once, at the
// start; `b` reopens it).
func (m *Model) briefingBody(w, n int) []string {
	o := m.obs
	f := o.Finance
	op := f.Opening
	h := m.hist
	lines := []string{titleStyle.Render(section(fmt.Sprintf("TAKING OVER · %s · %s %s, day %d", o.Practice.Name, o.Weekday, shortDate(o.Date), o.Day), w))}
	add := func(s string) { lines = append(lines, s) }
	add(fmt.Sprintf(" The practice has operated for %d days before you take over (simulated history, same rules as play).", op.HistoryDays))
	add("")
	adj := ""
	switch {
	case op.AdjustmentCents > 0:
		adj = fmt.Sprintf(" · owner capital %s added at takeover (history left cash at %s)", usd(op.AdjustmentCents), signed(op.CashBeforeCents))
	case op.AdjustmentCents < 0:
		adj = fmt.Sprintf(" · owner distribution %s at takeover", usd(-op.AdjustmentCents))
	}
	add(titleStyle.Render(" MONEY") + fmt.Sprintf("  Opening cash %s%s", usd(op.OpeningCashCents), adj))
	t := f.Trailing30
	add(fmt.Sprintf("   Last 30 days: net revenue %s · expenses %s · operating profit %s (margin %s)",
		usd(t.NetRevenueCents), usd(t.OperatingExpenseCents), signed(t.OperatingProfitCents), margin(t)))
	add(fmt.Sprintf("   Cash flow, last 30 days: collected %s · paid out %s", usd(t.CollectionsCents), usd(t.DisbursementsCents)))
	add(fmt.Sprintf("   Result over the %d days of history: %s", op.HistoryDays, signed(op.HistoryResultCents)))
	add(fmt.Sprintf("   Carried over: receivables %s in %d open claims (days in A/R %d, denial rate 30d %s) · payables %s · payroll owed %s",
		usd(op.ReceivablesCents), op.OpenClaims, f.Claims.DaysInAR, pct(t.DenialRateBP), usd(op.PayablesCents), usd(op.AccruedPayrollCents)))
	add(fmt.Sprintf("   Next payroll %s (%s accrued so far) · bills due %s (%s so far)",
		shortDate(f.NextPayroll.Date), usd(f.NextPayroll.Cents), shortDate(f.NextPayables.Date), usd(f.NextPayables.Cents)))
	add(warnStyle.Render("   No credit line is modelled: if cash goes below zero it is tracked as a financing shortfall."))
	add("")

	week := lastDays(h, 7)
	var orders, exams, lost, noShows, open int
	var waits, tats []int64
	busy, openMin, staffed := map[string]int64{}, map[string]int64{}, map[string]int64{}
	var mods []string
	for _, d := range week {
		orders += d.Orders
		exams += d.Exams
		lost += d.Lost
		noShows += d.NoShows
		if d.Exams > 0 {
			open++
			waits = append(waits, d.WaitMedianMin)
		}
		if d.ReportsFinal > 0 {
			tats = append(tats, d.TATMedianMin)
		}
		for _, md := range d.ByModality {
			if _, ok := busy[md.Modality]; !ok {
				mods = append(mods, md.Modality)
			}
			busy[md.Modality] += md.BusyMin
			openMin[md.Modality] += md.OpenMin
			staffed[md.Modality] += md.StaffedMin
		}
	}
	add(titleStyle.Render(" PATIENTS") + fmt.Sprintf("  Last 7 days: %d orders, %d exams on %d open days, %d lost orders, %d no-shows · median wait by day %s",
		orders, exams, open, lost, noShows, minutes(midOf(waits))))
	var uo, us []string
	for _, md := range mods {
		uo = append(uo, md+" "+pctOf(busy[md], openMin[md]))
		us = append(us, md+" "+pctOf(busy[md], staffed[md]))
	}
	add("   Room utilization (7 days), of open hours:    " + strings.Join(uo, " · "))
	add("   Room utilization (7 days), of staffed hours: " + strings.Join(us, " · "))
	add(fmt.Sprintf("   Booked ahead: %d appointments from today · recall list %d", op.BookedAppointments, op.RecallList))
	add("")
	add(titleStyle.Render(" REPORTING") + fmt.Sprintf("  Worklist now: %d unread, %d being read · report TAT median by day (7 days) %s",
		o.Worklist.Unread, o.Worklist.Reading, minutes(midOf(tats))))
	add("")
	appts := 0
	for _, d := range o.Demand.Modalities {
		appts += d.ApptsToday
	}
	roster := map[string][]string{}
	for _, e := range o.Staff {
		if e.Shift != "" {
			roster[e.Role] = append(roster[e.Role], fmt.Sprintf("%s %s", e.Name, e.Shift))
		}
	}
	add(titleStyle.Render(" TODAY") + fmt.Sprintf("  Open %s · %d appointments booked", o.Practice.Hours, appts))
	for _, role := range []string{"technologist", "radiologist", "front_desk", "ops_manager"} {
		if len(roster[role]) > 0 {
			add(fmt.Sprintf("   %-13s %s", strings.ReplaceAll(role, "_", " "), trim(strings.Join(roster[role], " · "), w-20)))
		}
	}
	add("")
	add(dimStyle.Render(" All figures are illustrative and uncalibrated (ROADMAP step 18)."))
	add(dimStyle.Render(" enter dashboard · h daily history · f finance · w worklist · s schedule · space start the day"))
	if len(lines) > n {
		lines = lines[:n]
	}
	return lines
}

func lastDays(h []api.DaySummary, n int) []api.DaySummary { return h[max(0, len(h)-n):] }

// midOf is the median of per-day medians (a summary, not a pooled median).
func midOf(xs []int64) int64 {
	if len(xs) == 0 {
		return 0
	}
	s := slices.Clone(xs)
	slices.Sort(s)
	return s[len(s)/2]
}

func pctOf(a, b int64) string {
	if b <= 0 {
		return "—"
	}
	return pct(a * 10_000 / b)
}

// historyBody is the daily history: one row per completed day, newest at
// the bottom; days before day 1 are marked as history.
func (m *Model) historyBody(w, n int) []string {
	lines := []string{titleStyle.Render(section(fmt.Sprintf("DAILY HISTORY · last %d days (· = before day 1) · ACTUAL CASH", len(m.hist)), w)),
		dimStyle.Render(fmt.Sprintf("   DATE         ORD EXAMS  NS LOST    WAIT     TAT UNREAD  %-19s    NET REV  EXPENSES    PROFIT   CASH END",
			strings.Join(m.obsMods(), " ")+" %")),
		dimStyle.Render("                                                         utilization, open hrs")}
	rows := n - len(lines)
	h := m.hist
	m.histScroll = max(0, min(m.histScroll, len(h)-rows))
	end := len(h) - m.histScroll
	for _, d := range h[max(0, end-rows):end] {
		mark := " "
		if d.BeforeStart {
			mark = "·"
		}
		var u []string
		for _, md := range d.ByModality {
			if md.OpenMin == 0 {
				u = append(u, "  —")
			} else {
				u = append(u, fmt.Sprintf("%3d", md.UtilOpenBP/100))
			}
		}
		row := fmt.Sprintf(" %s %s %-6s %4d %5d %3d %4d %7s %7s %6d  %-19s", mark, d.Weekday, shortDate(d.Date),
			d.Orders, d.Exams, d.NoShows, d.Lost, minutes(d.WaitMedianMin), minutes(d.TATMedianMin), d.UnreadAtClose,
			strings.Join(u, " ")) + padLeft(usd(d.NetRevenueCents), 10) + padLeft(usd(d.ExpenseCents), 10) +
			padLeft(signed(d.OperatingProfitCents), 10) + padLeft(signed(d.CashEndCents), 11)
		if d.BeforeStart {
			row = dimStyle.Render(row)
		}
		lines = append(lines, row)
	}
	if len(h) == 0 {
		lines = append(lines, dimStyle.Render("  (no completed days yet)"))
	}
	if op := m.obs.Finance.Opening; op.Done && op.AdjustmentCents != 0 {
		lines = append(lines, dimStyle.Render(fmt.Sprintf(" takeover: owner capital %s on %s brought cash from %s to %s (cash before day 1 is as it was)",
			plusUSD(op.AdjustmentCents), strings.Replace(op.Time[5:16], "T", " ", 1), usd(op.CashBeforeCents), usd(op.OpeningCashCents))))
	}
	return lines
}

// obsMods lists the modalities in room order.
func (m *Model) obsMods() []string {
	var out []string
	for _, d := range m.obs.Demand.Modalities {
		out = append(out, fmt.Sprintf("%3s", d.Modality))
	}
	return out
}
