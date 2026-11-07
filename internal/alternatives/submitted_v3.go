package alternatives

import (
	"bytes"
	"encoding/json"
	"slices"
	"strings"

	"github.com/georgezero/dreamteam-go/internal/api"
)

const SubmittedSchemaV3 = "dreamteam.submitted-action/v2"

// SubmittedAction is the decision of record. Empty submissions have one identity.
// Action and batch order are retained: application and validation use earlier items.
// Contract windows are a set in parseTeleWindows and teleCanon. Hire skills
// and cover windows follow hireCanon; sameCovers defines roster cover equality.
// Breaks stay ordered because parseRoster validates chronological order.
// Production callers supply the decision date to resolve extend_shift
// defaults exactly as the protocol and engine do.

func CanonicalSubmittedV3(batches [][]api.Action, decisionDates ...string) SubmittedAction {
	out := SubmittedAction{Schema: SubmittedSchemaV3, Batches: [][]api.Action{}}
	for _, batch := range batches {
		if len(batch) == 0 {
			continue
		}
		b := canon(batch)
		for i := range b {
			if b[i].Type == api.ActExtendShift && b[i].Date == "" && len(decisionDates) > 0 {
				b[i].Date = decisionDates[0]
			}
			if b[i].Type == api.ActHire {
				rank := func(s string) int {
					if n := slices.Index(viewSkills, s); n >= 0 {
						return n
					}
					return len(viewSkills)
				}
				slices.SortStableFunc(b[i].Skills, func(a, c string) int {
					if ra, rc := rank(a), rank(c); ra != rc {
						return ra - rc
					}
					return strings.Compare(a, c)
				})
			}
			if b[i].Roster != nil && (b[i].Type == api.ActHire || b[i].Type == api.ActSetRoster) {
				for _, week := range []*api.Week{&b[i].Roster.A, &b[i].Roster.B} {
					for _, day := range week.Days() {
						if day != nil {
							if day.Breaks == nil {
								day.Breaks = [][2]string{}
							}
							if day.Covers == nil {
								day.Covers = []api.Cover{}
							}
							slices.SortStableFunc(day.Covers, func(a, c api.Cover) int {
								if a.Room != c.Room {
									return strings.Compare(a.Room, c.Room)
								}
								return strings.Compare(a.From, c.From)
							})
						}
					}
				}
			}
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
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&value); err != nil {
		panic(err)
	}
	raw, _ = json.Marshal(value)
	out.Canonical = string(raw)
	out.Sha256 = sha256Hex(raw)
	return out
}
