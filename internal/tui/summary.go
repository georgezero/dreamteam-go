package tui

import (
	"fmt"
	"strings"

	"github.com/georgezero/dreamteam-go/internal/api"
)

// The end-of-day summary (TUI_DESIGN §4.10): `e` runs today's operations to
// their close and shows the day; ←→ page through earlier days; tab shows
// the evaluation window since the takeover, then the T90 evaluation.

// sumViews are the summary screen's views.
const (
	sumDay = iota
	sumWindow
	sumT90
	numSumViews
)

// alertMark is the severity glyph.
func alertMark(s api.Severity) string {
	if s == api.Alert {
		return alertStyle.Render("‼")
	}
	return warnStyle.Render("⚠")
}

// pctR formats a ratio as a whole percentage, "—" when undefined.
func pctR(r api.Ratio) string {
	if !r.Defined() {
		return "—"
	}
	return fmt.Sprintf("%d%%", (r.BP()+50)/100)
}

// dur formats seconds as minutes or hours and minutes.
func dur(sec int64) string { return minutes(sec / 60) }

// distText is "p50 9m · p90 34m", or "—" with no sample.
func distText(d api.Dist) string {
	if d.Count == 0 {
		return "—"
	}
	return fmt.Sprintf("p50 %s · p90 %s", dur(d.P50Sec), dur(d.P90Sec))
}

// modUtil lists each room's utilization by f, "XR 54% · US 92%".
func modUtil(rooms []api.RoomMetrics, f func(api.RoomMetrics) api.Ratio) string {
	var out []string
	for _, r := range rooms {
		out = append(out, r.Modality+" "+pctR(f(r)))
	}
	return strings.Join(out, " · ")
}

// staffUtil summarizes technologist and radiologist time.
func staffUtil(ops api.PeriodMetrics) (tech, read string) {
	var tb, td, rb, rd int64
	for _, s := range ops.Staff {
		switch s.Role {
		case "technologist":
			tb += s.BusySec
			td += s.OnDutySec
		case "radiologist":
			rb += s.ReadInBlockSec
			rd += s.BlockSec
		}
	}
	return pctR(api.Ratio{Num: tb, Den: td}), pctR(api.Ratio{Num: rb, Den: rd})
}

// summaryBody renders the summary screen. Below 110 columns every line
// uses its short form, so the day fits 80×24 without truncation.
func (m *Model) summaryBody(w, n int) []string {
	switch m.sumView {
	case sumWindow:
		return m.windowBody(w, n)
	case sumT90:
		return m.t90Body(w, n)
	}
	d := m.day
	ops := d.Ops
	wide := w >= 110
	pick := func(long, short string) string {
		if wide {
			return long
		}
		return short
	}
	head := fmt.Sprintf("END OF DAY · %s %s", d.Weekday, shortDate(d.Date))
	switch {
	case !d.Complete:
		head += fmt.Sprintf(" · so far, to %s", clockOf(d.AsOf))
	case d.BeforeStart:
		head += " · history (before day 1)"
	}
	if m.sumOffset == 0 && m.obs.Practice.OperationalClose != "" {
		head += pick(" · operations closed ", " · closed ") + m.obs.Practice.OperationalClose
	}
	lines := []string{titleStyle.Render(section(head, w))}
	add := func(label, text string) { lines = append(lines, fmt.Sprintf(" %-9s %s", label, text)) }
	if d.Takeover != "" {
		lines = append(lines, dimStyle.Render(pick(
			fmt.Sprintf(" Takeover day: figures run from midnight, including the hours before the %s takeover · tab: since takeover", clockOf(d.Takeover)),
			fmt.Sprintf(" Takeover day: from midnight, incl. before %s · tab: since takeover", clockOf(d.Takeover)))))
	}

	var byMod []string
	for _, md := range ops.Demand.ByModality {
		byMod = append(byMod, fmt.Sprintf("%s %d", md.Modality, md.Exams))
	}
	wi := ops.WalkIns
	add("Patients", pick(
		fmt.Sprintf("%d exams (%s) · walk-ins %d: admitted %d, offered later %d (booked %d), left %d",
			ops.Exams, strings.Join(byMod, " · "), wi.Arrived, wi.Admitted, wi.Offered, wi.OfferBooked, wi.Left),
		fmt.Sprintf("%d exams (%s) · walk-ins %d, left %d", ops.Exams, strings.Join(byMod, " "), wi.Arrived, wi.Left)))
	dm := ops.Demand
	lost := fmt.Sprintf("lost %d (%s)", dm.Lost, pctR(dm.LostRate))
	if dm.Lost > 0 {
		lost = warnStyle.Render(lost)
	}
	ap := ops.Appointments
	if wide {
		add("Demand", fmt.Sprintf("%d orders · booked %d · recall %d · %s · cancelled %d", dm.Received, dm.Booked, dm.Recall, lost, dm.Cancelled))
		add("Schedule", fmt.Sprintf("%d appointments · no-shows %d (%s) · cancelled %d (%s) · slot fill %s of bookable",
			ap.Scheduled, ap.NoShows, pctR(ap.NoShowRate), ap.Cancelled, pctR(ap.CancelRate), pctR(ap.Fill)))
	} else {
		add("Demand", fmt.Sprintf("%d orders · %s · no-shows %s · slot fill %s", dm.Received, lost, pctR(ap.NoShowRate), pctR(ap.Fill)))
	}
	wt := ops.Waits
	over := fmt.Sprintf("over 30m %d", wt.Over30)
	if wt.Over30 > 0 {
		over = warnStyle.Render(over)
	}
	add("Waits", pick(fmt.Sprintf("arrival→exam %s · %s · past appt %s · to register %s",
		distText(wt.ToExam), over, distText(wt.PastAppointment), distText(wt.Registration)),
		fmt.Sprintf("arrival→exam %s · %s", distText(wt.ToExam), over)))
	sep := pick(" · ", " ")
	add("Rooms", pick("busy of open hours    ", "busy/open    ")+modUtilSep(ops.Rooms, sep, func(r api.RoomMetrics) api.Ratio { return r.UtilOpen }))
	add("", pick("busy of staffed hours ", "busy/staffed ")+modUtilSep(ops.Rooms, sep, func(r api.RoomMetrics) api.Ratio { return r.UtilStaffed }))
	add("", pick("staffed of open hours ", "staffed/open ")+modUtilSep(ops.Rooms, sep, func(r api.RoomMetrics) api.Ratio { return r.Coverage }))
	tech, read := staffUtil(ops)
	wf := ops.Workforce
	ot := fmt.Sprintf("overtime %s (%s)", dur(wf.OvertimeSec), usd(wf.OvertimeCostCents))
	if wf.OvertimeSec > 0 {
		ot = warnStyle.Render(ot)
	}
	add("Staff", pick(fmt.Sprintf("technologists busy %s of duty · reading %s of site blocks · %s · breaks skipped %d",
		tech, read, ot, wf.SkippedBreaks),
		fmt.Sprintf("techs busy %s · reading %s of blocks · %s", tech, read, ot)))
	if wide && (wf.NoCoverage > 0 || wf.DeskSelfPatientSec > 0) {
		add("", fmt.Sprintf("rescheduled for coverage %d · patients waiting to register with no desk staff %s",
			wf.NoCoverage, dur(wf.DeskSelfPatientSec)))
	}
	rp := ops.Reporting
	add("Reporting", pick(fmt.Sprintf("final %d · TAT of studies imaged %s (%d unfinished) · over target %d · unread at close %d",
		rp.Final, distText(rp.TAT), rp.Unfinished, rp.OverTarget, d.UnreadAtClose),
		fmt.Sprintf("final %d · TAT %s · unread at close %d", rp.Final, distText(rp.TAT), d.UnreadAtClose)))
	mo := d.Money
	add("Money", pick(fmt.Sprintf("net revenue %s · expenses %s · operating profit %s (%s)",
		usd(mo.NetRevenueCents), usd(mo.OperatingExpenseCents), signed(mo.OperatingProfitCents), margin(mo)),
		fmt.Sprintf("revenue %s · expenses %s · profit %s (%s)",
			usd(mo.NetRevenueCents), usd(mo.OperatingExpenseCents), signed(mo.OperatingProfitCents), margin(mo))))
	add("Cash", pick(fmt.Sprintf("collected %s · paid %s · cash %s · owed to us %s · we owe %s + payroll %s",
		usd(mo.CollectionsCents), usd(mo.DisbursementsCents), signed(d.CashEndCents), usd(d.ARCents), usd(d.APCents), usd(d.AccruedPayrollCents)),
		fmt.Sprintf("in %s · out %s · cash %s · A/R %s", usd(mo.CollectionsCents), usd(mo.DisbursementsCents), signed(d.CashEndCents), usd(d.ARCents))))
	al := ops.Alerts
	if al.Warn+al.Alert == 0 {
		add("Alerts", "none raised")
	} else {
		add("Alerts", fmt.Sprintf("%d raised (%d urgent)", al.Warn+al.Alert, al.Alert))
		for i, t := range al.Top {
			if i == 2 || len(lines) >= n-2 {
				break
			}
			add("", trim(t, w-12))
		}
	}
	for _, s := range insights(d) {
		if len(lines) >= n {
			break
		}
		lines = append(lines, " "+titleStyle.Render("Insight:")+" "+trim(s, w-11))
	}
	if len(lines) > n {
		lines = lines[:n]
	}
	return lines
}

// modUtilSep lists each room's figure from f, joined by sep.
func modUtilSep(rooms []api.RoomMetrics, sep string, f func(api.RoomMetrics) api.Ratio) string {
	var out []string
	for _, r := range rooms {
		out = append(out, r.Modality+" "+pctR(f(r)))
	}
	return strings.Join(out, sep)
}

// insights are rule-based observations from the day's observed metrics:
// where the levers are. They never use hidden state.
func insights(d api.DaySummary) []string {
	var out []string
	ops := d.Ops
	for _, r := range ops.Rooms {
		if r.UtilStaffed.Defined() && r.UtilStaffed.BP() >= 8_500 && r.Coverage.Defined() && r.Coverage.BP() < 9_000 {
			out = append(out, fmt.Sprintf("%s was busy %s of its staffed time but staffed only %s of open hours: the roster, not the room, limits %s.",
				r.Modality, pctR(r.UtilStaffed), pctR(r.Coverage), r.Modality))
			break
		}
	}
	for _, md := range ops.Demand.ByModality {
		if md.Lost >= 3 {
			out = append(out, fmt.Sprintf("%s lost %d of %d orders: its earliest open slot was too far out.", md.Modality, md.Lost, md.Received))
			break
		}
	}
	if len(out) > 2 {
		out = out[:2]
	}
	return out
}

// windowBody shows the evaluation window: the takeover baseline and
// performance since, operating results apart from cash and working capital.
func (m *Model) windowBody(w, n int) []string {
	ev := m.eval
	if !ev.Started {
		return []string{titleStyle.Render(section("SINCE TAKEOVER", w)), dimStyle.Render(" (play has not started)")}
	}
	wide := w >= 110
	pick := func(long, short string) string {
		if wide {
			return long
		}
		return short
	}
	b := ev.Baseline
	head := fmt.Sprintf("SINCE TAKEOVER · %s %s → %s", shortDate(b.Time[:10]), clockOf(b.Time), clockOf(ev.AsOf))
	if ev.End != "" {
		head += " · ends " + shortDate(ev.End[:10]) + " " + clockOf(ev.End)
	}
	if ev.Ended {
		head += " · ENDED"
	}
	lines := []string{titleStyle.Render(section(head, w))}
	add := func(label, text string) { lines = append(lines, fmt.Sprintf(" %-9s %s", label, text)) }
	add("Inherited", pick(fmt.Sprintf("cash %s · A/R %s in %d claims · A/P %s · payroll %s · %d unreported · %d booked",
		signed(b.CashCents), usd(b.ReceivablesCents), b.OpenClaims, usd(b.PayablesCents), usd(b.AccruedPayrollCents),
		b.Unreported, b.BookedAppointments),
		fmt.Sprintf("cash %s · A/R %s · A/P %s · payroll %s", signed(b.CashCents), usd(b.ReceivablesCents),
			usd(b.PayablesCents), usd(b.AccruedPayrollCents))))
	mo := ev.Money
	mg := "—"
	if mo.Margin.Defined() {
		mg = fmt.Sprintf("%.1f%%", float64(mo.Margin.BP())/100)
	}
	add("Adjusted", pick(fmt.Sprintf("baseline-adjusted: net revenue %s · expenses %s · operating profit %s (%s)",
		usd(mo.NetRevenueCents), usd(mo.OperatingExpenseCents), signed(mo.OperatingProfitCents), mg),
		fmt.Sprintf("revenue %s · expenses %s · profit %s (%s)",
			usd(mo.NetRevenueCents), usd(mo.OperatingExpenseCents), signed(mo.OperatingProfitCents), mg)))
	add("", dimStyle.Render(pick(fmt.Sprintf("excludes inherited-claim collection variance (%s to revenue) · includes %s of takeover-day accruals",
		signed(-mo.InheritedVarianceCents), usd(mo.AllocatedAccrualCents)),
		fmt.Sprintf("excl. inherited variance %s · incl. accruals %s", signed(-mo.InheritedVarianceCents), usd(mo.AllocatedAccrualCents)))))
	add("Ledger", pick(fmt.Sprintf("as posted (the accrual P&L): net revenue %s · expenses %s · operating profit %s",
		usd(mo.Ledger.NetRevenueCents), usd(mo.Ledger.OperatingExpenseCents), signed(mo.Ledger.OperatingProfitCents)),
		fmt.Sprintf("as posted: revenue %s · profit %s", usd(mo.Ledger.NetRevenueCents), signed(mo.Ledger.OperatingProfitCents))))
	add("Cash flow", pick(fmt.Sprintf("collected %s (%s on inherited claims) · paid %s (%s settling inherited bills/payroll) · net %s",
		usd(mo.CollectionsCents), usd(mo.InheritedCollectionsCents), usd(mo.DisbursementsCents), usd(mo.InheritedPaidCents), signed(mo.OperatingCashFlowCents)),
		fmt.Sprintf("in %s (inherited %s) · out %s · net %s", usd(mo.CollectionsCents), usd(mo.InheritedCollectionsCents),
			usd(mo.DisbursementsCents), signed(mo.OperatingCashFlowCents))))
	add("Change", pick(fmt.Sprintf("A/R %s · A/P %s · payroll owed %s · capital transfers %s · cash now %s (low %s)",
		delta(mo.ARChangeCents), delta(mo.APChangeCents), delta(mo.PayrollChangeCents), delta(mo.CapitalTransfersCents),
		signed(mo.CashEndCents), signed(mo.MinCashCents)),
		fmt.Sprintf("A/R %s · A/P %s · payroll %s · cash %s", delta(mo.ARChangeCents), delta(mo.APChangeCents),
			delta(mo.PayrollChangeCents), signed(mo.CashEndCents))))
	rec := openStyle.Render("✓ cash bridge reconciles (profit → cash flow → cash)")
	if !mo.Reconciled {
		rec = alertStyle.Render("✗ cash bridge does not reconcile")
	}
	add("", rec)
	ops := ev.Ops
	add("Patients", fmt.Sprintf("%d exams · orders %d, lost %d (%s) · walk-ins left %d · no-shows %s",
		ops.Exams, ops.Demand.Received, ops.Demand.Lost, pctR(ops.Demand.LostRate), ops.WalkIns.Left,
		pctR(ops.Appointments.NoShowRate)))
	add("Waits", fmt.Sprintf("arrival→exam %s · over 30m %d", distText(ops.Waits.ToExam), ops.Waits.Over30))
	sep := pick(" · ", " ")
	add("Rooms", pick("busy of open hours    ", "busy/open    ")+modUtilSep(ops.Rooms, sep, func(r api.RoomMetrics) api.Ratio { return r.UtilOpen }))
	add("", pick("busy of staffed hours ", "busy/staffed ")+modUtilSep(ops.Rooms, sep, func(r api.RoomMetrics) api.Ratio { return r.UtilStaffed }))
	rp := ops.Reporting
	add("Reporting", pick(fmt.Sprintf("TAT of studies imaged %s (%d unfinished) · final %d (%d inherited) · backlog %d → %d",
		distText(rp.TAT), rp.Unfinished, rp.Final, rp.Inherited, rp.BacklogStart, rp.BacklogEnd),
		fmt.Sprintf("TAT %s · final %d (%d inherited) · backlog %d→%d", distText(rp.TAT), rp.Final, rp.Inherited,
			rp.BacklogStart, rp.BacklogEnd)))
	add("Overtime", fmt.Sprintf("%s (%s) · alerts raised %d", dur(ops.Workforce.OvertimeSec), usd(ops.Workforce.OvertimeCostCents),
		ops.Alerts.Warn+ops.Alerts.Alert))
	lines = append(lines, dimStyle.Render(" Measurements as played · tab: T90 evaluation · baseline "+b.Digest[:12]))
	if len(lines) > n {
		lines = lines[:n]
	}
	return lines
}

// delta formats a change in cents with its sign: "+$1,706", "−$45", "$0".
func delta(c int64) string {
	if c > 0 {
		return "+" + usd(c)
	}
	return signed(c)
}

// clockOf extracts "HH:MM" from a practice-local ISO time.
func clockOf(iso string) string {
	if len(iso) >= 16 {
		return iso[11:16]
	}
	return iso
}

// alertLines is the dashboard's alert panel: active alerts, most severe
// first, the first with its explanation when there is room.
func alertLines(o api.Observation, w, room int) []string {
	act := o.Alerts.Active
	head := fmt.Sprintf("ALERTS · %d active · %d raised today", len(act), o.Alerts.RaisedToday)
	out := []string{titleStyle.Render(section(head, w))}
	if len(act) == 0 {
		return append(out, dimStyle.Render(" ✓ nothing needs attention"))
	}
	for i, a := range act {
		if len(out) >= room {
			break
		}
		out = append(out, fmt.Sprintf(" %s %s %s %s", alertMark(a.Severity), clockOf(a.Since), a.Text, dimStyle.Render("["+a.Screen+"]")))
		if i == 0 && a.Why != "" && len(out) < room && len(act) < room-1 {
			out = append(out, dimStyle.Render("   why: "+trim(a.Why, w-9)))
		}
	}
	return out
}
