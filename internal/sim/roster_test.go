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

// V0.2 Slice 3: set_roster (V0.2_DESIGN §5.8, §6.3–§6.4). The established
// practice is taken over on Tue 2026-10-13; with 14 days' notice the
// earliest effective Monday is Nov 2. Riley (E03, XR+CT, 40 h inferred
// contract) works Mon–Fri 07:45–16:15 in R1 (XR) covering R3 (CT) until
// 11:30; Sam (E04, XR+CT, 39.5 h) Tue–Sat 11:30–20:00 in R3 covering R1
// from 16:15; Jordan (E01) is the front desk.

func setRoster(id, effective string, r api.Roster) api.Action {
	return api.Action{Type: api.ActSetRoster, Op: api.RosterOpSet, Employee: id, Effective: effective, Roster: &r}
}

func removeRoster(id, effective string) api.Action {
	return api.Action{Type: api.ActSetRoster, Op: api.RosterOpRemove, Employee: id, Effective: effective}
}

// rosterNow is employee id's roster in effect today.
func rosterNow(t *testing.T, w *sim.World, id string) api.Roster {
	t.Helper()
	p, ok := w.RosterPlan(id)
	if !ok {
		t.Fatalf("no employee %s", id)
	}
	for _, v := range p.Versions {
		if v.InEffect {
			return v.Roster
		}
	}
	return p.Versions[0].Roster
}

// both is the same week in both rota weeks.
func both(wk api.Week) api.Roster { return api.Roster{A: wk, B: wk} }

// shiftOf is employee id's planned shift on date ("" when off).
func shiftOf(w *sim.World, id, date string) string {
	for _, e := range w.Roster(dayOf(w, date)) {
		if e.ID == id {
			return e.Shift
		}
	}
	return ""
}

func codes(res []api.ActionResult) []string {
	var out []string
	for _, r := range res {
		out = append(out, r.Code)
	}
	return out
}

// digestsOf are the state, ledger and log digests.
func digestsOf(w *sim.World) [3]string {
	return [3]string{w.StateDigest(), w.LedgerHash(), w.LogHash()}
}

// Effective dates: a Monday at least 14 days ahead, within the horizon;
// never today or the past. Every refusal leaves the run exactly as it was.
func TestSetRosterEffectiveDates(t *testing.T) {
	w := atTakeover(t, `{}`, 1)
	r := rosterNow(t, w, "E03")
	r.A.Fri, r.B.Fri = nil, nil
	before := digestsOf(w)
	for _, c := range []struct{ date, code string }{
		{"2026-10-13", "past_date"},
		{"2026-10-12", "past_date"},
		{"2026-11-03", "not_monday"},
		{"2026-10-26", "notice_too_short"}, // 13 days
		{"2026-11-1", "invalid_date"},
		{"2027-03-01", "beyond_horizon"}, // 139 days
	} {
		res := w.Apply([]api.Action{setRoster("E03", c.date, r)})
		if res[0].OK || res[0].Code != c.code {
			t.Errorf("%s: %+v", c.date, res[0])
		}
	}
	if digestsOf(w) != before {
		t.Fatal("a refused set_roster changed the run")
	}
	res := mustApply(t, w, setRoster("E03", "2026-11-02", r))
	if res[0].Code != api.ActionAccepted || res[0].EffectiveAt != "2026-11-02" {
		t.Fatalf("%+v", res[0])
	}
	// From a Monday, the Monday exactly 14 days later is lawful.
	runChecked(t, w, w.Calendar().StartOfDay(dayOf(w, "2026-10-19")).Add(9*simtime.Hour))
	if res := w.Apply([]api.Action{setRoster("E03", "2026-11-02", rosterNow(t, w, "E03"))}); res[0].Code != api.ActionAccepted {
		t.Fatalf("14 days from a Monday: %+v", res[0])
	}
}

// Successive versions resolve by date; a pending version can be replaced
// or removed (its predecessor resumes); identical requests are no-ops;
// versions in effect or past never change, and the log keeps every
// operation.
func TestRosterVersions(t *testing.T) {
	w := atTakeover(t, `{}`, 1)
	w.Engine().CheckInvariants = true
	orig := rosterNow(t, w, "E03")
	noFri, noMon := orig, orig
	noFri.A.Fri, noFri.B.Fri = nil, nil
	noMon.A.Mon, noMon.B.Mon = nil, nil
	// Two successive changes in one batch (out of date order).
	mustApply(t, w, setRoster("E03", "2026-11-16", noMon), setRoster("E03", "2026-11-02", noFri))
	for _, c := range []struct{ date, want string }{
		{"2026-10-30", "07:45–16:15"}, // Fri before the first version
		{"2026-11-06", ""},            // Fri under the first
		{"2026-11-09", "07:45–16:15"}, // Mon under the first
		{"2026-11-13", ""},            // Fri still under the first
		{"2026-11-20", "07:45–16:15"}, // Fri under the second
		{"2026-11-16", ""},            // Mon under the second
	} {
		if got := shiftOf(w, "E03", c.date); got != c.want {
			t.Errorf("%s: %q, want %q", c.date, got, c.want)
		}
	}
	if e := employee(w, "E03"); len(e.RosterChanges) != 2 || e.RosterChanges[0].Effective != "2026-11-02" || e.RosterChanges[0].NetMinA != 1920 {
		t.Fatalf("pending: %+v", e.RosterChanges)
	}
	// The same version again, or the original where it is in force: no-ops.
	before := digestsOf(w)
	if res := w.Apply([]api.Action{setRoster("E03", "2026-11-02", noFri)}); res[0].Code != api.ActionUnchanged {
		t.Fatalf("repeat: %+v", res[0])
	}
	if res := w.Apply([]api.Action{setRoster("E03", "2026-11-09", noFri)}); res[0].Code != api.ActionUnchanged {
		t.Fatalf("same as the version in force: %+v", res[0])
	}
	if res := w.Apply([]api.Action{removeRoster("E03", "2026-11-09")}); res[0].Code != api.ActionUnchanged {
		t.Fatalf("remove absent: %+v", res[0])
	}
	if digestsOf(w) != before {
		t.Fatal("a no-op changed the run")
	}
	// Two writes to one date in a batch are refused, whatever the order.
	if res := w.Apply([]api.Action{setRoster("E03", "2026-11-23", noMon), removeRoster("E03", "2026-11-23")}); res[1].Code != "invalid_parameters" ||
		res[0].Code != "batch_rejected" {
		t.Fatalf("duplicate date: %v", codes(res))
	}
	// Replace the first; remove the second: the first resumes.
	mustApply(t, w, setRoster("E03", "2026-11-02", noMon), removeRoster("E03", "2026-11-16"))
	if got := shiftOf(w, "E03", "2026-11-06"); got != "07:45–16:15" {
		t.Fatalf("replaced version: Fri %q", got)
	}
	if got := shiftOf(w, "E03", "2026-11-23"); got != "" {
		t.Fatalf("after the removal: Mon %q", got)
	}
	p, _ := w.RosterPlan("E03")
	if len(p.Versions) != 2 || p.Versions[1].Effective != "2026-11-02" || !p.Versions[0].InEffect || !p.Versions[1].Pending {
		t.Fatalf("plan: %+v", p.Versions)
	}
	if n := w.RosterLogLen("E03"); n != 4 {
		t.Fatalf("roster log has %d operations, want 4", n)
	}
	// Once in effect a version is history: no change dated on or before
	// today, and the plan marks it in effect.
	runChecked(t, w, w.Calendar().StartOfDay(dayOf(w, "2026-11-03")).Add(7*simtime.Hour))
	if res := w.Apply([]api.Action{removeRoster("E03", "2026-11-02")}); res[0].Code != "past_date" {
		t.Fatalf("remove in effect: %+v", res[0])
	}
	p, _ = w.RosterPlan("E03")
	if !p.Versions[1].InEffect || p.Versions[1].Pending || p.EarliestEffective != "2026-11-23" {
		t.Fatalf("plan: %+v", p)
	}
}

// Only hourly technologists and front desk staff; skills, rooms, the
// contract, breaks, six days, and day-boundary rest are checked as for a
// hire.
func TestSetRosterValidation(t *testing.T) {
	w := atTakeover(t, `{}`, 1)
	r := rosterNow(t, w, "E03")
	desk := rosterNow(t, w, "E01")
	long := r
	long.A.Sat = shiftAt("R1", "08:00", "12:00") // 44 h in week A > 40 h contract
	mr := r
	mr.A.Mon = shiftAt("R4", "07:45", "16:15", "11:30", "12:00")
	cover := r
	c := *cover.A.Tue
	c.Covers = []api.Cover{{Room: "R4", From: "08:00", Until: "09:00"}}
	cover.A.Tue = &c
	deskRoom := desk
	deskRoom.A.Mon = shiftAt("R1", "07:45", "16:15", "12:00", "12:30")
	noBreak := r
	noBreak.B.Wed = shiftAt("R1", "07:45", "16:15")
	seven := both(weekOf(shiftAt("R1", "08:00", "12:00"), "Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"))
	// Rest inside the pattern: week A Sunday's cap (11:30 + 12 h) to week
	// B Monday 07:45 is 8 h 15 min; the cycle wraps B → A too.
	restAB := both(weekOf(shiftAt("R1", "07:45", "11:45"), "Mon", "Tue", "Wed", "Thu"))
	restAB.A.Sun = shiftAt("R1", "11:30", "15:30")
	restBA := both(weekOf(shiftAt("R1", "07:45", "11:45"), "Mon", "Tue", "Wed", "Thu"))
	restBA.B.Sun = shiftAt("R1", "11:30", "15:30")
	for _, c := range []struct {
		name string
		act  api.Action
		code string
		text string
	}{
		{"radiologist", setRoster("E07", "2026-11-02", r), "wrong_role", "radiologist"},
		{"ops manager", setRoster("E02", "2026-11-02", desk), "wrong_role", "ops manager"},
		{"unknown", setRoster("E99", "2026-11-02", r), "unknown_employee", ""},
		{"over contract", setRoster("E03", "2026-11-02", long), "invalid_roster", "more than the 40 h contract"},
		{"unqualified room", setRoster("E03", "2026-11-02", mr), "invalid_roster", "room R4 is MR"},
		{"unqualified cover", setRoster("E03", "2026-11-02", cover), "invalid_roster", "R4 is MR"},
		{"desk in a room", setRoster("E01", "2026-11-02", deskRoom), "invalid_roster", "front desk staff are assigned"},
		{"no break", setRoster("E03", "2026-11-02", noBreak), "invalid_roster", "roster.B.Wed"},
		{"seven days", setRoster("E03", "2026-11-02", seven), "invalid_roster", "7 working days"},
		{"rest A→B", setRoster("E03", "2026-11-02", restAB), "invalid_roster", "week A Sun"},
		{"rest B→A", setRoster("E03", "2026-11-02", restBA), "invalid_roster", "week B Sun"},
		{"no op", api.Action{Type: api.ActSetRoster, Employee: "E03", Effective: "2026-11-02", Roster: &r}, "invalid_schema", ""},
		{"remove with roster", api.Action{Type: api.ActSetRoster, Op: "remove", Employee: "E03", Effective: "2026-11-02", Roster: &r}, "invalid_schema", ""},
		{"set without roster", api.Action{Type: api.ActSetRoster, Op: "set", Employee: "E03", Effective: "2026-11-02"}, "invalid_schema", ""},
		{"foreign field", api.Action{Type: api.ActSetRoster, Op: "set", Employee: "E03", Effective: "2026-11-02", Roster: &r, Minutes: 5}, "invalid_schema", ""},
		{"mixed", api.Action{Type: api.ActExtendShift, Employee: "E03", Minutes: 30}, "", ""},
	} {
		acts := []api.Action{c.act}
		if c.name == "mixed" {
			acts = append(acts, setRoster("E03", "2026-11-02", mr))
			res := w.Apply(acts)
			if res[0].Code != "mixed_batch" || !strings.Contains(res[0].Message, "hire, terminate, set_roster") {
				t.Errorf("mixed: %+v", res[0])
			}
			continue
		}
		res := w.Apply(acts)
		if res[0].OK || res[0].Code != c.code || !strings.Contains(res[0].Message, c.text) {
			t.Errorf("%s: %+v", c.name, res[0])
		}
	}
	// Every initial technologist and front desk employee's own roster is
	// valid under these rules and is a no-op.
	for _, id := range []string{"E01", "E03", "E04", "E05", "E06"} {
		if res := w.Apply([]api.Action{setRoster(id, "2026-11-02", rosterNow(t, w, id))}); res[0].Code != api.ActionUnchanged {
			t.Errorf("%s's own roster: %+v", id, res[0])
		}
	}
	// Morgan's scenario roster has 40 h in week A against her inferred
	// 39.5 h contract: once changed, it can still be restored exactly, but
	// no other roster may exceed the contract.
	morgan := rosterNow(t, w, "E06")
	cut := morgan
	cut.A.Fri = nil
	mustApply(t, w, setRoster("E06", "2026-11-02", cut))
	over := morgan
	late := *over.A.Mon
	late.End = "18:15"
	over.A.Mon = &late
	if res := w.Apply([]api.Action{setRoster("E06", "2026-11-16", over)}); res[0].Code != "invalid_roster" {
		t.Fatalf("Morgan above her contract: %+v", res[0])
	}
	if res := w.Apply([]api.Action{setRoster("E06", "2026-11-16", morgan)}); res[0].Code != api.ActionAccepted {
		t.Fatalf("Morgan's original restored: %+v", res[0])
	}
	// An exactly-10-hour rest across the boundary is lawful (the shift cap
	// is start + 12 h).
	ok := both(weekOf(shiftAt("R1", "07:00", "11:00"), "Mon", "Tue", "Wed"))
	ok.A.Sun = shiftAt("R1", "09:00", "13:00") // cap 21:00, A Mon is not next
	ok.B.Sun = shiftAt("R1", "09:00", "13:00") // cap 21:00 → A Mon 07:00: exactly 10 h
	if res := w.Apply([]api.Action{setRoster("E03", "2026-11-02", ok)}); res[0].Code != api.ActionAccepted {
		t.Fatalf("10 h rest: %+v", res[0])
	}
}

// payroll is employee index emp's wages and guarantee (payroll, benefits)
// posted on days [from, to).
func payroll(w *sim.World, emp int64, from, to int64) (wages, benefits int64) {
	for _, kind := range []string{"wages", "guarantee"} {
		for _, e := range entries(w, kind, emp) {
			if d := w.Calendar().DayIndex(e.At); d >= from && d < to {
				wages += lineAmt(e, "payroll")
				benefits += lineAmt(e, "benefits")
			}
		}
	}
	return wages, benefits
}

// A roster cut, even to nothing, never cuts pay: a full week under a
// four-day or a zero-hour roster pays Riley's 40-hour contract exactly
// (r × C = $1,440 with benefits on it); notice and severance are as they
// were.
func TestRosterCutKeepsGuarantee(t *testing.T) {
	for _, c := range []struct {
		name string
		cut  func(r *api.Roster)
	}{
		{"four days", func(r *api.Roster) { r.A.Fri, r.B.Fri = nil, nil }},
		{"zero hours", func(r *api.Roster) { *r = api.Roster{} }},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := atTakeover(t, `{}`, 4)
			sev := w.PreviewStaffing([]api.Action{terminateOf("E03", "2026-11-30")}).Terminations[0].SeveranceCents
			r := rosterNow(t, w, "E03")
			c.cut(&r)
			res := mustApply(t, w, setRoster("E03", "2026-11-02", r))
			if !strings.Contains(res[0].Message, "contracted time not rostered is still paid") {
				t.Fatalf("message: %s", res[0].Message)
			}
			mon := dayOf(w, "2026-11-02")
			runChecked(t, w, w.Calendar().StartOfDay(mon+8))
			wages, benefits := payroll(w, 2, mon, mon+7)
			bp := w.Config().Finance.BenefitsBP
			if wages != 144_000 || benefits != (144_000*bp+5_000)/10_000 {
				t.Fatalf("week under the cut roster: wages %d, benefits %d; want 144000 and %d", wages, benefits, (144_000*bp+5_000)/10_000)
			}
			if p := w.PreviewStaffing([]api.Action{terminateOf("E03", "2026-11-30")}); p.Terminations[0].SeveranceCents != sev {
				t.Fatalf("severance %d after the cut, %d before", p.Terminations[0].SeveranceCents, sev)
			}
		})
	}
}

// The front desk: a zero-hour roster for Jordan leaves registration to
// the technologists (and the operations manager), displaces no booking,
// and pays the contract.
func TestRosterFrontDesk(t *testing.T) {
	w := atTakeover(t, `{}`, 1)
	w.Engine().CheckInvariants = true
	p := w.PreviewStaffing([]api.Action{setRoster("E01", "2026-11-02", api.Roster{})})
	if !p.OK || len(p.Displacements) != 0 || len(p.Rosters) != 1 || p.Rosters[0].AfterMinA != 0 || p.Rosters[0].BeforeMinA != 2400 {
		t.Fatalf("preview: %+v", p)
	}
	mustApply(t, w, setRoster("E01", "2026-11-02", api.Roster{}))
	if n := len(w.Displacements()); n != 0 {
		t.Fatalf("%d displaced by a desk change", n)
	}
	if got := shiftOf(w, "E01", "2026-11-02"); got != "" {
		t.Fatalf("Jordan still on %q", got)
	}
	if got := shiftOf(w, "E01", "2026-10-30"); got == "" {
		t.Fatal("Jordan's Friday before the change was lost")
	}
	mon := dayOf(w, "2026-11-02")
	runChecked(t, w, w.Calendar().StartOfDay(mon+8))
	if wages, _ := payroll(w, 0, mon, mon+7); wages != 2100*40 {
		t.Fatalf("Jordan's week: %d, want %d", wages, 2100*40)
	}
}

// Approved extensions: one whose shift the new roster keeps stands (and
// is shown in the preview); one the new roster would strand rejects the
// whole batch — no shift that day, a shift already to the close, or one
// whose cap the extension would pass — and nothing changes.
func TestRosterExtensionConflicts(t *testing.T) {
	w := atTakeover(t, `{}`, 1)
	mustApply(t, w, api.Action{Type: api.ActExtendShift, Employee: "E03", Minutes: 30, Date: "2026-11-04"})
	orig := rosterNow(t, w, "E03")
	noWed, late, capped, keep := orig, orig, orig, orig
	noWed.A.Wed, noWed.B.Wed = nil, nil
	lateWed := shiftAt("R1", "11:30", "20:00", "15:30", "16:00")
	late.A.Wed, late.B.Wed = lateWed, lateWed
	late.A.Thu, late.B.Thu = nil, nil // (rest after a 20:00 finish)
	capWed := &api.Shift{Start: "07:00", End: "18:45", Breaks: [][2]string{{"11:00", "11:30"}, {"15:00", "15:30"}}, Assignment: "R1", Covers: []api.Cover{}}
	capped.A.Wed, capped.B.Wed = capWed, capWed
	capped.A.Fri, capped.B.Fri = nil, nil
	keep.A.Fri, keep.B.Fri = nil, nil
	before := digestsOf(w)
	for _, c := range []struct {
		name string
		r    api.Roster
		text string
	}{
		{"no shift", noWed, "no shift that day"},
		{"to the close", late, "already runs to the close"},
		{"past the cap", capped, "past its cap 19:00"},
	} {
		res := w.Apply([]api.Action{removeRoster("E03", "2026-11-09"), setRoster("E03", "2026-11-02", c.r)})
		if res[1].Code != "extension_conflict" || !strings.Contains(res[1].Message, c.text) || !strings.Contains(res[1].Message, "Wed Nov 4") ||
			res[0].Code != "batch_rejected" {
			t.Errorf("%s: %v %s", c.name, codes(res), res[1].Message)
		}
	}
	if digestsOf(w) != before {
		t.Fatal("a rejected batch changed the run")
	}
	p := w.PreviewStaffing([]api.Action{setRoster("E03", "2026-11-02", keep)})
	if !p.OK || len(p.Rosters[0].Extensions) != 1 || p.Rosters[0].Extensions[0].Until != "16:45" {
		t.Fatalf("preview: %+v", p.Rosters)
	}
	mustApply(t, w, setRoster("E03", "2026-11-02", keep))
	if got := shiftOf(w, "E03", "2026-11-04"); got != "07:45–16:45" {
		t.Fatalf("kept extension: %q", got)
	}
}

// Termination and roster changes: a version after the last day is
// refused, in either order within a batch; a later termination voids the
// pending versions it overtakes (logged, never resurrected); a change
// during notice, before the last day, is lawful.
func TestRosterAndTermination(t *testing.T) {
	w := atTakeover(t, `{}`, 1)
	w.Engine().CheckInvariants = true
	r := rosterNow(t, w, "E03")
	r.A.Fri, r.B.Fri = nil, nil
	for _, batch := range [][]api.Action{
		{setRoster("E03", "2026-11-16", r), terminateOf("E03", "2026-11-13")},
		{terminateOf("E03", "2026-11-13"), setRoster("E03", "2026-11-16", r)},
	} {
		res := w.Apply(batch)
		k := slices.IndexFunc(batch, func(a api.Action) bool { return a.Type == api.ActSetRoster })
		if res[k].Code != "not_employed" || res[1-k].Code != "batch_rejected" {
			t.Errorf("%v", codes(res))
		}
	}
	mustApply(t, w, setRoster("E03", "2026-11-16", r), setRoster("E03", "2026-11-02", r))
	p := w.PreviewStaffing([]api.Action{terminateOf("E03", "2026-11-13")})
	if !p.OK || !slices.Equal(p.Terminations[0].VoidedRosters, []string{"2026-11-16"}) {
		t.Fatalf("preview: %+v", p.Terminations)
	}
	mustApply(t, w, terminateOf("E03", "2026-11-13"))
	if e := employee(w, "E03"); len(e.RosterChanges) != 1 || e.RosterChanges[0].Effective != "2026-11-02" {
		t.Fatalf("pending after the termination: %+v", e.RosterChanges)
	}
	var voided bool
	for _, l := range w.Observe(1000).RecentLog {
		voided = voided || strings.Contains(l.Text, "Roster change voided: Riley (E03) from Mon Nov 16")
	}
	if !voided {
		t.Fatal("no log line for the voided version")
	}
	if res := w.Apply([]api.Action{setRoster("E03", "2026-11-16", r)}); res[0].Code != "not_employed" {
		t.Fatalf("after the end: %+v", res[0])
	}
	// During notice, before the last day: lawful.
	r2 := rosterNow(t, w, "E03")
	r2.A.Mon, r2.B.Mon = nil, nil
	mustApply(t, w, setRoster("E03", "2026-11-09", r2))
	runChecked(t, w, w.Calendar().StartOfDay(dayOf(w, "2026-11-16")).Add(time7))
}

const time7 = 7 * simtime.Hour

// samShift is Sam's shift pattern: R3 (CT) covering R1 (XR) late.
func samShift(day string) *api.Shift {
	if day == "Sat" {
		s := shiftAt("R3", "10:00", "18:00", "13:30", "14:00")
		s.Covers = []api.Cover{{Room: "R1", From: "10:00", Until: "18:00"}}
		return s
	}
	s := shiftAt("R3", "11:30", "20:00", "15:30", "16:00")
	s.Covers = []api.Cover{{Room: "R1", From: "16:15", Until: "20:00"}}
	return s
}

func samWeek(days ...string) api.Week {
	var w api.Week
	p := map[string]**api.Shift{"Mon": &w.Mon, "Tue": &w.Tue, "Wed": &w.Wed, "Thu": &w.Thu, "Fri": &w.Fri, "Sat": &w.Sat, "Sun": &w.Sun}
	for _, d := range days {
		*p[d] = samShift(d)
	}
	return w
}

// Shared XR/CT coverage: Riley stops covering R3 (CT) on Mondays. Each
// CT booking on a Monday morning loses its technologist and is displaced
// (cause roster_changed) — exactly the set the preview named — then
// rebooked within the patient's original tolerance or lost; XR bookings
// in R1 stay. Deterministic.
func TestRosterDisplacement(t *testing.T) {
	run := func() (*sim.World, api.StaffingPreview) {
		w := atTakeover(t, `{}`, 1)
		w.Engine().CheckInvariants = true
		r := rosterNow(t, w, "E03")
		mon := *r.A.Mon
		mon.Covers = []api.Cover{}
		r.A.Mon, r.B.Mon = &mon, &mon
		act := setRoster("E03", "2026-11-02", r)
		p := w.PreviewStaffing([]api.Action{act})
		mustApply(t, w, act)
		return w, p
	}
	w, p := run()
	if !p.OK || len(p.Displacements) == 0 || len(p.SlotsClosed) == 0 || p.SlotsClosed[0].Modality != "CT" {
		t.Fatalf("preview: %d displaced, closed %+v", len(p.Displacements), p.SlotsClosed)
	}
	var previewed, displaced []string
	for _, d := range p.Displacements {
		previewed = append(previewed, d.Appointment)
		if d.Room != "R3" || d.Reason != "unstaffed" {
			t.Fatalf("previewed %+v", d)
		}
	}
	appts := map[string]sim.ApptInfo{}
	for _, a := range w.Appointments() {
		appts[a.ID] = a
	}
	for _, d := range w.Displacements() {
		displaced = append(displaced, d.Appt)
		old := appts[d.Appt]
		if d.Cause != "roster_changed" || old.State != "displaced" {
			t.Fatalf("record %+v", d)
		}
		if d.Resolution == "rebooked" {
			n := appts[d.Replacement]
			anchor, tol := w.OrderAnchor(old.Order)
			if n.Attempt != old.Attempt || n.Day-anchor > tol || w.Calendar().At(n.Day, n.Span.From) < w.Calendar().At(old.Day, old.Span.From) {
				t.Fatalf("replacement %+v for %+v", n, old)
			}
		}
	}
	slices.Sort(previewed)
	slices.Sort(displaced)
	if !slices.Equal(previewed, displaced) {
		t.Fatalf("preview named %v, commit displaced %v", previewed, displaced)
	}
	w2, _ := run()
	if digestsOf(w2) != digestsOf(w) {
		t.Fatal("not deterministic")
	}
	runChecked(t, w, w.Calendar().StartOfDay(dayOf(w, "2026-11-10")))
}

// A coordinated exchange of coverage: X (an XR+CT hire on Sam's pattern
// but Monday instead of Tuesday) and Sam swap Monday and Tuesday. Either
// change alone displaces the other's bookings; the batch, validated and
// reconciled against its final timeline once, displaces nobody, in
// either order.
func TestRosterExchangeInOneBatch(t *testing.T) {
	w := atTakeover(t, `{}`, 1)
	w.Engine().CheckInvariants = false
	res := mustApply(t, w, hireOf("x", "technologist", []string{"XR", "CT"}, 4200, 2370, samWeek("Mon", "Wed", "Thu", "Fri", "Sat"), "2026-11-03"))
	x := res[0].EmployeeID
	runChecked(t, w, w.Calendar().StartOfDay(dayOf(w, "2026-11-02")).Add(9*simtime.Hour))
	if e := employee(w, x); e.Hired.ProductiveDate != "2026-11-12" {
		t.Fatalf("X independent from %s", e.Hired.ProductiveDate)
	}
	samNew := setRoster("E04", "2026-11-16", both(samWeek("Mon", "Wed", "Thu", "Fri", "Sat")))
	xNew := setRoster(x, "2026-11-16", both(samWeek("Tue", "Wed", "Thu", "Fri", "Sat")))
	samAlone := w.PreviewStaffing([]api.Action{samNew})
	xAlone := w.PreviewStaffing([]api.Action{xNew})
	if len(samAlone.Displacements) == 0 {
		t.Fatal("Sam's change alone displaces nothing: no Tuesday bookings to test with")
	}
	for _, batch := range [][]api.Action{{samNew, xNew}, {xNew, samNew}} {
		if p := w.PreviewStaffing(batch); !p.OK || len(p.Displacements) != 0 || len(p.SlotsClosed) != 0 || len(p.SlotsOpened) != 0 {
			t.Fatalf("exchange preview: %d displaced, opened %v, closed %v", len(p.Displacements), p.SlotsOpened, p.SlotsClosed)
		}
	}
	t.Logf("alone: Sam %d, X %d displaced; together 0", len(samAlone.Displacements), len(xAlone.Displacements))
	mustApply(t, w, xNew, samNew)
	if n := len(w.Displacements()); n != 0 {
		t.Fatalf("%d displaced by the exchange", n)
	}
	if shiftOf(w, "E04", "2026-11-16") == "" || shiftOf(w, "E04", "2026-11-17") != "" || shiftOf(w, x, "2026-11-17") == "" || shiftOf(w, x, "2026-11-16") != "" {
		t.Fatal("the exchange did not take effect")
	}
	runChecked(t, w, w.Calendar().StartOfDay(dayOf(w, "2026-11-18")))
}

// Training follows the composed timeline. A CT hire (Mon 12:00–20:00,
// Thu 08:00–12:00: 11.5 h a week) starting Tue Nov 3 completes 40 h on
// Mon Nov 30 (independent Dec 1). Before the first working day, dropping
// Thursdays from Nov 9 delays completion to Mon Dec 7: bookings already
// made on that Monday lose their technologist (training_delayed)
// and capacity is never granted early. During training, a fuller roster
// from Nov 23 brings completion forward and the capacity back. A roster
// that would not complete training within 60 days is refused.
func TestRosterTrainee(t *testing.T) {
	w := atTakeover(t, `{}`, 1)
	w.Engine().CheckInvariants = false
	mon := shiftAt("R3", "12:00", "20:00", "15:30", "16:00")
	thu := shiftAt("R3", "08:00", "12:00")
	wk := api.Week{Mon: mon, Thu: thu}
	res := mustApply(t, w, hireOf("ct", "technologist", []string{"CT"}, 4400, 1200, wk, "2026-11-03"))
	h := res[0].EmployeeID
	if e := employee(w, h); e.Hired.ProductiveDate != "2026-12-01" {
		t.Fatalf("productive %s", e.Hired.ProductiveDate)
	}
	runChecked(t, w, w.Calendar().StartOfDay(dayOf(w, "2026-10-26")).Add(9*simtime.Hour))
	// Too thin: 4 h a week from Nov 9 cannot reach 40 h by Jan 2.
	thin := api.Roster{A: api.Week{Thu: thu}, B: api.Week{Thu: thu}}
	before := digestsOf(w)
	if res := w.Apply([]api.Action{setRoster(h, "2026-11-09", thin)}); res[0].Code != "training_incomplete" {
		t.Fatalf("thin: %+v", res[0])
	}
	if digestsOf(w) != before {
		t.Fatal("a refused batch changed the run")
	}
	monOnly := both(api.Week{Mon: mon})
	p := w.PreviewStaffing([]api.Action{setRoster(h, "2026-11-09", monOnly)})
	if !p.OK || p.Rosters[0].ProductiveBefore != "2026-12-01" || p.Rosters[0].ProductiveAfter != "2026-12-08" {
		t.Fatalf("preview: %+v", p.Rosters)
	}
	mustApply(t, w, setRoster(h, "2026-11-09", monOnly))
	if e := employee(w, h); e.Hired.ProductiveDate != "2026-12-08" {
		t.Fatalf("delayed: productive %s", e.Hired.ProductiveDate)
	}
	recs := w.Displacements()
	appts := map[string]sim.ApptInfo{}
	for _, a := range w.Appointments() {
		appts[a.ID] = a
	}
	// (On Monday Riley covers R3 from R1: without the hire, XR bookings can
	// lose their shared technologist too.)
	for _, d := range recs {
		a := appts[d.Appt]
		if !strings.Contains(d.Cause, "training_delayed") || a.Day != dayOf(w, "2026-12-07") ||
			(d.Reason != "unstaffed" && d.Reason != "shared_capacity") {
			t.Fatalf("displaced %+v (%+v)", d, a)
		}
	}
	t.Logf("%d bookings displaced by the delay (%d pending as previewed)", len(recs), len(p.Displacements))
	if len(recs) == 0 || len(recs) != len(p.Displacements) {
		t.Fatalf("preview named %d, commit displaced %d", len(p.Displacements), len(recs))
	}
	// During training (Thu Nov 5 and Mon Nov 9 attended, 11.5 h): from
	// Nov 23 a fuller roster (Mon, Tue 7.5 h, Wed 4 h; 19 h within the
	// 20 h contract) completes on Mon Nov 30, independent Dec 1 again.
	runChecked(t, w, w.Calendar().StartOfDay(dayOf(w, "2026-11-09")).Add(21*simtime.Hour))
	full := both(api.Week{Mon: mon, Tue: mon, Wed: shiftAt("R3", "12:00", "16:00")})
	p = w.PreviewStaffing([]api.Action{setRoster(h, "2026-11-23", full)})
	if !p.OK || p.Rosters[0].ProductiveBefore != "2026-12-08" || p.Rosters[0].ProductiveAfter != "2026-12-01" || len(p.SlotsOpened) == 0 {
		t.Fatalf("sooner: %+v, opened %v (%v)", p.Rosters, p.SlotsOpened, p.Results)
	}
	mustApply(t, w, setRoster(h, "2026-11-23", full))
	w.Engine().CheckInvariants = true
	runChecked(t, w, w.Calendar().StartOfDay(dayOf(w, "2026-12-02")))
	e := employee(w, h)
	if e.Employment != api.EmploymentActive || e.Hired.TrainedMin < 2400 || e.Hired.ProductiveDate != "2026-12-01" {
		t.Fatalf("after training: %+v %+v", e.Employment, e.Hired)
	}
}

// The rest rule across version boundaries, measured from the shift cap
// (start + 12 h, at most 23:59:59) like every other rest check: a Sunday
// shift from 11:30 (cap 23:30) under one version forbids a Monday start
// before 09:30 under the next. Withdrawing a version in between can bring
// two such shifts together, and is refused too; exactly 10 hours is lawful.
func TestRosterRestAcrossVersions(t *testing.T) {
	w := atTakeover(t, `{}`, 1)
	orig := rosterNow(t, w, "E03")
	sunday := orig // Tue–Fri and Sunday 11:30–15:30, no Monday: 36 h a week
	sunday.A.Mon, sunday.B.Mon = nil, nil
	sun := shiftAt("R1", "11:30", "15:30")
	sunday.A.Sun, sunday.B.Sun = sun, sun
	early := orig // Mon 07:45
	if res := w.Apply([]api.Action{setRoster("E03", "2026-11-02", sunday), setRoster("E03", "2026-11-09", early)}); res[1].Code != "invalid_roster" ||
		!strings.Contains(res[1].Message, "Sun Nov 8") || !strings.Contains(res[1].Message, "23:30") {
		t.Fatalf("boundary rest: %v %s", codes(res), res[1].Message)
	}
	late := orig
	mon := *orig.A.Mon
	mon.Start, mon.Covers = "09:30", []api.Cover{}
	mon.Breaks = [][2]string{{"13:00", "13:30"}}
	late.A.Mon, late.B.Mon = &mon, &mon
	mustApply(t, w, setRoster("E03", "2026-11-02", sunday), setRoster("E03", "2026-11-09", late), setRoster("E03", "2026-11-16", early))
	// Withdrawing Nov 9 would put Sunday Nov 15 (under Nov 2's) next to Monday Nov 16's 07:45.
	if res := w.Apply([]api.Action{removeRoster("E03", "2026-11-09")}); res[0].Code != "invalid_roster" || !strings.Contains(res[0].Message, "Sun Nov 15") {
		t.Fatalf("withdrawal: %+v", res[0])
	}
}

// samTueOff is Sam's roster without Tuesdays (dozens of bookings lose
// coverage) from the first lawful Monday.
func samTueOff(b *testing.B, w *sim.World) api.Action {
	p, _ := w.RosterPlan("E04")
	r := p.Versions[0].Roster
	r.A.Tue, r.B.Tue = nil, nil
	return setRoster("E04", p.EarliestEffective, r)
}

// A roster preview at the takeover: validation, the composed timeline,
// the booked horizon re-planned, survivors, coverage differences and
// provisional replacements.
func BenchmarkRosterPreview(b *testing.B) {
	w := sim.New(established(b), rng.SeedFromInt(1))
	w.Engine().RunUntil(w.StartTime(), nil)
	act := []api.Action{samTueOff(b, w)}
	b.ResetTimer()
	for range b.N {
		w.PreviewStaffing(act)
	}
}

// Committing it: the version, one reconciliation of the booked horizon and
// the replacement offers (world building excluded).
func BenchmarkRosterCommit(b *testing.B) {
	cfg := established(b)
	for range b.N {
		b.StopTimer()
		w := sim.New(cfg, rng.SeedFromInt(1))
		w.Engine().RunUntil(w.StartTime(), nil)
		act := []api.Action{samTueOff(b, w)}
		b.StartTimer()
		w.Apply(act)
	}
}

// Roster previews are pure: any number leaves every digest as it was,
// including after a version exists (its training view caches are not
// state), and one made on an identical past with a different hidden
// future is byte-identical.
func TestRosterPreviewReadOnly(t *testing.T) {
	a, b := atTakeover(t, `{}`, 7), atTakeover(t, `{}`, 7)
	b.DropOrdersFrom(b.StartTime().Add(simtime.Hour)) // another hidden future
	r := rosterNow(t, a, "E04")
	r.A.Tue, r.B.Tue = nil, nil
	r2 := rosterNow(t, a, "E03")
	r2.A.Fri = nil
	batch := []api.Action{setRoster("E04", "2026-11-02", r), setRoster("E03", "2026-11-09", r2), removeRoster("E03", "2026-11-16")}
	for _, w := range []*sim.World{a, b} {
		mustApply(t, w, setRoster("E03", "2026-11-16", r2))
	}
	digests := func(w *sim.World) []string {
		return []string{w.StateDigest(), w.LogHash(), w.LedgerHash(), w.ActionDigest()}
	}
	before := digests(a)
	pa := a.PreviewStaffing(batch)
	for range 3 {
		a.PreviewStaffing(batch)
		a.ValidateBatch(batch)
		a.RosterPlan("E03")
	}
	if !slices.Equal(before, digests(a)) {
		t.Fatal("previews changed the run")
	}
	ja, _ := json.Marshal(pa)
	jb, _ := json.Marshal(b.PreviewStaffing(batch))
	if string(ja) != string(jb) {
		t.Fatal("the preview depends on the hidden future")
	}
	if !pa.OK || len(pa.Rosters) != 3 || len(pa.Displacements) == 0 || len(pa.Guaranteed) != 3 || len(pa.SlotsClosed) == 0 {
		t.Fatalf("preview %+v", pa)
	}
}
