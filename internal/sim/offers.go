package sim

import (
	"fmt"
	"slices"
	"strings"

	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// The fixed hire offer table and employment terms (V0.2_DESIGN §5.3, §6.3;
// owner decision 2026-10-15: "Use the approved fixed offer table and
// employment lifecycle rules"). Only these role and skill sets can be
// recruited, each at exactly one of two pay tiers: the premium tier buys a
// shorter recruiting lead, not productivity. There is no candidate draw:
// an accepted hire starts on its start date.

// Offer is one recruitable role and skill set.
type Offer struct {
	Role   Role
	Skills []string // canonical order
	// Standard and premium hourly rates (cents) and recruiting leads (days).
	StandardCents, PremiumCents int64
	StandardLead, PremiumLead   int64
	FeeCents                    int64
}

// Offers is the fixed V0.2 offer table.
var Offers = []Offer{
	{FrontDesk, []string{"registration"}, 2100, 2400, 14, 7, 100_000},
	{Technologist, []string{"XR"}, 3600, 4000, 28, 21, 300_000},
	{Technologist, []string{"CT"}, 4000, 4400, 28, 21, 300_000},
	{Technologist, []string{"XR", "CT"}, 3800, 4200, 28, 21, 300_000},
	{Technologist, []string{"US"}, 4400, 4800, 28, 21, 300_000},
	{Technologist, []string{"MR"}, 4500, 4900, 28, 21, 300_000},
}

// Employment terms (V0.2_DESIGN §5.2–§5.4, §6.3).
const (
	// Contracted weekly minutes a hire may have.
	MinContractMin, MaxContractMin = 1, 2400
	// Attended paid training before independent work, by role.
	TechTraining = 40 * simtime.Hour
	DeskTraining = 24 * simtime.Hour
	// Training must complete within this many calendar days of the start.
	TrainingWindowDays = 60
	// A termination's last day is at least this many days after both
	// today and the start date.
	NoticeDays = 14
	// Concurrent headcount caps: recruiting, onboarding, active and notice
	// all count; terminated employees do not.
	MaxTechnologists = 8
	MaxDesk          = 4
	// Roster rules for a hire (§6.4): at most six working days a week, an
	// unpaid break of at least BreakMin in a shift longer than
	// LongShift, no uninterrupted stretch longer than LongShift, and
	// MinRest between the latest permitted release (the shift cap) and
	// the next start.
	MaxWorkDays = 6
	LongShift   = 6 * simtime.Hour
	BreakMin    = 30 * simtime.Minute
	MinRest     = 10 * simtime.Hour
)

// modalityOrder is the canonical skill order (offer table order).
var skillOrder = []string{"registration", "XR", "CT", "US", "MR"}

// canonicalSkills sorts skills into the table's order (unknown skills
// last, alphabetically); it does not deduplicate.
func canonicalSkills(skills []string) []string {
	out := slices.Clone(skills)
	rank := func(s string) int {
		if i := slices.Index(skillOrder, s); i >= 0 {
			return i
		}
		return len(skillOrder)
	}
	slices.SortStableFunc(out, func(a, b string) int {
		if ra, rb := rank(a), rank(b); ra != rb {
			return ra - rb
		}
		return strings.Compare(a, b)
	})
	return out
}

// findOffer returns the offer for a role and skill set, or nil.
func findOffer(role Role, skills []string) *Offer {
	c := canonicalSkills(skills)
	for k := range Offers {
		if Offers[k].Role == role && slices.Equal(Offers[k].Skills, c) {
			return &Offers[k]
		}
	}
	return nil
}

// trainingFor is the attended training a role needs.
func trainingFor(r Role) simtime.Duration {
	if r == FrontDesk {
		return DeskTraining
	}
	return TechTraining
}

// hours formats a duration in hours: "40 h", "27.5 h", "7 h 20 min".
func hours(d simtime.Duration) string {
	h, m := d/simtime.Hour, d%simtime.Hour/simtime.Minute
	switch m {
	case 0:
		return fmt.Sprintf("%d h", h)
	case 30:
		return fmt.Sprintf("%d.5 h", h)
	}
	return fmt.Sprintf("%d h %d min", h, m)
}

// offerLabel names an employee's offer: "XR+CT technologist", "front desk".
func offerLabel(t *EmployeeConfig) string {
	if t.Role == FrontDesk {
		return "front desk"
	}
	return strings.Join(canonicalSkills(t.Skills), "+") + " " + string(t.Role)
}
