package agent

import (
	"slices"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/session"
	"github.com/georgezero/dreamteam-go/internal/sim"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Catalog is the actions an agent can submit now: for each type its JSON
// Schema (the api.Action fields it takes), its constraints in words, and
// its current targets, read from the public action contract (the offer
// table) and the session's own views (the observation, roster plans,
// reading allocations, teleradiology). Validation stays the simulator's:
// the catalog says what to send, validate says whether it is accepted.
type Catalog struct {
	Time string `json:"time"`
	// Batch rules: atomicity and size.
	Batch []string     `json:"batch"`
	Types []ActionType `json:"types"`
}

// ActionType is one action type.
type ActionType struct {
	Type        string         `json:"type"`
	Summary     string         `json:"summary"`
	Schema      map[string]any `json:"schema"`
	Constraints []string       `json:"constraints"`
	Targets     any            `json:"targets"`
}

// JSON Schema helpers.
func object(required []string, props map[string]any) map[string]any {
	if required == nil {
		required = []string{}
	}
	return map[string]any{"type": "object", "required": required, "properties": props, "additionalProperties": false}
}
func str(desc string) map[string]any { return map[string]any{"type": "string", "description": desc} }
func enum(desc string, vals ...string) map[string]any {
	return map[string]any{"type": "string", "enum": vals, "description": desc}
}
func integer(desc string, lo, hi int64) map[string]any {
	return map[string]any{"type": "integer", "minimum": lo, "maximum": hi, "description": desc}
}
func constType(t string) map[string]any { return map[string]any{"const": t} }

const (
	dateDesc = "a practice-local date YYYY-MM-DD"
	hhmm     = "a practice-local time HH:MM"
)

// shiftSchema is one shift of a roster (api.Shift).
var shiftSchema = map[string]any{
	"oneOf": []any{map[string]any{"type": "null"}, object([]string{"start", "end", "breaks", "assignment", "covers"}, map[string]any{
		"start":      str(hhmm),
		"end":        str(hhmm),
		"breaks":     map[string]any{"type": "array", "items": map[string]any{"type": "array", "items": str(hhmm), "minItems": 2, "maxItems": 2}},
		"assignment": str("a room ID, or \"desk\" for the front desk"),
		"covers": map[string]any{"type": "array", "items": object([]string{"room", "from", "until"}, map[string]any{
			"room": str("a room ID"), "from": str(hhmm), "until": str(hhmm)})},
	})},
}

// rosterSchema is a two-week rota (api.Roster): weeks A and B, a shift or
// null for every weekday.
var rosterSchema = func() map[string]any {
	week := map[string]any{}
	for _, d := range []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"} {
		week[d] = shiftSchema
	}
	w := object([]string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}, week)
	return object([]string{"A", "B"}, map[string]any{"A": w, "B": w})
}()

// Targets of the V0.1 actions and the staffing actions.
type (
	// ExtendTarget is an employee whose shift can be extended.
	ExtendTarget struct {
		Employee    string `json:"employee"`
		Name        string `json:"name"`
		Role        string `json:"role"`
		ShiftToday  string `json:"shiftToday,omitempty"`
		ApprovedMin int64  `json:"approvedMin,omitempty"`
		CapAt       string `json:"capAt,omitempty"`
	}
	// OfferTarget is one recruitable role and skill set (the offer table).
	OfferTarget struct {
		Role               string   `json:"role"`
		Skills             []string `json:"skills"`
		StandardRateCents  int64    `json:"standardRateCents"`
		StandardLeadDays   int64    `json:"standardLeadDays"`
		PremiumRateCents   int64    `json:"premiumRateCents"`
		PremiumLeadDays    int64    `json:"premiumLeadDays"`
		RecruitingFeeCents int64    `json:"recruitingFeeCents"`
	}
	// HireTargets are the offers, the dates and rooms a hire can use.
	HireTargets struct {
		Offers            []OfferTarget `json:"offers"`
		EarliestStartDate string        `json:"earliestStartDate"`
		LatestStartDate   string        `json:"latestStartDate"`
		Rooms             []RoomTarget  `json:"rooms"`
		Technologists     int           `json:"technologists"` // counted against the cap now
		MaxTechnologists  int           `json:"maxTechnologists"`
		Desk              int           `json:"desk"`
		MaxDesk           int           `json:"maxDesk"`
	}
	// RoomTarget is a room an assignment or cover can name.
	RoomTarget struct {
		ID       string `json:"id"`
		Modality string `json:"modality"`
	}
	// TerminateTarget is an employee who can be given notice.
	TerminateTarget struct {
		Employee        string `json:"employee"`
		Name            string `json:"name"`
		Role            string `json:"role"`
		Employment      string `json:"employment"`
		EarliestLastDay string `json:"earliestLastDay"`
		SeveranceCents  int64  `json:"severanceCents"`
	}
)

// catalog builds the catalog from the session's views now.
func catalog(s *session.Session) Catalog {
	o := s.Observe(0)
	cal := s.Calendar()
	today := cal.DayIndex(s.Now())
	date := func(day int64) string { return cal.Date(cal.StartOfDay(day)).String() }
	horizon := o.Practice.BookingHorizonDays
	c := Catalog{Time: o.Time, Batch: []string{
		"A batch is 1–64 actions applied at the current time.",
		"apply applies a batch only if every action in it is valid; otherwise nothing changes and each action's result says why.",
		"V0.2 staffing actions (hire, terminate, set_roster, set_reading_allocation, contract_teleradiology) cannot share a batch with V0.1 actions (mixed_batch).",
		"Two extend_shift actions for the same employee and day go in separate batches.",
	}}

	var extend []ExtendTarget
	var terminate []TerminateTarget
	techs, desk := 0, 0
	for _, e := range o.Staff {
		if e.Role != "technologist" && e.Role != "front_desk" {
			continue
		}
		if e.Employment != api.EmploymentTerminated {
			if e.Role == "technologist" {
				techs++
			} else {
				desk++
			}
		}
		if e.HourlyCents > 0 && e.Employment == api.EmploymentActive {
			extend = append(extend, ExtendTarget{Employee: e.ID, Name: e.Name, Role: e.Role, ShiftToday: e.Shift, ApprovedMin: e.ApprovedMin, CapAt: e.CapAt})
		}
		if e.Leaving == nil && e.Employment != api.EmploymentTerminated {
			last := today + sim.NoticeDays
			if e.Hired != nil {
				if sd, err := simtime.ParseDate(e.Hired.StartDate); err == nil {
					last = max(last, cal.DayIndex(cal.StartOfDate(sd))+sim.NoticeDays)
				}
			}
			terminate = append(terminate, TerminateTarget{Employee: e.ID, Name: e.Name, Role: e.Role, Employment: e.Employment,
				EarliestLastDay: date(last), SeveranceCents: e.SeveranceCents})
		}
	}
	var offers []OfferTarget
	for _, f := range sim.Offers {
		offers = append(offers, OfferTarget{Role: string(f.Role), Skills: slices.Clone(f.Skills), StandardRateCents: f.StandardCents,
			StandardLeadDays: f.StandardLead, PremiumRateCents: f.PremiumCents, PremiumLeadDays: f.PremiumLead, RecruitingFeeCents: f.FeeCents})
	}
	var rooms []RoomTarget
	for _, r := range o.Rooms {
		rooms = append(rooms, RoomTarget{ID: r.ID, Modality: r.Modality})
	}
	var rosters []api.RosterPlan
	for _, e := range o.Staff {
		if p, ok := s.RosterPlan(e.ID); ok && p.Editable {
			rosters = append(rosters, p)
		}
	}
	var allocations []api.ReadingAllocation
	for _, a := range s.ReadingAllocations() {
		if a.Editable {
			allocations = append(allocations, a)
		}
	}
	tele := s.Teleradiology()
	var cancellable []api.TeleContract
	for _, k := range tele.Contracts {
		if k.EndDate == "" && k.Status != "ended" {
			cancellable = append(cancellable, k)
		}
	}

	c.Types = []ActionType{
		{Type: api.ActExtendShift, Summary: "Approve overtime: extend a technologist's or desk employee's shift today or on a later day.",
			Schema: object([]string{"type", "employee", "minutes"}, map[string]any{
				"type": constType(api.ActExtendShift), "employee": str("an employee ID from targets"),
				"minutes": integer("extension minutes", 1, int64(sim.MaxExtension/simtime.Minute)),
				"date":    str(dateDesc + "; today when omitted, at most the booking horizon ahead")}),
			Constraints: []string{"Within the employee's shift cap; ahead of time, the new slots open for booking at once.",
				"The date is today or a later day within the booking horizon (" + date(today+horizon) + ")."},
			Targets: extend},
		{Type: api.ActSetWalkInPolicy, Summary: "Set the walk-in policy.",
			Schema: object([]string{"type", "policy"}, map[string]any{
				"type":   constType(api.ActSetWalkInPolicy),
				"policy": enum("the policy", api.WalkInAcceptAlways, api.WalkInAcceptIfGap, api.WalkInAcceptUntil, api.WalkInScheduleLater),
				"cutoff": str(hhmm + "; required for accept_until only")}),
			Constraints: []string{"cutoff is given with accept_until and only then."},
			Targets:     o.Policies},
		{Type: api.ActSetWorklistPolicy, Summary: "Set the order radiologists read non-urgent studies in (urgent studies always come first).",
			Schema: object([]string{"type", "policy"}, map[string]any{
				"type":   constType(api.ActSetWorklistPolicy),
				"policy": enum("the policy", api.WorklistUrgentThenFIFO, api.WorklistFIFO, api.WorklistOldestFirst, api.WorklistShortestFirst)}),
			Constraints: []string{},
			Targets:     o.Policies},
		{Type: api.ActHire, Summary: "Recruit a technologist or front desk employee from the offer table.",
			Schema: object([]string{"type", "requestId", "role", "skills", "rateCents", "contractedWeeklyMinutes", "roster", "startDate"}, map[string]any{
				"type": constType(api.ActHire), "requestId": str("1–128 characters, unique to this hire; resending the same hire is a no-op"),
				"role":                    enum("the role", string(sim.Technologist), string(sim.FrontDesk)),
				"skills":                  map[string]any{"type": "array", "items": str("a skill"), "description": "one of the offers' skill sets"},
				"rateCents":               map[string]any{"type": "integer", "description": "the offer's standard or premium hourly rate"},
				"contractedWeeklyMinutes": integer("guaranteed weekly minutes; the roster may not exceed them", sim.MinContractMin, sim.MaxContractMin),
				"roster":                  rosterSchema,
				"startDate":               str(dateDesc + ", after today and within the booking horizon; at least the offer's lead ahead")}),
			Constraints: []string{
				"The recruiting fee is charged at once; paid onboarding and training come before patient capacity.",
				"Roster: qualified rooms only, at most 6 working days a week, a 30-minute break in a shift over 6 hours, 10 hours' rest from the shift cap to the next start.",
				"Headcount caps count recruiting, onboarding, active and notice staff."},
			Targets: HireTargets{Offers: offers, EarliestStartDate: date(today + 1), LatestStartDate: date(today + horizon), Rooms: rooms,
				Technologists: techs, MaxTechnologists: sim.MaxTechnologists, Desk: desk, MaxDesk: sim.MaxDesk}},
		{Type: api.ActTerminate, Summary: "Give notice to an hourly technologist or front desk employee.",
			Schema: object([]string{"type", "employee", "lastDay"}, map[string]any{
				"type": constType(api.ActTerminate), "employee": str("an employee ID from targets"),
				"lastDay": str(dateDesc + ", inclusive, at least the earliest last day")}),
			Constraints: []string{"Contract pay continues to the last day; two contracted weeks' severance is recognized at once.",
				"Booked patients who lose their technologist are rebooked or lost; preview names them."},
			Targets: terminate},
		{Type: api.ActSetRoster, Summary: "Set or remove a dated roster version of an hourly employee.",
			Schema: object([]string{"type", "op", "employee", "effective"}, map[string]any{
				"type": constType(api.ActSetRoster), "op": enum("set a version, or remove a pending one", api.RosterOpSet, api.RosterOpRemove),
				"employee": str("an employee ID from targets"), "effective": str(dateDesc + ": a Monday at least the earliest effective date"),
				"roster": rosterSchema}),
			Constraints: []string{"roster is given with op set and only then.", "The contract does not change: a roster may not exceed it."},
			Targets:     rosters},
		{Type: api.ActSetReadingAllocation, Summary: "Set or remove a dated site reading allocation of a salaried radiologist.",
			Schema: object([]string{"type", "op", "employee", "effective"}, map[string]any{
				"type": constType(api.ActSetReadingAllocation), "op": enum("set a version, or remove a pending one", api.RosterOpSet, api.RosterOpRemove),
				"employee": str("an employee ID from targets"), "effective": str(dateDesc + ", at least the earliest effective date"),
				"siteAllocationBP": integer("the site share of paid time in basis points; with op set only", sim.MinSiteBP, sim.MaxSiteBP)}),
			Constraints: []string{"It changes the radiologist's site reading blocks and the salary charged to the practice."},
			Targets:     allocations},
		{Type: api.ActContractTeleradiology, Summary: "Start a teleradiology contract from the plan catalog, or fix an open-ended contract's end.",
			Schema: object([]string{"type", "op"}, map[string]any{
				"type": constType(api.ActContractTeleradiology), "op": enum("start or cancel", api.TeleOpStart, api.TeleOpCancel),
				"requestId": str("op start: 1–128 characters, unique to this contract"),
				"plan":      str("op start: a plan ID from targets"),
				"windows":   map[string]any{"type": "array", "items": enum("a window", "weeknights", "saturday", "sunday"), "minItems": 1, "maxItems": 3},
				"scope":     enum("op start: which studies", "urgent", "all"),
				"effective": str("op start: " + dateDesc + ", the activation, at least 14 days ahead"),
				"contract":  str("op cancel: a contract ID from targets"),
				"endDate":   str("op cancel: " + dateDesc + ", exclusive, at least the contract's earliest end date")}),
			Constraints: []string{"Contracts never overlap; they share the practice's two remote lanes.",
				"A cancellation ends at least 30 days ahead and at least 30 days after activation."},
			Targets: struct {
				Plans                []api.TelePlan     `json:"plans"`
				EarliestEffective    string             `json:"earliestEffective"`
				CancellableContracts []api.TeleContract `json:"cancellableContracts"`
			}{tele.Plans, date(today + 14), cancellable}},
	}
	return c
}
