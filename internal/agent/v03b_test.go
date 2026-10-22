package agent

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/session"
	"github.com/georgezero/dreamteam-go/internal/simtime"
	"github.com/georgezero/dreamteam-go/internal/version"
)

// 1: the V0.3a transcript's requests get the same responses, byte for
// byte, except hello's engine and ops, which name the build.
func TestLegacyTranscript(t *testing.T) {
	if testing.Short() {
		t.Skip("a 30-day run")
	}
	f, err := os.Open("../../docs/agent/noop-30d.transcript.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	t.Chdir(t.TempDir()) // the transcript saves to a relative path
	_, h := open(t, "sustainable", 909)
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<26)
	n := 0
	for sc.Scan() {
		req := slices.Clone(sc.Bytes())
		if !sc.Scan() {
			t.Fatal("unpaired transcript")
		}
		want := sc.Bytes()
		got := h.Handle(req)
		if strings.Contains(string(req), `"op":"hello"`) {
			var a, b map[string]any
			json.Unmarshal(got, &a)
			json.Unmarshal(want, &b)
			for _, m := range []map[string]any{a, b} {
				r := m["result"].(map[string]any)
				delete(r, "engine")
				delete(r, "ops")
			}
			if !reflect.DeepEqual(a, b) {
				t.Fatalf("hello differs beyond engine and ops:\n%s\n%s", got, want)
			}
			if a, b := mustJSON(Ops), `["hello","observe","sections","actions","validate","preview","apply","advance","save","digests","t90"]`; a != b {
				t.Fatalf("ops %s", a)
			}
			if !strings.Contains(string(got), `"engine":"`+version.Number+`"`) {
				t.Fatal("engine not reported")
			}
		} else if !bytes.Equal(got, want) {
			t.Fatalf("request %s:\n got %s\nwant %s", req, got, want)
		}
		n++
	}
	if n != 32 {
		t.Fatalf("%d requests", n)
	}
}

// leaves is every scalar of a decoded JSON value by path.
func leaves(v any, path string, out map[string]any) {
	switch x := v.(type) {
	case map[string]any:
		for k, y := range x {
			leaves(y, path+"/"+k, out)
		}
	case []any:
		if len(x) == 0 {
			out[path] = "[]"
		}
		for i, y := range x {
			leaves(y, path+"/"+itoa(i), out)
		}
	default:
		out[path] = v
	}
}

func itoa(i int) string { return strings.TrimSpace(mustJSON(i)) }

// 4, 5, 6, 7: projections agree with the full observation on every value
// they show, show nothing it does not, leave out exactly what compact
// says, are much smaller, and an invalid selection changes nothing.
func TestProjections(t *testing.T) {
	s, h := open(t, "sustainable", 905)
	s.AdvanceTo(s.Now().Add(21*simtime.Day + 5*simtime.Hour))
	before := digests(t, h)
	fullR := call(t, h, `{"id":1,"op":"observe"}`)
	var full map[string]any
	json.Unmarshal(fullR.Result, &full)
	fullLeaves := map[string]any{}
	leaves(full["observation"], "", fullLeaves)
	var list SectionList
	if r := call(t, h, `{"id":2,"op":"sections"}`); !r.OK || json.Unmarshal(r.Result, &list) != nil {
		t.Fatalf("%+v", r)
	}
	var names []string
	for _, x := range list.Sections {
		names = append(names, x.Name)
	}
	obsKeys := []string{}
	raw, _ := json.Marshal(s.Observe(20))
	o, _ := decodeObject(raw)
	for _, k := range o.keys {
		obsKeys = append(obsKeys, k)
	}
	if len(slices.DeleteFunc(slices.Clone(obsKeys), func(k string) bool { return slices.Contains(names, k) })) > 0 {
		t.Fatalf("sections %v miss observation keys %v", names, obsKeys)
	}
	cases := []string{
		`{"compact":true}`,
		`{"sections":["worklist","alerts","staff"]}`,
		`{"sections":["finance"],"compact":true}`,
		`{"sections":["recentLog"],"recentLog":20}`,
	}
	for _, n := range names {
		cases = append(cases, `{"sections":["`+n+`"]}`)
	}
	for _, args := range cases {
		r := call(t, h, `{"id":3,"op":"observe","args":`+args+`}`)
		var p Projected
		if !r.OK || json.Unmarshal(r.Result, &p) != nil {
			t.Fatalf("%s: %+v", args, r)
		}
		var got any
		json.Unmarshal(p.Observation, &got)
		gotLeaves := map[string]any{}
		leaves(got, "", gotLeaves)
		for k, v := range gotLeaves {
			if fv, ok := fullLeaves[k]; !ok || !reflect.DeepEqual(fv, v) {
				t.Fatalf("%s: %s is %v, the full observation's %v", args, k, v, fv)
			}
		}
		for _, hk := range headerKeys {
			if _, ok := got.(map[string]any)[hk]; !ok {
				t.Fatalf("%s: header %s missing", args, hk)
			}
		}
		// Compact leaves out exactly its listings, and says how many.
		for _, om := range compactOmits {
			fp := "/" + strings.ReplaceAll(om.Path, ".", "/")
			inFull := false
			for k := range fullLeaves {
				inFull = inFull || strings.HasPrefix(k, fp+"/") || k == fp
			}
			inGot := false
			for k := range gotLeaves {
				inGot = inGot || strings.HasPrefix(k, fp+"/") || k == fp
			}
			if p.Compact && inGot {
				t.Fatalf("%s: compact keeps %s", args, om.Path)
			}
			if _, counted := p.Omitted[om.Path]; p.Compact && inFull && slices.Contains(p.Sections, strings.Split(om.Path, ".")[0]) && !counted {
				t.Fatalf("%s: %s left out uncounted", args, om.Path)
			}
		}
	}
	// Compact keeps every value outside its listings.
	r := call(t, h, `{"id":4,"op":"observe","args":{"compact":true}}`)
	var p Projected
	json.Unmarshal(r.Result, &p)
	var got any
	json.Unmarshal(p.Observation, &got)
	gotLeaves := map[string]any{}
	leaves(got, "", gotLeaves)
	for k := range fullLeaves {
		omitted := slices.ContainsFunc(compactOmits, func(o struct{ Path, What string }) bool {
			return strings.HasPrefix(k, "/"+strings.ReplaceAll(o.Path, ".", "/"))
		})
		if _, ok := gotLeaves[k]; !ok && !omitted {
			t.Fatalf("compact drops %s, which it does not declare", k)
		}
	}
	if len(r.Result)*3 > len(fullR.Result) {
		t.Errorf("compact %d bytes against %d: not a material reduction", len(r.Result), len(fullR.Result))
	}
	for _, bad := range []string{
		`{"id":5,"op":"observe","args":{"sections":["secrets"]}}`,
		`{"id":6,"op":"observe","args":{"sections":[]}}`,
		`{"id":7,"op":"observe","args":{"sections":["staff","staff"]}}`,
		`{"id":8,"op":"observe","args":{"compact":"yes"}}`,
	} {
		if r := call(t, h, bad); r.OK {
			t.Fatalf("%s accepted", bad)
		}
	}
	if r := call(t, h, `{"id":9,"op":"observe","args":{"sections":["nope"]}}`); r.Error.Code != ErrUnknownSection {
		t.Fatalf("%+v", r.Error)
	}
	if after := digests(t, h); after != before {
		t.Fatal("observing changed the session")
	}
}

// 2, 3: t90 is the session's own evaluation (the one dreamteam t90
// prints), for a native session and one saved by the V0.2.0 release; it
// changes nothing and draws nothing: the session goes on exactly as one
// never evaluated.
func TestT90Op(t *testing.T) {
	if testing.Short() {
		t.Skip("runs eight 200-day worlds")
	}
	l, rep, err := session.Load("../../testdata/sessions/sustainable-seed906-v020.session.json", session.ReplayOptions{})
	if err != nil || !rep.OK {
		t.Fatal(err)
	}
	native, hn := open(t, "sustainable", 906)
	for _, x := range []*session.Session{native} {
		x.Apply(mustActions(t, `[{"type":"set_reading_allocation","op":"set","employee":"E07","effective":"2026-10-27","siteAllocationBP":6250},
			{"type":"set_reading_allocation","op":"set","employee":"E08","effective":"2026-10-27","siteAllocationBP":5250}]`)...)
		x.AdvanceTo(x.Now().Add(40 * simtime.Day))
	}
	hl := New(l)
	var evals []json.RawMessage
	for _, h := range []*Handler{hl, hn} {
		before := digests(t, h)
		r := call(t, h, `{"id":"t","op":"t90"}`)
		if !r.OK {
			t.Fatalf("%+v", r.Error)
		}
		var e struct {
			Seed       int64           `json:"seed"`
			Evaluation json.RawMessage `json:"evaluation"`
		}
		json.Unmarshal(r.Result, &e)
		want, err := h.s.T90()
		if err != nil {
			t.Fatal(err)
		}
		if e.Seed != 906 || string(e.Evaluation) != mustJSON(want) {
			t.Fatal("the t90 op's evaluation is not the session's")
		}
		if digests(t, h) != before {
			t.Fatal("t90 changed the session")
		}
		evals = append(evals, e.Evaluation)
	}
	if string(evals[0]) != string(evals[1]) {
		t.Fatal("the V0.2.0-saved session and the same plan played here evaluate differently")
	}
	// No randomness drawn: after t90, the session advances exactly as an
	// identical one that was never evaluated.
	twin, _ := open(t, "sustainable", 906)
	twin.Apply(mustActions(t, `[{"type":"set_reading_allocation","op":"set","employee":"E07","effective":"2026-10-27","siteAllocationBP":6250},
		{"type":"set_reading_allocation","op":"set","employee":"E08","effective":"2026-10-27","siteAllocationBP":5250}]`)...)
	twin.AdvanceTo(twin.Now().Add(40 * simtime.Day))
	for _, x := range []*session.Session{native, twin} {
		x.AdvanceTo(x.Now().Add(3 * simtime.Day))
	}
	if native.Digests() != twin.Digests() {
		t.Fatal("evaluating changed how the session continues")
	}
	// Unsupported scenarios: the CLI's refusal, as a stable code.
	_, he := open(t, "established", 1)
	call(t, he, `{"id":1,"op":"advance","args":{"minutes":60}}`)
	if r := call(t, he, `{"id":2,"op":"t90"}`); r.OK || r.Error.Code != ErrT90Scenario {
		t.Fatalf("%+v", r.Error)
	}
	_, hs := open(t, "sustainable", 907)
	if r := call(t, hs, `{"id":3,"op":"t90"}`); r.OK || r.Error.Code != ErrT90NotStarted {
		t.Fatalf("%+v", r.Error)
	}
}

func mustActions(t *testing.T, s string) []api.Action {
	t.Helper()
	var a []api.Action
	if err := json.Unmarshal([]byte(s), &a); err != nil {
		t.Fatal(err)
	}
	return a
}
