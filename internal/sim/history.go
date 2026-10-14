package sim

import (
	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Daily history: one summary per calendar day, midnight to midnight, from
// the metric marks taken at each midnight (metrics.go) and the ledger's day
// books. The practice taken over at day 1 comes with its recent record;
// days before the start date are history (shown in trends, never scored),
// and the takeover day's summary includes its activity before the takeover.
// A day's summary is complete from the next midnight; today's is partial.

// recordDay notes what only exists at accounting close: the worklist left
// for the night.
func (st *state) recordDay() {
	for int64(len(st.met.unreadAtClose)) <= st.day {
		st.met.unreadAtClose = append(st.met.unreadAtClose, 0)
	}
	st.met.unreadAtClose[st.day] = len(st.dem.worklist)
}

// History returns the summaries of up to n completed days, oldest first.
func (w *World) History(n int) []api.DaySummary {
	st := w.st
	now := w.eng.Now()
	last := int64(len(st.met.marks)) - 2 // the latest day with a closing mark
	var out []api.DaySummary
	for d := max(0, last-int64(n)+1); d <= last; d++ {
		out = append(out, st.daySummary(d, now))
	}
	return out
}

// DaySummary returns day d's summary (today's is partial). ok is false for
// days not yet started.
func (w *World) DaySummary(d int64) (api.DaySummary, bool) {
	if d < 0 || d >= int64(len(w.st.met.marks)) {
		return api.DaySummary{}, false
	}
	return w.st.daySummary(d, w.eng.Now()), true
}

// OperationsDone reports whether today's operations are over: the
// practice is closed today, or operations closed after scheduled close.
func (w *World) OperationsDone() bool {
	st := w.st
	return st.cal.DayIndex(w.eng.Now()) == st.day && (st.plan.Hours == nil || st.opClosedAt != 0)
}

func (st *state) daySummary(d int64, now simtime.Time) api.DaySummary {
	if s, ok := st.met.done[d]; ok {
		return s
	}
	a, b, complete := st.dayBounds(d, now)
	ops := st.periodMetrics(&a, &b, complete)
	date := st.cal.Date(st.cal.StartOfDay(d))
	s := api.DaySummary{Date: date.String(), Weekday: date.Weekday().String(), BeforeStart: d < st.startDay,
		Complete: complete, AsOf: st.cal.ISO(b.at), Ops: ops,
		Orders: ops.Demand.Received, Booked: ops.Demand.Booked, Lost: ops.Demand.Lost, Cancelled: ops.Demand.Cancelled,
		WalkIns: ops.WalkIns.Arrived, Exams: ops.Exams, ReportsFinal: ops.Reporting.Final,
		WaitMedianMin: ops.Waits.ToExam.P50Sec / 60, TATMedianMin: ops.Reporting.FinalTAT.P50Sec / 60,
		OvertimeMin: ops.Workforce.OvertimeSec / 60}
	f := b.flow
	f.sub(&a.flow)
	s.Arrived, s.NoShows = f.arrived+f.walkIns, f.noShows
	if st.fin.opened && st.cal.DayIndex(st.fin.opening.at) == d {
		s.Takeover = st.cal.ISO(st.fin.opening.at)
	}
	if d < int64(len(st.met.unreadAtClose)) {
		s.UnreadAtClose = st.met.unreadAtClose[d]
	} else {
		s.UnreadAtClose = b.backlog
	}
	// Room time by modality (busy within open hours, so ≤ 100%).
	for _, m := range st.mods {
		md := api.ModalityDay{Modality: m}
		for _, r := range ops.Rooms {
			if r.Modality == m {
				md.Exams += r.Exams
				md.BusyMin += r.BusySec
				md.OpenMin += r.OpenSec
				md.StaffedMin += r.StaffedSec
			}
		}
		md.UtilOpenBP, md.UtilStaffedBP = ratio(md.BusyMin, md.OpenMin).BP(), ratio(md.BusyMin, md.StaffedMin).BP()
		md.BusyMin, md.OpenMin, md.StaffedMin = md.BusyMin/60, md.OpenMin/60, md.StaffedMin/60
		s.ByModality = append(s.ByModality, md)
	}

	led := &st.fin.led
	s.Money = st.period(date.Weekday().String()+" "+date.String(), d, d)
	s.NetRevenueCents, s.ExpenseCents = s.Money.NetRevenueCents, s.Money.OperatingExpenseCents
	s.OperatingProfitCents = s.Money.OperatingProfitCents
	s.CollectionsCents, s.DisbursementsCents = s.Money.CollectionsCents, s.Money.DisbursementsCents
	bal := led.balAtEnd(d)
	s.CashEndCents = bal[acctCash]
	s.ARCents, s.APCents, s.AccruedPayrollCents = bal[acctAR], -bal[acctAP], -bal[acctAccruedPayroll]
	s.ProFormaCashEndCents = s.CashEndCents
	if st.fin.opened && st.cal.StartOfDay(d+1) <= st.fin.opening.at {
		s.ProFormaCashEndCents += st.fin.opening.adjustment // as if the takeover capital had been there
	}
	// A completed day never changes once the takeover is behind it (the pro
	// forma figure depends on it).
	if complete && st.fin.opened {
		st.met.done[d] = s
	}
	return s
}

// balAtEnd is every account's balance at the end of day d.
func (l *ledger) balAtEnd(d int64) [numAccounts]int64 {
	var b [numAccounts]int64
	for k := int64(0); k <= d && k < int64(len(l.days)); k++ {
		for a := range numAccounts {
			b[a] += l.days[k].net[a]
		}
	}
	return b
}
