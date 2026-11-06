package alternatives

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
)

func TestV13eFinalActionBoundaries(t *testing.T) {
	s := at(t, t0)
	ops := api.Action{Type: api.ActSetWalkInPolicy, Policy: "schedule_later"}
	p := []api.Action{ops, term("E04", "2026-10-27")}
	expected := []string{"original_retained", "listed_alternative", "no_staffing_action", "other_batch", "abandoned", "invalid", "no_decision"}
	if !slices.Equal(FinalActions, expected) {
		t.Fatal(FinalActions)
	}
	for _, arm := range []string{ArmControl, ArmTreatment} {
		cases := []struct {
			r    *Reply
			want string
		}{
			{reply(ReplySelect, IDProposal), FinalRetained},
			{reply(ReplyRevise, "", ops, term("E04", "2026-11-10")), FinalListed},
			{reply(ReplyRevise, "", ops), FinalNoStaffingAction},
			{reply(ReplyRevise, "", term("E04", "2026-10-27")), FinalOther},
			{reply(ReplyRevise, "", ops, term("E04", "2026-11-24")), FinalOther},
			{reply(ReplyAbandon, ""), FinalAbandoned},
			{reply(ReplyRevise, "", ops, term("E04", "2026-10-14")), FinalInvalid},
			{nil, FinalNoDecision},
		}
		seen := map[string]bool{}
		before := s.Digests()
		for _, c := range cases {
			o := mustDecide(t, s, arm, p, nil, c.r)
			if o.Final != c.want {
				t.Fatalf("%s: got %s want %s", arm, o.Final, c.want)
			}
			seen[o.Final] = true
			if o.Final == FinalNoStaffingAction && (len(o.Batches) != 1 || !jsonEqual(o.Operations, canon([]api.Action{ops}))) {
				t.Fatal("no staffing must preserve operations")
			}
		}
		if len(seen) != len(expected) {
			t.Fatal("category coverage", seen)
		}
		if s.Digests() != before {
			t.Fatal("classification executed an action")
		}
	}
}

func TestV13eCitationBoundaries(t *testing.T) {
	s := at(t, t0)
	raw := func(v string) json.RawMessage { b, _ := json.Marshal(v); return b }
	p := []api.Action{term("E04", "2026-10-27")}
	for _, c := range []struct {
		name, value string
		inspected   []string
		want        bool
	}{
		{"exact delivered", "2,481", []string{IDProposal, IDNoAction}, true},
		{"wrong", "2,480", []string{IDProposal, IDNoAction}, false},
		{"paraphrase", "about 2481 slots", []string{IDProposal, IDNoAction}, false},
		{"undelivered", "2,481", []string{IDNoAction}, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			r := reply(ReplySelect, IDProposal)
			r.Comparison = []Citation{{Item: "slots_closed", Values: map[string]json.RawMessage{IDProposal: raw(c.value), IDNoAction: raw("0")}}}
			o := mustDecide(t, s, ArmTreatment, p, c.inspected, r)
			if slices.Contains(o.Flags, FlagCompared) != c.want {
				t.Fatal(o.CitationChecks)
			}
		})
	}
	// A short, fully copied table must fail even with exact delivered values.
	b, err := build(s, SessionValidator(s), p)
	if err != nil {
		t.Fatal(err)
	}
	for i := range b.view.Alternatives {
		x := &b.view.Alternatives[i]
		x.Estimates = nil
		x.Obligations = nil
		var facts []ViewItem
		for _, item := range x.Facts {
			if item.Key == "slots_closed" {
				facts = append(facts, item)
			}
		}
		x.Facts = facts
	}
	cs := []Citation{{Item: "slots_closed", Values: map[string]json.RawMessage{IDProposal: raw("2,481"), IDNoAction: raw("0")}}}
	if ok, _ := verifyViewCitations(b.view, []string{IDProposal, IDNoAction}, cs, IDProposal); ok {
		t.Fatal("copied table counted")
	}
}
