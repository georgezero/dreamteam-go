package sim_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/rng"
	"github.com/georgezero/dreamteam-go/internal/sim"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Step 9: studies, the worklist, reading and turnaround.

func studyByID(t *testing.T, w *sim.World, id string) sim.StudyInfo {
	t.Helper()
	for _, s := range w.Studies() {
		if s.ID == id {
			return s
		}
	}
	t.Fatalf("no study %s", id)
	return sim.StudyInfo{}
}

// A month of the mini practice: exactly one study per imaged visit, none
// for anyone else, each finalized at most once and only after imaging;
// claims only inside the claimer's site reading blocks; reads end by the
// radiologist's scheduled shift end; one read at a time per radiologist;
// backlog carried across days.
func TestReadingOverAMonth(t *testing.T) {
	cfg := miniConfig(t)
	w := sim.New(cfg, rng.SeedFromInt(4))
	w.Engine().CheckInvariants = true
	cal := w.Calendar()
	if err := runTo(w, cal.StartOfDate(cfg.Start.AddDays(28))); err != nil {
		t.Fatal(err)
	}
	studies := w.Studies()
	perVisit := map[string]int{}
	for _, s := range studies {
		perVisit[s.Visit]++
	}
	for _, v := range w.Visits() {
		want := 0
		if v.State == "done" {
			want = 1
		}
		if perVisit[v.ID] != want {
			t.Fatalf("%s (%s) has %d studies", v.ID, v.State, perVisit[v.ID])
		}
	}
	emp := map[string]int{}
	for i, e := range cfg.Employees {
		emp[e.Name] = i
	}
	finals := map[string]int{} // within the retained log (last 2,000 entries)
	for _, e := range w.Observe(2000).RecentLog {
		if strings.Contains(e.Text, "report final") {
			finals[strings.Fields(e.Text)[0]]++
		}
	}
	type iv struct{ from, to simtime.Time }
	busy := map[string][]iv{}
	crossDay := 0
	for _, s := range studies {
		if finals[s.ID] > 1 {
			t.Fatalf("%s finalized %d times", s.ID, finals[s.ID])
		}
		if s.State != "final" {
			continue
		}
		if !(s.Acquired <= s.Entered && s.Entered <= s.Claimed && s.Claimed <= s.Final) {
			t.Fatalf("%s timestamps out of order", s.ID)
		}
		if cal.DayIndex(s.Final) != cal.DayIndex(s.Acquired) {
			crossDay++
		}
		plan := cfg.Plan(cal.Date(s.Claimed))
		i := emp[s.By]
		sh := plan.Shifts[plan.ByEmp[i]]
		in := false
		for _, b := range plan.Reading[i] {
			if b.Contains(simtime.ClockOf(s.Claimed)) {
				in = true
			}
		}
		if !in {
			t.Fatalf("%s claimed by %s at %s, outside a site reading block", s.ID, s.By, cal.Short(s.Claimed))
		}
		if simtime.ClockOf(s.Final) > sh.To || cal.DayIndex(s.Final) != cal.DayIndex(s.Claimed) {
			t.Fatalf("%s read past %s's shift end", s.ID, s.By)
		}
		busy[s.By] = append(busy[s.By], iv{s.Claimed, s.Final})
	}
	for name, ivs := range busy {
		for a := range ivs {
			for b := a + 1; b < len(ivs); b++ {
				if ivs[a].from < ivs[b].to && ivs[b].from < ivs[a].to {
					t.Fatalf("%s reads two studies at once", name)
				}
			}
		}
	}
	final := 0
	for _, s := range studies {
		if s.State == "final" {
			final++
		}
	}
	if crossDay == 0 || final < 700 {
		t.Fatalf("%d reports, %d finalized on a later day than imaged (backlog should carry over weekends)", final, crossDay)
	}
}

// Priority and ties (micro-reading: the radiologist arrives at 09:30 to
// five studies). The urgent study goes first; S00001 and S00002 entered
// the worklist at the same instant (08:24) and go in study order.
func TestWorklistPriorityAndTies(t *testing.T) {
	w, day := micro(t, "micro-reading.json")
	if err := runTo(w, at(day, "12:00")); err != nil {
		t.Fatal(err)
	}
	s1, s2 := studyByID(t, w, "S00001"), studyByID(t, w, "S00002")
	if s1.Entered != s2.Entered || s1.Entered != at(day, "08:24") {
		t.Fatalf("S00001/S00002 entered %s and %s, want both 08:24", w.Calendar().Clock(s1.Entered), w.Calendar().Clock(s2.Entered))
	}
	want := []string{"S00003", "S00001", "S00002", "S00004", "S00005"}
	for k, id := range want {
		s := studyByID(t, w, id)
		if s.Claimed != at(day, "09:30").Add(simtime.Duration(k)*5*simtime.Minute) {
			t.Fatalf("%s claimed at %s, want claim #%d (order %v)", id, w.Calendar().Clock(s.Claimed), k+1, want)
		}
	}
	if s3 := studyByID(t, w, "S00003"); s3.Priority != "urgent" || s3.Final.Sub(s3.Acquired) != 48*simtime.Minute {
		t.Fatalf("S00003: %s, TAT %s (want urgent, 48m: imaging done 08:47, final 09:35)", s3.Priority, s3.Final.Sub(s3.Acquired))
	}
}

// TAT is imaging complete → report final, shown as such. Micro-overtime:
// P1 imaged 08:03–08:23, documented by 08:24, read 08:24–08:29.
func TestTATDefinition(t *testing.T) {
	w, day := micro(t, "micro-overtime.json")
	if err := runTo(w, at(day, "08:26")); err != nil {
		t.Fatal(err)
	}
	wl := w.Observe(0).Worklist
	if wl.Reading != 1 || wl.Items[0].State != api.StudyReading || wl.Items[0].AgeMin != 3 || wl.TATMedianMin != 3 {
		t.Fatalf("at 08:26: %+v", wl)
	}
	if err := runTo(w, at(day, "08:30")); err != nil {
		t.Fatal(err)
	}
	s := studyByID(t, w, "S00001")
	if s.Acquired != at(day, "08:23") || s.Final != at(day, "08:29") {
		t.Fatalf("S00001 acquired %s final %s", w.Calendar().Clock(s.Acquired), w.Calendar().Clock(s.Final))
	}
	j, _ := w.Journey("P00001")
	if d := step(j, "Report final").Detail; !strings.Contains(d, "TAT 6m (imaging complete → final)") {
		t.Fatalf("journey: %q", d)
	}
	if wl := w.Observe(0).Worklist; wl.Recent[0].TATMin != 6 || wl.FinalToday != 1 {
		t.Fatalf("recent %+v final today %d", wl.Recent, wl.FinalToday)
	}
}

// Overrun rule (§6.7): S00001's 20-min read from 08:41 runs 11 min past the
// 08:50 block end, so the next block (09:40) opens for claims at 09:51.
// S00002 waits since 09:11 and is claimed exactly then, on a wake-up, not
// at some later unrelated event. A read ending exactly at block end owes
// nothing: S00002 is claimed at 09:40.
func TestReadingOverrunAndWake(t *testing.T) {
	for _, tc := range []struct {
		file, claim string
	}{{"micro-overrun.json", "09:51"}, {"micro-overrun-exact.json", "09:40"}} {
		w, day := micro(t, tc.file)
		if err := runTo(w, at(day, "12:00")); err != nil {
			t.Fatal(err)
		}
		s1, s2 := studyByID(t, w, "S00001"), studyByID(t, w, "S00002")
		if s1.Claimed != at(day, "08:41") || s2.Entered != at(day, "09:11") || s2.Claimed != at(day, tc.claim) {
			t.Fatalf("%s: S00001 claimed %s, S00002 entered %s claimed %s (want %s)", tc.file,
				w.Calendar().Clock(s1.Claimed), w.Calendar().Clock(s2.Entered), w.Calendar().Clock(s2.Claimed), tc.claim)
		}
	}
}

// Radiologist distinctions (ROADMAP §2): a remote radiologist in a site
// block reads but never supervises; an on-site radiologist in an
// other-duty block supervises contrast but claims no reads.
func TestRadiologistDistinctions(t *testing.T) {
	w, day := micro(t, "micro-rads.json")
	if err := runTo(w, at(day, "12:00")); err != nil {
		t.Fatal(err)
	}
	contrast := 0
	for _, v := range w.Visits() {
		if v.Exam == "CT_C" {
			contrast++
			if v.Supervisor != "Dr. Bo" {
				t.Fatalf("%s supervised by %q, want the on-site Dr. Bo", v.ID, v.Supervisor)
			}
		}
	}
	for _, s := range w.Studies() {
		if s.By != "Dr. Rae" {
			t.Fatalf("%s read by %q, want remote Dr. Rae (Dr. Bo is in other duties)", s.ID, s.By)
		}
	}
	if contrast != 2 || len(w.Studies()) != 4 {
		t.Fatalf("%d contrast exams, %d studies", contrast, len(w.Studies()))
	}
}

// Reordering the worklist never changes a case's inherent read time: the
// same studies read under FIFO and ShortestFirst take exactly as long.
func TestReadTimeIndependentOfPolicy(t *testing.T) {
	base := miniConfig(t)
	alt := *base
	alt.Policies.Worklist = "shortest_first"
	a, b := run(t, base, 6, 10), run(t, &alt, 6, 10)
	rt := map[string]simtime.Duration{}
	for _, s := range a.Studies() {
		if s.State == "final" {
			rt[s.ID+s.Exam] = s.ReadTime
		}
	}
	same, reordered := 0, 0
	for _, s := range b.Studies() {
		if d, ok := rt[s.ID+s.Exam]; ok && s.State == "final" {
			if d != s.ReadTime {
				t.Fatalf("%s read %s under FIFO, %s under shortest-first", s.ID, d, s.ReadTime)
			}
			same++
		}
	}
	for i, s := range a.Studies() {
		if s.State == "final" && b.Studies()[i].Claimed != s.Claimed {
			reordered++
		}
	}
	if same < 200 || reordered == 0 {
		t.Fatalf("%d studies compared, %d claimed at different times (policy had no effect?)", same, reordered)
	}
}

// Hidden state: no observable type carries a latent or future value
// (read duration, remaining time, patience, no-show, arrival offset, a
// claim's payer outcome, remittance date or charge lag).
func TestObservationHasNoLatentFields(t *testing.T) {
	deny := []string{"latent", "remaining", "willnoshow", "patience", "readsec", "readtime", "sampled", "cancelat", "arrivaloffset", "endsin",
		"outcome", "willpay", "willdeny", "submitdue", "adjudicationat", "remitat", "paydate", "fracbp", "reworkdays", "delaydays", "chargelag"}
	seen := map[reflect.Type]bool{}
	var walk func(reflect.Type, string)
	walk = func(ty reflect.Type, path string) {
		for ty.Kind() == reflect.Pointer || ty.Kind() == reflect.Slice {
			ty = ty.Elem()
		}
		if ty.Kind() != reflect.Struct || seen[ty] {
			return
		}
		seen[ty] = true
		for i := 0; i < ty.NumField(); i++ {
			f := ty.Field(i)
			for _, d := range deny {
				if strings.Contains(strings.ToLower(f.Name), d) {
					t.Errorf("%s.%s looks like hidden state", path, f.Name)
				}
			}
			walk(f.Type, path+"."+f.Name)
		}
	}
	for _, ty := range []reflect.Type{reflect.TypeOf(api.Observation{}), reflect.TypeOf(api.DaySchedule{}),
		reflect.TypeOf(api.PatientRow{}), reflect.TypeOf(api.Journey{}), reflect.TypeOf(api.Finance{})} {
		walk(ty, ty.Name())
	}
}
