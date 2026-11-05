package alternatives

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/session"
)

// V0.13c evidence (docs/benchmark/v13c-dev): rule v2 against v1 on the F4
// cases, and the sensitivity pairs: real engine states of mini, seed 52,
// each member's control and treatment turns exactly as a driver would send
// them, and every visible item that differs between the two members. The
// files are written with DREAMTEAM_V13C_WRITE=1 and otherwise must match
// byte for byte, on every platform CI runs. No model is involved.

const evidenceDirV13c = "../../docs/benchmark/v13c-dev"

func evidenceV13c(t *testing.T, name string, data []byte) {
	t.Helper()
	p := filepath.Join(evidenceDirV13c, name)
	if os.Getenv("DREAMTEAM_V13C_WRITE") == "1" {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
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

func TestEvidenceSelectionV2(t *testing.T) {
	type row struct {
		Case     string       `json:"case"`
		At       string       `json:"at"`
		Proposal []api.Action `json:"proposal"`
		V1       *Selection   `json:"v1"`
		V2       *Selection   `json:"v2"`
	}
	out := struct {
		Schema string            `json:"schema"`
		Note   string            `json:"note"`
		Rules  map[string]string `json:"rules"`
		Cases  []row             `json:"cases"`
	}{Schema: "dreamteam.v13c-selection-v2/v1",
		Note:  "Rule v1 (unchanged) and rule v2 on the same proposals at the same engine state (mini, seed 52). v2 removes a candidate whose canonical batch equals an earlier candidate's, and nothing else.",
		Rules: map[string]string{SelectionRule: SelectionRuleSha256, SelectionRuleV2: SelectionRuleV2Sha256}}
	s := at(t, t0)
	for _, c := range []struct {
		name string
		p    []api.Action
	}{
		{"single termination (no duplicate)", []api.Action{term("E04", "2026-10-27")}},
		{"duplicate termination (F4)", []api.Action{term("E04", "2026-10-27"), term("E04", "2026-10-27")}},
		{"duplicate leave-one-out among three", []api.Action{term("E03", "2026-10-27"), term("E03", "2026-10-27"), term("E04", "2026-10-27")}},
		{"same actions, different order", []api.Action{term("E04", "2026-10-27"), fifo, term("E04", "2026-10-27")}},
		{"same employee, different last days", []api.Action{term("E04", "2026-10-27"), term("E04", "2026-11-10")}},
		{"two employees", []api.Action{term("E03", "2026-10-27"), term("E04", "2026-10-27")}},
	} {
		v1, v2 := mustSelect(t, s, c.p...), mustSelectV2(t, s, c.p...)
		sameButDuplicates(t, v1, v2)
		out.Cases = append(out.Cases, row{c.name, t0, canon(c.p), v1, v2})
	}
	evidenceV13c(t, "selection-v2.json", encode(t, out))
}

// member is one side of a sensitivity pair: a real engine state reached
// from the start of mini, seed 52, with no action, and a scripted
// proposal. No model made the proposal.
type member struct {
	Label    string       `json:"label"`
	Scenario string       `json:"scenario"`
	Seed     int64        `json:"seed"`
	At       string       `json:"at"`
	Proposal []api.Action `json:"proposal"`
}

type sensitivityPair struct {
	ID        string `json:"id"`
	Fact      string `json:"fact"`
	A, B      member `json:"-"`
	Isolation string `json:"isolation"`
	SameState bool   `json:"sameState"`
}

func tele(windows ...string) api.Action {
	return api.Action{Type: api.ActContractTeleradiology, Op: api.TeleOpStart, Plan: "standard-v1", RequestID: "tele-1", Scope: "all",
		Effective: "2026-10-27", Windows: windows}
}

func mini(label, when string, p ...api.Action) member {
	return member{label, "mini", 52, when, canon(p)}
}

// SensitivityPairs is the V0.13c design's pair list (SENSITIVITY_CASES.md).
var sensitivityPairs = []sensitivityPair{
	{ID: "S1", Fact: "displaced bookings: zero versus nonzero (decided two weeks apart)",
		A: mini("terminate Sam (E04), 14 days' notice, decided Oct 13", "2026-10-13T07:00:00", term("E04", "2026-10-27")),
		B: mini("terminate Sam (E04), 14 days' notice, decided Oct 27", "2026-10-27T07:00:00", term("E04", "2026-11-10"))},
	{ID: "S2", Fact: "displaced bookings: zero versus nonzero (same state)",
		A: mini("terminate Riley (E03), last day Feb 16", "2026-10-20T07:00:00", term("E03", "2027-02-16")),
		B: mini("terminate Riley (E03), last day Feb 9", "2026-10-20T07:00:00", term("E03", "2027-02-09"))},
	{ID: "S3", Fact: "severance",
		A: mini("terminate Casey (E05), last day Oct 27", "2026-10-13T07:00:00", term("E05", "2026-10-27")),
		B: mini("terminate Morgan (E06), last day Oct 27", "2026-10-13T07:00:00", term("E06", "2026-10-27"))},
	{ID: "S4", Fact: "an alternative available versus unavailable (the later-last-day adjustment)",
		A: mini("terminate Casey (E05), last day Jan 26", "2026-10-13T07:00:00", term("E05", "2027-01-26")),
		B: mini("terminate Casey (E05), last day Feb 2", "2026-10-13T07:00:00", term("E05", "2027-02-02"))},
	{ID: "S5", Fact: "guaranteed commitment (teleradiology retainer to the earliest end)",
		A: mini("teleradiology, weeknights, saturday and sunday, from Oct 27", "2026-10-13T07:00:00", tele("weeknights", "saturday", "sunday")),
		B: mini("teleradiology, weeknights only, from Oct 27", "2026-10-13T07:00:00", tele("weeknights"))},
	{ID: "S6", Fact: "capacity loss (scheduled slots no longer staffed)",
		A: mini("terminate Sam (E04), last day Oct 27", "2026-10-13T07:00:00", term("E04", "2026-10-27")),
		B: mini("terminate Sam (E04), last day Jan 26", "2026-10-13T07:00:00", term("E04", "2027-01-26"))},
}

// visible is every value a member's treatment turn shows, by
// alternative, section and key, as displayed. The proposal's employee ID
// in a key is written <employee>, so that two members terminating
// different employees are compared item by item (the labels keep names).
func visible(a *Agent, emp string) map[string]string {
	out := map[string]string{}
	norm := func(k string) string {
		if emp == "" {
			return k
		}
		return strings.ReplaceAll(k, "."+emp, ".<employee>")
	}
	for _, x := range a.Alternatives {
		out[x.ID+"/listed"] = "listed"
		out[x.ID+"/actions"] = mustJSON(x.Actions)
		out[x.ID+"/validation"] = mustJSON(x.Validation)
		for sec, list := range map[string][]Value{"fact": x.Facts, "estimate": x.Estimates, "obligation": x.Obligations, "reversibility": {x.Reversibility}} {
			for _, v := range list {
				out[x.ID+"/"+sec+"/"+norm(v.Key)] = v.Label + labelSep + formatValue(v) + " · " + v.When
			}
		}
		for _, u := range x.Unavailable {
			out[x.ID+"/unavailable/"+u.Key] = u.Reason
		}
	}
	return out
}

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func lineDiff(a, b string) []string {
	x, y := strings.Split(a, "\n"), strings.Split(b, "\n")
	var out []string
	for i := 0; i < max(len(x), len(y)); i++ {
		var l, r string
		if i < len(x) {
			l = x[i]
		}
		if i < len(y) {
			r = y[i]
		}
		if l != r {
			out = append(out, fmt.Sprintf("line %d: %q vs %q", i+1, l, r))
		}
	}
	return out
}

func TestEvidenceSensitivityPairs(t *testing.T) {
	type side struct {
		member
		Due              bool              `json:"due"`
		Alternatives     []string          `json:"alternatives"`
		Removed          []string          `json:"removed"`
		ControlSha256    string            `json:"controlMessageSha256"`
		TreatmentSha256  string            `json:"treatmentMessageSha256"`
		ComparisonSha256 string            `json:"comparisonSha256"`
		Values           map[string]string `json:"values"`
	}
	type pairOut struct {
		ID            string               `json:"id"`
		Fact          string               `json:"fact"`
		SameState     bool                 `json:"sameState"`
		ObservationEq bool                 `json:"observationEqual"`
		A             side                 `json:"a"`
		B             side                 `json:"b"`
		Differ        []string             `json:"visibleItemsThatDiffer"`
		ControlDiffer []string             `json:"controlLinesThatDiffer"`
		Isolation     string               `json:"isolation"`
		Sensitive     map[string][2]string `json:"scriptedResults"`
	}
	out := struct {
		Schema string            `json:"schema"`
		Note   string            `json:"note"`
		Arms   map[string]string `json:"texts"`
		Pairs  []pairOut         `json:"pairs"`
	}{Schema: "dreamteam.v13c-sensitivity-pairs/v1",
		Note: "Each member is a real engine state (mini, seed 52, reached from the start with no action) and a scripted proposal; no model made it. visibleItemsThatDiffer lists every value of the treatment turn (summary and every inspect answer) that differs between the members, by alternative/section/key; controlLinesThatDiffer every line of the control turn. Nothing is hidden or equalized: where the simulator does not permit isolating one fact, the other differences are listed here.",
		Arms: map[string]string{SelectionRuleV2: SelectionRuleV2Sha256, ControlID: ControlSha256, PresentationID: PresentationSha256}}
	for _, pr := range sensitivityPairs {
		po := pairOut{ID: pr.ID, Fact: pr.Fact, SameState: pr.A.At == pr.B.At, Differ: []string{}, ControlDiffer: []string{}, Sensitive: map[string][2]string{}}
		var obs [2]string
		var ag [2]*Agent
		var ctl [2]string
		var ss [2]*session.Session
		for k, m := range []member{pr.A, pr.B} {
			s := at(t, m.At)
			ss[k] = s
			c, tr, sel := mustSteps(t, s, m.Proposal...)
			if !c.Due || !tr.Due {
				t.Fatalf("%s: the step is not due (%s)", pr.ID, c.Why)
			}
			a := mustAgent(t, s, sel)
			ag[k], ctl[k], obs[k] = a, c.Message, mustJSON(s.Observe(0))
			sd := side{member: m, Due: true, ControlSha256: c.MessageSha256, TreatmentSha256: tr.MessageSha256, ComparisonSha256: tr.ComparisonSha256,
				Values: visible(a, m.Proposal[0].Employee), Removed: removedIDs(sel)}
			sd.Alternatives = ids(sel)
			if sd.Removed == nil {
				sd.Removed = []string{}
			}
			text := "=== control\n" + c.Message + "\n=== treatment\n" + tr.Message
			for _, id := range sd.Alternatives {
				text += "\n--- inspect:" + id + "\n" + tr.Inspect[id]
			}
			evidenceV13c(t, fmt.Sprintf("turns/%s-%c.txt", pr.ID, 'a'+k), []byte(text))
			if k == 0 {
				po.A = sd
			} else {
				po.B = sd
			}
		}
		po.ObservationEq = obs[0] == obs[1]
		if po.SameState != po.ObservationEq {
			t.Fatalf("%s: same state %v, observation equal %v", pr.ID, po.SameState, po.ObservationEq)
		}
		va, vb := po.A.Values, po.B.Values
		keys := slices.Sorted(func(yield func(string) bool) {
			seen := map[string]bool{}
			for _, m := range []map[string]string{va, vb} {
				for k := range m {
					if !seen[k] {
						seen[k] = true
						if !yield(k) {
							return
						}
					}
				}
			}
		})
		var nValue, nLabel, nOnly int
		for _, k := range keys {
			x, okx := va[k]
			y, oky := vb[k]
			lx, vx, _ := strings.Cut(x, labelSep)
			ly, vy, _ := strings.Cut(y, labelSep)
			switch {
			case !okx:
				po.Differ, nOnly = append(po.Differ, "[only in b] "+k), nOnly+1
			case !oky:
				po.Differ, nOnly = append(po.Differ, "[only in a] "+k), nOnly+1
			case vx != vy || (!strings.Contains(x, labelSep) && x != y):
				po.Differ, nValue = append(po.Differ, fmt.Sprintf("[value] %s: %s | %s", k, strings.ReplaceAll(x, labelSep, ": "), strings.ReplaceAll(y, labelSep, ": "))), nValue+1
			case lx != ly:
				po.Differ, nLabel = append(po.Differ, fmt.Sprintf("[label only] %s: %s | %s", k, lx, ly)), nLabel+1
			}
		}
		po.ControlDiffer = append(po.ControlDiffer, lineDiff(ctl[0], ctl[1])...)
		for _, pol := range append(slices.Clone(policies), v13cPolicies...) {
			ra, rpa := pol.rule(ag[0])
			rb, rpb := pol.rule(ag[1])
			da := Classify(ag[0], ra, rpa, func(b []api.Action) []api.ActionResult { return ss[0].ValidateBatch(b...) })
			db := Classify(ag[1], rb, rpb, func(b []api.Action) []api.ActionResult { return ss[1].ValidateBatch(b...) })
			po.Sensitive[pol.name] = [2]string{da.Result + ":" + da.Alternative, db.Result + ":" + db.Alternative}
		}
		po.Isolation = fmt.Sprintf("%d items differ in value, %d in label only, %d are shown in one member only", nValue, nLabel, nOnly)
		out.Pairs = append(out.Pairs, po)
	}
	// Each pair's fact is one a scripted rule can detect.
	want := map[string]string{"S1": "no-action-if-displacing", "S2": "no-action-if-displacing", "S3": "no-action-if-severance-over-4400",
		"S4": "later-last-day-if-listed", "S5": "no-action-if-guarantee-over-2000", "S6": "no-action-if-slots-over-1000"}
	for _, p := range out.Pairs {
		r := p.Sensitive[want[p.ID]]
		if r[0] == r[1] {
			t.Errorf("%s: %s does not detect the fact: %v", p.ID, want[p.ID], r)
		}
		if r := p.Sensitive["always-retain"]; r[0] != r[1] {
			t.Errorf("%s: always-retain differs", p.ID)
		}
	}
	evidenceV13c(t, "sensitivity-pairs.json", encode(t, out))
}

const labelSep = "\x00"

var v13cPolicies = []scripted{
	{"no-action-if-severance-over-4400", func(a *Agent) ([]string, *Reply) {
		for _, v := range a.Alternatives[0].Facts {
			if strings.HasPrefix(v.Key, "severance_recognized_now.") && v.Number != nil && *v.Number > 440000 {
				return []string{"inspect:proposal"}, &Reply{Kind: ReplySelect, ID: IDNoAction}
			}
		}
		return []string{"inspect:proposal"}, &Reply{Kind: ReplySelect, ID: IDProposal}
	}},
	{"no-action-if-guarantee-over-2000", func(a *Agent) ([]string, *Reply) {
		for _, list := range [][]Value{a.Alternatives[0].Facts, a.Alternatives[0].Obligations, a.Alternatives[0].Estimates} {
			for _, v := range list {
				if strings.Contains(v.Key, "guarant") && v.Unit == UnitCents && v.Number != nil && *v.Number > 200000 {
					return []string{"inspect:proposal"}, &Reply{Kind: ReplySelect, ID: IDNoAction}
				}
			}
		}
		return []string{"inspect:proposal"}, &Reply{Kind: ReplySelect, ID: IDProposal}
	}},
	{"no-action-if-slots-over-1000", func(a *Agent) ([]string, *Reply) {
		if n, ok := number(a, IDProposal, "slots_closed"); ok && n > 1000 {
			return []string{"inspect:proposal"}, &Reply{Kind: ReplySelect, ID: IDNoAction}
		}
		return []string{"inspect:proposal"}, &Reply{Kind: ReplySelect, ID: IDProposal}
	}},
}
