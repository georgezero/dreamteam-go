package agent

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/session"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Development seeds only (901–910).

func open(t *testing.T, ref string, seed int64) (*session.Session, *Handler) {
	t.Helper()
	s, err := session.Open(ref, seed)
	if err != nil {
		t.Fatal(err)
	}
	h := New(s)
	h.Ref = ref
	return s, h
}

type response struct {
	Protocol string          `json:"protocol"`
	ID       json.RawMessage `json:"id"`
	OK       bool            `json:"ok"`
	Result   json.RawMessage `json:"result"`
	Error    *Error          `json:"error"`
}

// call sends one request and decodes the response strictly.
func call(t *testing.T, h *Handler, line string) response {
	t.Helper()
	out := h.Handle([]byte(line))
	if bytes.ContainsRune(out, '\n') {
		t.Fatalf("a response spans lines: %s", out)
	}
	var r response
	dec := json.NewDecoder(bytes.NewReader(out))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&r); err != nil || r.Protocol != Protocol {
		t.Fatalf("response %s: %v", out, err)
	}
	return r
}

func digests(t *testing.T, h *Handler) Clock {
	t.Helper()
	var c Clock
	r := call(t, h, `{"id":"d","op":"digests"}`)
	if !r.OK || json.Unmarshal(r.Result, &c) != nil {
		t.Fatalf("digests: %+v", r)
	}
	return c
}

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// 1, 5: a complete 30-day session through the protocol, every line one
// valid JSON object in order with its id echoed; the same requests give
// byte-identical responses and digests; the saved session replays (6).
func TestThirtyDays(t *testing.T) {
	if testing.Short() {
		t.Skip("four 30-day runs")
	}
	dir := t.TempDir()
	run := func(p Policy, name string) (RunSummary, []byte) {
		_, h := open(t, "sustainable", 909)
		var tr bytes.Buffer
		c := &Client{H: h, Transcript: &tr}
		sum, err := Run(c, p, 30)
		if err != nil {
			t.Fatal(err)
		}
		c.Transcript = nil // the save names its own path
		if err := c.Call("save", map[string]any{"path": filepath.Join(dir, name)}, nil); err != nil {
			t.Fatal(err)
		}
		return sum, tr.Bytes()
	}
	for _, p := range []Policy{Noop{}, Greedy{}} {
		a, ta := run(p, p.Name()+"-a.json")
		b, tb := run(p, p.Name()+"-b.json")
		if !bytes.Equal(ta, tb) || a.Clock != b.Clock {
			t.Fatalf("%s: two identical runs differ", p.Name())
		}
		lines := strings.Split(strings.TrimSuffix(string(ta), "\n"), "\n")
		if len(lines)%2 != 0 {
			t.Fatalf("%s: unpaired transcript", p.Name())
		}
		for i := 0; i < len(lines); i += 2 {
			var req Request
			if err := json.Unmarshal([]byte(lines[i]), &req); err != nil {
				t.Fatal(err)
			}
			var r response
			if err := json.Unmarshal([]byte(lines[i+1]), &r); err != nil || !r.OK || string(r.ID) != string(req.ID) || r.Protocol != Protocol {
				t.Fatalf("%s line %d: %s → %s", p.Name(), i, lines[i], lines[i+1][:min(200, len(lines[i+1]))])
			}
		}
		if a.Clock.Day != 31 {
			t.Fatalf("%s: day %d", p.Name(), a.Clock.Day)
		}
		if p.Name() == "greedy" && a.Applied == 0 {
			t.Error("the greedy policy never decided: the run does not exercise apply")
		}
		s, rep, err := session.Load(filepath.Join(dir, p.Name()+"-a.json"), session.ReplayOptions{})
		if err != nil || !rep.OK || s.Digests() != a.Clock.Digests {
			t.Fatalf("%s: the saved session does not replay: %v", p.Name(), err)
		}
	}
}

// 2: every action type is in the catalog with a schema, and a valid
// action of every type, built from the catalog's targets alone, is
// applied.
func TestEveryActionType(t *testing.T) {
	_, h := open(t, "sustainable", 901)
	var cat struct {
		Types []struct {
			Type    string          `json:"type"`
			Schema  map[string]any  `json:"schema"`
			Targets json.RawMessage `json:"targets"`
		} `json:"types"`
	}
	r := call(t, h, `{"id":1,"op":"actions"}`)
	if !r.OK || json.Unmarshal(r.Result, &cat) != nil {
		t.Fatalf("%+v", r)
	}
	var types []string
	targets := map[string]json.RawMessage{}
	for _, x := range cat.Types {
		types = append(types, x.Type)
		targets[x.Type] = x.Targets
		if x.Schema["type"] != "object" || x.Schema["additionalProperties"] != false {
			t.Errorf("%s: schema %v", x.Type, x.Schema)
		}
	}
	want := []string{api.ActExtendShift, api.ActSetWalkInPolicy, api.ActSetWorklistPolicy, api.ActHire, api.ActTerminate,
		api.ActSetRoster, api.ActSetReadingAllocation, api.ActContractTeleradiology}
	if !slices.Equal(types, want) {
		t.Fatalf("types %v", types)
	}
	apply := func(a any) {
		t.Helper()
		var out Applied
		r := call(t, h, mustJSON(map[string]any{"id": "a", "op": "apply", "args": map[string]any{"actions": []any{a}}}))
		if !r.OK || json.Unmarshal(r.Result, &out) != nil || !out.Applied {
			t.Fatalf("%v: %s", a, r.Result)
		}
	}
	var ext []ExtendTarget
	json.Unmarshal(targets[api.ActExtendShift], &ext)
	i := slices.IndexFunc(ext, func(e ExtendTarget) bool { return e.ShiftToday != "" })
	if i < 0 {
		t.Fatal("no extension target with a shift today")
	}
	apply(map[string]any{"type": api.ActExtendShift, "employee": ext[i].Employee, "minutes": 30})
	apply(map[string]any{"type": api.ActSetWalkInPolicy, "policy": api.WalkInAcceptAlways})
	apply(map[string]any{"type": api.ActSetWorklistPolicy, "policy": api.WorklistFIFO})
	// hire: a technologist already working, copied: the offer with their
	// skills, at its standard rate and lead, on their roster (all from the
	// targets).
	var hire HireTargets
	json.Unmarshal(targets[api.ActHire], &hire)
	var rosters []api.RosterPlan
	json.Unmarshal(targets[api.ActSetRoster], &rosters)
	k := slices.IndexFunc(rosters, func(p api.RosterPlan) bool { return p.Role == "technologist" })
	if k < 0 {
		t.Fatal("no technologist's roster")
	}
	oi := slices.IndexFunc(hire.Offers, func(o OfferTarget) bool { return slices.Equal(o.Skills, rosters[k].Skills) })
	if oi < 0 {
		t.Fatalf("no offer for %v", rosters[k].Skills)
	}
	o := hire.Offers[oi]
	var cur api.RosterVersion
	for _, v := range rosters[k].Versions {
		if v.InEffect {
			cur = v
		}
	}
	first, _ := simtime.ParseDate(hire.EarliestStartDate)
	apply(map[string]any{"type": api.ActHire, "requestId": "agent-test-xr", "role": o.Role, "skills": o.Skills, "rateCents": o.StandardRateCents,
		"contractedWeeklyMinutes": max(cur.NetMinA, cur.NetMinB), "roster": cur.Roster, "startDate": first.AddDays(o.StandardLeadDays).String()})
	var term []TerminateTarget
	json.Unmarshal(targets[api.ActTerminate], &term)
	apply(map[string]any{"type": api.ActTerminate, "employee": term[0].Employee, "lastDay": term[0].EarliestLastDay})
	apply(map[string]any{"type": api.ActSetRoster, "op": "set", "employee": rosters[k].Employee, "effective": rosters[k].EarliestEffective, "roster": cur.Roster})
	var alloc []api.ReadingAllocation
	json.Unmarshal(targets[api.ActSetReadingAllocation], &alloc)
	apply(map[string]any{"type": api.ActSetReadingAllocation, "op": "set", "employee": alloc[0].Employee, "effective": alloc[0].EarliestEffective,
		"siteAllocationBP": alloc[0].InEffectBP + 250})
	var tele struct {
		Plans             []api.TelePlan `json:"plans"`
		EarliestEffective string         `json:"earliestEffective"`
	}
	json.Unmarshal(targets[api.ActContractTeleradiology], &tele)
	apply(map[string]any{"type": api.ActContractTeleradiology, "op": "start", "requestId": "agent-test-tele", "plan": tele.Plans[0].ID,
		"windows": []string{"saturday"}, "scope": "all", "effective": tele.EarliestEffective})
	// cancel: the new contract, now in the catalog's cancellable contracts.
	r = call(t, h, `{"id":2,"op":"actions"}`)
	json.Unmarshal(r.Result, &cat)
	var cancel struct {
		Cancellable []api.TeleContract `json:"cancellableContracts"`
	}
	for _, x := range cat.Types {
		if x.Type == api.ActContractTeleradiology {
			json.Unmarshal(x.Targets, &cancel)
		}
	}
	if len(cancel.Cancellable) != 1 {
		t.Fatalf("cancellable %v", cancel.Cancellable)
	}
	apply(map[string]any{"type": api.ActContractTeleradiology, "op": "cancel", "contract": cancel.Cancellable[0].ID, "endDate": cancel.Cancellable[0].EarliestEndDate})
}

// 3, 4: an invalid batch, validate and preview change nothing: not the
// state, not the action record, not a digest.
func TestNoSideEffects(t *testing.T) {
	_, h := open(t, "sustainable", 902)
	before := digests(t, h)
	for _, line := range []string{
		// an invalid allocation (out of band): refused, nothing applied
		`{"id":1,"op":"apply","args":{"actions":[{"type":"set_reading_allocation","op":"set","employee":"E07","effective":"2026-10-27","siteAllocationBP":100}]}}`,
		// a valid item before an invalid one: the whole batch refused
		`{"id":2,"op":"apply","args":{"actions":[{"type":"set_reading_allocation","op":"set","employee":"E08","effective":"2026-10-27","siteAllocationBP":5500},{"type":"terminate","employee":"E99","lastDay":"2026-11-30"}]}}`,
		// a mixed batch (V0.1 and V0.2)
		`{"id":3,"op":"apply","args":{"actions":[{"type":"set_worklist_policy","policy":"fifo"},{"type":"set_reading_allocation","op":"set","employee":"E08","effective":"2026-10-27","siteAllocationBP":5500}]}}`,
		// a V0.1 batch with one invalid action: nothing applied, not even the valid one
		`{"id":4,"op":"apply","args":{"actions":[{"type":"set_worklist_policy","policy":"fifo"},{"type":"set_walkin_policy","policy":"nonsense"}]}}`,
		`{"id":5,"op":"validate","args":{"actions":[{"type":"set_reading_allocation","op":"set","employee":"E08","effective":"2026-10-27","siteAllocationBP":5500}]}}`,
		`{"id":6,"op":"preview","args":{"actions":[{"type":"set_reading_allocation","op":"set","employee":"E08","effective":"2026-10-27","siteAllocationBP":5500}]}}`,
		`{"id":7,"op":"observe"}`,
		`{"id":8,"op":"actions"}`,
		`{"id":9,"op":"apply","args":{"actions":[{"type":"extend_shift","employee":"E03","minutes":30},{"type":"extend_shift","employee":"E03","minutes":30}]}}`,
		`{"id":10,"op":"apply","args":{"actions":[]}}`,
		`{"id":11,"op":"advance","args":{"to":"2026-10-12T06:00:00"}}`,
		`{"id":12,"op":"advance","args":{"minutes":30,"to":"2026-10-14T06:00:00"}}`,
		`{"id":13,"op":"save","args":{}}`,
		`{"id":14,"op":"preview","args":{"actions":[{"type":"set_worklist_policy","policy":"fifo"}]}}`,
		`{"id":15,"op":"apply","args":{"actions":[{"type":"hire","bogus":1}]}}`,
	} {
		r := call(t, h, line)
		var a Applied
		if r.OK && json.Unmarshal(r.Result, &a) == nil && a.Applied {
			t.Fatalf("applied: %s", line)
		}
		if after := digests(t, h); after != before {
			t.Fatalf("%s changed the session", line)
		}
	}
}

// Protocol errors: stable codes, the id echoed when it can be read, and
// the session usable after them.
func TestProtocolErrors(t *testing.T) {
	_, h := open(t, "mini", 903)
	for line, want := range map[string]string{
		`not json`:                             ErrInvalidJSON,
		`{"id":1,"op":"hello"} trailing`:       ErrInvalidJSON,
		`[1,2]`:                                ErrInvalidRequest,
		`{"id":2,"op":"hello","x":1}`:          ErrInvalidRequest,
		`{"id":{"a":1},"op":"hello"}`:          ErrInvalidRequest,
		`{"id":3}`:                             ErrInvalidRequest,
		`{"id":4,"op":"fly"}`:                  ErrUnknownOp,
		`{"id":5,"op":"hello","args":{"x":1}}`: ErrInvalidArgs,
		`{"id":6,"op":"observe","args":{"recentLog":-1}}`:    ErrInvalidArgs,
		`{"id":7,"op":"advance","args":{}}`:                  ErrInvalidArgs,
		`{"id":8,"op":"advance","args":{"next":"tomorrow"}}`: ErrInvalidArgs,
		`{"id":9,"op":"apply","args":{"actions":[{"type":"extend_shift","employee":"E03","minutes":30},{"type":"extend_shift","employee":"E03","minutes":20}]}}`: ErrDuplicate,
		`{"id":10,"op":"preview","args":{"actions":[{"type":"set_walkin_policy","policy":"accept_always"}]}}`:                                                    ErrNotPreviewable,
		`{"id":11,"op":"save","args":{"path":"/nonexistent-dir/x.json"}}`:                                                                                        ErrSaveFailed,
	} {
		r := call(t, h, line)
		if r.OK || r.Error == nil || r.Error.Code != want || r.Result != nil {
			t.Errorf("%s: %+v, want %s", line, r.Error, want)
		}
		if strings.HasPrefix(line, `{"id":2,`) && string(r.ID) != "2" {
			t.Errorf("%s: id %s not echoed", line, r.ID)
		}
	}
	if r := call(t, h, `{"id":"next","op":"hello"}`); !r.OK || string(r.ID) != `"next"` {
		t.Fatalf("after the errors: %+v", r)
	}
}

// 7, 8: the agent's transitions are the session API's, and what it shows
// is the session's own views: the observation byte for byte, and catalog
// targets naming only employees and contracts the observation shows.
func TestAgreesWithSession(t *testing.T) {
	a, h := open(t, "sustainable", 904)
	b, err := session.Open("sustainable", 904)
	if err != nil {
		t.Fatal(err)
	}
	act := api.Action{Type: api.ActSetReadingAllocation, Op: "set", Employee: "E08", Effective: "2026-10-27", SiteAllocationBP: 5500}
	if r := call(t, h, mustJSON(map[string]any{"id": 1, "op": "apply", "args": map[string]any{"actions": []any{act}}})); !r.OK {
		t.Fatal(r.Error)
	}
	b.Apply(act)
	call(t, h, `{"id":2,"op":"advance","args":{"minutes":2000}}`)
	b.Advance(2000*simtime.Minute, session.Options{})
	call(t, h, `{"id":3,"op":"advance","args":{"next":"visible"}}`)
	b.StepVisible()
	call(t, h, `{"id":4,"op":"advance","args":{"to":"2026-10-20T09:30:00"}}`)
	t2, _ := b.Calendar().ParseISO("2026-10-20T09:30:00")
	b.AdvanceTo(t2)
	call(t, h, `{"id":5,"op":"advance","args":{"next":"end_of_day"}}`)
	b.EndOfDay()
	if a.Digests() != b.Digests() || digests(t, h).Digests != b.Digests() {
		t.Fatal("the agent and the session API reached different states")
	}
	r := call(t, h, `{"id":6,"op":"observe","args":{"recentLog":50}}`)
	want := mustJSON(struct {
		Observation api.Observation `json:"observation"`
	}{b.Observe(50)})
	if string(r.Result) != want {
		t.Fatal("the agent's observation is not the session's")
	}
	var obs struct {
		Observation api.Observation `json:"observation"`
	}
	json.Unmarshal(r.Result, &obs)
	ids := map[string]bool{}
	for _, e := range obs.Observation.Staff {
		ids[e.ID] = true
	}
	cat := catalog(b)
	for _, x := range cat.Types {
		raw, _ := json.Marshal(x.Targets)
		var any []map[string]any
		if json.Unmarshal(raw, &any) != nil {
			continue
		}
		for _, t1 := range any {
			if e, ok := t1["employee"].(string); ok && !ids[e] {
				t.Errorf("%s target %s is not in the observation", x.Type, e)
			}
		}
	}
}
