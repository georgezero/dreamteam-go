package alternatives

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"slices"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/session"
)

// The alternatives turn as agent-protocol operations (V0.13d), served by
// dreamteam agent --stdio -alternatives and nowhere else:
//
//   - alternatives.turn {arm, actions}: the arm's final-decision turn for
//     the reply's actions at the session's current time (Turns);
//   - alternatives.decide {arm, actions, inspected, round, content}: the
//     model's reply to the turn, read (ParseTurnReply), and either the
//     fixed text to send next (an inspect answer, or the retry text) or the
//     classified outcome (Decide).
//
// Both are read-only: the agent handler checks the session's digests
// unchanged around every extension op. Every text either returns comes
// from presentation v2 and passes the boundary check. A workflow sends
// those texts verbatim and nothing else from the answers.

// ReplySchema identifies alternatives.decide's answer.
const ReplySchema = "dreamteam.alternatives-reply/v1"

// Reply answer kinds.
const (
	AnswerDecision     = "decision"
	AnswerInspect      = "inspect"
	AnswerInspectAgain = "inspect_again"
	AnswerUnreadable   = "unreadable"
)

// TurnAnswer is alternatives.decide's answer.
type TurnAnswer struct {
	Schema       string    `json:"schema"`
	Kind         string    `json:"kind"`
	Parse        TurnReply `json:"parse"`
	Answer       string    `json:"answer"`
	AnswerSha256 string    `json:"answerSha256"`
	Inspected    []string  `json:"inspected"`
	Outcome      *Outcome  `json:"outcome,omitempty"`
}

// Ops are the extension's operations, for agent.Handler.Extensions.
func Ops() map[string]func(*session.Session, func([]api.Action) ([]api.ActionResult, error), json.RawMessage) (any, error) {
	return map[string]func(*session.Session, func([]api.Action) ([]api.ActionResult, error), json.RawMessage) (any, error){
		"alternatives.turn":   opTurn,
		"alternatives.decide": opDecide,
	}
}

func strict(raw json.RawMessage, v any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("trailing data")
	}
	return nil
}

func armTurn(s *session.Session, v Validator, arm string, actions []api.Action) (*Turn, error) {
	c, t, err := TurnsWith(s, v, actions)
	if err != nil {
		return nil, err
	}
	switch arm {
	case ArmControl:
		return c, nil
	case ArmTreatment:
		return t, nil
	}
	return nil, fmt.Errorf("arm %q is not control or treatment", arm)
}

func opTurn(s *session.Session, v func([]api.Action) ([]api.ActionResult, error), raw json.RawMessage) (any, error) {
	var a struct {
		Arm     string       `json:"arm"`
		Actions []api.Action `json:"actions"`
	}
	if err := strict(raw, &a); err != nil {
		return nil, err
	}
	if len(a.Actions) == 0 {
		return nil, fmt.Errorf("actions: the reply's actions are required")
	}
	return armTurn(s, v, a.Arm, a.Actions)
}

func renderText(name string, d any) (string, error) {
	var b bytes.Buffer
	if err := presentationV2.ExecuteTemplate(&b, name, d); err != nil {
		return "", err
	}
	if err := checkText(b.String()); err != nil {
		return "", err
	}
	return b.String(), nil
}

func opDecide(s *session.Session, v func([]api.Action) ([]api.ActionResult, error), raw json.RawMessage) (any, error) {
	var a struct {
		Arm       string       `json:"arm"`
		Actions   []api.Action `json:"actions"`
		Inspected []string     `json:"inspected"`
		Round     int          `json:"round"` // inspect answers already sent
		Content   string       `json:"content"`
	}
	if err := strict(raw, &a); err != nil {
		return nil, err
	}
	t, err := armTurn(s, v, a.Arm, a.Actions)
	if err != nil {
		return nil, err
	}
	if !t.Due {
		return nil, fmt.Errorf("no turn is due (%s)", t.Why)
	}
	for _, id := range a.Inspected {
		if _, ok := t.Inspect[id]; !ok {
			return nil, fmt.Errorf("inspected: %q is not a listed alternative of the arm", id)
		}
	}
	ans := &TurnAnswer{Schema: ReplySchema, Parse: ParseTurnReply(a.Content), Inspected: []string{}}
	switch ans.Parse.Kind {
	case ReplyUnreadable:
		ans.Kind = AnswerUnreadable
		ans.Answer, err = renderText("retry", nil)
	case ReplyInspect:
		switch {
		case a.Round > 0:
			ans.Kind = AnswerInspectAgain
			ans.Answer, err = renderText("inspect_again", nil)
		case a.Arm == ArmControl:
			ans.Kind = AnswerInspect
			ans.Answer, err = renderText("inspect_control", nil)
		default:
			ans.Kind = AnswerInspect
			var details []string
			for _, id := range ans.Parse.Inspect {
				if d, ok := t.Inspect[id]; ok && !slices.Contains(ans.Inspected, id) {
					ans.Inspected = append(ans.Inspected, id)
					details = append(details, d)
				}
			}
			if len(details) == 0 {
				ans.Answer, err = renderText("inspect_unlisted", nil)
			} else {
				ans.Answer, err = renderText("inspect_answer", details)
			}
		}
	default:
		ans.Kind = AnswerDecision
		ans.Outcome, err = DecideWith(s, v, a.Arm, a.Actions, a.Inspected, ans.Parse.Reply)
	}
	if err != nil {
		return nil, err
	}
	ans.AnswerSha256 = sha256Hex([]byte(ans.Answer))
	return ans, nil
}
