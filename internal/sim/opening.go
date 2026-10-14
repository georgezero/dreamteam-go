package sim

import (
	"fmt"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// The opening boundary (SIMULATION_DESIGN §8.4, §8.8).
//
// The warm-up is the ordinary engine run from the epoch (cash 0) to the
// start time: every appointment, order, visit, study, claim, payable and
// payroll liability it creates is real state that carries on. At the start
// time one journal entry, the opening equity adjustment, sets cash to the
// scenario's opening figure against opening equity (owner capital in, or
// a distribution out). Nothing else is reset: receivables, payables,
// accrued payroll and open claims stay as the history left them, and the
// history's profit or loss stays in the books (retained earnings), so
// assets = liabilities + opening equity + earnings, always.

// openingSnap records the state at the opening adjustment.
type openingSnap struct {
	at                 simtime.Time
	target, before     int64 // opening cash wanted; cash just before
	adjustment         int64 // posted to cash (owner capital > 0, distribution < 0)
	ar, ap, accruedPay int64
	openClaims         int
	unread, reading    int
	booked             int // appointments booked from the start date on
	recall             int
	historyRevenue     int64 // net revenue over the history
	historyExpense     int64
	historyNetIncome   int64
}

// pushOpening queues the takeover event at the start time. Its handler only
// marks the adjustment due; the adjustment is posted at the very end of
// that instant's transaction, after every other event (day start with
// payroll or bills, remittances, completions) and after dispatch, so cash
// is exactly the target and the snapshot is the state play begins from.
func (st *state) pushOpening(e *Engine, at simtime.Time) {
	e.Push(Event{At: at, Class: ClassDayBoundary, Kind: kindOpening})
}

// openingAdjust posts the opening equity adjustment and records the
// opening state.
func (st *state) openingAdjust(e *Engine) {
	led := &st.fin.led
	now := e.Now()
	s := openingSnap{at: now, target: st.cfg.OpeningCashCents, before: led.bal[acctCash]}
	s.adjustment = s.target - s.before
	if s.adjustment != 0 {
		led.mustPost(now, entryKey{kind: entOpening}, line{acctCash, -1, s.adjustment, -1}, line{acctEquity, -1, -s.adjustment, -1})
	}
	s.ar, s.ap, s.accruedPay = led.bal[acctAR], -led.bal[acctAP], -led.bal[acctAccruedPayroll]
	for ci := range st.fin.claims {
		if !st.fin.claims[ci].state.closed() {
			s.openClaims++
		}
	}
	for _, si := range st.dem.worklist {
		if st.dem.studies[si].state == studyReading {
			s.reading++
		} else {
			s.unread++
		}
	}
	for d := st.startDay; d < int64(len(st.dem.days)); d++ {
		for _, ai := range st.dem.days[d].appts {
			if st.dem.appts[ai].state == apptBooked {
				s.booked++
			}
		}
	}
	s.recall = len(st.dem.recall)
	h := led.sumDays(0, st.startDay-1)
	s.historyRevenue, s.historyExpense = netRevenue(&h), opex(&h)
	s.historyNetIncome = s.historyRevenue - s.historyExpense
	st.fin.opening, st.fin.opened = s, true
	st.fin.minCash, st.fin.minCashAt = led.bal[acctCash], now
	text := fmt.Sprintf("Opening balance: cash %s", money(s.target))
	switch {
	case s.adjustment > 0:
		text += fmt.Sprintf(" (owner capital %s added at takeover)", money(s.adjustment))
	case s.adjustment < 0:
		text += fmt.Sprintf(" (owner distribution %s at takeover)", money(-s.adjustment))
	}
	if st.cfg.WarmupDays > 0 {
		text += fmt.Sprintf(" · %d days of history: net revenue %s, operating result %s · A/R %s, %d open claims",
			st.cfg.WarmupDays, money(s.historyRevenue), money(s.historyNetIncome), money(s.ar), s.openClaims)
	}
	st.emit(now, api.Info, "finance", text)
	// Play begins: alerts watch from here, and the evaluation window's
	// baseline is this state.
	st.alerts.on = true
	st.captureBaseline(e)
}

// noteCash tracks the financing shortfall after a cash movement, from the
// start time on: the lowest balance (the cash alerts in alerts.go say when
// it goes below zero). V0.1 has no credit line; negative cash is an
// unfunded shortfall, tracked and shown, never free borrowing.
func (st *state) noteCash(now simtime.Time) {
	if !st.fin.opened {
		return
	}
	c := st.fin.led.bal[acctCash]
	if c < st.fin.minCash {
		st.fin.minCash, st.fin.minCashAt = c, now
	}
	st.noteWindowCash()
}

// netRevenue and opex read a day-book sum.
func netRevenue(b *dayBook) int64 { return -b.net[acctRevenue] - b.net[acctVariance] }

func opex(b *dayBook) int64 {
	var x int64
	for a := range numAccounts {
		if accountInfo[a].kind == kindExpense {
			x += b.net[a]
		}
	}
	return x
}
