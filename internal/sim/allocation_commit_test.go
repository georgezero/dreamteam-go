package sim_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/sim"
)

// Minimum allocation commitment (0.2.0-dev.6, V0.2_DESIGN §5.11): an
// allocation, once in force, stays in force at least 14 calendar days
// before the next change for that radiologist, checked on the timeline the
// whole batch leaves. Dr. Blake (E08) reads Mon–Fri 11:30–20:00 and some
// Saturdays, never on Sunday: Sun Nov 1 and Mon Nov 2.

// The exploit, as dev.5 allowed it: 30% on Sunday, back on Monday. Every
// shift keeps all its site reading, and the Sunday's salary falls by 20
// points of a day's salary with benefits ($328.77 for Dr. Blake at 50%:
// $480,000 × 0.2 over 365 days, plus 25%). Under this build's rules it is
// refused.
func TestAllocationSundayArbitrageRefused(t *testing.T) {
	arb := []api.Action{setAlloc("E08", "2026-11-01", 3_000), setAlloc("E08", "2026-11-02", 5_000)}
	w := atTakeover(t, `{}`, 12)
	before := digestsOf(w)
	res := w.Apply(arb)
	if c := codes(res); !slices.Equal(c, []string{"commitment_too_short", "batch_rejected"}) || res[0].OK {
		t.Fatalf("%v %+v", c, res)
	}
	if !strings.Contains(res[0].Message, "30% from Sun Nov 1 would be followed by 50% from Mon Nov 2 after 1 day (the earliest next change is Sun Nov 15)") {
		t.Fatalf("message: %s", res[0].Message)
	}
	if digestsOf(w) != before || len(w.ReadingAllocations()[1].Versions) != 1 {
		t.Fatal("a refused batch changed the run")
	}
	// In two batches: the second is refused.
	mustApply(t, w, arb[0])
	if res := w.Apply(arb[1:]); res[0].Code != "commitment_too_short" {
		t.Fatalf("%+v", res[0])
	}

	// dev.5's rules accept it, and it pays: the same reading, a cheaper
	// Sunday.
	plain, cheap := atTakeover(t, `{}`, 12), atTakeover(t, `{}`, 12)
	cheap.SetDev5Rules(true)
	if c := codes(cheap.Apply(arb)); !slices.Equal(c, []string{"accepted", "accepted"}) {
		t.Fatalf("dev.5 rules: %v", c)
	}
	end := cheap.Calendar().StartOfDay(dayOf(cheap, "2026-11-04"))
	runChecked(t, plain, end)
	runChecked(t, cheap, end)
	for d := dayOf(w, "2026-10-26"); d < dayOf(w, "2026-11-04"); d++ {
		if !slices.Equal(plain.PlanOn(d).Reading[blake], cheap.PlanOn(d).Reading[blake]) {
			t.Fatalf("day %d: reading blocks differ", d)
		}
	}
	salary := func(w *sim.World, day string) (n int64) {
		for _, e := range entries(w, "salary", blake) {
			if e.B == dayOf(w, day) {
				n += lineAmt(e, "payroll") + lineAmt(e, "benefits")
			}
		}
		return n
	}
	for _, day := range []string{"2026-10-31", "2026-11-02", "2026-11-03"} {
		if salary(plain, day) != salary(cheap, day) {
			t.Fatalf("%s: salary differs", day)
		}
	}
	if saved := salary(plain, "2026-11-01") - salary(cheap, "2026-11-01"); saved < 32_870 || saved > 32_880 {
		t.Fatalf("Sunday saving %d cents", saved)
	}
}

// Spacing on the final timeline: 13 days apart refused, exactly 14
// accepted, inserted between two versions on both sides, and a batch whose
// items pass through a short pair on the way to a valid timeline accepted
// (the check is on the timeline the batch leaves, whatever the order).
func TestAllocationCommitmentSpacing(t *testing.T) {
	w := atTakeover(t, `{}`, 13)
	before := digestsOf(w)
	if res := w.Apply([]api.Action{setAlloc("E07", "2026-10-27", 7_000), setAlloc("E07", "2026-11-09", 6_500)}); res[0].Code != "commitment_too_short" || res[1].Code != "batch_rejected" {
		t.Fatalf("13 days: %+v", res)
	}
	if digestsOf(w) != before {
		t.Fatal("a refused batch changed the run")
	}
	mustApply(t, w, setAlloc("E07", "2026-10-27", 7_000), setAlloc("E07", "2026-11-24", 6_500))
	for _, c := range []struct {
		date, code string
	}{
		{"2026-11-09", "commitment_too_short"}, // 13 after Oct 27
		{"2026-11-11", "commitment_too_short"}, // 13 before Nov 24
		{"2026-11-10", "accepted"},             // exactly 14 on both sides
	} {
		if res := w.Apply([]api.Action{setAlloc("E07", c.date, 8_000)}); res[0].Code != c.code {
			t.Errorf("%s: %+v", c.date, res[0])
		}
	}
	// Replace a pending version's value: its dates are unchanged.
	if res := w.Apply([]api.Action{setAlloc("E07", "2026-11-10", 9_000)}); res[0].Code != "accepted" {
		t.Fatalf("replace: %+v", res[0])
	}
	// Move Nov 24 to Dec 1 by setting Dec 1 first (Nov 24 → Dec 1 is a
	// 7-day pair until the removal): the final timeline is valid.
	if c := codes(w.Apply([]api.Action{setAlloc("E07", "2026-12-01", 6_000), removeAlloc("E07", "2026-11-24")})); !slices.Equal(c, []string{"accepted", "accepted"}) {
		t.Fatalf("move: %v", c)
	}
	var dates []string
	for _, v := range w.ReadingAllocations()[0].Versions[1:] {
		dates = append(dates, v.Effective)
	}
	if !slices.Equal(dates, []string{"2026-10-27", "2026-11-10", "2026-12-01"}) {
		t.Fatalf("timeline %v", dates)
	}
	// Withdrawals under this build's rules only lengthen a pair.
	mustApply(t, w, removeAlloc("E07", "2026-11-10"))
	// A batch with a hire and a short change is refused as a whole.
	ct := weekOf(shiftAt("R3", "07:00", "15:30", "11:00", "11:30"), "Mon", "Tue", "Wed", "Thu", "Fri")
	before = digestsOf(w)
	res := w.Apply([]api.Action{hireOf("ct", "technologist", []string{"XR", "CT"}, 3800, 2400, ct, "2026-11-10"), setAlloc("E07", "2026-11-03", 9_000)})
	if c := codes(res); !slices.Equal(c, []string{"batch_rejected", "commitment_too_short"}) || digestsOf(w) != before || len(w.Observe(0).Staff) != 8 {
		t.Fatalf("mixed batch: %v", c)
	}
	// The check previews too.
	if r := w.Validate(setAlloc("E07", "2026-12-02", 5_000)); r.Code != "commitment_too_short" {
		t.Fatalf("validate: %+v", r)
	}
}

// A timeline from an earlier build (dev.5 had no commitment) keeps its
// short pairs: they are history or were accepted, and later changes are
// checked only on the pairs they make. Withdrawing the middle of a short
// pair is refused when it would leave another short pair; withdrawing the
// whole arbitrage is accepted.
func TestAllocationCommitmentEarlierTimeline(t *testing.T) {
	w := atTakeover(t, `{}`, 14)
	w.SetDev5Rules(true)
	mustApply(t, w, setAlloc("E08", "2026-10-27", 6_000), setAlloc("E08", "2026-10-31", 3_000), setAlloc("E08", "2026-11-02", 6_000))
	w.SetDev5Rules(false)
	// The other radiologist, and changes away from the short pairs, are
	// unaffected.
	mustApply(t, w, setAlloc("E07", "2026-10-27", 7_000), setAlloc("E08", "2026-11-16", 5_500))
	before := digestsOf(w)
	res := w.Apply([]api.Action{removeAlloc("E08", "2026-10-31")})
	if res[0].Code != "commitment_too_short" || !strings.Contains(res[0].Message, "Withdrawing Dr. Blake (E08)'s change effective Sat Oct 31 would leave 60% from Tue Oct 27 in force 6 days before 60% from Mon Nov 2") {
		t.Fatalf("%+v", res[0])
	}
	if digestsOf(w) != before {
		t.Fatal("a refused withdrawal changed the run")
	}
	mustApply(t, w, removeAlloc("E08", "2026-10-31"), removeAlloc("E08", "2026-11-02"))
	if v := w.ReadingAllocations()[1].Versions; len(v) != 3 || v[1].Effective != "2026-10-27" || v[2].Effective != "2026-11-16" {
		t.Fatalf("timeline %+v", v)
	}
}
