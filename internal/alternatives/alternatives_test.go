package alternatives

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/session"
)

// V0.13b: the candidate rule, the agent and evaluation audiences, the
// template and the classification, on real engine states (mini, seed 52).

func at(t testing.TB, when string) *session.Session {
	t.Helper()
	s, err := session.Open("mini", 52)
	if err != nil {
		t.Fatal(err)
	}
	x, err := s.Calendar().ParseISO(when)
	if err != nil {
		t.Fatal(err)
	}
	s.AdvanceTo(x)
	return s
}

func term(emp, last string) api.Action {
	return api.Action{Type: api.ActTerminate, Employee: emp, LastDay: last}
}

const t0 = "2026-10-13T07:00:00"

func ids(sel *Selection) []string {
	var out []string
	for _, c := range sel.Candidates {
		out = append(out, c.ID)
	}
	return out
}

func mustSelect(t testing.TB, s *session.Session, p ...api.Action) *Selection {
	t.Helper()
	sel, err := Select(s, p)
	if err != nil {
		t.Fatal(err)
	}
	return sel
}

func mustAgent(t testing.TB, s *session.Session, sel *Selection) *Agent {
	t.Helper()
	a, err := BuildAgent(s, sel)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestPinnedTexts(t *testing.T) {
	if sha256Hex(selectionText) != SelectionRuleSha256 {
		t.Fatalf("selection rule text changed: %s", sha256Hex(selectionText))
	}
	if sha256Hex(presentationText) != PresentationSha256 {
		t.Fatalf("presentation template changed: %s", sha256Hex(presentationText))
	}
}

func TestSelectionTermination(t *testing.T) {
	s := at(t, t0)
	before := s.Digests()
	sel := mustSelect(t, s, term("E04", "2026-10-27"))
	if !sel.Applies || !slices.Equal(ids(sel), []string{IDProposal, IDNoAction, IDAdjusted}) {
		t.Fatalf("%v %s", ids(sel), sel.Why)
	}
	if sel.Candidates[2].Actions[0].LastDay != "2026-11-10" || !sel.Candidates[2].OK {
		t.Fatalf("adjusted %+v", sel.Candidates[2])
	}
	if s.Digests() != before {
		t.Fatal("selection changed the session")
	}
	// Deterministic and replayable; an edited selection does not replay.
	if again := mustSelect(t, s, term("E04", "2026-10-27")); !jsonEqual(again, sel) {
		t.Fatal("selection is not deterministic")
	}
	if err := VerifySelection(s, sel); err != nil {
		t.Fatal(err)
	}
	edited := *sel
	edited.Candidates = slices.Clone(sel.Candidates)
	edited.Candidates[1], edited.Candidates[2] = edited.Candidates[2], edited.Candidates[1]
	if VerifySelection(s, &edited) == nil {
		t.Fatal("a reordered selection replayed")
	}
	// Candidate identities are stable and distinct.
	seen := map[string]bool{}
	for _, c := range sel.Candidates {
		if seen[c.Sha256] || c.Sha256 == "" {
			t.Fatal("candidate identities repeat")
		}
		seen[c.Sha256] = true
	}
}

func TestSelectionRules(t *testing.T) {
	s := at(t, t0)
	// Leave one out: two terminations; at most four candidates.
	sel := mustSelect(t, s, term("E03", "2026-10-27"), term("E04", "2026-10-27"))
	if !slices.Equal(ids(sel), []string{IDProposal, IDNoAction, "without_1", "without_2"}) {
		t.Fatalf("%v", ids(sel))
	}
	if !jsonEqual(sel.Candidates[2].Actions, []api.Action{term("E04", "2026-10-27")}) {
		t.Fatalf("without_1 %+v", sel.Candidates[2].Actions)
	}
	sel = mustSelect(t, s, term("E03", "2026-10-27"), term("E04", "2026-10-27"), term("E05", "2026-10-27"))
	if len(sel.Candidates) != MaxCandidates || !slices.Equal(ids(sel), []string{IDProposal, IDNoAction, "without_1", "without_2"}) {
		t.Fatalf("limit: %v", ids(sel))
	}
	// An operations batch: the rule does not apply.
	sel = mustSelect(t, s, api.Action{Type: api.ActSetWorklistPolicy, Policy: "fifo"})
	if sel.Applies || len(sel.Candidates) != 0 {
		t.Fatalf("operations: %v", ids(sel))
	}
	// An invalid proposal is listed refused, never repaired; its adjustment
	// is omitted when the engine refuses it too.
	sel = mustSelect(t, s, term("E04", "2026-10-20"))
	if sel.Candidates[0].OK || sel.Candidates[0].Validation[0].Code != "notice_too_short" || !jsonEqual(sel.Candidates[0].Actions, []api.Action{term("E04", "2026-10-20")}) {
		t.Fatalf("invalid proposal %+v", sel.Candidates[0])
	}
	if !slices.Equal(ids(sel), []string{IDProposal, IDNoAction, IDAdjusted}) {
		t.Fatalf("%v", ids(sel)) // Nov 3: 21 days' notice, valid
	}
	sel = mustSelect(t, s, term("E02", "2026-10-27"))
	if sel.Candidates[0].OK || len(sel.Omitted) != 1 || len(sel.Candidates) != 2 {
		t.Fatalf("refused adjustment: %v %+v", ids(sel), sel.Omitted)
	}
	// No adjustment past the booking horizon (Feb 10).
	sel = mustSelect(t, s, term("E04", "2027-02-01"))
	if !slices.Equal(ids(sel), []string{IDProposal, IDNoAction}) || len(sel.Omitted) != 0 {
		t.Fatalf("horizon: %v", ids(sel))
	}
	// A leave-one-out candidate the engine refuses stays listed.
	sel = mustSelect(t, s, term("E04", "2026-10-27"), term("E04", "2026-11-10"))
	if len(sel.Candidates) != 4 || sel.Candidates[0].OK {
		t.Fatalf("%v", ids(sel))
	}
}

func TestAgentTerminationVersusKeep(t *testing.T) {
	s := at(t, t0)
	before := s.Digests()
	a := mustAgent(t, s, mustSelect(t, s, term("E04", "2026-10-27")))
	if s.Digests() != before {
		t.Fatal("building the comparison changed the session")
	}
	get := func(id, key string) Value {
		v, ok := shownValue(a, id, key)
		if !ok {
			t.Fatalf("%s: no %s", id, key)
		}
		return v
	}
	if v := get(IDProposal, "severance_recognized_now.E04"); *v.Number != 395000 || v.When != "recognized now, paid 2026-10-30" {
		t.Fatalf("%+v", v)
	}
	if get(IDProposal, "last_day.E04").Date != "2026-10-27" || *get(IDProposal, "notice_days.E04").Number != 14 ||
		*get(IDProposal, "displaced_bookings").Number != 0 || *get(IDProposal, "slots_closed").Number != 2481 {
		t.Fatal("termination facts")
	}
	// Keeping is not shown as cost-free: the guarantee and severance
	// exposure continue, with the current employment and roster.
	if v := get(IDNoAction, "weekly_guarantee.E04"); *v.Number != 197500 || v.Class != ClassDerived {
		t.Fatalf("%+v", v)
	}
	if *get(IDNoAction, "severance_if_terminated_later.E04").Number != 395000 || get(IDNoAction, "employment.E04").Text != "active" ||
		*get(IDNoAction, "roster_minutes_week_a.E04").Number <= 0 || *get(IDNoAction, "contract_minutes.E04").Number != 2370 {
		t.Fatal("keep facts")
	}
	for _, k := range []string{"next_payroll", "payables"} {
		get(IDNoAction, k)
	}
	if len(a.Alternatives[1].Obligations) == 0 {
		t.Fatal("no obligations under no action")
	}
	// Compared: only keys every accepted alternative reports.
	keys := map[string]bool{}
	for _, c := range a.Compared {
		keys[c.Key] = true
		if len(c.Values) != 3 {
			t.Fatalf("%s: %v", c.Key, c.Values)
		}
	}
	if !keys["severance_recognized_now.E04"] || !keys["displaced_bookings"] || !keys["slots_closed"] || keys["weekly_guarantee.E04"] {
		t.Fatalf("compared %v", keys)
	}
}

// The agent audience never holds future information: class B, a
// projection, a continuation or a horizon cannot be decoded into it, and
// its bytes do not depend on any continuation.
func TestAgentAudienceBoundary(t *testing.T) {
	s := at(t, t0)
	sel := mustSelect(t, s, term("E04", "2026-10-27"))
	a := mustAgent(t, s, sel)
	raw, _ := a.Encode()
	if _, err := DecodeAgent(raw); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"projected", "continuation", "horizon", "branches", "seed", "differences"} {
		if bytes.Contains(raw, []byte(`"`+k+`"`)) {
			t.Fatalf("agent bytes hold %q", k)
		}
	}
	inject := func(edit func(m map[string]any)) []byte {
		var m map[string]any
		json.Unmarshal(raw, &m)
		edit(m)
		b, _ := json.Marshal(m)
		return b
	}
	rehashed := func(b []byte) []byte {
		var x Agent
		json.Unmarshal(b, &x)
		x.Sha256 = x.digest()
		out, _ := json.Marshal(&x)
		return out
	}
	classB := inject(func(m map[string]any) {
		m["alternatives"].([]any)[0].(map[string]any)["facts"].([]any)[0].(map[string]any)["class"] = ClassProjected
	})
	if _, err := DecodeAgent(rehashed(classB)); !errors.Is(err, ErrContract) {
		t.Errorf("class B with a matching digest: %v", err)
	}
	for name, edit := range map[string]func(m map[string]any){
		"projected field": func(m map[string]any) {
			m["alternatives"].([]any)[0].(map[string]any)["projected"] = []any{}
		},
		"continuation": func(m map[string]any) { m["continuation"] = map[string]any{"id": "x"} },
		"class B fact": func(m map[string]any) {
			f := m["alternatives"].([]any)[0].(map[string]any)["facts"].([]any)[0].(map[string]any)
			f["class"] = ClassProjected
		},
		"seed": func(m map[string]any) { m["decisionPoint"].(map[string]any)["seed"] = 52 },
	} {
		if _, err := DecodeAgent(inject(edit)); !errors.Is(err, ErrContract) {
			t.Errorf("%s: %v", name, err)
		}
	}
	// A projection relabelled as an engine fact, with a branch source, is
	// refused even with the compared table and digest recomputed; with a
	// decision-point source it decodes but does not bind to the session.
	smuggle := func(src string) []byte {
		var x Agent
		json.Unmarshal(raw, &x)
		x.Alternatives = slices.Clone(x.Alternatives)
		for i := range x.Alternatives {
			if x.Alternatives[i].Validation.OK {
				x.Alternatives[i].Facts = append(slices.Clone(x.Alternatives[i].Facts), Value{Key: "cash_at_horizon", Label: "Cash at the horizon",
					Class: ClassReported, Unit: UnitCents, Number: num(1000000 + int64(i)), When: "2026-12-11", Source: src})
			}
		}
		x.Compared = compare(x.Alternatives)
		x.Sha256 = x.digest()
		b, _ := json.Marshal(&x)
		return b
	}
	for _, src := range []string{"end.balances.cashCents", "branch.proposal", "window.money.netRevenueCents"} {
		if _, err := DecodeAgent(smuggle(src)); !errors.Is(err, ErrContract) {
			t.Errorf("smuggled projection with source %s: %v", src, err)
		}
	}
	disguised := smuggle("observe.finance.cashCents")
	if _, err := DecodeAgent(disguised); err != nil {
		t.Fatalf("structural decode: %v", err)
	}
	if _, err := DecodeAgentAt(disguised, s, sel); !errors.Is(err, ErrContract) {
		t.Fatalf("a disguised projection bound to the session: %v", err)
	}
	if _, err := DecodeAgentAt(raw, s, sel); err != nil {
		t.Fatal(err)
	}
	// The evaluation audience holds the agent's bytes and projections.
	r, _ := s.Record()
	e, err := BuildEvaluation(r, a, session.BranchOptions{Continuation: session.NoFurtherAction()})
	if err != nil {
		t.Fatal(err)
	}
	if !jsonEqual(e.Agent, a) || len(e.Branches) != 3 || len(e.Branches[0].Projected) == 0 || len(e.Differences) == 0 {
		t.Fatal("evaluation audience")
	}
	again := mustAgent(t, s, sel)
	if !jsonEqual(again, a) {
		t.Fatal("the agent comparison changed after branching")
	}
}

// Required fields, contradictions and edits are refused.
func TestAgentRefusals(t *testing.T) {
	s := at(t, t0)
	sel := mustSelect(t, s, term("E04", "2026-10-27"))
	a := mustAgent(t, s, sel)
	raw, _ := a.Encode()
	edit := func(f func(m map[string]any), rehash bool) []byte {
		var m map[string]any
		json.Unmarshal(raw, &m)
		f(m)
		b, _ := json.Marshal(m)
		if rehash {
			var x Agent
			json.Unmarshal(b, &x)
			x.Sha256 = x.digest()
			b, _ = json.Marshal(&x)
		}
		return b
	}
	alt := func(m map[string]any, i int) map[string]any { return m["alternatives"].([]any)[i].(map[string]any) }
	fact := func(m map[string]any, i, j int) map[string]any { return alt(m, i)["facts"].([]any)[j].(map[string]any) }
	cases := map[string]func(m map[string]any){
		"missing source":       func(m map[string]any) { delete(fact(m, 0, 0), "source") },
		"missing derivation":   func(m map[string]any) { delete(fact(m, 1, 0), "derivation") },
		"value edited":         func(m map[string]any) { fact(m, 0, 3)["number"] = 1 },
		"no_action not 2nd":    func(m map[string]any) { a := m["alternatives"].([]any); a[1], a[2] = a[2], a[1] },
		"validation contra":    func(m map[string]any) { alt(m, 0)["validation"].(map[string]any)["ok"] = false },
		"unknown action":       func(m map[string]any) { alt(m, 0)["actions"].([]any)[0].(map[string]any)["type"] = "rehire" },
		"ranking word":         func(m map[string]any) { fact(m, 0, 0)["label"] = "best last day" },
		"compared not derived": func(m map[string]any) { m["compared"] = []any{} },
		"no audience":          func(m map[string]any) { m["audience"] = "evaluation" },
		"missing facts":        func(m map[string]any) { delete(alt(m, 0), "facts") },
	}
	for name, f := range cases {
		for _, rehash := range []bool{false, true} {
			if _, err := DecodeAgent(edit(f, rehash)); !errors.Is(err, ErrContract) {
				t.Errorf("%s (rehash %v): %v", name, rehash, err)
			}
		}
	}
	// A tampered comparison does not rebuild from the session.
	x := *a
	x.Alternatives = slices.Clone(a.Alternatives)
	x.Alternatives[0].Description = "Terminate someone else"
	x.Sha256 = x.digest()
	if err := VerifyAgent(s, sel, &x); err == nil {
		t.Fatal("a tampered comparison verified")
	}
	if err := VerifyAgent(s, sel, a); err != nil {
		t.Fatal(err)
	}
	// An invalid preview (a refused proposal) shows only its validation.
	b := mustAgent(t, s, mustSelect(t, s, term("E04", "2026-10-20")))
	if b.Alternatives[0].Validation.OK || len(b.Alternatives[0].Facts) != 0 {
		t.Fatal("a refused proposal shows values")
	}
	// A selection that lists nothing is not a comparison.
	if _, err := BuildAgent(s, mustSelect(t, s, api.Action{Type: api.ActSetWorklistPolicy, Policy: "fifo"})); !errors.Is(err, ErrContract) {
		t.Fatal(err)
	}
}

// Every number the template shows comes from an identified field of the
// agent comparison; the template recommends nothing.
func TestPresentationFidelity(t *testing.T) {
	s := at(t, t0)
	a := mustAgent(t, s, mustSelect(t, s, term("E04", "2026-10-27"), term("E03", "2026-11-10")))
	texts := []string{}
	sum, err := Summary(a)
	if err != nil {
		t.Fatal(err)
	}
	texts = append(texts, sum)
	for _, x := range a.Alternatives {
		d, err := Detail(a, x.ID)
		if err != nil {
			t.Fatal(err)
		}
		texts = append(texts, d)
	}
	// The template's own text holds no number: every number it shows is
	// a field.
	body := regexp.MustCompile(`(?s)\{\{-? */\*.*?\*/ *-?\}\}`).ReplaceAllString(string(presentationText), "")
	if d := regexp.MustCompile(`[0-9]`).FindString(body); d != "" {
		t.Fatalf("the template text holds a digit %q", d)
	}
	// Every number token shown is a whole number token of a comparison
	// field (not merely a substring of one).
	numRe := regexp.MustCompile(`[-$]?[0-9][0-9,.:\-T]*`)
	tokens := func(s string) []string {
		var out []string
		for _, t := range numRe.FindAllString(s, -1) {
			out = append(out, strings.TrimRight(t, ".,:-"))
		}
		return out
	}
	allowed := map[string]bool{}
	for _, s := range allowedStrings(a) {
		for _, t := range tokens(s) {
			allowed[t] = true
		}
	}
	for _, text := range texts {
		for _, line := range strings.Split(text, "\n") {
			for _, tok := range tokens(line) {
				if !allowed[tok] {
					t.Errorf("number %q in %q comes from no comparison field", tok, line)
				}
			}
		}
		low := strings.ToLower(text)
		for _, w := range []string{"recommend", "best", "better", "worse", "optimal", "safe", "risky", "costly", "saving", "should"} {
			if strings.Contains(low, w) {
				t.Errorf("the presentation says %q", w)
			}
		}
	}
	if _, err := Detail(a, "nothing"); err == nil {
		t.Fatal("detail of an unlisted alternative")
	}
}

// allowedStrings are every string the comparison holds (values as
// formatted, sources' texts, actions as JSON) and the structural counts.
func allowedStrings(a *Agent) []string {
	out := []string{a.DecisionPoint.Time}
	for i := range a.Alternatives {
		out = append(out, string(rune('1'+i)))
	}
	out = append(out, string(rune('0'+len(a.Alternatives))))
	for _, x := range a.Alternatives {
		raw, _ := json.Marshal(x.Actions)
		out = append(out, string(raw), x.Description, x.Reversibility.Text, x.Reversibility.Basis, x.ID,
			string(rune('0'+len(x.Facts)%10)), string(rune('0'+len(x.Estimates)%10)), string(rune('0'+len(x.Obligations)%10)), string(rune('0'+len(x.Unavailable)%10)))
		for _, n := range []int{len(x.Facts), len(x.Estimates), len(x.Obligations), len(x.Unavailable)} {
			out = append(out, strings.TrimSpace(strings.Repeat(" ", 0)+itoa(n)))
		}
		for _, r := range x.Validation.Results {
			out = append(out, r.Message, r.Code)
		}
		for _, list := range [][]Value{x.Facts, x.Estimates, x.Obligations} {
			for _, v := range list {
				out = append(out, formatValue(v), v.When, v.Basis, v.Label)
			}
		}
		for _, u := range x.Unavailable {
			out = append(out, u.Key, u.Reason)
		}
	}
	return out
}

func itoa(n int) string { b, _ := json.Marshal(n); return string(b) }

func TestClassify(t *testing.T) {
	s := at(t, t0)
	a := mustAgent(t, s, mustSelect(t, s, term("E04", "2026-10-27")))
	rv := func(b []api.Action) []api.ActionResult { return s.ValidateBatch(b...) }
	cite := func(item string, vals map[string]any) Citation {
		c := Citation{Item: item, Values: map[string]json.RawMessage{}}
		for k, v := range vals {
			c.Values[k], _ = json.Marshal(v)
		}
		return c
	}
	good := []Citation{cite("severance_recognized_now.E04", map[string]any{IDProposal: 395000, IDNoAction: 0})}
	for _, c := range []struct {
		name   string
		reqs   []string
		reply  *Reply
		result string
		flags  []string
	}{
		{"retained", nil, &Reply{Kind: ReplySelect, ID: IDProposal}, ResultRetained, []string{FlagNoComparison}},
		{"inspected and retained", []string{"inspect:adjusted"}, &Reply{Kind: ReplySelect, ID: IDProposal}, ResultRetained, []string{FlagInspected}},
		{"inspect proposal only", []string{"inspect:proposal"}, &Reply{Kind: ReplySelect, ID: IDProposal}, ResultRetained, []string{FlagNoComparison}},
		{"selected", nil, &Reply{Kind: ReplySelect, ID: IDAdjusted, Comparison: good}, ResultSelected, []string{FlagCompared}},
		{"no action", []string{"inspect:no_action"}, &Reply{Kind: ReplySelect, ID: IDNoAction, Comparison: good}, ResultSelectedNoAction, []string{FlagInspected, FlagCompared}},
		{"revised", nil, &Reply{Kind: ReplyRevise, Actions: []api.Action{term("E04", "2026-11-03")}}, ResultRevised, []string{FlagNoComparison}},
		{"revise to listed", nil, &Reply{Kind: ReplyRevise, Actions: []api.Action{term("E04", "2026-11-10")}}, ResultSelected, []string{FlagNoComparison}},
		{"abandoned", nil, &Reply{Kind: ReplyAbandon}, ResultAbandoned, []string{FlagNoComparison}},
		{"unlisted", nil, &Reply{Kind: ReplySelect, ID: "without_1"}, ResultInvalidAlterative, []string{FlagNoComparison}},
		{"revise refused", nil, &Reply{Kind: ReplyRevise, Actions: []api.Action{term("E04", "2026-10-14")}}, ResultInvalidAlterative, []string{FlagNoComparison}},
		{"no reply", nil, nil, ResultNoDecision, []string{FlagNoComparison}},
		{"contradictory", nil, &Reply{Kind: ReplySelect, ID: IDProposal, Actions: []api.Action{term("E04", "2026-11-03")}}, ResultNoDecision, []string{FlagNoComparison}},
		{"unknown kind", nil, &Reply{Kind: "approve"}, ResultNoDecision, []string{FlagNoComparison}},
		{"wrong value", nil, &Reply{Kind: ReplySelect, ID: IDProposal, Comparison: []Citation{cite("severance_recognized_now.E04", map[string]any{IDProposal: 395000, IDNoAction: 1})}}, ResultRetained, []string{FlagNoComparison}},
		{"only chosen", nil, &Reply{Kind: ReplySelect, ID: IDProposal, Comparison: []Citation{cite("severance_recognized_now.E04", map[string]any{IDProposal: 395000})}}, ResultRetained, []string{FlagNoComparison}},
		{"no difference", nil, &Reply{Kind: ReplySelect, ID: IDProposal, Comparison: []Citation{cite("displaced_bookings", map[string]any{IDProposal: 0, IDNoAction: 0})}}, ResultRetained, []string{FlagNoComparison}},
		{"formatted value", nil, &Reply{Kind: ReplySelect, ID: IDProposal, Comparison: []Citation{cite("severance_recognized_now.E04", map[string]any{IDProposal: "$3,950.00", IDNoAction: "$0.00"})}}, ResultRetained, []string{FlagCompared}},
	} {
		d := Classify(a, c.reqs, c.reply, rv)
		if d.Result != c.result || !slices.Equal(d.Flags, c.flags) {
			t.Errorf("%s: %s %v (%s; %v)", c.name, d.Result, d.Flags, d.Why, d.CitationChecks)
		}
		if (d.Result != ResultRetained && d.Result != ResultSelected && d.Result != ResultRevised) != (len(d.Submitted) == 0) {
			t.Errorf("%s: submitted %v", c.name, d.Submitted)
		}
	}
	// Copying the whole compared table, or more than three entries, is not
	// a comparison.
	var all []Citation
	for _, c := range a.Compared {
		vals := map[string]any{}
		for k, v := range c.Values {
			vals[k] = v
		}
		all = append(all, cite(c.Key, vals))
	}
	if d := Classify(a, nil, &Reply{Kind: ReplySelect, ID: IDProposal, Comparison: all}, rv); slices.Contains(d.Flags, FlagCompared) {
		t.Fatal("the whole compared table counted as a comparison")
	}
	// Repeating a short compared table (two rows here) is still copying.
	short := *a
	short.Compared = a.Compared[:2]
	if d := Classify(&short, nil, &Reply{Kind: ReplySelect, ID: IDProposal, Comparison: all[:2]}, rv); slices.Contains(d.Flags, FlagCompared) {
		t.Fatal("a copied two-row compared table counted as a comparison")
	}
	four := append(all[:2:2], good[0], cite("notice_days.E04", map[string]any{IDProposal: 14, IDAdjusted: 28}))
	if d := Classify(a, nil, &Reply{Kind: ReplySelect, ID: IDProposal, Comparison: four}, rv); slices.Contains(d.Flags, FlagCompared) {
		t.Fatal("four entries counted as a comparison")
	}
	if d := Classify(a, nil, &Reply{Kind: ReplySelect, ID: IDProposal, Comparison: four[2:]}, rv); !slices.Contains(d.Flags, FlagCompared) {
		t.Fatalf("two verified entries: %v", d.CitationChecks)
	}
	// Citation checks are in a fixed order (they are part of the record).
	multi := []Citation{cite("severance_recognized_now.E04", map[string]any{"alpha": 1, "beta": 2, "gamma": 3, IDProposal: 1})}
	first := Classify(a, nil, &Reply{Kind: ReplySelect, ID: IDProposal, Comparison: multi}, rv)
	for range 50 {
		if again := Classify(a, nil, &Reply{Kind: ReplySelect, ID: IDProposal, Comparison: multi}, rv); !slices.Equal(again.CitationChecks, first.CitationChecks) {
			t.Fatal("citation checks change order")
		}
	}
	// The reason is never read.
	x := Classify(a, nil, &Reply{Kind: ReplySelect, ID: IDProposal, Reason: "I compared every alternative carefully"}, rv)
	if !slices.Equal(x.Flags, []string{FlagNoComparison}) {
		t.Fatal("the reason text changed a flag")
	}
}

func TestMain(m *testing.M) { os.Exit(m.Run()) }
