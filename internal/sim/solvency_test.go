package sim_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/sim"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// V0.7b: solvency (docs/V0.7_SOLVENCY_DESIGN.md). The established practice
// is taken over on Tue 2026-10-13 with $150,000; payroll is paid every 14
// days from Fri Oct 16, bills on the 1st. Dr. Avery (E07) reads at 60% of
// a $480,000 salary: raising her to 65% from Oct 27 commits $2,465.75 for
// 30 days (with benefits), the funding test's amount.

const raiseCommit = 3_205_479 - 2_958_904 // salary30 at 65% − at 60%, with benefits

// solvencyOverlay is the established practice with opening cash and a
// credit line (limit and rate).
func solvencyOverlay(cash, limit, rate int64) string {
	return fmt.Sprintf(`{"openingCashCents": %d, "finance": {"solvency": {"creditLimitCents": %d, "creditRateBP": %d}}}`, cash, limit, rate)
}

func solv(w *sim.World) *api.Solvency { return w.Observe(0).Finance.Financing.Solvency }

// openingFor is the opening cash at which the established practice's
// funding headroom at the takeover is h with credit limit limit (the
// headroom moves one for one with opening cash).
func openingFor(t *testing.T, h, limit int64) int64 {
	t.Helper()
	const probe = 15_000_000
	s := solv(atTakeover(t, solvencyOverlay(probe, limit, 0), 1))
	return probe + h - s.FundingHeadroomCents
}

func kinds(w *sim.World, kind string) []sim.EntryInfo {
	var out []sim.EntryInfo
	for _, e := range w.Journal() {
		if e.Kind == kind {
			out = append(out, e)
		}
	}
	return out
}

// 17. Without the setting nothing changes: no solvency view, no credit
// accounts. With it, while the practice never needs to borrow or fund a
// commitment, the run is identical to the run without it, digest for
// digest: the solvency state stays empty and its code posts nothing.
func TestSolvencyOffAndNonBindingUnchanged(t *testing.T) {
	off := atTakeover(t, `{}`, 3)
	on := atTakeover(t, solvencyOverlay(15_000_000, 5_000_000, 1000), 3)
	f := off.Observe(0).Finance.Financing
	if f.Solvency != nil || f.CreditLineModeled {
		t.Fatalf("solvency view without the setting: %+v", f)
	}
	if s := solv(on); s == nil || s.CreditLimitCents != 5_000_000 || s.CreditRateBP != 1000 || !on.Observe(0).Finance.Financing.CreditLineModeled {
		t.Fatalf("solvency view: %+v", s)
	}
	for _, d := range []int{0, 7, 30} {
		to := off.StartTime().Add(simtime.Duration(d) * simtime.Day)
		runChecked(t, off, to)
		runChecked(t, on, to)
		if digestsOf(off) != digestsOf(on) {
			t.Fatalf("day %d: a non-binding credit line changed the run", d)
		}
	}
	if off.Balance("credit_line") != 0 || off.Balance("interest") != 0 {
		t.Fatal("credit accounts used")
	}
}

// 1, 3, 13. Funded from cash: accepted when the headroom covers the
// commitment exactly, refused one cent short (insufficient_funds, nothing
// changes). Preview, validation and application agree in both cases.
func TestSolvencyCashFunded(t *testing.T) {
	for _, c := range []struct {
		short int64
		ok    bool
	}{{0, true}, {1, false}} {
		w := atTakeover(t, solvencyOverlay(openingFor(t, raiseCommit-c.short, 0), 0, 0), 1)
		raise := setAlloc("E07", "2026-10-27", 6500)
		p := w.PreviewStaffing([]api.Action{raise})
		if p.Funding == nil || p.Funding.CommitmentCents != raiseCommit || p.Funding.HeadroomCents != raiseCommit-c.short ||
			p.Funding.Funded != c.ok || p.OK != c.ok {
			t.Fatalf("short %d: preview %+v %+v", c.short, p.OK, p.Funding)
		}
		v := w.ValidateBatch([]api.Action{raise})[0]
		before := digestsOf(w)
		r := w.Apply([]api.Action{raise})[0]
		if r.OK != c.ok || v.OK != c.ok || p.Results[0].Code != r.Code || v.Code != r.Code || v.Message != r.Message {
			t.Fatalf("short %d: preview %s, validate %+v, apply %+v", c.short, p.Results[0].Code, v, r)
		}
		if !c.ok {
			if r.Code != "insufficient_funds" || !strings.Contains(r.Message, "funding headroom $2,466 =") {
				t.Fatalf("refusal: %+v", r)
			}
			if digestsOf(w) != before {
				t.Fatal("a refusal changed the run")
			}
			continue
		}
		// Accepted: the commitment is reserved in full until its effective
		// date, so the same raise again is not fundable.
		if s := solv(w); s.ReservedCents != raiseCommit || s.FundingHeadroomCents != 0 {
			t.Fatalf("after: %+v", s)
		}
		if r := w.Apply([]api.Action{setAlloc("E08", "2026-10-27", 5500)})[0]; r.Code != "insufficient_funds" {
			t.Fatalf("second raise: %+v", r)
		}
	}
}

// 2, 4. The same commitment, $1,000 more than cash can fund: accepted
// with a credit line, refused with none (a zero-credit scenario is
// valid and simply has no undrawn credit).
func TestSolvencyCreditFunded(t *testing.T) {
	for _, limit := range []int64{0, 100_000} {
		w := atTakeover(t, solvencyOverlay(openingFor(t, raiseCommit-100_000, 0), limit, 1000), 1)
		s := solv(w)
		if s.CreditAvailableCents != limit || s.FundingHeadroomCents != raiseCommit-100_000+limit {
			t.Fatalf("limit %d: %+v", limit, s)
		}
		r := w.Apply([]api.Action{setAlloc("E07", "2026-10-27", 6500)})[0]
		if r.OK != (limit > 0) {
			t.Fatalf("limit %d: %+v", limit, r)
		}
	}
}

// Batches are atomic: the first item past the headroom is refused, every
// other item batch_rejected; an item without a commitment (a cut) is
// never refused for funding, even with negative headroom.
func TestSolvencyBatchAndCuts(t *testing.T) {
	w := atTakeover(t, solvencyOverlay(openingFor(t, raiseCommit+1, 0), 0, 0), 1)
	res := w.Apply([]api.Action{setAlloc("E07", "2026-10-27", 6500), setAlloc("E08", "2026-10-27", 5500)})
	if res[0].Code != "batch_rejected" || res[1].Code != "insufficient_funds" || !strings.Contains(res[1].Message, "through action 2") {
		t.Fatalf("%+v", res)
	}
	w = atTakeover(t, solvencyOverlay(openingFor(t, -500_000, 0), 0, 0), 1)
	if r := w.Apply([]api.Action{setAlloc("E07", "2026-10-27", 5500)})[0]; !r.OK {
		t.Fatalf("a cut refused at negative headroom: %+v", r)
	}
	if r := w.Apply([]api.Action{setAlloc("E08", "2026-10-27", 5500)})[0]; r.Code != "insufficient_funds" {
		t.Fatalf("a raise at negative headroom: %+v", r)
	}
}

// 15, multi-day. A reservation is released evenly over its 30 days from
// the effective date, floored cumulatively: on day d it holds
// C − ⌊C × (d − from + 1) / 30⌋, and nothing after day 30.
func TestSolvencyReservationRelease(t *testing.T) {
	w := atTakeover(t, solvencyOverlay(15_000_000, 0, 0), 1)
	mustApply(t, w, setAlloc("E07", "2026-10-27", 6500))
	from := dayOf(w, "2026-10-27")
	for _, d := range []int64{from - 1, from, from + 1, from + 14, from + 29, from + 30} {
		runChecked(t, w, w.Calendar().StartOfDay(d).Add(7*simtime.Hour))
		el := min(30, max(0, d-from+1))
		if got, want := solv(w).ReservedCents, raiseCommit-raiseCommit*el/30; got != want {
			t.Fatalf("day %d: reserved %d, want %d", d-from, got, want)
		}
	}
}

// A paid extension commits its time at the overtime rate with benefits
// for hourly staff; the reservation lasts while it is still to be worked.
func TestSolvencyExtension(t *testing.T) {
	ext := api.Action{Type: api.ActExtendShift, Employee: "E03", Minutes: 60, Date: "2026-10-14"}
	w := atTakeover(t, solvencyOverlay(15_000_000, 0, 0), 1)
	h := solv(w).FundingHeadroomCents
	r := mustApply(t, w, ext)[0]
	c := h - solv(w).FundingHeadroomCents
	const w0 int64 = 3600 * 15_000 / 10_000 // Riley: $36/h, an hour at ×1.5
	if want := w0 + (w0*2500+5000)/10_000; c != want || solv(w).ReservedCents != want {
		t.Fatalf("extension commits %d, want %d (%s)", c, want, r.Message)
	}
	runChecked(t, w, w.Calendar().StartOfDay(dayOf(w, "2026-10-15")))
	if s := solv(w); s.ReservedCents != 0 {
		t.Fatalf("reservation after the day: %+v", s)
	}
	// Short by a cent: refused, with the same code from validation.
	w = atTakeover(t, solvencyOverlay(openingFor(t, c-1, 0), 0, 0), 1)
	if v, r := w.Validate(ext), w.Apply([]api.Action{ext})[0]; r.Code != "insufficient_funds" || v.Code != r.Code || v.Message != r.Message {
		t.Fatalf("validate %+v, apply %+v", v, r)
	}
}

// 5, 6, 7, 8, 9, 10, 11, 12, 16. No opening cash and a $20,000 line at
// 10%: the first payroll draws the line to its limit and leaves the rest
// unfunded (distress, recorded), and payroll is still paid in full, the
// same payroll as without the setting. Interest accrues once a day on the
// balance drawn, rounded half up, to payables; collections repay the line
// at midnight. Every invariant holds at every midnight.
func TestSolvencyCreditLine(t *testing.T) {
	const limit, rate = 2_000_000, 1000
	w := atTakeover(t, solvencyOverlay(0, limit, rate), 4)
	ref := atTakeover(t, `{"openingCashCents": 0}`, 4)
	var log []string
	w.SetLogSink(func(e api.LogEntry) { log = append(log, e.Text) })
	end := w.StartTime().Add(60 * simtime.Day)
	runChecked(t, w, end)
	runChecked(t, ref, end)

	// 8, 9: the obligations are the same as without the line; bills
	// add the interest.
	pay := func(w *sim.World, kind string) (out []int64) {
		for _, e := range kinds(w, kind) {
			if e.At > w.StartTime() {
				out = append(out, e.Lines[0].Amt)
			}
		}
		return
	}
	if p := pay(w, "payroll"); !slices.Equal(p, pay(ref, "payroll")) || len(p) < 4 {
		t.Fatalf("payroll %v, without the line %v", p, pay(ref, "payroll"))
	}
	var interest int64
	for _, e := range kinds(w, "interest") {
		interest += e.Lines[0].Amt
	}
	if s := pay(w, "severance_pay"); len(s) != 0 {
		t.Fatal("severance without a termination")
	}
	ap, apRef := pay(w, "pay_ap"), pay(ref, "pay_ap")
	if len(ap) != len(apRef) || len(ap) == 0 || ap[0]-apRef[0] <= 0 || ap[0]-apRef[0] > interest {
		t.Fatalf("bills %v, without the line %v, interest %d", ap, apRef, interest)
	}

	// 5, 11: draws never take the line past its limit; the first payroll
	// draws all of it, once.
	draws := kinds(w, "credit_draw")
	if len(draws) == 0 || draws[0].Lines[0].Amt != limit {
		t.Fatalf("draws %+v", draws)
	}
	seen := map[[2]int64]bool{}
	for _, e := range draws {
		if seen[[2]int64{e.A, e.B}] {
			t.Fatalf("duplicate draw %+v", e)
		}
		seen[[2]int64{e.A, e.B}] = true
	}
	// 10: distress at the first payroll, recorded.
	if !slices.ContainsFunc(log, func(l string) bool {
		return strings.HasPrefix(l, "Financial distress: payroll") && strings.Contains(l, "credit line fully drawn ($20,000)")
	}) {
		t.Fatal("no distress line")
	}
	if f := ref.Observe(0).Finance.Financing; f.Solvency != nil {
		t.Fatal("reference has solvency")
	}

	// 6, 12, 15: one interest entry per day, on the balance drawn when it
	// posts (00:00, before the day's draws), rounded half up.
	var drawn int64
	days := map[int64]bool{}
	for _, e := range w.Journal() {
		switch e.Kind {
		case "interest":
			if d := w.Calendar().DayIndex(e.At); days[e.A] || e.A != d {
				t.Fatalf("interest entry %+v", e)
			}
			days[e.A] = true
			if want := (drawn*rate + 5000*365) / (10_000 * 365); e.Lines[0].Amt != want || want == 0 {
				t.Fatalf("interest %d on %d, want %d", e.Lines[0].Amt, drawn, want)
			}
		case "credit_draw":
			drawn += e.Lines[0].Amt
		case "credit_repay":
			drawn -= e.Lines[0].Amt
		}
	}
	if len(days) == 0 {
		t.Fatal("no interest")
	}
	// 7: collections repay the line at midnight.
	if len(kinds(w, "credit_repay")) == 0 {
		t.Fatal("never repaid")
	}
	// 16: cash = opening + collections − disbursements + credit drawn.
	s := solv(w)
	if s.CreditDrawnCents != -w.Balance("credit_line") || s.CreditDrawnCents < 0 || s.CreditDrawnCents > limit ||
		s.InterestChargedCents != w.Balance("interest") || s.InterestChargedCents != interest {
		t.Fatalf("solvency %+v", s)
	}
	if err := w.Check(); err != nil {
		t.Fatal(err)
	}
}

// The funding test's commitments are the preview's guaranteed amounts: a
// hire's recruiting fee and earliest lawful exit; a teleradiology
// contract's retainer to its earliest end. Each is reserved (the fee
// excepted: it is a payable at acceptance) when the batch is accepted. A
// roster change commits nothing: a roster cannot exceed the contract,
// whose guarantee is owed already.
func TestSolvencyCommitments(t *testing.T) {
	line := func(p api.StaffingPreview, prefix string) int64 {
		for _, l := range p.Guaranteed {
			if strings.HasPrefix(l.Label, prefix) {
				return l.Cents
			}
		}
		t.Fatalf("no guaranteed line %q in %+v", prefix, p.Guaranteed)
		return 0
	}
	evening := weekOf(shiftAt("R3", "12:00", "20:00", "15:30", "16:00"), weekdays...)
	hire := hireOf("ct-evenings", "technologist", []string{"CT"}, 4000, 2250, evening, "2026-11-10")
	tele := startAct("tele-1", "weeknights", "all", "2026-10-27")
	for _, c := range []struct {
		name            string
		act             func(*sim.World) api.Action
		commit, reserve func(api.StaffingPreview) int64
	}{
		{"hire", func(*sim.World) api.Action { return hire },
			func(p api.StaffingPreview) int64 { return line(p, "Recruiting fee") + line(p, "Earliest lawful exit") },
			func(p api.StaffingPreview) int64 { return line(p, "Earliest lawful exit") }},
		{"teleradiology", func(*sim.World) api.Action { return tele },
			func(p api.StaffingPreview) int64 { return line(p, "Teleradiology retainer to the earliest end") },
			func(p api.StaffingPreview) int64 { return line(p, "Teleradiology retainer to the earliest end") }},
	} {
		w := atTakeover(t, solvencyOverlay(15_000_000, 0, 0), 1)
		a := c.act(w)
		p := w.PreviewStaffing([]api.Action{a})
		if !p.OK || p.Funding == nil || p.Funding.CommitmentCents != c.commit(p) || !p.Funding.Funded {
			t.Fatalf("%s: %+v %+v", c.name, p.Results, p.Funding)
		}
		h := solv(w).FundingHeadroomCents
		mustApply(t, w, a)
		if s := solv(w); s.ReservedCents != c.reserve(p) || s.FundingHeadroomCents != h-c.commit(p) {
			t.Fatalf("%s: reserved %d (want %d), headroom %d → %d, commitment %d", c.name, s.ReservedCents, c.reserve(p),
				h, s.FundingHeadroomCents, c.commit(p))
		}
		// One cent short of the commitment: refused.
		w = atTakeover(t, solvencyOverlay(openingFor(t, c.commit(p)-1, 0), 0, 0), 1)
		if r := w.Apply([]api.Action{c.act(w)})[0]; r.Code != "insufficient_funds" {
			t.Fatalf("%s short: %+v", c.name, r)
		}
	}
	w := atTakeover(t, solvencyOverlay(openingFor(t, -500_000, 0), 0, 0), 1)
	r := rosterNow(t, w, "E03")
	r.A.Fri, r.B.Fri = nil, nil
	p := w.PreviewStaffing([]api.Action{setRoster("E03", "2026-11-02", r)})
	if !p.OK || p.Funding == nil || p.Funding.CommitmentCents != 0 || !p.Funding.Funded || p.Funding.HeadroomCents >= 0 {
		t.Fatalf("roster at negative headroom: %+v %+v", p.Results, p.Funding)
	}
}
