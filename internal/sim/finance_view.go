package sim

import (
	"fmt"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Observable projections of the ledger and claims. Only facts the practice
// has: balances, posted entries, claim states and payments received. A
// claim's outcome, adjudication date and the charge-lag draw never appear
// before they happen.

const (
	openClaimsCap = 200
	journalCap    = 60
	cashHistory   = 30
)

var acctKindNames = [...]string{kindAsset: "asset", kindLiability: "liability", kindEquity: "equity",
	kindRevenue: "revenue", kindContra: "contra-revenue", kindExpense: "expense"}

func (st *state) observeFinance(o *api.Observation, now simtime.Time) {
	led := &st.fin.led
	day := st.cal.DayIndex(now)
	date := st.cal.Date(now)
	f := api.Finance{
		CashCents:           led.bal[acctCash],
		ARCents:             led.bal[acctAR],
		APCents:             -led.bal[acctAP],
		AccruedPayrollCents: -led.bal[acctAccruedPayroll],
		JournalTotal:        len(led.entries),
	}
	if day < int64(len(led.days)) {
		f.CashChangeTodayCents = led.days[day].net[acctCash]
	}
	f.Today = st.period("Today", day, day)
	f.MonthToDate = st.period("Month to date", day-int64(date.Day-1), day)
	f.Trailing30 = st.period("Trailing 30 days", day-29, day)
	if day >= st.startDay {
		f.SinceStart = st.period("Since day 1", st.startDay, day)
	} else {
		f.SinceStart = api.FinancePeriod{Label: "Since day 1"}
	}
	st.observeClaims(&f, now, day)
	if n := f.Trailing30.NetRevenueCents; n > 0 && f.Trailing30.Days > 0 {
		f.Claims.DaysInAR = f.ARCents * f.Trailing30.Days / n
	} else {
		f.Claims.DaysInAR = -1
	}

	from := max(0, day-cashHistory+1)
	cash := led.cashAtEnd(from - 1)
	for d := from; d <= day; d++ {
		if d < int64(len(led.days)) {
			cash += led.days[d].net[acctCash]
		}
		f.CashHistory = append(f.CashHistory, api.DayAmount{Date: st.cal.Date(st.cal.StartOfDay(d)).String(), Cents: cash})
	}
	f.NextPayroll = api.Obligation{Date: st.cal.Date(st.cal.StartOfDay(st.nextPayday(day))).String(), Cents: f.AccruedPayrollCents}
	f.NextPayables = api.Obligation{Date: date.AddDays(int64(date.DaysInMonth() - date.Day + 1)).String(), Cents: f.APCents}

	for a := range numAccounts {
		info := accountInfo[a]
		b := led.bal[a]
		switch info.kind {
		case kindLiability, kindEquity, kindRevenue:
			b = -b
		}
		f.Accounts = append(f.Accounts, api.Account{ID: info.id, Name: info.name, Type: acctKindNames[info.kind], BalanceCents: b})
	}
	for i := len(led.entries) - 1; i >= 0 && len(f.Journal) < journalCap; i-- {
		f.Journal = append(f.Journal, st.journalEntry(i))
	}
	o.Finance = f
}

// period builds the statement for days [from, to].
func (st *state) period(label string, from, to int64) api.FinancePeriod {
	from = max(from, 0)
	led := &st.fin.led
	b := led.sumDays(from, to)
	p := api.FinancePeriod{Label: label, From: st.cal.Date(st.cal.StartOfDay(from)).String(),
		To: st.cal.Date(st.cal.StartOfDay(to)).String(), Days: to - from + 1}
	n := &b.net
	p.GrossRevenueCents = -n[acctRevenue]
	p.CollectionVarianceCents = n[acctVariance]
	p.NetRevenueCents = p.GrossRevenueCents - p.CollectionVarianceCents
	p.WagesCents, p.BenefitsCents = n[acctWages], n[acctBenefits]
	p.FacilityCents, p.EquipmentCents = n[acctFacility], n[acctEquipment]
	p.SuppliesCents, p.ITCents, p.BillingCents = n[acctSupplies], n[acctIT], n[acctBilling]
	for a := range numAccounts {
		if accountInfo[a].kind == kindExpense {
			p.OperatingExpenseCents += n[a]
		}
	}
	p.LaborCents = p.WagesCents + p.BenefitsCents
	p.FixedCents = b.fixed
	p.VariableCents = p.OperatingExpenseCents - p.LaborCents - p.FixedCents
	p.OperatingProfitCents = p.NetRevenueCents - p.OperatingExpenseCents
	if p.NetRevenueCents > 0 {
		p.MarginBP, p.MarginDefined = p.OperatingProfitCents*10_000/p.NetRevenueCents, true
	}
	p.CollectionsCents, p.DisbursementsCents = b.receipts, b.paidOut
	p.NetCashFlowCents = b.receipts - b.paidOut

	var finals [maxMods]int
	for d := from; d <= to && d < int64(len(st.fin.days)); d++ {
		fd := &st.fin.days[d]
		for k := range maxMods {
			finals[k] += fd.finals[k]
		}
		p.Adjudicated += fd.adjudicated
		p.Denied += fd.denied
	}
	if p.Adjudicated > 0 {
		p.DenialRateBP = int64(p.Denied) * 10_000 / int64(p.Adjudicated)
	}
	for k, m := range st.mods {
		mm := api.ModalityMoney{Modality: m, ReportsFinal: finals[k],
			NetRevenueCents: -b.byMod[acctRevenue][k] - b.byMod[acctVariance][k]}
		for a := range numAccounts {
			if accountInfo[a].kind == kindExpense {
				mm.DirectCostCents += b.byMod[a][k]
			}
		}
		p.ReportsFinal += finals[k]
		p.ByModality = append(p.ByModality, mm)
	}
	return p
}

var agingLabels = [...]string{"0–30 days", "31–60 days", "61–90 days", "over 90 days"}

func (st *state) observeClaims(f *api.Finance, now simtime.Time, day int64) {
	c := &f.Claims
	c.Total = len(st.fin.claims)
	c.Aging = make([]api.AgingBucket, len(agingLabels))
	for i, l := range agingLabels {
		c.Aging[i].Label = l
	}
	for ci := range st.fin.claims {
		cl := &st.fin.claims[ci]
		switch cl.state {
		case claimPending:
			c.Pending++
		case claimSubmitted:
			c.Submitted++
		case claimDenied, claimResubmitted:
			c.Denied++
		case claimPaid:
			c.Paid++
		case claimShortPaid:
			c.ShortPaid++
		case claimWrittenOff:
			c.WrittenOff++
		}
		if cl.state.closed() {
			continue
		}
		c.Open++
		age := day - st.cal.DayIndex(cl.finalAt)
		b := min(int(max(0, age-1)/30), len(agingLabels)-1)
		if age <= 30 {
			b = 0
		}
		c.Aging[b].Count++
		c.Aging[b].Cents += cl.expected - cl.paid
		if len(c.OpenClaims) < openClaimsCap {
			c.OpenClaims = append(c.OpenClaims, st.claimItem(ci, day))
		}
	}
}

func (st *state) claimItem(ci int, day int64) api.ClaimItem {
	cl := &st.fin.claims[ci]
	s := &st.dem.studies[cl.study]
	o := &st.dem.orders[cl.order]
	x := &st.cfg.Catalog[s.exam]
	it := api.ClaimItem{Claim: claimID(ci), Study: studyID(cl.study), Patient: patientID(o.patient),
		Payer: st.cfg.Finance.Payers[cl.payer].Name, Modality: x.Modality, ExamName: x.Name,
		State: claimStateNames[cl.state], Final: st.cal.ISO(cl.finalAt), AgeDays: day - st.cal.DayIndex(cl.finalAt),
		ExpectedCents: cl.expected, PaidCents: cl.paid}
	if cl.state != claimPending || cl.denied {
		it.Submitted = st.cal.ISO(cl.submittedAt)
	}
	if !cl.state.closed() {
		it.BalanceCents = cl.expected - cl.paid
	}
	return it
}

// claimSteps describes a claim in a patient's journey: what has happened,
// never what will.
func (st *state) claimSteps(ci int) []api.JourneyStep {
	if ci < 0 {
		return nil
	}
	cl := &st.fin.claims[ci]
	p := &st.cfg.Finance.Payers[cl.payer]
	iso := st.cal.ISO
	out := []api.JourneyStep{{Time: iso(cl.finalAt), Label: "Revenue earned", Done: true,
		Detail: fmt.Sprintf("%s expected net (allowed %s × NCR %s, %s)", money(cl.expected), money(cl.allowed),
			ppmDecimal(p.NCRPPM()), p.Name)}}
	if cl.state == claimPending {
		return append(out, api.JourneyStep{Label: "Claim " + claimID(ci), Detail: "pending submission (charge lag)", Done: true})
	}
	what := "submitted to " + p.Name
	if p.SelfPay {
		what = "statement sent to the patient"
	}
	out = append(out, api.JourneyStep{Time: iso(cl.submittedAt), Label: "Claim " + claimID(ci), Detail: what, Done: true})
	if cl.denied {
		out = append(out, api.JourneyStep{Time: iso(cl.deniedAt), Label: "Denied", Detail: "reworking", Done: true})
		if cl.state != claimDenied {
			out = append(out, api.JourneyStep{Time: iso(cl.resubmittedAt), Label: "Resubmitted", Done: true})
		}
	}
	switch cl.state {
	case claimSubmitted, claimResubmitted:
		out = append(out, api.JourneyStep{Label: "Payment", Detail: "awaiting remittance"})
	case claimPaid:
		out = append(out, api.JourneyStep{Time: iso(cl.closedAt), Label: "Paid", Done: true,
			Detail: fmt.Sprintf("%s received · collection variance %s", money(cl.paid), money(cl.expected-cl.paid))})
	case claimShortPaid:
		out = append(out, api.JourneyStep{Time: iso(cl.closedAt), Label: "Short paid", Done: true,
			Detail: fmt.Sprintf("%s of %s allowed · collection variance %s", money(cl.paid), money(cl.allowed), money(cl.expected-cl.paid))})
	case claimWrittenOff:
		out = append(out, api.JourneyStep{Time: iso(cl.closedAt), Label: "Written off", Done: true,
			Detail: fmt.Sprintf("%s expected · nothing collected", money(cl.expected))})
	}
	return out
}

func ppmDecimal(v int64) string { return fmt.Sprintf("%d.%03d", v/1_000_000, (v%1_000_000+500)/1000) }

// journalEntry renders entry i with a memo derived from its key.
func (st *state) journalEntry(i int) api.JournalEntry {
	e := &st.fin.led.entries[i]
	je := api.JournalEntry{ID: journalID(i), Time: st.cal.ISO(e.at)}
	je.Memo, je.Ref = st.memo(e.key)
	for pass := range 2 { // debits first, then credits
		for _, ln := range e.lines {
			if (ln.amt > 0) != (pass == 0) {
				continue
			}
			jl := api.JournalLine{Account: accountInfo[ln.acct].name}
			if ln.mod >= 0 {
				jl.Modality = st.mods[ln.mod]
			}
			if ln.amt > 0 {
				jl.DebitCents = ln.amt
			} else {
				jl.CreditCents = -ln.amt
			}
			je.Lines = append(je.Lines, jl)
		}
	}
	return je
}

func (st *state) memo(k entryKey) (memo, ref string) {
	dayName := func(d int64) string { return st.cal.Date(st.cal.StartOfDay(d)).String() }
	switch k.kind {
	case entOpening:
		return "Opening cash", ""
	case entRevenue:
		s := &st.dem.studies[k.a]
		x := &st.cfg.Catalog[s.exam]
		return fmt.Sprintf("Revenue earned: report final %s %s (%s)", x.Modality, x.Name,
			st.cfg.Finance.Payers[st.dem.orders[s.order].payer].Name), studyID(int(k.a))
	case entStudyCost:
		x := &st.cfg.Catalog[st.dem.studies[k.a].exam]
		return fmt.Sprintf("Supplies and PACS: %s %s", x.Modality, x.Name), studyID(int(k.a))
	case entPayment:
		return fmt.Sprintf("Payment received (%s) and billing fee", st.cfg.Finance.Payers[st.fin.claims[k.a].payer].Name), claimID(int(k.a))
	case entSettle:
		return "Claim settled: collection variance", claimID(int(k.a))
	case entWages:
		return "Wages accrued " + dayName(k.b), st.cfg.Employees[k.a].ID
	case entSalary:
		return "Salary accrued " + dayName(k.b), st.cfg.Employees[k.a].ID
	case entFixed:
		return "Fixed costs accrued " + dayName(k.a), ""
	case entPayroll:
		return "Payroll paid", ""
	case entPayAP:
		return "Accounts payable paid", ""
	case entReversal:
		return "Reversal of " + journalID(int(k.a)), journalID(int(k.a))
	}
	return "", ""
}
