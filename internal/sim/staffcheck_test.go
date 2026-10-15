package sim_test

import (
	"slices"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/rng"
	"github.com/georgezero/dreamteam-go/internal/sim"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// V0.2 Slice 1: whole-schedule staffing feasibility (V0.2_DESIGN §7.1),
// read-only. CheckStaffing rebuilds every materialized future day's
// coverage under a hypothetical roster change and re-books its existing
// appointments in survivor order (priority, start, confirmation,
// appointment) against only the retained ones, with the booking rule's
// own predicates. It reports the appointments that would lose qualified
// coverage and changes nothing.

func establishedWorld(t *testing.T, seed int64) *sim.World {
	t.Helper()
	return sim.New(established(t), rng.SeedFromInt(seed))
}

// rosterOf returns an employee's current weekly roster.
func rosterOf(w *sim.World, id string) [7]*sim.ShiftTemplate {
	for _, e := range w.Config().Employees {
		if e.ID == id {
			return e.Roster
		}
	}
	panic("no employee " + id)
}

// With no change, every booked future appointment keeps its coverage: the
// booked schedule is consistent with the roster it was booked under.
func TestStaffingCheckUnchangedSchedule(t *testing.T) {
	for _, seed := range []int64{1, 2, 3} {
		w := establishedWorld(t, seed)
		for _, at := range []simtime.Duration{0, 30 * simtime.Day, 75 * simtime.Day} {
			w.Engine().RunUntil(w.StartTime().Add(at), nil)
			all := map[string][7]*sim.ShiftTemplate{}
			for _, e := range w.Config().Employees {
				all[e.ID] = e.Roster
			}
			for _, ch := range []sim.StaffingChange{{From: w.DayIndex() + 1}, {From: w.DayIndex() + 1, Rosters: all}} {
				got, err := w.CheckStaffing(ch)
				if err != nil {
					t.Fatal(err)
				}
				if len(got) != 0 {
					t.Fatalf("seed %d at +%s: %d conflicts with the unchanged roster, first %+v", seed, at, len(got), got[0])
				}
			}
		}
	}
}

// Removing Sam (E04) from the roster: the CT afternoons and evenings he
// staffs alone lose coverage. Every conflict is a booked appointment from
// the change on, reported once, in a room Sam staffs.
func TestStaffingCheckFindsLostCoverage(t *testing.T) {
	w := establishedWorld(t, 1)
	w.Engine().RunUntil(w.StartTime().Add(2*simtime.Hour), nil)
	from := w.DayIndex() + 7
	got, err := w.CheckStaffing(sim.StaffingChange{From: from, Rosters: map[string][7]*sim.ShiftTemplate{"E04": {}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) == 0 {
		t.Fatal("no conflicts after removing Sam's roster")
	}
	booked := map[string]sim.ApptInfo{}
	for _, a := range w.Appointments() {
		booked[a.ID] = a
	}
	seen := map[string]bool{}
	reasons := map[string]int{}
	for _, c := range got {
		a, ok := booked[c.Appointment]
		switch {
		case !ok || a.State != "booked":
			t.Fatalf("conflict for %s, which is not a booked appointment (%+v)", c.Appointment, a)
		case seen[c.Appointment]:
			t.Fatalf("%s reported twice", c.Appointment)
		case c.Day < from:
			t.Fatalf("%s on day %d, before the change (%d)", c.Appointment, c.Day, from)
		case c.Room != "R1" && c.Room != "R3":
			t.Fatalf("%s in room %s, which Sam never staffs", c.Appointment, c.Room)
		}
		seen[c.Appointment] = true
		reasons[c.Reason]++
	}
	if reasons["unstaffed"] == 0 {
		t.Fatalf("reasons %v: expected unstaffed CT evenings", reasons)
	}
	// Deterministic.
	again, _ := w.CheckStaffing(sim.StaffingChange{From: from, Rosters: map[string][7]*sim.ShiftTemplate{"E04": {}}})
	if !slices.Equal(got, again) {
		t.Fatal("two checks of the same change differ")
	}
}

// Both rooms staffed by one technologist (Riley covering CT all day, Sam
// gone): the daytime slots stay individually bookable, but overlapping
// X-ray and CT appointments now compete for Riley.
func TestStaffingCheckSharedCapacity(t *testing.T) {
	w := establishedWorld(t, 2)
	w.Engine().RunUntil(w.StartTime().Add(2*simtime.Hour), nil)
	from := w.DayIndex() + 7
	riley := rosterOf(w, "E03")
	both := [7]*sim.ShiftTemplate{}
	for wd, s := range riley {
		if s != nil {
			c := *s
			c.Covers = []sim.Cover{{Room: "R3", Span: s.Span}}
			both[wd] = &c
		}
	}
	got, err := w.CheckStaffing(sim.StaffingChange{From: from, Rosters: map[string][7]*sim.ShiftTemplate{"E03": both, "E04": {}}})
	if err != nil {
		t.Fatal(err)
	}
	reasons := map[string]int{}
	for k, c := range got {
		reasons[c.Reason]++
		// Survivor order within a day: urgent first.
		if k > 0 && got[k-1].Day == c.Day && got[k-1].Priority < c.Priority {
			t.Fatalf("conflicts out of survivor order: %+v before %+v", got[k-1], c)
		}
	}
	if reasons["shared_capacity"] == 0 {
		t.Fatalf("reasons %v: expected shared-capacity conflicts", reasons)
	}
}

// Without on-site radiologists, contrast slots lose supervision.
func TestStaffingCheckSupervision(t *testing.T) {
	w := establishedWorld(t, 3)
	w.Engine().RunUntil(w.StartTime().Add(2*simtime.Hour), nil)
	got, err := w.CheckStaffing(sim.StaffingChange{From: w.DayIndex() + 3,
		Rosters: map[string][7]*sim.ShiftTemplate{"E07": {}, "E08": {}}})
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, c := range got {
		if c.Reason != "no_supervision" {
			t.Fatalf("removing the radiologists: %+v", c)
		}
		n++
	}
	if n == 0 {
		t.Fatal("no contrast appointment lost supervision")
	}
}

// The check is read-only: no state, log, ledger or future outcome changes.
func TestStaffingCheckReadOnly(t *testing.T) {
	a, b := establishedWorld(t, 5), establishedWorld(t, 5)
	for _, w := range []*sim.World{a, b} {
		w.Engine().RunUntil(w.StartTime().Add(3*simtime.Hour), nil)
	}
	before := a.StateDigest()
	for _, ch := range []sim.StaffingChange{
		{From: a.DayIndex() + 1, Rosters: map[string][7]*sim.ShiftTemplate{"E04": {}}},
		{From: a.DayIndex() + 2, Rosters: map[string][7]*sim.ShiftTemplate{"E03": {}, "E07": {}}},
	} {
		if _, err := a.CheckStaffing(ch); err != nil {
			t.Fatal(err)
		}
	}
	if a.StateDigest() != before {
		t.Fatal("the check changed the state")
	}
	for _, w := range []*sim.World{a, b} {
		w.Engine().RunUntil(w.StartTime().Add(10*simtime.Day), nil)
	}
	if a.StateDigest() != b.StateDigest() || a.LogHash() != b.LogHash() || a.LedgerHash() != b.LedgerHash() {
		t.Fatal("a run with checks diverged from one without")
	}
}

func TestStaffingCheckRejects(t *testing.T) {
	w := establishedWorld(t, 1)
	w.Engine().RunUntil(w.StartTime(), nil)
	for name, ch := range map[string]sim.StaffingChange{
		"today":    {From: w.DayIndex(), Rosters: map[string][7]*sim.ShiftTemplate{"E04": {}}},
		"past":     {From: w.DayIndex() - 3},
		"unknown":  {From: w.DayIndex() + 1, Rosters: map[string][7]*sim.ShiftTemplate{"E99": {}}},
		"no rooms": {From: w.DayIndex() + 1, Rosters: map[string][7]*sim.ShiftTemplate{"E04": {simtime.Tuesday: {Span: sim.Span{From: 9 * simtime.Hour, To: 17 * simtime.Hour}, Assignment: "R9"}}}},
	} {
		if _, err := w.CheckStaffing(ch); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

// BenchmarkStaffingCheck: removing Sam from tomorrow, over the whole
// booked horizon of the established practice at the takeover.
func BenchmarkStaffingCheck(b *testing.B) {
	w := sim.New(established(b), rng.SeedFromInt(1))
	w.Engine().RunUntil(w.StartTime(), nil)
	ch := sim.StaffingChange{From: w.DayIndex() + 1, Rosters: map[string][7]*sim.ShiftTemplate{"E04": {}}}
	b.ResetTimer()
	for range b.N {
		if _, err := w.CheckStaffing(ch); err != nil {
			b.Fatal(err)
		}
	}
}
