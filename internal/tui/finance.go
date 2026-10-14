package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/georgezero/dreamteam-go/internal/api"
)

// Finance views, cycled with tab.
const (
	finOverview = iota
	finClaims
	finJournal
	numFinViews
)

var finViewNames = [numFinViews]string{"overview", "claims", "journal"}

// usd formats cents as whole dollars: $12,345 or −$12,345.
func usd(c int64) string {
	neg := c < 0
	if neg {
		c = -c
	}
	s := fmt.Sprint((c + 50) / 100)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	if neg {
		return "−$" + s
	}
	return "$" + s
}

// signed renders money red when negative.
func signed(c int64) string {
	if c < 0 {
		return alertStyle.Render(usd(c))
	}
	return usd(c)
}

// change renders a cash change with an arrow.
func change(c int64) string {
	switch {
	case c > 0:
		return openStyle.Render("▲" + usd(c))
	case c < 0:
		return alertStyle.Render("▼" + usd(-c))
	}
	return "±$0"
}

func margin(p api.FinancePeriod) string {
	if !p.MarginDefined {
		return "—"
	}
	s := pct(p.MarginBP)
	if p.MarginBP < 0 {
		return alertStyle.Render(s)
	}
	return s
}

// pct formats basis points as a percentage with one decimal, rounding
// toward zero, with a typographic minus.
func pct(bp int64) string {
	sign := ""
	if bp < 0 {
		sign, bp = "−", -bp
	}
	return fmt.Sprintf("%s%d.%d%%", sign, bp/100, bp%100/10)
}

// financeLine is the dashboard's compact money summary.
func financeLine(o api.Observation, w int) string {
	f := o.Finance
	t := f.Today
	l := fmt.Sprintf(" Cash %s %s today · Revenue today %s · Expenses today %s · Op. profit today %s · A/R %s",
		signed(f.CashCents), change(f.CashChangeTodayCents), usd(t.NetRevenueCents), usd(t.OperatingExpenseCents),
		signed(t.OperatingProfitCents), usd(f.ARCents))
	if w >= 140 {
		l += " · margin 30d " + margin(f.Trailing30)
	}
	return l
}

// financeBody renders the finance screen in n lines.
func (m *Model) financeBody(w, n int) []string {
	f := m.obs.Finance
	var tabs []string
	for i, name := range finViewNames {
		if i == m.finView {
			tabs = append(tabs, "["+name+"]")
		} else {
			tabs = append(tabs, name)
		}
	}
	head := fmt.Sprintf("FINANCE · %s %s · %s", m.obs.Weekday, shortDate(m.obs.Date), strings.Join(tabs, " "))
	lines := []string{titleStyle.Render(section(head, w))}
	switch m.finView {
	case finClaims:
		lines = append(lines, m.claimLines(f, w, n-1)...)
	case finJournal:
		lines = append(lines, m.journalLines(f, w, n-1)...)
	default:
		lines = append(lines, overviewLines(f, w)...)
	}
	if len(lines) > n {
		lines = lines[:n]
	}
	return lines
}

func overviewLines(f api.Finance, w int) []string {
	periods := []api.FinancePeriod{f.Today, f.MonthToDate, f.Trailing30, f.SinceStart}
	col := 15
	row := func(label string, v func(api.FinancePeriod) string) string {
		s := fmt.Sprintf(" %-28s", label)
		for _, p := range periods {
			s += padLeft(v(p), col)
		}
		return s
	}
	money := func(get func(api.FinancePeriod) int64) func(api.FinancePeriod) string {
		return func(p api.FinancePeriod) string { return signed(get(p)) }
	}
	out := []string{dimStyle.Render(fmt.Sprintf(" %-28s%s%s%s%s", "", padLeft("TODAY", col), padLeft("MONTH TO DATE", col),
		padLeft("TRAILING 30D", col), padLeft("SINCE DAY 1", col)))}
	out = append(out,
		row("Revenue earned (expected)", money(func(p api.FinancePeriod) int64 { return p.GrossRevenueCents })),
		row("  collection variance", money(func(p api.FinancePeriod) int64 { return -p.CollectionVarianceCents })),
		titleStyle.Render(row("Net revenue", money(func(p api.FinancePeriod) int64 { return p.NetRevenueCents }))),
		row("Operating expense", money(func(p api.FinancePeriod) int64 { return p.OperatingExpenseCents })),
		row("  Labor (pay + benefits)", money(func(p api.FinancePeriod) int64 { return p.LaborCents })),
		row("  Fixed (rent, leases, IT)", money(func(p api.FinancePeriod) int64 { return p.FixedCents })),
		row("  Supplies, contrast, PACS", money(func(p api.FinancePeriod) int64 { return p.VariableCents - p.BillingCents })),
		row("  Billing fees", money(func(p api.FinancePeriod) int64 { return p.BillingCents })),
		titleStyle.Render(row("Operating profit", money(func(p api.FinancePeriod) int64 { return p.OperatingProfitCents }))),
		row("  margin", margin),
		row("Cash collected", money(func(p api.FinancePeriod) int64 { return p.CollectionsCents })),
		row("Cash paid out", money(func(p api.FinancePeriod) int64 { return p.DisbursementsCents })),
		row("Net cash flow", money(func(p api.FinancePeriod) int64 { return p.NetCashFlowCents })),
		row("Reports final", func(p api.FinancePeriod) string { return fmt.Sprint(p.ReportsFinal) }),
		"",
	)
	out = append(out, fmt.Sprintf(" CASH %s  %s today   payroll %s %s · bills %s %s",
		signed(f.CashCents), change(f.CashChangeTodayCents), shortDate(f.NextPayroll.Date), usd(-f.NextPayroll.Cents),
		shortDate(f.NextPayables.Date), usd(-f.NextPayables.Cents)))
	c := f.Claims
	dar := "—"
	if c.DaysInAR >= 0 {
		dar = fmt.Sprint(c.DaysInAR)
	}
	out = append(out, fmt.Sprintf(" A/R  %s  days in A/R %s · denial rate 30d %s (%d of %d) · open claims %d (pending %d, submitted %d, in rework %d)",
		usd(f.ARCents), dar, pct(f.Trailing30.DenialRateBP), f.Trailing30.Denied, f.Trailing30.Adjudicated, c.Open, c.Pending, c.Submitted, c.Denied))
	var ag []string
	for _, b := range c.Aging {
		ag = append(ag, fmt.Sprintf("%s %s (%d)", b.Label, usd(b.Cents), b.Count))
	}
	out = append(out, dimStyle.Render(" A/R aging: "+strings.Join(ag, " · ")))
	out = append(out, dimStyle.Render(fmt.Sprintf(" Closed claims: paid %d · short paid %d · written off %d", c.Paid, c.ShortPaid, c.WrittenOff)))
	out = append(out, "", dimStyle.Render(" BY MODALITY (trailing 30 days)  reports   net revenue   direct cost   contribution   revenue/exam"))
	for _, mm := range f.Trailing30.ByModality {
		per := "—"
		if mm.ReportsFinal > 0 {
			per = usd(mm.NetRevenueCents / int64(mm.ReportsFinal))
		}
		out = append(out, fmt.Sprintf(" %-3s %35d %13s %13s %14s %14s", mm.Modality, mm.ReportsFinal, usd(mm.NetRevenueCents),
			usd(mm.DirectCostCents), signed(mm.NetRevenueCents-mm.DirectCostCents), per))
	}
	out = append(out, dimStyle.Render(" Direct cost: supplies, PACS, billing fees and the modality's lease and service; labor and rent are shared."))
	if len(f.CashHistory) > 1 {
		out = append(out, " Cash, last "+fmt.Sprint(len(f.CashHistory))+" days  "+sparkline(f.CashHistory))
	}
	return out
}

func padLeft(s string, n int) string {
	if pad := n - ansi.StringWidth(s); pad > 0 {
		return strings.Repeat(" ", pad) + s
	}
	return s
}

var sparks = []rune("▁▂▃▄▅▆▇█")

func sparkline(xs []api.DayAmount) string {
	lo, hi := xs[0].Cents, xs[0].Cents
	for _, x := range xs {
		lo, hi = min(lo, x.Cents), max(hi, x.Cents)
	}
	var b strings.Builder
	for _, x := range xs {
		k := 0
		if hi > lo {
			k = int((x.Cents - lo) * int64(len(sparks)-1) / (hi - lo))
		}
		b.WriteRune(sparks[k])
	}
	return b.String() + dimStyle.Render(fmt.Sprintf("  %s … %s", usd(lo), usd(hi)))
}

// claimLines lists open claims, oldest first.
func (m *Model) claimLines(f api.Finance, w, n int) []string {
	c := f.Claims
	out := []string{fmt.Sprintf(" Open claims %d · A/R %s · showing the oldest %d", c.Open, usd(f.ARCents), len(c.OpenClaims)),
		dimStyle.Render("  CLAIM   STUDY   PATIENT  PAYER       MOD EXAM                 FINAL        AGE  STATE        EXPECTED    BALANCE")}
	rows := n - len(out)
	m.finScroll = max(0, min(m.finScroll, len(c.OpenClaims)-rows))
	if len(c.OpenClaims) == 0 {
		out = append(out, dimStyle.Render("  (no open claims)"))
	}
	for i := m.finScroll; i < len(c.OpenClaims) && i < m.finScroll+rows; i++ {
		it := c.OpenClaims[i]
		state := it.State
		if state == api.ClaimDenied || state == api.ClaimResubmitted {
			state = warnStyle.Render(fmt.Sprintf("%-12s", state))
		} else {
			state = fmt.Sprintf("%-12s", state)
		}
		out = append(out, fmt.Sprintf("  %s  %s  %s  %-11s %-3s %-20s %s %4dd  %s %10s %10s", it.Claim, it.Study, it.Patient,
			trim(it.Payer, 11), it.Modality, trim(it.ExamName, 20), shortDate(it.Final[:10]), it.AgeDays, state,
			usd(it.ExpectedCents), usd(it.BalanceCents)))
	}
	return out
}

// journalLines shows the latest journal entries with their lines.
func (m *Model) journalLines(f api.Finance, w, n int) []string {
	out := []string{fmt.Sprintf(" Journal: %d entries, newest first (latest %d). Every entry balances.", f.JournalTotal, len(f.Journal))}
	var all []string
	for _, e := range f.Journal {
		ref := ""
		if e.Ref != "" {
			ref = " · " + e.Ref
		}
		all = append(all, titleStyle.Render(fmt.Sprintf(" %s %s %s", e.ID, strings.Replace(e.Time[5:16], "T", " ", 1), e.Memo))+dimStyle.Render(ref))
		for _, l := range e.Lines {
			acct := l.Account
			if l.Modality != "" {
				acct += " [" + l.Modality + "]"
			}
			if l.DebitCents > 0 {
				all = append(all, fmt.Sprintf("     Dr %-48s %12s", trim(acct, 48), usd(l.DebitCents)))
			} else {
				all = append(all, fmt.Sprintf("        Cr %-45s %25s", trim(acct, 45), usd(l.CreditCents)))
			}
		}
	}
	rows := n - len(out)
	m.finScroll = max(0, min(m.finScroll, len(all)-rows))
	end := min(len(all), m.finScroll+rows)
	return append(out, all[m.finScroll:end]...)
}
