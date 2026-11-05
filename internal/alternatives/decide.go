package alternatives

import (
	"bytes"
	"encoding/json"
	"slices"
	"strings"

	"github.com/georgezero/dreamteam-go/internal/api"
)

// Decisions against a comparison (V0.13a EVALUATION_CONTRACT, implemented
// for scripted replies only): one result per decision point, from the
// reply and what would be submitted, and behaviour flags from what is
// observable in the record. The reply's reason is recorded, never scored.

// Results.
const (
	ResultRetained          = "retained"
	ResultSelected          = "selected"
	ResultSelectedNoAction  = "selected_no_action"
	ResultRevised           = "revised"
	ResultAbandoned         = "abandoned"
	ResultInvalidAlterative = "invalid_alternative"
	ResultNoDecision        = "no_decision"
)

// Flags.
const (
	FlagInspected    = "inspected"
	FlagCompared     = "compared"
	FlagNoComparison = "no_observable_comparison"
)

// Reply kinds.
const (
	ReplySelect  = "select"
	ReplyRevise  = "revise"
	ReplyAbandon = "abandon"
)

// MaxCitations bounds a stated comparison.
const MaxCitations = 3

// Citation is one entry of a stated comparison: an item key and the
// values the reply says the named alternatives have.
type Citation struct {
	Item   string                     `json:"item"`
	Values map[string]json.RawMessage `json:"values"`
}

// Reply is a parsed reply.
type Reply struct {
	Kind       string       `json:"kind"`
	ID         string       `json:"id,omitempty"`
	Actions    []api.Action `json:"actions,omitempty"`
	Comparison []Citation   `json:"comparison,omitempty"`
	Reason     string       `json:"reason,omitempty"`
}

// Decision is the classified record of one decision point.
type Decision struct {
	ComparisonSha256 string             `json:"comparisonSha256"`
	Requests         []string           `json:"requests"`
	Reply            *Reply             `json:"reply"`
	Result           string             `json:"result"`
	Alternative      string             `json:"alternative,omitempty"` // the listed alternative the result names
	Why              string             `json:"why"`
	Flags            []string           `json:"flags"`
	CitationChecks   []string           `json:"citationChecks"`
	Submitted        []api.Action       `json:"submitted"`
	Revalidation     []api.ActionResult `json:"revalidation"`
}

// Classify classifies a reply to comparison a, after the agent's requests
// (inspect:<id> and others, in order). revalidate is the engine's final
// validation of a batch about to be submitted. Nothing is ever defaulted:
// a missing, contradictory or invalid reply submits nothing.
func Classify(a *Agent, requests []string, reply *Reply, revalidate func([]api.Action) []api.ActionResult) Decision {
	d := Decision{ComparisonSha256: a.Sha256, Requests: append([]string{}, requests...), Reply: reply, Flags: []string{},
		CitationChecks: []string{}, Submitted: []api.Action{}, Revalidation: []api.ActionResult{}}
	listed := func(id string) (AgentAlternative, bool) {
		i := slices.IndexFunc(a.Alternatives, func(x AgentAlternative) bool { return x.ID == id })
		if i < 0 {
			return AgentAlternative{}, false
		}
		return a.Alternatives[i], true
	}
	submit := func(alt string, batch []api.Action, ok string) {
		d.Revalidation = revalidate(batch)
		for _, r := range d.Revalidation {
			if !r.OK {
				d.Result, d.Alternative, d.Why = ResultInvalidAlterative, alt, "the engine refuses the batch at final validation"
				return
			}
		}
		d.Result, d.Alternative, d.Submitted = ok, alt, canon(batch)
	}
	// match names the listed alternative with exactly this batch.
	match := func(batch []api.Action) (string, bool) {
		for _, x := range a.Alternatives {
			if jsonEqual(canon(x.Actions), canon(batch)) {
				return x.ID, true
			}
		}
		return "", false
	}
	byID := func(id string, batch []api.Action) {
		switch id {
		case IDNoAction:
			d.Result, d.Alternative, d.Why = ResultSelectedNoAction, id, "no action selected; nothing submitted"
		case IDProposal:
			d.Why = "the proposal, unchanged"
			submit(id, batch, ResultRetained)
		default:
			d.Why = "a listed alternative other than the proposal"
			submit(id, batch, ResultSelected)
		}
	}
	switch {
	case reply == nil:
		d.Result, d.Why = ResultNoDecision, "no usable reply"
	case reply.Kind == ReplySelect && (reply.ID == "" || len(reply.Actions) > 0):
		d.Result, d.Why = ResultNoDecision, "contradictory select: needs an ID and no batch"
	case reply.Kind == ReplySelect:
		alt, ok := listed(reply.ID)
		if !ok {
			d.Result, d.Alternative, d.Why = ResultInvalidAlterative, reply.ID, "names an alternative that is not listed"
			break
		}
		if !alt.Validation.OK {
			d.Result, d.Alternative, d.Why = ResultInvalidAlterative, reply.ID, "names an alternative the engine refused"
			break
		}
		byID(alt.ID, alt.Actions)
	case reply.Kind == ReplyRevise && (reply.ID != "" || len(reply.Actions) == 0):
		d.Result, d.Why = ResultNoDecision, "contradictory revise: needs a batch and no ID"
	case reply.Kind == ReplyRevise:
		if id, ok := match(reply.Actions); ok {
			// The label does not decide the result: a listed batch is that
			// alternative.
			alt, _ := listed(id)
			if !alt.Validation.OK {
				d.Result, d.Alternative, d.Why = ResultInvalidAlterative, id, "revised to an alternative the engine refused"
				break
			}
			byID(id, reply.Actions)
			d.Why += " (submitted as a revision)"
			break
		}
		d.Why = "a new batch, different from every listed alternative"
		submit("", reply.Actions, ResultRevised)
	case reply.Kind == ReplyAbandon && (reply.ID != "" || len(reply.Actions) > 0):
		d.Result, d.Why = ResultNoDecision, "contradictory abandon: takes no ID and no batch"
	case reply.Kind == ReplyAbandon:
		d.Result, d.Why = ResultAbandoned, "abandoned; nothing submitted"
	default:
		d.Result, d.Why = ResultNoDecision, "unknown reply kind"
	}

	// Flags.
	for _, r := range requests {
		if id, ok := strings.CutPrefix(r, "inspect:"); ok && id != IDProposal {
			if _, ok := listed(id); ok {
				d.Flags = append(d.Flags, FlagInspected)
				break
			}
		}
	}
	if reply != nil && len(reply.Comparison) > 0 {
		if ok, checks := verifyCitations(a, reply.Comparison, d.Alternative); ok {
			d.Flags = append(d.Flags, FlagCompared)
		} else {
			d.CitationChecks = checks
		}
	}
	if len(d.Flags) == 0 {
		d.Flags = append(d.Flags, FlagNoComparison)
	}
	return d
}

// verifyCitations applies the five checks of a stated comparison: each
// item is shown for each alternative named; each entry names two or more
// alternatives, one of them not the one chosen; each value equals the
// shown value exactly; one entry's values differ; at most three entries,
// fewer than the items shown, and not every row of the compared table
// (repeating the table does not count).
func verifyCitations(a *Agent, cs []Citation, chosen string) (bool, []string) {
	var fails []string
	shown := map[string]bool{}
	for _, x := range a.Alternatives {
		for _, list := range [][]Value{x.Facts, x.Estimates, x.Obligations} {
			for _, v := range list {
				shown[v.Key] = true
			}
		}
	}
	if len(cs) > MaxCitations || len(cs) >= len(shown) {
		fails = append(fails, "too many entries: at most three, and fewer than the items shown")
	}
	// Repeating every row of the compared table is copying, not comparing.
	cited := map[string]bool{}
	for _, c := range cs {
		cited[c.Item] = true
	}
	all := len(a.Compared) > 0
	for _, c := range a.Compared {
		all = all && cited[c.Key]
	}
	if all {
		fails = append(fails, "repeats every row of the compared table")
	}
	differ := false
	for _, c := range cs {
		if len(c.Values) < 2 {
			fails = append(fails, c.Item+": names fewer than two alternatives")
			continue
		}
		other := false
		var seen []string
		for id, raw := range c.Values {
			if id != chosen {
				other = true
			}
			v, ok := shownValue(a, id, c.Item)
			if !ok {
				fails = append(fails, c.Item+": not shown for "+id)
				continue
			}
			if !valueEquals(v, raw) {
				fails = append(fails, c.Item+": the value cited for "+id+" is not the value shown")
				continue
			}
			seen = append(seen, string(bytes.TrimSpace(raw)))
		}
		if !other {
			fails = append(fails, c.Item+": names only the alternative chosen")
		}
		for _, s := range seen {
			if s != seen[0] {
				differ = true
			}
		}
	}
	if !differ {
		fails = append(fails, "no entry shows a difference")
	}
	return len(fails) == 0, fails
}

func shownValue(a *Agent, id, key string) (Value, bool) {
	for _, x := range a.Alternatives {
		if x.ID != id {
			continue
		}
		for _, list := range [][]Value{x.Facts, x.Estimates, x.Obligations} {
			for _, v := range list {
				if v.Key == key {
					return v, true
				}
			}
		}
	}
	return Value{}, false
}

// valueEquals: a number equals the shown number; a string equals the
// shown date, text or formatted value.
func valueEquals(v Value, raw json.RawMessage) bool {
	var n int64
	if v.Number != nil && json.Unmarshal(raw, &n) == nil {
		return n == *v.Number
	}
	var s string
	if json.Unmarshal(raw, &s) != nil {
		return false
	}
	return s == formatValue(v) || (v.Date != "" && s == v.Date) || (v.Text != "" && s == v.Text)
}
