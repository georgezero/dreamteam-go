// Package alternatives implements V0.13b's deterministic management
// alternatives: the candidate rule (dreamteam.alternative-selection/v1),
// the comparison contract (dreamteam.management-alternatives/v1) in its
// two audiences, the fixed presentation template, and the classification
// of decisions against a comparison. It uses only the session API: it
// never changes a session it reads, and it never contacts a model.
package alternatives

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/session"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// SelectionRule identifies the candidate rule; its text is embedded and
// pinned by SHA-256. SelectionRuleV2 (V0.13c) is v1 with one change: a
// candidate whose canonical batch equals an earlier candidate's is removed
// (text/selection-v2.txt). v1 is kept unchanged: its recorded selections
// and the evidence made with it replay as they were.
const (
	SelectionRule         = "dreamteam.alternative-selection/v1"
	SelectionRuleSha256   = "158a6254952ee3c19dba0b2ea045d2d02de510e7f7f7537cf435357facc5e358"
	SelectionRuleV2       = "dreamteam.alternative-selection/v2"
	SelectionRuleV2Sha256 = "5f32b2c84a0b8383f1a213ce52745e56f5d5aa8292bc714096d1beec70debcf9"
	MaxCandidates         = 4
)

//go:embed text/selection-v1.txt
var selectionText []byte

//go:embed text/selection-v2.txt
var selectionTextV2 []byte

// ruleSha256 is the pin of a rule this build implements ("" for none).
func ruleSha256(rule string) string {
	switch rule {
	case SelectionRule:
		return SelectionRuleSha256
	case SelectionRuleV2:
		return SelectionRuleV2Sha256
	}
	return ""
}

// Candidate IDs.
const (
	IDProposal = "proposal"
	IDNoAction = "no_action"
	IDAdjusted = "adjusted"
)

// Candidate is one listed alternative.
type Candidate struct {
	ID         string             `json:"id"`
	Source     string             `json:"source"` // proposal, no_action, leave_one_out, adjustment
	Actions    []api.Action       `json:"actions"`
	OK         bool               `json:"ok"`
	Validation []api.ActionResult `json:"validation"`
	Sha256     string             `json:"sha256"` // of the rule, the ID and the actions
}

// Omitted is an adjustment the rule defined and the engine refused.
type Omitted struct {
	ID         string             `json:"id"`
	Actions    []api.Action       `json:"actions"`
	Validation []api.ActionResult `json:"validation"`
}

// Removed is a candidate rule v2 removed because its batch equals an
// earlier candidate's (SameAs). Nothing is listed in its place.
type Removed struct {
	ID      string       `json:"id"`
	Source  string       `json:"source"`
	SameAs  string       `json:"sameAs"`
	Actions []api.Action `json:"actions"`
	// BatchSha256 is the SHA-256 of the canonical batch both share.
	BatchSha256 string `json:"batchSha256"`
}

// Selection is the rule's answer at one decision point.
type Selection struct {
	Rule        string       `json:"rule"`
	RuleSha256  string       `json:"ruleSha256"`
	At          string       `json:"at"`
	Today       string       `json:"today"`
	HorizonDays int64        `json:"bookingHorizonDays"`
	Proposal    []api.Action `json:"proposal"`
	Applies     bool         `json:"applies"`
	Why         string       `json:"why"`
	Candidates  []Candidate  `json:"candidates"`
	Omitted     []Omitted    `json:"omitted"`
	// Removed lists rule v2's removed duplicates; it is never set by v1,
	// and is absent when nothing was removed.
	Removed []Removed `json:"removed,omitempty"`
}

// irreversible reports the actions the rule leaves out one at a time.
func irreversible(a api.Action) bool {
	return a.Type == api.ActTerminate || a.Type == api.ActHire || (a.Type == api.ActContractTeleradiology && a.Op == api.TeleOpStart)
}

func isStaffing(t string) bool {
	switch t {
	case api.ActHire, api.ActTerminate, api.ActSetRoster, api.ActSetReadingAllocation, api.ActContractTeleradiology:
		return true
	}
	return false
}

// Select applies rule v1 to proposal at s's current time. s is only read
// (its digests are checked unchanged).
func Select(s *session.Session, proposal []api.Action) (*Selection, error) {
	return SelectRule(s, SelectionRule, proposal)
}

// SelectV2 applies rule v2: v1's list, then later duplicates removed.
func SelectV2(s *session.Session, proposal []api.Action) (*Selection, error) {
	return SelectRule(s, SelectionRuleV2, proposal)
}

// SelectRule applies the named rule (v1 or v2) to proposal at s's current
// time. s is only read (its digests are checked unchanged).
func SelectRule(s *session.Session, rule string, proposal []api.Action) (*Selection, error) {
	pin := ruleSha256(rule)
	if pin == "" {
		return nil, fmt.Errorf("alternatives: selection rule %q is not this build's", rule)
	}
	before := s.Digests()
	o := s.Observe(0)
	sel := &Selection{Rule: rule, RuleSha256: pin, At: o.Time, Today: o.Date,
		HorizonDays: o.Practice.BookingHorizonDays, Proposal: canon(proposal), Candidates: []Candidate{}, Omitted: []Omitted{}}
	switch {
	case len(proposal) == 0:
		sel.Why = "empty proposal"
	case slices.ContainsFunc(proposal, irreversible):
		sel.Applies, sel.Why = true, "holds an irreversible action"
	case slices.ContainsFunc(proposal, func(a api.Action) bool { return isStaffing(a.Type) }) &&
		len(s.PreviewStaffing(proposal...).Displacements) > 0:
		sel.Applies, sel.Why = true, "the staffing preview reports displaced bookings"
	default:
		sel.Why = "no irreversible action and no displaced booking"
	}
	if sel.Applies {
		add := func(id, source string, acts []api.Action) {
			c := Candidate{ID: id, Source: source, Actions: canon(acts), OK: true, Validation: []api.ActionResult{}}
			if len(acts) > 0 {
				c.Validation = s.ValidateBatch(acts...)
				for _, r := range c.Validation {
					c.OK = c.OK && r.OK
				}
			}
			c.Sha256 = candidateSha256(rule, id, c.Actions)
			sel.Candidates = append(sel.Candidates, c)
		}
		add(IDProposal, "proposal", proposal)
		add(IDNoAction, "no_action", nil)
		if len(proposal) >= 2 {
			for k, a := range proposal {
				if len(sel.Candidates) >= MaxCandidates {
					break
				}
				if irreversible(a) {
					rest := append(slices.Clone(proposal[:k]), proposal[k+1:]...)
					add(fmt.Sprintf("without_%d", k+1), "leave_one_out", rest)
				}
			}
		} else if adj, ok := adjust(proposal[0], sel); ok {
			v := s.ValidateBatch(adj)
			valid := true
			for _, r := range v {
				valid = valid && r.OK
			}
			if valid {
				add(IDAdjusted, "adjustment", []api.Action{adj})
			} else {
				sel.Omitted = append(sel.Omitted, Omitted{ID: IDAdjusted, Actions: canon([]api.Action{adj}), Validation: v})
			}
		}
	}
	if rule == SelectionRuleV2 {
		sel.Candidates, sel.Removed = dedupe(sel.Candidates)
	}
	if s.Digests() != before {
		return nil, fmt.Errorf("alternatives: selection changed the session")
	}
	return sel, nil
}

// dedupe is rule v2's only change: of the candidates in their listed
// order, each whose canonical batch equals an earlier one's is removed
// (and recorded), the first kept. It removes, and never adds, reorders or
// changes a candidate, so it cannot exceed the limit or repair anything.
func dedupe(cands []Candidate) ([]Candidate, []Removed) {
	kept, first := []Candidate{}, map[string]string{}
	var removed []Removed
	for _, c := range cands {
		k := batchKey(c.Actions)
		if id, ok := first[k]; ok {
			removed = append(removed, Removed{ID: c.ID, Source: c.Source, SameAs: id, Actions: c.Actions, BatchSha256: k})
			continue
		}
		first[k] = c.ID
		kept = append(kept, c)
	}
	return kept, removed
}

// candidateSha256 is a candidate's identity: the SHA-256 of the rule, its
// ID and its canonical actions.
func candidateSha256(rule, id string, actions []api.Action) string {
	return sha(struct {
		Rule, ID string
		Actions  []api.Action
	}{rule, id, canon(actions)})
}

// batchKey is the SHA-256 of a batch's canonical form, in batch order: the
// identity the branch builder gives a batch (session.checkAlternatives).
func batchKey(actions []api.Action) string { return sha(canon(actions)) }

// adjust is the rule's one adjustment of a single action, if it defines
// one within the booking horizon.
func adjust(a api.Action, sel *Selection) (api.Action, bool) {
	today, err := simtime.ParseDate(sel.Today)
	if err != nil {
		return a, false
	}
	last := today.AddDays(sel.HorizonDays)
	later := func(date string, days int64) (string, bool) {
		d, err := simtime.ParseDate(date)
		if err != nil {
			return "", false
		}
		n := d.AddDays(days)
		return n.String(), n.DaysSince(last) <= 0
	}
	b := canon([]api.Action{a})[0]
	var ok bool
	switch {
	case a.Type == api.ActTerminate:
		b.LastDay, ok = later(a.LastDay, 14)
	case a.Type == api.ActHire:
		b.StartDate, ok = later(a.StartDate, 14)
	case a.Type == api.ActContractTeleradiology && a.Op == api.TeleOpStart && a.Scope == "all":
		b.Scope, ok = "urgent", true
	case a.Type == api.ActContractTeleradiology && a.Op == api.TeleOpStart && a.Scope == "urgent":
		b.Effective, ok = later(a.Effective, 14)
	case a.Type == api.ActSetRoster && a.Op == api.RosterOpSet:
		b.Effective, ok = later(a.Effective, 7)
	}
	return b, ok
}

// VerifySelection recomputes the selection at s under its own rule and
// requires it equal.
func VerifySelection(s *session.Session, sel *Selection) error {
	if pin := ruleSha256(sel.Rule); pin == "" || sel.RuleSha256 != pin {
		return fmt.Errorf("alternatives: selection rule %s (%s) is not this build's", sel.Rule, sel.RuleSha256)
	}
	again, err := SelectRule(s, sel.Rule, sel.Proposal)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(again, sel) {
		return fmt.Errorf("alternatives: the selection does not replay")
	}
	return nil
}

// canon is a batch as JSON-decoded actions ([] for none).
func canon(actions []api.Action) []api.Action {
	if len(actions) == 0 {
		return []api.Action{}
	}
	raw, err := json.Marshal(actions)
	if err != nil {
		panic(err)
	}
	var out []api.Action
	if err := json.Unmarshal(raw, &out); err != nil {
		panic(err)
	}
	return out
}

func sha(v any) string {
	raw, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return sha256Hex(raw)
}

func sha256Hex(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}
