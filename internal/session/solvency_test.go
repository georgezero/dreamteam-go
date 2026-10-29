package session

import (
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/scenario"
)

// V0.7b: a session under a credit line embeds it in its scenario, saves
// and replays exactly — every draw, repayment, interest charge and funding
// refusal — and continues identically after loading.
func TestSolvencySessionReplays(t *testing.T) {
	credit := func(limit int64) func(*scenario.File) {
		return func(f *scenario.File) {
			rate := int64(1200)
			f.OpeningCashCents = 0
			f.Finance.Solvency = &scenario.Solvency{CreditLimitCents: &limit, CreditRateBP: &rate}
		}
	}
	date := func(s *Session, d int64) string {
		cal := s.Calendar()
		return cal.Date(cal.StartOfDay(cal.DayIndex(s.w.StartTime()) + d)).String()
	}
	run := func(s *Session) []api.ActionResult {
		at(t, s, 0, "07:00")
		raise := api.Action{Type: api.ActSetReadingAllocation, Op: api.RosterOpSet, Employee: "E07", Effective: date(s, 15), SiteAllocationBP: 6500}
		cut := api.Action{Type: api.ActSetReadingAllocation, Op: api.RosterOpSet, Employee: "E08", Effective: date(s, 15), SiteAllocationBP: 4500}
		out := append(s.Apply(raise), s.Apply(cut)...)
		at(t, s, 20, "12:00")
		return out
	}
	s := openFile(t, "mini", 4, credit(2_000_000))
	if res := run(s); !res[0].OK || !res[1].OK { // funded by the line
		t.Fatalf("%+v", res)
	}
	sv := s.w.Observe(0).Finance.Financing.Solvency
	if sv == nil || sv.InterestChargedCents == 0 || sv.CreditLimitCents != 2_000_000 {
		t.Fatalf("no borrowing: %+v", sv)
	}
	d := s.Digests()
	p := saveTemp(t, s)
	l, rep, err := Load(p, ReplayOptions{})
	if err != nil || !rep.OK || l.Digests() != d {
		t.Fatalf("%v %s", err, repString(rep))
	}
	for _, x := range []*Session{s, l} {
		at(t, x, 40, "08:00")
	}
	if s.Digests() != l.Digests() {
		t.Fatal("the loaded session diverged")
	}
	// With no credit and no cash the raise is refused, the cut is not; a
	// zero-credit line is valid and replays the same way.
	z := openFile(t, "mini", 4, credit(0))
	if res := run(z); res[0].Code != "insufficient_funds" || !res[1].OK {
		t.Fatalf("zero credit: %+v", res)
	}
	l, rep, err = Load(saveTemp(t, z), ReplayOptions{})
	if err != nil || !rep.OK || l.Digests() != z.Digests() {
		t.Fatalf("zero credit: %v %s", err, repString(rep))
	}
}
