package bench

import (
	"encoding/json"
	"slices"

	"github.com/georgezero/dreamteam-go/internal/agent"
)

// A Policy decides at each decision point, from what the protocol shows
// it, which batches to submit (agent.Policy). Its client reaches the
// session through the gate: it can read and check, never apply, advance,
// save or evaluate. A new instance plays each run, so nothing a policy
// keeps between decisions outlives its run.
//
// An external policy process can stand in for a built-in one by
// implementing the same exchange: it is handed the gated connection at
// each decision point and answers with its batches (docs/BENCHMARK.md §4).

// PolicySpec identifies a policy: its ID, a version that changes whenever
// its behaviour does, what it does, and a constructor for one run.
type PolicySpec struct {
	ID          string              `json:"id"`
	Version     string              `json:"version"`
	Description string              `json:"description"`
	New         func() agent.Policy `json:"-"`
}

// policies are the built-in policies, in a stable order.
var policies = []PolicySpec{
	{ID: "noop", Version: "1", Description: "never submits an action (agent.Noop)",
		New: func() agent.Policy { return agent.Noop{} }},
	{ID: "greedy", Version: "1", Description: "the V0.3 protocol demonstration (agent.Greedy): raises one radiologist's site reading allocation by 5 points whenever the oldest unread study is over a day old; not a strategy",
		New: func() agent.Policy { return agent.Greedy{} }},
	{ID: "rules", Version: "1", Description: "a modest rule-based baseline: reading allocation follows the reading backlog with hysteresis, every change validated and previewed first",
		New: func() agent.Policy { return &Rules{} }},
}

// PolicyIDs lists the built-in policies' IDs.
func PolicyIDs() []string {
	ids := make([]string, len(policies))
	for i, p := range policies {
		ids[i] = p.ID
	}
	return ids
}

func lookupPolicy(id string) (PolicySpec, bool) {
	i := slices.IndexFunc(policies, func(p PolicySpec) bool { return p.ID == id })
	if i < 0 {
		return PolicySpec{}, false
	}
	return policies[i], true
}

// Rules is the rule-based baseline (version 1). It keeps a radiologist's
// site reading allocation in step with the reading backlog, from the
// public observation and the action catalog only:
//
//   - pressure: the oldest unfinished study is more than 24 hours old.
//     The editable radiologist with the lowest allocation in effect,
//     room to grow and no pending change gets 5 points more (at most the
//     maximum), from the earliest lawful date.
//   - clear: the oldest unfinished study is at most 8 hours old at three
//     decision points in a row. The editable radiologist furthest above
//     the scenario allocation, with no pending change, gets 5 points less
//     (never below the scenario allocation).
//   - otherwise nothing.
//
// A change is submitted only if validate finds it valid and the staffing
// preview accepts it without displacing a booked patient. One change per
// decision point. The thresholds are round numbers chosen before any
// benchmark run and are not tuned to outcomes.
type Rules struct {
	clear int // consecutive decision points with a clear backlog
}

func (*Rules) Name() string { return "rules" }

const (
	rulesPressureMin = 24 * 60
	rulesClearMin    = 8 * 60
	rulesClearTurns  = 3
	rulesStepBP      = 500
)

func (r *Rules) Decide(c *agent.Client) ([][]json.RawMessage, error) {
	var obs struct {
		Observation struct {
			Worklist struct {
				OldestMin int64 `json:"oldestMin"`
			} `json:"worklist"`
		} `json:"observation"`
	}
	if err := c.Call("observe", map[string]any{"sections": []string{"worklist"}, "compact": true, "recentLog": 0}, &obs); err != nil {
		return nil, err
	}
	oldest := obs.Observation.Worklist.OldestMin
	pressure := oldest > rulesPressureMin
	if oldest <= rulesClearMin {
		r.clear++
	} else {
		r.clear = 0
	}
	if !pressure && r.clear < rulesClearTurns {
		return nil, nil
	}
	var cat struct {
		Types []struct {
			Type    string          `json:"type"`
			Targets json.RawMessage `json:"targets"`
		} `json:"types"`
	}
	if err := c.Call("actions", nil, &cat); err != nil {
		return nil, err
	}
	var targets []struct {
		Employee          string `json:"employee"`
		InEffectBP        int64  `json:"inEffectBP"`
		Editable          bool   `json:"editable"`
		EarliestEffective string `json:"earliestEffective"`
		MaxBP             int64  `json:"maxBP"`
		Versions          []struct {
			Effective string `json:"effective"`
			SiteBP    int64  `json:"siteBP"`
			Pending   bool   `json:"pending"`
		} `json:"versions"`
	}
	for _, t := range cat.Types {
		if t.Type == "set_reading_allocation" {
			if err := json.Unmarshal(t.Targets, &targets); err != nil {
				return nil, err
			}
		}
	}
	best, bp := -1, int64(0)
	for i, a := range targets {
		scenarioBP, pending := a.InEffectBP, false
		for _, v := range a.Versions {
			pending = pending || v.Pending
			if v.Effective == "" {
				scenarioBP = v.SiteBP
			}
		}
		if !a.Editable || pending {
			continue
		}
		switch {
		case pressure && a.InEffectBP < a.MaxBP:
			if best < 0 || a.InEffectBP < targets[best].InEffectBP {
				best, bp = i, min(a.MaxBP, a.InEffectBP+rulesStepBP)
			}
		case !pressure && a.InEffectBP > scenarioBP:
			if best < 0 || a.InEffectBP-scenarioBP > bp {
				best, bp = i, a.InEffectBP-scenarioBP
			}
		}
	}
	if best < 0 {
		return nil, nil
	}
	a := targets[best]
	if !pressure {
		bp = max(a.InEffectBP-bp, a.InEffectBP-rulesStepBP)
	}
	act, err := json.Marshal(map[string]any{"type": "set_reading_allocation", "op": "set", "employee": a.Employee,
		"effective": a.EarliestEffective, "siteAllocationBP": bp})
	if err != nil {
		return nil, err
	}
	batch := []json.RawMessage{act}
	var v struct {
		Valid bool `json:"valid"`
	}
	if err := c.Call("validate", map[string]any{"actions": batch}, &v); err != nil {
		return nil, err
	}
	var p struct {
		Preview struct {
			OK            bool              `json:"ok"`
			Displacements []json.RawMessage `json:"displacements"`
		} `json:"preview"`
	}
	if err := c.Call("preview", map[string]any{"actions": batch}, &p); err != nil {
		return nil, err
	}
	if !v.Valid || !p.Preview.OK || len(p.Preview.Displacements) > 0 {
		return nil, nil
	}
	if !pressure {
		r.clear = 0 // the next decrease waits for three more clear decision points
	}
	return [][]json.RawMessage{batch}, nil
}
