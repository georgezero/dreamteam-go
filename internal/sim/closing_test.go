package sim_test

import (
	"os"
	"strings"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/rng"
	"github.com/georgezero/dreamteam-go/internal/scenario"
	"github.com/georgezero/dreamteam-go/internal/sim"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Closing rules (SIMULATION_DESIGN §6.9) on hand-built micro practices
// (testdata/micro-*.json): open 08:00–09:00, fixed durations (20 min exams,
// 1 min documentation, 3 min self-registration), patients exactly on time.

func micro(t *testing.T, name string) (*sim.World, simtime.Time) {
	t.Helper()
	f, err := scenario.Load(os.DirFS("testdata"), name)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := scenario.Compile(f)
	if err != nil {
		t.Fatal(err)
	}
	w := sim.New(cfg, rng.SeedFromInt(1))
	w.Engine().CheckInvariants = true
	return w, w.Calendar().StartOfDate(cfg.Start)
}

func at(day simtime.Time, hhmm string) simtime.Time {
	c, err := simtime.ParseClock(hhmm)
	if err != nil {
		panic(err)
	}
	return day.Add(c)
}

func logHas(w *sim.World, text string) bool {
	for _, e := range w.Observe(2000).RecentLog {
		if strings.Contains(e.Text, text) {
			return true
		}
	}
	return false
}

func step(j api.Journey, label string) api.JourneyStep {
	for _, s := range j.Steps {
		if s.Label == label {
			return s
		}
	}
	return api.JourneyStep{}
}

func staffByName(o api.Observation, name string) api.Employee {
	for _, e := range o.Staff {
		if e.Name == name {
			return e
		}
	}
	return api.Employee{}
}

// One XR technologist (07:30–09:00), four patients 08:00–08:45. The
// timeline: P1 08:03–08:23, P2 08:27–08:47, P3 08:54–09:14, P4 09:15–09:35
// (after scheduled close), last documentation ends 09:36.
func TestWaitingPatientsServedAfterClose(t *testing.T) {
	w, day := micro(t, "micro-overtime.json")
	w.Engine().RunUntil(at(day, "09:00"), nil)
	o := w.Observe(0)
	if o.Practice.Phase != api.PhaseClosing || o.Practice.OnSite != 2 {
		t.Fatalf("at 09:00: phase %s, on site %d", o.Practice.Phase, o.Practice.OnSite)
	}
	w.Engine().RunUntil(at(day, "09:16"), nil)
	o = w.Observe(0)
	if tess := staffByName(o, "Tess"); !tess.Held || tess.OvertimeMin != 16 || o.Rooms[0].State != api.RoomOccupied {
		t.Fatalf("at 09:16: Tess %+v, XR %s", tess, o.Rooms[0].State)
	}
	w.Engine().RunUntil(at(day, "12:00"), nil)
	o = w.Observe(0)
	var afterHours int
	for _, v := range w.Visits() {
		if v.State != "done" {
			t.Fatalf("%s ended %s", v.ID, v.State)
		}
	}
	afterHours = o.Flow.Today.AfterHours
	if afterHours != 1 || o.Practice.Phase != api.PhaseClosed || o.Practice.OperationalClose != "09:36" ||
		!logHas(w, "Operations closed 09:36 (scheduled 09:00) · 1 exam after hours") {
		t.Fatalf("after hours %d, phase %s, operational close %s", afterHours, o.Practice.Phase, o.Practice.OperationalClose)
	}
}

// Overtime is recorded and costed: 36 min past 09:00 at $60/h is $36 while
// the week is under 40 paid hours, ×1.5 ($54) once it is past.
func TestRequiredOvertimeCost(t *testing.T) {
	for _, tc := range []struct {
		file string
		cost int64
	}{{"micro-overtime.json", 3600}, {"micro-overtime-premium.json", 5400}} {
		w, day := micro(t, tc.file)
		w.Engine().RunUntil(at(day, "12:00"), nil)
		f := w.Observe(0).Flow.Today
		if f.OvertimeMin != 36 || f.OvertimeCostCents != tc.cost || f.HeldOver != 1 {
			t.Errorf("%s: overtime %dm, cost %d, held %d", tc.file, f.OvertimeMin, f.OvertimeCostCents, f.HeldOver)
		}
		if !logHas(w, "Tess stays past shift end: 1 patient still need XR (overtime)") ||
			!logHas(w, "Tess off duty after 36m overtime") {
			t.Errorf("%s: hold-over not logged", tc.file)
		}
	}
}

// With a 2-hour cap (07:30–09:30) P4 cannot be finished safely: from 09:14
// (P3's exam over) an exam plus documentation would run past 09:30. P4 is
// rescheduled then, with the reason, and Tess goes home after documenting
// P3 at 09:15 instead of working toward her cap.
func TestCoverageExhaustedReschedules(t *testing.T) {
	w, day := micro(t, "micro-cap.json")
	w.Engine().RunUntil(at(day, "12:00"), nil)
	vs := w.Visits()
	last := vs[len(vs)-1]
	if last.State != "rescheduled" || last.EndedAt != at(day, "09:14") {
		t.Fatalf("P4: %+v", last)
	}
	j, _ := w.Journey("P00004")
	if j.Status != "booked" || !strings.Contains(step(j, "Rescheduled").Detail, "within the 2-hour shift limit") {
		t.Fatalf("journey: status %s, steps %+v", j.Status, j.Steps)
	}
	f := w.Observe(0).Flow.Today
	if f.NoCoverage != 1 || f.OvertimeMin != 15 || !logHas(w, "Tess off duty after 15m overtime") {
		t.Fatalf("no coverage %d, overtime %dm", f.NoCoverage, f.OvertimeMin)
	}
}

// One technologist shared between XR and CT runs one exam at a time: while
// she scans in one room the other is idle with its patient waiting.
func TestSharedTechnologistContention(t *testing.T) {
	w, day := micro(t, "micro-shared.json")
	w.Engine().RunUntil(at(day, "08:10"), nil)
	o := w.Observe(0)
	occupied, waitingRoom := 0, ""
	for _, r := range o.Rooms {
		if r.State == api.RoomOccupied {
			occupied++
		} else if r.State == api.RoomIdle {
			waitingRoom = r.Modality
		}
	}
	if occupied != 1 || waitingRoom == "" || !o.Rooms[0].Shared {
		t.Fatalf("at 08:10: %d rooms scanning, idle %q, shared %v", occupied, waitingRoom, o.Rooms[0].Shared)
	}
	w.Engine().RunUntil(at(day, "12:00"), nil)
	type iv struct{ from, to simtime.Time }
	var exams []iv
	for _, v := range w.Visits() {
		if v.State != "done" {
			t.Fatalf("%s ended %s", v.ID, v.State)
		}
		exams = append(exams, iv{v.ExamStart, v.ExamEnd})
	}
	for i := range exams {
		for j := range exams {
			if i != j && exams[i].from < exams[j].to && exams[j].from < exams[i].to {
				t.Fatalf("exams %d and %d overlap for the one technologist", i, j)
			}
		}
	}
}

// Contrast after hours: the on-site radiologist's shift ends at 09:00 but
// she is held over to supervise the contrast patients still admitted, and
// released when none remain. With her 2-hour cap (09:30), the last patient
// (who would run to 09:35) is rescheduled because nobody can supervise.
func TestContrastSupervisionAfterHours(t *testing.T) {
	w, day := micro(t, "micro-contrast.json")
	w.Engine().RunUntil(at(day, "09:20"), nil)
	if rae := staffByName(w.Observe(0), "Dr. Rae"); !rae.Held {
		t.Fatalf("at 09:20 Dr. Rae is not held over: %+v", rae)
	}
	w.Engine().RunUntil(at(day, "12:00"), nil)
	for _, v := range w.Visits() {
		if v.State != "done" {
			t.Fatalf("%s ended %s", v.ID, v.State)
		}
	}
	if !logHas(w, "Dr. Rae stays past shift end: 2 contrast patients need an on-site radiologist") ||
		!logHas(w, "Dr. Rae off duty after 35m overtime") {
		for _, e := range w.Observe(2000).RecentLog {
			t.Log(e.Time[11:16], e.Text)
		}
		t.Fatal("radiologist hold-over not logged as expected")
	}

	w, day = micro(t, "micro-contrast-cap.json")
	w.Engine().RunUntil(at(day, "12:00"), nil)
	vs := w.Visits()
	if last := vs[len(vs)-1]; last.State != "rescheduled" {
		t.Fatalf("last contrast patient: %s", last.State)
	}
	j, _ := w.Journey("P00004")
	if d := step(j, "Rescheduled").Detail; !strings.Contains(d, "no radiologist on site to supervise contrast") {
		t.Fatalf("reason: %q", d)
	}
}

// Across a month of the mini practice: every visit ends imaged, left or
// rescheduled with a reason; nobody is on site at accounting close (checked
// by the engine); overtime is recorded; some days run past scheduled close.
func TestClosingRulesOverAMonth(t *testing.T) {
	cfg := miniConfig(t)
	w := sim.New(cfg, rng.SeedFromInt(3))
	w.Engine().CheckInvariants = true
	cal := w.Calendar()
	late := 0
	for d := int64(0); d < 28; d++ {
		w.Engine().RunUntil(cal.StartOfDay(d+1)-2, nil)
		p := w.Observe(0).Practice
		if p.OperationalClose != "" && p.OperationalClose > p.ScheduledClose {
			late++
		}
	}
	total := w.Observe(0).Flow.SinceStart
	for _, v := range w.Visits() {
		switch v.State {
		case "done", "left":
		case "rescheduled":
			if v.Reason == "" {
				t.Fatalf("%s rescheduled without a reason", v.ID)
			}
		default:
			t.Fatalf("%s still %s", v.ID, v.State)
		}
	}
	if late == 0 || total.OvertimeMin == 0 || total.OvertimeCostCents == 0 || total.ExamsDone < 800 {
		t.Fatalf("late closes %d, overtime %dm $%d, exams %d", late, total.OvertimeMin, total.OvertimeCostCents/100, total.ExamsDone)
	}
}

// Chunking: the full practice advanced minute by minute matches one jump.
func TestOperationsChunkingInvariance(t *testing.T) {
	cfg := miniConfig(t)
	a := sim.New(cfg, rng.SeedFromInt(5))
	b := sim.New(cfg, rng.SeedFromInt(5))
	end := a.Calendar().StartOfDate(cfg.Start.AddDays(3))
	a.Engine().RunUntil(end, nil)
	for now := simtime.Time(0); now < end; now = now.Add(simtime.Minute) {
		b.Engine().RunUntil(now, nil)
	}
	b.Engine().RunUntil(end, nil)
	if a.LogHash() != b.LogHash() {
		t.Fatal("minute-by-minute advance changed the outcome")
	}
}
