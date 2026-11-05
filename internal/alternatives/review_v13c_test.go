package alternatives

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/agent"
	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/session"
)

// Regression tests for the V0.13c independent review
// (docs/audits/v0.13c-independent-review.md): each was a witness of a
// defect and now requires the fixed behaviour.

// F1 (the part fixed in code): a digest in upper or mixed case is refused
// like a lowercase one, in the turn text and in every inspect answer. The
// rest of F1 (meaningful text in a valid free-text field) is not a lexical
// matter; it awaits the owner's threat-model decision (D7).
func TestReviewF1DigestAnyCase(t *testing.T) {
	s := at(t, t0)
	for _, payload := range []string{strings.ToUpper(s.Digests().State), "AbCdEf0123456789AbCd"} {
		p := tele("weeknights")
		p.RequestID = payload
		if _, _, _, err := Steps(s, []api.Action{p}); !errors.Is(err, ErrBoundary) {
			t.Fatalf("requestId %q: %v", payload, err)
		}
	}
	if !errors.Is(checkText("x DEADBEEFDEADBEEF0"), ErrBoundary) {
		t.Fatal("an uppercase digest-like run passed")
	}
}

// F3: a forged candidate identity is refused structurally.
func TestReviewF3CandidateIdentity(t *testing.T) {
	s := at(t, t0)
	a := mustAgent(t, s, mustSelectV2(t, s, term("E04", "2026-10-27")))
	a.Alternatives[0].CandidateSha256 = "forged"
	a.Sha256 = a.digest()
	raw, _ := a.Encode()
	if _, err := DecodeAgent(raw); !errors.Is(err, ErrContract) || !strings.Contains(err.Error(), "candidate identity") {
		t.Fatalf("forged identity: %v", err)
	}
}

// F4: anything after the one JSON value is refused, a stray closing
// delimiter included, by both decoders.
func TestReviewF4TrailingData(t *testing.T) {
	s := at(t, t0)
	sel := mustSelectV2(t, s, term("E04", "2026-10-27"))
	a := mustAgent(t, s, sel)
	raw, _ := a.Encode()
	if _, err := DecodeAgentAt(raw, s, sel); err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{"]", "}", "] garbage", "{}", "\n{}"} {
		if _, err := DecodeAgentAt(append(slices.Clone(raw), suffix...), s, sel); !errors.Is(err, ErrContract) {
			t.Fatalf("suffix %q accepted", suffix)
		}
	}
	// Trailing white space is not data.
	if _, err := DecodeAgent(append(slices.Clone(raw), " \n\t"...)); err != nil {
		t.Fatal(err)
	}
	rec, err := s.Record()
	if err != nil {
		t.Fatal(err)
	}
	ev, err := BuildEvaluation(rec, a, session.BranchOptions{Continuation: session.NoFurtherAction()})
	if err != nil {
		t.Fatal(err)
	}
	eraw, _ := ev.Encode()
	if _, err := DecodeEvaluation(eraw); err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeEvaluation(append(slices.Clone(eraw), "]"...)); !errors.Is(err, ErrContract) {
		t.Fatal("an evaluation with a stray ']' decoded")
	}
}

// F5: the transcript feed checks the point's own read-only requests that
// follow the digests answer, as the Python to_point does.
func TestReviewF5PointTail(t *testing.T) {
	want := at(t, t0).Digests()
	write := func(lines ...string) string {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "run.json"), []byte(`{"scenario":{"ref":"mini"},"seed":52}`), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "transcript.jsonl"), []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		return dir
	}
	fresh, err := session.Open("mini", 52)
	if err != nil {
		t.Fatal(err)
	}
	h := agent.New(fresh)
	adv := `{"id":"a","op":"advance","args":{"to":"2026-10-13T07:00:00"}}`
	advAns := string(h.Handle([]byte(adv)))
	dig := `{"id":"d","op":"digests"}`
	digAns := string(h.Handle([]byte(dig)))
	obs := `{"id":"o","op":"observe"}`
	obsAns := string(h.Handle([]byte(obs)))
	later := `{"id":"x","op":"advance","args":{"to":"2026-10-14T07:00:00"}}`
	bad := `{"protocol":"dreamteam.agent/v1","id":"o","ok":true,"result":{"observation":"CORRUPTED"}}`
	if _, err := atPointErr(write(adv, advAns, dig, digAns, obs, bad), want); err == nil || !strings.Contains(err.Error(), "observe") {
		t.Fatalf("a corrupt observation after the point was accepted: %v", err)
	}
	// A genuine tail is fed and the state is the point's; a later
	// non-read-only request is not fed.
	s, err := atPointErr(write(adv, advAns, dig, digAns, obs, obsAns, later, `{"id":"x","ok":true}`), want)
	if err != nil || s.Digests() != want {
		t.Fatalf("%v", err)
	}
}

// F6: the turn is due only when the engine accepts the reply's operations
// batch as well as its staffing batch, and the inventory agrees.
func TestReviewF6WholeReply(t *testing.T) {
	s := at(t, t0)
	staff := term("E04", "2026-10-27")
	badOp := api.Action{Type: api.ActSetWorklistPolicy, Policy: "not-a-policy"}
	c, tr, _ := mustSteps(t, s, badOp, staff)
	if c.Due || tr.Due || c.Why != StepNotDueOps || tr.Why != StepNotDueOps {
		t.Fatalf("refused operations: %s %s", c.Why, tr.Why)
	}
	c, tr, sel := mustSteps(t, s, fifo, staff)
	if !c.Due || !tr.Due || !jsonEqual(c.Proposal, []api.Action{staff}) || !jsonEqual(c.Operations, []api.Action{fifo}) ||
		!jsonEqual(sel.Proposal, []api.Action{staff}) {
		t.Fatalf("accepted operations: %s %+v", c.Why, c.Operations)
	}
	// The comparison is of the staffing batch: the same as without the
	// operations action.
	_, alone, _ := mustSteps(t, s, staff)
	if tr.Message != alone.Message {
		t.Fatal("the operations batch changed the comparison")
	}
	cl := contextLine{Validation: "refused"}
	cl.Proposal.Actions = []api.Action{badOp, staff}
	row := inventoryRow(t, s, poolEntry{ID: "review-f6", Validation: "refused"}, cl)
	if row.Valid || row.StepDue || row.StepWhy != StepNotDueOps {
		t.Fatalf("inventory row %+v", row)
	}
}

// F8: the control classification submits the batch its step shows, never
// a batch supplied separately.
func TestReviewF8ControlBound(t *testing.T) {
	s := at(t, t0)
	c, _, _ := mustSteps(t, s, term("E04", "2026-10-27"))
	rv := func(b []api.Action) []api.ActionResult { return s.ValidateBatch(b...) }
	d := ClassifyControl(c, nil, &Reply{Kind: ReplySelect, ID: IDProposal}, rv)
	if d.Result != ResultRetained || len(d.Submitted) != 1 || d.Submitted[0].Employee != "E04" {
		t.Fatalf("%+v", d)
	}
	edited := *c
	edited.Proposal = nil
	if d := ClassifyControl(&edited, nil, &Reply{Kind: ReplySelect, ID: IDProposal}, rv); d.Result != ResultNoDecision || len(d.Submitted) != 0 {
		t.Fatal("a control step with no proposal submitted something")
	}
}
