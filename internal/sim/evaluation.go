package sim

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// The evaluation window (owner decision, 2026-10-14): from the exact
// takeover time to the configured endpoint, measured against a baseline
// captured at the takeover instant. Calendar reporting (days, statements,
// the TUI) is separate and runs midnight to midnight.
//
// The baseline is taken at the end of the takeover transaction, after the
// opening adjustment: everything at that instant belongs to the inherited
// state, and the window is (takeover, end]. Inherited obligations stay in
// play: a claim recognized before takeover and paid after is window cash
// flow, not window revenue (its collection variance is a true-up of the
// history's revenue, reported separately); a study imaged before takeover
// and finalized after earns its revenue in the window (revenue is
// recognized at report final). Capital transfers are never operating
// performance.

type evalState struct {
	on, ended bool
	endAt     simtime.Time // 0: open-ended
	endDue    bool
	base      api.Baseline
	start     tally // the metric tally at the takeover
	end       tally // at the endpoint, once reached
	claims    int   // claims existing at the takeover (inherited: index < claims)

	// Window bookkeeping the ledger cannot know: collection variance and
	// collections on inherited claims, the baseline liabilities still to
	// be paid, reports finalized and claims adjudicated in the window.
	inhVariance, inhCollections int64
	inhPaid                     int64
	baseAP, basePayroll         int64 // still unpaid
	finals                      [maxMods]int
	adjudicated, denied         int
	minCash                     int64
}

// inWindow reports whether a posting or event now falls in the window.
func (st *state) inWindow(now simtime.Time) bool {
	ev := &st.eval
	return ev.on && (ev.endAt == 0 || now <= ev.endAt)
}

// captureBaseline runs at the end of the takeover transaction, after the
// opening adjustment.
func (st *state) captureBaseline(e *Engine) {
	now := e.Now()
	led := &st.fin.led
	ev := &st.eval
	b := &ev.base
	b.Time = st.cal.ISO(now)
	b.CashCents = led.bal[acctCash]
	b.ReceivablesCents = led.bal[acctAR]
	b.PayablesCents = -led.bal[acctAP]
	b.AccruedPayrollCents = -led.bal[acctAccruedPayroll]
	for a := range numAccounts {
		if !led.listed(a) {
			continue // a V0.2 account with no balance: the V0.1 chart's listing (§11)
		}
		info := accountInfo[a]
		bal := led.bal[a]
		switch info.kind {
		case kindLiability, kindEquity, kindRevenue:
			bal = -bal
		}
		b.Accounts = append(b.Accounts, api.Account{ID: info.id, Name: info.name, Type: acctKindNames[info.kind], BalanceCents: bal})
		if info.kind != kindAsset && info.kind != kindLiability {
			b.EquityCents -= led.bal[a]
		}
	}
	loads := map[claimState]*api.ClaimLoad{}
	for ci := range st.fin.claims {
		c := &st.fin.claims[ci]
		if c.state.closed() {
			continue
		}
		l, ok := loads[c.state]
		if !ok {
			l = &api.ClaimLoad{State: claimStateNames[c.state]}
			loads[c.state] = l
		}
		l.Count++
		l.Cents += c.expected - c.paid
		b.OpenClaims++
	}
	for s := claimPending; s < claimPaid; s++ {
		if l, ok := loads[s]; ok {
			b.Claims = append(b.Claims, *l)
		}
	}
	// Time-based accruals posted at the takeover day's midnight (salaries,
	// fixed costs) cover the whole day; the window bears the share after
	// the takeover, already in the baseline's liabilities.
	day := st.cal.DayIndex(now)
	share := st.cal.StartOfDay(day + 1).Sub(now)
	for i := len(led.entries) - 1; i >= 0 && st.cal.DayIndex(led.entries[i].at) == day; i-- {
		en := &led.entries[i]
		if en.key.kind != entSalary && en.key.kind != entFixed {
			continue
		}
		for _, ln := range en.lines {
			if accountInfo[ln.acct].kind == kindExpense {
				b.AllocatedAccrualCents += ln.amt * int64(share) / int64(simtime.Day)
			}
		}
	}
	for si := range st.dem.studies {
		s := &st.dem.studies[si]
		if s.state == studyFinal {
			continue
		}
		b.Unreported++
		b.OldestUnreportedS = max(b.OldestUnreportedS, secs(now.Sub(s.acquiredAt)))
		switch s.state {
		case studyUnread:
			b.Unread++
		case studyReading:
			b.Reading++
		}
	}
	for d := st.cal.DayIndex(now); d < int64(len(st.dem.days)); d++ {
		for _, ai := range st.dem.days[d].appts {
			if st.dem.appts[ai].state == apptBooked && st.apptStart(ai) > now {
				b.BookedAppointments++
			}
		}
	}
	b.RecallList = len(st.dem.recall)
	b.OnSite = len(st.dem.waiting)
	for i := range st.emps {
		if s := st.emps[i].status; s == available || s == busy {
			b.StaffOnDuty = append(b.StaffOnDuty, st.emps[i].cfg.Name)
		}
	}
	for r := range st.rooms {
		if s := st.rooms[r].state; s == roomIdle || s == roomOccupied {
			b.RoomsStaffed++
		}
	}
	b.LedgerEntries = len(led.entries)
	b.LogSeq = st.log.total
	raw, err := json.Marshal(b)
	if err != nil {
		panic(err)
	}
	sum := sha256.Sum256(raw)
	b.Digest = hex.EncodeToString(sum[:])

	ev.on = true
	ev.start = st.snapshot(now)
	ev.claims = len(st.fin.claims)
	ev.baseAP, ev.basePayroll = b.PayablesCents, b.AccruedPayrollCents
	ev.minCash = b.CashCents
	if st.cfg.EvaluationDays > 0 {
		ev.endAt = now.Add(simtime.Duration(st.cfg.EvaluationDays) * simtime.Day)
		e.Push(Event{At: ev.endAt, Class: ClassDayClose, Kind: kindEvalEnd})
	}
	led.openWindow(ev.endAt)
}

// evalEnd records the endpoint snapshot at the end of its transaction.
func (st *state) evalEnd(now simtime.Time) {
	ev := &st.eval
	ev.endDue = false
	ev.ended = true
	ev.end = st.snapshot(now)
}

// evalPaid apportions a disbursement to the baseline's liabilities first.
func (st *state) evalPaid(baseLeft *int64, amt int64) {
	if !st.inWindow(st.now) {
		return
	}
	p := min(amt, *baseLeft)
	*baseLeft -= p
	st.eval.inhPaid += p
}

// noteWindowCash tracks the lowest cash in the window.
func (st *state) noteWindowCash() {
	if st.inWindow(st.now) {
		st.eval.minCash = min(st.eval.minCash, st.fin.led.bal[acctCash])
	}
}

// Evaluation reports the window as of now (or its endpoint, once reached).
func (w *World) Evaluation() api.Evaluation {
	st := w.st
	ev := &st.eval
	if !ev.on {
		return api.Evaluation{}
	}
	now := w.eng.Now()
	out := api.Evaluation{Started: true, Baseline: ev.base, Ended: ev.ended}
	if ev.endAt != 0 {
		out.End = st.cal.ISO(ev.endAt)
	}
	end := ev.end
	if !ev.ended {
		end = st.snapshot(now)
	}
	out.AsOf = st.cal.ISO(end.at)
	out.Ops = st.periodMetrics(&ev.start, &end, ev.ended)
	out.Money = st.windowMoney()
	return out
}

func (st *state) windowMoney() api.WindowMoney {
	ev := &st.eval
	led := &st.fin.led
	b := led.win
	var m api.WindowMoney
	p := st.bookPeriod("Since takeover", b)
	p.From, p.To = ev.base.Time[:10], st.cal.Date(st.now).String()
	for k := range maxMods {
		if k < len(p.ByModality) {
			p.ByModality[k].ReportsFinal = ev.finals[k]
		}
		p.ReportsFinal += ev.finals[k]
	}
	p.Adjudicated, p.Denied = ev.adjudicated, ev.denied
	if p.Adjudicated > 0 {
		p.DenialRateBP = int64(p.Denied) * 10_000 / int64(p.Adjudicated)
	}
	m.Ledger = p

	m.InheritedVarianceCents = ev.inhVariance
	m.AllocatedAccrualCents = ev.base.AllocatedAccrualCents
	m.RevenueCents = p.GrossRevenueCents
	m.NetRevenueCents = p.GrossRevenueCents - (p.CollectionVarianceCents - ev.inhVariance)
	m.OperatingExpenseCents = p.OperatingExpenseCents + m.AllocatedAccrualCents
	m.OperatingProfitCents = m.NetRevenueCents - m.OperatingExpenseCents
	if m.NetRevenueCents > 0 {
		m.Margin = ratio(m.OperatingProfitCents, m.NetRevenueCents)
	}

	m.CollectionsCents, m.DisbursementsCents = b.receipts, b.paidOut
	m.InheritedCollectionsCents, m.InheritedPaidCents = ev.inhCollections, ev.inhPaid
	m.OperatingCashFlowCents = b.receipts - b.paidOut
	m.CapitalTransfersCents = -b.net[acctEquity]
	m.CashChangeCents = b.net[acctCash]
	m.ARChangeCents = b.net[acctAR]
	m.APChangeCents = -b.net[acctAP]
	m.PayrollChangeCents = -b.net[acctAccruedPayroll]
	m.SeverancePayableChangeCents = -b.net[acctSeverancePayable]
	m.CashEndCents = ev.base.CashCents + m.CashChangeCents
	m.MinCashCents = ev.minCash
	m.ShortfallCents = max(0, -ev.minCash)

	bridge := m.OperatingProfitCents - m.InheritedVarianceCents + m.AllocatedAccrualCents -
		m.ARChangeCents + m.APChangeCents + m.PayrollChangeCents + m.SeverancePayableChangeCents
	m.Reconciled = bridge == m.OperatingCashFlowCents &&
		m.OperatingCashFlowCents+m.CapitalTransfersCents == m.CashChangeCents
	return m
}
