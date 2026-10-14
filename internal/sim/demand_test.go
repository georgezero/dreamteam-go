package sim_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/rng"
	"github.com/georgezero/dreamteam-go/internal/scenario"
	"github.com/georgezero/dreamteam-go/internal/sim"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// run builds a mini world and runs it for days with invariants checked
// after every transaction.
func run(t testing.TB, cfg *sim.Config, seed int64, days int64) *sim.World {
	t.Helper()
	w := sim.New(cfg, rng.SeedFromInt(seed))
	w.Engine().CheckInvariants = true
	w.Engine().RunUntil(w.Calendar().StartOfDate(cfg.Start.AddDays(days)), nil)
	return w
}

// schedules projects every materialized day's schedule.
func schedules(w *sim.World, days int64) []api.DaySchedule {
	var out []api.DaySchedule
	for d := int64(0); d < days; d++ {
		out = append(out, w.Schedule(d))
	}
	return out
}

func TestDemandDeterministic(t *testing.T) {
	cfg := miniConfig(t)
	a, b := run(t, cfg, 7, 21), run(t, cfg, 7, 21)
	if a.LogHash() != b.LogHash() || !reflect.DeepEqual(a.Orders(), b.Orders()) ||
		!reflect.DeepEqual(schedules(a, 140), schedules(b, 140)) {
		t.Fatal("same seed and scenario gave different demand or bookings")
	}
	c := run(t, cfg, 8, 21)
	if reflect.DeepEqual(a.Orders(), c.Orders()) {
		t.Fatal("different seeds gave identical demand")
	}
}

// exogenous strips what booking decides, leaving what the referral
// sources sent.
func exogenous(os []sim.OrderInfo) []sim.OrderInfo {
	out := make([]sim.OrderInfo, len(os))
	for i, o := range os {
		o.State, o.LostWhy = "", ""
		out[i] = o
	}
	return out
}

// Booking decisions cannot alter exogenous demand: with very different
// capacity and booking rules, the same orders arrive at the same times with
// the same attributes and IDs.
func TestBookingCannotAlterDemand(t *testing.T) {
	base := miniConfig(t)
	alt := *base
	alt.Policies.BookingHorizonDays = 10
	alt.Policies.BookingNotice = 6 * simtime.Hour
	// The opening schedule (Tuesdays) stays; MR capacity vanishes on every
	// other day and Wednesday US is cut to two slots.
	alt.Templates = append([][7][]sim.SlotTemplate(nil), base.Templates...)
	alt.Templates[3] = [7][]sim.SlotTemplate{simtime.Tuesday: base.Templates[3][simtime.Tuesday]}
	alt.Templates[1][simtime.Wednesday] = alt.Templates[1][simtime.Wednesday][:2]

	a, b := run(t, base, 3, 28), run(t, &alt, 3, 28)
	oa, ob := a.Orders(), b.Orders()
	if !reflect.DeepEqual(exogenous(oa), exogenous(ob)) {
		t.Fatalf("demand changed with booking capacity: %d vs %d orders", len(oa), len(ob))
	}
	var lostA, lostB int
	for i := range oa {
		if oa[i].State == "lost" {
			lostA++
		}
		if ob[i].State == "lost" {
			lostB++
		}
	}
	if lostB <= lostA {
		t.Fatalf("less capacity should lose more demand: %d vs %d", lostB, lostA)
	}
}

// No two active appointments overlap in a room, each sits in a slot of its
// exam's category, and every booked slot was staffed by a qualified
// technologist (and a radiologist on site for contrast) per the roster.
func TestBookingsFeasible(t *testing.T) {
	cfg := miniConfig(t)
	w := run(t, cfg, 5, 35)
	cal := w.Calendar()
	type roomDay struct {
		room int
		day  int64
	}
	held := map[roomDay][]sim.Span{}
	active := 0
	for _, a := range w.Appointments() {
		if a.Cancelled {
			continue
		}
		active++
		date := cal.Date(cal.StartOfDay(a.Day))
		plan := cfg.Plan(date)
		if plan.Hours == nil || a.Span.From < plan.Hours.From || a.Span.To > plan.Hours.To {
			t.Fatalf("%s outside open hours on %s", a.ID, date)
		}
		var exam sim.ExamConfig
		for _, x := range cfg.Catalog {
			if x.Code == a.Exam {
				exam = x
			}
		}
		if cfg.Rooms[a.Room].Modality != exam.Modality {
			t.Fatalf("%s: %s exam in %s room", a.ID, exam.Modality, cfg.Rooms[a.Room].Modality)
		}
		cov := plan.Coverage()
		if ok, why := cov.SlotBookable(a.Room, a.Span, exam.Contrast); !ok {
			t.Fatalf("%s on %s %s: %s", a.ID, date, simtime.FormatClock(a.Span.From), why)
		}
		k := roomDay{a.Room, a.Day}
		for _, s := range held[k] {
			if s.From < a.Span.To && a.Span.From < s.To {
				t.Fatalf("%s overlaps another appointment in room %d on %s", a.ID, a.Room, date)
			}
		}
		held[k] = append(held[k], a.Span)
	}
	if active < 500 {
		t.Fatalf("only %d active appointments after 35 days", active)
	}
}

// Every order ends in exactly one state, and the counters agree with the
// states: nothing received goes unaccounted for.
func TestUnfulfilledDemandAccounted(t *testing.T) {
	cfg := miniConfig(t)
	alt := *cfg
	alt.Policies.BookingHorizonDays = 20 // more recall traffic
	w := run(t, &alt, 9, 42)
	obs := w.Observe(0)
	states := map[string]int{}
	orders := w.Orders()[len(cfg.Opening):]
	for _, o := range orders {
		states[o.State]++
	}
	var total api.DemandCounts
	for _, m := range obs.Demand.Modalities {
		c := m.SinceStart
		total.Received += c.Received
		total.Booked += c.Booked
		total.Recall += c.Recall
		total.Lost += c.Lost
		total.Cancelled += c.Cancelled
	}
	cancelled := 0
	for _, a := range w.Appointments() {
		if a.Cancelled {
			cancelled++
		}
	}
	switch {
	case total.Received+obs.Flow.SinceStart.WalkIns != len(orders):
		t.Fatalf("received %d + walk-ins %d, orders %d", total.Received, obs.Flow.SinceStart.WalkIns, len(orders))
	case states["booked"]+states["recall"]+states["lost"]+states["cancelled"]+states["visit"]+
		states["no-show"]+states["left"]+states["imaged"]+states["reported"] != len(orders):
		t.Fatalf("states %v do not cover %d orders", states, len(orders))
	case total.Lost != states["lost"]:
		t.Fatalf("lost counter %d, lost orders %d", total.Lost, states["lost"])
	case states["recall"] != obs.Demand.RecallList:
		t.Fatalf("recall orders %d, list %d", states["recall"], obs.Demand.RecallList)
	case total.Cancelled != cancelled:
		t.Fatalf("cancelled counter %d, cancelled appointments %d", total.Cancelled, cancelled)
	case states["lost"] == 0 || total.Recall == 0 || cancelled == 0 || total.Recall == states["recall"]:
		t.Fatalf("scenario did not exercise loss, recall release and cancellation: %v %+v", states, total)
	}
}

// Recall orders are booked once their date enters the horizon, and a
// cancelled slot is free for others at once.
func TestRecallAndCancellationPaths(t *testing.T) {
	cfg := miniConfig(t)
	alt := *cfg
	alt.Policies.BookingHorizonDays = 20
	w := run(t, &alt, 9, 42)
	log := w.Observe(2000).RecentLog
	var released, rebooked bool
	for _, e := range log {
		released = released || strings.Contains(e.Text, "from recall list → booked")
		rebooked = rebooked || strings.Contains(e.Text, "new date requested → booked")
	}
	if !released || !rebooked {
		t.Fatalf("recall release %v, rebooking %v", released, rebooked)
	}
	// A cancelled appointment's slot is either open again or held by a
	// different appointment.
	type key struct {
		room int
		day  int64
		from simtime.Duration
	}
	holder := map[key]string{}
	for _, a := range w.Appointments() {
		if !a.Cancelled {
			holder[key{a.Room, a.Day, a.Span.From}] = a.ID
		}
	}
	for _, a := range w.Appointments() {
		if a.Cancelled && holder[key{a.Room, a.Day, a.Span.From}] == a.ID {
			t.Fatalf("%s still holds its slot after cancelling", a.ID)
		}
	}
}

// Public IDs are allocated only at disclosure: removing future demand (all
// orders from a time T on) changes nothing observable before T — not the
// log, not the IDs, not the schedule.
func TestIDsRevealNoFutureDemand(t *testing.T) {
	cfg := miniConfig(t)
	a := sim.New(cfg, rng.SeedFromInt(4))
	b := sim.New(cfg, rng.SeedFromInt(4))
	T := a.Calendar().StartOfDate(cfg.Start.AddDays(2)).Add(13 * simtime.Hour)
	b.DropOrdersFrom(T)
	for _, w := range []*sim.World{a, b} {
		w.Engine().CheckInvariants = true
		w.Engine().RunUntil(T-1, nil)
	}
	if a.LogHash() != b.LogHash() || !reflect.DeepEqual(a.Observe(2000), b.Observe(2000)) ||
		!reflect.DeepEqual(schedules(a, 130), schedules(b, 130)) {
		t.Fatal("future demand leaked into observations before it arrived")
	}
	a.Engine().RunUntil(T.Add(simtime.Hour), nil)
	b.Engine().RunUntil(T.Add(simtime.Hour), nil)
	if a.LogHash() == b.LogHash() {
		t.Fatal("dropping future orders changed nothing after T; the test is vacuous")
	}
}

// BenchmarkMini180 runs 180 days of the mini scenario: roster, staffing,
// demand and booking (no arrivals or imaging yet).
func BenchmarkMini180(b *testing.B) {
	cfg := miniConfig(b)
	for i := 0; i < b.N; i++ {
		w := sim.New(cfg, rng.SeedFromInt(1))
		w.Engine().RunUntil(w.Calendar().StartOfDate(cfg.Start.AddDays(180)), nil)
	}
}

// BenchmarkEstablishedWarmup builds the established practice: the 180-day
// warm-up from the epoch to the start time (ROADMAP step 11, target
// < 1.5 s on the reference Pi).
func BenchmarkEstablishedWarmup(b *testing.B) {
	cfg, err := scenario.Resolve("established")
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < b.N; i++ {
		w := sim.New(cfg, rng.SeedFromInt(1))
		w.Engine().RunUntil(w.StartTime(), nil)
	}
}
