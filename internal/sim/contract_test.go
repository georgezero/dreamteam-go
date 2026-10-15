package sim_test

import (
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/rng"
	"github.com/georgezero/dreamteam-go/internal/sim"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// V0.2 Slice 1: employee contracts in running practices (V0.2_DESIGN
// §5.2, §5.4, §9). No V0.2 action exists yet, so a roster below the
// contract is authored in the scenario: Sam (E04, $40/h) contracted for
// 40 hours but rostered Tuesday to Friday only (32 net hours).

const samBelowContract = `{"employees": [{"id": "E04", "contractedWeeklyMinutes": 2400, "roster": {"Sat": null}}]}`

// samIndex is Sam's employee index in the established practice.
const samIndex = 3

// weekLabor sums an employee's wage and guarantee postings to the payroll
// account in the week starting on day index from.
func weekLabor(w *sim.World, emp int64, from int64) (worked, guarantee int64) {
	cal := w.Calendar()
	for _, en := range w.Journal() {
		if en.A != emp || (en.Kind != "wages" && en.Kind != "guarantee") {
			continue
		}
		if d := cal.DayIndex(en.At); d < from || d >= from+7 {
			continue
		}
		for _, ln := range en.Lines {
			if ln.Account != "payroll" {
				continue
			}
			if en.Kind == "wages" {
				worked += ln.Amt
			} else {
				guarantee += ln.Amt
			}
		}
	}
	return worked, guarantee
}

// mondayOnOrAfter returns the first Monday's day index from day.
func mondayOnOrAfter(w *sim.World, day int64) int64 {
	for w.Calendar().Weekday(w.Calendar().StartOfDay(day)) != simtime.Monday {
		day++
	}
	return day
}

// Guaranteed pay when rostered hours are below the contract: every full
// week costs r × C (no less), the shortfall is never worked time, and a
// payday never pays the current week's still-reversible guarantee.
func TestGuaranteeWhenRosterBelowContract(t *testing.T) {
	cfg := withOverlay(t, samBelowContract)
	w := sim.New(cfg, rng.SeedFromInt(3))
	w.Engine().RunUntil(w.StartTime(), nil)
	w.Engine().CheckInvariants = true
	cal := w.Calendar()
	takeover := w.DayIndex()
	first := mondayOnOrAfter(w, cal.DayIndex(w.StartTime()))

	// Paydays (every 14 days from Friday 2026-10-16) in the next 28 days:
	// what is paid is the accrued payroll less the open guarantee.
	paid := 0
	for d := takeover + 1; d <= first+21; d++ {
		at := cal.StartOfDay(d)
		w.Engine().RunUntil(at-1, nil)
		before, open := -w.Balance("accrued_payroll"), w.OpenGuarantee()
		w.Engine().RunUntil(at, nil)
		j := w.Journal()
		for k := len(j) - 1; k >= 0 && j[k].At == at; k-- {
			if j[k].Kind != "payroll" {
				continue
			}
			paid++
			if cal.Weekday(at) == simtime.Friday && open == 0 {
				t.Fatalf("payday %s: no open guarantee before a Friday payday", cal.Date(at))
			}
			for _, ln := range j[k].Lines {
				if ln.Account == "accrued_payroll" && ln.Amt != before-open {
					t.Fatalf("payday %s paid %d, accrued %d less open guarantee %d", cal.Date(at), ln.Amt, before, open)
				}
			}
		}
	}
	if paid == 0 {
		t.Fatal("no payday in the run")
	}
	w.Engine().RunUntil(cal.StartOfDay(first+28), nil)
	for wk := int64(0); wk < 4; wk++ {
		from := first + 7*wk
		worked, guarantee := weekLabor(w, samIndex, from)
		if worked >= 160_000 || guarantee <= 0 {
			t.Fatalf("week %d: worked %d, guarantee %d: expected a shortfall", wk, worked, guarantee)
		}
		if worked+guarantee != 160_000 {
			t.Fatalf("week %d: worked %d + guarantee %d, want r×C = 160000", wk, worked, guarantee)
		}
	}
	var sam api.Employee
	for _, e := range w.Observe(0).Staff {
		if e.ID == "E04" {
			sam = e
		}
	}
	if sam.ContractMin != 2400 || sam.ContractInferred || sam.Employment != "active" {
		t.Fatalf("Sam's contract: %+v", sam)
	}
	if want := int64(2 * 160_000 * 125 / 100); sam.SeveranceCents != want {
		t.Fatalf("severance %d, want two contracted weeks with benefits %d", sam.SeveranceCents, want)
	}
	if err := w.Check(); err != nil {
		t.Fatal(err)
	}
}

// No double payment: a contract at or below the roster never adds pay.
// The practice's ledger and log are those of the unchanged scenario,
// entry for entry.
func TestContractAtOrBelowRosterIsNeutral(t *testing.T) {
	base := sim.New(withOverlay(t, `{}`), rng.SeedFromInt(2))
	below := sim.New(withOverlay(t, `{"employees": [{"id": "E04", "contractedWeeklyMinutes": 1200}, {"id": "E03", "contractedWeeklyMinutes": 2400}]}`), rng.SeedFromInt(2))
	for _, w := range []*sim.World{base, below} {
		w.Engine().RunUntil(w.StartTime().Add(35*simtime.Day), nil)
		for _, en := range w.Journal() {
			if en.Kind == "guarantee" {
				t.Fatalf("guarantee posted with every contract covered by the roster: %+v", en)
			}
		}
	}
	if base.LedgerHash() != below.LedgerHash() || base.LogHash() != below.LogHash() {
		t.Fatal("a contract at or below the roster changed the ledger or the log")
	}
	var obs []api.Employee
	for _, e := range below.Observe(0).Staff {
		obs = append(obs, e)
	}
	for _, c := range []struct {
		id       string
		min      int64
		inferred bool
	}{{"E01", 2400, true}, {"E03", 2400, false}, {"E04", 1200, false}, {"E05", 2400, true}, {"E06", 2370, true}, {"E07", 0, false}} {
		for _, e := range obs {
			if e.ID == c.id && (e.ContractMin != c.min || e.ContractInferred != c.inferred) {
				t.Errorf("%s: contract %d min inferred %v, want %d %v", c.id, e.ContractMin, e.ContractInferred, c.min, c.inferred)
			}
		}
	}
}

// Overtime: approved extensions take a below-contract roster past the
// weekly threshold. Worked pay replaces the guarantee, so there is no
// top-up that week, and the premium is paid on the hours past 40. Sam
// works days here (08:00–16:30, 32 net hours), so a shift can be extended.
func TestGuaranteeWithOvertime(t *testing.T) {
	day := `{"start": "08:00", "end": "16:30", "breaks": [["12:00", "12:30"]], "covers": null}`
	w := sim.New(withOverlay(t, `{"employees": [{"id": "E04", "contractedWeeklyMinutes": 2400, "roster": {"Sat": null,
		"Tue": `+day+`, "Wed": `+day+`, "Thu": `+day+`, "Fri": `+day+`}}]}`), rng.SeedFromInt(4))
	w.Engine().RunUntil(w.StartTime(), nil)
	cal := w.Calendar()
	mon := mondayOnOrAfter(w, w.DayIndex()+1)
	var acts []api.Action
	for d := mon + 1; d <= mon+4; d++ {
		acts = append(acts, api.Action{Type: api.ActExtendShift, Employee: "E04", Minutes: 180, Date: cal.Date(cal.StartOfDay(d)).String()})
	}
	for _, r := range w.Apply(acts) {
		if !r.OK {
			t.Fatalf("extension rejected: %+v", r)
		}
	}
	w.Engine().RunUntil(cal.StartOfDay(mon+7), nil)
	worked, guarantee := weekLabor(w, samIndex, mon)
	if guarantee != 0 {
		t.Fatalf("guarantee %d in a week worked past the contract", guarantee)
	}
	// 44 rostered hours at least: 40 at $40 and 4 at $60.
	if worked < 4000*40+6000*4 {
		t.Fatalf("worked pay %d, want at least 184000 (premium on hours past 40)", worked)
	}
	if paid := w.WeekPaid("Sam"); paid != 0 {
		t.Fatalf("week paid not reset on Monday: %s", paid)
	}
}
