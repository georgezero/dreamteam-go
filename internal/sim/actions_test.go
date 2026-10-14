package sim_test

import (
	"math/rand/v2"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/rng"
	"github.com/georgezero/dreamteam-go/internal/scenario"
	"github.com/georgezero/dreamteam-go/internal/sim"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Step 13: the three V0.1 actions through World.Apply, the one entry point
// for the TUI and agents.

func extend(id string, min int64) api.Action {
	return api.Action{Type: api.ActExtendShift, Employee: id, Minutes: min}
}

func walkIn(policy, cutoff string) api.Action {
	return api.Action{Type: api.ActSetWalkInPolicy, Policy: policy, Cutoff: cutoff}
}

func worklist(policy string) api.Action {
	return api.Action{Type: api.ActSetWorklistPolicy, Policy: policy}
}

func applyAt(t *testing.T, w *sim.World, at simtime.Time, as ...api.Action) []api.ActionResult {
	t.Helper()
	if err := runTo(w, at); err != nil {
		t.Fatal(err)
	}
	return w.Apply(as)
}

func staffID(o api.Observation, name string) api.Employee {
	for _, e := range o.Staff {
		if e.Name == name {
			return e
		}
	}
	panic("no " + name)
}

// miniDay starts the mini practice (Tuesday 08:00–20:00; Morgan, the only
// US technologist, 09:30–18:00, cap 21:30).
func miniDay(t *testing.T, seed int64) (*sim.World, simtime.Time) {
	t.Helper()
	cfg := miniConfig(t)
	w := sim.New(cfg, rng.SeedFromInt(seed))
	w.Engine().CheckInvariants = true
	return w, w.Calendar().StartOfDate(cfg.Start)
}

// Validation: every rejection has a code and a message, and none changes
// the state; the accepted ones do what they say.
func TestExtendShiftValidation(t *testing.T) {
	w, d0 := miniDay(t, 1)
	cases := []struct {
		at   string
		a    api.Action
		ok   bool
		code string
	}{
		{"07:00", extend("E99", 60), false, "unknown_employee"},
		{"07:00", extend("E06", 0), false, "invalid_minutes"},
		{"07:00", extend("E06", 241), false, "invalid_minutes"},
		{"07:00", extend("E03", 240), false, "exceeds_max_shift"}, // Riley 16:15 + 4h = 20:15 > cap 19:45
		{"07:00", extend("E04", 30), false, "after_close"},        // Sam already ends at 20:00
		{"07:00", extend("E06", 60), true, api.ActionAccepted},    // Morgan to 19:00
		{"07:00", extend("E06", 60), true, api.ActionAccepted},    // and again, to 20:00
		{"07:00", extend("E06", 60), false, "after_close"},        // now ending at close
		{"07:00", api.Action{Type: "hire", Employee: "E01"}, false, "unknown_action"},
		{"07:00", walkIn("accept_sometimes", ""), false, "unknown_policy"},
		{"07:00", walkIn(api.WalkInAcceptUntil, ""), false, "invalid_cutoff"},
		{"07:00", walkIn(api.WalkInAcceptIfGap, "18:00"), false, "invalid_cutoff"},
		{"07:00", walkIn(api.WalkInAcceptIfGap, ""), true, api.ActionUnchanged},
		{"07:00", worklist("random"), false, "unknown_policy"},
		{"07:00", worklist(api.WorklistUrgentThenFIFO), true, api.ActionUnchanged},
		{"16:30", extend("E05", 30), false, "shift_ended"}, // Casey ended at 16:30: no retroactive approval
	}
	for _, c := range cases {
		at := at(d0, c.at)
		if w.Engine().Now() > at {
			t.Fatalf("cases out of order at %s", c.at)
		}
		r := applyAt(t, w, at, c.a)[0]
		if r.OK != c.ok || r.Code != c.code || r.Message == "" {
			t.Errorf("%s %+v: got %+v", c.at, c.a, r)
		}
	}
	o := w.Observe(0)
	if m := staffID(o, "Morgan"); m.Shift != "09:30–20:00" || m.ApprovedMin != 120 || m.RosterEnd != "18:00" {
		t.Fatalf("Morgan %+v", m)
	}
	if o.Actions.Attempted != len(cases) || o.Actions.Accepted != 4 || o.Actions.Rejected != len(cases)-4 {
		t.Fatalf("action log %+v", o.Actions)
	}
}

// An approved extension adds capacity; required hold-over only finishes.
// micro-extend: Tess (XR) 07:30–09:00, practice open until 10:00, four
// patients 08:00–08:45. Without the action she is held over to finish
// them (09:36, 36 minutes of required overtime) and XR is unstaffed after.
// With +60 approved in advance she stays on regular duty to 10:00: 60
// minutes of approved overtime, none required, XR staffed all day.
func TestExtensionVersusHoldOver(t *testing.T) {
	plain, d0 := micro(t, "micro-extend.json")
	ext, _ := micro(t, "micro-extend.json")
	if r := applyAt(t, ext, at(d0, "07:00"), extend("T1", 60))[0]; !r.OK {
		t.Fatal(r.Message)
	}
	for _, w := range []*sim.World{plain, ext} {
		if err := runTo(w, d0.Add(simtime.Day)); err != nil {
			t.Fatal(err)
		}
	}
	a, b := day(t, plain, 0).Ops, day(t, ext, 0).Ops
	ta, tb := person(a, "Tess"), person(b, "Tess")
	if ta.OvertimeSec != 36*60 || ta.ApprovedOvertimeSec != 0 || a.Workforce.ApprovedOvertimeSec != 0 || a.Workforce.HeldOver != 1 {
		t.Fatalf("hold-over: %+v %+v", ta, a.Workforce)
	}
	if tb.OvertimeSec != 60*60 || tb.ApprovedOvertimeSec != 60*60 || b.Workforce.OvertimeSec != 3600 ||
		b.Workforce.ApprovedOvertimeSec != 3600 || b.Workforce.HeldOver != 0 {
		t.Fatalf("extension: %+v %+v", tb, b.Workforce)
	}
	if room(a, "R1").StaffedSec >= room(b, "R1").StaffedSec || room(b, "R1").StaffedSec != 7200 {
		t.Fatalf("XR staffed %d without, %d with the extension", room(a, "R1").StaffedSec, room(b, "R1").StaffedSec)
	}
	// Payroll: one wage entry for Tess today, the hourly pay of her paid
	// time (on-duty time), and the approved overtime cost is all of it.
	for _, c := range []struct {
		w    *sim.World
		tess api.StaffMetrics
		ops  api.PeriodMetrics
	}{{plain, ta, a}, {ext, tb, b}} {
		var wages int64
		n := 0
		for _, e := range c.w.Journal() {
			if e.Kind == "wages" && e.A == 0 {
				n++
				for _, l := range e.Lines {
					if l.Account == "payroll" {
						wages += l.Amt
					}
				}
			}
		}
		want, _ := sim.HourlyPay(6000, 0, simtime.Duration(c.tess.OnDutySec), 40*simtime.Hour, 15000)
		if n != 1 || wages != want {
			t.Fatalf("Tess wages %d in %d entries, want %d", wages, n, want)
		}
	}
	if b.Workforce.ApprovedOvertimeCostCents != b.Workforce.OvertimeCostCents || b.Workforce.OvertimeCostCents != 6000 {
		t.Fatalf("overtime cost %+v", b.Workforce)
	}
}

// The extension changes what happens next — US capacity after 18:00 —
// but never the demand: the same orders and walk-ins arrive at the same
// times with the same attributes.
func TestExtensionAddsCapacityNotDemand(t *testing.T) {
	plain, d0 := miniDay(t, 5)
	ext, _ := miniDay(t, 5)
	if r := applyAt(t, ext, at(d0, "07:00"), extend("E06", 120))[0]; !r.OK {
		t.Fatal(r.Message)
	}
	// Today's US slots 18:00–20:00 are bookable now.
	if err := runTo(plain, at(d0, "07:00")); err != nil {
		t.Fatal(err)
	}
	count := func(w *sim.World) (open int) {
		for _, rs := range w.Schedule(w.Calendar().DayIndex(d0)).Rooms {
			if rs.Modality != "US" {
				continue
			}
			for _, s := range rs.Slots {
				if s.From >= 18*simtime.Hour && (s.Status == api.SlotOpen || s.Status == api.SlotBooked) {
					open++
				}
			}
		}
		return open
	}
	if count(plain) != 0 || count(ext) == 0 {
		t.Fatalf("US slots after 18:00: %d without, %d with the extension", count(plain), count(ext))
	}
	for _, w := range []*sim.World{plain, ext} {
		if err := runTo(w, d0.Add(3*simtime.Day)); err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(exogenous(plain.Orders()), exogenous(ext.Orders())) {
		t.Fatal("the action changed exogenous demand")
	}
	us := func(w *sim.World) api.RoomMetrics {
		for _, r := range day(t, w, 0).Ops.Rooms {
			if r.Modality == "US" {
				return r
			}
		}
		panic("no US")
	}
	if us(ext).StaffedSec-us(plain).StaffedSec != 7200 || reflect.DeepEqual(plain.Visits(), ext.Visits()) {
		t.Fatalf("US staffed %d vs %d; visits identical: the extension had no effect", us(plain).StaffedSec, us(ext).StaffedSec)
	}
	if day(t, ext, 0).Ops.Workforce.ApprovedOvertimeSec != 7200 {
		t.Fatalf("approved overtime %+v", day(t, ext, 0).Ops.Workforce)
	}
}

// Walk-in policy: the same walk-ins arrive whatever the policy; declined
// ones are counted apart and still become orders offered appointments, so
// they stay in the demand figures. accept_until declines exactly those
// arriving from the cutoff.
func TestWalkInPolicies(t *testing.T) {
	plain, d0 := miniDay(t, 7)
	decline, _ := miniDay(t, 7)
	until, _ := miniDay(t, 7)
	applyAt(t, decline, at(d0, "07:00"), walkIn(api.WalkInScheduleLater, ""))
	applyAt(t, until, at(d0, "07:00"), walkIn(api.WalkInAcceptUntil, "14:00"))
	for _, w := range []*sim.World{plain, decline, until} {
		if err := runTo(w, d0.Add(simtime.Day-2)); err != nil {
			t.Fatal(err)
		}
	}
	walkIns := func(w *sim.World) (all, late []sim.OrderInfo) {
		for _, o := range w.Orders() {
			if o.WalkIn && o.Received >= d0 {
				o.State = ""
				all = append(all, o)
				if o.Received >= at(d0, "14:00") {
					late = append(late, o)
				}
			}
		}
		return
	}
	pa, _ := walkIns(plain)
	da, _ := walkIns(decline)
	ua, late := walkIns(until)
	if len(pa) == 0 || !reflect.DeepEqual(pa, da) || !reflect.DeepEqual(pa, ua) || len(late) == 0 {
		t.Fatalf("walk-in arrivals changed with the policy (%d, %d, %d)", len(pa), len(da), len(ua))
	}
	fp, fd, fu := plain.Observe(0).Flow.Today, decline.Observe(0).Flow.Today, until.Observe(0).Flow.Today
	if fp.WalkInsDeclined != 0 || fd.WalkInsDeclined != len(da) || fd.WalkInsBooked < fd.WalkInsDeclined ||
		fu.WalkInsDeclined != len(late) || fd.WalkIns != fp.WalkIns {
		t.Fatalf("declined: plain %+v, decline %+v, until %+v (late arrivals %d)", fp, fd, fu, len(late))
	}
	for _, v := range decline.Visits() {
		if v.WalkIn {
			t.Fatalf("a declined walk-in was admitted: %+v", v)
		}
	}
	// Declined walk-ins remain demand: each became an order offered
	// appointments and is counted once as booked, put on recall or lost.
	offers := func(w *sim.World) (n int) {
		for _, m := range w.Observe(0).Demand.Modalities {
			n += m.Today.Booked + m.Today.Recall + m.Today.Lost
		}
		return n
	}
	if offers(decline)-offers(plain) < len(da) || offers(until)-offers(plain) < len(late) {
		t.Fatalf("offers %d (plain), %d (decline all %d), %d (until, %d late)", offers(plain), offers(decline), len(da), offers(until), len(late))
	}
	booked := map[string]bool{}
	for _, ap := range decline.Appointments() {
		booked[ap.Order] = true
	}
	for _, o := range decline.Orders() {
		if o.WalkIn && o.Received >= d0 && !booked[o.ID] && o.State != "lost" && o.State != "recall" {
			t.Fatalf("declined walk-in %s neither booked, on recall nor lost: %q", o.ID, o.State)
		}
	}
	if m := day(t, decline, 0).Ops.WalkIns; m.Declined != len(da) || m.Arrived != len(da) || m.Acceptance.Num != 0 {
		t.Fatalf("walk-in metrics %+v", m)
	}
}

// A policy change never removes a walk-in already admitted.
func TestWalkInPolicyNotRetroactive(t *testing.T) {
	w, d0 := miniDay(t, 7)
	for now := at(d0, "08:00"); now < at(d0, "20:00"); now = now.Add(5 * simtime.Minute) {
		if err := runTo(w, now); err != nil {
			t.Fatal(err)
		}
		var waiting []string
		for _, p := range w.Observe(0).Flow.Waiting {
			if p.WalkIn {
				waiting = append(waiting, p.Visit)
			}
		}
		if len(waiting) == 0 {
			continue
		}
		w.Apply([]api.Action{walkIn(api.WalkInScheduleLater, "")})
		if err := runTo(w, d0.Add(simtime.Day-2)); err != nil {
			t.Fatal(err)
		}
		for _, v := range w.Visits() {
			for _, id := range waiting {
				if v.ID == id && v.State != "done" && v.State != "left" {
					t.Fatalf("%s ended %s after the policy change", id, v.State)
				}
			}
		}
		return
	}
	t.Fatal("no walk-in ever waited: the test is vacuous")
}

// Worklist policy: ordering changes, nothing else does — every study is
// read once, by its own read time, and reads in progress run on.
func TestWorklistPolicyOrdering(t *testing.T) {
	plain, d0 := miniDay(t, 3)
	short, _ := miniDay(t, 3)
	if err := runTo(plain, at(d0, "11:00")); err != nil {
		t.Fatal(err)
	}
	if err := runTo(short, at(d0, "11:00")); err != nil {
		t.Fatal(err)
	}
	var reading []sim.StudyInfo
	for _, s := range short.Studies() {
		if s.State == "reading" {
			reading = append(reading, s)
		}
	}
	short.Apply([]api.Action{worklist(api.WorklistShortestFirst)})
	// The offered order now follows the published expected read time.
	cfg := miniConfig(t)
	p50 := map[string]int64{}
	for _, x := range cfg.Catalog {
		p50[x.Name] = x.ReadTime.At(500_000)
	}
	items := short.Observe(0).Worklist.Items
	for i := 1; i < len(items); i++ {
		a, b := items[i-1], items[i]
		if a.State == api.StudyUnread && b.State == api.StudyUnread && p50[a.ExamName] > p50[b.ExamName] {
			t.Fatalf("shortest-first offers %s (%ds) before %s (%ds)", a.Study, p50[a.ExamName], b.Study, p50[b.ExamName])
		}
	}
	for _, w := range []*sim.World{plain, short} {
		if err := runTo(w, d0.Add(simtime.Day)); err != nil {
			t.Fatal(err)
		}
	}
	ps, ss := plain.Studies(), short.Studies()
	if len(ps) != len(ss) {
		t.Fatalf("%d vs %d studies", len(ps), len(ss))
	}
	differ := false
	for i := range ps {
		if ps[i].ID != ss[i].ID || ps[i].Acquired != ss[i].Acquired {
			t.Fatalf("imaging changed: %+v vs %+v", ps[i], ss[i])
		}
		if ps[i].State == "final" && ss[i].State == "final" && ps[i].ReadTime != ss[i].ReadTime {
			t.Fatalf("%s read in %s vs %s: the policy changed an inherent read time", ps[i].ID, ps[i].ReadTime, ss[i].ReadTime)
		}
		differ = differ || ps[i].Claimed != ss[i].Claimed
	}
	if !differ {
		t.Fatal("the policy changed no read order: the test is vacuous")
	}
	// Reads in progress at the change finished as they would have.
	for _, r := range reading {
		for _, s := range ss {
			if s.ID == r.ID && (s.Claimed != r.Claimed || s.By != r.By || s.State != "final") {
				t.Fatalf("read of %s interrupted or reassigned: %+v → %+v", r.ID, r, s)
			}
		}
	}
	seen := map[int64]bool{}
	for _, e := range short.Journal() {
		if e.Kind == "revenue" {
			if seen[e.A] {
				t.Fatalf("study %d finalized twice", e.A)
			}
			seen[e.A] = true
		}
	}
}

// Actions in one batch apply in order, each against the state left by the
// previous; the same batch always gives the same results.
func TestSimultaneousActionsOrdered(t *testing.T) {
	run := func(as ...api.Action) []api.ActionResult {
		w, d0 := miniDay(t, 2)
		return applyAt(t, w, at(d0, "07:00"), as...)
	}
	a := run(extend("E06", 60), extend("E06", 200)) // 19:00, then 22:20 > cap 21:30
	b := run(extend("E06", 200), extend("E06", 60)) // 21:20, then 22:20 > cap
	if !a[0].OK || a[1].OK || a[1].Code != "exceeds_max_shift" || !b[0].OK || b[1].OK || b[1].Code != "exceeds_max_shift" ||
		!strings.Contains(a[1].Message, "at most +2h 30m") || !strings.Contains(b[1].Message, "at most +10m") {
		t.Fatalf("order a %+v\norder b %+v", a, b)
	}
	if !reflect.DeepEqual(a, run(extend("E06", 60), extend("E06", 200))) {
		t.Fatal("the same batch gave different results")
	}
	// Intermediate policies inside a batch have no effect: no dispatch
	// runs between actions.
	w1, d0 := miniDay(t, 4)
	w2, _ := miniDay(t, 4)
	applyAt(t, w1, at(d0, "10:30"), walkIn(api.WalkInScheduleLater, ""), walkIn(api.WalkInAcceptAlways, ""))
	applyAt(t, w2, at(d0, "10:30"), walkIn(api.WalkInAcceptAlways, ""))
	for _, w := range []*sim.World{w1, w2} {
		if err := runTo(w, d0.Add(simtime.Day)); err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(w1.Visits(), w2.Visits()) || w1.LedgerHash() != w2.LedgerHash() {
		t.Fatal("an intermediate policy within a batch had an effect")
	}
}

// A rejected action changes nothing: the run continues exactly as one
// without it (only the action record differs).
func TestInvalidActionAtomicity(t *testing.T) {
	plain, d0 := miniDay(t, 6)
	bad, _ := miniDay(t, 6)
	if err := runTo(plain, at(d0, "12:00")); err != nil {
		t.Fatal(err)
	}
	rs := applyAt(t, bad, at(d0, "12:00"), extend("E03", 239), extend("E99", 10), walkIn("x", ""), worklist(""),
		api.Action{Type: api.ActExtendShift, Employee: "E05", Minutes: -5})
	for _, r := range rs {
		if r.OK {
			t.Fatalf("accepted %+v", r)
		}
	}
	po, bo := plain.Observe(500), bad.Observe(500)
	// The action record is the only difference.
	bo.Actions, po.Actions = api.ActionLog{}, api.ActionLog{}
	bo.Today.Actions, po.Today.Actions = api.ActionCounts{}, api.ActionCounts{}
	if !reflect.DeepEqual(po, bo) {
		t.Fatal("a rejected action changed the observable state")
	}
	for _, w := range []*sim.World{plain, bad} {
		if err := runTo(w, d0.Add(2*simtime.Day)); err != nil {
			t.Fatal(err)
		}
	}
	if plain.LogHash() != bad.LogHash() || plain.LedgerHash() != bad.LedgerHash() || !reflect.DeepEqual(plain.Visits(), bad.Visits()) {
		t.Fatal("rejected actions changed the outcome")
	}
}

// script is a fixed set of decisions over two days.
var script = []struct {
	day   int64
	clock string
	as    []api.Action
}{
	{0, "07:00", []api.Action{extend("E06", 90), extend("E03", 300)}},
	{0, "10:17", []api.Action{walkIn(api.WalkInAcceptAlways, "")}},
	{0, "13:45", []api.Action{worklist(api.WorklistOldestFirst), walkIn(api.WalkInAcceptUntil, "17:30")}},
	{0, "16:30", []api.Action{extend("E05", 30)}}, // Casey just ended: rejected
	{1, "08:00", []api.Action{extend("E05", 45), worklist(api.WorklistShortestFirst)}},
}

// scripted applies the script, advancing with advance(target).
func scripted(t *testing.T, w *sim.World, d0 simtime.Time, advance func(simtime.Time)) {
	t.Helper()
	for _, b := range script {
		target := at(d0.Add(simtime.Duration(b.day)*simtime.Day), b.clock)
		advance(target)
		w.Apply(b.as)
	}
}

func plainAdvance(t *testing.T, w *sim.World) func(simtime.Time) {
	return func(target simtime.Time) {
		if err := runTo(w, target); err != nil {
			t.Fatal(err)
		}
	}
}

// Replay: re-applying the recorded batches at their times reproduces the
// run exactly, with the same results.
func TestActionReplayEquivalence(t *testing.T) {
	orig, d0 := miniDay(t, 8)
	scripted(t, orig, d0, plainAdvance(t, orig))
	end := d0.Add(3 * simtime.Day)
	if err := runTo(orig, end); err != nil {
		t.Fatal(err)
	}
	log := orig.ActionLog()
	rep, _ := miniDay(t, 8)
	var got []api.ActionResult
	for i := 0; i < len(log); {
		j := i
		var batch []api.Action
		for ; j < len(log) && log[j].Batch == log[i].Batch; j++ {
			batch = append(batch, log[j].Action)
		}
		got = append(got, applyAt(t, rep, isoTime(t, rep, log[i].Time), batch...)...)
		i = j
	}
	if err := runTo(rep, end); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, log) || rep.LogHash() != orig.LogHash() || rep.LedgerHash() != orig.LedgerHash() {
		t.Fatal("replaying the action record diverged")
	}
}

// Actions never reach back: everything before the batch is unchanged, in
// the log, the history and the books.
func TestActionsNotRetroactive(t *testing.T) {
	plain, d0 := miniDay(t, 9)
	act, _ := miniDay(t, 9)
	t1 := d0.Add(simtime.Day).Add(12 * simtime.Hour)
	for _, w := range []*sim.World{plain, act} {
		if err := runTo(w, t1); err != nil {
			t.Fatal(err)
		}
	}
	before := plain.Observe(2000)
	act.Apply([]api.Action{extend("E06", 120), walkIn(api.WalkInScheduleLater, ""), worklist(api.WorklistFIFO)})
	after := act.Observe(2000)
	for i, e := range before.RecentLog {
		if !reflect.DeepEqual(e, after.RecentLog[i]) {
			t.Fatalf("log entry %d changed", e.Seq)
		}
	}
	if !reflect.DeepEqual(plain.History(5), act.History(5)) {
		t.Fatal("completed days changed")
	}
	if d, _ := act.DaySummary(1); d.Ops.Workforce != (api.WorkforceMetrics{}) && d.Ops.Workforce.ApprovedOvertimeSec != 0 {
		t.Fatal("approved overtime counted before it was worked")
	}
}

// Outcomes with actions do not depend on chunking or on unrelated events.
func TestActionsChunkingAndNoise(t *testing.T) {
	whole, d0 := miniDay(t, 10)
	scripted(t, whole, d0, plainAdvance(t, whole))
	end := d0.Add(2*simtime.Day + 15*simtime.Hour)
	if err := runTo(whole, end); err != nil {
		t.Fatal(err)
	}
	r := rand.New(rand.NewPCG(3, 4))
	for _, noise := range []bool{false, true} {
		w, _ := miniDay(t, 10)
		advance := func(target simtime.Time) {
			for now := w.Engine().Now(); now < target; {
				now = min(target, now.Add(simtime.Duration(r.IntN(3600))+1))
				if noise && now < target {
					w.PushLiveNoise(now)
				}
				if err := runTo(w, now); err != nil {
					t.Fatal(err)
				}
			}
		}
		scripted(t, w, d0, advance)
		advance(end)
		if w.LogHash() != whole.LogHash() || w.LedgerHash() != whole.LedgerHash() ||
			!reflect.DeepEqual(w.ActionLog(), whole.ActionLog()) {
			t.Fatalf("noise %v: chunking or unrelated events changed an outcome with actions", noise)
		}
	}
}

// The evaluation window counts the actions and the approved overtime, and
// the extension's wages are in its operating expenses.
func TestEvaluationAttributesActions(t *testing.T) {
	plain, d0 := miniDay(t, 11)
	act, _ := miniDay(t, 11)
	applyAt(t, act, at(d0, "07:00"), extend("E06", 120), extend("E99", 5))
	for _, w := range []*sim.World{plain, act} {
		if err := runTo(w, d0.Add(simtime.Day)); err != nil {
			t.Fatal(err)
		}
	}
	ep, ea := plain.Evaluation(), act.Evaluation()
	if ea.Ops.Actions != (api.ActionCounts{Attempted: 2, Accepted: 1, Rejected: 1}) || ep.Ops.Actions.Attempted != 0 {
		t.Fatalf("actions %+v / %+v", ea.Ops.Actions, ep.Ops.Actions)
	}
	if ea.Ops.Workforce.ApprovedOvertimeSec != 7200 || ea.Money.Ledger.WagesCents <= ep.Money.Ledger.WagesCents ||
		!ea.Money.Reconciled {
		t.Fatalf("window workforce %+v, wages %d vs %d", ea.Ops.Workforce, ea.Money.Ledger.WagesCents, ep.Money.Ledger.WagesCents)
	}
}

// isoTime parses a practice-local ISO time ("2026-10-13T07:00:00").
func isoTime(t *testing.T, w *sim.World, s string) simtime.Time {
	t.Helper()
	d, err := simtime.ParseDate(s[:10])
	if err != nil {
		t.Fatal(err)
	}
	c, err := simtime.ParseClock(s[11:])
	if err != nil {
		t.Fatal(err)
	}
	return w.Calendar().StartOfDate(d).Add(c)
}

// interventions approves, at the takeover, the extensions that keep US
// (Morgan) and MR (Casey) staffed to the 20:00 close on every weekday of
// the next days, so their evening slots can be booked ahead; and stops
// walk-ins an hour before close.
func interventions(t testing.TB, w *sim.World, days int) {
	cal := w.Calendar()
	start := cal.DayIndex(w.StartTime())
	as := []api.Action{walkIn(api.WalkInAcceptUntil, "19:00")}
	for d := int64(0); d < int64(days); d++ {
		for _, e := range w.Roster(start + d) {
			if e.Name != "Morgan" && e.Name != "Casey" {
				continue
			}
			end, _ := simtime.ParseClock(e.RosterEnd)
			if gap := (20*simtime.Hour - end) / simtime.Minute; gap > 0 && gap <= 240 {
				a := extend(e.ID, int64(gap))
				a.Date = cal.Date(cal.StartOfDay(start + d)).String()
				as = append(as, a)
			}
		}
	}
	w.Engine().RunUntil(w.StartTime(), nil)
	for _, r := range w.Apply(as) {
		if !r.OK && r.Code != "after_close" && r.Code != "not_on_shift" {
			t.Fatalf("%+v", r)
		}
	}
}

// Decisions matter, demand does not move: the same seed with and without
// interventions sees identical orders and walk-ins but different
// operations and money.
func TestInterventionsChangeOutcomesNotDemand(t *testing.T) {
	cfg, err := scenario.Resolve("established")
	if err != nil {
		t.Fatal(err)
	}
	const days = 5
	plain := sim.New(cfg, rng.SeedFromInt(1))
	act := sim.New(cfg, rng.SeedFromInt(1))
	interventions(t, act, days)
	for _, w := range []*sim.World{plain, act} {
		w.Engine().RunUntil(w.StartTime().Add(days*simtime.Day), nil)
	}
	if !reflect.DeepEqual(exogenous(plain.Orders()), exogenous(act.Orders())) {
		t.Fatal("interventions changed exogenous demand")
	}
	ep, ea := plain.Evaluation(), act.Evaluation()
	if reflect.DeepEqual(ep.Ops.Rooms, ea.Ops.Rooms) || ea.Ops.Workforce.ApprovedOvertimeSec == 0 {
		t.Fatalf("interventions had no operational effect: %+v", ea.Ops.Workforce)
	}
	t.Logf("%-34s %14s %14s", "5 days from takeover", "no actions", "interventions")
	row := func(name string, a, b any) { t.Logf("%-34s %14v %14v", name, a, b) }
	us := func(e api.Evaluation, m string) string {
		for _, r := range e.Ops.Rooms {
			if r.Modality == m {
				return pctR(r.Coverage) + " / " + pctR(r.UtilOpen)
			}
		}
		return ""
	}
	row("orders received", ep.Ops.Demand.Received, ea.Ops.Demand.Received)
	row("walk-ins arrived", ep.Ops.WalkIns.Arrived, ea.Ops.WalkIns.Arrived)
	row("exams", ep.Ops.Exams, ea.Ops.Exams)
	row("orders lost", ep.Ops.Demand.Lost, ea.Ops.Demand.Lost)
	row("walk-ins imaged / left / declined", ints(ep.Ops.WalkIns.Imaged, ep.Ops.WalkIns.Left, ep.Ops.WalkIns.Declined),
		ints(ea.Ops.WalkIns.Imaged, ea.Ops.WalkIns.Left, ea.Ops.WalkIns.Declined))
	row("US staffed/open, busy/open", us(ep, "US"), us(ea, "US"))
	row("MR staffed/open, busy/open", us(ep, "MR"), us(ea, "MR"))
	row("wait p90 (min)", ep.Ops.Waits.ToExam.P90Sec/60, ea.Ops.Waits.ToExam.P90Sec/60)
	row("overtime h (approved)", ints(int(ep.Ops.Workforce.OvertimeSec/3600), int(ep.Ops.Workforce.ApprovedOvertimeSec/3600)),
		ints(int(ea.Ops.Workforce.OvertimeSec/3600), int(ea.Ops.Workforce.ApprovedOvertimeSec/3600)))
	row("net revenue $", ep.Money.NetRevenueCents/100, ea.Money.NetRevenueCents/100)
	row("operating expense $", ep.Money.OperatingExpenseCents/100, ea.Money.OperatingExpenseCents/100)
	row("operating profit $", ep.Money.OperatingProfitCents/100, ea.Money.OperatingProfitCents/100)
	row("alerts raised", ep.Ops.Alerts.Warn+ep.Ops.Alerts.Alert, ea.Ops.Alerts.Warn+ea.Ops.Alerts.Alert)
}

func ints(xs ...int) string {
	var s []string
	for _, x := range xs {
		s = append(s, strconv.Itoa(x))
	}
	return strings.Join(s, "/")
}

func pctR(r api.Ratio) string {
	if !r.Defined() {
		return "—"
	}
	return strconv.FormatInt((r.BP()+50)/100, 10) + "%"
}

// BenchmarkApplyValidate is one action's validation and application cost
// (a policy switch back and forth; each switch runs a dispatch).
func BenchmarkApplyValidate(b *testing.B) {
	cfg := miniConfig(b)
	w := sim.New(cfg, rng.SeedFromInt(1))
	w.Engine().RunUntil(w.StartTime().Add(5*simtime.Hour), nil)
	ps := []string{api.WalkInAcceptAlways, api.WalkInAcceptIfGap}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Apply([]api.Action{walkIn(ps[i%2], "")})
	}
}

// BenchmarkValidateOnly is a preview (no change, no dispatch).
func BenchmarkValidateOnly(b *testing.B) {
	cfg := miniConfig(b)
	w := sim.New(cfg, rng.SeedFromInt(1))
	w.Engine().RunUntil(w.StartTime().Add(time5h), nil)
	for i := 0; i < b.N; i++ {
		_ = w.Validate(extend("E06", 60))
	}
}

const time5h = 5 * simtime.Hour

// BenchmarkThirtyDaysWithActions runs 30 days of the established practice
// after takeover, with and without the daily interventions.
func BenchmarkThirtyDaysWithActions(b *testing.B) {
	cfg, err := scenario.Resolve("established")
	if err != nil {
		b.Fatal(err)
	}
	for _, withActions := range []bool{false, true} {
		name := "none"
		if withActions {
			name = "daily"
		}
		b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				w := sim.New(cfg, rng.SeedFromInt(1))
				w.Engine().RunUntil(w.StartTime(), nil)
				b.StartTimer()
				if withActions {
					interventions(b, w, 25)
				}
				w.Engine().RunUntil(w.StartTime().Add(30*simtime.Day), nil)
			}
		})
	}
}

// An extension approved for a later day opens that day's slots for
// booking at once, and on the day the employee works to the new end on
// regular duty: approved, not required, overtime.
func TestExtensionAheadOpensBookings(t *testing.T) {
	plain, d0 := miniDay(t, 12)
	ahead, _ := miniDay(t, 12)
	cal := ahead.Calendar()
	thu := cal.Date(d0.Add(9 * simtime.Day)).String() // Thursday next week
	cases := []struct {
		a    api.Action
		code string
	}{
		{api.Action{Type: api.ActExtendShift, Employee: "E06", Minutes: 60, Date: "2026-10-12"}, "past_date"},
		{api.Action{Type: api.ActExtendShift, Employee: "E06", Minutes: 60, Date: "2027-10-12"}, "beyond_horizon"},
		{api.Action{Type: api.ActExtendShift, Employee: "E06", Minutes: 60, Date: "2026-13-01"}, "invalid_date"},
		{api.Action{Type: api.ActExtendShift, Employee: "E06", Minutes: 60, Date: "2026-10-18"}, "not_on_shift"}, // Sunday
		{api.Action{Type: api.ActExtendShift, Employee: "E06", Minutes: 240, Date: thu}, "exceeds_max_shift"},
		{api.Action{Type: api.ActExtendShift, Employee: "E06", Minutes: 120, Date: thu}, api.ActionAccepted},
	}
	var as []api.Action
	for _, c := range cases {
		as = append(as, c.a)
	}
	rs := applyAt(t, ahead, at(d0, "07:00"), as...)
	for i, c := range cases {
		if rs[i].Code != c.code {
			t.Errorf("%+v: %+v", c.a, rs[i])
		}
	}
	thuIdx := cal.DayIndex(d0) + 9
	openAfter18 := func(w *sim.World) (n int) {
		for _, rs := range w.Schedule(thuIdx).Rooms {
			for _, s := range rs.Slots {
				if rs.Modality == "US" && s.From >= 18*simtime.Hour && s.Status != api.SlotUnstaffed {
					n++
				}
			}
		}
		return n
	}
	if err := runTo(plain, at(d0, "07:00")); err != nil {
		t.Fatal(err)
	}
	if openAfter18(plain) != 0 || openAfter18(ahead) != 3 {
		t.Fatalf("Thursday US slots after 18:00 bookable: %d without, %d with", openAfter18(plain), openAfter18(ahead))
	}
	o := ahead.Observe(0)
	if len(o.Extensions) != 1 || o.Extensions[0].Until != "20:00" || o.Extensions[0].Date != thu {
		t.Fatalf("extensions %+v", o.Extensions)
	}
	for _, w := range []*sim.World{plain, ahead} {
		if err := runTo(w, d0.Add(10*simtime.Day)); err != nil {
			t.Fatal(err)
		}
	}
	thuOps := day(t, ahead, thuIdx).Ops
	var booked int
	for _, a := range ahead.Appointments() {
		if a.Day == thuIdx && !a.Cancelled && a.Span.From >= 18*simtime.Hour && ahead.Config().Rooms[a.Room].Modality == "US" {
			booked++
		}
	}
	if booked == 0 || person(thuOps, "Morgan").ApprovedOvertimeSec != 7200 || thuOps.Workforce.ApprovedOvertimeSec != 7200 {
		t.Fatalf("Thursday: %d evening US bookings, Morgan %+v", booked, person(thuOps, "Morgan"))
	}
	if !reflect.DeepEqual(exogenous(plain.Orders()), exogenous(ahead.Orders())) {
		t.Fatal("demand changed")
	}
}
