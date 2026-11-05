package alternatives

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/session"
)

func mustSteps(t testing.TB, s *session.Session, p ...api.Action) (*Step, *Step, *Selection) {
	t.Helper()
	c, tr, sel, err := Steps(s, p)
	if err != nil {
		t.Fatal(err)
	}
	return c, tr, sel
}

func TestArmsPinnedTexts(t *testing.T) {
	if sha256Hex(controlText) != ControlSha256 {
		t.Fatalf("control template changed: %s", sha256Hex(controlText))
	}
	if StepSelectionRule != SelectionRuleV2 {
		t.Fatal("the arms do not use rule v2")
	}
}

func TestArmsTermination(t *testing.T) {
	s := at(t, t0)
	before := s.Digests()
	p := term("E04", "2026-10-27")
	c, tr, sel := mustSteps(t, s, p)
	if s.Digests() != before {
		t.Fatal("the steps changed the session")
	}
	if !c.Due || !tr.Due || c.Arm != ArmControl || tr.Arm != ArmTreatment || c.Rule != SelectionRuleV2 || tr.Rule != SelectionRuleV2 {
		t.Fatalf("%+v %+v", c, tr)
	}
	// Treatment: exactly the presentation of the bound agent audience.
	a := mustAgent(t, s, sel)
	want, err := Summary(a)
	if err != nil {
		t.Fatal(err)
	}
	if tr.Message != want || tr.ComparisonSha256 != a.Sha256 || tr.MessageSha256 != sha256Hex([]byte(want)) {
		t.Fatal("the treatment message is not the comparison's summary")
	}
	if len(tr.Inspect) != len(a.Alternatives) {
		t.Fatal("inspect answers")
	}
	for _, x := range a.Alternatives {
		d, _ := Detail(a, x.ID)
		if tr.Inspect[x.ID] != d {
			t.Fatalf("inspect:%s", x.ID)
		}
	}
	if !slices.Equal(tr.Allowed, []string{"select:proposal", "select:no_action", "select:adjusted", "revise", "abandon"}) {
		t.Fatalf("%v", tr.Allowed)
	}
	// Control: the proposal and its validation, nothing else.
	if !strings.Contains(c.Message, `[{"type":"terminate","employee":"E04","lastDay":"2026-10-27"}]`) || !strings.Contains(c.Message, "accepted") ||
		len(c.Inspect) != 0 || !slices.Equal(c.Allowed, []string{"select:proposal", "revise", "abandon"}) || c.ComparisonSha256 != "" {
		t.Fatalf("control %q", c.Message)
	}
	for _, w := range []string{"no_action", "adjusted", "$", "slot", "severance", "notice", "guarantee", "displaced", "irreversible", "obligation"} {
		if strings.Contains(strings.ToLower(c.Message), w) {
			t.Fatalf("control shows %q", w)
		}
	}
	// Deterministic.
	c2, tr2, _ := mustSteps(t, s, p)
	if !reflect.DeepEqual(c, c2) || !reflect.DeepEqual(tr, tr2) {
		t.Fatal("the steps are not deterministic")
	}
}

// The step is due by the same test in both arms.
func TestArmsNotDue(t *testing.T) {
	s := at(t, t0)
	for _, x := range []struct {
		p   []api.Action
		why string
	}{
		{[]api.Action{fifo}, StepNotDueRule},
		{nil, StepNotDueRule},
		{[]api.Action{term("E04", "2026-10-20")}, StepNotDueRefused},
		{[]api.Action{term("E04", "2026-10-27"), term("E04", "2026-10-27")}, StepNotDueRefused},
	} {
		c, tr, _ := mustSteps(t, s, x.p...)
		if c.Due || tr.Due || c.Why != x.why || tr.Why != x.why || c.Message != "" || tr.Message != "" || len(tr.Inspect) != 0 {
			t.Fatalf("%v: %+v %+v", x.p, c, tr)
		}
	}
}

// The boundary: nothing evaluation-only, no seed, no digest reaches a
// message, by construction and by the text check.
func TestArmsBoundary(t *testing.T) {
	s := at(t, t0)
	p := term("E04", "2026-10-27")
	_, tr, sel := mustSteps(t, s, p)
	all := tr.Message
	for _, d := range tr.Inspect {
		all += d
	}
	// The evaluation audience of the same decision point: none of its
	// projected labels is shown.
	rec, err := s.Record()
	if err != nil {
		t.Fatal(err)
	}
	ev, err := BuildEvaluation(rec, mustAgent(t, s, sel), session.BranchOptions{Continuation: session.NoFurtherAction()})
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, b := range ev.Branches {
		for _, v := range b.Projected {
			n++
			if strings.Contains(all, v.Label) {
				t.Fatalf("a projected item is shown: %s", v.Label)
			}
		}
	}
	if n == 0 {
		t.Fatal("no projected items to check")
	}
	for _, x := range []string{rec.Checksum, ev.Branches[0].Identity.SeedDigest, rec.Digests.State, rec.Digests.Run, tr.ComparisonSha256} {
		if x == "" || strings.Contains(all, x) {
			t.Fatalf("a digest is shown: %q", x)
		}
	}
	for _, bad := range []string{"seed 52", "deadbeefdeadbeef0", "Projected orders", "realized", "lost orders", "Branch"} {
		if !errors.Is(checkText("x "+bad), ErrBoundary) {
			t.Fatalf("checkText let %q through", bad)
		}
	}
	// Every message is checked: a banned word that the treatment text does
	// contain makes Steps refuse.
	saved := shownBanned
	shownBanned = append(slices.Clone(saved), "management alternatives")
	_, _, _, err = Steps(s, []api.Action{p})
	shownBanned = saved
	if !errors.Is(err, ErrBoundary) {
		t.Fatalf("the treatment text was not checked: %v", err)
	}
	shownBanned = append(slices.Clone(saved), "nothing has been applied")
	_, _, _, err = Steps(s, []api.Action{p})
	shownBanned = saved
	if !errors.Is(err, ErrBoundary) {
		t.Fatalf("the control text was not checked: %v", err)
	}
	shownBanned = append(slices.Clone(saved), "obligations that continue:") // only in the inspect answers
	_, _, _, err = Steps(s, []api.Action{p})
	shownBanned = saved
	if !errors.Is(err, ErrBoundary) {
		t.Fatalf("the inspect answers were not checked: %v", err)
	}
	// The step's type holds no evaluation value.
	for _, f := range reflect.VisibleFields(reflect.TypeOf(Step{})) {
		if f.Type == reflect.TypeOf(Evaluation{}) || f.Type == reflect.TypeOf(&Evaluation{}) || f.Type == reflect.TypeOf([]Value{}) {
			t.Fatalf("Step.%s can hold evaluation values", f.Name)
		}
	}
}

func TestArmsClassifyControl(t *testing.T) {
	s := at(t, t0)
	p := []api.Action{term("E04", "2026-10-27")}
	c, tr, _ := mustSteps(t, s, p...)
	rv := func(b []api.Action) []api.ActionResult { return s.ValidateBatch(b...) }
	for _, x := range []struct {
		reply  *Reply
		result string
		sub    bool
	}{
		{&Reply{Kind: ReplySelect, ID: IDProposal}, ResultRetained, true},
		{&Reply{Kind: ReplyRevise, Actions: p}, ResultRetained, true},
		{&Reply{Kind: ReplyRevise, Actions: []api.Action{term("E04", "2026-11-10")}}, ResultRevised, true},
		{&Reply{Kind: ReplyRevise, Actions: []api.Action{term("E04", "2026-10-20")}}, ResultInvalidAlterative, false},
		{&Reply{Kind: ReplyAbandon}, ResultAbandoned, false},
		{&Reply{Kind: ReplySelect, ID: IDNoAction}, ResultInvalidAlterative, false},
		{&Reply{Kind: ReplySelect, ID: IDAdjusted}, ResultInvalidAlterative, false},
		{&Reply{Kind: ReplySelect, ID: IDProposal, Actions: p}, ResultNoDecision, false},
		{&Reply{Kind: ReplyAbandon, ID: IDProposal}, ResultNoDecision, false},
		{&Reply{Kind: "submit"}, ResultNoDecision, false},
		{nil, ResultNoDecision, false},
	} {
		d := ClassifyControl(c, []string{"inspect:no_action"}, x.reply, rv)
		if d.Result != x.result || (len(d.Submitted) > 0) != x.sub || !slices.Equal(d.Flags, []string{FlagNoComparison}) || d.ComparisonSha256 != "" {
			t.Fatalf("%+v: %+v", x.reply, d)
		}
	}
	if d := ClassifyControl(tr, nil, &Reply{Kind: ReplySelect, ID: IDProposal}, rv); d.Result != ResultNoDecision {
		t.Fatal("a treatment turn classified as control")
	}
	if !NoActionSubmitted(ResultAbandoned) || !NoActionSubmitted(ResultSelectedNoAction) || NoActionSubmitted(ResultRevised) {
		t.Fatal("NoActionSubmitted")
	}
}
