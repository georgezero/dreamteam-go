package sim_test

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/rng"
	"github.com/georgezero/dreamteam-go/internal/scenario"
	"github.com/georgezero/dreamteam-go/internal/sim"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Finance (SIMULATION_DESIGN §8). Amounts are integer cents.

func payerByID(cfg *sim.Config, id string) (int, *sim.PayerConfig) {
	for i := range cfg.Finance.Payers {
		if cfg.Finance.Payers[i].ID == id {
			return i, &cfg.Finance.Payers[i]
		}
	}
	panic("no payer " + id)
}

func examByCode(cfg *sim.Config, code string) *sim.ExamConfig {
	for i := range cfg.Catalog {
		if cfg.Catalog[i].Code == code {
			return &cfg.Catalog[i]
		}
	}
	panic("no exam " + code)
}

func microCfg(t *testing.T, name string) *sim.Config {
	t.Helper()
	f, err := scenario.Load(os.DirFS("testdata"), name)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := scenario.Compile(f)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

// runDays advances w day by day to end, checking every invariant
// (ledger, cash, A/R, claims) at each day boundary.
func runDays(t *testing.T, w *sim.World, end simtime.Time) {
	t.Helper()
	for now := w.Engine().Now(); now < end; {
		now = min(end, w.Calendar().StartOfDay(w.Calendar().DayIndex(now)+1))
		if err := runTo(w, now); err != nil {
			t.Fatal(err)
		}
		if err := w.Check(); err != nil {
			t.Fatalf("%s: %v", w.Calendar().Short(now), err)
		}
	}
}

// The documented arithmetic: NCR per payer class, allowed and expected net,
// daily accruals that add up exactly, and weekly overtime pricing.
func TestFinanceArithmetic(t *testing.T) {
	cfg := miniConfig(t)
	for id, want := range map[string]int64{"commercial": 950_000, "medicare": 980_000, "medicaid": 934_000, "self_pay": 400_000} {
		if _, p := payerByID(cfg, id); p.NCRPPM() != want {
			t.Errorf("%s NCR %d ppm, want %d", id, p.NCRPPM(), want)
		}
	}
	// $260 × 1.60 × 0.95 = $395.20 (TUI_DESIGN §4.4).
	ci, _ := payerByID(cfg, "commercial")
	x := examByCode(cfg, "CT_ABD_PEL_WC")
	if a, e := cfg.Finance.AllowedCents(x, ci), cfg.Finance.ExpectedNetCents(x, ci); a != 41_600 || e != 39_520 {
		t.Errorf("allowed %d, expected net %d", a, e)
	}
	for _, n := range []int{28, 29, 30, 31} {
		var sum int64
		for k := 1; k <= n; k++ {
			sum += sim.DailyAccrual(2_200_000, k, n)
		}
		if sum != 2_200_000 {
			t.Errorf("%d-day month accrues %d", n, sum)
		}
	}
	var year int64
	for d := int64(1000); d < 1365; d++ {
		year += sim.SalaryAccrual(28_800_000, d)
	}
	if year != 28_800_000 {
		t.Errorf("365 daily salary accruals sum to %d", year)
	}
	h := simtime.Hour
	for _, c := range []struct {
		before, paid simtime.Duration
		wage         int64
		premium      simtime.Duration
	}{
		{0, 8 * h, 8 * 4000, 0},
		{36 * h, 8 * h, 4*4000 + 4*6000, 4 * h},
		{40 * h, 2 * h, 2 * 6000, 2 * h},
		{32 * h, 8 * h, 8 * 4000, 0},
	} {
		w, p := sim.HourlyPay(4000, c.before, c.paid, 40*h, 15_000)
		if w != c.wage || p != c.premium {
			t.Errorf("week %s + %s: wage %d premium %s, want %d %s", c.before, c.paid, w, p, c.wage, c.premium)
		}
	}
}

// Expected net is the mean of what is collected: over many claims the
// collected fraction of allowed matches each payer's NCR, so collection
// variance averages to about zero (a persistent bias is a calibration bug).
func TestCollectionVarianceMeanZero(t *testing.T) {
	cfg := miniConfig(t)
	w := sim.New(cfg, rng.SeedFromInt(3))
	const n = 30_000
	for pi, p := range cfg.Finance.Payers {
		var paid int64
		for k := int64(0); k < n; k++ {
			out, _, frac, _ := w.DrawClaim([4]int64{k / 97, k % 7, k % 13, k}, pi)
			switch out {
			case sim.OutPaid, sim.OutRecovered:
				paid += 10_000
			case sim.OutShort:
				paid += frac
			}
		}
		mean := paid * 100 / n // ppm of allowed
		if d := mean - p.NCRPPM(); d < -6_000 || d > 6_000 {
			t.Errorf("%s: collected %d ppm of allowed, NCR %d ppm", p.ID, mean, p.NCRPPM())
		}
	}
}

// Revenue is recognized when, and only when, a report is finalized: never
// at booking, arrival, registration, imaging or worklist entry, and never
// for cancelled, no-show, left, rescheduled-unperformed or unread exams.
func TestRevenueOnlyAtReportFinal(t *testing.T) {
	cfg := miniConfig(t)
	w := sim.New(cfg, rng.SeedFromInt(1))
	cal := w.Calendar()
	start := w.StartTime()
	sawGap := false
	for now := start; now < start.Add(18*simtime.Hour); now = now.Add(simtime.Minute) {
		if err := runTo(w, now); err != nil {
			t.Fatal(err)
		}
		final, acquired := 0, 0
		for _, s := range w.Studies() {
			acquired++
			if s.State == "final" {
				final++
			}
		}
		revenue := 0
		for _, e := range w.Journal() {
			if e.Kind == "revenue" {
				revenue++
			}
		}
		if revenue != final {
			t.Fatalf("%s: %d revenue entries, %d final reports", cal.Short(now), revenue, final)
		}
		sawGap = sawGap || acquired > final
	}
	if !sawGap {
		t.Fatal("no study was ever imaged but unread; the test is vacuous")
	}

	runDays(t, w, cal.StartOfDate(cfg.Start.AddDays(30)))
	studies := map[string]sim.StudyInfo{}
	for _, s := range w.Studies() {
		studies[s.ID] = s
	}
	revenueAt := map[int64]simtime.Time{} // study index → entry time
	for _, e := range w.Journal() {
		if e.Kind == "revenue" {
			if _, dup := revenueAt[e.A]; dup {
				t.Fatalf("study %d recognized twice", e.A)
			}
			revenueAt[e.A] = e.At
		}
	}
	n := 0
	for _, s := range w.Studies() {
		at, ok := revenueAt[idNum(s.ID)-1]
		if (s.State == "final") != ok || (ok && at != s.Final) {
			t.Fatalf("%s %s: revenue %v at %d (final %d)", s.ID, s.State, ok, at, s.Final)
		}
		if ok {
			n++
		}
	}
	states := map[string]int{}
	for _, o := range w.Orders() {
		states[o.State]++
	}
	if n != len(w.Claims()) || n != states["reported"] {
		t.Fatalf("%d revenue entries, %d claims, %d reported orders", n, len(w.Claims()), states["reported"])
	}
	for _, s := range []string{"cancelled", "no-show", "left", "lost", "booked", "imaged"} {
		if states[s] == 0 {
			t.Errorf("no %s orders in 30 days; the test is vacuous for them", s)
		}
	}
}

// idNum parses the number of a public ID such as S00012.
func idNum(id string) int64 {
	n, err := strconv.ParseInt(id[1:], 10, 64)
	if err != nil {
		panic(err)
	}
	return n
}

// Claims follow the documented lifecycle and timing, with outcomes and
// amounts given by the origin-keyed payer draw; A/R, revenue and cash
// reconcile to the journal.
func TestClaimLifecycleAndTiming(t *testing.T) {
	cfg := miniConfig(t)
	w := sim.New(cfg, rng.SeedFromInt(2))
	cal := w.Calendar()
	runDays(t, w, cal.StartOfDate(cfg.Start.AddDays(150)))
	f := &cfg.Finance
	business := func(t simtime.Time) bool {
		wd := cal.Weekday(t)
		return wd != simtime.Saturday && wd != simtime.Sunday
	}
	nextBusiness := func(day int64) int64 {
		for !business(cal.StartOfDay(day)) {
			day++
		}
		return day
	}
	seen := map[string]int{}
	for _, c := range w.Claims() {
		pi, p := payerByID(cfg, c.Payer)
		x := examByCode(cfg, c.Exam)
		if c.Allowed != f.AllowedCents(x, pi) || c.Expected != f.ExpectedNetCents(x, pi) {
			t.Fatalf("%s: allowed %d expected %d", c.ID, c.Allowed, c.Expected)
		}
		if c.State == "pending" {
			seen["pending"]++
			continue
		}
		// Charge lag: the drawn 1–3 business days after final, at the
		// submit time.
		fd, sd := cal.DayIndex(c.Final), cal.DayIndex(c.Submitted)
		lag := 0
		for d := fd + 1; d <= sd; d++ {
			if business(cal.StartOfDay(d)) {
				lag++
			}
		}
		if !business(c.Submitted) || simtime.ClockOf(c.Submitted) != f.SubmitClock || lag != int(w.DrawChargeLag(c.Origin)) {
			t.Fatalf("%s: final %s submitted %s (%d business days)", c.ID, cal.Short(c.Final), cal.Short(c.Submitted), lag)
		}
		out, delay, frac, rework := w.DrawClaim(c.Origin, pi)
		remit := cal.At(nextBusiness(sd+delay), f.PaymentClock)
		if c.WasDenied {
			if p.SelfPay || c.Denied != remit {
				t.Fatalf("%s: denied %s, remittance due %s", c.ID, cal.Short(c.Denied), cal.Short(remit))
			}
			if c.State != "denied" && (cal.DayIndex(c.Resubmitted) <= cal.DayIndex(c.Denied) || !business(c.Resubmitted) ||
				cal.DayIndex(c.Resubmitted)-cal.DayIndex(c.Denied) > 3) {
				t.Fatalf("%s: denied %s resubmitted %s", c.ID, cal.Short(c.Denied), cal.Short(c.Resubmitted))
			}
			remit = cal.At(nextBusiness(cal.DayIndex(c.Denied)+rework), f.PaymentClock)
		}
		switch c.State {
		case "submitted", "denied", "resubmitted":
			seen["open"]++
			if remit <= w.Engine().Now() {
				t.Fatalf("%s still %s after its remittance date %s", c.ID, c.State, cal.Short(remit))
			}
			continue
		}
		if c.Closed != remit {
			t.Fatalf("%s %s closed %s, remittance due %s", c.ID, c.State, cal.Short(c.Closed), cal.Short(remit))
		}
		want := map[int]struct {
			state string
			paid  int64
		}{
			sim.OutPaid:       {"paid", c.Allowed},
			sim.OutShort:      {"short_paid", (c.Allowed*frac + 5_000) / 10_000},
			sim.OutRecovered:  {"paid", c.Allowed},
			sim.OutWrittenOff: {"written_off", 0},
		}[out]
		if c.State != want.state || c.Paid != want.paid || c.WasDenied != (out == sim.OutRecovered || out == sim.OutWrittenOff) && !p.SelfPay {
			t.Fatalf("%s: %s paid %d (denied %v), draw %d wants %s %d", c.ID, c.State, c.Paid, c.WasDenied, out, want.state, want.paid)
		}
		key := c.State
		if c.WasDenied {
			key = "denied→" + c.State
		}
		if p.SelfPay && c.State == "written_off" {
			key = "self-pay unpaid"
		}
		seen[key]++
	}
	for _, k := range []string{"paid", "short_paid", "denied→paid", "denied→written_off", "self-pay unpaid", "open", "pending"} {
		if seen[k] == 0 {
			t.Errorf("no %s claim in 150 days: %v", k, seen)
		}
	}

	// Reconcile from the journal alone: per claim, A/R postings net to the
	// open balance; cash moves only on payments and payroll/bills paid.
	studyClaim := map[int64]int{}
	for ci, c := range w.Claims() {
		studyClaim[idNum(c.Study)-1] = ci
	}
	arByClaim := map[int]int64{}
	var cash, receipts, paidOut int64
	for _, e := range w.Journal() {
		for _, l := range e.Lines {
			if l.Account == "accounts_receivable" {
				ci := int(e.A)
				if e.Kind == "revenue" {
					ci = studyClaim[e.A]
				}
				arByClaim[ci] += l.Amt
			}
			if l.Account == "cash" {
				cash += l.Amt
				switch e.Kind {
				case "payment":
					receipts += l.Amt
				case "payroll", "pay_ap":
					paidOut -= l.Amt
				case "opening":
				default:
					t.Fatalf("cash moved on a %s entry", e.Kind)
				}
			}
		}
	}
	var ar int64
	for ci, c := range w.Claims() {
		open := c.State != "paid" && c.State != "short_paid" && c.State != "written_off"
		bal := arByClaim[ci]
		if open && bal != c.Expected-c.Paid || !open && bal != 0 {
			t.Fatalf("%s %s: A/R postings net to %d", c.ID, c.State, bal)
		}
		ar += bal
	}
	if ar != w.Balance("accounts_receivable") || cash != w.Balance("cash") ||
		cash != cfg.OpeningCashCents+receipts-paidOut || receipts == 0 || paidOut == 0 {
		t.Fatalf("A/R %d vs %d; cash %d vs %d (opening %d + %d − %d)", ar, w.Balance("accounts_receivable"),
			cash, w.Balance("cash"), cfg.OpeningCashCents, receipts, paidOut)
	}
}

// Weekly overtime: hourly pay is regular until the week's paid time
// reaches 40 h, then ×1.5; unpaid breaks are excluded; the week resets on
// Monday. Tess works 10.5 paid hours Tue–Fri (11 h on site, 30 min unpaid
// lunch) and 4 h on Saturday, then Monday again.
func TestPayrollWeeklyOvertime(t *testing.T) {
	w, day := micro(t, "micro-week.json")
	runDays(t, w, day.Add(7*simtime.Day))
	cal := w.Calendar()
	wages := map[string]int64{} // date → Tess's wages
	var benefits int64
	for _, e := range w.Journal() {
		if e.Kind == "wages" && e.A == 0 {
			if cal.DayIndex(e.At) != e.B {
				t.Fatalf("wages for day %d posted on %s", e.B, cal.Short(e.At))
			}
			wages[cal.Date(e.At).String()] = e.Lines[0].Amt
			benefits += e.Lines[1].Amt
		}
	}
	want := map[string]int64{
		"2026-10-13": 63_000, "2026-10-14": 63_000, "2026-10-15": 63_000,
		"2026-10-16": 8*6_000 + 6_000/2 + 2*9_000, // 8.5 h regular reaches 40 h, then 2 h at ×1.5
		"2026-10-17": 4 * 9_000,                   // Saturday: all past 40 h
		"2026-10-19": 63_000,                      // Monday: a new week
	}
	if len(wages) != len(want) {
		t.Fatalf("wage entries %v", wages)
	}
	var total int64
	for d, v := range want {
		if wages[d] != v {
			t.Errorf("%s: wages %d, want %d", d, wages[d], v)
		}
		total += v
	}
	if benefits != total/4 {
		t.Errorf("benefits %d, want 25%% of %d", benefits, total)
	}
}

// Required overtime after scheduled close is paid as worked time, once:
// the day's wages are the scheduled part plus exactly the overtime cost
// recorded by the staffing engine, at ×1.5 when the week is past its
// threshold (the premium fixture sets it to 0 h).
func TestPayrollRequiredOvertime(t *testing.T) {
	for _, tc := range []struct {
		file          string
		wages, otCost int64
	}{{"micro-overtime.json", 12_600, 3_600}, {"micro-overtime-premium.json", 18_900, 5_400}} {
		w, day := micro(t, tc.file)
		runDays(t, w, day.Add(simtime.Day))
		var tess int64
		n := 0
		for _, e := range w.Journal() {
			if e.Kind == "wages" && e.A == 0 {
				tess += e.Lines[0].Amt
				n++
			}
		}
		// Scheduled 07:30–09:00 (1.5 h) plus 36 min held over.
		if n != 1 || tess != tc.wages || w.OvertimeCostTotal() != tc.otCost {
			t.Errorf("%s: %d entries, wages %d (want %d), overtime cost %d (want %d)", tc.file, n, tess, tc.wages,
				w.OvertimeCostTotal(), tc.otCost)
		}
	}
}

// Fixed costs accrue daily at 00:00 and add up to the monthly amounts;
// payables are paid on the 1st (before that day's accruals); supplies and
// PACS post at acquisition; billing fees are 5% of each payment; payroll
// is paid every other Friday from the anchor and covers what was accrued
// through Thursday; wages post on the day worked and salaries once a day.
func TestCostAndPaymentTiming(t *testing.T) {
	cfg := miniConfig(t)
	w := sim.New(cfg, rng.SeedFromInt(4))
	cal := w.Calendar()
	end := cal.StartOfDate(simtime.Date{Year: 2026, Month: 12, Day: 1}).Add(simtime.Hour)
	runDays(t, w, end)
	f := &cfg.Finance
	var monthly, novFixed int64
	for _, fc := range f.Fixed {
		monthly += fc.Monthly
	}
	fixedDays := map[int64]bool{}
	salaries := map[[2]int64]int{}
	var ap int64 // payables balance (credit-positive) before the entry
	studyCost := map[int64]bool{}
	payAP, paydays := 0, 0
	journal := w.Journal()
	for i, e := range journal {
		var apDelta, sum int64
		for _, l := range e.Lines {
			sum += l.Amt
			if l.Account == "accounts_payable" {
				apDelta -= l.Amt
			}
		}
		if sum != 0 {
			t.Fatalf("J%d unbalanced", i+1)
		}
		date := cal.Date(e.At)
		switch e.Kind {
		case "fixed":
			if simtime.ClockOf(e.At) != 0 || fixedDays[e.A] || cal.DayIndex(e.At) != e.A {
				t.Fatalf("fixed accrual %d at %s", e.A, cal.Short(e.At))
			}
			fixedDays[e.A] = true
			var want int64
			for _, fc := range f.Fixed {
				want += sim.DailyAccrual(fc.Monthly, date.Day, date.DaysInMonth())
			}
			if apDelta != want {
				t.Fatalf("%s fixed accrual %d, want %d", date, apDelta, want)
			}
			if date.Month == 11 {
				novFixed += want
			}
		case "pay_ap":
			payAP++
			if date.Day != 1 || simtime.ClockOf(e.At) != 0 || -apDelta != ap || journal[i+1].At != e.At {
				t.Fatalf("payables paid %d on %s, balance %d", -apDelta, date, ap)
			}
		case "payroll":
			paydays++
			n := date.DaysSince(f.PayrollAnchor)
			var accrued int64 // accrued payroll through the previous day
			for _, p := range journal[:i] {
				for _, l := range p.Lines {
					if l.Account == "accrued_payroll" {
						accrued -= l.Amt
					}
				}
			}
			if date.Weekday() != simtime.Friday || n%14 != 0 || simtime.ClockOf(e.At) != 0 || e.Lines[0].Amt != accrued {
				t.Fatalf("payroll %d paid %s (accrued %d)", e.Lines[0].Amt, date, accrued)
			}
			for _, p := range journal[i+1:] {
				if p.At != e.At {
					break
				}
				if p.Kind == "payroll" {
					t.Fatal("two payrolls at once")
				}
			}
		case "salary":
			salaries[[2]int64{e.A, e.B}]++
			if cal.DayIndex(e.At) != e.B || simtime.ClockOf(e.At) != 0 {
				t.Fatalf("salary for day %d posted %s", e.B, cal.Short(e.At))
			}
		case "wages":
			if cal.DayIndex(e.At) != e.B {
				t.Fatalf("wages for day %d posted %s", e.B, cal.Short(e.At))
			}
		case "study_cost":
			if studyCost[e.A] {
				t.Fatalf("study %d costed twice", e.A)
			}
			studyCost[e.A] = true
		case "payment":
			cash, fee := e.Lines[0].Amt, int64(0)
			for _, l := range e.Lines {
				if l.Account == "billing" {
					fee = l.Amt
				}
			}
			if fee != (cash*f.BillingFeeBP+5_000)/10_000 {
				t.Fatalf("payment %d, billing fee %d", cash, fee)
			}
		}
		ap += apDelta
	}
	if novFixed != monthly {
		t.Errorf("November fixed costs %d, monthly total %d", novFixed, monthly)
	}
	if payAP != 2 || paydays != 4 { // Nov 1, Dec 1; Oct 16 and 30, Nov 13 and 27
		t.Errorf("%d payables payments, %d paydays", payAP, paydays)
	}
	days := cal.DayIndex(end) + 1
	salaried := 0
	for _, e := range cfg.Employees {
		if e.Pay == sim.Salaried {
			salaried++
		}
	}
	if len(salaries) != int(days)*salaried || len(fixedDays) != int(days) {
		t.Errorf("%d salary accruals for %d salaried × %d days; %d fixed accruals", len(salaries), salaried, days, len(fixedDays))
	}
	for k, n := range salaries {
		if n != 1 {
			t.Fatalf("salary %v accrued %d times", k, n)
		}
	}
	// Supplies and PACS: one entry per acquired study, at acquisition.
	studies := w.Studies()
	if len(studyCost) != len(studies) {
		t.Fatalf("%d study cost entries for %d studies", len(studyCost), len(studies))
	}
	for _, e := range journal {
		if e.Kind != "study_cost" {
			continue
		}
		s := studies[e.A]
		x := examByCode(cfg, s.Exam)
		if e.At != s.Acquired || e.Lines[0].Amt != x.SupplyCostCents || e.Lines[1].Amt != f.PerStudyITCents {
			t.Fatalf("%s costed %v at %s (acquired %s)", s.ID, e.Lines, cal.Short(e.At), cal.Short(s.Acquired))
		}
	}
}

// Results are a function of the seed alone: identical across chunking
// (simulation speed), replays and sparse jumps; a different seed differs.
func TestFinanceDeterminism(t *testing.T) {
	cfg := miniConfig(t)
	end := sim.New(cfg, rng.SeedFromInt(1)).Calendar().StartOfDate(cfg.Start.AddDays(75))
	jump := sim.New(cfg, rng.SeedFromInt(1))
	jump.Engine().RunUntil(end, nil)
	daily := sim.New(cfg, rng.SeedFromInt(1))
	runDays(t, daily, end)
	fine := sim.New(cfg, rng.SeedFromInt(1))
	for now := simtime.Time(0); now < end; now = now.Add(13 * simtime.Minute) {
		fine.Engine().RunUntil(now, nil)
	}
	fine.Engine().RunUntil(end, nil)
	other := sim.New(cfg, rng.SeedFromInt(2))
	other.Engine().RunUntil(end, nil)
	for name, w := range map[string]*sim.World{"daily": daily, "13-minute chunks": fine} {
		if w.LedgerHash() != jump.LedgerHash() || w.LogHash() != jump.LogHash() {
			t.Errorf("%s run differs from one jump", name)
		}
	}
	replay := sim.New(cfg, rng.SeedFromInt(1))
	replay.Engine().RunUntil(end, nil)
	if replay.LedgerHash() != jump.LedgerHash() {
		t.Error("replay differs")
	}
	if other.LedgerHash() == jump.LedgerHash() {
		t.Error("a different seed gives the same ledger")
	}
	if n := len(jump.Claims()); n < 1000 {
		t.Fatalf("only %d claims in 75 days", n)
	}
}

// Handled or stale events unrelated to finance (including at the claim
// submission and remittance times) change no posting.
func TestFinanceUnaffectedByUnrelatedEvents(t *testing.T) {
	cfg := miniConfig(t)
	plain := sim.New(cfg, rng.SeedFromInt(5))
	noisy := sim.New(cfg, rng.SeedFromInt(5))
	cal := plain.Calendar()
	for d := int64(0); d < 50; d++ {
		start := cal.StartOfDay(d)
		for _, w := range []*sim.World{plain, noisy} {
			if err := runTo(w, start.Add(1)); err != nil {
				t.Fatal(err)
			}
		}
		for _, c := range []string{"00:00:02", "09:59:59", "10:00", "10:00:01", "16:59", "17:00", "17:00:01", "23:59:58"} {
			noisy.PushLiveNoise(at(start, c))
			noisy.PushStaleNoise(at(start, c))
		}
	}
	end := cal.StartOfDay(50)
	for _, w := range []*sim.World{plain, noisy} {
		runDays(t, w, end)
	}
	if plain.LedgerHash() != noisy.LedgerHash() || plain.LogHash() != noisy.LogHash() {
		t.Fatal("unrelated events changed the books")
	}
}

// Payer outcomes are keyed by the order's origin: changing the worklist
// policy changes when reports are finalized (and so claim dates and IDs)
// but never a claim's payer, amounts or outcome.
func TestClaimOutcomesIndependentOfOperations(t *testing.T) {
	a := miniConfig(t)
	b := *a
	b.Policies.Worklist = "shortest_first"
	wa, wb := sim.New(a, rng.SeedFromInt(6)), sim.New(&b, rng.SeedFromInt(6))
	end := wa.Calendar().StartOfDate(a.Start.AddDays(100))
	runDays(t, wa, end)
	runDays(t, wb, end)
	byOrigin := map[[4]int64]sim.ClaimInfo{}
	for _, c := range wa.Claims() {
		byOrigin[c.Origin] = c
	}
	common, moved, closed := 0, 0, 0
	for _, c := range wb.Claims() {
		o, ok := byOrigin[c.Origin]
		if !ok {
			continue
		}
		common++
		if o.Final != c.Final {
			moved++
		}
		if o.Payer != c.Payer || o.Allowed != c.Allowed || o.Expected != c.Expected {
			t.Fatalf("%v: %+v vs %+v", c.Origin, o, c)
		}
		done := func(s string) bool { return s == "paid" || s == "short_paid" || s == "written_off" }
		if done(o.State) && done(c.State) {
			closed++
			if o.State != c.State || o.Paid != c.Paid || o.WasDenied != c.WasDenied {
				t.Fatalf("%v: %s %d vs %s %d", c.Origin, o.State, o.Paid, c.State, c.Paid)
			}
		}
	}
	if common < 1000 || moved == 0 || closed < 500 {
		t.Fatalf("vacuous: %d common claims, %d finalized at other times, %d closed in both", common, moved, closed)
	}
}

// A claim's outcome is hidden until its remittance: making every claim
// fail changes nothing observable before the first remittance date, and
// does change things after.
func TestFinanceOutcomesNotLeaked(t *testing.T) {
	cfg := miniConfig(t)
	plain, forced := sim.New(cfg, rng.SeedFromInt(7)), sim.New(cfg, rng.SeedFromInt(7))
	forced.ForceDeniedClaims()
	cal := plain.Calendar()
	obs := func(w *sim.World, journeys bool) string {
		b, err := json.Marshal(w.Observe(2000))
		if err != nil {
			t.Fatal(err)
		}
		s := string(b)
		for i := 0; journeys && i < len(w.Orders()); i++ {
			j, _ := w.Journey(fmt.Sprintf("P%05d", i+1))
			jb, _ := json.Marshal(j)
			s += string(jb)
		}
		return s
	}
	// The earliest remittance is 14 days after the earliest submission.
	for d := int64(1); d <= 15; d++ {
		at := cal.StartOfDate(cfg.Start.AddDays(d)).Add(9 * simtime.Hour)
		runDays(t, plain, at)
		runDays(t, forced, at)
		if obs(plain, d == 15) != obs(forced, d == 15) {
			t.Fatalf("day %d: hidden claim outcomes changed the observation", d)
		}
	}
	if len(plain.Claims()) == 0 || plain.Observe(0).Finance.Claims.Submitted == 0 {
		t.Fatal("no submitted claims; the test is vacuous")
	}
	end := cal.StartOfDate(cfg.Start.AddDays(60))
	runDays(t, plain, end)
	runDays(t, forced, end)
	if plain.Observe(0).Finance.CashCents == forced.Observe(0).Finance.CashCents {
		t.Fatal("forcing every claim to fail changed nothing; the test is vacuous")
	}
}
