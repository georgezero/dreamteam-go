package alternatives

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/session"
)

// V0.13c, F4: rule dreamteam.alternative-selection/v2 removes a candidate
// whose canonical batch equals an earlier candidate's. Everything else is
// v1's, which is unchanged.

func mustSelectV2(t testing.TB, s *session.Session, p ...api.Action) *Selection {
	t.Helper()
	sel, err := SelectV2(s, p)
	if err != nil {
		t.Fatal(err)
	}
	return sel
}

func removedIDs(sel *Selection) []string {
	var out []string
	for _, r := range sel.Removed {
		out = append(out, r.ID+"="+r.SameAs)
	}
	return out
}

var fifo = api.Action{Type: api.ActSetWorklistPolicy, Policy: "fifo"}

// sameButDuplicates requires v2's candidates to be v1's with exactly the
// removed IDs taken out, every kept candidate equal to v1's apart from its
// identity (which names the rule), and nothing added or reordered.
func sameButDuplicates(t *testing.T, v1, v2 *Selection) {
	t.Helper()
	if v1.Rule != SelectionRule || v2.Rule != SelectionRuleV2 || v2.RuleSha256 != SelectionRuleV2Sha256 {
		t.Fatalf("rules %s %s", v1.Rule, v2.Rule)
	}
	if v1.Applies != v2.Applies || v1.Why != v2.Why || !jsonEqual(v1.Proposal, v2.Proposal) || !jsonEqual(v1.Omitted, v2.Omitted) ||
		v1.At != v2.At || v1.Today != v2.Today || v1.HorizonDays != v2.HorizonDays {
		t.Fatal("v2 changed something other than the candidate list")
	}
	gone := map[string]bool{}
	for _, r := range v2.Removed {
		gone[r.ID] = true
	}
	var want []Candidate
	for _, c := range v1.Candidates {
		if !gone[c.ID] {
			want = append(want, c)
		}
	}
	if len(want) != len(v2.Candidates) || len(v2.Candidates)+len(v2.Removed) != len(v1.Candidates) {
		t.Fatalf("v1 %v, v2 %v removed %v", ids(v1), ids(v2), removedIDs(v2))
	}
	for k, c := range v2.Candidates {
		w := want[k]
		if c.ID != w.ID || c.Source != w.Source || c.OK != w.OK || !jsonEqual(c.Actions, w.Actions) || !jsonEqual(c.Validation, w.Validation) {
			t.Fatalf("candidate %s differs from v1's", c.ID)
		}
		if c.Sha256 == w.Sha256 {
			t.Fatalf("candidate %s: the identity does not name the rule", c.ID)
		}
	}
	if len(v2.Candidates) > MaxCandidates {
		t.Fatalf("%d candidates", len(v2.Candidates))
	}
	// Every removal names an earlier kept candidate with the same batch.
	for _, r := range v2.Removed {
		k := slices.IndexFunc(v2.Candidates, func(c Candidate) bool { return c.ID == r.SameAs })
		if k < 0 || batchKey(v2.Candidates[k].Actions) != r.BatchSha256 || batchKey(r.Actions) != r.BatchSha256 {
			t.Fatalf("removal %s=%s", r.ID, r.SameAs)
		}
		if slices.Index(ids(v1), r.SameAs) > slices.Index(ids(v1), r.ID) {
			t.Fatalf("removal %s kept a later candidate", r.ID)
		}
	}
	// The kept batches are distinct.
	seen := map[string]bool{}
	for _, c := range v2.Candidates {
		if k := batchKey(c.Actions); seen[k] {
			t.Fatalf("v2 lists one batch twice")
		} else {
			seen[k] = true
		}
	}
}

func TestSelectionV2PinnedText(t *testing.T) {
	if sha256Hex(selectionTextV2) != SelectionRuleV2Sha256 {
		t.Fatalf("selection rule v2 text changed: %s", sha256Hex(selectionTextV2))
	}
	if sha256Hex(selectionText) != SelectionRuleSha256 {
		t.Fatal("selection rule v1 text changed")
	}
	// v2's text is v1's with its name and one appended section.
	v1, v2 := string(selectionText), string(selectionTextV2)
	head := strings.Replace(v1, SelectionRule, SelectionRuleV2, 1)
	if !strings.HasPrefix(v2, head) || !strings.HasPrefix(v2[len(head):], "\nDuplicates (the only change from "+SelectionRule+")") {
		t.Fatal("v2's text is not v1's plus the duplicates section")
	}
	if _, err := SelectRule(at(t, t0), "dreamteam.alternative-selection/v3", []api.Action{term("E04", "2026-10-27")}); err == nil {
		t.Fatal("an unknown rule was applied")
	}
}

// 1 and 6: the same termination twice. The proposal stays refused with the
// engine's codes; the second leave-one-out candidate is removed.
func TestSelectionV2DuplicateTermination(t *testing.T) {
	s := at(t, t0)
	before := s.Digests()
	p := []api.Action{term("E04", "2026-10-27"), term("E04", "2026-10-27")}
	v1, v2 := mustSelect(t, s, p...), mustSelectV2(t, s, p...)
	if !slices.Equal(ids(v1), []string{IDProposal, IDNoAction, "without_1", "without_2"}) {
		t.Fatalf("v1 %v", ids(v1))
	}
	if !slices.Equal(ids(v2), []string{IDProposal, IDNoAction, "without_1"}) || !slices.Equal(removedIDs(v2), []string{"without_2=without_1"}) {
		t.Fatalf("v2 %v removed %v", ids(v2), removedIDs(v2))
	}
	sameButDuplicates(t, v1, v2)
	// 6: the invalid proposal is not repaired.
	pr := v2.Candidates[0]
	if pr.OK || !jsonEqual(pr.Actions, p) || !jsonEqual(pr.Validation, v1.Candidates[0].Validation) {
		t.Fatalf("proposal %+v", pr)
	}
	if !slices.ContainsFunc(pr.Validation, func(r api.ActionResult) bool { return !r.OK && r.Code != "" }) {
		t.Fatal("the refusal codes are gone")
	}
	if s.Digests() != before {
		t.Fatal("selection changed the session")
	}
	// v1's duplicate pair cannot be branched; v2's list can.
	a1, a2 := mustAgent(t, s, v1), mustAgent(t, s, v2)
	rec, err := s.Record()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := BuildEvaluation(rec, a1, session.BranchOptions{Continuation: session.NoFurtherAction()}); !errors.Is(err, session.ErrBranchInput) {
		t.Fatalf("v1's duplicates branched: %v", err)
	}
	ev, err := BuildEvaluation(rec, a2, session.BranchOptions{Continuation: session.NoFurtherAction()})
	if err != nil {
		t.Fatal(err)
	}
	if len(ev.Branches) != 3 {
		t.Fatalf("%d branches", len(ev.Branches))
	}
	if err := VerifyEvaluation(rec, ev); err != nil {
		t.Fatal(err)
	}
}

// 2 and 8: a duplicate among three: the list keeps the first and is not
// refilled with without_3 (no substitution, never more than four).
func TestSelectionV2DuplicateLeaveOneOut(t *testing.T) {
	s := at(t, t0)
	p := []api.Action{term("E03", "2026-10-27"), term("E03", "2026-10-27"), term("E04", "2026-10-27")}
	v1, v2 := mustSelect(t, s, p...), mustSelectV2(t, s, p...)
	if !slices.Equal(ids(v1), []string{IDProposal, IDNoAction, "without_1", "without_2"}) {
		t.Fatalf("v1 %v", ids(v1))
	}
	if !slices.Equal(ids(v2), []string{IDProposal, IDNoAction, "without_1"}) || !slices.Equal(removedIDs(v2), []string{"without_2=without_1"}) {
		t.Fatalf("v2 %v removed %v", ids(v2), removedIDs(v2))
	}
	sameButDuplicates(t, v1, v2)
	// 8: over a grid of proposals, v2 never lists more than v1, never more
	// than four, and only ever removes.
	e := []string{"E01", "E03", "E04", "E05"}
	for _, x := range e {
		for _, y := range e {
			for _, z := range append(e, "") {
				p := []api.Action{term(x, "2026-10-27"), term(y, "2026-10-27")}
				if z != "" {
					p = append(p, term(z, "2026-10-27"))
				}
				sameButDuplicates(t, mustSelect(t, s, p...), mustSelectV2(t, s, p...))
			}
		}
	}
	for _, p := range [][]api.Action{{fifo, term("E04", "2026-10-27"), fifo, term("E04", "2026-10-27")},
		{term("E04", "2026-10-27"), term("E04", "2026-10-27"), term("E04", "2026-10-27"), term("E04", "2026-10-27"), term("E04", "2026-10-27")}} {
		v1, v2 := mustSelect(t, s, p...), mustSelectV2(t, s, p...)
		sameButDuplicates(t, v1, v2)
	}
}

// 3: labels, IDs and sources are not compared; canonical content is (an
// action built with an empty list where another has none is the same).
func TestSelectionV2LabelsNotCompared(t *testing.T) {
	a := api.Action{Type: api.ActContractTeleradiology, Op: api.TeleOpStart, Plan: "standard-v1", Scope: "urgent", Effective: "2026-11-10"}
	b := a
	b.Windows = []string{}
	if batchKey([]api.Action{a}) != batchKey([]api.Action{b}) {
		t.Fatal("canonical forms differ")
	}
	in := []Candidate{
		{ID: IDProposal, Source: "proposal", Actions: canon([]api.Action{a, term("E04", "2026-10-27")})},
		{ID: IDNoAction, Source: "no_action", Actions: canon(nil)},
		{ID: "x", Source: "leave_one_out", Actions: canon([]api.Action{a})},
		{ID: "y", Source: "adjustment", Actions: canon([]api.Action{b})},
	}
	kept, removed := dedupe(in)
	if len(kept) != 3 || kept[2].ID != "x" || len(removed) != 1 || removed[0].ID != "y" || removed[0].SameAs != "x" || removed[0].Source != "adjustment" {
		t.Fatalf("%+v %+v", kept, removed)
	}
	// Different IDs, identical actions, in the rule itself: without_1 and
	// without_2 above (TestSelectionV2DuplicateTermination).
}

// 4: batch order is part of the batch: the engine validates a batch in
// order and the branch builder identifies it so. Two leave-one-out
// candidates with the same actions in a different order are both kept.
func TestSelectionV2ActionOrder(t *testing.T) {
	s := at(t, t0)
	p := []api.Action{term("E04", "2026-10-27"), fifo, term("E04", "2026-10-27")}
	v1, v2 := mustSelect(t, s, p...), mustSelectV2(t, s, p...)
	if !slices.Equal(ids(v2), []string{IDProposal, IDNoAction, "without_1", "without_3"}) || len(v2.Removed) != 0 {
		t.Fatalf("v2 %v removed %v", ids(v2), removedIDs(v2))
	}
	if !jsonEqual(v2.Candidates[2].Actions, []api.Action{fifo, term("E04", "2026-10-27")}) ||
		!jsonEqual(v2.Candidates[3].Actions, []api.Action{term("E04", "2026-10-27"), fifo}) {
		t.Fatal("not the reordered pair")
	}
	sameButDuplicates(t, v1, v2)
	x := []Candidate{{ID: "a", Actions: canon([]api.Action{fifo, term("E04", "2026-10-27")})}, {ID: "b", Actions: canon([]api.Action{term("E04", "2026-10-27"), fifo})}}
	if kept, _ := dedupe(x); len(kept) != 2 {
		t.Fatal("reordered batches were merged")
	}
}

// 5: different values stay distinct.
func TestSelectionV2DifferentValues(t *testing.T) {
	s := at(t, t0)
	for _, p := range [][]api.Action{
		{term("E04", "2026-10-27"), term("E04", "2026-11-10")},
		{term("E03", "2026-10-27"), term("E04", "2026-10-27")},
		{term("E04", "2026-10-27"), term("E04", "2026-10-27"), term("E04", "2026-11-10")},
	} {
		v1, v2 := mustSelect(t, s, p...), mustSelectV2(t, s, p...)
		sameButDuplicates(t, v1, v2)
		if len(p) == 2 && (len(v2.Removed) != 0 || len(v2.Candidates) != 4) {
			t.Fatalf("%v: distinct batches removed", ids(v2))
		}
	}
	// [E04 Oct 27, E04 Oct 27, E04 Nov 10]: without_1 = [Oct 27, Nov 10] =
	// without_2; without_3 = [Oct 27, Oct 27] is never reached in v1 either.
	p := []api.Action{term("E04", "2026-10-27"), term("E04", "2026-10-27"), term("E04", "2026-11-10")}
	if v2 := mustSelectV2(t, s, p...); !slices.Equal(ids(v2), []string{IDProposal, IDNoAction, "without_1"}) {
		t.Fatalf("%v", ids(v2))
	}
}

// 7: no action is always kept, in second place, empty. Only an earlier
// empty batch could remove it, and the proposal is never empty when the
// rule applies; a later empty batch would be removed in its favour.
func TestSelectionV2NoAction(t *testing.T) {
	s := at(t, t0)
	for _, p := range [][]api.Action{
		{term("E04", "2026-10-27")},
		{term("E04", "2026-10-27"), term("E04", "2026-10-27")},
		{term("E04", "2026-10-20")},
		{term("E02", "2026-10-27")},
	} {
		v2 := mustSelectV2(t, s, p...)
		if len(v2.Candidates) < 2 || v2.Candidates[1].ID != IDNoAction || len(v2.Candidates[1].Actions) != 0 || !v2.Candidates[1].OK {
			t.Fatalf("%v", ids(v2))
		}
	}
	if v2 := mustSelectV2(t, s); v2.Applies || len(v2.Candidates) != 0 {
		t.Fatal("an empty proposal listed candidates")
	}
	kept, removed := dedupe([]Candidate{{ID: IDProposal, Actions: canon([]api.Action{fifo})}, {ID: IDNoAction, Actions: canon(nil)}, {ID: "z", Actions: canon(nil)}})
	if len(kept) != 2 || kept[1].ID != IDNoAction || len(removed) != 1 || removed[0].SameAs != IDNoAction {
		t.Fatal("no action was not kept first")
	}
}

// 9 and 10: deterministic order and stable identities; a v2 selection
// replays only as v2.
func TestSelectionV2Stable(t *testing.T) {
	s := at(t, t0)
	p := []api.Action{term("E04", "2026-10-27"), term("E04", "2026-10-27")}
	v2 := mustSelectV2(t, s, p...)
	for range 3 {
		if again := mustSelectV2(t, s, p...); !jsonEqual(again, v2) {
			t.Fatal("v2 is not deterministic")
		}
	}
	if err := VerifySelection(s, v2); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(v2)
	var back Selection
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	if err := VerifySelection(s, &back); err != nil {
		t.Fatalf("a decoded v2 selection: %v", err)
	}
	// A v2 selection under v1's pin is refused by the rule check itself.
	pinned := *v2
	pinned.RuleSha256 = SelectionRuleSha256
	if err := VerifySelection(s, &pinned); err == nil || !strings.Contains(err.Error(), "is not this build's") {
		t.Fatalf("v1's pin on a v2 selection: %v", err)
	}
	for _, edit := range []func(*Selection){
		func(x *Selection) { x.Rule, x.RuleSha256 = SelectionRule, SelectionRuleSha256 },
		func(x *Selection) { x.RuleSha256 = SelectionRuleSha256 },
		func(x *Selection) { x.Removed = nil },
		func(x *Selection) { x.Candidates = append(slices.Clone(x.Candidates), x.Candidates[2]) },
		func(x *Selection) {
			x.Candidates = slices.Clone(x.Candidates)
			x.Candidates[1], x.Candidates[2] = x.Candidates[2], x.Candidates[1]
		},
	} {
		e := *v2
		edit(&e)
		if VerifySelection(s, &e) == nil {
			t.Fatal("an edited v2 selection replayed")
		}
	}
	// Identities: stable, distinct, and naming the rule.
	seen := map[string]bool{}
	for _, c := range v2.Candidates {
		want := sha(struct {
			Rule, ID string
			Actions  []api.Action
		}{SelectionRuleV2, c.ID, c.Actions})
		if c.Sha256 != want || seen[c.Sha256] {
			t.Fatalf("identity of %s", c.ID)
		}
		seen[c.Sha256] = true
	}
	// The agent audience under v2: built, decoded, bound; a comparison
	// that lists one batch twice under v2 is refused even when re-hashed.
	a := mustAgent(t, s, v2)
	if a.Selection.Rule != SelectionRuleV2 || a.Selection.RuleSha256 != SelectionRuleV2Sha256 {
		t.Fatal("the comparison does not name rule v2")
	}
	enc, err := a.Encode()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeAgentAt(enc, s, v2); err != nil {
		t.Fatal(err)
	}
	dup := *a
	dup.Alternatives = append(slices.Clone(a.Alternatives), a.Alternatives[2])
	dup.Alternatives[3].ID = "without_2"
	dup.Compared = compare(dup.Alternatives)
	dup.Sha256 = dup.digest()
	if err := dup.check(); !errors.Is(err, ErrContract) || !strings.Contains(err.Error(), "repeats") {
		t.Fatalf("a duplicate under v2: %v", err)
	}
	mixed := *a
	mixed.Selection.RuleSha256 = SelectionRuleSha256
	mixed.Sha256 = mixed.digest()
	if mixed.check() == nil {
		t.Fatal("rule v2 with v1's pin was accepted")
	}
}

// 11: v1 is unchanged: it still lists the duplicates, its selections and
// comparisons verify, and it never writes the removed field.
func TestSelectionV2KeepsV1(t *testing.T) {
	s := at(t, t0)
	p := []api.Action{term("E04", "2026-10-27"), term("E04", "2026-10-27")}
	v1 := mustSelect(t, s, p...)
	if v1.Rule != SelectionRule || v1.RuleSha256 != SelectionRuleSha256 || v1.Removed != nil || len(v1.Candidates) != 4 {
		t.Fatalf("v1 %v", ids(v1))
	}
	raw, _ := json.Marshal(v1)
	if strings.Contains(string(raw), `"removed"`) {
		t.Fatal("v1 writes the removed field")
	}
	if err := VerifySelection(s, v1); err != nil {
		t.Fatal(err)
	}
	for _, c := range v1.Candidates {
		want := sha(struct {
			Rule, ID string
			Actions  []api.Action
		}{SelectionRule, c.ID, c.Actions})
		if c.Sha256 != want {
			t.Fatalf("v1 identity of %s changed", c.ID)
		}
	}
	a := mustAgent(t, s, v1)
	enc, _ := a.Encode()
	if _, err := DecodeAgentAt(enc, s, v1); err != nil {
		t.Fatalf("a v1 comparison listing a duplicate: %v", err)
	}
	// A v2 selection with no duplicate lists v1's IDs, in v1's order.
	one := mustSelectV2(t, s, term("E04", "2026-10-27"))
	if !slices.Equal(ids(one), ids(mustSelect(t, s, term("E04", "2026-10-27")))) || one.Removed != nil {
		t.Fatal("v2 differs from v1 without duplicates")
	}
}
