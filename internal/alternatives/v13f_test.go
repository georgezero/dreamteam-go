package alternatives

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
)

func TestV13fCanonicalSubmitted(t *testing.T) {
	parse := func(raw string) [][]api.Action {
		var b [][]api.Action
		if err := json.Unmarshal([]byte(raw), &b); err != nil {
			t.Fatal(err)
		}
		return b
	}
	for _, pair := range [][2]string{
		{`[]`, `[[],[]]`},
		{`[[{"type":"set_walkin_policy","policy":"schedule_later"}]]`, ` [ [ {"policy":"schedule_later", "minutes":-0, "type":"set_walkin_policy"} ] ] `},
		{`[[{"type":"contract_teleradiology","op":"start","requestId":"x","plan":"standard-v1","windows":["sunday","weeknights"],"scope":"all","effective":"2026-10-27"}]]`, `[[{"windows":["weeknights","sunday"],"effective":"2026-10-27","scope":"all","plan":"standard-v1","requestId":"x","op":"start","type":"contract_teleradiology"}]]`},
	} {
		a, b := CanonicalSubmitted(parse(pair[0])), CanonicalSubmitted(parse(pair[1]))
		if a.Sha256 != b.Sha256 {
			t.Fatalf("equivalent submissions differ: %s %s", a.Canonical, b.Canonical)
		}
	}
	a := api.Action{Type: api.ActSetWalkInPolicy, Policy: "schedule_later"}
	b := api.Action{Type: api.ActSetWalkInPolicy, Policy: "accept_always"}
	c := CanonicalSubmitted([][]api.Action{{a, b}})
	for _, other := range [][][]api.Action{{{b, a}}, {{a}}, {{{Type: api.ActExtendShift, Employee: "E03", Minutes: 30}}}, {{{Type: api.ActExtendShift, Employee: "E04", Minutes: 30}}}} {
		if c.Sha256 == CanonicalSubmitted(other).Sha256 {
			t.Fatal("different target, count or ordered actions collapsed")
		}
	}
}
func TestV13fDifferentCountsAndTargets(t *testing.T) {
	a := api.Action{Type: api.ActExtendShift, Employee: "E03", Minutes: 30}
	for _, other := range []api.Action{{Type: api.ActExtendShift, Employee: "E03", Minutes: 31}, {Type: api.ActExtendShift, Employee: "E04", Minutes: 30}} {
		if CanonicalSubmitted([][]api.Action{{a}}).Sha256 == CanonicalSubmitted([][]api.Action{{other}}).Sha256 {
			t.Fatal("different count or target collapsed")
		}
	}
}
func TestV13fEmptyActionCounterexample(t *testing.T) {
	s := at(t, t0)
	p := []api.Action{term("E04", "2026-10-27")}
	c, err := DecideV2With(s, SessionValidator(s), ArmControl, p, nil, reply(ReplyAbandon, ""))
	if err != nil {
		t.Fatal(err)
	}
	tr, err := DecideV2With(s, SessionValidator(s), ArmTreatment, p, nil, reply(ReplySelect, IDNoAction))
	if err != nil {
		t.Fatal(err)
	}
	rev, err := DecideV2With(s, SessionValidator(s), ArmControl, p, nil, reply(ReplyRevise, ""))
	if err != nil {
		t.Fatal(err)
	}
	if c.SubmittedAction.Canonical != "[]" || c.SubmittedAction.Sha256 != tr.SubmittedAction.Sha256 || c.SubmittedAction.Sha256 != rev.SubmittedAction.Sha256 || !rev.Valid {
		t.Fatal("empty submissions differ")
	}
	if c.ResponseRoute == tr.ResponseRoute || c.Final == tr.Final || rev.ResponseRoute != "revise" {
		t.Fatal("secondary routes lost")
	}
	// Five observations in each arm have the same primary distribution.
	a, b := map[string]int{}, map[string]int{}
	for range 5 {
		a[c.SubmittedAction.Sha256]++
		b[tr.SubmittedAction.Sha256]++
	}
	if !jsonEqual(a, b) {
		t.Fatal("audit primary difference is nonzero")
	}
}
func TestV13fSystemSchemaPrecedence(t *testing.T) {
	s := at(t, t0)
	p := []api.Action{term("E04", "2026-10-27")}
	r := ParseTurnReplyV2(`{"reason":"system-compliant","actions":[]}`)
	if r.Kind != ReplyRevise {
		t.Fatal(r)
	}
	o, err := DecideV2With(s, SessionValidator(s), ArmControl, p, nil, r.Reply)
	if err != nil || !o.Valid || len(o.Batches) != 0 {
		t.Fatal(err, o)
	}
	for _, bad := range []string{`{"reason":"x","actions":[],"actions":[]}`, `{"decision":"revise"}`, `{"decision":"revise","actions":[],"id":"proposal"}`} {
		if ParseTurnReplyV2(bad).Kind != ReplyUnreadable {
			t.Fatal("bad schema accepted", bad)
		}
	}
}
func TestV13fContinuingCommitments(t *testing.T) {
	for _, acts := range [][]api.Action{
		{ctHire("E04", "2026-11-10")},
		{term("E03", "2026-10-27"), term("E04", "2026-10-27")},
		{term("E03", "2026-10-27"), {Type: api.ActSetRoster, Op: api.RosterOpSet, Employee: "E04", Effective: "2026-11-02", Roster: &api.Roster{}}},
	} {
		s := at(t, t0)
		ops, staff, pos := splitPositions(acts)
		sel, err := SelectV2(s, staff)
		if err != nil {
			t.Fatal(err)
		}
		a := mustAgent(t, s, sel)
		v, _, err := BuildViewV2(s, a, pos, len(ops))
		if err != nil {
			t.Fatal(err)
		}
		for _, alt := range v.Alternatives {
			if !alt.Accepted {
				continue
			}
			seen := map[string]bool{}
			for _, c := range alt.Commitments {
				if seen[c.Ref] {
					t.Fatal("commitment counted twice")
				}
				seen[c.Ref] = true
				if strings.HasPrefix(c.Ref, "H") || strings.HasPrefix(c.Ref, "C") || strings.HasPrefix(c.Ref, "P") {
					t.Fatal("employee reference crossed a namespace", c.Ref)
				}
				if c.Ref == "E1" || c.Ref == "E2" {
					if c.BaselineGuarantee == nil || c.ContinuingGuarantee == nil {
						t.Fatal("known guarantee missing", alt.ID, c)
					}
				}
				if !c.Terminating && c.BaselineExposure != nil && !jsonEqual(c.BaselineExposure, c.ConditionalExposure) {
					t.Fatal("retained exposure omitted")
				}
			}
		}
	}
}
func TestV13fCapacityWithdrawalProxy(t *testing.T) {
	s := at(t, t0)
	p := []api.Action{term("E04", "2026-10-27")}
	c, tr, err := TurnsV2(s, p)
	if err != nil {
		t.Fatal(err)
	}
	if !tr.Due || !strings.Contains(tr.Inspect[IDProposal], "2,481 slots") || !strings.Contains(tr.Inspect[IDProposal], "0 bookings") {
		t.Fatal("capacity witness missing", tr.Why)
	}
	foot := "Reply with one JSON object (the shared system schema governs this phase):"
	if c.Message[strings.Index(c.Message, foot):] != tr.Message[strings.Index(tr.Message, foot):] {
		t.Fatal("feet differ")
	}
	s = at(t, "2026-10-27T07:00:00")
	roster := api.Action{Type: api.ActSetRoster, Op: api.RosterOpSet, Employee: "E04", Effective: "2026-11-16", Roster: &api.Roster{}}
	_, tr, err = TurnsV2(s, []api.Action{roster})
	if err != nil {
		t.Fatal(err)
	}
	if !tr.Due || !strings.Contains(tr.Inspect[IDProposal], "2,354 slots") || !strings.Contains(tr.Inspect[IDProposal], "2026-11-02 (inclusive)") {
		t.Fatal("roster witness missing", tr.Why, tr.Record.ViewError)
	}
	if rs := s.ValidateBatch(roster); len(rs) != 1 || !rs[0].OK {
		t.Fatal(rs)
	}
	s.Apply(roster)
	atTime, _ := s.Calendar().ParseISO("2026-11-03T07:00:00")
	s.AdvanceTo(atTime)
	roster.Op = api.RosterOpRemove
	roster.Roster = nil
	if rs := s.ValidateBatch(roster); len(rs) != 1 || rs[0].OK || rs[0].Code != "notice_too_short" {
		t.Fatal("withdrawal incorrectly allowed", rs)
	}
	s = at(t, t0)
	_, tr, err = TurnsV2(s, []api.Action{p[0], tele("weeknights")})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(tr.Inspect[IDProposal], "Slots closed: unavailable") {
		t.Fatal("unsupported mixed capacity not disclosed")
	}
	_, tr, err = TurnsV2(s, []api.Action{tele("weeknights")})
	if err != nil {
		t.Fatal(err)
	}
	d := tr.Inspect[IDProposal]
	for _, want := range []string{"0 relevant studies", "Incomplete lookback: true", "Two lanes", "not a guaranteed future fee", "not a utilisation bound", "2026-10-13"} {
		if !strings.Contains(d, want) {
			t.Fatal("proxy context missing", want)
		}
	}
}
func TestV13fTypedNumericCitations(t *testing.T) {
	s := at(t, t0)
	p := []api.Action{term("E04", "2026-10-27")}
	for _, val := range []string{`2481`, `2481.0`, `"2481"`, `"2,481"`, `{"value":2481,"unit":"slots"}`} {
		r := reply(ReplySelect, IDNoAction)
		r.Comparison = []Citation{{Item: "slots_closed", Values: map[string]json.RawMessage{IDProposal: json.RawMessage(val), IDNoAction: json.RawMessage(`0`)}}}
		o, err := DecideV2With(s, SessionValidator(s), ArmTreatment, p, []string{IDProposal, IDNoAction}, r)
		if err != nil || !slices.Contains(o.Flags, FlagCompared) {
			t.Fatal("equivalent citation refused", val, err, o.CitationChecks)
		}
	}
	for _, val := range []string{`"24,81"`, `{"value":2481,"unit":"minutes"}`, `"$2,481.00"`, `2480`} {
		if citationEqualV2(json.RawMessage(val), citationValueV2{Text: "2,481", Number: num(2481), Unit: "slots"}) {
			t.Fatal("wrong units/value accepted", val)
		}
	}
	for _, raw := range []string{`{"value":2481,"unit":"cents"}`, `{"value":24.81,"unit":"dollars"}`} {
		if !citationEqualV2(json.RawMessage(raw), citationValueV2{Text: "$24.81", Number: num(2481), Unit: "cents"}) {
			t.Fatal("explicit currency representation rejected", raw)
		}
	}
	for _, raw := range []string{`{"value":2481,"unit":"dollars"}`, `{"value":24.81,"unit":"cents"}`, `{"value":"$24.81","unit":"cents"}`} {
		if citationEqualV2(json.RawMessage(raw), citationValueV2{Text: "$24.81", Number: num(2481), Unit: "cents"}) {
			t.Fatal("currency units conflated", raw)
		}
	}
	if citationEqualV2(json.RawMessage(`2481`), citationValueV2{Text: "$24.81", Number: num(2481), Unit: "cents"}) {
		t.Fatal("cents and bare numbers conflated")
	}
}
func TestEvidenceV13f(t *testing.T) {
	write := func(name string, body []byte) {
		t.Helper()
		p := filepath.Join("../../docs/benchmark/v13f-dev", name)
		if os.Getenv("DREAMTEAM_V13F_WRITE") == "1" {
			if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, body, 0644); err != nil {
				t.Fatal(err)
			}
		} else {
			want, err := os.ReadFile(p)
			if err != nil || !bytes.Equal(want, body) {
				t.Fatal("successor evidence differs", name, err)
			}
		}
	}
	write("presentation/view-table.txt", []byte(ViewV2TableText()))
	write("presentation/presentation-v3.tmpl", presentationV3Text)
	for _, pair := range sensitivityPairs {
		for i, m := range []member{pair.A, pair.B} {
			s := at(t, m.At)
			c, tr, err := TurnsV2(s, m.Proposal)
			if err != nil {
				t.Fatal(err)
			}
			raw, _ := json.MarshalIndent(map[string]any{"control": c, "treatment": tr}, "", " ")
			write("turns/"+pair.ID+"-"+[]string{"a", "b"}[i]+".json", append(raw, '\n'))
		}
	}
}
func TestV13fPinnedTexts(t *testing.T) {
	if sha256Hex(presentationV3Text) != PresentationV3Sha256 {
		t.Fatal("presentation v3 pin differs", sha256Hex(presentationV3Text))
	}
	if sha256Hex([]byte(ViewV2TableText())) != ViewV2TableSha256 {
		t.Fatal("view v2 pin differs", sha256Hex([]byte(ViewV2TableText())))
	}
}

// The historical path is byte-identical to the pre-remediation baseline.
// These witnesses reproduce its defects without replacing old evidence.
func TestV13fBeforeFixWitnesses(t *testing.T) {
	s := at(t, t0)
	p := []api.Action{term("E04", "2026-10-27")}
	c := mustDecide(t, s, ArmControl, p, nil, reply(ReplyAbandon, ""))
	tr := mustDecide(t, s, ArmTreatment, p, nil, reply(ReplySelect, IDNoAction))
	rev := mustDecide(t, s, ArmControl, p, nil, reply(ReplyRevise, ""))
	if !jsonEqual(c.Batches, tr.Batches) || c.Final == tr.Final || rev.Final != FinalNoDecision {
		t.Fatal("historical counterexample no longer reproduces")
	}
	t.Log("DT13E-01 before fix: five abandonments versus five no_action choices; identical empty batches; legacy route TV 1, action TV 0; empty revision classified no_decision")
	if ParseTurnReply(`{"reason":"system compliant","actions":[]}`).Kind != ReplyUnreadable {
		t.Fatal("historical schema witness changed")
	}
	t.Log("DT13E-02 before fix: system-prescribed reason/actions rejected")
	if citedEquals(json.RawMessage(`2481`), "2,481") {
		t.Fatal("historical citation witness changed")
	}
	t.Log("DT13E-09 before fix: numeric 2481 rejected for displayed 2,481")
	p = append([]api.Action{term("E03", "2026-10-27")}, p...)
	_, old := mustTurns(t, s, p...)
	if strings.Contains(old.Inspect["without_1"], "[weekly_guarantee.E1]") {
		t.Fatal("historical obligation witness changed")
	}
	t.Log("DT13E-03 before fix: leave-one-out omits retained employee guarantee")
}

func TestV13fCanonicalEngineSets(t *testing.T) {
	s := at(t, t0)
	a := ctHire("canonical-hire", "2026-11-10")
	a.Skills = []string{"CT", "XR"}
	a.RateCents = 3800
	for _, week := range []*api.Week{&a.Roster.A, &a.Roster.B} {
		for _, day := range week.Days() {
			if day != nil {
				day.Covers = []api.Cover{{Room: "R1", From: "14:00", Until: "15:00"}, {Room: "R1", From: "12:00", Until: "13:00"}}
			}
		}
	}
	b := canon([]api.Action{a})[0]
	slices.Reverse(b.Skills)
	for _, week := range []*api.Week{&b.Roster.A, &b.Roster.B} {
		for _, day := range week.Days() {
			if day != nil {
				slices.Reverse(day.Covers)
			}
		}
	}
	for _, act := range []api.Action{a, b} {
		if res := s.ValidateBatch(act); len(res) != 1 || !res[0].OK {
			t.Fatal("engine rejects equivalence witness", res)
		}
	}
	if CanonicalSubmitted([][]api.Action{{a}}).Sha256 != CanonicalSubmitted([][]api.Action{{b}}).Sha256 {
		t.Fatal("engine-equivalent sets have different identities")
	}
	if res := s.Apply(a); !res[0].OK {
		t.Fatal(res)
	}
	if res := s.ValidateBatch(b); len(res) != 1 || !res[0].OK || res[0].Code != api.ActionUnchanged {
		t.Fatal("engine canonical idempotency disagrees", res)
	}
}

func TestV13fCapacityHorizonBounds(t *testing.T) {
	s := at(t, t0)
	act := ctHire("late-hire", "2027-02-09")
	p := s.PreviewStaffing(act)
	if !p.OK || len(p.Hires) != 1 {
		t.Fatal(p.Results)
	}
	q := capacityV2(s, s.Observe(0), []api.Action{act}, p).Opened
	if q.Number == nil || *q.Number != 0 || q.From > q.To {
		t.Fatal("capacity horizon is not a valid current-horizon interval", q)
	}
}

func TestV13fCanonicalEngineDefaults(t *testing.T) {
	s := at(t, t0)
	today := s.Observe(0).Date
	a := api.Action{Type: api.ActExtendShift, Employee: "E03", Minutes: 30}
	b := a
	b.Date = today
	for _, act := range []api.Action{a, b} {
		if r := s.ValidateBatch(act); len(r) != 1 || !r[0].OK {
			t.Fatal(r)
		}
	}
	if CanonicalSubmitted([][]api.Action{{a}}, today).Sha256 != CanonicalSubmitted([][]api.Action{{b}}, today).Sha256 {
		t.Fatal("omitted date differs from the engine's current-day default")
	}
	b.Date = shiftDateV2(today, 1)
	if CanonicalSubmitted([][]api.Action{{a}}, today).Sha256 == CanonicalSubmitted([][]api.Action{{b}}, today).Sha256 {
		t.Fatal("different target dates collapsed")
	}
}
