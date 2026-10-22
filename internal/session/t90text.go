package session

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/georgezero/dreamteam-go/internal/eval"
)

// Lines renders the T90 evaluation as plain text for the terminal and the
// CLI, for a width of at least 80 columns (below 112, compact: prose
// wrapped and the vector's explanations listed once): the metric vector
// for the plan and the baseline with their difference, every service gate
// of the plan's seed, the other gates and the policy's verdict. It formats
// the report; it decides nothing.
func (t *T90) Lines(width int) []string {
	wide := width >= 112
	rep := t.Report
	plan, base := t90PlanReport(rep, T90Plan), t90PlanReport(rep, T90Baseline)
	var cmp *eval.Comparison
	for i := range rep.Comparisons {
		if rep.Comparisons[i].Candidate == T90Plan {
			cmp = &rep.Comparisons[i]
		}
	}
	var out []string
	add := func(f string, a ...any) {
		s := fmt.Sprintf(f, a...)
		if wide {
			out = append(out, s)
			return
		}
		out = append(out, wrapText(s, width)...)
	}
	add("T90 EVALUATION · policy %s (the evaluation policy in force) · seed %d · scenario %s", t.PolicyID, t.Seed, t.Scenario[:12])
	later := ""
	if t.Later > 0 {
		later = fmt.Sprintf("; %d later batch(es) belong to a later period and are not evaluated", t.Later)
	}
	add("Your plan: %d decision batch(es) before T90 (%s)%s. Judged against no action on the same seed.", t.Decisions, isoText(t.T90), later)
	if t.Projected {
		add("PROJECTED: you are on day %.1f. Days %.1f–90 assume your plan as it stands, with no further decision.", t.AsOfDay, t.AsOfDay)
	} else {
		add("MEASURED to T90: you are on day %.1f.", t.AsOfDay)
	}
	add("After T90 both plans run with no decision: the evaluation tail to %s, observed to %s.", isoText(t.HorizonEnd), isoText(t.ObservationEnd))
	add("Tail and T180 values describe that fixed policy, not your play after T90.")
	if len(rep.Integrity) > 0 || plan == nil || base == nil || cmp == nil {
		add("INVALID: %s", strings.Join(append(rep.Integrity, t90Integrity(plan, base, cmp)...), "; "))
		return out
	}
	add("")
	add("VERDICT on this seed, by the policy's rules: %s · %s", cmp.Verdict, cmp.Classification)
	for i, r := range cmp.Reasons {
		if i == 8 {
			add("  … %d more", len(cmp.Reasons)-i)
			break
		}
		add("  · %s", r)
	}
	pm, bm := plan.PerSeed[0].Measures, base.PerSeed[0].Measures
	pw, bw := pm.Windows[eval.WinT90], bm.Windows[eval.WinT90]
	add("")
	var legend []string
	row := func(name, a, b, d, meaning string) {
		if wide {
			out = append(out, fmt.Sprintf("  %-28s %13s %13s %13s  %s", name, a, b, d, meaning))
			return
		}
		out = append(out, fmt.Sprintf("  %-24s %11s %11s %11s", name, a, b, d))
		if l := fmt.Sprintf("  %s: %s", strings.TrimSpace(name), meaning); meaning != "" && !slices.Contains(legend, l) {
			legend = append(legend, l)
		}
	}
	if wide {
		out = append(out, "METRIC VECTOR · takeover to T90 unless noted     your plan      no action     difference")
	} else {
		out = append(out, "METRIC VECTOR · to T90 unless noted  your plan   no action  difference")
	}
	money := func(name string, a, b int64, meaning string) {
		row(name, signedUSD(a), signedUSD(b), signedUSD(a-b), meaning)
	}
	count := func(name string, a, b int64, meaning string) {
		row(name, fmt.Sprint(a), fmt.Sprint(b), fmt.Sprintf("%+d", a-b), meaning)
	}
	rate := func(name string, a, b eval.Ratio, meaning string) {
		d := "—"
		if a.Defined && b.Defined {
			d = fmt.Sprintf("%+.1f pts", 100*(a.Value-b.Value))
		}
		row(name, ratioText(a), ratioText(b), d, meaning)
	}
	money("Operating profit", pw.Money.OperatingProfitCents, bw.Money.OperatingProfitCents, "net revenue − operating expenses")
	rate("Operating margin", pw.Money.Margin, bw.Money.Margin, "profit ÷ net revenue")
	money("Net revenue", pw.Money.NetRevenueCents, bw.Money.NetRevenueCents, "accrued, after contractual adjustments")
	if po, bo := pm.Windows[eval.WinOps], bm.Windows[eval.WinOps]; po != nil && bo != nil {
		money("Profit, days 31–90", po.Money.OperatingProfitCents, bo.Money.OperatingProfitCents, "after the transition month")
	}
	money("Cash at T90", pw.Money.End.CashCents, bw.Money.End.CashCents, "on hand at the endpoint")
	money("Lowest cash", pm.MinCash, bm.MinCash, "to the end of observation")
	money("Funding gap", pm.FundingGap, bm.FundingGap, "committed liabilities over cash at the end (0: covered)")
	rate("Served due demand", pw.Access.ServedFraction, bw.Access.ServedFraction, "orders wanted by T90 imaged by the end of observation")
	count("Lost due orders", pw.Access.LostDue, bw.Access.LostDue, "never imaged (patient-caused included)")
	count("  of them patient-caused", pw.Access.PatientCaused, bw.Access.PatientCaused, "cancellations and no-shows not rebooked")
	count("Due orders still open", pw.Access.UnresolvedDue, bw.Access.UnresolvedDue, "incomplete: open when observation ended")
	for _, q := range []string{"urgent", "soon", "routine"} {
		pc, bc := pw.Cohorts[q], bw.Cohorts[q]
		if pc == nil || bc == nil {
			continue
		}
		rate(cap1(q)+" report breaches", pc.BreachRate, bc.BreachRate, "reports later than the target, ÷ studies imaged")
		row("  completed TAT p90", minText(pc.CompletedTATMin), minText(bc.CompletedTATMin), "", "of finished reports")
		count("  unfinished at T90", pc.Unfinished, bc.Unfinished, "incomplete: no report yet at the endpoint")
	}
	count("Remote reports", pw.Remote.RemoteReports, bw.Remote.RemoteReports, "read by the teleradiology vendor")
	money("Remote reading cost", pw.Remote.TotalCostCents, bw.Remote.TotalCostCents, "retainers and fees")
	if pt, bt := pm.Windows[eval.WinTail], bm.Windows[eval.WinTail]; pt != nil && bt != nil {
		money("Tail operating profit", pt.Money.OperatingProfitCents, bt.Money.OperatingProfitCents, "T90 to the tail end, fixed policy")
	}
	if len(legend) > 0 {
		add("")
		add("WHAT THE ROWS MEAN")
		for _, l := range legend {
			add("%s", l)
		}
	}
	add("")
	add("SERVICE GATES · report-breach limits per priority; \"decides\" gates set the seed's service, the rest are reported")
	add("  pass/fail: the rate against its limit · insufficient: too few studies for the policy to decide")
	for _, g := range plan.PerSeed[0].Gates {
		role := "reported"
		if g.PerSeed {
			role = "decides"
		}
		val := "—"
		if g.Den > 0 {
			val = fmt.Sprintf("%d/%d = %.1f%%", g.Num, g.Den, 100*g.Value)
		}
		if !wide {
			out = append(out, fmt.Sprintf("  %-7s %-6s %-12s %-17s ≤%3.0f%% %s", g.Stratum, g.Window, strings.ToUpper(g.Status), val, 100*g.Threshold, role))
			if g.Reason != "" {
				out = append(out, "      "+g.Reason)
			}
			continue
		}
		line := fmt.Sprintf("  %-8s %-7s %-12s %-19s limit %4.0f%%  %-12s %s", g.Stratum, g.Window, strings.ToUpper(g.Status), val, 100*g.Threshold, role, g.Gate)
		if g.Reason != "" {
			line += " · " + g.Reason
		}
		out = append(out, line)
	}
	add("")
	add("PLAN GATES          your plan     no action")
	gate := func(name, a, b, meaning string) {
		if wide {
			out = append(out, fmt.Sprintf("  %-17s %-13s %-13s %s", name, strings.ToUpper(a), strings.ToUpper(b), meaning))
			return
		}
		out = append(out, fmt.Sprintf("  %-15s %-12s %-12s", name, strings.ToUpper(a), strings.ToUpper(b)), "      "+meaning)
	}
	gate("Service", plan.ServiceStatus, base.ServiceStatus, "every deciding gate and every gated pooled rate")
	gate("Backlog", plan.BacklogStatus, base.BacklogStatus, "the reading backlog does not grow")
	gate("Solvency", plan.SolvencyStatus, base.SolvencyStatus, "cash never below the floor")
	gate("Funding", plan.FundingStatus, base.FundingStatus, "liabilities covered at the end")
	gate("Case selection", cmp.CaseSelectionStatus, "", "no stratum of patients served less than with no action")
	return out
}

func t90PlanReport(r *eval.Report, name string) *eval.PlanReport {
	for i := range r.Plans {
		if r.Plans[i].Plan == name && len(r.Plans[i].PerSeed) == 1 && r.Plans[i].PerSeed[0].Measures != nil {
			return &r.Plans[i]
		}
	}
	return nil
}

func t90Integrity(plan, base *eval.PlanReport, cmp *eval.Comparison) []string {
	var out []string
	for _, p := range []*eval.PlanReport{plan, base} {
		if p != nil {
			out = append(out, p.Integrity...)
		}
	}
	if cmp != nil {
		out = append(out, cmp.Integrity...)
	}
	return out
}

// wrapText breaks s at spaces into lines of at most width runes, the
// continuation lines indented two more than s.
func wrapText(s string, width int) []string {
	indent := strings.Repeat(" ", len(s)-len(strings.TrimLeft(s, " ")))
	var out []string
	line := ""
	for _, w := range strings.Fields(s) {
		switch {
		case line == "":
			line = indent + w
		case utf8.RuneCountInString(line)+1+utf8.RuneCountInString(w) > width:
			out = append(out, line)
			line = indent + "  " + w
		default:
			line += " " + w
		}
	}
	return append(out, line)
}

func isoText(iso string) string { return strings.Replace(iso[:16], "T", " ", 1) }

func cap1(s string) string { return strings.ToUpper(s[:1]) + s[1:] }

func ratioText(r eval.Ratio) string {
	if !r.Defined {
		return "—"
	}
	return fmt.Sprintf("%.1f%%", 100*r.Value)
}

func minText(d eval.Dist) string {
	if !d.Defined {
		return "—"
	}
	if d.P90 < 60 {
		return fmt.Sprintf("%dm", d.P90)
	}
	return fmt.Sprintf("%dh %02dm", d.P90/60, d.P90%60)
}

// signedUSD formats cents as whole dollars with a sign: "−$1,706", "$0".
func signedUSD(c int64) string {
	sign := ""
	if c < 0 {
		sign, c = "−", -c
	}
	d := fmt.Sprint((c + 50) / 100)
	for i := len(d) - 3; i > 0; i -= 3 {
		d = d[:i] + "," + d[i:]
	}
	return sign + "$" + d
}
