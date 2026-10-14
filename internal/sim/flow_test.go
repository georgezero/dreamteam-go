package sim_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/rng"
	"github.com/georgezero/dreamteam-go/internal/sim"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// flowRun is a shared 56-day mini run, so the statistical tests have a
// reasonable sample (about 1,800 scheduled arrivals and 400 walk-ins).
var flowRun *sim.World

func longRun(t *testing.T) *sim.World {
	t.Helper()
	if flowRun == nil {
		cfg := miniConfig(t)
		w := sim.New(cfg, rng.SeedFromInt(21))
		w.Engine().CheckInvariants = true
		w.Engine().RunUntil(w.Calendar().StartOfDate(cfg.Start.AddDays(56)), nil)
		flowRun = w
	}
	return flowRun
}

// Arrival offsets follow the punctuality classes: about 60% early, 28% on
// time, 9% late and 3% very late (SIMULATION_DESIGN §5.2).
func TestArrivalDistribution(t *testing.T) {
	w := longRun(t)
	var n, early, onTime, late, veryLate int
	for _, v := range w.Visits() {
		if v.WalkIn {
			continue
		}
		n++
		switch m := v.Late; {
		case m < -5*simtime.Minute:
			early++
		case m <= 5*simtime.Minute:
			onTime++
		case m <= 20*simtime.Minute:
			late++
		default:
			veryLate++
		}
		if v.Late < -25*simtime.Minute || v.Late > 45*simtime.Minute {
			t.Fatalf("%s arrived %s from start, outside the classes", v.ID, v.Late)
		}
	}
	share := func(k int) int { return k * 1000 / n }
	if n < 1500 || share(early) < 550 || share(early) > 640 || share(onTime) < 245 || share(onTime) > 335 ||
		share(late) < 60 || share(late) > 115 || share(veryLate) < 15 || share(veryLate) > 45 {
		t.Fatalf("n=%d early %d‰ on time %d‰ late %d‰ very late %d‰", n, share(early), share(onTime), share(late), share(veryLate))
	}
}

// No-shows run at the base rates plus the lead-time effect (≈ 4–10%), are
// finalized at accounting close, and never have a visit.
func TestNoShows(t *testing.T) {
	w := longRun(t)
	cal := w.Calendar()
	var reached, noShows, missedThenCame int
	for _, a := range w.Appointments() {
		switch a.State {
		case "no-show":
			noShows++
			reached++
			if a.Visit >= 0 {
				t.Fatalf("%s is a no-show with a visit", a.ID)
			}
			if a.EndedAt != cal.StartOfDay(a.Day+1)-1 {
				t.Fatalf("%s finalized at %s, not at accounting close", a.ID, cal.ISO(a.EndedAt))
			}
		case "arrived", "rescheduled":
			reached++
		}
	}
	for _, v := range w.Visits() {
		if v.AfterGrace {
			missedThenCame++
			if v.Late <= 15*simtime.Minute {
				t.Fatalf("%s fitted in after grace but only %s late", v.ID, v.Late)
			}
		}
	}
	rate := noShows * 1000 / reached
	if rate < 40 || rate > 100 || missedThenCame == 0 {
		t.Fatalf("no-shows %d of %d (%d‰); missed then came %d", noShows, reached, rate, missedThenCame)
	}
}

// Every appointment of a finished day is accounted for: arrived, no-show,
// cancelled or rescheduled. None is left booked or provisionally missed,
// and each has at most one visit.
func TestAppointmentConservation(t *testing.T) {
	w := longRun(t)
	end := w.DayIndex()
	visits := map[int]bool{}
	for _, a := range w.Appointments() {
		if a.Day >= end {
			continue
		}
		switch a.State {
		case "arrived", "no-show", "cancelled", "rescheduled":
		default:
			t.Fatalf("%s on finished day %d is %s", a.ID, a.Day, a.State)
		}
		if a.Visit >= 0 {
			if visits[a.Visit] {
				t.Fatalf("visit %d belongs to two appointments", a.Visit)
			}
			visits[a.Visit] = true
		}
	}
	// An order can have a second visit only after the first ended
	// (rescheduled for lack of coverage, then back on a later day).
	last := map[string]sim.VisitInfo{}
	for _, v := range w.Visits() {
		if p, ok := last[v.Order]; ok && (p.State != "rescheduled" || p.EndedAt > v.Arrived) {
			t.Fatalf("order %s has overlapping visits %s and %s", v.Order, p.ID, v.ID)
		}
		last[v.Order] = v
		if v.State != "done" && v.State != "left" && v.State != "rescheduled" && v.Arrived < w.Calendar().StartOfDay(end) {
			t.Fatalf("%s from an earlier day still %s", v.ID, v.State)
		}
	}
}

// Walk-ins: about 6–12 on weekdays, overwhelmingly X-ray, never MR, only
// walk-in eligible exams, only while open, none on Sunday.
func TestWalkIns(t *testing.T) {
	w := longRun(t)
	cfg := miniConfig(t)
	cal := w.Calendar()
	eligible := map[string]string{}
	for _, x := range cfg.Catalog {
		if x.WalkInEligible {
			eligible[x.Code] = x.Modality
		}
	}
	perDay := map[int64]int{}
	byMod := map[string]int{}
	total := 0
	for _, o := range w.Orders() {
		if o.Origin[1] != -1 || o.Origin[0] < 0 {
			continue // not a walk-in
		}
		total++
		mod, ok := eligible[o.Exam]
		if !ok {
			t.Fatalf("%s walked in for non-eligible %s", o.ID, o.Exam)
		}
		byMod[mod]++
		day := cal.DayIndex(o.Received)
		perDay[day]++
		date := cal.Date(o.Received)
		h := cfg.Hours[date.Weekday()]
		if h == nil || !h.Contains(simtime.ClockOf(o.Received)) {
			t.Fatalf("%s walked in at %s, outside open hours", o.ID, cal.Short(o.Received))
		}
	}
	weekdays, sum := 0, 0
	for d := int64(0); d < 56; d++ {
		wd := cal.Date(cal.StartOfDay(d)).Weekday()
		if wd >= simtime.Monday && wd <= simtime.Friday {
			weekdays++
			sum += perDay[d]
		}
	}
	avg := sum * 10 / weekdays
	if avg < 60 || avg > 120 || byMod["XR"]*100/total < 75 || byMod["MR"] != 0 {
		t.Fatalf("weekday walk-ins %d.%d/day, by modality %v of %d", avg/10, avg%10, byMod, total)
	}
}

// A walk-in who is still waiting leaves after a patience of 30–120 min,
// and the counter matches.
func TestWalkInPatience(t *testing.T) {
	w := longRun(t)
	left := 0
	for _, v := range w.Visits() {
		switch v.State {
		case "left":
			left++
			waited := v.EndedAt.Sub(v.Arrived)
			if !v.WalkIn || waited < 30*simtime.Minute || waited > 120*simtime.Minute {
				t.Fatalf("%s left after %s (walk-in %v)", v.ID, waited, v.WalkIn)
			}
		}
	}
	if obs := w.Observe(0); left == 0 || obs.Flow.SinceStart.Left != left {
		t.Fatalf("left %d, counter %d", left, obs.Flow.SinceStart.Left)
	}
}

// The journey shows a provisional miss followed by a late arrival.
func TestMissedThenLateJourney(t *testing.T) {
	w := longRun(t)
	for _, v := range w.Visits() {
		if !v.AfterGrace {
			continue
		}
		var patient string
		for _, o := range w.Orders() {
			if o.ID == v.Order {
				patient = "P" + strings.TrimPrefix(o.ID, "O")
			}
		}
		j, ok := w.Journey(patient)
		if !ok {
			t.Fatalf("no journey for %s", patient)
		}
		var labels []string
		for _, s := range j.Steps {
			labels = append(labels, s.Label)
		}
		got := strings.Join(labels, " > ")
		if !strings.Contains(got, "Missed (provisional) > Arrived") {
			t.Fatalf("journey %s: %s", patient, got)
		}
		return
	}
	t.Fatal("no late arrival after grace in the run")
}

// Hidden latent behavior never leaks: making every later patient a
// no-show and removing later walk-ins and orders changes nothing that is
// observable before it could have been seen.
func TestHiddenArrivalNoninterference(t *testing.T) {
	cfg := miniConfig(t)
	a := sim.New(cfg, rng.SeedFromInt(6))
	b := sim.New(cfg, rng.SeedFromInt(6))
	T := a.Calendar().StartOfDate(cfg.Start.AddDays(1)).Add(11 * simtime.Hour)
	b.ForceNoShowsFrom(T.Add(30 * simtime.Minute)) // earliest early arrival is 25 min before start
	b.DropOrdersFrom(T)
	for _, w := range []*sim.World{a, b} {
		w.Engine().CheckInvariants = true
		w.Engine().RunUntil(T, nil)
	}
	day := a.DayIndex()
	if a.LogHash() != b.LogHash() || !reflect.DeepEqual(a.Observe(2000), b.Observe(2000)) ||
		!reflect.DeepEqual(a.Patients(day), b.Patients(day)) || !reflect.DeepEqual(schedules(a, 10), schedules(b, 10)) {
		t.Fatal("hidden future behavior changed observations before it happened")
	}
	a.Engine().RunUntil(T.Add(3*simtime.Hour), nil)
	b.Engine().RunUntil(T.Add(3*simtime.Hour), nil)
	if reflect.DeepEqual(a.Patients(day), b.Patients(day)) {
		t.Fatal("the perturbation never showed; the test is vacuous")
	}
}

// pendingAppt finds an appointment still booked whose start is after t.
func pendingAppt(t *testing.T, w *sim.World, after simtime.Time) (int64, sim.ApptInfo) {
	t.Helper()
	cal := w.Calendar()
	for i, a := range w.Appointments() {
		if a.State == "booked" && a.Day == cal.DayIndex(after) && cal.At(a.Day, a.Span.From) > after.Add(time30) {
			return int64(i), a
		}
	}
	t.Fatal("no pending appointment")
	return 0, sim.ApptInfo{}
}

const time30 = 30 * simtime.Minute

// At one instant an arrival (class Arrival) is handled before a grace
// check and a cancellation (class Deadline), whatever the push order; the
// later events then find the appointment arrived and do nothing.
func TestSameInstantArrivalBeatsGraceAndCancel(t *testing.T) {
	cfg := miniConfig(t)
	w := sim.New(cfg, rng.SeedFromInt(2))
	w.Engine().CheckInvariants = true
	T := w.Calendar().StartOfDate(cfg.Start).Add(9 * simtime.Hour)
	w.Engine().RunUntil(T-60, nil)
	ai, a := pendingAppt(t, w, T)
	w.PushForTest(T, sim.ClassDeadline, "cancel", ai)
	w.PushForTest(T, sim.ClassDeadline, "grace", ai)
	w.PushForTest(T, sim.ClassArrival, "arrive", ai)
	w.Engine().RunUntil(T, nil)
	got := w.Appointments()[ai]
	if got.State != "arrived" || got.Visit < 0 {
		t.Fatalf("%s: %s", a.ID, got.State)
	}
	w.Engine().RunUntil(T.Add(4*simtime.Hour), nil) // natural events for it are now stale
	if s := w.Appointments()[ai].State; s != "arrived" {
		t.Fatalf("%s changed to %s later", a.ID, s)
	}
}

// Grace then cancellation at one instant: the first (by push order within
// the class) marks it missed and the cancellation is harmless.
func TestSameInstantGraceThenCancel(t *testing.T) {
	cfg := miniConfig(t)
	w := sim.New(cfg, rng.SeedFromInt(2))
	w.Engine().CheckInvariants = true
	T := w.Calendar().StartOfDate(cfg.Start).Add(9 * simtime.Hour)
	w.Engine().RunUntil(T-60, nil)
	ai, a := pendingAppt(t, w, T)
	w.PushForTest(T, sim.ClassDeadline, "grace", ai)
	w.PushForTest(T, sim.ClassDeadline, "cancel", ai)
	w.Engine().RunUntil(T, nil)
	if s := w.Appointments()[ai].State; s != "missed" {
		t.Fatalf("%s: %s, want missed", a.ID, s)
	}
	w.Engine().RunUntil(w.Calendar().StartOfDate(cfg.Start.AddDays(1)), nil)
	if s := w.Appointments()[ai].State; s != "arrived" && s != "no-show" {
		t.Fatalf("%s ended %s", a.ID, s)
	}
}

// Patience running out at the instant a room frees up: the deadline
// (class Deadline) is handled before dispatch, so the walk-in leaves
// rather than starting the exam; a walk-in already registering stays.
func TestPatienceBeatsDispatchAtSameInstant(t *testing.T) {
	cfg := miniConfig(t)
	w := sim.New(cfg, rng.SeedFromInt(2))
	w.Engine().CheckInvariants = true
	day := w.Calendar().StartOfDate(cfg.Start)
	var waiting, registering int64 = -1, -1
	at := day.Add(8 * simtime.Hour)
	for ; at < day.Add(19*simtime.Hour) && (waiting < 0 || registering < 0); at = at.Add(simtime.Minute) {
		w.Engine().RunUntil(at, nil)
		for i, v := range w.Visits() {
			if waiting < 0 && v.State == "waiting" {
				waiting = int64(i)
			}
			if registering < 0 && v.State == "registering" {
				registering = int64(i)
			}
		}
		if waiting >= 0 && registering >= 0 {
			break
		}
		if waiting >= 0 && registering < 0 {
			waiting = -1 // need both at the same instant
		}
		if registering >= 0 && waiting < 0 {
			registering = -1
		}
	}
	if waiting < 0 || registering < 0 {
		t.Fatal("no instant with both a waiting and a registering patient")
	}
	T := at.Add(1)
	w.PushForTest(T, sim.ClassDeadline, "patience", waiting)
	w.PushForTest(T, sim.ClassDeadline, "patience", registering)
	w.Engine().RunUntil(T, nil)
	vs := w.Visits()
	if vs[waiting].State != "left" {
		t.Fatalf("waiting patient: %s, want left", vs[waiting].State)
	}
	if vs[registering].State == "left" {
		t.Fatal("a patient being registered left")
	}
}
