package alternatives

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"text/template"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/session"
)

// The V0.13d final-decision turn, dreamteam.alternatives-arms/v2: both
// arms from one decision point and one reply, rendered from the typed view
// (view.go) through presentation v2 (text/presentation-v2.tmpl).
//
//   - Both arms get the same head and foot, byte for byte apart from the
//     typed values they print (the time, the IDs a reply may select, the
//     number of operations actions): "Nothing has been applied yet" is in
//     both (owner decision D8).
//   - The treatment adds the comparison between them, and answers inspect
//     with the details. That is the only content difference.
//   - The turn is due by one test for both arms: rule v2 applies to the
//     reply's staffing batch, the engine accepts both of the reply's
//     batches, and the comparison has a typed view. If the view cannot be
//     made, neither arm is shown a turn (not_due:not_representable).
//
// V0.13c's Steps (arms.go) and its pinned texts are kept unchanged.

const (
	ArmsV2ID             = "dreamteam.alternatives-arms/v2"
	TurnSchema           = "dreamteam.alternatives-turn/v1"
	PresentationV2ID     = "dreamteam.alternatives-presentation/v2"
	PresentationV2Sha256 = "8327f254ed2e25865db781f5f1cdcc70db129ea0df72ab6d0ce924ddf79ce496"
	ViewTableSha256      = "ce7f0e238c2d8cf71bf87cf720f12b56bd667035971308ae9abae7399d66aba0"
	StepNotDueView       = "not_due:not_representable" // the comparison has no typed view; no turn in either arm
)

//go:embed text/presentation-v2.tmpl
var presentationV2Text []byte

var presentationV2 = template.Must(template.New("p2").Funcs(template.FuncMap{
	"refText":       refText,
	"actions":       viewActions,
	"validation":    viewValidation,
	"ids":           func(d turnData) string { return idList(d.IDs) },
	"line":          viewLine,
	"reversibility": func(a ViewAlt) string { return viewReversibility[a.Reversibility] },
	"unavailable":   viewUnavailableText,
}).Parse(string(presentationV2Text)))

type turnData struct {
	Time         string
	Refs         []Ref
	Alternatives []ViewAlt
	Operations   int
	IDs          []string
}

func refText(r Ref) string {
	switch r.Kind {
	case RefEmployee:
		return fmt.Sprintf("the employee in action %d of your reply", r.Action)
	case RefHire:
		return fmt.Sprintf("the hire request in action %d of your reply", r.Action)
	case RefNewContract:
		return fmt.Sprintf("the new teleradiology contract action %d of your reply would start", r.Action)
	case RefContract:
		return fmt.Sprintf("the teleradiology contract in action %d of your reply", r.Action)
	case RefPlan:
		return fmt.Sprintf("the teleradiology plan in action %d of your reply", r.Action)
	}
	return "?"
}

func viewActions(a ViewAlt) string {
	if len(a.Actions) == 0 {
		return "no staffing action (no batch)"
	}
	var parts []string
	for _, x := range a.Actions {
		s := fmt.Sprintf("action %d: %s %s", x.Action, x.Type, x.Target)
		for _, f := range x.Fields {
			s += ", " + f.Name + " " + f.Value
		}
		if len(x.Changed) > 0 {
			s += " (" + strings.Join(x.Changed, ", ") + " differs from your reply)"
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, "; ")
}

func viewValidation(a ViewAlt) string {
	if a.Accepted {
		if len(a.Actions) == 0 {
			return "accepted (nothing to apply)"
		}
		return "accepted"
	}
	var parts []string
	for _, r := range a.Refused {
		parts = append(parts, fmt.Sprintf("action %d %s", r.Action, r.Code))
	}
	return "refused: " + strings.Join(parts, "; ")
}

func idList(ids []string) string {
	var out []string
	for _, id := range ids {
		if id == IDProposal {
			id += " (your proposal)"
		}
		out = append(out, id)
	}
	return strings.Join(out, ", ")
}

func viewValue(it ViewItem, k viewKind) string {
	switch {
	case it.Of != nil:
		return count(*it.Number) + " of " + count(*it.Of)
	case k.show == showMinutes:
		return fmt.Sprintf("%+d min", *it.Number)
	case it.Date != "":
		return it.Date
	case it.Enum != "":
		switch it.Enum {
		case "true":
			return "yes"
		case "false":
			return "no"
		}
		return it.Enum
	}
	switch k.unit {
	case UnitCents:
		return money(*it.Number)
	case UnitDays:
		return fmt.Sprintf("%d days", *it.Number)
	}
	return count(*it.Number)
}

// count is a whole number with thousands separators.
func count(n int64) string {
	s := strconv.FormatInt(n, 10)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

func viewLabel(it ViewItem, k viewKind) string {
	if strings.Contains(k.label, "%s") {
		return fmt.Sprintf(k.label, it.Ref)
	}
	return k.label
}

func viewLine(it ViewItem) string {
	k := viewKinds[it.Kind]
	s := viewLabel(it, k) + ": " + viewValue(it, k)
	if len(it.List) > 0 {
		s += " (" + strings.Join(it.List, ", ") + ")"
	}
	if it.Form >= 0 && it.Form < len(k.when) {
		s += " · " + k.when[it.Form].render(it.When)
	}
	basis := k.basis
	if it.Form < 0 && k.basisNoWhen != "" {
		basis = k.basisNoWhen
	}
	if basis != "" && basis != "-" {
		s += " · " + basis
	}
	return s
}

func viewUnavailableText(it ViewItem) string {
	u := viewUnavailable[it.Kind]
	label := u.label
	if strings.Contains(label, "%s") {
		label = fmt.Sprintf(label, it.Ref)
	}
	return label + ": " + u.reason
}

// Turn is one arm's final-decision turn. Message and Inspect are the only
// texts a driver may send; Record is for the decision record, never sent.
type Turn struct {
	Schema  string            `json:"schema"`
	Arms    string            `json:"arms"`
	Arm     string            `json:"arm"`
	Due     bool              `json:"due"`
	Why     string            `json:"why"`
	Message string            `json:"message"`
	Inspect map[string]string `json:"inspect"`
	Allowed []string          `json:"allowed"`
	Record  TurnRecord        `json:"record"`
}

// TurnRecord is what the record keeps about a turn.
type TurnRecord struct {
	Proposal           []api.Action      `json:"proposal"`   // the reply's staffing batch
	Operations         []api.Action      `json:"operations"` // the reply's operations batch, carried unchanged
	ReplyPositions     []int             `json:"replyPositions"`
	Rule               string            `json:"rule"`
	RuleSha256         string            `json:"ruleSha256"`
	Presentation       string            `json:"presentation"`
	PresentationSha256 string            `json:"presentationSha256"`
	ViewSchema         string            `json:"viewSchema"`
	ViewTableSha256    string            `json:"viewTableSha256"`
	ComparisonSha256   string            `json:"comparisonSha256,omitempty"`
	ViewSha256         string            `json:"viewSha256,omitempty"`
	MessageSha256      string            `json:"messageSha256"`
	InspectSha256      map[string]string `json:"inspectSha256"`
	Map                *ViewMap          `json:"map,omitempty"`
	ViewError          string            `json:"viewError,omitempty"`
}

// split is SplitReply with each staffing action's 1-based position in the
// reply.
func splitPositions(reply []api.Action) (ops, staff []api.Action, pos []int) {
	ops, staff = SplitReply(reply)
	for i, a := range reply {
		if isStaffing(a.Type) {
			pos = append(pos, i+1)
		}
	}
	return ops, staff, pos
}

// built is everything one decision point gives both arms.
type built struct {
	sel        *Selection
	agent      *Agent
	view       *View
	vmap       *ViewMap
	ops, staff []api.Action
	pos        []int
	why        string
	viewErr    string
}

func build(s *session.Session, reply []api.Action) (*built, error) {
	b := &built{}
	b.ops, b.staff, b.pos = splitPositions(reply)
	sel, err := SelectV2(s, b.staff)
	if err != nil {
		return nil, err
	}
	b.sel = sel
	opsOK := true
	if len(b.ops) > 0 {
		for _, r := range s.ValidateBatch(b.ops...) {
			opsOK = opsOK && r.OK
		}
	}
	switch {
	case !sel.Applies:
		b.why = StepNotDueRule
	case len(sel.Candidates) < 2:
		b.why = StepNotDueNoSelect
	case !sel.Candidates[0].OK:
		b.why = StepNotDueRefused
	case !opsOK:
		b.why = StepNotDueOps
	}
	if b.why != "" {
		return b, nil
	}
	a, err := BuildAgent(s, sel)
	if err != nil {
		return nil, err
	}
	raw, err := a.Encode()
	if err != nil {
		return nil, err
	}
	if b.agent, err = DecodeAgentAt(raw, s, sel); err != nil {
		return nil, err
	}
	if b.view, b.vmap, err = BuildView(s, b.agent, b.pos, len(b.ops)); err != nil {
		b.why, b.viewErr, b.agent = StepNotDueView, err.Error(), nil
	}
	return b, nil
}

// Turns builds both arms' turns for a reply at s's current time. s is
// only read (its digests are checked unchanged).
func Turns(s *session.Session, reply []api.Action) (control, treatment *Turn, err error) {
	before := s.Digests()
	b, err := build(s, reply)
	if err != nil {
		return nil, nil, err
	}
	control, treatment, err = b.turns()
	if err != nil {
		return nil, nil, err
	}
	return control, treatment, digestsUnchanged(s, before)
}

func (b *built) turns() (control, treatment *Turn, err error) {
	mk := func(arm string) *Turn {
		return &Turn{Schema: TurnSchema, Arms: ArmsV2ID, Arm: arm, Inspect: map[string]string{}, Allowed: []string{},
			Record: TurnRecord{Proposal: b.staff, Operations: b.ops, ReplyPositions: b.pos, Rule: b.sel.Rule, RuleSha256: b.sel.RuleSha256,
				Presentation: PresentationV2ID, PresentationSha256: PresentationV2Sha256, ViewSchema: ViewSchema, ViewTableSha256: ViewTableSha256,
				InspectSha256: map[string]string{}, ViewError: b.viewErr}}
	}
	control, treatment = mk(ArmControl), mk(ArmTreatment)
	if b.why != "" {
		for _, t := range []*Turn{control, treatment} {
			t.Why, t.Record.MessageSha256 = b.why, sha256Hex(nil)
		}
		return control, treatment, nil
	}
	v := b.view
	var allIDs []string
	for _, x := range v.Alternatives {
		allIDs = append(allIDs, x.ID)
	}
	render := func(name string, d any) (string, error) {
		var buf bytes.Buffer
		err := presentationV2.ExecuteTemplate(&buf, name, d)
		return buf.String(), err
	}
	text := func(ids []string, comparison bool) (string, error) {
		d := turnData{Time: v.Time, Refs: v.Refs, Alternatives: v.Alternatives, Operations: v.Operations, IDs: ids}
		head, err := render("head", d)
		if err != nil {
			return "", err
		}
		foot, err := render("foot", d)
		if err != nil {
			return "", err
		}
		mid := ""
		if comparison {
			if mid, err = render("comparison", d); err != nil {
				return "", err
			}
		}
		return head + mid + foot, nil
	}
	if control.Message, err = text([]string{IDProposal}, false); err != nil {
		return nil, nil, err
	}
	control.Allowed = []string{"select:" + IDProposal, ReplyRevise, ReplyAbandon}
	if treatment.Message, err = text(allIDs, true); err != nil {
		return nil, nil, err
	}
	for _, x := range v.Alternatives {
		d, err := render("detail", x)
		if err != nil {
			return nil, nil, err
		}
		treatment.Inspect[x.ID] = d
		treatment.Record.InspectSha256[x.ID] = sha256Hex([]byte(d))
		treatment.Allowed = append(treatment.Allowed, "select:"+x.ID)
	}
	treatment.Allowed = append(treatment.Allowed, ReplyRevise, ReplyAbandon)
	treatment.Record.ComparisonSha256, treatment.Record.ViewSha256, treatment.Record.Map = b.agent.Sha256, sha(v), b.vmap
	for _, t := range []*Turn{control, treatment} {
		t.Due, t.Why = true, "due"
		if err := checkText(t.Message); err != nil {
			return nil, nil, err
		}
		for id, d := range t.Inspect {
			if err := checkText(d); err != nil {
				return nil, nil, fmt.Errorf("inspect:%s: %w", id, err)
			}
		}
		t.Record.MessageSha256 = sha256Hex([]byte(t.Message))
	}
	return control, treatment, nil
}

// --- the reply ------------------------------------------------------------------

// Reply kinds a turn's model reply can have, besides a decision.
const (
	ReplyInspect    = "inspect"
	ReplyUnreadable = "unreadable"
	MaxReason       = 4000
)

// TurnReply is a parsed model reply to a turn.
type TurnReply struct {
	Kind    string   `json:"kind"` // select, revise, abandon, inspect, unreadable
	Reply   *Reply   `json:"reply,omitempty"`
	Inspect []string `json:"inspect,omitempty"`
	Error   string   `json:"error,omitempty"`
}

// ParseTurnReply reads a model's reply to a turn: exactly one JSON
// object, nothing before or after it but white space, with only the
// fields of the foot (decision, id, actions, reason, cite) or of an
// inspect request (inspect, and optionally reason). Anything else is
// unreadable; nothing is guessed.
func ParseTurnReply(content string) TurnReply {
	bad := func(f string, a ...any) TurnReply {
		return TurnReply{Kind: ReplyUnreadable, Error: fmt.Sprintf(f, a...)}
	}
	var raw struct {
		Decision *string        `json:"decision"`
		ID       *string        `json:"id"`
		Actions  *[]api.Action  `json:"actions"`
		Reason   *string        `json:"reason"`
		Cite     *[]Citation    `json:"cite"`
		Inspect  *[]string      `json:"inspect"`
		Extra    map[string]any `json:"-"`
	}
	dec := json.NewDecoder(strings.NewReader(content))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&raw); err != nil {
		return bad("not one JSON object with the reply's fields: %v", err)
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return bad("text after the JSON object")
	}
	reason := ""
	if raw.Reason != nil {
		reason = *raw.Reason
		if len(reason) > MaxReason {
			return bad("reason longer than %d bytes", MaxReason)
		}
	}
	if raw.Inspect != nil {
		if raw.Decision != nil || raw.ID != nil || raw.Actions != nil || raw.Cite != nil {
			return bad("inspect with a decision")
		}
		if len(*raw.Inspect) == 0 || len(*raw.Inspect) > MaxCandidates {
			return bad("inspect needs one to %d IDs", MaxCandidates)
		}
		return TurnReply{Kind: ReplyInspect, Inspect: slices.Clone(*raw.Inspect)}
	}
	if raw.Decision == nil {
		return bad("no decision")
	}
	r := &Reply{Kind: *raw.Decision, Reason: reason}
	if raw.ID != nil {
		r.ID = *raw.ID
	}
	if raw.Actions != nil {
		r.Actions = canon(*raw.Actions)
	}
	if raw.Cite != nil {
		if len(*raw.Cite) > MaxCitations {
			return bad("more than %d citations", MaxCitations)
		}
		r.Comparison = *raw.Cite
	}
	switch r.Kind {
	case ReplySelect, ReplyRevise, ReplyAbandon:
	default:
		return bad("decision %q is not select, revise or abandon", r.Kind)
	}
	return TurnReply{Kind: r.Kind, Reply: r}
}

// --- the decision ---------------------------------------------------------------

// OutcomeSchema identifies a turn's classified outcome.
const OutcomeSchema = "dreamteam.alternatives-outcome/v1"

// Outcome is a turn's decision, classified, and what would be submitted:
// the batches in order (operations, then staffing; each validated as a
// batch at the decision point, as the agents' batches are). Nothing is
// defaulted: when a batch is refused nothing is submitted.
type Outcome struct {
	Schema           string               `json:"schema"`
	Arm              string               `json:"arm"`
	Result           string               `json:"result"`
	Alternative      string               `json:"alternative,omitempty"`
	Why              string               `json:"why"`
	Flags            []string             `json:"flags"`
	CitationChecks   []string             `json:"citationChecks"`
	Operations       []api.Action         `json:"operations"`
	Staffing         []api.Action         `json:"staffing"`
	Batches          [][]api.Action       `json:"batches"`
	Validation       [][]api.ActionResult `json:"validation"`
	Valid            bool                 `json:"valid"`
	ComparisonSha256 string               `json:"comparisonSha256,omitempty"`
}

// Decide classifies a decision reply to arm's turn at s for the same
// reply the turn was built from, after the inspect requests. It rebuilds
// the turn (the same function of s and reply), so the decision is bound to
// what was shown. s is only read.
func Decide(s *session.Session, arm string, reply []api.Action, inspected []string, r *Reply) (*Outcome, error) {
	before := s.Digests()
	b, err := build(s, reply)
	if err != nil {
		return nil, err
	}
	if b.why != "" {
		return nil, fmt.Errorf("%w: no turn is due (%s)", ErrContract, b.why)
	}
	requests := []string{}
	for _, id := range inspected {
		requests = append(requests, "inspect:"+id)
	}
	out := &Outcome{Schema: OutcomeSchema, Arm: arm, Flags: []string{}, CitationChecks: []string{}, Operations: []api.Action{},
		Staffing: []api.Action{}, Batches: [][]api.Action{}, Validation: [][]api.ActionResult{}}
	revalidate := func(batch []api.Action) []api.ActionResult { return s.ValidateBatch(batch...) }
	// A revision replaces the whole reply. When its operations batch is
	// the reply's, its staffing batch is classified against the listed
	// alternatives; otherwise it is a revision of the whole proposal.
	cls := r
	whole := false
	var revOps, revStaff []api.Action
	if r != nil && r.Kind == ReplyRevise && r.ID == "" && len(r.Actions) > 0 {
		revOps, revStaff = SplitReply(r.Actions)
		if !jsonEqual(revOps, b.ops) || len(revStaff) == 0 {
			whole = true
		} else {
			c := *r
			c.Actions = revStaff
			cls = &c
		}
	}
	var d Decision
	switch arm {
	case ArmControl:
		st := &Step{Arm: ArmControl, Due: true, Proposal: b.staff}
		if whole {
			d = Decision{Requests: requests, Reply: r, Flags: []string{FlagNoComparison}, CitationChecks: []string{}}
			d.Result, d.Why = ResultRevised, "a new reply: its operations actions differ from the reply's, or it has no staffing action"
		} else {
			d = ClassifyControl(st, requests, cls, revalidate)
		}
	case ArmTreatment:
		noCite := cls
		if cls != nil {
			c := *cls
			c.Comparison = nil
			noCite = &c
		}
		if whole {
			d = Decision{Requests: requests, Reply: r, Flags: []string{}, CitationChecks: []string{}}
			d.Result, d.Why = ResultRevised, "a new reply: its operations actions differ from the reply's, or it has no staffing action"
		} else {
			d = Classify(b.agent, requests, noCite, revalidate)
		}
		out.ComparisonSha256 = b.agent.Sha256
		// Flags: inspected as Classify; compared from the view.
		d.Flags = slices.DeleteFunc(d.Flags, func(f string) bool { return f == FlagNoComparison || f == FlagCompared })
		if r != nil && len(r.Comparison) > 0 {
			if ok, checks := verifyViewCitations(b.view, r.Comparison, d.Alternative); ok {
				d.Flags = append(d.Flags, FlagCompared)
			} else {
				d.CitationChecks = checks
			}
		}
		if len(d.Flags) == 0 {
			d.Flags = append(d.Flags, FlagNoComparison)
		}
	default:
		return nil, fmt.Errorf("%w: arm %q", ErrContract, arm)
	}
	out.Result, out.Alternative, out.Why, out.Flags, out.CitationChecks = d.Result, d.Alternative, d.Why, d.Flags, d.CitationChecks
	// What would be submitted (owner decision D10): select keeps the
	// reply's operations batch with the chosen staffing batch; a revision
	// replaces both; abandon submits nothing.
	switch {
	case whole:
		out.Operations, out.Staffing = revOps, revStaff
	case out.Result == ResultRetained || out.Result == ResultSelected || out.Result == ResultSelectedNoAction ||
		(out.Result == ResultRevised && r != nil && r.Kind == ReplyRevise):
		out.Operations, out.Staffing = b.ops, d.Submitted
		if out.Result == ResultSelectedNoAction {
			out.Staffing = []api.Action{}
		}
	}
	if out.Operations == nil {
		out.Operations = []api.Action{}
	}
	if out.Staffing == nil {
		out.Staffing = []api.Action{}
	}
	out.Valid = true
	if out.Result == ResultAbandoned || out.Result == ResultNoDecision || out.Result == ResultInvalidAlterative {
		out.Operations, out.Staffing, out.Valid = []api.Action{}, []api.Action{}, out.Result == ResultAbandoned
	}
	for _, batch := range [][]api.Action{out.Operations, out.Staffing} {
		if len(batch) == 0 {
			continue
		}
		res := s.ValidateBatch(batch...)
		out.Validation = append(out.Validation, res)
		out.Batches = append(out.Batches, batch)
		for _, x := range res {
			out.Valid = out.Valid && x.OK
		}
	}
	if !out.Valid && out.Result != ResultNoDecision {
		// A refused batch: nothing is submitted.
		out.Result, out.Why = ResultInvalidAlterative, "the engine refuses a batch at final validation; nothing is submitted"
		out.Batches = [][]api.Action{}
	}
	if out.Result == ResultNoDecision {
		out.Valid = false
	}
	return out, digestsUnchanged(s, before)
}

// verifyViewCitations applies V0.13b's five checks to citations of the
// view's items, by their shown keys and values.
func verifyViewCitations(v *View, cs []Citation, chosen string) (bool, []string) {
	var fails []string
	shown := map[string]bool{}
	values := map[string]map[string]string{} // alt -> key -> shown value
	for _, x := range v.Alternatives {
		values[x.ID] = map[string]string{}
		for _, list := range [][]ViewItem{x.Facts, x.Estimates, x.Obligations} {
			for _, it := range list {
				shown[it.Key] = true
				values[x.ID][it.Key] = viewValue(it, viewKinds[it.Kind])
			}
		}
	}
	if len(cs) > MaxCitations || len(cs) >= len(shown) {
		fails = append(fails, "too many entries: at most three, and fewer than the items shown")
	}
	// Every numeric item shown for every accepted alternative: citing all
	// of them is copying the table.
	var rows []string
	for k := range shown {
		all := true
		for _, x := range v.Alternatives {
			if !x.Accepted {
				continue
			}
			if _, ok := values[x.ID][k]; !ok {
				all = false
			}
		}
		if all {
			rows = append(rows, k)
		}
	}
	cited := map[string]bool{}
	for _, c := range cs {
		cited[c.Item] = true
	}
	if len(rows) > 0 && !slices.ContainsFunc(rows, func(k string) bool { return !cited[k] }) {
		fails = append(fails, "repeats every row shown for every alternative")
	}
	differ := false
	for _, c := range cs {
		if len(c.Values) < 2 {
			fails = append(fails, c.Item+": names fewer than two alternatives")
			continue
		}
		other := false
		var seen []string
		ids := make([]string, 0, len(c.Values))
		for id := range c.Values {
			ids = append(ids, id)
		}
		slices.Sort(ids)
		for _, id := range ids {
			if id != chosen {
				other = true
			}
			want, ok := values[id][c.Item]
			if !ok {
				fails = append(fails, c.Item+": not shown for "+id)
				continue
			}
			if !citedEquals(c.Values[id], want) {
				fails = append(fails, c.Item+": the value cited for "+id+" is not the value shown")
				continue
			}
			seen = append(seen, want)
		}
		if !other {
			fails = append(fails, c.Item+": names only the alternative chosen")
		}
		for _, x := range seen {
			if x != seen[0] {
				differ = true
			}
		}
	}
	if !differ {
		fails = append(fails, "no entry shows a difference")
	}
	return len(fails) == 0, fails
}

// citedEquals: a cited value equals the shown text, or, for a shown whole
// number, the same number.
func citedEquals(raw json.RawMessage, shown string) bool {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s == shown
	}
	var n int64
	if json.Unmarshal(raw, &n) == nil {
		return strconv.FormatInt(n, 10) == shown
	}
	return false
}
