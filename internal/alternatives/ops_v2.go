package alternatives

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/session"
)

type TurnAnswerV2 struct {
	Schema       string     `json:"schema"`
	Kind         string     `json:"kind"`
	Parse        TurnReply  `json:"parse"`
	Answer       string     `json:"answer"`
	AnswerSha256 string     `json:"answerSha256"`
	Inspected    []string   `json:"inspected"`
	Outcome      *OutcomeV2 `json:"outcome,omitempty"`
}

// OpsV2 preserves every historical operation and adds versioned successors.
func OpsV2() map[string]func(*session.Session, func([]api.Action) ([]api.ActionResult, error), json.RawMessage) (any, error) {
	ops := Ops()
	ops["alternatives.v2.turn"] = opTurnV2
	ops["alternatives.v2.decide"] = opDecideV2
	ops["alternatives.v2.canonical"] = func(s *session.Session, v func([]api.Action) ([]api.ActionResult, error), raw json.RawMessage) (any, error) {
		var a struct {
			Batches [][]api.Action `json:"batches"`
		}
		if err := strict(raw, &a); err != nil {
			return nil, err
		}
		return CanonicalSubmitted(a.Batches), nil
	}
	return ops
}

func opTurnV2(s *session.Session, v func([]api.Action) ([]api.ActionResult, error), raw json.RawMessage) (any, error) {
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
	return armTurnV2(s, v, a.Arm, a.Actions)
}

func opDecideV2(s *session.Session, v func([]api.Action) ([]api.ActionResult, error), raw json.RawMessage) (any, error) {
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
	t, err := armTurnV2(s, v, a.Arm, a.Actions)
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
	ans := &TurnAnswerV2{Schema: "dreamteam.alternatives-reply/v2", Parse: ParseTurnReplyV2(a.Content), Inspected: []string{}}
	switch ans.Parse.Kind {
	case ReplyUnreadable:
		ans.Kind = AnswerUnreadable
		ans.Answer, err = renderV3("retry", nil)
	case ReplyInspect:
		switch {
		case a.Round > 0:
			ans.Kind = AnswerInspectAgain
			ans.Answer, err = renderV3("inspect_again", nil)
		case a.Arm == ArmControl:
			ans.Kind = AnswerInspect
			ans.Answer, err = renderV3("inspect_control", nil)
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
				ans.Answer, err = renderV3("inspect_unlisted", nil)
			} else {
				ans.Answer, err = renderV3("inspect_answer", details)
			}
		}
	default:
		ans.Kind = AnswerDecision
		ans.Outcome, err = DecideV2With(s, v, a.Arm, a.Actions, a.Inspected, ans.Parse.Reply)
	}
	if err != nil {
		return nil, err
	}
	ans.AnswerSha256 = sha256Hex([]byte(ans.Answer))
	return ans, nil
}
