package alternatives

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/session"
	"github.com/georgezero/dreamteam-go/internal/sim"
)

// The comparison contract, dreamteam.management-alternatives/v1, in two
// audiences. The boundary is in the types: Agent has no field that can
// hold a projection, and BuildAgent reads only the decision point (the
// engine's validation, previews and observation), never a branch. Only
// Evaluation, built after the decision from branches, holds class B.

// Contract identities and classes.
const (
	ContractSchema     = "dreamteam.management-alternatives/v1"
	AudienceAgent      = "agent"
	AudienceEvaluation = "evaluation"

	ClassReported    = "A1" // read from an engine field
	ClassDerived     = "A2" // computed from engine fields by a stated rule
	ClassProjected   = "B"  // read from a branch under a declared continuation
	ClassUnavailable = "C"  // not available: listed with the reason, no value

	UnitCents = "cents"
	UnitCount = "count"
	UnitDays  = "days"
	UnitSec   = "seconds"
	UnitDate  = "date"
	UnitText  = "text"
)

// ErrContract: a comparison is malformed, contradictory or edited.
var ErrContract = errors.New("comparison does not satisfy " + ContractSchema)

// Value is one known value with its class and engine source. Number holds
// cents, counts, days or seconds (Unit says which); Date and Text hold the
// others.
type Value struct {
	Key        string `json:"key"`
	Label      string `json:"label"`
	Class      string `json:"class"`
	Unit       string `json:"unit"`
	Number     *int64 `json:"number,omitempty"`
	Date       string `json:"date,omitempty"`
	Text       string `json:"text,omitempty"`
	When       string `json:"when,omitempty"`
	Source     string `json:"source"`
	Basis      string `json:"basis,omitempty"`
	Derivation string `json:"derivation,omitempty"`
}

// Unavailable is a relevant consequence with no value.
type Unavailable struct {
	Key    string `json:"key"`
	Class  string `json:"class"`
	Reason string `json:"reason"`
}

// Validation is the engine's answer to the batch.
type Validation struct {
	OK      bool               `json:"ok"`
	Results []api.ActionResult `json:"results"`
}

// AgentAlternative is one alternative as the agent may see it.
type AgentAlternative struct {
	ID              string        `json:"id"`
	Source          string        `json:"source"`
	CandidateSha256 string        `json:"candidateSha256"`
	Actions         []api.Action  `json:"actions"`
	Description     string        `json:"description"`
	Validation      Validation    `json:"validation"`
	Facts           []Value       `json:"facts"`
	Estimates       []Value       `json:"estimates"`
	Obligations     []Value       `json:"obligations"`
	Reversibility   Value         `json:"reversibility"`
	Unavailable     []Unavailable `json:"unavailable"`
}

// DecisionPoint, as the agent may see it: no seed and no record.
type DecisionPoint struct {
	Engine      string `json:"engine"`
	Scenario    string `json:"scenarioFingerprint"`
	Time        string `json:"time"`
	StateDigest string `json:"stateDigest"`
}

// SelectionRef names the candidate rule.
type SelectionRef struct {
	Rule       string `json:"rule"`
	RuleSha256 string `json:"ruleSha256"`
	Why        string `json:"why"`
}

// Compared is one immediate (class A) value every accepted alternative
// reports, in one unit.
type Compared struct {
	Key    string           `json:"key"`
	Unit   string           `json:"unit"`
	Values map[string]int64 `json:"values"`
}

// Request is one engine request made to build the comparison, with the
// SHA-256 of the engine's answer.
type Request struct {
	Request        string `json:"request"`
	ResponseSha256 string `json:"responseSha256"`
}

// Agent is the agent audience: classes A1, A2 and C only.
type Agent struct {
	Schema        string             `json:"schema"`
	Audience      string             `json:"audience"`
	DecisionPoint DecisionPoint      `json:"decisionPoint"`
	Selection     SelectionRef       `json:"selection"`
	Alternatives  []AgentAlternative `json:"alternatives"`
	Compared      []Compared         `json:"compared"`
	Provenance    []Request          `json:"provenance"`
	Sha256        string             `json:"sha256"`
}

// BuildAgent builds the agent audience for a selection at s's current
// time. It reads s only through the decision point's validation, previews
// and observation, and checks s's digests unchanged.
func BuildAgent(s *session.Session, sel *Selection) (*Agent, error) {
	if !sel.Applies || len(sel.Candidates) < 2 {
		return nil, fmt.Errorf("%w: the selection lists no alternatives (%s)", ErrContract, sel.Why)
	}
	before := s.Digests()
	o := s.Observe(0)
	a := &Agent{Schema: ContractSchema, Audience: AudienceAgent,
		DecisionPoint: DecisionPoint{Engine: s.Engine(), Scenario: s.Fingerprint(), Time: o.Time, StateDigest: before.State},
		Selection:     SelectionRef{Rule: sel.Rule, RuleSha256: sel.RuleSha256, Why: sel.Why}, Compared: []Compared{}, Provenance: []Request{}}
	if o.Time != sel.At {
		return nil, fmt.Errorf("%w: selection made at %s, session at %s", ErrContract, sel.At, o.Time)
	}
	b := &builder{s: s, o: o, a: a}
	a.Provenance = append(a.Provenance, Request{"observe", sha(o)})
	// The proposal's preview is the source of the guarantee no_action keeps.
	b.proposal = sel.Candidates[0]
	if b.proposal.OK && slices.ContainsFunc(b.proposal.Actions, func(x api.Action) bool { return isStaffing(x.Type) }) {
		p := s.PreviewStaffing(b.proposal.Actions...)
		b.proposalPreview = &p
	}
	for _, c := range sel.Candidates {
		a.Alternatives = append(a.Alternatives, b.alternative(c))
	}
	a.Compared = compare(a.Alternatives)
	if s.Digests() != before {
		return nil, fmt.Errorf("%w: building the comparison changed the session", ErrContract)
	}
	a.Sha256 = a.digest()
	if err := a.check(); err != nil {
		return nil, err
	}
	return a, nil
}

type builder struct {
	s               *session.Session
	o               api.Observation
	a               *Agent
	proposal        Candidate
	proposalPreview *api.StaffingPreview
}

func num(n int64) *int64 { return &n }

func (b *builder) staff(id string) (api.Employee, bool) {
	for _, e := range b.o.Staff {
		if e.ID == id {
			return e, true
		}
	}
	return api.Employee{}, false
}

func (b *builder) describe(actions []api.Action) string {
	if len(actions) == 0 {
		return "No batch: current staffing and every existing commitment continue"
	}
	var parts []string
	for _, x := range actions {
		who := x.Employee
		if e, ok := b.staff(x.Employee); ok {
			who = e.Name + " (" + e.ID + ")"
		}
		switch x.Type {
		case api.ActTerminate:
			parts = append(parts, "terminate "+who+", last day "+x.LastDay)
		case api.ActHire:
			parts = append(parts, "hire request "+x.RequestID+" ("+x.Role+"), start "+x.StartDate)
		case api.ActSetRoster:
			parts = append(parts, "set_roster "+x.Op+" "+who+", effective "+x.Effective)
		case api.ActSetReadingAllocation:
			parts = append(parts, "set_reading_allocation "+x.Op+" "+who+", effective "+x.Effective)
		case api.ActContractTeleradiology:
			parts = append(parts, "contract_teleradiology "+x.Op+" "+x.Plan+x.Contract+", scope "+x.Scope+", effective "+x.Effective+x.EndDate)
		default:
			parts = append(parts, x.Type+" "+who+x.Policy)
		}
	}
	return strings.Join(parts, "; ")
}

// reversibility is the engine's rule for the batch's least reversible
// action.
func reversibility(actions []api.Action) Value {
	v := Value{Key: "reversibility", Label: "Reversibility", Class: ClassReported, Unit: UnitText, Source: "engine rule"}
	switch {
	case len(actions) == 0:
		v.Text, v.Basis = "nothing to reverse", "no batch; a termination stays possible later with at least 14 days' notice"
	case slices.ContainsFunc(actions, func(x api.Action) bool { return x.Type == api.ActTerminate }):
		v.Text, v.Basis = "irreversible", "already_terminating: notice cannot be moved or withdrawn; no action rehires"
	case slices.ContainsFunc(actions, func(x api.Action) bool { return x.Type == api.ActHire }):
		v.Text, v.Basis = "irreversible", "the recruiting fee is not refundable; ending the contract needs notice and severance"
	case slices.ContainsFunc(actions, irreversible):
		v.Text, v.Basis = "irreversible", "a contract start binds its minimum term"
	case slices.ContainsFunc(actions, func(x api.Action) bool { return isStaffing(x.Type) }):
		v.Text, v.Basis = "withdrawable before its effective date", "a pending version can be removed (op remove) or a contract cancelled under its terms"
	default:
		v.Text, v.Basis = "applies now", "operations changes apply at once; a later change can set them again"
	}
	return v
}

func (b *builder) request(name string, v any) {
	b.a.Provenance = append(b.a.Provenance, Request{name, sha(v)})
}

func (b *builder) alternative(c Candidate) AgentAlternative {
	alt := AgentAlternative{ID: c.ID, Source: c.Source, CandidateSha256: c.Sha256, Actions: c.Actions,
		Description: b.describe(c.Actions), Validation: Validation{OK: c.OK, Results: c.Validation},
		Facts: []Value{}, Estimates: []Value{}, Obligations: []Value{}, Reversibility: reversibility(c.Actions), Unavailable: []Unavailable{}}
	b.request("validate "+c.ID, c.Validation)
	if !c.OK {
		return alt // a refused alternative shows only its validation
	}
	if len(c.Actions) == 0 {
		b.keep(&alt)
		return alt
	}
	if slices.ContainsFunc(c.Actions, func(x api.Action) bool { return isStaffing(x.Type) }) {
		p := b.s.PreviewStaffing(c.Actions...)
		b.request("preview "+c.ID, p)
		b.preview(&alt, p)
	} else {
		alt.Unavailable = append(alt.Unavailable, Unavailable{"preview", ClassUnavailable, "operations actions have no engine preview; they apply one by one"})
	}
	if !slices.ContainsFunc(c.Actions, func(x api.Action) bool { return x.Type != api.ActTerminate }) {
		r := b.s.PreviewSlotsClosed(c.Actions...)
		b.request("slotsClosed "+c.ID, r)
		b.slots(&alt, r)
	}
	b.common(&alt)
	return alt
}

func (b *builder) preview(alt *AgentAlternative, p api.StaffingPreview) {
	f := func(v Value) { alt.Facts = append(alt.Facts, v) }
	for i, t := range p.Terminations {
		src := fmt.Sprintf("preview.terminations[%d].", i)
		k := "." + t.Employee
		f(Value{Key: "last_day" + k, Label: "Last day · " + t.Name, Class: ClassReported, Unit: UnitDate, Date: t.LastDay, Source: src + "lastDay"})
		f(Value{Key: "end_date" + k, Label: "Not employed from · " + t.Name, Class: ClassReported, Unit: UnitDate, Date: t.EndDate, Source: src + "endDate"})
		f(Value{Key: "notice_days" + k, Label: "Notice · " + t.Name, Class: ClassReported, Unit: UnitDays, Number: num(t.NoticeDays), Source: src + "noticeDays"})
		f(Value{Key: "severance_recognized_now" + k, Label: "Severance recognized now · " + t.Name, Class: ClassReported, Unit: UnitCents, Number: num(t.SeveranceCents),
			When: "recognized now, paid " + t.SeverancePayday, Source: src + "severanceCents"})
		f(Value{Key: "severance_payday" + k, Label: "Severance payday · " + t.Name, Class: ClassReported, Unit: UnitDate, Date: t.SeverancePayday, Source: src + "severancePayday"})
		f(Value{Key: "voided_rosters" + k, Label: "Pending roster versions voided · " + t.Name, Class: ClassDerived, Unit: UnitCount,
			Number: num(int64(len(t.VoidedRosters))), Text: strings.Join(t.VoidedRosters, ", "), Source: src + "voidedRosters", Derivation: "the number of dates listed"})
		f(Value{Key: "voided_extensions" + k, Label: "Approved extensions voided · " + t.Name, Class: ClassDerived, Unit: UnitCount,
			Number: num(int64(len(t.VoidedExtensions))), Source: src + "voidedExtensions", Derivation: "the number of extensions listed"})
	}
	for i, h := range p.Hires {
		src := fmt.Sprintf("preview.hires[%d].", i)
		k := "." + h.RequestID
		f(Value{Key: "start_date" + k, Label: "Start · " + h.Offer, Class: ClassReported, Unit: UnitDate, Date: h.StartDate, Source: src + "startDate"})
		f(Value{Key: "productive_date" + k, Label: "Independent work from · " + h.Offer, Class: ClassReported, Unit: UnitDate, Date: h.ProductiveDate, Source: src + "productiveDate"})
		f(Value{Key: "rate" + k, Label: "Hourly rate · " + h.Offer + " (" + h.Tier + ")", Class: ClassReported, Unit: UnitCents, Number: num(h.RateCents), Source: src + "rateCents"})
	}
	for i, r := range p.Rosters {
		f(Value{Key: "roster_effective." + r.Employee, Label: "Roster effective · " + r.Name, Class: ClassReported, Unit: UnitDate, Date: r.Effective,
			Source: fmt.Sprintf("preview.rosters[%d].effective", i)})
	}
	f(Value{Key: "displaced_bookings", Label: "Booked appointments displaced (whole batch)", Class: ClassDerived, Unit: UnitCount,
		Number: num(int64(len(p.Displacements))), Source: "preview.displacements", Derivation: "the number of displacements listed"})
	for i, m := range p.Guaranteed {
		f(Value{Key: fmt.Sprintf("guaranteed.%d", i), Label: m.Label, Class: ClassReported, Unit: UnitCents, Number: num(m.Cents), When: m.When,
			Source: fmt.Sprintf("preview.guaranteed[%d].cents", i), Basis: m.Basis})
	}
	for i, m := range p.Forecast {
		alt.Estimates = append(alt.Estimates, Value{Key: fmt.Sprintf("forecast.%d", i), Label: m.Label, Class: ClassReported, Unit: UnitCents,
			Number: num(m.Cents), When: m.When, Source: fmt.Sprintf("preview.forecast[%d].cents", i), Basis: m.Basis})
	}
	if fu := p.Funding; fu != nil {
		f(Value{Key: "funding.commitment", Label: "Commitment tested against funding", Class: ClassReported, Unit: UnitCents, Number: num(fu.CommitmentCents),
			Source: "preview.funding.commitmentCents", Basis: fu.Basis})
		f(Value{Key: "funding.headroom", Label: "Funding headroom", Class: ClassReported, Unit: UnitCents, Number: num(fu.HeadroomCents), Source: "preview.funding.headroomCents"})
		f(Value{Key: "funding.funded", Label: "Funding test", Class: ClassReported, Unit: UnitText, Text: fmt.Sprint(fu.Funded), Source: "preview.funding.funded"})
	}
	if len(p.Displacements) > 0 {
		alt.Unavailable = append(alt.Unavailable, Unavailable{"replacement_acceptance", ClassUnavailable,
			"whether each displaced patient accepts a replacement is decided at commit by the patient's tolerance, which is not observable"})
	}
}

func (b *builder) slots(alt *AgentAlternative, r sim.SlotsClosedReport) {
	if !r.Available {
		alt.Unavailable = append(alt.Unavailable, Unavailable{"slots_closed", ClassUnavailable, "the engine does not report it: " + r.Reason})
		return
	}
	window := "no day in the booking horizon is affected"
	if r.Days > 0 {
		window = r.From + " to " + r.To + " (" + fmt.Sprint(r.Days) + " days in the booking horizon)"
	}
	alt.Facts = append(alt.Facts,
		Value{Key: "slots_closed", Label: "Scheduled slots no longer staffed (booked or not)", Class: ClassReported, Unit: UnitCount, Number: num(int64(r.Total)),
			When: window, Source: "slotsClosed.total", Basis: "template appointment slots bookable now by staffing that the termination leaves unstaffed; capacity, not orders"},
		Value{Key: "slots_closed_booked_now", Label: "Of these, booked now", Class: ClassReported, Unit: UnitCount, Number: num(int64(r.BookedNow)),
			Source: "slotsClosed.bookedNow"})
	for i, m := range r.ByModality {
		alt.Facts = append(alt.Facts, Value{Key: "slots_closed." + m.Modality, Label: "Scheduled slots no longer staffed · " + m.Modality, Class: ClassReported,
			Unit: UnitCount, Number: num(int64(m.Count)), Source: fmt.Sprintf("slotsClosed.byModality[%d].count", i)})
	}
	alt.Unavailable = append(alt.Unavailable,
		Unavailable{"slots_beyond_horizon", ClassUnavailable, "slots after the booking horizon do not exist yet and are not counted"},
		Unavailable{"demand_for_closed_slots", ClassUnavailable, "whether a closed slot would have been booked depends on future orders, which are not known before deciding"})
}

// practice are the practice's existing commitments, the same under every
// accepted alternative: appointments booked ahead, payroll accrued and
// accounts payable.
func (b *builder) practice(alt *AgentAlternative) {
	f := func(v Value) { alt.Facts = append(alt.Facts, v) }
	ob := func(v Value) { alt.Obligations = append(alt.Obligations, v) }
	booked := int64(0)
	for off := int64(1); off <= b.o.Practice.BookingHorizonDays; off++ {
		for _, rs := range b.s.Schedule(off).Rooms {
			for _, sl := range rs.Slots {
				if sl.Status == api.SlotBooked {
					booked++
				}
			}
		}
	}
	f(Value{Key: "booked_ahead", Label: "Appointments booked from tomorrow in the booking horizon", Class: ClassDerived, Unit: UnitCount, Number: num(booked),
		Source: "schedule(+1..+bookingHorizonDays)", Derivation: "booked slots counted day by day, as the run result's bookedAhead"})
	ob(Value{Key: "next_payroll", Label: "Payroll accrued, next payday", Class: ClassReported, Unit: UnitCents, Number: num(b.o.Finance.NextPayroll.Cents),
		When: "paid " + b.o.Finance.NextPayroll.Date, Source: "observe.finance.nextPayroll"})
	ob(Value{Key: "payables", Label: "Accounts payable", Class: ClassReported, Unit: UnitCents, Number: num(b.o.Finance.APCents), Source: "observe.finance.apCents"})
}

// common are the items every accepted batch lists as unavailable, and the
// practice's existing commitments.
func (b *builder) common(alt *AgentAlternative) {
	b.practice(alt)
	alt.Unavailable = append(alt.Unavailable,
		Unavailable{"future_orders_and_revenue", ClassUnavailable, "orders, exams and revenue after today depend on future demand; not projected for the decision"},
		Unavailable{"unmodeled", ClassUnavailable, "staff morale, workload, resignation, clinical outcomes and reputation are not modeled by the engine"})
}

// keep is the no-action alternative: what continues, from the
// observation, for each employee the proposal would terminate, and the
// practice's existing commitments.
func (b *builder) keep(alt *AgentAlternative) {
	f := func(v Value) { alt.Facts = append(alt.Facts, v) }
	ob := func(v Value) { alt.Obligations = append(alt.Obligations, v) }
	terms := slices.ContainsFunc(b.proposal.Actions, func(x api.Action) bool { return x.Type == api.ActTerminate })
	f(Value{Key: "displaced_bookings", Label: "Booked appointments displaced (whole batch)", Class: ClassDerived, Unit: UnitCount, Number: num(0),
		Source: "no batch", Derivation: "no batch is applied, so no booking loses coverage"})
	if terms && !slices.ContainsFunc(b.proposal.Actions, func(x api.Action) bool { return x.Type != api.ActTerminate }) {
		f(Value{Key: "slots_closed", Label: "Scheduled slots no longer staffed (booked or not)", Class: ClassDerived, Unit: UnitCount, Number: num(0),
			Source: "no batch", Derivation: "no batch is applied, so no scheduled slot loses its staffing"})
	}
	for _, x := range b.proposal.Actions {
		if x.Type != api.ActTerminate {
			continue
		}
		e, ok := b.staff(x.Employee)
		if !ok {
			alt.Unavailable = append(alt.Unavailable, Unavailable{"employment." + x.Employee, ClassUnavailable, "no such employee in the observation"})
			continue
		}
		k := "." + e.ID
		i := slices.IndexFunc(b.o.Staff, func(y api.Employee) bool { return y.ID == e.ID })
		src := fmt.Sprintf("observe.staff[%d].", i)
		f(Value{Key: "severance_recognized_now" + k, Label: "Severance recognized now · " + e.Name, Class: ClassDerived, Unit: UnitCents, Number: num(0),
			Source: "no batch", Derivation: "no termination is committed, so no severance is recognized"})
		f(Value{Key: "employment" + k, Label: "Employment · " + e.Name, Class: ClassReported, Unit: UnitText, Text: e.Employment, Source: src + "employment"})
		f(Value{Key: "contract_minutes" + k, Label: "Contracted weekly minutes · " + e.Name, Class: ClassReported, Unit: UnitCount, Number: num(e.ContractMin),
			Source: src + "contractMin"})
		if rp, ok := b.s.RosterPlan(e.ID); ok {
			b.request("rosterPlan "+e.ID, rp)
			for j, v := range rp.Versions {
				if v.InEffect {
					f(Value{Key: "roster_minutes_week_a" + k, Label: "Rostered minutes, week A · " + e.Name, Class: ClassReported, Unit: UnitCount,
						Number: num(v.NetMinA), Source: fmt.Sprintf("rosterPlan(%s).versions[%d].netMinA", e.ID, j)})
					f(Value{Key: "roster_minutes_week_b" + k, Label: "Rostered minutes, week B · " + e.Name, Class: ClassReported, Unit: UnitCount,
						Number: num(v.NetMinB), Source: fmt.Sprintf("rosterPlan(%s).versions[%d].netMinB", e.ID, j)})
				}
			}
			pending := 0
			for _, v := range rp.Versions {
				if v.Pending {
					pending++
				}
			}
			f(Value{Key: "pending_rosters" + k, Label: "Pending roster versions · " + e.Name, Class: ClassDerived, Unit: UnitCount, Number: num(int64(pending)),
				Source: fmt.Sprintf("rosterPlan(%s).versions", e.ID), Derivation: "the number of versions marked pending"})
		}
		ext := 0
		for _, x := range b.o.Extensions {
			if x.Employee == e.ID {
				ext++
			}
		}
		f(Value{Key: "approved_extensions" + k, Label: "Approved extensions · " + e.Name, Class: ClassDerived, Unit: UnitCount, Number: num(int64(ext)),
			Source: "observe.extensions", Derivation: "the number of extensions listed for the employee"})
		ob(Value{Key: "severance_if_terminated_later" + k, Label: "Severance if terminated later · " + e.Name, Class: ClassReported, Unit: UnitCents,
			Number: num(e.SeveranceCents), When: "owed only on a later termination", Source: src + "severanceCents", Basis: "two contracted weeks with benefits"})
		b.guarantee(alt, e)
	}
	b.practice(alt)
	alt.Unavailable = append(alt.Unavailable,
		Unavailable{"future_orders_and_revenue", ClassUnavailable, "orders, exams and revenue after today depend on future demand; not projected for the decision"},
		Unavailable{"unmodeled", ClassUnavailable, "staff morale, workload, resignation, clinical outcomes and reputation are not modeled by the engine"})
}

// guarantee is the weekly contract guarantee no action keeps: the
// engine's own figure in the proposal's preview, cross-checked against the
// observed severance (two contracted weeks), within a cent of rounding.
func (b *builder) guarantee(alt *AgentAlternative, e api.Employee) {
	p := b.proposalPreview
	label := "Contract cost no longer owed · " + e.Name + " (" + e.ID + ")"
	if p != nil {
		for i, m := range p.Forecast {
			if m.Label == label {
				if d := m.Cents - e.SeveranceCents/2; d < -1 || d > 1 {
					break
				}
				alt.Obligations = append(alt.Obligations, Value{Key: "weekly_guarantee." + e.ID, Label: "Contract guarantee continues · " + e.Name, Class: ClassDerived,
					Unit: UnitCents, Number: num(m.Cents), When: "per full week, until a later decision", Source: fmt.Sprintf("preview(proposal).forecast[%d].cents", i),
					Derivation: "the engine's weekly guarantee with benefits for the employee, read from the proposal's preview; it continues with no action. " +
						"Cross-checked: observe.staff.severanceCents / 2, within one cent"})
				return
			}
		}
	}
	alt.Unavailable = append(alt.Unavailable, Unavailable{"weekly_guarantee." + e.ID, ClassUnavailable,
		"the engine reports the weekly guarantee only in the proposal's preview, which did not give it"})
}

// compare lists the immediate numbers every accepted alternative reports
// under the same key and unit (classes A1 and A2 are both exact).
func compare(alts []AgentAlternative) []Compared {
	out := []Compared{}
	var ok []AgentAlternative
	for _, a := range alts {
		if a.Validation.OK {
			ok = append(ok, a)
		}
	}
	if len(ok) < 2 {
		return out
	}
	for _, v := range ok[0].Facts {
		if v.Number == nil {
			continue
		}
		c := Compared{Key: v.Key, Unit: v.Unit, Values: map[string]int64{}}
		for _, a := range ok {
			i := slices.IndexFunc(a.Facts, func(x Value) bool { return x.Key == v.Key && x.Unit == v.Unit && x.Number != nil })
			if i < 0 {
				c.Values = nil
				break
			}
			c.Values[a.ID] = *a.Facts[i].Number
		}
		if c.Values != nil {
			out = append(out, c)
		}
	}
	return out
}

func (a *Agent) digest() string {
	c := *a
	c.Sha256 = ""
	return sha(&c)
}

// Encode is the comparison's canonical bytes.
func (a *Agent) Encode() ([]byte, error) {
	raw, err := json.MarshalIndent(a, "", " ")
	return append(raw, '\n'), err
}

// forbidden words in any key or label: no ranking, score or verdict.
var forbidden = []string{"score", "rank", "optimal", "best", "recommend", "preferred", "better", "worse", "saving", "harm"}

func checkValue(v Value, classes ...string) error {
	if !slices.Contains(classes, v.Class) {
		return fmt.Errorf("%w: %s: class %q not allowed here", ErrContract, v.Key, v.Class)
	}
	if v.Key == "" || v.Label == "" || v.Source == "" || v.Unit == "" {
		return fmt.Errorf("%w: value %q lacks a key, label, unit or source", ErrContract, v.Key)
	}
	if v.Class == ClassDerived && v.Derivation == "" {
		return fmt.Errorf("%w: %s: a derived value needs its derivation", ErrContract, v.Key)
	}
	switch v.Unit {
	case UnitCents, UnitCount, UnitDays, UnitSec:
		if v.Number == nil {
			return fmt.Errorf("%w: %s: no number", ErrContract, v.Key)
		}
	case UnitDate:
		if v.Date == "" || v.Number != nil {
			return fmt.Errorf("%w: %s: a date value needs a date only", ErrContract, v.Key)
		}
	case UnitText:
		if v.Text == "" || v.Number != nil {
			return fmt.Errorf("%w: %s: a text value needs text only", ErrContract, v.Key)
		}
	default:
		return fmt.Errorf("%w: %s: unit %q", ErrContract, v.Key, v.Unit)
	}
	low := strings.ToLower(v.Key + " " + v.Label)
	for _, w := range forbidden {
		if strings.Contains(low, w) {
			return fmt.Errorf("%w: %s: %q", ErrContract, v.Key, w)
		}
	}
	return nil
}

// check enforces the agent audience's rules.
func (a *Agent) check() error {
	switch {
	case a.Schema != ContractSchema || a.Audience != AudienceAgent:
		return fmt.Errorf("%w: schema %q audience %q", ErrContract, a.Schema, a.Audience)
	case a.Sha256 != a.digest():
		return fmt.Errorf("%w: digest", ErrContract)
	case a.DecisionPoint.Engine == "" || a.DecisionPoint.Scenario == "" || a.DecisionPoint.Time == "" || a.DecisionPoint.StateDigest == "":
		return fmt.Errorf("%w: decision point incomplete", ErrContract)
	case a.Selection.Rule != SelectionRule || a.Selection.RuleSha256 != SelectionRuleSha256:
		return fmt.Errorf("%w: selection rule", ErrContract)
	case len(a.Alternatives) < 2 || len(a.Alternatives) > MaxCandidates:
		return fmt.Errorf("%w: %d alternatives", ErrContract, len(a.Alternatives))
	case a.Alternatives[0].ID != IDProposal || a.Alternatives[1].ID != IDNoAction || len(a.Alternatives[1].Actions) != 0:
		return fmt.Errorf("%w: the proposal and no action must come first", ErrContract)
	}
	ids := map[string]bool{}
	for _, alt := range a.Alternatives {
		if alt.ID == "" || ids[alt.ID] || alt.Actions == nil || alt.Validation.Results == nil || alt.Facts == nil || alt.Estimates == nil ||
			alt.Obligations == nil || alt.Unavailable == nil || alt.CandidateSha256 == "" || alt.Description == "" {
			return fmt.Errorf("%w: alternative %q incomplete or repeated", ErrContract, alt.ID)
		}
		ids[alt.ID] = true
		ok := len(alt.Validation.Results) == len(alt.Actions)
		for _, r := range alt.Validation.Results {
			ok = ok && (r.OK || !alt.Validation.OK)
		}
		allOK := true
		for _, r := range alt.Validation.Results {
			allOK = allOK && r.OK
		}
		if !ok || allOK != alt.Validation.OK {
			return fmt.Errorf("%w: %s: validation contradicts its results", ErrContract, alt.ID)
		}
		if !alt.Validation.OK && (len(alt.Facts) > 0 || len(alt.Estimates) > 0 || len(alt.Obligations) > 0) {
			return fmt.Errorf("%w: %s: a refused alternative shows values", ErrContract, alt.ID)
		}
		for _, x := range alt.Actions {
			if !slices.Contains([]string{api.ActExtendShift, api.ActSetWalkInPolicy, api.ActSetWorklistPolicy, api.ActHire, api.ActTerminate,
				api.ActSetRoster, api.ActSetReadingAllocation, api.ActContractTeleradiology}, x.Type) {
				return fmt.Errorf("%w: %s: action type %q", ErrContract, alt.ID, x.Type)
			}
		}
		for _, list := range [][]Value{alt.Facts, alt.Estimates, alt.Obligations, {alt.Reversibility}} {
			for _, v := range list {
				if err := checkValue(v, ClassReported, ClassDerived); err != nil {
					return fmt.Errorf("%s: %w", alt.ID, err)
				}
			}
		}
		for _, u := range alt.Unavailable {
			if u.Class != ClassUnavailable || u.Key == "" || u.Reason == "" {
				return fmt.Errorf("%w: %s: unavailable item %q", ErrContract, alt.ID, u.Key)
			}
		}
	}
	if !jsonEqual(compare(a.Alternatives), a.Compared) {
		return fmt.Errorf("%w: the compared values do not follow from the alternatives", ErrContract)
	}
	return nil
}

func jsonEqual(x, y any) bool {
	a, _ := json.Marshal(x)
	b, _ := json.Marshal(y)
	return bytes.Equal(a, b)
}

// DecodeAgent reads an agent-audience comparison and checks every rule:
// unknown fields (a projection included) are refused.
func DecodeAgent(raw []byte) (*Agent, error) {
	var a Agent
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&a); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrContract, err)
	}
	if dec.More() {
		return nil, fmt.Errorf("%w: trailing data", ErrContract)
	}
	if err := a.check(); err != nil {
		return nil, err
	}
	return &a, nil
}

// VerifyAgent rebuilds the comparison at s for sel and requires it equal,
// byte for byte.
func VerifyAgent(s *session.Session, sel *Selection, a *Agent) error {
	if err := VerifySelection(s, sel); err != nil {
		return err
	}
	again, err := BuildAgent(s, sel)
	if err != nil {
		return err
	}
	x, _ := again.Encode()
	y, _ := a.Encode()
	if !bytes.Equal(x, y) {
		return fmt.Errorf("%w: the comparison does not rebuild", ErrContract)
	}
	return nil
}
