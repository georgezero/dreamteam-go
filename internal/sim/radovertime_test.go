package sim_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/rng"
	"github.com/georgezero/dreamteam-go/internal/sim"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// V0.7a: radiologist overtime costing (docs/V0.7_OVERTIME_DESIGN.md).
// Under Finance.RadOvertimePaid an on-site salaried radiologist's time on
// duty past the rostered end costs annual ÷ (52 × WeeklyHours) per hour,
// straight time, rounded down per radiologist-day, plus benefits, posted
// when they go home. Off, it costs nothing, as before V0.7.

// radCost is the expected charge for sec seconds at a 40-hour week.
func radCost(annual, sec int64) int64 { return annual * sec / (52 * 40 * 3600) }

// withRadOT returns a world on a copy of cfg with the policy set.
func withRadOT(t *testing.T, cfg *sim.Config, paid bool, seed int64) (*sim.World, simtime.Time) {
	t.Helper()
	c := *cfg
	c.Finance.RadOvertimePaid = paid
	w := sim.New(&c, rng.SeedFromInt(seed))
	w.Engine().CheckInvariants = true
	return w, w.Calendar().StartOfDate(c.Start)
}

// radEntries lists the radiologist overtime entries: payroll and benefits
// amounts by entry.
func radEntries(w *sim.World) (entries []sim.EntryInfo, wages, benefits int64) {
	for _, e := range w.Journal() {
		if e.Kind != "rad_overtime" {
			continue
		}
		entries = append(entries, e)
		for _, l := range e.Lines {
			switch l.Account {
			case "payroll":
				wages += l.Amt
			case "benefits":
				benefits += l.Amt
			}
		}
	}
	return entries, wages, benefits
}

// amounts lists every line amount of the entries of one kind, in order.
func amounts(w *sim.World, kind string) []int64 {
	var out []int64
	for _, e := range w.Journal() {
		if e.Kind == kind {
			for _, l := range e.Lines {
				out = append(out, l.Amt)
			}
		}
	}
	return out
}

// 1. No extension, no patients: no overtime and no charge, for a week.
func TestRadOvertimeNoExtension(t *testing.T) {
	w, d0 := withRadOT(t, rosterOnly(t), true, 1)
	if err := runTo(w, d0.Add(7*simtime.Day)); err != nil {
		t.Fatal(err)
	}
	if es, _, _ := radEntries(w); len(es) != 0 {
		t.Fatalf("charged with no overtime: %+v", es)
	}
	for d := int64(0); d < 7; d++ {
		if ops := day(t, w, d).Ops; person(ops, "Dr. Avery").OvertimeSec != 0 || person(ops, "Dr. Blake").OvertimeSec != 0 {
			t.Fatalf("day %d: overtime with no extension", d)
		}
	}
}

// 2, 3, 5, 7, 8, 9. A 120-minute extension of Dr. Avery (07:30–16:00 on
// Tuesday, $480,000) with no studies to read: the time is on duty, so it
// is charged in full, $461.53 (46,153.8 cents rounded down) plus 25%
// benefits. Paid and unpaid runs differ only in payroll: revenue, the
// salary accrual and hourly wages are the same; profit falls by exactly
// the charge; the payroll pays it in cash.
func TestRadOvertimeExtensionWorked(t *testing.T) {
	cfg := rosterOnly(t)
	var ws [2]*sim.World
	var d0 simtime.Time
	for k, paid := range []bool{false, true} {
		ws[k], d0 = withRadOT(t, cfg, paid, 1)
		r := applyAt(t, ws[k], at(d0, "07:00"), extend("E07", 120))[0]
		if !r.OK || strings.Contains(r.Message, "salary's hourly rate") != paid {
			t.Fatalf("paid %v: %+v", paid, r)
		}
		if paid && !strings.Contains(r.Message, "≈$462 at the salary's hourly rate, if worked") {
			t.Fatalf("estimate: %s", r.Message)
		}
		if err := runTo(ws[k], d0.Add(simtime.Day)); err != nil {
			t.Fatal(err)
		}
	}
	unpaid, paid := ws[0], ws[1]
	const want, wantBen = 46153, 11538 // 48,000,000 × 7,200 / 7,488,000; 25% rounded
	for _, w := range ws {
		a := person(day(t, w, 0).Ops, "Dr. Avery")
		if a.OvertimeSec != 7200 || a.ApprovedOvertimeSec != 7200 {
			t.Fatalf("Avery overtime %+v", a)
		}
	}
	if es, _, _ := radEntries(unpaid); len(es) != 0 {
		t.Fatalf("unpaid charged: %+v", es)
	}
	es, wages, ben := radEntries(paid)
	if len(es) != 1 || wages != want || ben != wantBen {
		t.Fatalf("paid: %d entries, %d + %d", len(es), wages, ben)
	}
	if wf := day(t, paid, 0).Ops.Workforce; wf.OvertimeCostCents != want || wf.ApprovedOvertimeCostCents != want {
		t.Fatalf("overtime cost %+v", wf)
	}
	if wf := day(t, unpaid, 0).Ops.Workforce; wf.OvertimeCostCents != 0 {
		t.Fatalf("unpaid overtime cost %+v", wf)
	}
	diff := func(acct string) int64 { return paid.Balance(acct) - unpaid.Balance(acct) }
	if diff("payroll") != want || diff("benefits") != wantBen || diff("accrued_payroll") != -(want+wantBen) ||
		diff("patient_service_revenue") != 0 || diff("cash") != 0 {
		t.Fatalf("balances: payroll %d benefits %d accrued %d revenue %d cash %d", diff("payroll"), diff("benefits"),
			diff("accrued_payroll"), diff("patient_service_revenue"), diff("cash"))
	}
	for _, kind := range []string{"salary", "wages"} {
		if !reflect.DeepEqual(amounts(paid, kind), amounts(unpaid, kind)) {
			t.Fatalf("%s entries differ", kind)
		}
	}
	// After the next payroll (every 14 days) the charge has left cash and
	// the accrual is cleared.
	for _, w := range ws {
		if err := runTo(w, d0.Add(16*simtime.Day)); err != nil {
			t.Fatal(err)
		}
	}
	if diff("cash") != -(want+wantBen) || diff("accrued_payroll") != 0 || diff("payroll") != want {
		t.Fatalf("after payroll: cash %d accrued %d payroll %d", diff("cash"), diff("accrued_payroll"), diff("payroll"))
	}
}

// 4. Two extensions of the same shift move one scheduled end: the time is
// charged once, as one 120-minute extension (two 60-minute charges would
// round to 46,152).
func TestRadOvertimeTwoExtensionsChargedOnce(t *testing.T) {
	w, d0 := withRadOT(t, rosterOnly(t), true, 1)
	for _, r := range applyAt(t, w, at(d0, "07:00"), extend("E07", 60), extend("E07", 60)) {
		if !r.OK {
			t.Fatal(r.Message)
		}
	}
	if err := runTo(w, d0.Add(simtime.Day)); err != nil {
		t.Fatal(err)
	}
	if es, wages, _ := radEntries(w); len(es) != 1 || wages != 46153 || 2*radCost(48_000_000, 3600) != 46152 {
		t.Fatalf("%d entries, %d", len(es), wages)
	}
}

// Boundary: an extension past the scheduled close, with studies the
// radiologist cannot read in it. micro-rad-ot: practice 08:00–10:00; Dr.
// Rae ($400,000) 07:30–09:00 at 100% site reading; three X-rays at 08:00,
// 08:15 and 08:30. +120 runs to 11:00, an hour past close. Reading blocks
// stay on the roster (the extension is presence: contrast supervision),
// so a study documented after 09:00 waits: the extension is partly idle,
// and all of it is charged, 7,200 s: 38,461 cents.
func TestRadOvertimePastClose(t *testing.T) {
	w0, _ := micro(t, "micro-rad-ot.json")
	w, d0 := withRadOT(t, w0.Config(), true, 1)
	if r := applyAt(t, w, at(d0, "07:00"), extend("D1", 120))[0]; !r.OK || !strings.Contains(r.Message, "until 11:00") {
		t.Fatalf("%+v", r)
	}
	if err := runTo(w, d0.Add(simtime.Day)); err != nil {
		t.Fatal(err)
	}
	ops := day(t, w, 0).Ops
	if p := person(ops, "Dr. Rae"); p.OvertimeSec != 7200 || p.ReadOutsideSec != 0 {
		t.Fatalf("Rae %+v", p)
	}
	if es, wages, _ := radEntries(w); len(es) != 1 || wages != radCost(40_000_000, 7200) || wages != 38461 {
		t.Fatalf("%d entries, %d", len(es), wages)
	}
}

// 2, 6, 7, 9 and multiple radiologists, with patients: two weeks of the
// mini practice with Dr. Avery extended every weekday and any hold-over
// as it comes. Every radiologist-day's charge is its overtime at the
// salary rate, rounded down; the paid and unpaid runs have identical
// operations, revenue, salary and hourly wages, and differ in payroll by
// exactly the charges.
func TestRadOvertimeTwoWeeks(t *testing.T) {
	cfg := miniConfig(t)
	var ws [2]*sim.World
	var d0 simtime.Time
	const days = 14
	for k, paid := range []bool{false, true} {
		ws[k], d0 = withRadOT(t, cfg, paid, 3)
		for d := range int64(days) {
			res := applyAt(t, ws[k], at(d0.Add(simtime.Duration(d)*simtime.Day), "07:00"), extend("E07", 90))
			_ = res // weekends and Avery's days off refuse it
		}
		if err := runTo(ws[k], d0.Add(days*simtime.Day)); err != nil {
			t.Fatal(err)
		}
	}
	unpaid, paid := ws[0], ws[1]
	var want int64
	rads := map[string]bool{}
	for d := range int64(days) {
		a, b := day(t, unpaid, d), day(t, paid, d)
		if a.Exams != b.Exams || !reflect.DeepEqual(a.Ops.Staff, b.Ops.Staff) || !reflect.DeepEqual(a.Ops.Reporting, b.Ops.Reporting) {
			t.Fatalf("day %d: operations differ", d)
		}
		for _, s := range b.Ops.Staff {
			if s.Role == "radiologist" && s.OvertimeSec > 0 {
				want += radCost(48_000_000, s.OvertimeSec)
				rads[s.Name] = true
			}
		}
	}
	es, wages, ben := radEntries(paid)
	if wages != want || want == 0 || len(es) == 0 {
		t.Fatalf("charged %d in %d entries, want %d", wages, len(es), want)
	}
	if len(rads) < 1 {
		t.Fatal("no radiologist overtime")
	}
	diff := func(acct string) int64 { return paid.Balance(acct) - unpaid.Balance(acct) }
	if diff("payroll") != wages || diff("benefits") != ben || diff("patient_service_revenue") != 0 ||
		diff("accounts_receivable") != 0 || diff("supplies") != 0 {
		t.Fatalf("payroll %d (want %d) benefits %d (want %d) revenue %d", diff("payroll"), wages, diff("benefits"), ben,
			diff("patient_service_revenue"))
	}
	for _, kind := range []string{"salary", "wages", "revenue"} {
		if !reflect.DeepEqual(amounts(paid, kind), amounts(unpaid, kind)) {
			t.Fatalf("%s entries differ", kind)
		}
	}
}
