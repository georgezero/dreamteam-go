package sim_test

import (
	"math/rand/v2"
	"reflect"
	"strings"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/rng"
	"github.com/georgezero/dreamteam-go/internal/scenario"
	"github.com/georgezero/dreamteam-go/internal/sim"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Step 12 metric definitions, on hand-built micro practices with fixed
// durations (20 min exams, 1 min documentation, 3 min self-registration,
// patients on time) where every figure can be worked out by hand.

func room(p api.PeriodMetrics, id string) api.RoomMetrics {
	for _, r := range p.Rooms {
		if r.Room == id {
			return r
		}
	}
	panic("no room " + id)
}

func person(p api.PeriodMetrics, name string) api.StaffMetrics {
	for _, s := range p.Staff {
		if s.Name == name {
			return s
		}
	}
	panic("no employee " + name)
}

func day(t *testing.T, w *sim.World, d int64) api.DaySummary {
	t.Helper()
	s, ok := w.DaySummary(d)
	if !ok {
		t.Fatalf("no summary for day %d", d)
	}
	return s
}

// clip is the overlap of [a, b) with [lo, hi) in seconds.
func clip(a, b, lo, hi simtime.Time) int64 {
	return int64(max(0, min(b, hi).Sub(max(a, lo))))
}

// micro-overtime: one XR technologist 07:30–09:00, open 08:00–09:00, four
// patients at 08:00, 08:15, 08:30, 08:45. Exams 08:03–08:23, 08:27–08:47,
// 08:54–09:14 and 09:15–09:35; Tess goes home at 09:36. The CT room is
// staffed all hour and never used.
func TestMetricsRoomAndStaffDenominators(t *testing.T) {
	w, d0 := micro(t, "micro-overtime.json")
	if err := runTo(w, d0.Add(simtime.Day)); err != nil {
		t.Fatal(err)
	}
	ops := day(t, w, 0).Ops
	xr, ct := room(ops, "R1"), room(ops, "R2")
	open := at(d0, "08:00")
	close := at(d0, "09:00")
	var busy, after int64
	var toExam []int64
	for _, v := range w.Visits() {
		busy += clip(v.ExamStart, v.ExamEnd, open, close)
		after += int64(v.ExamEnd.Sub(v.ExamStart)) - clip(v.ExamStart, v.ExamEnd, open, close)
		toExam = append(toExam, int64(v.ExamStart.Sub(v.Arrived)))
	}
	if xr.OpenSec != 3600 || xr.StaffedSec != 3600 || xr.BusySec != 46*60 || xr.AfterHoursBusySec != 34*60 ||
		busy != xr.BusySec || after != xr.AfterHoursBusySec || xr.Exams != 4 {
		t.Fatalf("XR %+v (independent busy %d, after hours %d)", xr, busy, after)
	}
	if xr.UtilOpen != (api.Ratio{Num: 2760, Den: 3600}) || xr.UtilStaffed != (api.Ratio{Num: 2760, Den: 3600}) ||
		xr.Coverage != (api.Ratio{Num: 3600, Den: 3600}) {
		t.Fatalf("XR ratios %+v", xr)
	}
	// An unused but staffed room is 0% utilized, not undefined.
	if ct.OpenSec != 3600 || ct.StaffedSec != 3600 || ct.BusySec != 0 || !ct.UtilStaffed.Defined() || ct.UtilStaffed.BP() != 0 {
		t.Fatalf("CT %+v", ct)
	}
	// Tess: on duty 07:30–09:36 (2h06m), 36m of it overtime; busy with four
	// registrations (3m), four exams (20m) and four documentations (1m).
	tess, cody := person(ops, "Tess"), person(ops, "Cody")
	if tess.OnDutySec != 126*60 || tess.OvertimeSec != 36*60 || tess.BusySec != 4*(3+20+1)*60 || tess.BreakSec != 0 ||
		tess.Util != (api.Ratio{Num: 96 * 60, Den: 126 * 60}) {
		t.Fatalf("Tess %+v", tess)
	}
	if cody.OnDutySec != 90*60 || cody.BusySec != 0 || cody.Util.BP() != 0 {
		t.Fatalf("Cody %+v", cody)
	}
	// Waits: arrival to exam start, nearest-rank percentiles.
	if ops.Waits.ToExam.Count != 4 || ops.Waits.ToExam.P50Sec != nearestRank(toExam, 50) ||
		ops.Waits.ToExam.P90Sec != nearestRank(toExam, 90) || ops.Waits.ToExam.MaxSec != 30*60 {
		t.Fatalf("waits %+v from %v", ops.Waits, toExam)
	}
	if ops.Exams != 4 || ops.AfterHoursExams != 1 || ops.Workforce.OvertimeSec != 36*60 {
		t.Fatalf("exams %d after hours %d overtime %d", ops.Exams, ops.AfterHoursExams, ops.Workforce.OvertimeSec)
	}
}

// nearestRank is an independent nearest-rank percentile.
func nearestRank(xs []int64, p int) int64 {
	s := append([]int64(nil), xs...)
	for i := range s {
		for j := i + 1; j < len(s); j++ {
			if s[j] < s[i] {
				s[i], s[j] = s[j], s[i]
			}
		}
	}
	k := (len(s)*p + 99) / 100
	return s[k-1]
}

// One technologist covers XR and CT: each room counts her as staffing it
// for the full hour, but her own time is counted once, and the rooms' busy
// time adds up to her exam time.
func TestMetricsSharedTechnologist(t *testing.T) {
	w, d0 := micro(t, "micro-shared.json")
	if err := runTo(w, d0.Add(simtime.Day)); err != nil {
		t.Fatal(err)
	}
	ops := day(t, w, 0).Ops
	xr, ct := room(ops, "R1"), room(ops, "R2")
	tess := person(ops, "Tess")
	var exams int64
	for _, v := range w.Visits() {
		exams += int64(v.ExamEnd.Sub(v.ExamStart))
	}
	if xr.StaffedSec != 3600 || ct.StaffedSec != 3600 {
		t.Fatalf("shared coverage: XR staffed %d, CT staffed %d", xr.StaffedSec, ct.StaffedSec)
	}
	if xr.BusySec+xr.AfterHoursBusySec+ct.BusySec+ct.AfterHoursBusySec != exams || tess.BusySec < exams ||
		tess.BusySec > tess.OnDutySec || xr.UtilStaffed.BP() > 10_000 || ct.UtilStaffed.BP() > 10_000 {
		t.Fatalf("XR %+v CT %+v Tess %+v, exam time %d", xr, ct, tess, exams)
	}
	// Her busy time is the sum of her tasks (4 × (3 + 20 + 1) minutes),
	// never double-counted across the two rooms.
	if tess.BusySec != 4*24*60 {
		t.Fatalf("Tess busy %d", tess.BusySec)
	}
}

// Staffed time is open time with a technologist on duty: her breaks leave
// the room open but unstaffed.
func TestMetricsStaffedVersusOpen(t *testing.T) {
	w, d0 := micro(t, "micro-breaks.json")
	if err := runTo(w, d0.Add(simtime.Day)); err != nil {
		t.Fatal(err)
	}
	ops := day(t, w, 0).Ops
	xr := room(ops, "R1")
	tess := person(ops, "Tess")
	if tess.BreakSec <= 0 || xr.OpenSec != 8*3600 || xr.StaffedSec != xr.OpenSec-tess.BreakSec ||
		xr.Coverage.BP() >= 10_000 || xr.UtilStaffed.BP() < xr.UtilOpen.BP() {
		t.Fatalf("XR %+v, Tess breaks %d", xr, tess.BreakSec)
	}
}

// Radiologist reading utilization: reads inside site-reading blocks over
// block time; reads past a block's end are counted apart.
func TestMetricsReadingBlocks(t *testing.T) {
	w, d0 := micro(t, "micro-reading.json")
	if err := runTo(w, d0.Add(simtime.Day)); err != nil {
		t.Fatal(err)
	}
	rae := person(day(t, w, 0).Ops, "Dr. Rae")
	block := at(d0, "09:30")
	end := at(d0, "11:00")
	var in, out int64
	for _, s := range w.Studies() {
		if s.State != "final" {
			continue
		}
		in += clip(s.Claimed, s.Final, block, end)
		out += int64(s.Final.Sub(s.Claimed)) - clip(s.Claimed, s.Final, block, end)
	}
	if rae.BlockSec != 90*60 || rae.ReadInBlockSec != in || rae.ReadOutsideSec != out || in == 0 ||
		rae.ReadUtil != (api.Ratio{Num: in, Den: 90 * 60}) || rae.BusySec != in+out {
		t.Fatalf("Dr. Rae %+v; independent reads in block %d, outside %d", rae, in, out)
	}
	// TAT over the studies imaged that day, censored at midnight for any
	// still unread, with nearest-rank percentiles.
	rp := day(t, w, 0).Ops.Reporting
	var tats []int64
	for _, s := range w.Studies() {
		end := d0.Add(simtime.Day)
		if s.State == "final" {
			end = s.Final
		}
		tats = append(tats, int64(end.Sub(s.Acquired)))
	}
	if rp.Acquired != len(tats) || rp.TAT.P50Sec != nearestRank(tats, 50) || rp.TAT.P90Sec != nearestRank(tats, 90) ||
		rp.Final != len(tats)-rp.Unfinished {
		t.Fatalf("reporting %+v from %v", rp, tats)
	}
}

// Empty denominators are undefined, not zero: a closed day has no open
// hours, no waits and no reports.
func TestMetricsEmptyDenominators(t *testing.T) {
	w, d0 := micro(t, "micro-overtime.json")
	if err := runTo(w, d0.Add(2*simtime.Day)); err != nil {
		t.Fatal(err)
	}
	ops := day(t, w, 1).Ops // Wednesday: closed
	xr := room(ops, "R1")
	if xr.OpenSec != 0 || xr.UtilOpen.Defined() || xr.UtilStaffed.Defined() || xr.Coverage.Defined() ||
		ops.Waits.ToExam != (api.Dist{}) || ops.Demand.LostRate.Defined() || ops.Appointments.NoShowRate.Defined() ||
		ops.WalkIns.Acceptance.Defined() || person(ops, "Tess").Util.Defined() || ops.Reporting.TAT.Count != 0 {
		t.Fatalf("closed day %+v", ops)
	}
}

// Today's figures are partial (so far) and become the complete day at
// midnight; the figures so far are exact for the time elapsed.
func TestMetricsPartialDay(t *testing.T) {
	w, d0 := micro(t, "micro-overtime.json")
	if err := runTo(w, at(d0, "08:30")); err != nil {
		t.Fatal(err)
	}
	o := w.Observe(0)
	today := day(t, w, 0)
	xr := room(o.Today, "R1")
	if o.Today.Complete || today.Complete || !reflect.DeepEqual(o.Today, today.Ops) || today.AsOf != "2026-10-13T08:30:00" ||
		xr.OpenSec != 1800 || xr.StaffedSec != 1800 || xr.BusySec != 20*60+3*60 {
		t.Fatalf("at 08:30: %+v (XR %+v)", today, xr)
	}
	if len(w.History(5)) != 0 {
		t.Fatal("today listed as a completed day")
	}
	if err := runTo(w, d0.Add(simtime.Day)); err != nil {
		t.Fatal(err)
	}
	h := w.History(5)
	if len(h) != 1 || !h[0].Complete || room(h[0].Ops, "R1").OpenSec != 3600 || h[0].Ops.To != "2026-10-14T00:00:00" {
		t.Fatalf("after midnight: %+v", h)
	}
}

// Calendar days run midnight to midnight: work 11:45–23:30 (the latest a
// 12-hour cap allows before accounting close) belongs to the
// day it was done, the day read at 23:59:59 (partial) has the same figures
// as the completed day, and the next day starts from zero.
func TestMetricsMidnightBoundary(t *testing.T) {
	w, d0 := micro(t, "micro-late.json")
	if err := runTo(w, d0.Add(simtime.Day-1)); err != nil {
		t.Fatal(err)
	}
	late := day(t, w, 0)
	if err := runTo(w, d0.Add(simtime.Day+simtime.Hour)); err != nil {
		t.Fatal(err)
	}
	done, next := day(t, w, 0), day(t, w, 1)
	tess := person(done.Ops, "Tess")
	if late.Complete || !done.Complete || done.Ops.To != "2026-10-14T00:00:00" || next.Ops.From != "2026-10-14T00:00:00" ||
		tess.OnDutySec != int64(11*3600+45*60) || room(done.Ops, "R1").BusySec != 20*60 || done.Ops.Exams != 1 {
		t.Fatalf("day 0 %+v", done.Ops)
	}
	a, b := late.Ops, done.Ops
	// An unread study's TAT is censored at the period end: it ages to
	// midnight, one second more. Everything else is unchanged.
	if a.Reporting.Unfinished != 1 || b.Reporting.TAT.MaxSec != a.Reporting.TAT.MaxSec+1 {
		t.Fatalf("censored TAT at 23:59:59 %+v, at midnight %+v", a.Reporting.TAT, b.Reporting.TAT)
	}
	a.To, a.Complete, b.To, b.Complete = "", false, "", false
	a.Reporting, b.Reporting = api.ReportingMetrics{}, api.ReportingMetrics{}
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("figures changed between 23:59:59 and midnight:\n%+v\n%+v", a, b)
	}
	if n := person(next.Ops, "Tess"); n.OnDutySec != 0 || next.Ops.Exams != 0 || room(next.Ops, "R1").OpenSec != 0 {
		t.Fatalf("day 1 starts with %+v", n)
	}
}

// A 06:00 takeover: the takeover day's midnight accruals (salaries, fixed
// costs) are posted before it but cover the whole day; the window bears
// the 18 hours after takeover, line by line, rounded down.
func TestEvaluationAllocatesTakeoverDayAccruals(t *testing.T) {
	cfg := miniConfig(t)
	w := sim.New(cfg, rng.SeedFromInt(2))
	start := w.StartTime()
	w.Engine().RunUntil(start.Add(simtime.Hour), nil)
	if simtime.ClockOf(start) != 6*simtime.Hour {
		t.Fatalf("mini starts at %s", w.Calendar().Clock(start))
	}
	var want int64
	midnight := w.Calendar().StartOfDay(w.Calendar().DayIndex(start))
	for _, e := range w.Journal() {
		if e.At != midnight || (e.Kind != "salary" && e.Kind != "fixed") {
			continue
		}
		for _, l := range e.Lines {
			if l.Amt > 0 {
				want += l.Amt * 18 / 24
			}
		}
	}
	ev := w.Evaluation()
	if want == 0 || ev.Baseline.AllocatedAccrualCents != want || ev.Money.AllocatedAccrualCents != want ||
		ev.Money.OperatingExpenseCents != ev.Money.Ledger.OperatingExpenseCents+want || !ev.Money.Reconciled {
		t.Fatalf("allocated %d (money %+v), want %d", ev.Baseline.AllocatedAccrualCents, ev.Money, want)
	}
}

// Days, today and the window add up: over whole days the daily summaries
// sum to the since-day-1 counters and statements, cash rolls forward day
// by day, and the window (from the takeover) reconciles to the cent.
func TestDailySummaryReconciliation(t *testing.T) {
	cfg := miniConfig(t)
	w := run(t, cfg, 3, 10)
	o := w.Observe(0)
	hist := w.History(10)
	if len(hist) != 10 {
		t.Fatalf("%d days", len(hist))
	}
	today, _ := w.DaySummary(w.Calendar().DayIndex(w.Engine().Now()))
	if today.Complete {
		t.Fatal("today is complete")
	}
	hist = append(hist, today) // midnight accruals and payments are today's
	var exams, lost, received, finals int
	var rev, exp, coll, disb int64
	cash := int64(0)
	for i, d := range hist {
		exams += d.Ops.Exams
		lost += d.Ops.Demand.Lost
		received += d.Ops.Demand.Received
		finals += d.ReportsFinal
		rev += d.NetRevenueCents
		exp += d.ExpenseCents
		coll += d.CollectionsCents
		disb += d.DisbursementsCents
		want := cash + d.CollectionsCents - d.DisbursementsCents
		if i == 0 {
			want += o.Finance.Opening.AdjustmentCents // the takeover capital, on day 1
		}
		if d.CashEndCents != want || d.Complete == (d.Date == today.Date) {
			t.Fatalf("%s: cash end %d, rolled forward %d", d.Date, d.CashEndCents, want)
		}
		cash = d.CashEndCents
		if d.Exams != d.Ops.Exams || d.NetRevenueCents != d.Money.NetRevenueCents {
			t.Fatalf("%s: headline and detail disagree", d.Date)
		}
		for _, r := range d.Ops.Rooms {
			if r.BusySec > r.StaffedSec || r.StaffedSec > r.OpenSec {
				t.Fatalf("%s %s: busy %d > staffed %d > open %d", d.Date, r.Room, r.BusySec, r.StaffedSec, r.OpenSec)
			}
		}
	}
	since := o.Finance.SinceStart
	var dRecv, dLost int
	for _, m := range o.Demand.Modalities {
		dRecv += m.SinceStart.Received
		dLost += m.SinceStart.Lost
	}
	if exams != o.Flow.SinceStart.ExamsDone || lost != dLost || received != dRecv || finals != since.ReportsFinal ||
		rev != since.NetRevenueCents || exp != since.OperatingExpenseCents || coll != since.CollectionsCents ||
		disb != since.DisbursementsCents || cash != o.Finance.CashCents {
		t.Fatalf("days sum to exams %d lost %d received %d finals %d revenue %d expenses %d cash %d; since day 1 %d %d %d %d %d %d %d",
			exams, lost, received, finals, rev, exp, cash, o.Flow.SinceStart.ExamsDone, dLost, dRecv, since.ReportsFinal,
			since.NetRevenueCents, since.OperatingExpenseCents, o.Finance.CashCents)
	}
	ev := w.Evaluation()
	if !ev.Started || !ev.Money.Reconciled || ev.Money.CashEndCents != o.Finance.CashCents {
		t.Fatalf("window %+v", ev.Money)
	}
}

// The mini practice starts at 00:00 with no history: its window covers
// whole days, so the window's operations equal the sum of the days.
func TestWindowEqualsDaysFromMidnightTakeover(t *testing.T) {
	cfg := *miniConfig(t)
	cfg.StartClock = 0
	w := run(t, &cfg, 4, 6)
	ev := w.Evaluation()
	var exams, recv int
	var busy int64
	for _, d := range w.History(6) {
		exams += d.Ops.Exams
		recv += d.Ops.Demand.Received
		for _, r := range d.Ops.Rooms {
			busy += r.BusySec
		}
	}
	var wbusy int64
	for _, r := range ev.Ops.Rooms {
		wbusy += r.BusySec
	}
	if ev.Ops.Exams != exams || ev.Ops.Demand.Received != recv || wbusy != busy {
		t.Fatalf("window exams %d orders %d busy %d; days %d %d %d", ev.Ops.Exams, ev.Ops.Demand.Received, wbusy, exams, recv, busy)
	}
	// Money: whole days, no history, no inherited claims; the takeover
	// day's midnight accruals are fully the window's.
	var rev, exp int64
	for _, d := range w.History(6) {
		rev += d.NetRevenueCents
		exp += d.ExpenseCents
	}
	today, _ := w.DaySummary(w.Calendar().DayIndex(w.Engine().Now()))
	rev += today.NetRevenueCents
	exp += today.ExpenseCents
	if ev.Money.NetRevenueCents != rev || ev.Money.OperatingExpenseCents != exp || ev.Money.InheritedVarianceCents != 0 || !ev.Money.Reconciled {
		t.Fatalf("window revenue %d expenses %d; days %d %d", ev.Money.NetRevenueCents, ev.Money.OperatingExpenseCents, rev, exp)
	}
}

// The window starts at the exact takeover instant: everything at that
// instant (day start, payroll, remittances) belongs to the inherited
// state; the window's books are exactly the journal entries posted after
// the baseline.
func TestEvaluationExactTakeoverBoundary(t *testing.T) {
	cfg, err := scenario.Resolve("established")
	if err != nil {
		t.Fatal(err)
	}
	c := *cfg
	c.StartClock = 10 * simtime.Hour // remittances post at 10:00 on the takeover day
	w := sim.New(&c, rng.SeedFromInt(2))
	w.Engine().RunUntil(w.StartTime().Add(3*simtime.Day), nil)
	ev := w.Evaluation()
	start := w.StartTime()
	var coll, paid, atStart int64
	for _, e := range w.Journal() {
		if e.Index < ev.Baseline.LedgerEntries {
			if e.At > start {
				t.Fatalf("baseline entry %d after the takeover", e.Index)
			}
			if e.At == start && e.Kind == "payment" {
				atStart++
			}
			continue
		}
		if e.At <= start {
			t.Fatalf("window entry %d (%s) at or before the takeover", e.Index, e.Kind)
		}
		for _, l := range e.Lines {
			if l.Account == "cash" && l.Amt > 0 {
				coll += l.Amt
			}
			if l.Account == "cash" && l.Amt < 0 {
				paid -= l.Amt
			}
		}
	}
	if atStart == 0 {
		t.Fatal("no remittance at the takeover instant: the boundary is not exercised")
	}
	if ev.Money.CollectionsCents != coll || ev.Money.DisbursementsCents != paid || !ev.Money.Reconciled {
		t.Fatalf("window collections %d paid %d; journal after baseline %d %d", ev.Money.CollectionsCents, ev.Money.DisbursementsCents, coll, paid)
	}
	// Orders received on the takeover day before 10:00 are in the calendar
	// day, not in the window; every order after it is in the window.
	cal := w.Calendar()
	td := cal.DayIndex(start)
	var before, sameDay, after int
	for _, o := range w.Orders() {
		if o.Origin[1] < 0 || cal.DayIndex(o.Received) < td {
			continue // walk-ins are counted apart; earlier days are history
		}
		switch {
		case o.Received <= start:
			before++
		case cal.DayIndex(o.Received) == td:
			sameDay++
			after++
		default:
			after++
		}
	}
	takeoverDay, _ := w.DaySummary(td)
	if before == 0 || ev.Ops.Demand.Received != after || takeoverDay.Ops.Demand.Received != before+sameDay ||
		takeoverDay.Takeover != "2026-10-13T10:00:00" {
		t.Fatalf("orders before %d, after %d (same day %d); window %d; takeover day %d (%s)", before, after, sameDay,
			ev.Ops.Demand.Received, takeoverDay.Ops.Demand.Received, takeoverDay.Takeover)
	}
}

// Inherited claims paid after the takeover are window cash flow, not
// window revenue; their collection variance is reported apart; only claims
// recognized in the window bring revenue.
func TestEvaluationInheritedClaims(t *testing.T) {
	cfg, err := scenario.Resolve("established")
	if err != nil {
		t.Fatal(err)
	}
	w := sim.New(cfg, rng.SeedFromInt(1))
	w.Engine().RunUntil(w.StartTime().Add(8*simtime.Day), nil)
	ev := w.Evaluation()
	base := 0
	for _, c := range w.Claims() {
		if c.Final <= w.StartTime() {
			base++
		}
	}
	var inhColl, inhVar, newVar, rev int64
	var newClaims int
	for _, e := range w.Journal() {
		if e.Index < ev.Baseline.LedgerEntries {
			continue
		}
		for _, l := range e.Lines {
			switch {
			case e.Kind == "payment" && l.Account == "cash" && e.A < int64(base):
				inhColl += l.Amt
			case e.Kind == "settle" && l.Account == "collection_variance" && e.A < int64(base):
				inhVar += l.Amt
			case e.Kind == "settle" && l.Account == "collection_variance":
				newVar += l.Amt
			case e.Kind == "revenue" && l.Account == "patient_service_revenue":
				rev -= l.Amt
				newClaims++
			}
		}
	}
	m := ev.Money
	if inhColl == 0 || m.InheritedCollectionsCents != inhColl || m.InheritedVarianceCents != inhVar ||
		m.RevenueCents != rev || m.NetRevenueCents != rev-newVar || newClaims != len(w.Claims())-base || !m.Reconciled {
		t.Fatalf("window %+v; journal: inherited collections %d variance %d, new variance %d, revenue %d", m, inhColl, inhVar, newVar, rev)
	}
	if m.InheritedCollectionsCents == m.CollectionsCents {
		t.Log("all collections so far are on inherited claims")
	}
}

// A study imaged before the takeover and finalized after it earns its
// revenue in the window (revenue at report final) but stays out of the
// window's TAT cohort (studies imaged in the window).
func TestEvaluationPreTakeoverStudyFinalizedAfter(t *testing.T) {
	cfg, err := scenario.Resolve("established")
	if err != nil {
		t.Fatal(err)
	}
	c := *cfg
	c.StartClock = 13 * simtime.Hour
	w := sim.New(&c, rng.SeedFromInt(1))
	start := w.StartTime()
	w.Engine().RunUntil(start.Add(simtime.Day), nil)
	ev := w.Evaluation()
	var carried, inWindow int
	for _, s := range w.Studies() {
		switch {
		case s.Acquired <= start && s.State == "final" && s.Final > start:
			carried++
		case s.Acquired > start:
			inWindow++
		}
	}
	if ev.Baseline.Unreported == 0 || carried == 0 || ev.Ops.Reporting.Inherited != carried ||
		ev.Ops.Reporting.Acquired != inWindow {
		t.Fatalf("baseline unreported %d; carried over and finalized %d (window says %d); imaged in window %d (window says %d)",
			ev.Baseline.Unreported, carried, ev.Ops.Reporting.Inherited, inWindow, ev.Ops.Reporting.Acquired)
	}
	var rev int64
	for _, e := range w.Journal() {
		if e.Index >= ev.Baseline.LedgerEntries && e.Kind == "revenue" {
			rev += e.Lines[0].Amt
		}
	}
	if ev.Money.RevenueCents != rev || rev == 0 {
		t.Fatalf("window revenue %d, journal %d", ev.Money.RevenueCents, rev)
	}
}

// Owner capital is not operating performance: two practices that differ
// only in opening cash have identical operations and operating results;
// only cash differs, by exactly the capital.
func TestEvaluationExcludesCapitalTransfers(t *testing.T) {
	cfg, err := scenario.Resolve("established")
	if err != nil {
		t.Fatal(err)
	}
	rich := *cfg
	rich.OpeningCashCents = cfg.OpeningCashCents + 35_000_000
	var evs []api.Evaluation
	for _, c := range []*sim.Config{cfg, &rich} {
		w := sim.New(c, rng.SeedFromInt(3))
		w.Engine().RunUntil(w.StartTime().Add(5*simtime.Day), nil)
		evs = append(evs, w.Evaluation())
	}
	a, b := evs[0], evs[1]
	opsA, opsB := a.Ops, b.Ops
	opsA.Alerts, opsB.Alerts = api.AlertCounts{}, api.AlertCounts{} // cash alerts may differ
	ma, mb := a.Money, b.Money
	if !reflect.DeepEqual(opsA, opsB) || ma.OperatingProfitCents != mb.OperatingProfitCents ||
		ma.NetRevenueCents != mb.NetRevenueCents || ma.OperatingCashFlowCents != mb.OperatingCashFlowCents ||
		ma.CapitalTransfersCents != 0 || mb.CapitalTransfersCents != 0 ||
		mb.CashEndCents-ma.CashEndCents != 35_000_000 || b.Baseline.CashCents-a.Baseline.CashCents != 35_000_000 {
		t.Fatalf("operating %+v vs %+v", ma, mb)
	}
}

// Metrics and alerts never see hidden state: with the same past and a
// different hidden future (orders not yet received, patients who will not
// come), every observation, summary and window figure is identical.
func TestMetricsAndAlertsRevealNoFuture(t *testing.T) {
	cfg := miniConfig(t)
	a := sim.New(cfg, rng.SeedFromInt(6))
	b := sim.New(cfg, rng.SeedFromInt(6))
	T := a.Calendar().StartOfDate(cfg.Start.AddDays(3)).Add(11 * simtime.Hour)
	b.DropOrdersFrom(T)
	b.ForceNoShowsFrom(T)
	for _, w := range []*sim.World{a, b} {
		w.Engine().RunUntil(T-1, nil)
	}
	if !reflect.DeepEqual(a.Observe(2000), b.Observe(2000)) || !reflect.DeepEqual(a.History(10), b.History(10)) ||
		!reflect.DeepEqual(a.Evaluation(), b.Evaluation()) {
		t.Fatal("a hidden future leaked into metrics, alerts or summaries")
	}
	a.Engine().RunUntil(T.Add(4*simtime.Hour), nil)
	b.Engine().RunUntil(T.Add(4*simtime.Hour), nil)
	if reflect.DeepEqual(a.Observe(0).Today, b.Observe(0).Today) {
		t.Fatal("the futures did not diverge; the test is vacuous")
	}
}

// Metrics, alerts and summaries are the same however the run is chunked.
func TestMetricsChunkingInvariance(t *testing.T) {
	cfg := miniConfig(t)
	whole := sim.New(cfg, rng.SeedFromInt(9))
	end := whole.Calendar().StartOfDate(cfg.Start.AddDays(4)).Add(15 * simtime.Hour)
	whole.Engine().RunUntil(end, nil)
	r := rand.New(rand.NewPCG(1, 2))
	for trial := 0; trial < 3; trial++ {
		w := sim.New(cfg, rng.SeedFromInt(9))
		for now := w.Engine().Now(); now < end; {
			now = min(end, now.Add(simtime.Duration(r.IntN(5*3600))+1))
			w.Engine().RunUntil(now, nil)
			_ = w.Observe(0) // reading in between changes nothing
			_ = w.Evaluation()
		}
		if w.LogHash() != whole.LogHash() || !reflect.DeepEqual(w.Observe(200), whole.Observe(200)) ||
			!reflect.DeepEqual(w.History(6), whole.History(6)) || !reflect.DeepEqual(w.Evaluation(), whole.Evaluation()) {
			t.Fatalf("trial %d: chunking changed metrics or alerts", trial)
		}
	}
}

// --- alerts -------------------------------------------------------------

func alertLog(w *sim.World) []api.LogEntry {
	var out []api.LogEntry
	for _, e := range w.Observe(2000).RecentLog {
		if e.Severity != api.Info || strings.HasPrefix(e.Text, "Resolved:") {
			out = append(out, e)
		}
	}
	return out
}

// micro-alerts: the only technologist works in CT and covers XR only
// 08:00–08:15 and from 09:30. The 08:00 XR patient arrives 20 minutes late
// (after being marked missed; fitted in), so the XR room is unstaffed with
// a patient on site from 08:20. The wait (from arrival) passes 30 minutes
// at 08:50 (warn) and 60 at 09:20 (escalates to alert, on its own wake).
// Coverage returns at 09:30; registration takes until 09:33, the exam
// starts then. Each alert resolves once its condition has been clear for
// 10 minutes: the room at 09:40, the wait at 09:43.
func TestAlertsTriggerEscalateResolve(t *testing.T) {
	w, d0 := micro(t, "micro-alerts.json")
	if err := runTo(w, at(d0, "08:55")); err != nil {
		t.Fatal(err)
	}
	o := w.Observe(0)
	if len(o.Alerts.Active) != 2 {
		t.Fatalf("at 08:55: %+v", o.Alerts.Active)
	}
	for _, a := range o.Alerts.Active {
		if a.Key == "unstaffed.0" && (a.Since != "2026-10-13T08:20:00" || !strings.Contains(a.Why, "no XR technologist rostered until 09:30")) ||
			a.Key == "wait.0" && (a.Since != "2026-10-13T08:50:00" || a.Severity != api.Warn || !strings.Contains(a.Text, "longest 35m")) {
			t.Fatalf("at 08:55: %+v", a)
		}
	}
	if err := runTo(w, at(d0, "12:00")); err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, e := range alertLog(w) {
		if e.Category == "rooms" || strings.Contains(e.Text, "waiting over") {
			got = append(got, e.Time[11:16]+" "+string(e.Severity)+" "+e.Text)
		}
	}
	want := []string{
		"08:20 warn XR room unstaffed with 1 patient on site",
		"08:50 warn XR: 1 patient waiting over 30m · longest 30m",
		"09:20 alert XR: 1 patient waiting over 30m · longest 1h",
		"09:40 info Resolved: XR room unstaffed with 1 patient on site (raised 08:20)",
		"09:43 info Resolved: XR: 1 patient waiting over 30m · longest 1h (raised 08:50)",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("alert log:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	o = w.Observe(0)
	ops := day(t, w, 0).Ops
	if len(o.Alerts.Active) != 0 || len(o.Alerts.Resolved) != 2 || ops.Alerts.Warn != 2 || ops.Alerts.Alert != 1 {
		t.Fatalf("active %+v resolved %+v counts %+v", o.Alerts.Active, o.Alerts.Resolved, ops.Alerts)
	}
}

// Alerts are evaluated after the whole transaction: with coverage back at
// 08:47, the exam starts at 08:50, the instant the wait reaches 30
// minutes, and no wait alert is raised.
func TestAlertSameInstantOrdering(t *testing.T) {
	w, d0 := micro(t, "micro-alerts-tie.json")
	if err := runTo(w, at(d0, "12:00")); err != nil {
		t.Fatal(err)
	}
	v := w.Visits()[0]
	if v.ExamStart != at(d0, "08:50") {
		t.Fatalf("exam started %s, the fixture expects 08:30", w.Calendar().ISO(v.ExamStart))
	}
	for _, e := range alertLog(w) {
		if strings.Contains(e.Text, "waiting over") {
			t.Fatalf("raised at the threshold instant: %+v", e)
		}
	}
}

// Day-scoped alerts expire at midnight without a resolution entry, and an
// episode is raised once however long it lasts.
func TestAlertsExpireAndDeduplicate(t *testing.T) {
	cfg := *miniConfig(t)
	cfg.Alerts = sim.DefaultAlerts()
	cfg.Alerts.LostWarn = 1
	w := run(t, &cfg, 9, 8)
	o := w.Observe(2000)
	expired := 0
	for _, r := range o.Alerts.Resolved {
		if strings.HasPrefix(r.Key, "lost.") {
			if !r.Expired || r.Resolved[11:] != "00:00:00" {
				t.Fatalf("lost-demand alert ended %+v", r)
			}
			expired++
		}
	}
	if expired == 0 {
		t.Fatal("no lost-demand alert expired: the test is vacuous")
	}
	raised := map[string]int{}
	for _, e := range o.RecentLog {
		if strings.HasPrefix(e.Text, "Resolved:") && strings.Contains(e.Text, " lost today") {
			t.Fatalf("an expiry was logged as a resolution: %s", e.Text)
		}
		if e.Category == "demand" && strings.Contains(e.Text, " lost today") {
			raised[e.Time[:10]+e.Text[:3]]++
		}
	}
	for k, n := range raised {
		if n > 1 {
			t.Fatalf("%s: lost-demand alert raised %d times in one day", k, n)
		}
	}
}

// BenchmarkObserveMidday is one TUI frame's observation on the established
// practice at 11:00 on day 1 (today's metrics and alerts included).
func BenchmarkObserveMidday(b *testing.B) {
	cfg, err := scenario.Resolve("established")
	if err != nil {
		b.Fatal(err)
	}
	w := sim.New(cfg, rng.SeedFromInt(1))
	w.Engine().RunUntil(w.StartTime().Add(5*simtime.Hour), nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = w.Observe(12)
	}
}

// BenchmarkEvaluation30Days is the evaluation window after 30 days of play
// (computed on request, not per frame).
func BenchmarkEvaluation30Days(b *testing.B) {
	cfg, err := scenario.Resolve("established")
	if err != nil {
		b.Fatal(err)
	}
	w := sim.New(cfg, rng.SeedFromInt(1))
	w.Engine().RunUntil(w.StartTime().Add(30*simtime.Day), nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = w.Evaluation()
	}
}
