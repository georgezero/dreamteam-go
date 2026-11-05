package sim_test

import (
	"encoding/json"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/sim"
)

// V0.13b E4: the slots a termination-only batch closes are exactly the
// slots that stop being bookable when it is committed, from its end date
// to the last materialized day; asking changes nothing; anything but a
// valid termination-only batch is refused; the default preview is
// unchanged.

func TestSlotsClosedExact(t *testing.T) {
	for _, c := range []struct {
		emp, last string
		seed      int64
	}{{"E03", "2026-10-27", 1}, {"E04", "2026-10-27", 1}, {"E03", "2026-11-10", 2}} {
		w := atTakeover(t, `{}`, c.seed)
		act := terminateOf(c.emp, c.last)
		before := digestsOf(w)
		pv := w.PreviewStaffing([]api.Action{act})
		rep := w.PreviewSlotsClosed([]api.Action{act})
		if digestsOf(w) != before {
			t.Fatal("PreviewSlotsClosed changed the run")
		}
		if !rep.Available || rep.Schema != sim.SlotsClosedSchema {
			t.Fatalf("%s: %+v", c.emp, rep)
		}
		end := pv.Terminations[0].EndDate
		was := w.BookableByCoverage(dayOf(w, end))
		mustApply(t, w, act)
		now := w.BookableByCoverage(dayOf(w, end))
		closed, booked := map[string]int{}, 0
		total := 0
		for k, m := range was {
			if _, ok := now[k]; !ok {
				closed[m]++
				total++
			}
		}
		for k := range now {
			if _, ok := was[k]; !ok {
				t.Fatalf("%s: slot %s opened by a termination", c.emp, k)
			}
		}
		if rep.Total != total || rep.From != end {
			t.Fatalf("%s: report %d slots from %s; committed run closed %d from %s", c.emp, rep.Total, rep.From, total, end)
		}
		sum := 0
		for _, mc := range rep.ByModality {
			if closed[mc.Modality] != mc.Count {
				t.Fatalf("%s %s: report %d, committed %d", c.emp, mc.Modality, mc.Count, closed[mc.Modality])
			}
			sum += mc.Count
		}
		for _, d := range rep.ByDay {
			booked += d.BookedNow
		}
		if sum != total || booked != rep.BookedNow || rep.BookedNow > rep.Total {
			t.Fatalf("%s: inconsistent report %+v", c.emp, rep)
		}
		t.Logf("%s last day %s (seed %d): %d slots closed %s..%s (%d days), %d booked now, displaced %d",
			c.emp, c.last, c.seed, rep.Total, rep.From, rep.To, rep.Days, rep.BookedNow, pv.Terminations[0].Displaced)
	}
}

func TestSlotsClosedRefusals(t *testing.T) {
	w := atTakeover(t, `{}`, 1)
	before := digestsOf(w)
	for _, c := range []struct {
		acts []api.Action
		why  string
	}{
		{nil, sim.SlotsClosedNotTerminationOnly},
		{[]api.Action{{Type: api.ActSetWorklistPolicy, Policy: "fifo"}}, sim.SlotsClosedNotTerminationOnly},
		{[]api.Action{terminateOf("E03", "2026-10-27"), {Type: api.ActSetWorklistPolicy, Policy: "fifo"}}, sim.SlotsClosedNotTerminationOnly},
		{[]api.Action{terminateOf("E03", "2026-10-20")}, sim.SlotsClosedInvalid}, // 7 days' notice
		{[]api.Action{terminateOf("E99", "2026-10-27")}, sim.SlotsClosedInvalid},
	} {
		rep := w.PreviewSlotsClosed(c.acts)
		if rep.Available || rep.Reason != c.why || rep.Total != 0 || len(rep.ByDay) != 0 {
			t.Errorf("%+v: %+v", c.acts, rep)
		}
	}
	if digestsOf(w) != before {
		t.Fatal("a refused report changed the run")
	}
}

// The default preview is byte-identical whether or not the report is
// asked for, before and after, and a termination-only preview still has
// no slotsClosed field.
func TestSlotsClosedOptInOnly(t *testing.T) {
	w := atTakeover(t, `{}`, 1)
	act := []api.Action{terminateOf("E03", "2026-10-27")}
	a, _ := json.Marshal(w.PreviewStaffing(act))
	w.PreviewSlotsClosed(act)
	b, _ := json.Marshal(w.PreviewStaffing(act))
	if string(a) != string(b) {
		t.Fatal("the preview changed after a slots-closed report")
	}
	var m map[string]any
	if err := json.Unmarshal(a, &m); err != nil {
		t.Fatal(err)
	}
	if _, ok := m["slotsClosed"]; ok {
		t.Fatal("a termination-only preview reports slotsClosed by default")
	}
}

// The counting rule is the roster preview's: for a roster batch, the same
// count over every materialized day equals PreviewStaffing's SlotsClosed.
func TestSlotsClosedRuleMatchesRosterPreview(t *testing.T) {
	w := atTakeover(t, `{}`, 1)
	r := rosterNow(t, w, "E03")
	mon := *r.A.Mon
	mon.Covers = []api.Cover{}
	r.A.Mon, r.B.Mon = &mon, &mon
	act := []api.Action{setRoster("E03", "2026-11-02", r)}
	p := w.PreviewStaffing(act)
	got := w.CoverageClosedAll(act)
	if len(p.SlotsClosed) == 0 {
		t.Fatal("no slots closed by the roster batch")
	}
	n := 0
	for _, mc := range p.SlotsClosed {
		if got[mc.Modality] != mc.Count {
			t.Fatalf("%s: rule %d, preview %d", mc.Modality, got[mc.Modality], mc.Count)
		}
		n += mc.Count
	}
	for _, c := range got {
		n -= c
	}
	if n != 0 {
		t.Fatalf("rule %v, preview %+v", got, p.SlotsClosed)
	}
}
