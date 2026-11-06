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
)

// V0.13d evidence: presentation v2's closed table, every sensitivity-pair
// member's turns exactly as a driver would send them (control, treatment
// and every inspect answer), and, per pair, every typed item and every
// rendered line that differs. Written with DREAMTEAM_V13D_WRITE=1 and
// otherwise required byte for byte. No model is involved.

const evidenceDirV13d = "../../docs/benchmark/v13d-dev"

func evidenceV13d(t *testing.T, name string, data []byte) {
	t.Helper()
	p := filepath.Join(evidenceDirV13d, name)
	if os.Getenv("DREAMTEAM_V13D_WRITE") == "1" {
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

// viewItems is every typed item of a view, by alternative/section/key,
// as shown (the rendered line).
func viewItems(v *View) map[string]string {
	out := map[string]string{}
	for _, a := range v.Alternatives {
		out[a.ID+"/actions"] = viewActions(a)
		out[a.ID+"/validation"] = viewValidation(a)
		out[a.ID+"/reversibility"] = viewReversibility[a.Reversibility]
		for sec, list := range map[string][]ViewItem{"facts": a.Facts, "estimates": a.Estimates, "obligations": a.Obligations} {
			for i, it := range list {
				out[fmt.Sprintf("%s/%s/%s", a.ID, sec, it.Key)] = viewLine(it)
				out[fmt.Sprintf("%s/%s/#%d", a.ID, sec, i+1)] = it.Key
			}
		}
		for i, it := range a.Unavailable {
			out[fmt.Sprintf("%s/unavailable/#%d", a.ID, i+1)] = viewUnavailableText(it)
		}
	}
	for i, r := range v.Refs {
		out[fmt.Sprintf("refs/#%d", i+1)] = r.ID + " " + refText(r)
	}
	out["time"] = v.Time
	return out
}

func TestEvidenceV13d(t *testing.T) {
	evidenceV13d(t, "presentation/view-table.txt", []byte(ViewTableText()))
	evidenceV13d(t, "presentation/presentation-v2.tmpl", presentationV2Text)
	type side struct {
		member
		Due              bool              `json:"due"`
		Why              string            `json:"why"`
		Alternatives     []string          `json:"alternatives"`
		Refs             map[string]string `json:"refs"`
		ControlSha256    string            `json:"controlMessageSha256"`
		TreatmentSha256  string            `json:"treatmentMessageSha256"`
		ComparisonSha256 string            `json:"comparisonSha256"`
		ViewSha256       string            `json:"viewSha256"`
	}
	type pairOut struct {
		ID              string   `json:"id"`
		Fact            string   `json:"fact"`
		A               side     `json:"a"`
		B               side     `json:"b"`
		ValueDiffer     []string `json:"itemsWithDifferentValues"`
		OnlyOne         []string `json:"itemsShownForOneMemberOnly"`
		ControlDiffer   []string `json:"controlLinesThatDiffer"`
		TreatmentDiffer []string `json:"treatmentLinesThatDiffer"`
	}
	out := struct {
		Schema string            `json:"schema"`
		Note   string            `json:"note"`
		Texts  map[string]string `json:"texts"`
		Pairs  []pairOut         `json:"pairs"`
	}{Schema: "dreamteam.v13d-sensitivity-pairs/v1",
		Note:  "The V0.13c pairs, unchanged, shown through presentation v2 (the typed view). itemsWithDifferentValues lists every shown item (by alternative/section/key, including each list's order) whose rendered line differs between the members; itemsShownForOneMemberOnly every item one member shows and the other does not; the line lists every rendered line that differs, position by position, in the control text, and in the treatment summary followed by every inspect answer. They are context pairs: none isolates one fact.",
		Texts: map[string]string{PresentationV2ID: PresentationV2Sha256, ViewSchema: ViewTableSha256, SelectionRuleV2: SelectionRuleV2Sha256}}
	for _, pr := range sensitivityPairs {
		var sides [2]side
		var views [2]map[string]string
		var texts [2][2]string
		for k, m := range []member{pr.A, pr.B} {
			s := at(t, m.At)
			c, tr := mustTurns(t, s, m.Proposal...)
			b, err := build(s, m.Proposal)
			if err != nil || b.view == nil {
				t.Fatalf("%s: %v", m.Label, err)
			}
			var ids []string
			for _, x := range b.view.Alternatives {
				ids = append(ids, x.ID)
			}
			sides[k] = side{member: m, Due: tr.Due, Why: tr.Why, Alternatives: ids, Refs: tr.Record.Map.Refs,
				ControlSha256: c.Record.MessageSha256, TreatmentSha256: tr.Record.MessageSha256,
				ComparisonSha256: tr.Record.ComparisonSha256, ViewSha256: tr.Record.ViewSha256}
			views[k] = viewItems(b.view)
			full := tr.Message
			for _, id := range ids {
				full += "\n--- inspect:" + id + "\n" + tr.Inspect[id]
			}
			texts[k] = [2]string{c.Message, full}
			var f bytes.Buffer
			fmt.Fprintf(&f, "%s %s: %s\n(record only, never sent: %v)\n\n=== CONTROL (sent)\n%s\n=== TREATMENT (sent)\n%s\n", pr.ID, []string{"a", "b"}[k], m.Label, tr.Record.Map.Refs, c.Message, tr.Message)
			for _, id := range ids {
				fmt.Fprintf(&f, "=== TREATMENT inspect:%s (sent on request)\n%s\n", id, tr.Inspect[id])
			}
			evidenceV13d(t, fmt.Sprintf("turns/%s-%s.txt", pr.ID, []string{"a", "b"}[k]), f.Bytes())
		}
		po := pairOut{ID: pr.ID, Fact: pr.Fact, A: sides[0], B: sides[1], ValueDiffer: []string{}, OnlyOne: []string{}}
		var keys []string
		for k := range views[0] {
			keys = append(keys, k)
		}
		for k := range views[1] {
			if _, ok := views[0][k]; !ok {
				keys = append(keys, k)
			}
		}
		slices.Sort(keys)
		for _, k := range keys {
			a, okA := views[0][k]
			b, okB := views[1][k]
			switch {
			case okA && okB && a != b:
				po.ValueDiffer = append(po.ValueDiffer, fmt.Sprintf("%s: %q vs %q", k, a, b))
			case okA != okB:
				po.OnlyOne = append(po.OnlyOne, fmt.Sprintf("%s: %q vs %q", k, a, b))
			}
		}
		po.ControlDiffer, po.TreatmentDiffer = lineDiff(texts[0][0], texts[1][0]), lineDiff(texts[0][1], texts[1][1])
		if po.ControlDiffer == nil {
			po.ControlDiffer = []string{}
		}
		out.Pairs = append(out.Pairs, po)
	}
	raw, _ := json.MarshalIndent(out, "", " ")
	evidenceV13d(t, "sensitivity-pairs.json", append(raw, '\n'))
	// The summary table SENSITIVITY_CASES.md prints.
	var md strings.Builder
	md.WriteString("| Pair | Items with different values | Items shown for one member only | Control lines that differ | Treatment lines that differ |\n|---|---|---|---|---|\n")
	for _, p := range out.Pairs {
		fmt.Fprintf(&md, "| %s | %d | %d | %d | %d |\n", p.ID, len(p.ValueDiffer), len(p.OnlyOne), len(p.ControlDiffer), len(p.TreatmentDiffer))
	}
	evidenceV13d(t, "sensitivity-summary.md", []byte(md.String()))
}
