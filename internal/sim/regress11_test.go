package sim_test

import (
	"math"
	"reflect"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/rng"
	"github.com/georgezero/dreamteam-go/internal/sim"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Regression tests for the step 11 review (docs/reviews/step-11-astra-review.md).

// Finding 1: the opening adjustment must come after everything else at the
// takeover instant (day start with payroll or bills, 10:00 remittances,
// dispatch), so cash is exactly the target and the opening snapshot
// matches the books when play begins.
func TestRegress11TakeoverCollisions(t *testing.T) {
	for _, tc := range []struct {
		name  string
		date  simtime.Date
		clock simtime.Duration
		want  string // an operating entry that must share the takeover instant
	}{
		{"midnight payday (Fri Oct 16)", simtime.Date{Year: 2026, Month: 10, Day: 16}, 0, "payroll"},
		{"midnight first of month (Sun Nov 1)", simtime.Date{Year: 2026, Month: 11, Day: 1}, 0, "pay_ap"},
		{"remittance time (Tue Oct 13 10:00)", simtime.Date{Year: 2026, Month: 10, Day: 13}, 10 * simtime.Hour, "payment"},
	} {
		cfg := *established(t)
		cfg.Start, cfg.StartClock = tc.date, tc.clock
		w := sim.New(&cfg, rng.SeedFromInt(1))
		start := w.StartTime()
		w.Engine().RunUntil(start, nil)
		if err := w.Check(); err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		f := w.Observe(0).Finance
		op := f.Opening
		var cashBefore int64
		openings, collided, after := 0, false, 0
		for _, e := range w.Journal() {
			if e.Kind == "opening" {
				openings++
				continue
			}
			if openings > 0 && e.At == start {
				after++
			}
			if e.At == start && e.Kind == tc.want {
				collided = true
			}
			if openings == 0 {
				for _, l := range e.Lines {
					if l.Account == "cash" {
						cashBefore += l.Amt
					}
				}
			}
		}
		switch {
		case !collided:
			t.Errorf("%s: no %s entry at the takeover instant; the test is vacuous", tc.name, tc.want)
		case openings != 1 || after != 0:
			t.Errorf("%s: %d opening entries, %d operating entries after it at the same instant", tc.name, openings, after)
		case f.CashCents != cfg.OpeningCashCents:
			t.Errorf("%s: cash at takeover %d, want %d", tc.name, f.CashCents, cfg.OpeningCashCents)
		case op.CashBeforeCents != cashBefore || op.AdjustmentCents != cfg.OpeningCashCents-cashBefore:
			t.Errorf("%s: opening %+v, cash before %d", tc.name, op, cashBefore)
		case op.ReceivablesCents != f.ARCents || op.PayablesCents != f.APCents || op.AccruedPayrollCents != f.AccruedPayrollCents ||
			op.Unread != w.Observe(0).Worklist.Unread:
			t.Errorf("%s: snapshot A/R %d A/P %d payroll %d unread %d; books now %d %d %d %d", tc.name, op.ReceivablesCents,
				op.PayablesCents, op.AccruedPayrollCents, op.Unread, f.ARCents, f.APCents, f.AccruedPayrollCents, w.Observe(0).Worklist.Unread)
		}
	}
}

// Finding 2: historical cash is shown as it actually was; the takeover
// capital appears once, at takeover. A pro forma series, if offered, is
// separate and named as such.
func TestRegress11ActualHistoricalCash(t *testing.T) {
	cfg := established(t)
	w := built(t, cfg, 1)
	cal := w.Calendar()
	actual := map[string]int64{} // date → end-of-day cash from the journal
	var cash int64
	for _, e := range w.Journal() {
		for _, l := range e.Lines {
			if l.Account == "cash" {
				cash += l.Amt
			}
		}
		actual[cal.Date(e.At).String()] = cash
	}
	f := w.Observe(0).Finance
	for _, d := range f.CashHistory[:len(f.CashHistory)-1] {
		if d.Cents != actual[d.Date] {
			t.Fatalf("cash history %s: %d, actual %d", d.Date, d.Cents, actual[d.Date])
		}
	}
	for _, d := range w.History(30) {
		if d.CashEndCents != actual[d.Date] {
			t.Fatalf("day record %s: cash end %d, actual %d", d.Date, d.CashEndCents, actual[d.Date])
		}
		if d.ProFormaCashEndCents != actual[d.Date]+f.Opening.AdjustmentCents {
			t.Fatalf("day record %s: pro forma %d", d.Date, d.ProFormaCashEndCents)
		}
	}
	if last := f.CashHistory[len(f.CashHistory)-1]; last.Cents != cfg.OpeningCashCents || f.Opening.AdjustmentCents <= 0 {
		t.Fatalf("today %+v, adjustment %d", last, f.Opening.AdjustmentCents)
	}
}

// Review B: a split run (build, then play) equals one uninterrupted run in
// all pending state — the event queue, claims, appointments and studies —
// not only in the log and ledger hashes.
func TestRegress11SplitEqualsWholePendingState(t *testing.T) {
	cfg := established(t)
	split := built(t, cfg, 4)
	end := split.StartTime().Add(2*simtime.Day + 7*simtime.Hour)
	split.Engine().RunUntil(end, nil)
	whole := sim.New(cfg, rng.SeedFromInt(4))
	whole.Engine().RunUntil(end, nil)
	for name, eq := range map[string]bool{
		"queue":        reflect.DeepEqual(split.Engine().PendingEvents(), whole.Engine().PendingEvents()),
		"claims":       reflect.DeepEqual(split.Claims(), whole.Claims()),
		"appointments": reflect.DeepEqual(split.Appointments(), whole.Appointments()),
		"studies":      reflect.DeepEqual(split.Studies(), whole.Studies()),
		"visits":       reflect.DeepEqual(split.Visits(), whole.Visits()),
		"orders":       reflect.DeepEqual(split.Orders(), whole.Orders()),
	} {
		if !eq {
			t.Errorf("%s differ between split and uninterrupted runs", name)
		}
	}
	if len(split.Engine().PendingEvents()) < 100 {
		t.Fatal("almost nothing pending; the test is vacuous")
	}
}

// Review C: the convergence comparator must not pass on non-finite ratios
// (0/0 is NaN, which fails no ordered comparison).
func TestRegress11ConvergenceRejectsNaN(t *testing.T) {
	var p, ref startProfile
	ref.ar, p.ar = 1, 1
	for b := range 4 {
		ref.fill[b], p.fill[b], ref.aging[b], p.aging[b] = 1, 1, 1, 1
	}
	if bad := converged(p, ref); len(bad) != 0 {
		t.Fatalf("identical profiles: %v", bad)
	}
	p.fill[0], ref.fill[0] = 0, 0
	if bad := converged(p, ref); len(bad) == 0 {
		t.Fatal("0/0 passed the convergence check")
	}
	p.fill[0], ref.fill[0] = math.Inf(1), 1
	if bad := converged(p, ref); len(bad) == 0 {
		t.Fatal("an infinite value passed the convergence check")
	}
}
