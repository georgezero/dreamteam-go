package sim_test

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/rng"
	"github.com/georgezero/dreamteam-go/internal/scenario"
	"github.com/georgezero/dreamteam-go/internal/sim"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// The established practice: a 180-day warm-up of the ordinary engine from
// the epoch to the start time (SIMULATION_DESIGN §8.4, §8.8).

func established(t testing.TB) *sim.Config {
	t.Helper()
	cfg, err := scenario.Resolve("established")
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func built(t testing.TB, cfg *sim.Config, seed int64) *sim.World {
	t.Helper()
	w := sim.New(cfg, rng.SeedFromInt(seed))
	w.Engine().RunUntil(w.StartTime(), nil)
	if err := w.Check(); err != nil {
		t.Fatal(err)
	}
	return w
}

// At the start time the books reconcile: one opening adjustment sets cash
// to the scenario's figure against opening equity; receivables, payables,
// accrued payroll, open claims and the history's loss all carry over;
// assets = liabilities + equity.
func TestOpeningBalanceSheet(t *testing.T) {
	cfg := established(t)
	w := built(t, cfg, 1)
	start := w.StartTime()
	f := w.Observe(0).Finance
	b, op := f.BalanceSheet, f.Opening
	var openings, before int
	var cashBefore, pl int64
	for _, e := range w.Journal() {
		if e.Kind == "opening" {
			openings++
			if e.At != start || before != e.Index {
				t.Fatalf("opening entry J%d at %d; %d entries precede it", e.Index+1, e.At, before)
			}
			continue
		}
		if e.At > start {
			continue
		}
		before++
		for _, l := range e.Lines {
			switch l.Account {
			case "cash":
				cashBefore += l.Amt
			case "patient_service_revenue", "collection_variance", "payroll", "benefits", "facility", "equipment", "supplies", "it", "billing":
				pl -= l.Amt
			}
		}
	}
	switch {
	case openings != 1:
		t.Fatalf("%d opening entries", openings)
	case !op.Done || op.CashBeforeCents != cashBefore || op.AdjustmentCents != cfg.OpeningCashCents-cashBefore:
		t.Fatalf("opening %+v, cash before %d", op, cashBefore)
	case b.CashCents != cfg.OpeningCashCents || w.Balance("cash") != cfg.OpeningCashCents:
		t.Fatalf("cash %d", b.CashCents)
	case b.AssetsCents != b.LiabilitiesCents+b.EquityCents:
		t.Fatalf("assets %d ≠ liabilities %d + equity %d", b.AssetsCents, b.LiabilitiesCents, b.EquityCents)
	case b.OpeningCapitalCents != op.AdjustmentCents || -w.Balance("opening_equity") != op.AdjustmentCents:
		t.Fatalf("opening capital %d, adjustment %d", b.OpeningCapitalCents, op.AdjustmentCents)
	// The history is the days before day 1; day 1's 00:00 accruals (before
	// the 06:00 start) are already "since day 1".
	case op.HistoryResultCents != b.HistoryEarningsCents || b.HistoryEarningsCents+b.EarningsSinceStartCents != pl:
		t.Fatalf("history result %d / %d+%d, journal P&L %d", op.HistoryResultCents, b.HistoryEarningsCents, b.EarningsSinceStartCents, pl)
	case op.ReceivablesCents != b.ReceivablesCents || op.ReceivablesCents == 0 || op.OpenClaims < 1000:
		t.Fatalf("receivables %d (%d open claims), balance %d", op.ReceivablesCents, op.OpenClaims, b.ReceivablesCents)
	case op.PayablesCents != b.PayablesCents || op.AccruedPayrollCents != b.AccruedPayrollCents || op.PayablesCents == 0 || op.AccruedPayrollCents == 0:
		t.Fatalf("payables %d / %d, payroll %d / %d", op.PayablesCents, b.PayablesCents, op.AccruedPayrollCents, b.AccruedPayrollCents)
	case op.BookedAppointments < 500 || op.HistoryDays != 180:
		t.Fatalf("%d appointments booked ahead, %d history days", op.BookedAppointments, op.HistoryDays)
	}
	// Day 1 starts clean: today's figures hold only today's events (the
	// 00:00 accruals and overnight cancellations), never the history, and
	// the opening adjustment is not a cash flow.
	o := w.Observe(0)
	if o.Finance.CashChangeTodayCents != 0 || o.Finance.Today.NetRevenueCents != 0 || o.Finance.Today.CollectionsCents != 0 ||
		o.Flow.Today.Arrived != 0 || o.Worklist.FinalToday != 0 || o.Finance.SinceStart.From != cfg.Start.String() ||
		o.Finance.Financing.MinCashCents != cfg.OpeningCashCents || o.Finance.Financing.DaysBelowZero != 0 {
		t.Fatalf("day-1 figures carry history: %+v %+v", o.Finance.Today, o.Finance.Financing)
	}
	h := w.History(30)
	// History cash is actual (what the history left); pro forma adds the
	// takeover capital.
	if len(h) != 30 || !h[29].BeforeStart || h[29].Date != cfg.Start.AddDays(-1).String() ||
		h[29].CashEndCents != op.CashBeforeCents || h[29].ProFormaCashEndCents != cfg.OpeningCashCents {
		t.Fatalf("history ends %+v", h[len(h)-1])
	}
}

// Play continues the history seamlessly: every claim open at the start
// settles after it, appointments booked during the warm-up are kept, and the
// week's paid hours (Monday was worked before the start) still count
// toward weekly overtime. A mid-day start also carries patients on site,
// the worklist and reads in progress across the boundary.
func TestStateSurvivesTheBoundary(t *testing.T) {
	cfg := *established(t)
	cfg.StartClock = 13 * simtime.Hour
	w := built(t, &cfg, 2)
	start := w.StartTime()
	cal := w.Calendar()
	o := w.Observe(0)
	if o.Worklist.Unread+o.Worklist.Reading == 0 || len(o.Flow.Waiting) == 0 {
		t.Fatalf("mid-day start has no worklist (%d) or patients on site (%d)", o.Worklist.Unread+o.Worklist.Reading, len(o.Flow.Waiting))
	}
	pendingStudies := map[string]bool{}
	for _, s := range w.Studies() {
		if s.State != "final" {
			pendingStudies[s.ID] = true
		}
	}
	openClaims := map[string]bool{}
	for _, c := range w.Claims() {
		if c.State != "paid" && c.State != "short_paid" && c.State != "written_off" {
			openClaims[c.ID] = true
		}
	}
	var future []sim.ApptInfo
	for _, a := range w.Appointments() {
		if a.BookedAt < start && a.Day > w.DayIndex()+1 && a.State == "booked" {
			future = append(future, a)
		}
	}
	// Riley worked Monday: the week's paid time before Tuesday's shift.
	var monWages int64
	for _, e := range w.Journal() {
		if e.Kind == "wages" && e.B == w.DayIndex()-1 && cfg.Employees[e.A].Name == "Riley" {
			monWages = e.Lines[0].Amt
		}
	}
	rate := cfg.Employees[0].RateCents
	for _, e := range cfg.Employees {
		if e.Name == "Riley" {
			rate = e.RateCents
		}
	}
	if wp := w.WeekPaid("Riley"); monWages == 0 || (int64(wp)*rate/3600-monWages) != 0 {
		t.Fatalf("Riley: week paid %s at start, Monday wages %d", wp, monWages)
	}

	// The longest claim path is a Medicaid denial (≤ 77 days) plus rework
	// (≤ 62 days).
	runDays(t, w, cal.StartOfDay(w.DayIndex()+145))
	for _, s := range w.Studies() {
		if pendingStudies[s.ID] && s.State != "final" {
			t.Fatalf("%s unread at the start is still %s", s.ID, s.State)
		}
	}
	paid := 0
	for _, c := range w.Claims() {
		if openClaims[c.ID] {
			if c.State != "paid" && c.State != "short_paid" && c.State != "written_off" {
				t.Fatalf("%s open at the start is still %s", c.ID, c.State)
			}
			paid++
		}
	}
	appts := map[string]sim.ApptInfo{}
	for _, a := range w.Appointments() {
		appts[a.ID] = a
	}
	kept := 0
	for _, a := range future {
		now := appts[a.ID]
		if now.State == "booked" && now.Day < w.DayIndex() {
			t.Fatalf("%s from the warm-up never resolved", a.ID)
		}
		if now.State == "arrived" {
			kept++
		}
	}
	if paid < 1000 || kept < 300 || len(pendingStudies) == 0 {
		t.Fatalf("vacuous: %d claims settled, %d warm-up bookings seen, %d studies pending", paid, kept, len(pendingStudies))
	}
}

// The build is a pure function of the seed: chunking the warm-up, or
// building and then playing versus one uninterrupted run, changes nothing;
// unrelated live and stale events during the warm-up change nothing.
func TestWarmupDeterminism(t *testing.T) {
	cfg := established(t)
	a := built(t, cfg, 1)
	start := a.StartTime()
	cal := a.Calendar()
	daily := sim.New(cfg, rng.SeedFromInt(1))
	noisy := sim.New(cfg, rng.SeedFromInt(1))
	for d := int64(0); cal.StartOfDay(d) < start; d++ {
		for _, w := range []*sim.World{daily, noisy} {
			w.Engine().RunUntil(min(start, cal.StartOfDay(d)+1), nil)
		}
		if d%3 == 0 {
			for _, c := range []string{"03:00", "10:00", "17:00", "23:59:58"} {
				if at := at(cal.StartOfDay(d), c); at < start {
					noisy.PushLiveNoise(at)
					noisy.PushStaleNoise(at)
				}
			}
		}
	}
	for _, w := range []*sim.World{daily, noisy} {
		w.Engine().RunUntil(start, nil)
	}
	b := built(t, cfg, 1)
	other := built(t, cfg, 2)
	for name, w := range map[string]*sim.World{"daily chunks": daily, "noisy": noisy, "rebuild": b} {
		if w.LogHash() != a.LogHash() || w.LedgerHash() != a.LedgerHash() {
			t.Errorf("%s: start state differs", name)
		}
	}
	if other.LedgerHash() == a.LedgerHash() || other.LogHash() == a.LogHash() {
		t.Error("seed 2 builds the same practice as seed 1")
	}
	oa, ob := mustJSON(t, a.Observe(500)), mustJSON(t, b.Observe(500))
	if oa != ob {
		t.Error("observations differ between identical builds")
	}
	// Build, then play three days, equals one uninterrupted run.
	end := start.Add(3 * simtime.Day)
	a.Engine().RunUntil(end, nil)
	whole := sim.New(cfg, rng.SeedFromInt(1))
	whole.Engine().RunUntil(end, nil)
	if whole.LogHash() != a.LogHash() || whole.LedgerHash() != a.LedgerHash() {
		t.Error("build + play differs from one run")
	}
}

func mustJSON(t testing.TB, v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// Start-state digests for seed 1 (golden). Regenerate only for an
// intentional change to gameplay, and say so in the commit.
func TestStartStateGolden(t *testing.T) {
	w := built(t, established(t), 1)
	const (
		wantLedger = "eff39e05918a05336a287beb4e7603427daa7500da059bc55a5871f4e9ad9784"
		wantLog    = "6b164b8984d88d7b51eb976004adb0ca6ed78bf9f16260922d82f1b09c00a180"
	)
	if w.LedgerHash() != wantLedger || w.LogHash() != wantLog {
		t.Fatalf("start state changed:\n ledger %s\n log    %s", w.LedgerHash(), w.LogHash())
	}
}

// Hidden future facts are not in the opening state: making every patient
// from day 2 a no-show and every claim adjudicated after the start fail
// leaves the observation, the history and the briefing data unchanged.
func TestOpeningStateHidesTheFuture(t *testing.T) {
	cfg := established(t)
	plain := built(t, cfg, 3)
	forced := sim.New(cfg, rng.SeedFromInt(3))
	start := forced.StartTime()
	forced.ForceNoShowsFrom(start.Add(simtime.Day))
	forced.ForceDeniedClaimsFrom(start)
	forced.Engine().RunUntil(start, nil)
	view := func(w *sim.World) string {
		return mustJSON(t, struct {
			O api.Observation
			H []api.DaySummary
			S api.DaySchedule
		}{w.Observe(2000), w.History(180), w.Schedule(w.DayIndex() + 1)})
	}
	if view(plain) != view(forced) {
		t.Fatal("hidden future outcomes changed the opening state")
	}
	end := start.Add(40 * simtime.Day)
	plain.Engine().RunUntil(end, nil)
	forced.Engine().RunUntil(end, nil)
	if plain.LedgerHash() == forced.LedgerHash() {
		t.Fatal("the forced changes had no effect; the test is vacuous")
	}
}

// startProfile is what the warm-up must have converged: forward-booking
// fill by lead bucket (booked ÷ bookable slots, days 0–7, 7–30, 30–90,
// 90–120 from the start) and A/R by age (0–30, 31–60, 61–90, > 90 days).
type startProfile struct {
	fill  [4]float64
	aging [4]float64
	ar    float64
}

func profileAtStart(cfg *sim.Config, seed int64) startProfile {
	w := sim.New(cfg, rng.SeedFromInt(seed))
	w.Engine().RunUntil(w.StartTime(), nil)
	var p startProfile
	edges := []int64{0, 7, 30, 90, 120}
	for b := range 4 {
		var booked, bookable int
		for d := edges[b]; d < edges[b+1]; d++ {
			for _, r := range w.Schedule(w.DayIndex() + d).Rooms {
				for _, s := range r.Slots {
					if s.Status == api.SlotUnstaffed || s.Status == api.SlotNoSupervision {
						continue
					}
					bookable++
					if s.Appt != nil {
						booked++
					}
				}
			}
		}
		p.fill[b] = float64(booked) / float64(bookable)
	}
	f := w.Observe(0).Finance
	p.ar = float64(f.ARCents)
	for b, a := range f.Claims.Aging {
		p.aging[b] = float64(a.Cents)
	}
	return p
}

// meanProfile builds the start state for seeds 1..n with a given warm-up,
// in parallel (each world is independent), and averages the profiles.
func meanProfile(t *testing.T, warmup int64, n int) startProfile {
	cfg := *established(t)
	cfg.WarmupDays = warmup
	out := make([]startProfile, n)
	sem := make(chan struct{}, 4)
	done := make(chan struct{})
	for i := range n {
		go func() {
			sem <- struct{}{}
			out[i] = profileAtStart(&cfg, int64(i+1))
			<-sem
			done <- struct{}{}
		}()
	}
	for range n {
		<-done
	}
	var m startProfile
	for _, p := range out {
		for b := range 4 {
			m.fill[b] += p.fill[b] / float64(n)
			m.aging[b] += p.aging[b] / float64(n)
		}
		m.ar += p.ar / float64(n)
	}
	return m
}

// converged compares a profile with the reference: the bulk measures (fill
// to 90 days, total A/R, A/R up to 60 days) within ±5%; the thin tails
// (fill 90–120 days, A/R over 60 days: a few hundred slots or claims)
// within ±25%.
func converged(p, ref startProfile) []string {
	var bad []string
	check := func(name string, got, want, tol float64) {
		if d := got/want - 1; math.IsNaN(d) || math.IsInf(d, 0) || d < -tol || d > tol {
			bad = append(bad, fmt.Sprintf("%s %.4g vs %.4g (%+.1f%%)", name, got, want, 100*d))
		}
	}
	for b, name := range []string{"fill 0–7d", "fill 7–30d", "fill 30–90d", "fill 90–120d"} {
		check(name, p.fill[b], ref.fill[b], map[bool]float64{true: 0.05, false: 0.25}[b < 3])
	}
	check("A/R", p.ar, ref.ar, 0.05)
	for b, name := range []string{"A/R 0–30d", "A/R 31–60d", "A/R 61–90d", "A/R > 90d"} {
		check(name, p.aging[b], ref.aging[b], map[bool]float64{true: 0.05, false: 0.25}[b < 2])
	}
	return bad
}

// Warm-up convergence (ROADMAP step 11): the start state after the 180-day
// warm-up matches a 365-day warm-up, averaged over nine seeds. A 90-day
// warm-up does not (non-vacuity: the forward schedule is still filling).
func TestWarmupConvergence(t *testing.T) {
	if testing.Short() {
		t.Skip("builds 27 practices")
	}
	ref := meanProfile(t, 365, 9)
	if bad := converged(meanProfile(t, 180, 9), ref); len(bad) > 0 {
		t.Errorf("180-day warm-up not converged: %v", bad)
	}
	if bad := converged(meanProfile(t, 90, 9), ref); len(bad) == 0 {
		t.Error("a 90-day warm-up also passes; the test cannot detect an unconverged start")
	} else {
		t.Logf("90-day warm-up (expected to fail): %v", bad)
	}
}

// Negative cash is a tracked financing shortfall, not free borrowing: with
// a thin opening balance the first payroll takes cash below zero, which
// raises one alert per dip, and the lowest balance and the days that ended
// below zero are reported (agreeing with the daily history).
func TestFinancingShortfallTracked(t *testing.T) {
	cfg := *established(t)
	cfg.OpeningCashCents = 1_000_000 // $10,000
	w := built(t, &cfg, 1)
	start := w.StartTime()
	end := start.Add(30 * simtime.Day)
	alerts := 0
	for w.Engine().Now() < end {
		last := w.LastAlert()
		w.Engine().RunUntil(end, func() bool { return w.LastAlert() != last })
		for _, e := range w.Observe(50).RecentLog {
			if e.Seq == w.LastAlert() && e.Category == "finance" && strings.HasPrefix(e.Text, "Cash below zero") {
				alerts++
			}
		}
	}
	if err := w.Check(); err != nil {
		t.Fatal(err)
	}
	o := w.Observe(0)
	fi := o.Finance.Financing
	var low int64 = 1 << 62
	var below int64
	for _, d := range w.History(29) {
		if d.BeforeStart {
			continue
		}
		low = min(low, d.CashEndCents)
		if d.CashEndCents < 0 {
			below++
		}
	}
	if alerts == 0 || fi.MinCashCents >= 0 || fi.MinCashCents > low || fi.DaysBelowZero != below || below == 0 ||
		fi.CreditLineModeled || (o.Finance.CashCents < 0) != (fi.ShortfallCents > 0) {
		t.Fatalf("financing %+v, %d alerts, history low %d, %d days below zero", fi, alerts, low, below)
	}
}
