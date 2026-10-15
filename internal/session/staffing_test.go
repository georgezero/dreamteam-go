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

// V0.2 Slice 2: session records v2 — the staffing actions and their
// lifecycle replayed authoritatively, saved at every lifecycle boundary;
// v1 records read, continued and saved as v2 with their v1 prefix.

// dateOf is the date d days after the start date.
func dateOf(s *Session, d int64) string {
	cal := s.Calendar()
	return cal.Date(cal.StartOfDay(cal.DayIndex(s.w.StartTime()) + d)).String()
}

func eveningHire(s *Session, req string, startDay int64) api.Action {
	sh := func() *api.Shift {
		return &api.Shift{Start: "12:00", End: "20:00", Breaks: [][2]string{{"15:30", "16:00"}}, Assignment: "R3", Covers: []api.Cover{}}
	}
	wk := api.Week{Mon: sh(), Tue: sh(), Wed: sh(), Thu: sh(), Fri: sh()}
	return api.Action{Type: api.ActHire, RequestID: req, Role: "technologist", Skills: []string{"CT"}, RateCents: 4000,
		ContractedWeeklyMinutes: 2250, Roster: &api.Roster{A: wk, B: wk}, StartDate: dateOf(s, startDay)}
}

// staffingPlay applies V0.2 and V0.1 batches, at the same instant too.
func staffingPlay(t *testing.T, s *Session) {
	t.Helper()
	at(t, s, 0, "13:00")
	res := s.Apply(eveningHire(s, "ct-1", 28), api.Action{Type: api.ActTerminate, Employee: "E04", LastDay: dateOf(s, 14)})
	if !res[0].OK || !res[1].OK {
		t.Fatalf("staffing batch: %+v", res)
	}
	// A legacy batch and a refused V0.2 batch at the same instant.
	s.Apply(api.Action{Type: api.ActSetWorklistPolicy, Policy: api.WorklistFIFO})
	s.Apply(api.Action{Type: api.ActTerminate, Employee: "E04", LastDay: dateOf(s, 20)})
	at(t, s, 3, "10:00")
	s.Apply(eveningHire(s, "ct-1", 28)) // an identical repeat: unchanged
}

// Saved at each lifecycle boundary — on notice, after the departure,
// after severance, in training, after training — a record replays to the
// same digests, and the loaded session continues identically.
func TestStaffingSaveReplayAcrossLifecycle(t *testing.T) {
	s := openFile(t, "mini", 11, nil)
	staffingPlay(t, s)
	for _, d := range []int64{3, 15, 17, 29, 37} {
		at(t, s, d, "21:00")
		if d == 37 {
			// A refused legacy action naming the hire (recorded, replayed).
			if r := s.Apply(api.Action{Type: api.ActExtendShift, Employee: "H00001", Minutes: 0}); r[0].OK {
				t.Fatalf("zero-minute extension accepted: %+v", r)
			}
		}
		p := saveTemp(t, s)
		l, rep, err := Load(p, ReplayOptions{})
		if err != nil || !rep.OK || rep.Compatibility() != "native_exact" {
			t.Fatalf("day %d: %v %s", d, err, repString(rep))
		}
		if l.Digests() != s.Digests() {
			t.Fatalf("day %d: loaded digests differ", d)
		}
		at(t, l, d+1, "21:00")
		c := openFile(t, "mini", 11, nil)
		staffingPlay(t, c)
		if d == 37 {
			at(t, c, 37, "21:00")
			c.Apply(api.Action{Type: api.ActExtendShift, Employee: "H00001", Minutes: 0})
		}
		at(t, c, d+1, "21:00")
		if l.Digests() != c.Digests() {
			t.Fatalf("day %d: the loaded session continues differently", d)
		}
	}
	raw, err := os.ReadFile(saveTemp(t, s))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"schema": "dreamteam.session/v2"`) || !strings.Contains(string(raw), `"employeeId": "H00001"`) {
		t.Fatal("record is not v2 with the hire's ID")
	}
}

// Chunking, unrelated events and the Max-speed event budget never change
// a run with staffing actions.
func TestStaffingReplayChunkingAndSpeed(t *testing.T) {
	s := openFile(t, "mini", 12, nil)
	staffingPlay(t, s)
	at(t, s, 31, "12:00")
	r, err := s.Record()
	if err != nil {
		t.Fatal(err)
	}
	for _, step := range []simtime.Duration{37 * simtime.Minute, 7 * simtime.Hour} {
		_, rep, err := Replay(r, ReplayOptions{Step: step})
		if err != nil || rep.Replayed != s.Digests() {
			t.Fatalf("step %s: %v %s", step, err, repString(rep))
		}
	}
	// Max speed: advance in event-budgeted frames to the same instants.
	m := openFile(t, "mini", 12, nil)
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

// A v1 record (here as a V0.2 Slice 1 build wrote it: another state
// encoding) loads with its outcomes verified, continues with V0.2 actions
// and saves as v2 with the v1 record embedded verbatim; that v2 record
// replays — the prefix in the legacy mode, the transition digests, then
// the native batches exactly. Tampering anywhere is refused.
func TestV1ContinuationToV2(t *testing.T) {
	s := openFile(t, "mini", 13, nil)
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
	l, rep, err := Load(p, ReplayOptions{})
	if err != nil || !rep.StateChanged || rep.Compatibility() != "legacy_outcomes_verified" {
		t.Fatalf("v1 load: %v %s", err, repString(rep))
	}
	at(t, l, 2, "09:00")
	if res := l.Apply(eveningHire(l, "after-v1", 30)); !res[0].OK {
		t.Fatalf("hire after v1: %+v", res)
	}
	at(t, l, 4, "12:00")
	p2 := saveTemp(t, l)
	raw, _ := os.ReadFile(p2)
	cont, err := DecodeRecord(raw)
	if err != nil || cont.Legacy == nil || cont.Legacy.Engine != "0.2.0-dev.1" || cont.Legacy.Outcome != CompatVerified ||
		cont.Legacy.State != CompatSchemaChanged || len(cont.Batches) != 1 || cont.Batches[0].Results[0].EmployeeID != "H00001" {
		t.Fatalf("continued record: %v %+v", err, cont)
	}
	// Verbatim up to JSON whitespace (the file is indented): the v1 record's
	// own checksum still verifies.
	var emb bytes.Buffer
	if err := json.Compact(&emb, cont.Legacy.Record); err != nil || emb.String() != string(mustCompact(t, v1)) {
		t.Fatal("the v1 record is not embedded verbatim")
	}
	l2, rep, err := Load(p2, ReplayOptions{})
	if err != nil || rep.Prefix == nil || !rep.Prefix.StateChanged || rep.Compatibility() != "prefix:legacy_outcomes_verified+native_exact" {
		t.Fatalf("v2 continued load: %v %s", err, repString(rep))
	}
	if l2.Digests() != l.Digests() {
		t.Fatal("continued replay differs")
	}
	// Saving the reloaded session again keeps the same prefix.
	raw2, _ := os.ReadFile(saveTemp(t, l2))
	if again, _ := DecodeRecord(raw2); again == nil || again.Legacy == nil || again.Legacy.Checksum != cont.Legacy.Checksum {
		t.Fatal("prefix lost on a second save")
	}
	// Tampering: the transition digests, the prefix's own content, a
	// native batch moved into the prefix.
	for _, c := range []struct {
		name string
		edit func(*Record)
		want error
	}{
		{"transition", func(r *Record) { r.Legacy.Transition.State = strings.Repeat("0", 64) }, ErrMismatch},
		{"prefix content", func(r *Record) {
			var c bytes.Buffer
			json.Compact(&c, r.Legacy.Record)
			r.Legacy.Record = []byte(strings.Replace(c.String(), `"minutes":120`, `"minutes":121`, 1))
		}, ErrCorrupt},
		{"batch numbering", func(r *Record) { r.Batches[0].Batch = r.Legacy.LastBatch }, ErrCorrupt},
		{"native result", func(r *Record) { r.Batches[0].Results[0].EmployeeID = "H00002" }, ErrMismatch},
	} {
		q := copyFile(t, p2)
		rewrite(t, q, c.edit)
		if _, _, err := Load(q, ReplayOptions{}); !errors.Is(err, c.want) {
			t.Errorf("%s: %v, want %v", c.name, err, c.want)
		}
	}
	// A v1 file carrying a V0.2 field is not a v1 file.
	bad := strings.Replace(string(v1), `"type": "extend_shift"`, `"type": "extend_shift", "lastDay": "2026-11-01"`, 1)
	if _, err := DecodeRecord([]byte(bad)); !errors.Is(err, ErrCorrupt) {
		t.Errorf("v1 with a V0.2 field: %v", err)
	}
}

// mustCompact is a record file's compact JSON (as embedded).
func mustCompact(t *testing.T, raw []byte) []byte {
	t.Helper()
	r, err := decodeV1(raw)
	if err != nil {
		t.Fatal(err)
	}
	return r.v1
}

// A V0.1 build recorded "hire" and "terminate" as unknown actions; a v1
// record holding them replays with those V0.1 results (legacy batch
// semantics), while a v2 batch gives them their V0.2 meaning.
func TestV1RecordWithV02TypeNamesReplaysLegacy(t *testing.T) {
	s := openFile(t, "mini", 14, nil)
	at(t, s, 0, "09:00")
	res := s.w.ApplyLegacy([]api.Action{{Type: api.ActHire, Employee: "E01"}, {Type: api.ActTerminate, Employee: "E04"},
		{Type: api.ActSetWorklistPolicy, Policy: api.WorklistFIFO}})
	if res[0].Code != "unknown_action" || res[1].Code != "unknown_action" || !strings.Contains(res[0].Message, "V0.1:") || !res[2].OK {
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
		t.Fatalf("v1 record with V0.2 type names: %v %s", err, repString(rep))
	}
	if got := s.Apply(api.Action{Type: api.ActHire, Employee: "E01"})[0]; got.Code != "invalid_schema" {
		t.Fatalf("v2 meaning: %+v", got)
	}
}
