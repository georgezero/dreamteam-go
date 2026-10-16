package sim_test

import (
	"slices"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/sim"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// V0.2 Slice 4: set_reading_allocation (V0.2_DESIGN §5.10, §6.3, §8.1).
// The established practice is taken over on Tue 2026-10-13; with 14 days'
// notice the earliest effective date is Tue Oct 27 (any weekday). Dr.
// Avery (E07, index 6) reads Mon–Fri 07:30–16:00 at 60%, Dr. Blake (E08,
// index 7) Mon–Fri 11:30–20:00 at 50%; each has a $480,000 salary.

const avery, blake = 6, 7

func setAlloc(id, effective string, bp int64) api.Action {
	return api.Action{Type: api.ActSetReadingAllocation, Op: api.RosterOpSet, Employee: id, Effective: effective, SiteAllocationBP: bp}
}

func removeAlloc(id, effective string) api.Action {
	return api.Action{Type: api.ActSetReadingAllocation, Op: api.RosterOpRemove, Employee: id, Effective: effective}
}

// siteMin is employee i's site reading time on date, in minutes.
func siteMin(w *sim.World, i int, date string) float64 {
	var n simtime.Duration
	for _, b := range w.PlanOn(dayOf(w, date)).Reading[i] {
		n += b.To - b.From
	}
	return float64(n) / float64(simtime.Minute)
}

// Effective dates: at least 14 days ahead on any weekday, within the
// horizon; never today or the past. Every refusal leaves the run exactly
// as it was.
func TestAllocationEffectiveDates(t *testing.T) {
	w := atTakeover(t, `{}`, 1)
	before := digestsOf(w)
	for _, c := range []struct{ date, code string }{
		{"2026-10-13", "past_date"},
		{"2026-10-12", "past_date"},
		{"2026-10-26", "notice_too_short"}, // 13 days
		{"2026-10-1", "invalid_date"},
		{"2027-03-01", "beyond_horizon"}, // 139 days
	} {
		res := w.Apply([]api.Action{setAlloc("E07", c.date, 6250)})
		if res[0].OK || res[0].Code != c.code {
			t.Errorf("%s: %+v", c.date, res[0])
		}
	}
	if digestsOf(w) != before {
		t.Fatal("a refused set_reading_allocation changed the run")
	}
	res := mustApply(t, w, setAlloc("E07", "2026-10-27", 6250)) // a Tuesday
	if res[0].Code != api.ActionAccepted || res[0].EffectiveAt != "2026-10-27" {
		t.Fatalf("%+v", res[0])
	}
	// The day before keeps the old blocks; the date itself has the new.
	if a, b := siteMin(w, avery, "2026-10-26"), siteMin(w, avery, "2026-10-27"); a != 310 || b != 322.5 {
		t.Fatalf("site reading minutes %v on Oct 26, %v on Oct 27; want 310 and 322.5", a, b)
	}
}

// Shape, role and bounds. The bounds are inclusive; refusals change nothing.
func TestAllocationValidation(t *testing.T) {
	w := atTakeover(t, `{}`, 2)
	before := digestsOf(w)
	for _, c := range []struct {
		name string
		a    api.Action
		code string
	}{
		{"technologist", setAlloc("E03", "2026-10-27", 6000), "wrong_role"},
		{"ops manager", setAlloc("E02", "2026-10-27", 6000), "wrong_role"},
		{"front desk", setAlloc("E01", "2026-10-27", 6000), "wrong_role"},
		{"unknown", setAlloc("E99", "2026-10-27", 6000), "unknown_employee"},
		{"below the floor", setAlloc("E07", "2026-10-27", 2999), "allocation_out_of_band"},
		{"above whole time", setAlloc("E07", "2026-10-27", 10001), "allocation_out_of_band"},
		{"set without a share", setAlloc("E07", "2026-10-27", 0), "invalid_schema"},
		{"remove with a share", api.Action{Type: api.ActSetReadingAllocation, Op: api.RosterOpRemove, Employee: "E07", Effective: "2026-10-27", SiteAllocationBP: 6000}, "invalid_schema"},
		{"no op", api.Action{Type: api.ActSetReadingAllocation, Employee: "E07", Effective: "2026-10-27", SiteAllocationBP: 6000}, "invalid_schema"},
		{"roster field", api.Action{Type: api.ActSetReadingAllocation, Op: api.RosterOpSet, Employee: "E07", Effective: "2026-10-27", SiteAllocationBP: 6000, Minutes: 30}, "invalid_schema"},
		{"no employee", api.Action{Type: api.ActSetReadingAllocation, Op: api.RosterOpSet, Effective: "2026-10-27", SiteAllocationBP: 6000}, "invalid_schema"},
	} {
		if res := w.Apply([]api.Action{c.a}); res[0].OK || res[0].Code != c.code {
			t.Errorf("%s: %+v", c.name, res[0])
		}
	}
	// A share field on another V0.2 type is a schema error too.
	h := hireOf("r1", "technologist", []string{"CT"}, 4000, 2400, weekOf(shiftAt("R3", "07:00", "15:30", "11:00", "11:30"), "Mon", "Tue", "Wed", "Thu", "Fri"), "2026-11-10")
	h.SiteAllocationBP = 6000
	if res := w.Apply([]api.Action{h}); res[0].Code != "invalid_schema" {
		t.Errorf("hire with a share: %+v", res[0])
	}
	// Mixed with a V0.1 action: the whole batch is refused.
	res := w.Apply([]api.Action{setAlloc("E07", "2026-10-27", 6250), {Type: api.ActSetWorklistPolicy, Policy: api.WorklistFIFO}})
	if c := codes(res); !slices.Equal(c, []string{"batch_rejected", "mixed_batch"}) {
		t.Errorf("mixed: %v", c)
	}
	// Two writes to one radiologist and date, in either order.
	res = w.Apply([]api.Action{setAlloc("E07", "2026-10-27", 6250), setAlloc("E07", "2026-10-27", 6500)})
	if c := codes(res); !slices.Equal(c, []string{"batch_rejected", "invalid_parameters"}) {
		t.Errorf("same date twice: %v", c)
	}
	res = w.Apply([]api.Action{setAlloc("E07", "2026-10-27", 6000), removeAlloc("E07", "2026-10-27")})
	if c := codes(res); !slices.Equal(c, []string{"batch_rejected", "invalid_parameters"}) {
		t.Errorf("no-op then remove on one date: %v", c)
	}
	if digestsOf(w) != before {
		t.Fatal("a refused batch changed the run")
	}
	// The bounds themselves are lawful.
	mustApply(t, w, setAlloc("E07", "2026-10-27", 3000), setAlloc("E08", "2026-10-27", 10000))
	if a, b := siteMin(w, avery, "2026-10-27"), siteMin(w, blake, "2026-10-27"); a != 160 || b != 510 {
		t.Fatalf("30%%: %v min (want 160); 100%%: %v min (want 510, the whole shift)", a, b)
	}
}

// Versions resolve by date; a pending version can be replaced or removed
// (its predecessor resumes); identical requests are no-ops; a version in
// effect or past never changes.
func TestAllocationVersions(t *testing.T) {
	w := atTakeover(t, `{}`, 3)
	w.Engine().CheckInvariants = true
	// Two successive versions in one batch, out of date order.
	mustApply(t, w, setAlloc("E07", "2026-11-10", 7000), setAlloc("E07", "2026-10-27", 6250))
	for _, c := range []struct {
		date string
		want float64
	}{
		{"2026-10-26", 310}, {"2026-10-27", 322.5}, {"2026-11-09", 322.5}, {"2026-11-10", 360}, {"2027-01-04", 360},
	} {
		if got := siteMin(w, avery, c.date); got != c.want {
			t.Errorf("%s: %v site minutes, want %v", c.date, got, c.want)
		}
	}
	before := digestsOf(w)
	for _, c := range []struct {
		name string
		a    api.Action
	}{
		{"repeat", setAlloc("E07", "2026-10-27", 6250)},
		{"same as in force", setAlloc("E07", "2026-11-02", 6250)},
		{"same as the scenario's", setAlloc("E08", "2026-11-02", 5000)},
		{"remove absent", removeAlloc("E07", "2026-11-03")},
	} {
		if res := w.Apply([]api.Action{c.a}); res[0].Code != api.ActionUnchanged || !res[0].OK {
			t.Errorf("%s: %+v", c.name, res[0])
		}
	}
	if s := digestsOf(w); s[0] != before[0] || s[1] != before[1] || s[2] != before[2] {
		t.Fatal("a no-op changed the run")
	}
	// Replace the second, then withdraw it: the first resumes.
	mustApply(t, w, setAlloc("E07", "2026-11-10", 6500))
	if got := siteMin(w, avery, "2026-11-10"); got != 335 {
		t.Fatalf("replaced: %v", got)
	}
	mustApply(t, w, removeAlloc("E07", "2026-11-10"))
	if got := siteMin(w, avery, "2026-11-10"); got != 322.5 {
		t.Fatalf("withdrawn: %v", got)
	}
	// Once in effect a version is history: it can neither be removed nor
	// replaced, and a removal needs the same notice as a change.
	runChecked(t, w, w.Calendar().StartOfDay(dayOf(w, "2026-10-28")).Add(9*simtime.Hour))
	for _, c := range []struct {
		a    api.Action
		code string
	}{
		{removeAlloc("E07", "2026-10-27"), "past_date"},
		{setAlloc("E07", "2026-10-27", 6000), "past_date"},
		{setAlloc("E07", "2026-11-10", 6000), "notice_too_short"},
	} {
		if res := w.Apply([]api.Action{c.a}); res[0].Code != c.code {
			t.Errorf("%+v: %+v", c.a, res[0])
		}
	}
	if got := employee(w, "E07").SiteBP; got != 6250 {
		t.Fatalf("observed allocation in effect: %d", got)
	}
}

// Salary follows the allocation in force each day, at the existing daily
// rounding, with benefits; one salary accrual per radiologist per day; two
// changes are two date-bounded rates; the books reconcile.
func TestAllocationSalaryProration(t *testing.T) {
	w := atTakeover(t, `{}`, 5)
	mustApply(t, w, setAlloc("E07", "2026-10-27", 6250), setAlloc("E08", "2026-10-28", 5250))
	runChecked(t, w, w.Calendar().StartOfDay(dayOf(w, "2026-10-20")).Add(10*simtime.Hour))
	mustApply(t, w, setAlloc("E07", "2026-11-10", 6000))
	end := dayOf(w, "2026-11-20")
	runChecked(t, w, w.Calendar().StartOfDay(end))
	bp := w.Config().Finance.BenefitsBP
	for _, c := range []struct {
		emp  int64
		rate func(d int64) int64 // annual site salary on day d
	}{
		{avery, func(d int64) int64 {
			switch {
			case d < dayOf(w, "2026-10-27"):
				return 28_800_000
			case d < dayOf(w, "2026-11-10"):
				return 30_000_000
			}
			return 28_800_000
		}},
		{blake, func(d int64) int64 {
			if d < dayOf(w, "2026-10-28") {
				return 24_000_000
			}
			return 25_200_000
		}},
	} {
		seen := map[int64]int{}
		for _, e := range entries(w, "salary", c.emp) {
			seen[e.B]++
			want := sim.SalaryAccrual(c.rate(e.B), e.B)
			if got := lineAmt(e, "payroll"); got != want {
				t.Fatalf("emp %d day %d: salary %d, want %d", c.emp, e.B, got, want)
			}
			if got := lineAmt(e, "benefits"); got != (want*bp+5_000)/10_000 {
				t.Fatalf("emp %d day %d: benefits %d", c.emp, e.B, got)
			}
		}
		for d := dayOf(w, "2026-10-13"); d < end; d++ {
			if seen[d] != 1 {
				t.Fatalf("emp %d day %d: %d salary accruals", c.emp, d, seen[d])
			}
		}
	}
	if !w.Evaluation().Money.Reconciled {
		t.Fatal("the evaluation's cash bridge does not reconcile")
	}
}

// Supervision is presence and booking is technologist coverage: neither
// moves with the allocation. Every future day's booking view — days built
// before the change, days built after it as the horizon advances, and days
// a later hire rebuilds — and every day's supervision spans are
// bit-identical with and without the change, and no booking is displaced.
func TestAllocationSupervisionAndBookingInvariant(t *testing.T) {
	a, b := atTakeover(t, `{}`, 6), atTakeover(t, `{}`, 6)
	mustApply(t, b, setAlloc("E07", "2026-10-27", 10000), setAlloc("E08", "2026-10-27", 3000))
	ct := weekOf(shiftAt("R3", "07:00", "15:30", "11:00", "11:30"), "Mon", "Tue", "Wed", "Thu", "Fri")
	end := a.Calendar().StartOfDay(dayOf(a, "2026-10-29")).Add(7 * simtime.Hour)
	for _, w := range []*sim.World{a, b} {
		runChecked(t, w, end)
		mustApply(t, w, hireOf("ct", "technologist", []string{"CT"}, 4000, 2400, ct, "2026-11-30"))
	}
	if a.DaysBuilt() != b.DaysBuilt() {
		t.Fatal("different horizons")
	}
	for d := dayOf(a, "2026-10-29"); d < a.DaysBuilt(); d++ {
		if a.BookingView(d) != b.BookingView(d) {
			t.Fatalf("day %d: booking coverage differs", d)
		}
		if sa, sb := a.SupervisionOn(d), b.SupervisionOn(d); !slices.Equal(sa, sb) {
			t.Fatalf("day %d: supervision %v, with the change %v", d, sa, sb)
		}
	}
	if siteMin(b, avery, "2026-10-27") == siteMin(a, avery, "2026-10-27") {
		t.Fatal("the change did not change reading")
	}
	if n := len(b.Displacements()); n != 0 {
		t.Fatalf("%d bookings displaced", n)
	}
}

// Reading after an increase: claims only inside the blocks in force that
// day, never past the shift end, and never a study of a lower class while
// an urgent study that fits waits (the default policy promotes overdue
// studies to the urgent class); the same studies are reported sooner.
func TestAllocationReadingRules(t *testing.T) {
	a, b := atTakeover(t, `{}`, 7), atTakeover(t, `{}`, 7)
	b.Engine().CheckInvariants = true
	mustApply(t, b, setAlloc("E07", "2026-10-27", 8000), setAlloc("E08", "2026-10-27", 7000))
	cal := b.Calendar()
	end := cal.StartOfDay(dayOf(b, "2026-11-10"))
	runChecked(t, a, end)
	runChecked(t, b, end)
	cfg := b.Config()
	idx := map[string]int{}
	for i, e := range cfg.Employees {
		idx[e.Name] = i
	}
	p100 := map[string]simtime.Duration{}
	for _, x := range cfg.Catalog {
		p100[x.Code] = simtime.Duration(x.ReadTime.Max())
	}
	target := map[string]simtime.Duration{}
	for p := range sim.Priority(3) {
		target[p.String()] = cfg.Policies.TATTarget[p]
	}
	studies := b.Studies()
	from := cal.StartOfDay(dayOf(b, "2026-10-27"))
	for _, s := range studies {
		if s.State != "final" || s.Claimed < from {
			continue
		}
		i := idx[s.By]
		plan := b.PlanOn(cal.DayIndex(s.Claimed))
		in := false
		for _, blk := range plan.Reading[i] {
			in = in || blk.Contains(simtime.ClockOf(s.Claimed))
		}
		sh := plan.Shifts[plan.ByEmp[i]]
		if !in || simtime.ClockOf(s.Final) > sh.To {
			t.Fatalf("%s claimed %s by %s outside the blocks in force or read past the shift", s.ID, cal.Short(s.Claimed), s.By)
		}
		if s.Priority == "urgent" || s.Claimed >= s.Acquired.Add(target[s.Priority]) {
			continue // the urgent class
		}
		for _, u := range studies {
			if u.Priority == "urgent" && u.Entered <= s.Claimed && (u.State != "final" || u.Claimed > s.Claimed) &&
				s.Claimed.Add(p100[u.Exam]) <= cal.StartOfDay(cal.DayIndex(s.Claimed)).Add(sh.To) {
				t.Fatalf("%s (%s) claimed at %s while urgent %s waited", s.ID, s.Priority, cal.Short(s.Claimed), u.ID)
			}
		}
	}
	// Imaging does not depend on reading: the same studies, read sooner.
	tat := map[string]simtime.Duration{}
	for _, s := range a.Studies() {
		if s.State == "final" && s.Acquired >= from {
			tat[s.ID] = s.Final.Sub(s.Acquired)
		}
	}
	var without, with simtime.Duration
	n := 0
	for _, s := range studies {
		if was, ok := tat[s.ID]; ok && s.State == "final" {
			without += was
			with += s.Final.Sub(s.Acquired)
			n++
		}
	}
	if n < len(tat)*9/10 || with >= without {
		t.Fatalf("%d of %d studies compared: total turnaround %s with the increase, %s without", n, len(tat), with, without)
	}
}

// A mixed staffing and allocation batch commits atomically; the same batch
// with one invalid item commits nothing.
func TestAllocationMixedBatch(t *testing.T) {
	ct := weekOf(shiftAt("R3", "07:00", "15:30", "11:00", "11:30"), "Mon", "Tue", "Wed", "Thu", "Fri")
	batch := func(bp int64) []api.Action {
		return []api.Action{
			hireOf("ct", "technologist", []string{"XR", "CT"}, 3800, 2400, ct, "2026-11-10"),
			terminateOf("E04", "2026-11-20"),
			setAlloc("E07", "2026-10-27", 6250),
			setAlloc("E08", "2026-10-27", bp),
		}
	}
	w := atTakeover(t, `{}`, 8)
	before := digestsOf(w)
	res := w.Apply(batch(2500))
	if c := codes(res); !slices.Equal(c, []string{"batch_rejected", "batch_rejected", "batch_rejected", "allocation_out_of_band"}) {
		t.Fatalf("%v", c)
	}
	if digestsOf(w) != before || len(w.Observe(0).Staff) != 8 {
		t.Fatal("a refused batch changed the run")
	}
	res = mustApply(t, w, batch(5250)...)
	if c := codes(res); !slices.Equal(c, []string{"accepted", "accepted", "accepted", "accepted"}) {
		t.Fatalf("%v", c)
	}
	if siteMin(w, blake, "2026-10-27") != 272.5 || employee(w, "E04").Leaving == nil || len(w.Observe(0).Staff) != 9 {
		t.Fatal("the batch did not commit as a whole")
	}
}

// Chunked and uninterrupted runs across the effective date agree exactly.
func TestAllocationChunking(t *testing.T) {
	end := func(w *sim.World) simtime.Time { return w.Calendar().StartOfDay(dayOf(w, "2026-11-05")) }
	a, b := atTakeover(t, `{}`, 9), atTakeover(t, `{}`, 9)
	for _, w := range []*sim.World{a, b} {
		mustApply(t, w, setAlloc("E07", "2026-10-27", 6250), setAlloc("E08", "2026-10-29", 5250))
	}
	a.Engine().RunUntil(end(a), nil)
	for b.Engine().Now() < end(b) {
		b.Engine().RunUntil(min(end(b), b.Engine().Now().Add(13*simtime.Hour)), nil)
	}
	if digestsOf(a) != digestsOf(b) {
		t.Fatal("chunked run differs")
	}
}

// Previewing is read-only.
func TestAllocationPreviewReadOnly(t *testing.T) {
	w := atTakeover(t, `{}`, 10)
	before := digestsOf(w)
	for range 3 {
		p := w.PreviewStaffing([]api.Action{setAlloc("E07", "2026-10-27", 6250)})
		if !p.OK {
			t.Fatalf("%+v", p.Results)
		}
	}
	if digestsOf(w) != before {
		t.Fatal("a preview changed the run")
	}
}
