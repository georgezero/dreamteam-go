package session

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// V0.2 Slice 4: set_reading_allocation in session records — allocations
// accepted on one day and taking effect weeks later replay authoritatively
// from any save point; 0.2.0-dev.4 records replay with Slice 3's action
// set, where set_reading_allocation was an unknown V0.1-path action.

func allocAct(id, effective string, bp int64) api.Action {
	return api.Action{Type: api.ActSetReadingAllocation, Op: api.RosterOpSet, Employee: id, Effective: effective, SiteAllocationBP: bp}
}

// allocPlay: on day 0 two allocation changes (Oct 27 and Nov 2); on day 3
// Blake's is withdrawn and set again for Nov 3; on day 4 a mixed
// staffing and allocation batch (a hire, Avery back to 60% from Nov 10,
// the 14-day minimum commitment after Oct 27; 0.2.0-dev.6).
func allocPlay(t *testing.T, s *Session) {
	t.Helper()
	at(t, s, 0, "13:30")
	for _, b := range [][]api.Action{
		{allocAct("E07", "2026-10-27", 6250), allocAct("E08", "2026-11-02", 5250)},
	} {
		if res := s.Apply(b...); !res[0].OK || !res[1].OK {
			t.Fatalf("%+v", res)
		}
	}
	at(t, s, 3, "10:00")
	if res := s.Apply(api.Action{Type: api.ActSetReadingAllocation, Op: api.RosterOpRemove, Employee: "E08", Effective: "2026-11-02"},
		allocAct("E08", "2026-11-03", 5500)); !res[0].OK || !res[1].OK {
		t.Fatalf("%+v", res)
	}
	at(t, s, 4, "11:00")
	if res := s.Apply(api.Action{Type: api.ActHire, RequestID: "ct-eve", Role: "technologist", Skills: []string{"CT"}, RateCents: 4000,
		ContractedWeeklyMinutes: 2250, Roster: rosterPtr(eveningRoster("Mon", "Tue", "Wed", "Thu", "Fri")), StartDate: "2026-11-16"},
		allocAct("E07", "2026-11-10", 6000)); !res[0].OK || !res[1].OK {
		t.Fatalf("%+v", res)
	}
}

func rosterPtr(r api.Roster) *api.Roster { return &r }

// Saved before, on and after each effective date, a record replays to the
// same digests and the loaded session continues identically; chunking and
// the Max-speed event budget change nothing.
func TestAllocationSaveReplayAcrossEffectiveDates(t *testing.T) {
	s := openFile(t, "mini", 41, nil)
	allocPlay(t, s)
	for _, d := range []int64{9, 13, 14, 20, 21, 26, 27, 28} {
		at(t, s, d, "21:00")
		p := saveTemp(t, s)
		l, rep, err := Load(p, ReplayOptions{})
		if err != nil || !rep.OK || rep.Compatibility() != "native_exact" || l.Digests() != s.Digests() {
			t.Fatalf("day %d: %v %s", d, err, repString(rep))
		}
		at(t, l, d+1, "21:00")
		c := openFile(t, "mini", 41, nil)
		allocPlay(t, c)
		at(t, c, d+1, "21:00")
		if l.Digests() != c.Digests() {
			t.Fatalf("day %d: the loaded session continues differently", d)
		}
	}
	if a := s.ReadingAllocations(); len(a) != 2 || a[0].InEffectBP != 6000 || len(a[0].Versions) != 3 || a[1].InEffectBP != 5500 {
		t.Fatalf("allocations after replay: %+v", a)
	}
	r, err := s.Record()
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := r.Encode()
	if !strings.Contains(string(raw), `"siteAllocationBP": 5500`) {
		t.Fatal("record lacks the allocation fields")
	}
	for _, step := range []simtime.Duration{41 * simtime.Minute, 13 * simtime.Hour} {
		_, rep, err := Replay(r, ReplayOptions{Step: step})
		if err != nil || rep.Replayed != s.Digests() {
			t.Fatalf("step %s: %v %s", step, err, repString(rep))
		}
	}
	m := openFile(t, "mini", 41, nil)
	for _, b := range r.Batches {
		for m.Now() < simtime.Time(b.AtSec) {
			m.Advance(simtime.Time(b.AtSec).Sub(m.Now()), Options{EventBudget: 50})
		}
		m.Apply(b.Actions...)
	}
	for m.Now() < s.Now() {
		m.Advance(s.Now().Sub(m.Now()), Options{EventBudget: 50})
	}
	if m.Digests() != s.Digests() {
		t.Fatal("event-budgeted advance differs")
	}
}

// A V0.1 build recorded "set_reading_allocation" as an unknown action: a
// v1 record holding it replays with that result.
func TestV1RecordWithAllocationTypeReplaysLegacy(t *testing.T) {
	s := openFile(t, "mini", 15, nil)
	at(t, s, 0, "09:00")
	res := s.w.ApplyLegacy([]api.Action{{Type: api.ActSetReadingAllocation, Employee: "E07"}})
	if res[0].Code != "unknown_action" {
		t.Fatalf("legacy semantics: %+v", res)
	}
	at(t, s, 0, "15:00")
	r, err := s.Record()
	if err != nil {
		t.Fatal(err)
	}
	r.Engine = "0.1.0"
	v1, err := encodeV1(r)
	if err != nil {
		t.Fatal(err)
	}
	p := saveTemp(t, s)
	if err := os.WriteFile(p, v1, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, rep, err := Load(p, ReplayOptions{}); err != nil || !rep.OK {
		t.Fatalf("v1 record with the type name: %v %s", err, repString(rep))
	}
	if got := s.Apply(api.Action{Type: api.ActSetReadingAllocation, Employee: "E07"})[0]; got.Code != "invalid_schema" {
		t.Fatalf("v2 meaning: %+v", got)
	}
}

const dev4Fixture = "../../testdata/sessions/mini-seed23-dev4.session.json"

// A record written by 0.2.0-dev.4 (testdata/sessions): there
// "set_reading_allocation" was an unknown V0.1-path action, and a batch
// mixing it with a hire was mixed_batch naming hire, terminate and
// set_roster. It replays with Slice 3's action set and every digest
// verified, continues with a real allocation change, and saves with the
// dev.4 record embedded verbatim as its prefix.
func TestDev4RecordReplaysAndContinues(t *testing.T) {
	l, rep, err := Load(dev4Fixture, ReplayOptions{})
	if err != nil || !rep.OK || !rep.Earlier || rep.Compatibility() != "earlier_build_exact" || rep.Replayed != rep.Recorded {
		t.Fatalf("dev.4 record: %v %s", err, repString(rep))
	}
	for _, step := range []simtime.Duration{37 * simtime.Minute, 9 * simtime.Hour} {
		if _, rep, err := Load(dev4Fixture, ReplayOptions{Step: step}); err != nil || rep.Replayed != rep.Recorded {
			t.Fatalf("step %s: %v %s", step, err, repString(rep))
		}
	}
	at(t, l, 10, "09:00")
	if res := l.Apply(allocAct("E07", "2026-11-09", 6250)); res[0].Code != api.ActionAccepted {
		t.Fatalf("allocation after the dev.4 record: %+v", res)
	}
	at(t, l, 12, "12:00")
	p := saveTemp(t, l)
	raw, _ := os.ReadFile(p)
	cont, err := DecodeRecord(raw)
	if err != nil || cont.Legacy == nil || cont.Legacy.Engine != "0.2.0-dev.4" || cont.Legacy.BatchMode != "v0.2_slice3" ||
		cont.Legacy.Outcome != CompatExact || cont.Legacy.State != CompatExact || len(cont.Batches) != 1 {
		t.Fatalf("continued record: %v %+v", err, cont.Legacy)
	}
	fixture, _ := os.ReadFile(dev4Fixture)
	var a, b bytes.Buffer
	json.Compact(&a, fixture)
	json.Compact(&b, cont.Legacy.Record)
	if a.String() != b.String() {
		t.Fatal("the dev.4 record is not embedded verbatim")
	}
	l2, rep, err := Load(p, ReplayOptions{})
	if err != nil || rep.Compatibility() != "prefix:earlier_build_exact+native_exact" || l2.Digests() != l.Digests() {
		t.Fatalf("continued load: %v %s", err, repString(rep))
	}
	q := copyFile(t, p)
	rewrite(t, q, func(r *Record) { r.Legacy.BatchMode = batchModeCurrent })
	if _, _, err := Load(q, ReplayOptions{}); !errors.Is(err, ErrIncompatible) && !errors.Is(err, ErrMismatch) {
		t.Errorf("batch mode tampered: %v", err)
	}
}
