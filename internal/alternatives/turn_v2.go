package alternatives

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"text/template"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/session"
)

const (
	ArmsV3ID             = "dreamteam.alternatives-arms/v3"
	PresentationV3ID     = "dreamteam.alternatives-presentation/v3"
	ParserV2ID           = "dreamteam.phase-response-parser/v2"
	PresentationV3Sha256 = "19caebcbfc53fd2d36fea9c08851a5607cf6bf258cb496c491e6e2de296f7640"
	ViewV2TableSha256    = "f96e2c4f9875b1b2593b09340bd6b752aa73ac7d7486886a53548c4c34bd5282"
)

//go:embed text/presentation-v3.tmpl
var presentationV3Text []byte

var reversalTextsV2 = map[string]string{
	"termination":   "a committed termination cannot be withdrawn or moved; no action rehires the employee",
	"hire":          "the recruiting fee is non-refundable; ending employment requires lawful notice and severance",
	"roster_remove": "a removal is not undone by another removal; restoring a version needs a new set under the same 14-day notice and validation rules",
	"roster":        "removal or replacement of a pending roster also requires 14 days notice; the version must still be pending and the employee employed",
	"allocation":    "a pending allocation may be withdrawn only within the engine notice rule",
	"tele_start":    "activation binds the minimum term; cancellation requires an end at least 30 days after activation and at least 30 days after the cancellation decision",
	"tele_cancel":   "the fixed exclusive contract end cannot be moved or withdrawn; claims in flight complete at locked fees",
}

func reversalTextV2(r ReversalV2) string {
	text := reversalTextsV2[r.Kind]
	if (r.Kind == "roster" || r.Kind == "roster_remove") || r.Kind == "allocation" {
		if r.Deadline == "" {
			text += "; exact withdrawal deadline unavailable"
		} else {
			text += "; last notice-compliant withdrawal date: " + r.Deadline + " (inclusive), subject to the other validation conditions"
		}
	}
	return text
}

var presentationV3 = template.Must(template.New("p3").Funcs(template.FuncMap{
	"refText": refText, "actions": func(x ViewAltV2) string { return viewActions(x.ViewAlt) },
	"validation": func(x ViewAltV2) string { return viewValidation(x.ViewAlt) }, "line": viewLine,
	"unavailable": viewUnavailableText, "cents": centsTextV2, "quantity": quantityTextV2,
	"reversal": reversalTextV2, "proxyMoney": money, "windows": windowLabel,
}).Parse(string(presentationV3Text)))

type turnDataV2 struct {
	Time         string
	Refs         []Ref
	Alternatives []ViewAltV2
	Operations   int
	IDs          []string
}

func renderV3(name string, d any) (string, error) {
	var b bytes.Buffer
	if err := presentationV3.ExecuteTemplate(&b, name, d); err != nil {
		return "", err
	}
	if err := checkText(b.String()); err != nil {
		return "", err
	}
	return b.String(), nil
}

func TurnsV2With(s *session.Session, validator Validator, reply []api.Action) (*Turn, *Turn, error) {
	before := s.Digests()
	b, err := build(s, validator, reply)
	if err != nil {
		return nil, nil, err
	}
	c, tr, err := b.turns()
	if err != nil {
		return nil, nil, err
	}
	for _, t := range []*Turn{c, tr} {
		t.Schema = "dreamteam.alternatives-turn/v2"
		t.Arms = ArmsV3ID
		t.Record.Presentation = PresentationV3ID
		t.Record.PresentationSha256 = PresentationV3Sha256
		t.Record.ViewSchema = ViewV2Schema
		t.Record.ViewTableSha256 = ViewV2TableSha256
	}
	if !c.Due {
		return c, tr, nil
	}
	v, m, err := BuildViewV2(s, b.agent, b.pos, len(b.ops))
	if err != nil {
		return nil, nil, err
	}
	data := turnDataV2{Time: v.Time, Refs: v.Refs, Alternatives: v.Alternatives, Operations: v.Operations}
	head, err := renderV3("head", data)
	if err != nil {
		return nil, nil, err
	}
	foot, err := renderV3("foot", data)
	if err != nil {
		return nil, nil, err
	}
	mid, err := renderV3("comparison", data)
	if err != nil {
		return nil, nil, err
	}
	c.Message = head + foot
	tr.Message = head + mid + foot
	for _, a := range v.Alternatives {
		detail, err := renderV3("detail", a)
		if err != nil {
			return nil, nil, err
		}
		tr.Inspect[a.ID] = detail
		tr.Record.InspectSha256[a.ID] = sha256Hex([]byte(detail))
	}
	tr.Record.ViewSha256 = sha(v)
	tr.Record.Map = m
	for _, t := range []*Turn{c, tr} {
		t.Record.MessageSha256 = sha256Hex([]byte(t.Message))
	}
	return c, tr, digestsUnchanged(s, before)
}
func TurnsV2(s *session.Session, reply []api.Action) (*Turn, *Turn, error) {
	return TurnsV2With(s, SessionValidator(s), reply)
}

// System-level final schemas permit reason/actions as a whole-reply revision.
// Lower-priority templates cannot remove this route.
func ParseTurnReplyV2(content string) TurnReply {
	var w replyWire
	if err := exactJSON([]byte(content), reflect.TypeOf(w)); err != nil {
		return TurnReply{Kind: ReplyUnreadable, Error: err.Error()}
	}
	if err := json.Unmarshal([]byte(content), &w); err != nil {
		return TurnReply{Kind: ReplyUnreadable, Error: err.Error()}
	}
	if w.Decision == nil && w.Inspect == nil && w.Actions != nil {
		var obj map[string]json.RawMessage
		if err := json.Unmarshal([]byte(content), &obj); err != nil {
			return TurnReply{Kind: ReplyUnreadable, Error: err.Error()}
		}
		obj["decision"] = json.RawMessage(`"revise"`)
		raw, _ := json.Marshal(obj)
		content = string(raw)
	}
	r := ParseTurnReply(content)
	if r.Kind == ReplyRevise && (w.Actions == nil || w.ID != nil) {
		return TurnReply{Kind: ReplyUnreadable, Error: "revision requires actions and no id"}
	}
	return r
}

type OutcomeV2 struct {
	*Outcome
	ResponseRoute   string          `json:"responseRoute"`
	IntendedAction  [][]api.Action  `json:"intendedAction"`
	ValidatedAction [][]api.Action  `json:"validatedAction"`
	SubmittedAction SubmittedAction `json:"submittedAction"`
	ExecutedAction  string          `json:"executedAction"`
}

func DecideV2With(s *session.Session, v Validator, arm string, proposal []api.Action, inspected []string, r *Reply) (*OutcomeV2, error) {
	o, err := DecideWith(s, v, arm, proposal, inspected, r)
	if err != nil {
		return nil, err
	}
	route := "unreadable"
	var intended [][]api.Action
	if r != nil {
		route = r.Kind
		switch r.Kind {
		case ReplyRevise:
			ops, staff := SplitReply(r.Actions)
			intended = [][]api.Action{}
			for _, b := range [][]api.Action{ops, staff} {
				if len(b) > 0 {
					intended = append(intended, b)
				}
			}
			if len(r.Actions) == 0 && r.ID == "" {
				o.Result = ResultRevised
				o.Valid = true
				o.Batches = [][]api.Action{}
				o.Operations = []api.Action{}
				o.Staffing = []api.Action{}
				o.Final = FinalNoStaffingAction
				o.Why = "deliberate empty revision"
				ops, _ := SplitReply(proposal)
				if len(ops) > 0 {
					o.Final = FinalOther
				}
			}
		case ReplyAbandon:
			intended = [][]api.Action{}
		case ReplySelect:
			ops, staff := SplitReply(proposal)
			sel, er := SelectV2(s, staff)
			if er != nil {
				return nil, er
			}
			for _, c := range sel.Candidates {
				if c.ID == r.ID && (arm == ArmTreatment || c.ID == IDProposal) {
					intended = [][]api.Action{}
					for _, b := range [][]api.Action{ops, c.Actions} {
						if len(b) > 0 {
							intended = append(intended, b)
						}
					}
				}
			}
		}
	}
	// Recompute the secondary citation flag against the successor's delivered
	// typed values, including capacity fields and unit-aware integer counts.
	if arm == ArmTreatment && r != nil {
		b, er := build(s, v, proposal)
		if er != nil {
			return nil, er
		}
		vv, _, er := BuildViewV2(s, b.agent, b.pos, len(b.ops))
		if er != nil {
			return nil, er
		}
		o.Flags = slices.DeleteFunc(o.Flags, func(f string) bool { return f == FlagCompared || f == FlagNoComparison })
		if len(r.Comparison) > 0 {
			ok, checks := verifyCitationsV2(vv, inspected, r.Comparison, o.Alternative)
			o.CitationChecks = checks
			if ok {
				o.Flags = append(o.Flags, FlagCompared)
			}
		}
		if len(o.Flags) == 0 {
			o.Flags = append(o.Flags, FlagNoComparison)
		}
	}
	sub := CanonicalSubmitted(o.Batches)
	o.Schema = "dreamteam.alternatives-outcome/v2"
	// Keep the seven-way legacy field as a process measure. The new primary
	// identity is submittedAction.sha256 and never depends on response route.
	out := &OutcomeV2{Outcome: o, ResponseRoute: route, IntendedAction: intended, ValidatedAction: nil, SubmittedAction: sub, ExecutedAction: "NOT_EXECUTED"}
	if o.Valid {
		out.ValidatedAction = o.Batches
	}
	return out, nil
}

func armTurnV2(s *session.Session, v Validator, arm string, actions []api.Action) (*Turn, error) {
	c, t, err := TurnsV2With(s, v, actions)
	if err != nil {
		return nil, err
	}
	if arm == ArmControl {
		return c, nil
	}
	if arm == ArmTreatment {
		return t, nil
	}
	return nil, fmt.Errorf("unknown arm %q", arm)
}
