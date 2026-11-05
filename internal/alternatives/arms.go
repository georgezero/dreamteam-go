package alternatives

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"text/template"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/session"
)

// The V0.13c experiment's final-decision turn (EXPERIMENTAL_DESIGN.md),
// built by the engine for both arms from the same decision point:
//
//   - the step is due when rule v2 applies to the proposal and the engine
//     accepts the proposal, the same test in both arms;
//   - control shows a fixed turn naming the proposal and its validation
//     only (text/control-v1.tmpl);
//   - treatment shows the agent audience of the comparison, through the
//     pinned presentation template, and answers inspect:<id> with that
//     alternative's detail.
//
// The treatment's text is made only from an *Agent that DecodeAgentAt has
// rebuilt from the decision-point session: the type has no field for a
// projection, and the evaluation audience is never built. The text is
// checked again before it is returned (checkShown). No model is contacted
// here; a driver sends Message and the inspect answers, and nothing else.

const (
	ArmsID             = "dreamteam.alternatives-arms/v1"
	ArmControl         = "control"
	ArmTreatment       = "treatment"
	ControlID          = "dreamteam.alternatives-control/v1"
	ControlSha256      = "331a1542ed79e15fcf10965fb924b134cd5ca206f740be96a51d4ce49f65d5a9"
	StepSelectionRule  = SelectionRuleV2
	StepNotDueRule     = "not_due:rule"    // rule v2 does not apply to the proposal
	StepNotDueRefused  = "not_due:refused" // the engine refuses the proposal; validation feedback comes first
	StepNotDueNoSelect = "not_due:no_alternatives"
)

//go:embed text/control-v1.tmpl
var controlText []byte

var controlTurn = template.Must(template.New("control").Funcs(template.FuncMap{"json": jsonText}).Parse(string(controlText)))

// Step is the final-decision turn of one arm at one decision point.
type Step struct {
	Arms string `json:"arms"`
	Arm  string `json:"arm"`
	// Due: the turn is shown. When it is not, both arms submit the
	// validated proposal as the validation-first workflow does, and Why
	// says why.
	Due bool   `json:"due"`
	Why string `json:"why"`
	// Message is the exact text the model is sent ("" when not due).
	Message string `json:"message"`
	// Inspect answers inspect:<id> (treatment only), by alternative ID.
	Inspect map[string]string `json:"inspect"`
	// Allowed are the replies the turn accepts: select:<id>, revise,
	// abandon.
	Allowed []string `json:"allowed"`
	// For the record, never sent: the rule, the template and the
	// comparison the text was rendered from.
	Rule             string `json:"rule"`
	RuleSha256       string `json:"ruleSha256"`
	Template         string `json:"template"`
	TemplateSha256   string `json:"templateSha256"`
	ComparisonSha256 string `json:"comparisonSha256,omitempty"`
	MessageSha256    string `json:"messageSha256"`
}

// Steps builds both arms' turns for proposal at s's current time, from one
// selection. s is only read (its digests are checked unchanged).
func Steps(s *session.Session, proposal []api.Action) (control, treatment *Step, sel *Selection, err error) {
	before := s.Digests()
	sel, err = SelectV2(s, proposal)
	if err != nil {
		return nil, nil, nil, err
	}
	o := s.Observe(0)
	mk := func(arm, tmpl, pin string) *Step {
		return &Step{Arms: ArmsID, Arm: arm, Inspect: map[string]string{}, Allowed: []string{}, Rule: sel.Rule, RuleSha256: sel.RuleSha256,
			Template: tmpl, TemplateSha256: pin}
	}
	control, treatment = mk(ArmControl, ControlID, ControlSha256), mk(ArmTreatment, PresentationID, PresentationSha256)
	why := ""
	switch {
	case !sel.Applies:
		why = StepNotDueRule
	case len(sel.Candidates) < 2:
		why = StepNotDueNoSelect
	case !sel.Candidates[0].OK:
		why = StepNotDueRefused
	}
	if why != "" {
		control.Why, treatment.Why = why, why
		control.MessageSha256, treatment.MessageSha256 = sha256Hex(nil), sha256Hex(nil)
		return control, treatment, sel, digestsUnchanged(s, before)
	}
	// Control: the proposal and its validation.
	var b bytes.Buffer
	if err := controlTurn.Execute(&b, struct {
		Time    string
		Actions []api.Action
	}{o.Time, sel.Candidates[0].Actions}); err != nil {
		return nil, nil, nil, err
	}
	control.Due, control.Why, control.Message = true, "due", b.String()
	control.Allowed = []string{"select:" + IDProposal, ReplyRevise, ReplyAbandon}
	// Treatment: the comparison, decoded and bound to the session.
	a, err := BuildAgent(s, sel)
	if err != nil {
		return nil, nil, nil, err
	}
	raw, err := a.Encode()
	if err != nil {
		return nil, nil, nil, err
	}
	bound, err := DecodeAgentAt(raw, s, sel)
	if err != nil {
		return nil, nil, nil, err
	}
	if treatment.Message, err = Summary(bound); err != nil {
		return nil, nil, nil, err
	}
	treatment.Due, treatment.Why, treatment.ComparisonSha256 = true, "due", bound.Sha256
	for _, x := range bound.Alternatives {
		d, err := Detail(bound, x.ID)
		if err != nil {
			return nil, nil, nil, err
		}
		treatment.Inspect[x.ID] = d
		treatment.Allowed = append(treatment.Allowed, "select:"+x.ID)
	}
	treatment.Allowed = append(treatment.Allowed, ReplyRevise, ReplyAbandon)
	for _, st := range []*Step{control, treatment} {
		if err := st.checkShown(); err != nil {
			return nil, nil, nil, err
		}
		st.MessageSha256 = sha256Hex([]byte(st.Message))
	}
	return control, treatment, sel, digestsUnchanged(s, before)
}

// jsonText is the presentation template's json function.
func jsonText(v any) (string, error) { b, err := json.Marshal(v); return string(b), err }

func digestsUnchanged(s *session.Session, before session.Digests) error {
	if s.Digests() != before {
		return fmt.Errorf("%w: building the decision turn changed the session", ErrContract)
	}
	return nil
}

// The information boundary, checked on every text a step would send:
// nothing that names a seed, a digest, a branch, a projection or a
// realized outcome. The structure is the first guard (only the agent
// audience is rendered); this is the second.
var (
	hexRun      = regexp.MustCompile(`[0-9a-f]{16,}`)
	shownBanned = []string{"seed", "digest", "sha256", "branch", "projection", "projected orders", "projected outcome", "realized", "evaluation", "continuation",
		"orders lost", "lost orders", "orderslost", "profit", "future outcome", "hidden"}
)

// ErrBoundary: a step's text would show something outside the agent
// audience.
var ErrBoundary = fmt.Errorf("alternatives: information boundary")

func checkText(text string) error {
	if m := hexRun.FindString(text); m != "" {
		return fmt.Errorf("%w: a digest-like string %q", ErrBoundary, m)
	}
	low := strings.ToLower(text)
	for _, w := range shownBanned {
		if strings.Contains(low, w) {
			return fmt.Errorf("%w: %q", ErrBoundary, w)
		}
	}
	return nil
}

func (st *Step) checkShown() error {
	if err := checkText(st.Message); err != nil {
		return err
	}
	for id, d := range st.Inspect {
		if err := checkText(d); err != nil {
			return fmt.Errorf("inspect:%s: %w", id, err)
		}
	}
	return nil
}

// ClassifyControl classifies a reply to the control arm's turn, with the
// results Classify uses: select:proposal is retained (the proposal
// submitted after final validation), a revision to the same batch is
// retained, a different batch revised, abandon abandoned, any other
// select an invalid alternative, and a missing or contradictory reply no
// decision. Nothing is defaulted. The control turn shows no comparison, so
// its only flag is no_observable_comparison; an inspect request is
// recorded and has no answer.
func ClassifyControl(st *Step, proposal []api.Action, requests []string, reply *Reply, revalidate func([]api.Action) []api.ActionResult) Decision {
	d := Decision{Requests: append([]string{}, requests...), Reply: reply, Flags: []string{FlagNoComparison},
		CitationChecks: []string{}, Submitted: []api.Action{}, Revalidation: []api.ActionResult{}}
	submit := func(batch []api.Action, result, why string) {
		d.Revalidation = revalidate(batch)
		for _, r := range d.Revalidation {
			if !r.OK {
				d.Result, d.Why = ResultInvalidAlterative, "the engine refuses the batch at final validation"
				if result == ResultRetained {
					d.Alternative = IDProposal
				}
				return
			}
		}
		d.Result, d.Why, d.Submitted = result, why, canon(batch)
		if result == ResultRetained {
			d.Alternative = IDProposal
		}
	}
	switch {
	case st == nil || !st.Due || st.Arm != ArmControl:
		d.Result, d.Why = ResultNoDecision, "no control turn was shown"
	case reply == nil:
		d.Result, d.Why = ResultNoDecision, "no usable reply"
	case reply.Kind == ReplySelect && (reply.ID == "" || len(reply.Actions) > 0):
		d.Result, d.Why = ResultNoDecision, "contradictory select: needs an ID and no batch"
	case reply.Kind == ReplySelect && reply.ID == IDProposal:
		submit(proposal, ResultRetained, "the proposal, unchanged")
	case reply.Kind == ReplySelect:
		d.Result, d.Alternative, d.Why = ResultInvalidAlterative, reply.ID, "names an alternative the control turn does not list"
	case reply.Kind == ReplyRevise && (reply.ID != "" || len(reply.Actions) == 0):
		d.Result, d.Why = ResultNoDecision, "contradictory revise: needs a batch and no ID"
	case reply.Kind == ReplyRevise && jsonEqual(canon(reply.Actions), canon(proposal)):
		submit(proposal, ResultRetained, "the proposal, unchanged (submitted as a revision)")
	case reply.Kind == ReplyRevise:
		submit(reply.Actions, ResultRevised, "a new batch, different from the proposal")
	case reply.Kind == ReplyAbandon && (reply.ID != "" || len(reply.Actions) > 0):
		d.Result, d.Why = ResultNoDecision, "contradictory abandon: takes no ID and no batch"
	case reply.Kind == ReplyAbandon:
		d.Result, d.Why = ResultAbandoned, "abandoned; nothing submitted"
	default:
		d.Result, d.Why = ResultNoDecision, "unknown reply kind"
	}
	return d
}

// NoActionSubmitted is the outcome both arms share for "nothing
// submitted by choice": abandoned (either arm) or no_action selected
// (treatment).
func NoActionSubmitted(result string) bool {
	return result == ResultAbandoned || result == ResultSelectedNoAction
}
