package alternatives

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/session"
)

// V0.13b scripted evidence (docs/benchmark/v13b-dev): the termination
// case in both audiences and as presented, the scripted decision cases,
// and decision sensitivity on paired real engine states. Every reply is
// scripted; no model is involved. The files are written with
// DREAMTEAM_V13B_WRITE=1 and otherwise must match byte for byte (on every
// platform CI runs).

const evidenceDir = "../../docs/benchmark/v13b-dev"

func evidence(t *testing.T, name string, data []byte) {
	t.Helper()
	p := filepath.Join(evidenceDir, name)
	if os.Getenv("DREAMTEAM_V13B_WRITE") == "1" {
		if err := os.MkdirAll(evidenceDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, data, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(want, data) {
		t.Fatalf("%s differs from the committed evidence", name)
	}
}

func encode(t *testing.T, v any) []byte {
	raw, err := json.MarshalIndent(v, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	return append(raw, '\n')
}

func TestEvidenceTermination(t *testing.T) {
	s := at(t, t0)
	sel := mustSelect(t, s, term("E04", "2026-10-27"))
	a := mustAgent(t, s, sel)
	raw, _ := a.Encode()
	evidence(t, "termination-selection.json", encode(t, sel))
	evidence(t, "termination-agent.json", raw)
	text, err := Summary(a)
	if err != nil {
		t.Fatal(err)
	}
	for _, x := range a.Alternatives {
		d, err := Detail(a, x.ID)
		if err != nil {
			t.Fatal(err)
		}
		text += "\n--- inspect:" + x.ID + "\n" + d
	}
	evidence(t, "termination-presentation.txt", []byte(text))
	r, err := s.Record()
	if err != nil {
		t.Fatal(err)
	}
	e, err := BuildEvaluation(r, a, session.BranchOptions{Continuation: session.NoFurtherAction(), Verify: true})
	if err != nil {
		t.Fatal(err)
	}
	eraw, _ := e.Encode()
	if _, err := DecodeEvaluation(eraw); err != nil {
		t.Fatal(err)
	}
	evidence(t, "termination-evaluation.json", eraw)
}

// scripted is a deterministic reply rule: a pure function of the agent
// comparison. It stands in for a decision maker in these tests only.
type scripted struct {
	name string
	rule func(a *Agent) ([]string, *Reply)
}

func number(a *Agent, id, key string) (int64, bool) {
	v, ok := shownValue(a, id, key)
	if !ok || v.Number == nil {
		return 0, false
	}
	return *v.Number, true
}

func listed(a *Agent, id string) bool {
	return slices.ContainsFunc(a.Alternatives, func(x AgentAlternative) bool { return x.ID == id && x.Validation.OK })
}

func citeKey(a *Agent, key string, ids ...string) Citation {
	c := Citation{Item: key, Values: map[string]json.RawMessage{}}
	for _, id := range ids {
		n, _ := number(a, id, key)
		c.Values[id], _ = json.Marshal(n)
	}
	return c
}

var policies = []scripted{
	{"always-retain", func(a *Agent) ([]string, *Reply) { return nil, &Reply{Kind: ReplySelect, ID: IDProposal} }},
	{"no-action-if-displacing", func(a *Agent) ([]string, *Reply) {
		if n, _ := number(a, IDProposal, "displaced_bookings"); n > 0 {
			return []string{"inspect:no_action"}, &Reply{Kind: ReplySelect, ID: IDNoAction,
				Comparison: []Citation{citeKey(a, "displaced_bookings", IDProposal, IDNoAction)}}
		}
		return []string{"inspect:no_action"}, &Reply{Kind: ReplySelect, ID: IDProposal}
	}},
	{"later-last-day-if-listed", func(a *Agent) ([]string, *Reply) {
		if listed(a, IDAdjusted) {
			return []string{"inspect:adjusted"}, &Reply{Kind: ReplySelect, ID: IDAdjusted}
		}
		return nil, &Reply{Kind: ReplySelect, ID: IDProposal}
	}},
}

type point struct {
	Label    string       `json:"label"`
	At       string       `json:"at"`
	Proposal []api.Action `json:"proposal"`
}

func TestEvidenceScripted(t *testing.T) {
	type caseOut struct {
		Name     string   `json:"name"`
		Point    point    `json:"point"`
		Decision Decision `json:"decision"`
	}
	type pairOut struct {
		Fact      string               `json:"fact"`
		A         point                `json:"a"`
		B         point                `json:"b"`
		Differ    []string             `json:"comparedFactsThatDiffer"`
		ListedA   []string             `json:"alternativesA"`
		ListedB   []string             `json:"alternativesB"`
		Results   map[string][2]string `json:"resultsByScriptedRule"`
		Sensitive map[string]bool      `json:"sensitive"`
	}
	out := struct {
		Schema string    `json:"schema"`
		Note   string    `json:"note"`
		Cases  []caseOut `json:"cases"`
		Pairs  []pairOut `json:"sensitivity"`
	}{Schema: "dreamteam.v13b-scripted-evaluation/v1",
		Note: "Scripted replies only; no model. Results and flags follow EVALUATION_CONTRACT; a reason is recorded, never scored. Sensitivity pairs are real engine states of mini, seed 52, that differ in the fact named and possibly others (listed)."}

	build := func(p point) (*session.Session, *Agent) {
		s := at(t, p.At)
		return s, mustAgent(t, s, mustSelect(t, s, p.Proposal...))
	}
	base := point{"terminate E04, last day Oct 27", t0, []api.Action{term("E04", "2026-10-27")}}
	s, a := build(base)
	rv := func(b []api.Action) []api.ActionResult { return s.ValidateBatch(b...) }
	sev := []Citation{citeKey(a, "severance_recognized_now.E04", IDProposal, IDNoAction)}
	for _, c := range []struct {
		name  string
		reqs  []string
		reply *Reply
	}{
		{"alternative inspected", []string{"inspect:no_action", "inspect:adjusted"}, &Reply{Kind: ReplySelect, ID: IDProposal, Reason: "scripted"}},
		{"alternative compared", nil, &Reply{Kind: ReplySelect, ID: IDProposal, Comparison: sev, Reason: "scripted"}},
		{"original retained", nil, &Reply{Kind: ReplySelect, ID: IDProposal}},
		{"another alternative selected", []string{"inspect:adjusted"}, &Reply{Kind: ReplySelect, ID: IDAdjusted}},
		{"no action selected", []string{"inspect:no_action"}, &Reply{Kind: ReplySelect, ID: IDNoAction, Comparison: sev}},
		{"proposal revised", nil, &Reply{Kind: ReplyRevise, Actions: []api.Action{term("E04", "2026-11-03")}}},
		{"proposal abandoned", nil, &Reply{Kind: ReplyAbandon, Reason: "scripted"}},
		{"invalid alternative", nil, &Reply{Kind: ReplySelect, ID: "without_1"}},
		{"no decision", nil, nil},
		{"whole table copied", nil, &Reply{Kind: ReplySelect, ID: IDProposal, Comparison: func() []Citation {
			var cs []Citation
			for _, c := range a.Compared {
				cs = append(cs, citeKey(a, c.Key, IDProposal, IDNoAction, IDAdjusted))
			}
			return cs
		}()}},
	} {
		out.Cases = append(out.Cases, caseOut{c.name, base, Classify(a, c.reqs, c.reply, rv)})
	}

	for _, pr := range []struct {
		fact string
		a, b point
	}{
		{"displaced bookings", base, point{"terminate E04, last day Nov 10, decided Oct 27", "2026-10-27T07:00:00", []api.Action{term("E04", "2026-11-10")}}},
		{"severance", point{"terminate E05, last day Oct 27", t0, []api.Action{term("E05", "2026-10-27")}},
			point{"terminate E06, last day Oct 27", t0, []api.Action{term("E06", "2026-10-27")}}},
		{"later-last-day alternative listed", base, point{"terminate E04, last day Feb 1", t0, []api.Action{term("E04", "2027-02-01")}}},
	} {
		sa, aa := build(pr.a)
		sb, ab := build(pr.b)
		po := pairOut{Fact: pr.fact, A: pr.a, B: pr.b, Differ: []string{}, Results: map[string][2]string{}, Sensitive: map[string]bool{}}
		for _, x := range aa.Alternatives {
			po.ListedA = append(po.ListedA, x.ID)
		}
		for _, x := range ab.Alternatives {
			po.ListedB = append(po.ListedB, x.ID)
		}
		for _, v := range aa.Alternatives[0].Facts {
			if v.Number == nil {
				continue
			}
			key := v.Key
			w, ok := shownValue(ab, IDProposal, key)
			if !ok {
				continue
			}
			if *w.Number != *v.Number {
				po.Differ = append(po.Differ, fmt.Sprintf("%s: %d vs %d", key, *v.Number, *w.Number))
			}
		}
		for _, pol := range policies {
			ra, rpa := pol.rule(aa)
			rb, rpb := pol.rule(ab)
			da := Classify(aa, ra, rpa, func(b []api.Action) []api.ActionResult { return sa.ValidateBatch(b...) })
			db := Classify(ab, rb, rpb, func(b []api.Action) []api.ActionResult { return sb.ValidateBatch(b...) })
			po.Results[pol.name] = [2]string{da.Result + ":" + da.Alternative, db.Result + ":" + db.Alternative}
			po.Sensitive[pol.name] = po.Results[pol.name][0] != po.Results[pol.name][1]
		}
		out.Pairs = append(out.Pairs, po)
	}
	// The measurement detects what the rules use, and only that.
	p := out.Pairs
	if !p[0].Sensitive["no-action-if-displacing"] || p[0].Sensitive["always-retain"] || !p[2].Sensitive["later-last-day-if-listed"] ||
		p[1].Sensitive["no-action-if-displacing"] || p[1].Sensitive["always-retain"] {
		t.Fatalf("sensitivity %+v", p)
	}
	want := []string{ResultRetained, "", ResultRetained, ResultSelected, ResultSelectedNoAction, ResultRevised, ResultAbandoned, ResultInvalidAlterative, ResultNoDecision, ResultRetained}
	for i, c := range out.Cases {
		if want[i] != "" && c.Decision.Result != want[i] {
			t.Errorf("%s: %s", c.Name, c.Decision.Result)
		}
	}
	if !slices.Contains(out.Cases[0].Decision.Flags, FlagInspected) || !slices.Contains(out.Cases[1].Decision.Flags, FlagCompared) ||
		slices.Contains(out.Cases[9].Decision.Flags, FlagCompared) {
		t.Fatal("flags")
	}
	evidence(t, "scripted-evaluation.json", encode(t, out))
}
