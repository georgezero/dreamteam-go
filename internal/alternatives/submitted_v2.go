package alternatives

import (
	"encoding/json"
	"slices"

	"github.com/georgezero/dreamteam-go/internal/api"
)

const SubmittedSchema = "dreamteam.submitted-action/v1"

// SubmittedAction is the decision of record. Empty submissions have one identity.
// Action and batch order are retained: application and validation use earlier items.
// Contract windows are a set in parseTeleWindows and teleCanon.
type SubmittedAction struct {
	Schema    string         `json:"schema"`
	Batches   [][]api.Action `json:"batches"`
	Canonical string         `json:"canonical"`
	Sha256    string         `json:"sha256"`
}

func CanonicalSubmitted(batches [][]api.Action) SubmittedAction {
	out := SubmittedAction{Schema: SubmittedSchema, Batches: [][]api.Action{}}
	for _, batch := range batches {
		if len(batch) == 0 {
			continue
		}
		b := canon(batch)
		for i := range b {
			if b[i].Type == api.ActContractTeleradiology && b[i].Op == api.TeleOpStart {
				order := []string{"weeknights", "saturday", "sunday"}
				slices.SortStableFunc(b[i].Windows, func(a, c string) int { return slices.Index(order, a) - slices.Index(order, c) })
			}
		}
		out.Batches = append(out.Batches, b)
	}
	// Maps supply deterministic object-key order independently of struct layout.
	raw, _ := json.Marshal(out.Batches)
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		panic(err)
	}
	raw, _ = json.Marshal(value)
	out.Canonical = string(raw)
	out.Sha256 = sha256Hex(raw)
	return out
}
