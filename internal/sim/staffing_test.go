package sim_test

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/rng"
	"github.com/georgezero/dreamteam-go/internal/sim"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// V0.2 Slice 2: hire and terminate (V0.2_DESIGN §5.3–§5.4, §6, §7). The
// established practice is taken over on Tue 2026-10-13 at 06:00; paydays
// fall every other Friday from Fri 2026-10-16 (Oct 30, Nov 13, Nov 27, …).

// shiftAt is a shift in room (or "desk") with an optional break.
func shiftAt(room, start, end string, brk ...string) *api.Shift {
	s := &api.Shift{Start: start, End: end, Breaks: [][2]string{}, Assignment: room, Covers: []api.Cover{}}
	if len(brk) == 2 {
		s.Breaks = [][2]string{{brk[0], brk[1]}}
	}
	return s
}

// weekOf puts a copy of sh on the given weekdays ("Mon" … "Sun").
func weekOf(sh *api.Shift, days ...string) api.Week {
	var w api.Week
	for _, d := range days {
		c := *sh
		p := map[string]**api.Shift{"Mon": &w.Mon, "Tue": &w.Tue, "Wed": &w.Wed, "Thu": &w.Thu, "Fri": &w.Fri, "Sat": &w.Sat, "Sun": &w.Sun}[d]
		*p = &c
	}
	return w
}

var weekdays = []string{"Mon", "Tue", "Wed", "Thu", "Fri"}

func hireOf(req, role string, skills []string, rate, minutes int64, wk api.Week, start string) api.Action {
	return api.Action{Type: api.ActHire, RequestID: req, Role: role, Skills: skills, RateCents: rate,
		ContractedWeeklyMinutes: minutes, Roster: &api.Roster{A: wk, B: wk}, StartDate: start}
}

func terminateOf(id, last string) api.Action {
	return api.Action{Type: api.ActTerminate, Employee: id, LastDay: last}
}

// atTakeover is a world at its takeover.
func atTakeover(t *testing.T, overlay string, seed int64) *sim.World {
	t.Helper()
	w := sim.New(withOverlay(t, overlay), rng.SeedFromInt(seed))
	w.Engine().RunUntil(w.StartTime(), nil)
	return w
}

// dayOf is a date's day index.
func dayOf(w *sim.World, date string) int64 {
	d, err := simtime.ParseDate(date)
	if err != nil {
		panic(err)
	}
	return w.Calendar().DayIndex(w.Calendar().StartOfDate(d))
}

func employee(w *sim.World, id string) api.Employee {
	for _, e := range w.Observe(0).Staff {
		if e.ID == id {
			return e
		}
	}
	panic("no employee " + id)
}

func mustApply(t *testing.T, w *sim.World, acts ...api.Action) []api.ActionResult {
	t.Helper()
	res := w.Apply(acts)
	for _, r := range res {
		if !r.OK {
			t.Fatalf("refused: %+v", r)
		}
	}
	return res
}

// runChecked advances w to time to, checking every invariant at each
// midnight. (Invariants are checked after every transaction around the
// action batches themselves, and at scale in the acceptance suite.)
func runChecked(t *testing.T, w *sim.World, to simtime.Time) {
	t.Helper()
	w.Engine().CheckInvariants = false
	for w.Engine().Now() < to {
		w.Engine().RunUntil(min(to, w.Calendar().StartOfDay(w.DayIndex()+1)), nil)
		if err := w.Check(); err != nil {
			t.Fatalf("%s: %v", w.Calendar().ISO(w.Engine().Now()), err)
		}
	}
}

// entries lists journal entries of a kind for employee index emp.
func entries(w *sim.World, kind string, emp int64) []sim.EntryInfo {
	var out []sim.EntryInfo
	for _, e := range w.Journal() {
		if e.Kind == kind && e.A == emp {
			out = append(out, e)
		}
	}
	return out
}

func lineAmt(e sim.EntryInfo, acct string) int64 {
	var n int64
	for _, l := range e.Lines {
		if l.Account == acct {
			n += l.Amt
		}
	}
	return n
}

// A CT technologist hired for Monday-to-Friday 12:00–20:00 evenings (CT is
// otherwise unstaffed after 16:15 on Mondays, when Sam is off). Recruiting:
// fee on acceptance, nothing else. Onboarding: paid shifts, no capacity —
// the Monday evening of the training week stays unstaffed. Active from the
// midnight after 40 attended hours: Monday evenings are then staffed.
func TestHireLifecycleAndTraining(t *testing.T) {
	const req = "ct-evenings"
	evening := weekOf(shiftAt("R3", "12:00", "20:00", "15:30", "16:00"), weekdays...)
	base := atTakeover(t, `{}`, 1)
	w := atTakeover(t, `{}`, 1)
	w.Engine().CheckInvariants = true
	res := mustApply(t, w, hireOf(req, "technologist", []string{"CT"}, 4000, 2250, evening, "2026-11-10"))
	if r := res[0]; r.Code != api.ActionAccepted || r.EmployeeID != "H00001" || r.EffectiveAt != "2026-11-10" || w.RegistryLen() != 9 {
		t.Fatalf("hire: %+v (registry %d)", r, w.RegistryLen())
	}
	h := employee(w, "H00001")
	// Tue–Fri 7.5 h each (30 h), Mon 37.5 h, Tue 45 h: training ends Tue Nov 17.
	if h.Employment != api.EmploymentRecruiting || h.ContractMin != 2250 || h.Hired == nil ||
		h.Hired.ProductiveDate != "2026-11-18" || h.Hired.TrainingMin != 2400 || h.Hired.RecruitingFeeCents != 300_000 {
		t.Fatalf("recruiting: %+v %+v", h, h.Hired)
	}
	fee := entries(w, "recruiting", 8)
	if len(fee) != 1 || lineAmt(fee[0], "recruiting") != 300_000 || lineAmt(fee[0], "accounts_payable") != -300_000 {
		t.Fatalf("recruiting fee entries %+v", fee)
	}
	cal := w.Calendar()
	staffedUntil := func(w *sim.World, day int64) simtime.Duration {
		s := w.Schedule(day)
		var end simtime.Duration
		for _, r := range s.Rooms {
			if r.Room == "R3" && len(r.Staffed) > 0 {
				end = r.Staffed[len(r.Staffed)-1].To
			}
		}
		return end
	}
	trainingMon, activeMon := dayOf(w, "2026-11-16"), dayOf(w, "2026-11-23")
	for _, d := range []int64{trainingMon, activeMon} {
		if got, want := staffedUntil(w, d), staffedUntil(base, d); d == trainingMon && got != want {
			t.Fatalf("training Monday: CT staffed until %s, without the hire %s", got, want)
		} else if d == activeMon && (got != 20*simtime.Hour || want >= got) {
			t.Fatalf("active Monday: CT staffed until %s (without the hire %s)", got, want)
		}
	}
	// On the training Monday evening the trainee is on duty, paid, and
	// staffs nothing.
	runChecked(t, w, cal.At(trainingMon, 17*simtime.Hour))
	h = employee(w, "H00001")
	if h.Employment != api.EmploymentOnboarding || h.Status != api.StaffTraining || len(h.Rooms) != 0 || h.Hired.TrainedMin != 30*60 {
		t.Fatalf("training: %+v %+v", h, h.Hired)
	}
	for _, r := range w.Observe(0).Rooms {
		if slices.Contains(r.StaffBy, "H00001") {
			t.Fatalf("trainee staffs %s", r.ID)
		}
	}
	runChecked(t, w, cal.At(activeMon, 17*simtime.Hour))
	h = employee(w, "H00001")
	if h.Employment != api.EmploymentActive || h.Hired.TrainedMin != 45*60 || !slices.Contains(h.Rooms, "R3") {
		t.Fatalf("active: %+v %+v", h, h.Hired)
	}
	if err := w.Check(); err != nil {
		t.Fatal(err)
	}
}

// §5.4's partial-week example: C = 40 h at $40, employed Wednesday to
// Sunday, 16 hours worked: guarantee 40 × 5/7 h = $1,142.85 (per-day cent
// truncation), worked $640 replaces part of it, top-up $502.85; benefits
// $285.71; total $1,428.56. The next full week guarantees r × C less at
// most a cent a day.
func TestHirePartialWeekGuarantee(t *testing.T) {
	wk := weekOf(shiftAt("R3", "07:00", "15:30", "11:00", "11:30"), "Mon", "Wed", "Thu")
	w := atTakeover(t, `{}`, 2)
	mustApply(t, w, hireOf("ct-part", "technologist", []string{"CT"}, 4000, 2400, wk, "2026-11-11"))
	wed := dayOf(w, "2026-11-11")
	w.Engine().RunUntil(w.Calendar().StartOfDay(wed+12), nil)
	week := func(from, to int64) (wages, benefits int64) {
		for _, kind := range []string{"wages", "guarantee"} {
			for _, e := range entries(w, kind, 8) {
				if d := w.Calendar().DayIndex(e.At); d >= from && d < to {
					wages += lineAmt(e, "payroll")
					benefits += lineAmt(e, "benefits")
				}
			}
		}
		return wages, benefits
	}
	if wg, b := week(wed, wed+5); wg != 114_285 || b != 28_571 {
		t.Fatalf("partial week: wages %d, benefits %d; want 114285, 28571", wg, b)
	}
	if wg, b := week(wed+5, wed+12); wg != 159_999 || b != 40_000 {
		t.Fatalf("full week: wages %d, benefits %d; want 159999, 40000", wg, b)
	}
	if err := w.Check(); err != nil {
		t.Fatal(err)
	}
}

// Notice and severance (§5.4's example): Sam contracted for 40 h with no
// roster at all. Terminated with last day Sun Nov 1: the contract pays
// every notice week in full (r × C, a cent a day of truncation), the last
// week closes on the last day, nothing accrues after it, and severance —
// two contracted weeks with benefits, $4,000, recognized at acceptance —
// is paid on the first payday after the last day (Fri Nov 13). A zero
// roster never removes either.
func TestTerminationNoticeAndSeverance(t *testing.T) {
	w := atTakeover(t, `{"employees": [{"id": "E04", "contractedWeeklyMinutes": 2400, "roster": {"Tue": null, "Wed": null, "Thu": null, "Fri": null, "Sat": null}}]}`, 3)
	w.Engine().CheckInvariants = true
	res := mustApply(t, w, terminateOf("E04", "2026-11-01"))
	if res[0].EffectiveAt != "2026-11-02" {
		t.Fatalf("termination: %+v", res[0])
	}
	sev := entries(w, "severance", 3)
	if len(sev) != 1 || lineAmt(sev[0], "severance") != 400_000 || lineAmt(sev[0], "severance_payable") != -400_000 {
		t.Fatalf("severance entries %+v", sev)
	}
	sam := employee(w, "E04")
	if sam.Employment != api.EmploymentNotice || sam.Leaving == nil || sam.Leaving.SeverancePayday != "2026-11-13" || sam.Leaving.SeveranceCents != 400_000 {
		t.Fatalf("on notice: %+v %+v", sam, sam.Leaving)
	}
	runChecked(t, w, w.StartTime().Add(40*simtime.Day))
	mon := dayOf(w, "2026-10-19")
	for wk := int64(0); wk < 2; wk++ {
		worked, guarantee := weekLabor(w, samIndex, mon+7*wk)
		if worked != 0 || guarantee != 159_999 {
			t.Fatalf("notice week %d: worked %d, guarantee %d; want 0, 159999", wk, worked, guarantee)
		}
	}
	if worked, guarantee := weekLabor(w, samIndex, mon+14); worked != 0 || guarantee != 0 {
		t.Fatalf("after the last day: worked %d, guarantee %d", worked, guarantee)
	}
	if pay := entries(w, "severance_pay", 3); len(pay) != 1 || lineAmt(pay[0], "cash") != -400_000 ||
		w.Calendar().Date(pay[0].At).String() != "2026-11-13" {
		t.Fatalf("severance payment %+v", pay)
	}
	if w.Balance("severance_payable") != 0 || w.Balance("accrued_payroll") != 0 && w.OpenGuarantee() < 0 {
		t.Fatalf("severance payable %d after payment", w.Balance("severance_payable"))
	}
	if sam = employee(w, "E04"); sam.Employment != api.EmploymentTerminated || !sam.Leaving.SeverancePaid {
		t.Fatalf("after: %+v", sam)
	}
}

// Every rejection: its code, nothing committed, the action recorded.
func TestStaffingRejections(t *testing.T) {
	day := shiftAt("R3", "07:00", "15:30", "11:00", "11:30")
	five := weekOf(day, weekdays...)
	ct := func(mut func(*api.Action)) api.Action {
		a := hireOf("r", "technologist", []string{"CT"}, 4000, 2400, five, "2026-11-10")
		mut(&a)
		return a
	}
	roster := func(wk api.Week, minutes int64) func(*api.Action) {
		return func(a *api.Action) { a.Roster, a.ContractedWeeklyMinutes = &api.Roster{A: wk, B: wk}, minutes }
	}
	long := weekOf(shiftAt("R3", "07:00", "15:00"), weekdays...)
	noBreak := weekOf(shiftAt("R3", "07:00", "14:00", "13:30", "14:00"), weekdays...)
	_ = noBreak
	seven := weekOf(shiftAt("R3", "08:00", "12:00"), "Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun")
	tight := api.Week{Mon: shiftAt("R3", "12:00", "16:00"), Tue: shiftAt("R3", "05:00", "09:00")}
	cover := *day
	cover.Covers = []api.Cover{{Room: "R4", From: "08:00", Until: "10:00"}}
	sparse := weekOf(shiftAt("R3", "08:00", "12:00"), "Mon")
	for _, c := range []struct {
		name string
		a    api.Action
		code string
	}{
		{"schema", ct(func(a *api.Action) { a.Minutes = 5 }), "invalid_schema"},
		{"no request", ct(func(a *api.Action) { a.RequestID = "" }), "invalid_schema"},
		{"role", ct(func(a *api.Action) { a.Role = "radiologist" }), "wrong_role"},
		{"skills", ct(func(a *api.Action) { a.Skills = []string{"CT", "MR"} }), "invalid_skills"},
		{"duplicate skills", ct(func(a *api.Action) { a.Skills = []string{"CT", "CT"} }), "invalid_skills"},
		{"date", ct(func(a *api.Action) { a.StartDate = "2026-11-31" }), "invalid_date"},
		{"past", ct(func(a *api.Action) { a.StartDate = "2026-10-13" }), "past_date"},
		{"horizon", ct(func(a *api.Action) { a.StartDate = "2027-03-01" }), "beyond_horizon"},
		{"rate", ct(func(a *api.Action) { a.RateCents = 4100 }), "rate_out_of_band"},
		{"contract", ct(func(a *api.Action) { a.ContractedWeeklyMinutes = 2401 }), "invalid_contract_hours"},
		{"lead", ct(func(a *api.Action) { a.StartDate = "2026-11-09" }), "lead_too_short"},
		{"premium lead", ct(func(a *api.Action) { a.RateCents, a.StartDate = 4400, "2026-11-02" }), "lead_too_short"},
		{"no break", ct(roster(long, 2400)), "invalid_roster"},
		{"seven days", ct(roster(seven, 2400)), "invalid_roster"},
		{"over contract", ct(func(a *api.Action) { a.ContractedWeeklyMinutes = 2399 }), "invalid_roster"},
		{"rest", ct(roster(tight, 2400)), "invalid_roster"},
		{"unqualified cover", ct(roster(weekOf(&cover, weekdays...), 2400)), "invalid_roster"},
		{"desk with a room", api.Action{Type: api.ActHire, RequestID: "d", Role: "front_desk", Skills: []string{"registration"},
			RateCents: 2100, ContractedWeeklyMinutes: 2400, Roster: &api.Roster{A: five, B: five}, StartDate: "2026-11-10"}, "invalid_roster"},
		{"training", ct(roster(sparse, 2400)), "training_incomplete"},
		{"terminate schema", api.Action{Type: api.ActTerminate, Employee: "E04"}, "invalid_schema"},
		{"unknown", terminateOf("E99", "2026-11-01"), "unknown_employee"},
		{"radiologist", terminateOf("E07", "2026-11-01"), "wrong_role"},
		{"ops manager", terminateOf("E02", "2026-11-01"), "wrong_role"},
		{"notice", terminateOf("E04", "2026-10-26"), "notice_too_short"},
		{"last day", terminateOf("E04", "26 Oct"), "invalid_date"},
		{"last day horizon", terminateOf("E04", "2027-03-01"), "beyond_horizon"},
	} {
		w := atTakeover(t, `{}`, 1)
		ledger, log, n := w.LedgerHash(), w.LogHash(), w.RegistryLen()
		r := w.Apply([]api.Action{c.a})[0]
		if r.OK || r.Code != c.code {
			t.Errorf("%s: %+v, want %s", c.name, r, c.code)
		}
		if w.LedgerHash() != ledger || w.LogHash() != log || w.RegistryLen() != n || len(w.ActionLog()) != 1 {
			t.Errorf("%s: a refusal changed the run", c.name)
		}
	}
}

// Identical repeats are no-ops (even after the hire left); a reused
// request with other terms, or a moved last day, is refused.
func TestStaffingRepeats(t *testing.T) {
	w := atTakeover(t, `{}`, 4)
	five := weekOf(shiftAt("R3", "07:00", "15:30", "11:00", "11:30"), weekdays...)
	h := hireOf("dup", "technologist", []string{"CT", "XR"}, 3800, 2400, five, "2026-11-10")
	mustApply(t, w, h)
	again := h
	again.Skills = []string{"XR", "CT"} // canonically the same
	ledger := w.LedgerHash()
	if r := w.Apply([]api.Action{again})[0]; !r.OK || r.Code != api.ActionUnchanged || w.RegistryLen() != 9 || w.LedgerHash() != ledger {
		t.Fatalf("repeat: %+v", r)
	}
	other := h
	other.StartDate = "2026-11-11"
	if r := w.Apply([]api.Action{other})[0]; r.Code != "request_conflict" {
		t.Fatalf("conflict: %+v", r)
	}
	mustApply(t, w, terminateOf("H00001", "2026-11-24"))
	if r := w.Apply([]api.Action{terminateOf("H00001", "2026-11-24")})[0]; r.Code != api.ActionUnchanged {
		t.Fatalf("same notice: %+v", r)
	}
	if r := w.Apply([]api.Action{terminateOf("H00001", "2026-11-25")})[0]; r.Code != "already_terminating" {
		t.Fatalf("moved notice: %+v", r)
	}
	w.Engine().RunUntil(w.StartTime().Add(50*simtime.Day), nil)
	if r := w.Apply([]api.Action{h})[0]; r.Code != api.ActionUnchanged {
		t.Fatalf("repeat after departure: %+v", r)
	}
	if r := w.Apply([]api.Action{terminateOf("H00001", "2026-12-31")})[0]; r.Code != "not_employed" {
		t.Fatalf("after departure: %+v", r)
	}
	// A pending hire terminated before its start owes the fee (never
	// refunded), the contract from the start and severance.
	if e := employee(w, "H00001"); e.Employment != api.EmploymentTerminated || e.Leaving.SeveranceCents != 2*152_000*125/100 {
		t.Fatalf("pending hire terminated: %+v %+v", e, e.Leaving)
	}
}

// A V0.2 batch is atomic: one refusal refuses the batch and commits
// nothing (no ID, posting, log line or event); legacy actions cannot join it.
func TestStaffingAtomicBatch(t *testing.T) {
	five := weekOf(shiftAt("R3", "07:00", "15:30", "11:00", "11:30"), weekdays...)
	good := hireOf("a", "technologist", []string{"CT"}, 4000, 2400, five, "2026-11-10")
	for _, c := range []struct {
		name  string
		batch []api.Action
		fail  int
		code  string
	}{
		{"bad terminate", []api.Action{good, terminateOf("E04", "2026-10-20")}, 1, "notice_too_short"},
		{"mixed", []api.Action{good, {Type: api.ActSetWorklistPolicy, Policy: api.WorklistFIFO}}, 1, "mixed_batch"},
		{"duplicate request", []api.Action{good, good}, 1, "invalid_parameters"},
		{"terminated twice", []api.Action{terminateOf("E04", "2026-11-01"), terminateOf("E04", "2026-11-01")}, 1, "invalid_parameters"},
	} {
		w := atTakeover(t, `{}`, 5)
		before := []string{w.StateDigest(), w.LedgerHash(), w.LogHash()}
		res := w.Apply(c.batch)
		for k, r := range res {
			want := "batch_rejected"
			if k == c.fail {
				want = c.code
			}
			if r.OK || r.Code != want || r.EmployeeID != "" {
				t.Errorf("%s: item %d %+v, want %s", c.name, k, r, want)
			}
		}
		after := []string{w.StateDigest(), w.LedgerHash(), w.LogHash()}
		if after[1] != before[1] || after[2] != before[2] || w.RegistryLen() != 8 {
			t.Errorf("%s: a refused batch changed the run", c.name)
		}
	}
	// Valid: committed together, IDs in input order.
	w := atTakeover(t, `{}`, 5)
	second := good
	second.RequestID = "b"
	res := mustApply(t, w, good, terminateOf("E04", "2026-11-01"), second)
	if res[0].EmployeeID != "H00001" || res[2].EmployeeID != "H00002" || res[1].EffectiveAt != "2026-11-02" {
		t.Fatalf("batch: %+v", res)
	}
}

// Concurrent caps: recruiting, onboarding, active and notice count; a
// departed employee does not. The established practice has 4 technologists.
func TestStaffingHeadcountCap(t *testing.T) {
	w := atTakeover(t, `{}`, 6)
	five := weekOf(shiftAt("R1", "07:00", "15:30", "11:00", "11:30"), weekdays...)
	hire := func(n int) api.Action {
		return hireOf(strings.Repeat("x", n), "technologist", []string{"XR"}, 3600, 2400, five, "2026-11-10")
	}
	for n := 1; n <= 4; n++ {
		mustApply(t, w, hire(n))
	}
	if r := w.Apply([]api.Action{hire(5)})[0]; r.Code != "headcount_limit" {
		t.Fatalf("ninth technologist: %+v", r)
	}
	mustApply(t, w, terminateOf("E06", "2026-10-28"))
	if r := w.Apply([]api.Action{hire(5)})[0]; r.Code != "headcount_limit" {
		t.Fatalf("on notice still counts: %+v", r)
	}
	w.Engine().RunUntil(w.Calendar().StartOfDay(dayOf(w, "2026-10-29")), nil)
	hire5 := hire(5)
	hire5.StartDate = "2026-11-26"
	if r := w.Apply([]api.Action{hire5})[0]; !r.OK {
		t.Fatalf("after departure: %+v", r)
	}
	if r := w.Apply([]api.Action{hireOf("desk", "front_desk", []string{"registration"}, 2400, 2400,
		weekOf(shiftAt("desk", "07:45", "16:15", "12:00", "12:30"), weekdays...), "2026-11-05")})[0]; !r.OK {
		t.Fatalf("desk hire (premium, 7-day lead): %+v", r)
	}
}

// Terminating Sam (E04) displaces the bookings his CT evenings and
// Saturdays covered — never silently: each is released, recorded and
// offered a replacement at or after its original time (same behavioral
// attempt, no new random draw), or lost to displacement; the retained
// schedule is exactly feasible; the preview named the same appointments.
func TestTerminationDisplacement(t *testing.T) {
	w := atTakeover(t, `{}`, 1)
	w.Engine().CheckInvariants = true
	act := terminateOf("E04", "2026-10-27")
	prev := w.PreviewStaffing([]api.Action{act})
	if !prev.OK || len(prev.Terminations) != 1 || len(prev.Displacements) == 0 {
		t.Fatalf("preview: %+v", prev)
	}
	o0 := w.Observe(0)
	before := map[string]sim.ApptInfo{}
	for _, a := range w.Appointments() {
		before[a.ID] = a
	}
	mustApply(t, w, act)
	o1 := w.Observe(0)
	if !slices.Equal(o1.Demand.Modalities, o0.Demand.Modalities) {
		t.Fatalf("displacement changed demand counters: %+v → %+v", o0.Demand.Modalities, o1.Demand.Modalities)
	}
	recs := w.Displacements()
	var previewed, displaced []string
	for _, d := range prev.Displacements {
		previewed = append(previewed, d.Appointment)
	}
	appts := map[string]sim.ApptInfo{}
	for _, a := range w.Appointments() {
		appts[a.ID] = a
	}
	rebooked, lost := 0, 0
	for _, d := range recs {
		displaced = append(displaced, d.Appt)
		old := appts[d.Appt]
		if old.State != "displaced" || before[d.Appt].State != "booked" || d.Cause != "employment_ended" || len(d.Actions) != 1 {
			t.Fatalf("record %+v, appointment %+v", d, old)
		}
		switch d.Resolution {
		case "rebooked":
			rebooked++
			n := appts[d.Replacement]
			if n.Order != old.Order || n.Attempt != old.Attempt || n.Placement != 1 || n.NoShowLead != old.NoShowLead ||
				n.Confirmed != old.Confirmed || w.Calendar().At(n.Day, n.Span.From) < w.Calendar().At(old.Day, old.Span.From) {
				t.Fatalf("replacement %+v for %+v", n, old)
			}
			anchor, tol := w.OrderAnchor(old.Order)
			if n.Day-anchor > tol {
				t.Fatalf("replacement on day %d past anchor %d + tolerance %d", n.Day, anchor, tol)
			}
		case "replacement_declined", "no_feasible_slot":
			lost++
			for _, o := range w.Orders() {
				if o.ID == old.Order && o.State != "displaced-lost" {
					t.Fatalf("lost order %+v", o)
				}
			}
		default:
			t.Fatalf("resolution %q", d.Resolution)
		}
	}
	slices.Sort(previewed)
	slices.Sort(displaced)
	if !slices.Equal(previewed, displaced) {
		t.Fatalf("preview named %d appointments, commit displaced %d", len(previewed), len(displaced))
	}
	m := w.Evaluation().Ops.Displacement
	if m == nil || m.Events != len(recs) || m.Rebooked != rebooked || m.Declined+m.NoFeasibleSlot != lost || m.Orders+m.Repeated != len(recs) {
		t.Fatalf("metrics %+v for %d records (%d rebooked, %d lost)", m, len(recs), rebooked, lost)
	}
	// Same seed, same decision: the same displacements.
	w2 := atTakeover(t, `{}`, 1)
	mustApply(t, w2, act)
	if w2.StateDigest() != w.StateDigest() || w2.LogHash() != w.LogHash() {
		t.Fatal("displacement is not deterministic")
	}
	runChecked(t, w, w.StartTime().Add(21*simtime.Day))
}

// Atomic composition: terminating Sam and hiring a replacement on his
// roster (XR+CT, premium lead, independent the day after his last day) in
// one batch displaces nobody; the termination alone displaces dozens.
func TestReplacementInSameBatchRetainsBookings(t *testing.T) {
	sam := func(day string) *api.Shift {
		if day == "Sat" {
			s := shiftAt("R3", "10:00", "18:00", "13:30", "14:00")
			s.Covers = []api.Cover{{Room: "R1", From: "10:00", Until: "18:00"}}
			return s
		}
		s := shiftAt("R3", "11:30", "20:00", "15:30", "16:00")
		s.Covers = []api.Cover{{Room: "R1", From: "16:15", Until: "20:00"}}
		return s
	}
	wk := api.Week{Tue: sam("Tue"), Wed: sam("Wed"), Thu: sam("Thu"), Fri: sam("Fri"), Sat: sam("Sat")}
	repl := hireOf("sam2", "technologist", []string{"XR", "CT"}, 4200, 2370, wk, "2026-11-03")
	term := terminateOf("E04", "2026-11-10")
	w := atTakeover(t, `{}`, 1)
	alone := w.PreviewStaffing([]api.Action{term})
	both := w.PreviewStaffing([]api.Action{term, repl})
	if !both.OK || len(both.Displacements) != 0 || len(alone.Displacements) < 10 || both.Hires[0].ProductiveDate != "2026-11-11" {
		t.Fatalf("alone %d displaced, with the replacement %d (%+v)", len(alone.Displacements), len(both.Displacements), both.Hires)
	}
	mustApply(t, w, term, repl)
	if n := len(w.Displacements()); n != 0 {
		t.Fatalf("%d displaced", n)
	}
}

// Previews are pure: any number of them leaves every digest as it was, and
// one made on an identical past with a different hidden future is the same.
func TestStaffingPreviewReadOnly(t *testing.T) {
	five := weekOf(shiftAt("R3", "07:00", "15:30", "11:00", "11:30"), weekdays...)
	batch := []api.Action{terminateOf("E04", "2026-10-27"), hireOf("p", "technologist", []string{"CT"}, 4000, 2400, five, "2026-11-10")}
	a, b := atTakeover(t, `{}`, 7), atTakeover(t, `{}`, 7)
	b.DropOrdersFrom(b.StartTime().Add(simtime.Hour)) // another hidden future
	digests := func(w *sim.World) []string {
		return []string{w.StateDigest(), w.LogHash(), w.LedgerHash(), w.ActionDigest()}
	}
	before := digests(a)
	pa := a.PreviewStaffing(batch)
	for range 3 {
		a.PreviewStaffing(batch)
		a.ValidateBatch(batch)
	}
	if !slices.Equal(before, digests(a)) {
		t.Fatal("previews changed the run")
	}
	ja, _ := json.Marshal(pa)
	jb, _ := json.Marshal(b.PreviewStaffing(batch))
	if string(ja) != string(jb) {
		t.Fatal("the preview depends on the hidden future")
	}
	if !pa.OK || len(pa.Guaranteed) < 5 || len(pa.Forecast) == 0 || pa.Results[1].EmployeeID != "" {
		t.Fatalf("preview %+v", pa)
	}
}

// More than 64 employees: slot staffing sets grow past one word, a
// scenario above the technologist cap loads and runs, and hires of that
// role are refused until it is below the cap.
func TestStaffBeyond64(t *testing.T) {
	var extra []string
	for k := 0; k < 66; k++ {
		extra = append(extra, `{"id": "T`+pad3(k)+`", "name": "Tech `+pad3(k)+`", "role": "technologist", "skills": ["XR"], "pay": {"type": "hourly", "rateCents": 3600},
			"roster": {"Mon": {"start": "08:00", "end": "16:30", "breaks": [["12:00", "12:30"]], "assignment": "R1"}}}`)
	}
	w := atTakeover(t, `{"employees": [`+strings.Join(extra, ",")+`]}`, 1)
	w.Engine().CheckInvariants = true
	if w.RegistryLen() != 74 || w.SlotStaffWords() < 2 {
		t.Fatalf("registry %d, slot staff words %d", w.RegistryLen(), w.SlotStaffWords())
	}
	five := weekOf(shiftAt("R1", "07:00", "15:30", "11:00", "11:30"), weekdays...)
	if r := w.Apply([]api.Action{hireOf("x", "technologist", []string{"XR"}, 3600, 2400, five, "2026-11-10")})[0]; r.Code != "headcount_limit" {
		t.Fatalf("hire above the cap: %+v", r)
	}
	mustApply(t, w, hireOf("d", "front_desk", []string{"registration"}, 2100, 2400,
		weekOf(shiftAt("desk", "07:45", "16:15", "12:00", "12:30"), weekdays...), "2026-10-27"), terminateOf("T065", "2026-10-28"))
	runChecked(t, w, w.StartTime().Add(18*simtime.Day))
}

func pad3(k int) string {
	return string([]byte{byte('0' + k/100), byte('0' + k/10%10), byte('0' + k%10)})
}

// Across the evaluation endpoint (here 20 days): a termination accepted in
// the window with its last day and severance payday after the endpoint
// counts its severance in the window as recognized and unpaid; the cash
// bridge reconciles with the payable's change; the frozen window does not
// move when severance is later paid; a hire's fee is in the window, its
// payment after it.
func TestStaffingAcrossEvaluationEndpoint(t *testing.T) {
	w := atTakeover(t, `{"evaluationDays": 20}`, 8)
	five := weekOf(shiftAt("R3", "07:00", "15:30", "11:00", "11:30"), weekdays...)
	mustApply(t, w, terminateOf("E04", "2026-11-05"), hireOf("e", "technologist", []string{"CT"}, 4000, 2400, five, "2026-11-10"))
	w.Engine().RunUntil(w.StartTime().Add(20*simtime.Day), nil)
	m := w.Evaluation().Money
	if !m.Reconciled || m.SeverancePayableChangeCents != 395_000 || m.Ledger.SeveranceCents != 395_000 || m.Ledger.RecruitingCents != 300_000 {
		t.Fatalf("at the endpoint: %+v", m)
	}
	w.Engine().RunUntil(w.StartTime().Add(40*simtime.Day), nil)
	later := w.Evaluation().Money
	later.Ledger.To = m.Ledger.To // the statement's label is the date it is viewed (V0.1)
	ja, _ := json.Marshal(m)
	jb, _ := json.Marshal(later)
	if string(ja) != string(jb) {
		t.Fatalf("the window moved after its endpoint:\n%s\n%s", ja, jb)
	}
	if w.Balance("severance_payable") != 0 {
		t.Fatal("severance not paid by day 40")
	}
	if err := w.Check(); err != nil {
		t.Fatal(err)
	}
}

// Shared-technologist displacement: Riley covers CT all day as well as
// X-ray, so with Sam gone, afternoon X-ray and CT bookings that each had
// their own technologist now compete for Riley. Survivor order keeps the
// most urgent; the rest are displaced as shared_capacity, and what is
// retained is exactly feasible (the invariants re-check every booking).
func TestTerminationSharedCapacityDisplacement(t *testing.T) {
	cover := `{"covers": [{"room": "R3"}]}`
	w := atTakeover(t, `{"employees": [{"id": "E03", "roster": {"Tue": `+cover+`, "Wed": `+cover+`, "Thu": `+cover+`, "Fri": `+cover+`}}]}`, 1)
	w.Engine().CheckInvariants = true
	act := terminateOf("E04", "2026-10-27")
	p := w.PreviewStaffing([]api.Action{act})
	reasons := map[string]int{}
	for _, d := range p.Displacements {
		reasons[d.Reason]++
	}
	if reasons["shared_capacity"] == 0 {
		t.Fatalf("reasons %v: expected shared-capacity displacements", reasons)
	}
	mustApply(t, w, act)
	got := map[string]int{}
	for _, d := range w.Displacements() {
		got[d.Reason]++
	}
	if got["shared_capacity"] != reasons["shared_capacity"] || got["unstaffed"] != reasons["unstaffed"] {
		t.Fatalf("committed %v, previewed %v", got, reasons)
	}
	runChecked(t, w, w.StartTime().Add(16*simtime.Day))
}

// Over several seeds and terminations, every accepted replacement is
// within the patient's tolerance of the tolerance anchor — never of the
// displaced appointment's own day — and some offers are declined.
func TestDisplacementToleranceAnchor(t *testing.T) {
	if testing.Short() {
		t.Skip("12 terminations")
	}
	declined, rebooked := 0, 0
	for _, seed := range []int64{1, 2, 3, 4} {
		for _, id := range []string{"E04", "E05", "E06"} {
			w := atTakeover(t, `{}`, seed)
			mustApply(t, w, terminateOf(id, "2026-10-27"))
			appts := map[string]sim.ApptInfo{}
			for _, a := range w.Appointments() {
				appts[a.ID] = a
			}
			for _, d := range w.Displacements() {
				switch d.Resolution {
				case "replacement_declined":
					declined++
				case "rebooked":
					rebooked++
					n := appts[d.Replacement]
					if anchor, tol := w.OrderAnchor(n.Order); n.Day-anchor > tol {
						t.Fatalf("seed %d, %s: %s rebooked on day %d, anchor %d + tolerance %d", seed, id, d.Appt, n.Day, anchor, tol)
					}
				}
			}
			if err := w.Check(); err != nil {
				t.Fatal(err)
			}
		}
	}
	if declined == 0 || rebooked == 0 {
		t.Fatalf("%d rebooked, %d declined: the acceptance rule is not exercised", rebooked, declined)
	}
}

// Preview of a termination at the takeover: validation plus the whole
// booked horizon re-planned, survivors and provisional replacements.
func BenchmarkStaffingPreviewTermination(b *testing.B) {
	w := sim.New(established(b), rng.SeedFromInt(1))
	w.Engine().RunUntil(w.StartTime(), nil)
	act := []api.Action{terminateOf("E04", "2026-10-27")}
	b.ResetTimer()
	for range b.N {
		w.PreviewStaffing(act)
	}
}

// Committing that termination: severance, reconciliation of the booked
// horizon and the replacement offers (world building excluded).
func BenchmarkStaffingCommitTermination(b *testing.B) {
	cfg := established(b)
	for range b.N {
		b.StopTimer()
		w := sim.New(cfg, rng.SeedFromInt(1))
		w.Engine().RunUntil(w.StartTime(), nil)
		b.StartTimer()
		w.Apply([]api.Action{terminateOf("E04", "2026-10-27")})
	}
}
