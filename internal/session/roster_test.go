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

// V0.2 Slice 3: set_roster in session records — versions accepted on one
// day and taking effect weeks later replay authoritatively from any save
// point; a 0.2.0-dev.2 (Slice 2) record replays with its own action set,
// every digest verified, and continues as the prefix of a new record.

// riley is Riley's (E03) roster without the given weekdays (both weeks).
func riley(s *Session, off ...string) api.Roster {
	p, _ := s.RosterPlan("E03")
	r := p.Versions[0].Roster
	for _, w := range []*api.Week{&r.A, &r.B} {
		days := map[string]**api.Shift{"Mon": &w.Mon, "Tue": &w.Tue, "Wed": &w.Wed, "Thu": &w.Thu, "Fri": &w.Fri, "Sat": &w.Sat, "Sun": &w.Sun}
		for _, d := range off {
			*days[d] = nil
		}
	}
	return r
}

func rosterAct(id, effective string, r api.Roster) api.Action {
	return api.Action{Type: api.ActSetRoster, Op: api.RosterOpSet, Employee: id, Effective: effective, Roster: &r}
}

// rosterPlay accepts roster versions effective weeks ahead — Riley's
// (two successive, one later withdrawn), a recruited hire's before its
// start — with a hire and V0.1 actions around them.
func rosterPlay(t *testing.T, s *Session) {
	t.Helper()
	at(t, s, 0, "13:00")
	if res := s.Apply(eveningHire(s, "ct-1", 28), rosterAct("E03", "2026-11-02", riley(s, "Fri"))); !res[0].OK || !res[1].OK {
		t.Fatalf("batch: %+v", res)
	}
	s.Apply(api.Action{Type: api.ActSetWorklistPolicy, Policy: api.WorklistFIFO})
	at(t, s, 2, "09:00")
	if res := s.Apply(rosterAct("E03", "2026-11-16", riley(s, "Mon")), rosterAct("H00001", "2026-11-09", eveningRoster("Mon", "Tue", "Wed"))); !res[0].OK || !res[1].OK {
		t.Fatalf("batch: %+v", res)
	}
	at(t, s, 9, "10:00")
	if res := s.Apply(api.Action{Type: api.ActSetRoster, Op: api.RosterOpRemove, Employee: "E03", Effective: "2026-11-16"}); !res[0].OK {
		t.Fatalf("remove: %+v", res)
	}
	s.Apply(rosterAct("E03", "2026-11-02", riley(s, "Fri"))) // identical: unchanged
}

// eveningRoster is the evening CT hire's roster on the given weekdays.
func eveningRoster(days ...string) api.Roster {
	var w api.Week
	p := map[string]**api.Shift{"Mon": &w.Mon, "Tue": &w.Tue, "Wed": &w.Wed, "Thu": &w.Thu, "Fri": &w.Fri}
	for _, d := range days {
		*p[d] = &api.Shift{Start: "12:00", End: "20:00", Breaks: [][2]string{{"15:30", "16:00"}}, Assignment: "R3", Covers: []api.Cover{}}
	}
	return api.Roster{A: w, B: w}
}

// Saved before, on and after each effective date — and in training — a
// record replays to the same digests, and the loaded session continues
// identically; chunking and the Max-speed budget change nothing.
func TestRosterSaveReplayAcrossEffectiveDates(t *testing.T) {
	s := openFile(t, "mini", 31, nil)
	rosterPlay(t, s)
	for _, d := range []int64{9, 20, 21, 29, 34, 36} {
		at(t, s, d, "21:00")
		p := saveTemp(t, s)
		l, rep, err := Load(p, ReplayOptions{})
		if err != nil || !rep.OK || rep.Compatibility() != "native_exact" {
			t.Fatalf("day %d: %v %s", d, err, repString(rep))
		}
		if l.Digests() != s.Digests() {
			t.Fatalf("day %d: loaded digests differ", d)
		}
		at(t, l, d+1, "21:00")
		c := openFile(t, "mini", 31, nil)
		rosterPlay(t, c)
		at(t, c, d+1, "21:00")
		if l.Digests() != c.Digests() {
			t.Fatalf("day %d: the loaded session continues differently", d)
		}
	}
	r, err := s.Record()
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := r.Encode()
	if !strings.Contains(string(raw), `"op": "remove"`) || !strings.Contains(string(raw), `"effective": "2026-11-16"`) {
		t.Fatal("record lacks the set_roster fields")
	}
	for _, step := range []simtime.Duration{41 * simtime.Minute, 11 * simtime.Hour} {
		_, rep, err := Replay(r, ReplayOptions{Step: step})
		if err != nil || rep.Replayed != s.Digests() {
			t.Fatalf("step %s: %v %s", step, err, repString(rep))
		}
	}
	m := openFile(t, "mini", 31, nil)
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

const slice2Fixture = "../../testdata/sessions/mini-seed21-slice2.session.json"

// A record written by the Slice 2 build (0.2.0-dev.2, testdata/sessions):
// there "set_roster" was an unknown V0.1-path action, and a batch mixing
// it with a termination was mixed_batch naming hire and terminate only.
// It replays with that action set and every digest verified (the state
// encoding is unchanged), continues with a real set_roster, and saves as
// v2 with the Slice 2 record embedded as its prefix; tampering is refused.
func TestSlice2RecordReplaysAndContinues(t *testing.T) {
	l, rep, err := Load(slice2Fixture, ReplayOptions{})
	if err != nil || !rep.OK || !rep.Earlier || rep.Compatibility() != "earlier_build_exact" || rep.Replayed != rep.Recorded {
		t.Fatalf("Slice 2 record: %v %s", err, repString(rep))
	}
	var unknown, mixed bool
	for _, x := range l.w.ActionLog() {
		unknown = unknown || (x.Action.Type == api.ActSetRoster && x.Code == "unknown_action")
		mixed = mixed || (x.Code == "mixed_batch" && strings.Contains(x.Message, "(hire, terminate)"))
	}
	if !unknown || !mixed {
		t.Fatal("the fixture's Slice 2 results are not there")
	}
	// The same input now has its Slice 3 meaning.
	if got := l.ValidateBatch(api.Action{Type: api.ActSetRoster, Employee: "E03"}); got[0].Code != "invalid_schema" {
		t.Fatalf("set_roster without op now: %+v", got[0])
	}
	at(t, l, 10, "09:00")
	if res := l.Apply(rosterAct("E03", "2026-11-09", riley(l, "Fri"))); !res[0].OK {
		t.Fatalf("set_roster after the Slice 2 record: %+v", res)
	}
	at(t, l, 12, "12:00")
	p := saveTemp(t, l)
	raw, _ := os.ReadFile(p)
	cont, err := DecodeRecord(raw)
	if err != nil || cont.Legacy == nil || cont.Legacy.Schema != RecordSchema || cont.Legacy.Engine != "0.2.0-dev.2" ||
		cont.Legacy.BatchMode != "v0.2_slice2" || cont.Legacy.Outcome != CompatExact || cont.Legacy.State != CompatExact || len(cont.Batches) != 1 {
		t.Fatalf("continued record: %v %+v", err, cont.Legacy)
	}
	fixture, _ := os.ReadFile(slice2Fixture)
	var a, b bytes.Buffer
	json.Compact(&a, fixture)
	json.Compact(&b, cont.Legacy.Record)
	if a.String() != b.String() {
		t.Fatal("the Slice 2 record is not embedded verbatim")
	}
	l2, rep, err := Load(p, ReplayOptions{})
	if err != nil || rep.Compatibility() != "prefix:earlier_build_exact+native_exact" || l2.Digests() != l.Digests() {
		t.Fatalf("continued load: %v %s", err, repString(rep))
	}
	for _, c := range []struct {
		name string
		edit func(*Record)
		want error
	}{
		{"batch mode", func(r *Record) { r.Legacy.BatchMode = batchModeLegacy }, ErrIncompatible},
		{"prefix content", func(r *Record) {
			var c bytes.Buffer
			json.Compact(&c, r.Legacy.Record)
			r.Legacy.Record = []byte(strings.Replace(c.String(), `"minutes":30`, `"minutes":31`, 1))
		}, ErrCorrupt},
		{"prefix engine", func(r *Record) { r.Legacy.Engine = "0.2.0-dev.3" }, ErrCorrupt},
		{"transition", func(r *Record) { r.Legacy.Transition.Log = strings.Repeat("0", 64) }, ErrMismatch},
	} {
		q := copyFile(t, p)
		rewrite(t, q, c.edit)
		if _, _, err := Load(q, ReplayOptions{}); !errors.Is(err, c.want) {
			t.Errorf("%s: %v, want %v", c.name, err, c.want)
		}
	}
}

// A v1 record continued with roster versions saves as v2 with the v1
// prefix, and replays: prefix, transition digests, then the native batches
// across the effective date.
func TestV1ContinuationWithRoster(t *testing.T) {
	s := openFile(t, "mini", 32, nil)
	play(t, s)
	r, err := s.Record()
	if err != nil {
		t.Fatal(err)
	}
	r.Engine, r.Digests.State, r.Digests.Run = "0.2.0-dev.1", strings.Repeat("a", 64), strings.Repeat("b", 64)
	v1, err := encodeV1(r)
	if err != nil {
		t.Fatal(err)
	}
	p := saveTemp(t, s)
	if err := os.WriteFile(p, v1, 0o644); err != nil {
		t.Fatal(err)
	}
	l, _, err := Load(p, ReplayOptions{})
	if err != nil {
		t.Fatal(err)
	}
	at(t, l, 3, "09:00")
	if res := l.Apply(rosterAct("E03", "2026-11-02", riley(l, "Wed"))); !res[0].OK {
		t.Fatalf("%+v", res)
	}
	at(t, l, 22, "12:00")
	l2, rep, err := Load(saveTemp(t, l), ReplayOptions{})
	if err != nil || rep.Compatibility() != "prefix:legacy_outcomes_verified+native_exact" || l2.Digests() != l.Digests() {
		t.Fatalf("v1 → v2 with set_roster: %v %s", err, repString(rep))
	}
}

const slice3Fixture = "../../testdata/sessions/mini-seed22-slice3.session.json"

// A record written by the Slice 3 build (0.2.0-dev.3, testdata/sessions):
// set_roster set and remove, hire, terminate and a V0.1 extension, all
// accepted. This build has the same action set and state encoding, so the
// record replays with every digest verified, continues, and saves with the
// Slice 3 record embedded verbatim as its prefix.
func TestSlice3RecordReplaysAndContinues(t *testing.T) {
	l, rep, err := Load(slice3Fixture, ReplayOptions{})
	if err != nil || !rep.OK || !rep.Earlier || rep.Compatibility() != "earlier_build_exact" || rep.Replayed != rep.Recorded {
		t.Fatalf("Slice 3 record: %v %s", err, repString(rep))
	}
	if p, _ := l.RosterPlan("E03"); len(p.Versions) != 2 || p.Versions[1].Effective != "2026-11-09" {
		t.Fatalf("Riley's roster plan after replay: %+v", p.Versions)
	}
	for _, step := range []simtime.Duration{37 * simtime.Minute, 9 * simtime.Hour} {
		if _, rep, err := Load(slice3Fixture, ReplayOptions{Step: step}); err != nil || rep.Replayed != rep.Recorded {
			t.Fatalf("step %s: %v %s", step, err, repString(rep))
		}
	}
	at(t, l, 10, "09:00")
	if res := l.Apply(rosterAct("E03", "2026-11-16", riley(l, "Mon"))); !res[0].OK {
		t.Fatalf("set_roster after the Slice 3 record: %+v", res)
	}
	at(t, l, 12, "12:00")
	p := saveTemp(t, l)
	raw, _ := os.ReadFile(p)
	cont, err := DecodeRecord(raw)
	if err != nil || cont.Legacy == nil || cont.Legacy.Engine != "0.2.0-dev.3" || cont.Legacy.BatchMode != "v0.2_slice3" ||
		cont.Legacy.Outcome != CompatExact || cont.Legacy.State != CompatExact || len(cont.Batches) != 1 {
		t.Fatalf("continued record: %v %+v", err, cont.Legacy)
	}
	fixture, _ := os.ReadFile(slice3Fixture)
	var a, b bytes.Buffer
	json.Compact(&a, fixture)
	json.Compact(&b, cont.Legacy.Record)
	if a.String() != b.String() {
		t.Fatal("the Slice 3 record is not embedded verbatim")
	}
	l2, rep, err := Load(p, ReplayOptions{})
	if err != nil || rep.Compatibility() != "prefix:earlier_build_exact+native_exact" || l2.Digests() != l.Digests() {
		t.Fatalf("continued load: %v %s", err, repString(rep))
	}
	q := copyFile(t, p)
	rewrite(t, q, func(r *Record) { r.Legacy.BatchMode = batchModeCurrent })
	if _, _, err := Load(q, ReplayOptions{}); !errors.Is(err, ErrIncompatible) {
		t.Errorf("batch mode tampered: %v", err)
	}
}
