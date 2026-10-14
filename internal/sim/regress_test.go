package sim_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/rng"
	"github.com/georgezero/dreamteam-go/internal/scenario"
	"github.com/georgezero/dreamteam-go/internal/sim"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Regression tests for the step 8 review (docs/reviews/step-08-astra-review.md).
// Each reproduces the reviewer's scenario on a micro practice
// (testdata/micro-*.json: fixed durations, 20 min exams, 1 min
// documentation, 3 min self-registration, 2 min desk registration).

// run advances w to t, turning an invariant panic into an error.
func runTo(w *sim.World, t simtime.Time) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	w.Engine().RunUntil(t, nil)
	return nil
}

func staffNamed(w *sim.World, name string) sim.StaffInfo {
	for _, s := range w.Staff() {
		if s.Name == name {
			return s
		}
	}
	panic("no employee " + name)
}

func onlyVisit(t *testing.T, w *sim.World) sim.VisitInfo {
	t.Helper()
	vs := w.Visits()
	if len(vs) != 1 {
		t.Fatalf("%d visits, want 1", len(vs))
	}
	return vs[0]
}

// Finding 1: Dr. Bo (cap 10:00) supervises a 45-min contrast exam from
// 08:48 that Dr. Rae (cap 09:30) cannot cover. Bo's shift ends at 09:00;
// she must stay until 09:33, and Rae must leave at her 09:30 shift end.
func TestRegress01RadiologistHandoverCap(t *testing.T) {
	w, day := micro(t, "micro-rad-handover.json")
	if err := runTo(w, at(day, "12:00")); err != nil {
		t.Fatal(err)
	}
	if v := onlyVisit(t, w); v.State != "done" || v.ExamEnd != at(day, "09:33") {
		t.Fatalf("visit %+v", v)
	}
	bo, rae := staffNamed(w, "Dr. Bo"), staffNamed(w, "Dr. Rae")
	if bo.Overtime != 33*simtime.Minute || rae.Overtime != 0 {
		t.Fatalf("overtime: Dr. Bo %s (want 33m), Dr. Rae %s (want 0, cap 09:30)", bo.Overtime, rae.Overtime)
	}
}

// Finding 2: the only patient is due at 10:00 (early starts off) and Tess's
// cap is 09:30, so holding her past 09:00 can't help. She must go home at
// 09:00, not idle until Tom arrives at 09:45.
func TestRegress02HeldPastUsefulnessAndCap(t *testing.T) {
	w, day := micro(t, "micro-cap-wake.json")
	if err := runTo(w, at(day, "12:00")); err != nil {
		t.Fatal(err)
	}
	if tess := staffNamed(w, "Tess"); tess.Overtime != 0 {
		t.Fatalf("Tess overtime %s, want 0 (cap 09:30, patient not due until 10:00)", tess.Overtime)
	}
	if v := onlyVisit(t, w); v.State != "done" || v.ExamStart != at(day, "10:00") {
		t.Fatalf("visit %+v", v)
	}
}

// Finding 3: Dana's cap (09:00) leaves no room for a 2-min registration at
// 08:59, so she must not block self-registration: Tess registers the
// patient (08:59–09:02) and images them; nobody is stranded.
func TestRegress03DeskCannotBlockSelfRegistration(t *testing.T) {
	w, day := micro(t, "micro-desk-cap.json")
	if err := runTo(w, at(day, "23:59:59")); err != nil {
		t.Fatal(err)
	}
	if v := onlyVisit(t, w); v.State != "done" || v.ExamStart != at(day, "09:02") {
		t.Fatalf("visit %+v", v)
	}
}

// Finding 4: the patient registers with the desk 08:59–09:01 while Tess's
// shift ends at 09:00. Tess is needed and must stay to image them.
func TestRegress04HoldForRegisteringPatient(t *testing.T) {
	w, day := micro(t, "micro-registering.json")
	if err := runTo(w, at(day, "12:00")); err != nil {
		t.Fatal(err)
	}
	if v := onlyVisit(t, w); v.State != "done" || v.ExamStart != at(day, "09:01") {
		t.Fatalf("visit %+v", v)
	}
	if tess := staffNamed(w, "Tess"); tess.Overtime != 22*simtime.Minute {
		t.Fatalf("Tess overtime %s, want 22m", tess.Overtime)
	}
}

// Finding 5: a shift whose 12-hour cap reaches midnight lets documentation
// end at 00:00, after the next day has started. Either the scenario is
// rejected (cap must end by accounting close), or, if it runs, the
// overtime and the operational close must still be recorded for the day.
func TestRegress05MidnightBoundary(t *testing.T) {
	f, err := scenario.Load(os.DirFS("testdata"), "micro-midnight.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := scenario.Compile(f); err != nil {
		if !strings.Contains(err.Error(), "accounting close") {
			t.Fatalf("rejected for the wrong reason: %v", err)
		}
		return // fixed: the day's work must end by accounting close
	}
	w, day := micro(t, "micro-midnight.json")
	if err := runTo(w, day.Add(30*simtime.Hour)); err != nil {
		t.Fatal(err)
	}
	if ot := w.Observe(0).Flow.SinceStart.OvertimeMin; ot != 15 {
		t.Fatalf("overtime %dm recorded, want 15m (23:45–24:00)", ot)
	}
}

// Finding 6: an afternoon-preferring order must get the earliest
// afternoon slot across rooms (13:00 in X-ray 2), not 15:00 in X-ray 1.
func TestRegress06FirstFitPreferredAcrossRooms(t *testing.T) {
	w, day := micro(t, "micro-firstfit.json")
	if err := runTo(w, at(day, "06:00")); err != nil {
		t.Fatal(err)
	}
	room, from, ok := w.TryBook("XR_A", sim.Afternoon, w.DayIndex()+7)
	if !ok || room != 2 || from != 13*simtime.Hour {
		t.Fatalf("booked room %d at %s (ok %v), want room 2 at 13:00", room, simtime.FormatClock(from), ok)
	}
}

// Finding 7: with CT 09:30–10:00 booked, the XR slot 09:00–10:00 needs the
// same technologist from 09:30 and must not be bookable.
func TestRegress07SharedCapacityAcrossSlot(t *testing.T) {
	w, day := micro(t, "micro-shared-span.json")
	if err := runTo(w, at(day, "06:00")); err != nil {
		t.Fatal(err)
	}
	d := w.DayIndex() + 7
	if _, from, ok := w.TryBook("CT_L", sim.AnyWindow, d); !ok || from != 9*simtime.Hour+30*simtime.Minute {
		t.Fatalf("CT booking failed: %v %s", ok, simtime.FormatClock(from))
	}
	if day, room, from, ok := w.TryBookDay("XR_L", sim.AnyWindow, d); ok && day == d {
		t.Fatalf("XR booked room %d %s on day %d although its technologist is booked in CT 09:30–10:00 that day",
			room, simtime.FormatClock(from), d)
	}
}

// Finding 8: an opening schedule that double-books a shared technologist
// must be rejected like any other booking.
func TestRegress08OpeningRespectsSharedCapacity(t *testing.T) {
	f, err := scenario.Load(os.DirFS("testdata"), "micro-shared-conflict.json")
	if err != nil {
		t.Fatal(err)
	}
	_, err = scenario.Compile(f)
	if err == nil || !strings.Contains(err.Error(), "shared technologist") {
		t.Fatalf("want a shared-technologist validation error, got %v", err)
	}
}

// Finding 9: the first break, deferred from 13:00 to 13:09 by an exam,
// runs 13:09–13:24; the second (due 13:20) must not overwrite it. Both
// count: 30 min of unpaid break.
func TestRegress09DeferredBreaksDoNotOverwrite(t *testing.T) {
	w, day := micro(t, "micro-breaks.json")
	if err := runTo(w, at(day, "14:00")); err != nil {
		t.Fatal(err)
	}
	if tess := staffNamed(w, "Tess"); tess.BreakTaken != 30*simtime.Minute {
		t.Fatalf("break taken %s, want 30m", tess.BreakTaken)
	}
	if f := w.Observe(0).Flow.Today; f.SkippedBreaks != 0 {
		t.Fatalf("%d skipped breaks", f.SkippedBreaks)
	}
}

// Finding 10: a patient registering at 08:01 (arrived 08:00) has waited
// 1 minute, not a negative time.
func TestRegress10WaitWhileRegistering(t *testing.T) {
	w, day := micro(t, "micro-overtime.json")
	if err := runTo(w, at(day, "08:01")); err != nil {
		t.Fatal(err)
	}
	for _, r := range w.Patients(w.DayIndex()) {
		if r.Patient == "P00001" {
			if r.Status != api.PatientRegistering || r.WaitedMin != 1 {
				t.Fatalf("P00001 %s waited %d min, want registering, 1", r.Status, r.WaitedMin)
			}
			return
		}
	}
	t.Fatal("P00001 not listed")
}

// Wake completeness: unrelated events that are handled (and so run
// dispatch) or stale must not change any outcome. Missing wake-ups would
// make results depend on them.
func TestRegressOutcomeIndependentOfUnrelatedEvents(t *testing.T) {
	files := []string{"micro-overtime.json", "micro-cap.json", "micro-contrast.json", "micro-contrast-cap.json",
		"micro-shared.json", "micro-rad-handover.json", "micro-cap-wake.json", "micro-desk-cap.json",
		"micro-registering.json", "micro-breaks.json"}
	for _, f := range files {
		plain, day := micro(t, f)
		live, _ := micro(t, f)
		stale, _ := micro(t, f)
		end := day.Add(20 * simtime.Hour)
		errs := []error{runTo(plain, end)}
		for _, w := range []*sim.World{live, stale} {
			if err := runTo(w, day.Add(7*simtime.Hour)); err != nil {
				errs = append(errs, err)
				continue
			}
			for t := day.Add(7*simtime.Hour + 7*simtime.Minute); t < end; t = t.Add(7 * simtime.Minute) {
				if w == live {
					w.PushLiveNoise(t)
				} else {
					w.PushStaleNoise(t)
				}
			}
			errs = append(errs, runTo(w, end))
		}
		for _, err := range errs {
			if err != nil {
				t.Errorf("%s: %v", f, err)
			}
		}
		if plain.LogHash() != live.LogHash() {
			t.Errorf("%s: unrelated live events changed the outcome", f)
		}
		if plain.LogHash() != stale.LogHash() {
			t.Errorf("%s: stale events changed the outcome", f)
		}
	}
}

// The same over four days of the full mini practice, with invariants
// checked after every transaction: unrelated handled events (re-armed each
// day, since day start renews generations) leave every outcome unchanged.
func TestRegressMiniUnaffectedByUnrelatedEvents(t *testing.T) {
	cfg := miniConfig(t)
	plain := sim.New(cfg, rng.SeedFromInt(8))
	noisy := sim.New(cfg, rng.SeedFromInt(8))
	cal := plain.Calendar()
	for _, w := range []*sim.World{plain, noisy} {
		w.Engine().CheckInvariants = true
	}
	for d := int64(0); d < 4; d++ {
		start := cal.StartOfDay(d)
		for _, w := range []*sim.World{plain, noisy} {
			if err := runTo(w, start.Add(1)); err != nil {
				t.Fatal(err)
			}
		}
		for t := start.Add(7 * simtime.Minute); t < start.Add(simtime.Day-2); t = t.Add(7 * simtime.Minute) {
			noisy.PushLiveNoise(t)
		}
		for _, w := range []*sim.World{plain, noisy} {
			if err := runTo(w, cal.StartOfDay(d+1)); err != nil {
				t.Fatal(err)
			}
		}
		if plain.LogHash() != noisy.LogHash() {
			t.Fatalf("day %d: unrelated events changed the outcome", d)
		}
	}
}
