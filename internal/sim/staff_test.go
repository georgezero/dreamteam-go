package sim_test

import (
	"slices"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/rng"
	"github.com/georgezero/dreamteam-go/internal/scenario"
	"github.com/georgezero/dreamteam-go/internal/sim"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

func miniConfig(t testing.TB) *sim.Config {
	t.Helper()
	cfg, err := scenario.Resolve("mini")
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

// rosterOnly is the mini scenario with no patients at all, so staffing
// follows the roster exactly (no tasks, deferred breaks or hold-overs).
func rosterOnly(t testing.TB) *sim.Config {
	cfg := *miniConfig(t)
	cfg.Opening = nil
	cfg.Sources = append([]sim.SourceConfig(nil), cfg.Sources...)
	for i := range cfg.Sources {
		cfg.Sources[i].BaseDailyMilli = 0
	}
	cfg.WalkIns.DailyMilli = 0
	return &cfg
}

func TestReadingBlocks(t *testing.T) {
	h := simtime.Hour
	m := simtime.Minute
	got := sim.ReadingBlocks(sim.Span{From: 7*h + 30*m, To: 16 * h}, 6000)
	want := []sim.Span{
		{From: 7*h + 30*m, To: 8*h + 30*m}, {From: 9*h + 10*m, To: 10*h + 10*m},
		{From: 10*h + 50*m, To: 11*h + 50*m}, {From: 12*h + 30*m, To: 13*h + 30*m},
		{From: 14*h + 10*m, To: 15*h + 10*m}, {From: 15*h + 50*m, To: 16 * h},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("blocks = %v", got)
	}
	if sim.ReadingBlocks(sim.Span{From: 0, To: h}, 0) != nil {
		t.Fatal("zero allocation has blocks")
	}
}

var deskByLevel = map[int]string{
	sim.DeskClosed: api.DeskClosed, sim.DeskSelf: api.DeskSelf,
	sim.DeskOps: api.DeskOps, sim.DeskStaffed: api.DeskStaffed,
}

// The running simulation must agree, minute by minute, with the static
// coverage plan the report is built from: room states, who staffs each
// room, desk coverage and radiologist reading blocks. Covers a whole
// fortnight, so Mon/Tue/Sat of both rota weeks are included.
func TestRuntimeMatchesPlan(t *testing.T) {
	cfg := rosterOnly(t)
	w := sim.New(cfg, rng.SeedFromInt(1))
	eng := w.Engine()
	eng.CheckInvariants = true
	cal := w.Calendar()
	start := w.StartTime()
	end := cal.StartOfDate(cfg.Start.AddDays(14))
	eng.RunUntil(start, nil)

	var plan sim.DayPlan
	var cov sim.Coverage
	planDay := int64(-1)
	checked := 0
	for now := start; now < end; now = now.Add(simtime.Minute) {
		eng.RunUntil(now, nil)
		if d := cal.DayIndex(now); d != planDay {
			planDay = d
			plan = cfg.Plan(cal.Date(now))
			cov = plan.Coverage()
		}
		clock := simtime.ClockOf(now)
		var seg *sim.Segment
		for i := range cov.Segments {
			if cov.Segments[i].Contains(clock) {
				seg = &cov.Segments[i]
			}
		}
		o := w.Observe(0)
		if o.Practice.Open != seg.Open {
			t.Fatalf("%s: open=%v, plan %v", cal.Short(now), o.Practice.Open, seg.Open)
		}
		for r, room := range o.Rooms {
			var want []string
			for _, e := range seg.Rooms[r] {
				want = append(want, cfg.Employees[e].ID)
			}
			state := api.RoomClosed
			if seg.Open {
				state = api.RoomUnstaffed
				if len(want) > 0 {
					state = api.RoomIdle
				}
			}
			if room.State != state || !slices.Equal(room.StaffBy, want) {
				t.Fatalf("%s %s: %s %v, plan %s %v", cal.Short(now), room.Modality, room.State, room.StaffBy, state, want)
			}
		}
		if o.Desk.Coverage != deskByLevel[seg.Desk] {
			t.Fatalf("%s desk: %s, plan %s", cal.Short(now), o.Desk.Coverage, deskByLevel[seg.Desk])
		}
		reading := 0
		for _, e := range o.Staff {
			if e.Status == api.StaffReading {
				reading++
			}
		}
		if reading != seg.Reading {
			t.Fatalf("%s: %d reading, plan %d", cal.Short(now), reading, seg.Reading)
		}
		checked++
	}
	if checked != 14*24*60-6*60 {
		t.Fatalf("checked %d minutes", checked)
	}
}

// Advancing minute by minute and in one jump give the same log.
func TestStaffChunkingInvariance(t *testing.T) {
	cfg := rosterOnly(t)
	a := sim.New(cfg, rng.SeedFromInt(1))
	b := sim.New(cfg, rng.SeedFromInt(1))
	end := a.Calendar().StartOfDate(cfg.Start.AddDays(7))
	a.Engine().RunUntil(end, nil)
	for now := simtime.Time(0); now < end; now = now.Add(7 * simtime.Minute) {
		b.Engine().RunUntil(now, nil)
	}
	b.Engine().RunUntil(end, nil)
	if a.LogHash() != b.LogHash() || a.LogTotal() != b.LogTotal() || a.LogTotal() < 200 {
		t.Fatalf("logs differ or too short: %d vs %d entries", a.LogTotal(), b.LogTotal())
	}
}
