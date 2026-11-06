package alternatives

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/session"
)

// The typed agent-facing view, dreamteam.alternatives-view/v1 (V0.13d,
// owner decision D7). The comparison's free text never reaches a model:
// a view is made from a comparison already bound to the decision point
// (DecodeAgentAt), and holds only
//
//   - neutral references (E1, H1, C1, P1) in place of employee IDs and
//     names, hire request IDs, contract IDs and plan IDs, each defined by
//     the action of the proposal that names it;
//   - the alternatives' fixed IDs and sources;
//   - closed enumerations (action types, ops, roles, windows, scopes,
//     modalities, employment stages, validation codes);
//   - numbers, and dates checked to be dates;
//   - item kinds from a closed table (viewKinds), whose labels and bases
//     are fixed text in this file.
//
// Nothing is sanitized: no source string is inspected to decide whether
// it is safe to show. A comparison string is used only when it has an
// exact form this file defines (a fixed text with typed date or number
// slots), and then only the typed values are kept; the text shown is
// made again from the form. A key, label, form or field the table does
// not define makes the view fail (ErrView): fail closed.

const ViewSchema = "dreamteam.alternatives-view/v1"

// ErrView: the comparison has something the typed view does not define.
var ErrView = errors.New("alternatives: not representable in " + ViewSchema)

// Reference kinds.
const (
	RefEmployee = "employee"
	RefHire     = "hire"
	RefContract = "contract"
	// RefNewContract is a teleradiology contract a start action would make.
	RefNewContract = "new_contract"
	RefPlan        = "plan"
)

// Ref is one neutral reference: its ID and the proposal action (1-based,
// in the reply's order) that first names it.
type Ref struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Action int    `json:"action"`
}

// ViewField is one shown field of an action, made from a typed value.
type ViewField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// ViewAction is one action of an alternative: the proposal action it is,
// or adjusts (1-based, in the reply's order), its type, its neutral
// target and its typed fields; Changed names the fields that differ from
// that proposal action.
type ViewAction struct {
	Action  int         `json:"action"`
	Type    string      `json:"type"`
	Target  string      `json:"target,omitempty"`
	Fields  []ViewField `json:"fields"`
	Changed []string    `json:"changed"`
}

// ViewItem is one value: a kind of viewKinds, its reference (or a
// modality), and typed values. Key is the same for the same item in
// every alternative (kind, or kind.ref).
type ViewItem struct {
	Key    string   `json:"key"`
	Kind   string   `json:"kind"`
	Ref    string   `json:"ref,omitempty"`
	Number *int64   `json:"number,omitempty"`
	Of     *int64   `json:"of,omitempty"`
	Date   string   `json:"date,omitempty"`
	Enum   string   `json:"enum,omitempty"`
	When   []string `json:"when,omitempty"` // the typed slots of the kind's form
	Form   int      `json:"form"`           // which of the kind's forms
	List   []string `json:"list,omitempty"` // dates
}

// ViewRefusal is a refused action: its place in the reply. Neither the
// engine's code nor its message is shown: the code is an open set of
// strings, and a refused alternative cannot be selected anyway.
type ViewRefusal struct {
	Action int `json:"action"`
}

// ViewAlt is one alternative.
type ViewAlt struct {
	ID            string        `json:"id"`
	Source        string        `json:"source"`
	Actions       []ViewAction  `json:"actions"`
	Accepted      bool          `json:"accepted"`
	Refused       []ViewRefusal `json:"refused"`
	Facts         []ViewItem    `json:"facts"`
	Estimates     []ViewItem    `json:"estimates"`
	Obligations   []ViewItem    `json:"obligations"`
	Reversibility string        `json:"reversibility"`
	Unavailable   []ViewItem    `json:"unavailable"`
}

// View is what a model may be shown.
type View struct {
	Schema       string    `json:"schema"`
	Time         string    `json:"time"`
	Refs         []Ref     `json:"refs"`
	Operations   int       `json:"operations"` // the reply's operations actions, carried unchanged
	Alternatives []ViewAlt `json:"alternatives"`
}

// ViewMap is the record-only side of a view, never shown: what each
// neutral reference stands for, and each shown item's comparison key per
// alternative.
type ViewMap struct {
	Refs  map[string]string            `json:"refs"`
	Items map[string]map[string]string `json:"items"` // alternative ID -> view key -> comparison key
}

// --- closed vocabularies ---------------------------------------------------

var (
	viewSources    = []string{"proposal", "no_action", "leave_one_out", "adjustment"}
	viewModalities = []string{"XR", "US", "CT", "MR"}
	viewEmployment = []string{api.EmploymentRecruiting, api.EmploymentOnboarding, api.EmploymentActive, api.EmploymentNotice, api.EmploymentTerminated}
	viewRoles      = []string{"front_desk", "technologist", "radiologist", "ops_manager"}
	viewWindows    = []string{"weeknights", "saturday", "sunday"}
	viewScopes     = []string{"urgent", "all"}
	viewRosterOps  = []string{api.RosterOpSet, api.RosterOpRemove}
	viewTeleOps    = []string{api.TeleOpStart, api.TeleOpCancel}
	viewSkills     = []string{"registration", "XR", "CT", "US", "MR"}
	altIDForm      = regexp.MustCompile(`^(proposal|no_action|adjusted|without_[1-9][0-9]?)$`)
	dateForm       = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`)
	timeForm       = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}$`)
)

func isDate(s string) bool {
	if !dateForm.MatchString(s) {
		return false
	}
	_, err := time.Parse("2006-01-02", s)
	return err == nil
}

// A form is fixed text with typed slots: {D} a date, {N} a whole number,
// {S} a signed whole number. It parses a string exactly, giving the
// slots' values, and renders the same text from them.
type form struct {
	text string
	re   *regexp.Regexp
}

func mkForm(text string) form {
	q := regexp.QuoteMeta(text)
	q = strings.NewReplacer(`\{D\}`, `([0-9]{4}-[0-9]{2}-[0-9]{2})`, `\{N\}`, `(0|[1-9][0-9]{0,11})`, `\{S\}`, `([+-](?:0|[1-9][0-9]{0,11}))`).Replace(q)
	return form{text: text, re: regexp.MustCompile("^" + q + "$")}
}

func (f form) parse(s string) ([]string, bool) {
	m := f.re.FindStringSubmatch(s)
	if m == nil {
		return nil, false
	}
	for i, v := range m[1:] {
		if f.slot(i) == 'D' && !isDate(v) {
			return nil, false
		}
	}
	return m[1:], true
}

func (f form) slot(i int) byte {
	k := 0
	for j := 0; j+2 < len(f.text); j++ {
		if f.text[j] == '{' && f.text[j+2] == '}' {
			if k == i {
				return f.text[j+1]
			}
			k++
		}
	}
	return 0
}

func (f form) render(vals []string) string {
	out, k := f.text, 0
	var b strings.Builder
	for j := 0; j < len(out); j++ {
		if j+2 < len(out) && out[j] == '{' && out[j+2] == '}' && strings.ContainsRune("DNS", rune(out[j+1])) {
			b.WriteString(vals[k])
			k++
			j += 2
			continue
		}
		b.WriteByte(out[j])
	}
	return b.String()
}

// viewKind is one row of the closed item table.
type viewKind struct {
	label string // fixed; %s is the reference (or modality)
	ref   string // "", a reference kind, or "modality"
	unit  string // the comparison value's unit
	when  []form // the comparison's When must have one of these forms (none: When empty)
	empty bool   // When may also be empty
	basis string // fixed text shown after the value
	// basisNoWhen replaces basis for a value with no when (the no-action
	// form of a kind); "-" shows none.
	basisNoWhen string
	list        bool // the comparison's Text is a ", "-joined list of dates
	enum        []string
	// money lines: the engine label's prefix (before " · "), and where
	// the line is (guaranteed: facts; forecast: estimates).
	money    string
	forecast bool
	drop     string // the line repeats this kind's item for the same reference; it must equal it and is not shown again
	show     string // how the value is shown, when not by its unit
}

const (
	showCountOf = "count_of" // K of N, read from the line's basis form
	showMinutes = "minutes"  // signed minutes, read from the line's basis form
)

var (
	fPaid        = mkForm("paid {D}")
	fRecognized  = mkForm("recognized now, paid {D}")
	fTodayTo     = mkForm("today to {D}")
	fFrom        = mkForm("from {D}")
	fFromUntil   = mkForm("from {D} until {D}")
	fWeekFrom    = mkForm("per full week from {D}")
	fPerWeek     = mkForm("per week from {D}")
	fPerWeekTo   = mkForm("per week from {D} until {D}")
	fLastDay     = mkForm("last day {D}")
	fAllocUntil  = mkForm("from {D} until changed (14 days' notice)")
	fAccrued     = mkForm("accrued daily {D} to {D}, paid on the 1st")
	fAccruedTo   = mkForm("accrued daily to {D}, paid on the 1st")
	fMonthAfter  = mkForm("per month after {D}")
	fPerDays     = mkForm("per {N} days")
	fAtCommit    = mkForm("at commit")
	fNoDay       = mkForm("no day in the booking horizon is affected")
	fWindow      = mkForm("{D} to {D} ({N} days in the booking horizon)")
	fLater       = mkForm("owed only on a later termination")
	fUntilLater  = mkForm("per full week, until a later decision")
	fUnchanged   = mkForm("unchanged by the roster")
	displacedFm  = mkForm("{N} appointments lose coverage; {N} have a feasible replacement by capacity. Whether each patient accepts is decided only at commit; lost exams lose their revenue")
	siteBlocksFm = mkForm("estimate: {S} min of site reading blocks a week (mean of rota weeks A and B) inside the same shifts; reports gained depend on the worklist and read times; supervision unchanged")
)

// viewKinds is the closed item table. Its texts are the presentation's
// and are pinned with it (ViewTableSha256).
var viewKinds = map[string]viewKind{
	// Termination preview.
	"last_day":                 {label: "Last day · %s", ref: RefEmployee, unit: UnitDate},
	"end_date":                 {label: "Not employed from · %s", ref: RefEmployee, unit: UnitDate},
	"notice_days":              {label: "Notice · %s", ref: RefEmployee, unit: UnitDays},
	"severance_recognized_now": {label: "Severance recognized now · %s", ref: RefEmployee, unit: UnitCents, when: []form{fRecognized}, empty: true, basis: "two contracted weeks with benefits, owed once the termination is committed", basisNoWhen: "no termination is committed, so none is recognized"},
	"severance_payday":         {label: "Severance payday · %s", ref: RefEmployee, unit: UnitDate},
	"voided_rosters":           {label: "Pending roster versions voided · %s", ref: RefEmployee, unit: UnitCount, list: true},
	"voided_extensions":        {label: "Approved extensions voided · %s", ref: RefEmployee, unit: UnitCount},
	// Hire preview.
	"start_date":      {label: "Start · %s", ref: RefHire, unit: UnitDate},
	"productive_date": {label: "Independent work from · %s", ref: RefHire, unit: UnitDate},
	"rate":            {label: "Hourly rate · %s", ref: RefHire, unit: UnitCents},
	// Roster preview.
	"roster_effective": {label: "Roster effective · %s", ref: RefEmployee, unit: UnitDate},
	// The whole batch.
	"displaced_bookings": {label: "Booked appointments displaced (whole batch)", unit: UnitCount, basis: "appointments already booked that would lose their staffing"},
	"funding.commitment": {label: "Commitment tested against funding", unit: UnitCents},
	"funding.headroom":   {label: "Funding headroom", unit: UnitCents},
	"funding.funded":     {label: "Funding test passed", unit: UnitText, enum: []string{"true", "false"}},
	// Slots closed (dreamteam.slots-closed/v1).
	"slots_closed":            {label: "Scheduled slots no longer staffed, booked or not", unit: UnitCount, when: []form{fNoDay, fWindow}, empty: true, basis: "appointment slots in the current booking horizon that would have no staff; this counts capacity, not patients, orders or revenue", basisNoWhen: "no batch is applied, so no scheduled slot loses its staffing"},
	"slots_closed_booked_now": {label: "Of these, booked now", unit: UnitCount},
	"slots_closed.":           {label: "Scheduled slots no longer staffed · %s", ref: "modality", unit: UnitCount},
	// No action: what continues.
	"employment":            {label: "Employment stage · %s", ref: RefEmployee, unit: UnitText, enum: viewEmployment},
	"contract_minutes":      {label: "Contracted weekly minutes · %s", ref: RefEmployee, unit: UnitCount},
	"roster_minutes_week_a": {label: "Rostered minutes, week A · %s", ref: RefEmployee, unit: UnitCount},
	"roster_minutes_week_b": {label: "Rostered minutes, week B · %s", ref: RefEmployee, unit: UnitCount},
	"pending_rosters":       {label: "Pending roster versions · %s", ref: RefEmployee, unit: UnitCount},
	"approved_extensions":   {label: "Approved extensions · %s", ref: RefEmployee, unit: UnitCount},
	// The practice, the same under every accepted alternative.
	"booked_ahead": {label: "Appointments booked from tomorrow within the booking horizon", unit: UnitCount},
	"next_payroll": {label: "Payroll accrued, next payday", unit: UnitCents, when: []form{fPaid}},
	"payables":     {label: "Accounts payable", unit: UnitCents},
	// Obligations no action keeps.
	"severance_if_terminated_later": {label: "Severance if terminated later · %s", ref: RefEmployee, unit: UnitCents, when: []form{fLater}, basis: "two contracted weeks with benefits"},
	"weekly_guarantee":              {label: "Contract guarantee continues · %s", ref: RefEmployee, unit: UnitCents, when: []form{fUntilLater}, basis: "the weekly contract guarantee with benefits, owed whether or not the time is worked"},
	// Money lines of the staffing preview (guaranteed: facts).
	"severance_line":           {money: "Severance", ref: RefEmployee, unit: UnitCents, when: []form{fRecognized}, drop: "severance_recognized_now"},
	"contract_pay_to_last_day": {money: "Contract pay to the last day", label: "Contract pay to the last day · %s", ref: RefEmployee, unit: UnitCents, when: []form{fTodayTo}, basis: "minimum: the contract guarantee with benefits on the days still employed"},
	"recruiting_fee":           {money: "Recruiting fee", label: "Recruiting fee · %s", ref: RefHire, unit: UnitCents, when: []form{fRecognized}, basis: "fixed offer table; not refundable"},
	"weekly_guarantee_hire":    {money: "Contract guarantee per full week", label: "Contract guarantee per full week · %s", ref: RefHire, unit: UnitCents, when: []form{fFrom}, basis: "with benefits; worked pay replaces it"},
	"weekly_guarantee_roster":  {money: "Contract guarantee per full week", label: "Contract guarantee per full week · %s", ref: RefEmployee, unit: UnitCents, when: []form{fUnchanged}, basis: "with benefits, paid whatever is rostered"},
	"earliest_exit":            {money: "Earliest lawful exit", label: "Earliest lawful exit · %s", ref: RefHire, unit: UnitCents, when: []form{fLastDay}, basis: "the contract guarantee from the start to the earliest last day, plus two contracted weeks' severance with benefits"},
	"salary_allocation_change": {money: "Salary allocation change", label: "Salary allocation change · %s", ref: RefEmployee, unit: UnitCents, when: []form{fFromUntil, fAllocUntil}, basis: "salary share with benefits, accrued daily"},
	"retainer_to_earliest_end": {money: "Teleradiology retainer to the earliest end", label: "Teleradiology retainer to the earliest end · %s", ref: RefNewContract, unit: UnitCents, when: []form{fAccrued}, basis: "contract: minimum term and notice, even with no reads"},
	"retainer_to_end":          {money: "Teleradiology retainer to the end", label: "Teleradiology retainer to the end · %s", ref: RefContract, unit: UnitCents, when: []form{fAccruedTo}, basis: "contract: the fixed end"},
	// Money lines of the staffing preview (forecast: estimates).
	"contract_cost_no_longer_owed": {money: "Contract cost no longer owed", forecast: true, label: "Contract cost no longer owed · %s", ref: RefEmployee, unit: UnitCents, when: []form{fWeekFrom}, basis: "estimate: the contract guarantee with benefits; overtime and hold-over not modeled"},
	"displaced_replacement":        {money: "Displaced bookings", forecast: true, label: "Displaced bookings with a feasible replacement by capacity", unit: UnitCents, show: showCountOf, when: []form{fAtCommit}, basis: "whether each patient accepts a replacement is decided only at commit"},
	"time_not_rostered":            {money: "Contracted time not rostered", forecast: true, label: "Contracted time not rostered · %s", ref: RefEmployee, unit: UnitCents, when: []form{fPerWeek, fPerWeekTo}, basis: "estimate: part of the guarantee above, not extra"},
	"extensions_kept":              {money: "Approved extensions kept", forecast: true, label: "Approved extensions kept · %s", ref: RefEmployee, unit: UnitCents, when: []form{fFrom, fFromUntil}, basis: "estimate at the base rate"},
	"site_reading_capacity":        {money: "Site reading capacity", forecast: true, label: "Site reading blocks a week, change · %s", ref: RefEmployee, unit: UnitCents, show: showMinutes, when: []form{fFrom}, basis: "estimate, in minutes; reports gained depend on the worklist and read times"},
	"remote_read_fees_existing":    {money: "Remote read fees at trailing demand", forecast: true, label: "Remote read fees at trailing demand · %s", ref: RefContract, unit: UnitCents, when: []form{fPerDays}, basis: "an upper bound from studies that waited unread in these windows and scope recently"},
	"retainer_run_rate":            {money: "Retainer run rate while open-ended", forecast: true, label: "Retainer run rate while open-ended · %s", ref: RefNewContract, unit: UnitCents, when: []form{fMonthAfter}, basis: "until a cancellation fixes the end"},
	"remote_read_fees":             {money: "Remote read fees at trailing demand", forecast: true, label: "Remote read fees at trailing demand · %s", ref: RefNewContract, unit: UnitCents, when: []form{fPerDays}, basis: "an upper bound from studies that waited unread in these windows and scope recently"},
}

// viewKindNames are viewKinds' names in order (map order is not used).
var viewKindNames = func() []string {
	var out []string
	for k := range viewKinds {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}()

// viewUnavailable is the closed table of items with no value, with their
// fixed reasons.
var viewUnavailable = map[string]struct{ label, reason string }{
	"preview":                   {"Engine preview", "operations actions have no engine preview; they apply one by one"},
	"replacement_acceptance":    {"Replacement acceptance", "whether each displaced patient accepts a replacement is decided at commit and is not observable before"},
	"slots_closed":              {"Scheduled slots no longer staffed", "the engine does not report it for this batch"},
	"slots_beyond_horizon":      {"Slots beyond the booking horizon", "they do not exist yet and are not counted"},
	"demand_for_closed_slots":   {"Demand for the closed slots", "whether a closed slot would have been booked depends on future orders, which are not known before deciding"},
	"future_orders_and_revenue": {"Future orders and revenue", "they depend on future demand and are not projected for the decision"},
	"unmodeled":                 {"Not modeled", "staff morale, workload, resignation, clinical outcomes and reputation are not modeled by the engine"},
	"employment.":               {"Employment · %s", "the employee is not in the current observation"},
	"weekly_guarantee.":         {"Contract guarantee · %s", "the engine reports it only in the proposal's preview, which did not give it"},
}

// viewReversibility maps the engine rule's (text, basis) to a kind, and
// gives each kind its fixed text.
var viewReversibility = map[string]string{
	"none":                     "nothing to reverse · no batch is applied",
	"irreversible_termination": "irreversible · a committed termination's notice cannot be moved or withdrawn, and no action re-hires the employee",
	"irreversible_hire":        "irreversible · the recruiting fee is not refundable; ending the contract needs notice and severance",
	"irreversible_contract":    "irreversible · a contract start binds its minimum term",
	"withdrawable":             "withdrawable before its effective date · a pending version can be removed, or a contract cancelled under its terms",
	"applies_now":              "applies now · operations changes apply at once; a later change can set them again",
}

// reversibilityKind is the engine rule's kind for a batch, decided from
// the batch's action types alone (the rule of reversibility in
// comparison.go), and the comparison's value must be exactly what that
// rule gives for the batch: no prefix or partial match.
func reversibilityKind(v Value, actions []api.Action) (string, bool) {
	if !jsonEqual(v, reversibility(actions)) {
		return "", false
	}
	has := func(f func(api.Action) bool) bool { return slices.ContainsFunc(actions, f) }
	switch {
	case len(actions) == 0:
		return "none", true
	case has(func(x api.Action) bool { return x.Type == api.ActTerminate }):
		return "irreversible_termination", true
	case has(func(x api.Action) bool { return x.Type == api.ActHire }):
		return "irreversible_hire", true
	case has(irreversible):
		return "irreversible_contract", true
	case has(func(x api.Action) bool { return isStaffing(x.Type) }):
		return "withdrawable", true
	}
	return "applies_now", true
}

// --- building ---------------------------------------------------------------

type viewer struct {
	o        api.Observation
	reply    []int // the reply position (1-based) of each staffing action of the proposal
	proposal []api.Action
	refs     []Ref
	byKey    map[string]string // kind|source value -> ref ID
	m        *ViewMap
}

func vf(err string, a ...any) error { return fmt.Errorf("%w: "+err, append([]any{ErrView}, a...)...) }

// BuildView makes the typed view of a comparison that DecodeAgentAt has
// bound to s. replyPos gives, for each action of the comparison's
// proposal, its 1-based position in the model's reply (the comparison is
// of the reply's staffing batch); ops is the number of the reply's
// operations actions. s is only read.
func BuildView(s *session.Session, a *Agent, replyPos []int, ops int) (*View, *ViewMap, error) {
	if err := a.check(); err != nil {
		return nil, nil, err
	}
	if !timeForm.MatchString(a.DecisionPoint.Time) {
		return nil, nil, vf("time %q", a.DecisionPoint.Time)
	}
	prop := a.Alternatives[0].Actions
	if len(replyPos) != len(prop) {
		return nil, nil, vf("%d reply positions for %d proposal actions", len(replyPos), len(prop))
	}
	w := &viewer{o: s.Observe(0), reply: replyPos, proposal: prop, byKey: map[string]string{},
		m: &ViewMap{Refs: map[string]string{}, Items: map[string]map[string]string{}}}
	// References, in the proposal's order.
	for i, x := range prop {
		k := replyPos[i]
		switch x.Type {
		case api.ActTerminate, api.ActSetRoster, api.ActSetReadingAllocation:
			w.ref(RefEmployee, x.Employee, k)
		case api.ActHire:
			w.ref(RefHire, x.RequestID, k)
		case api.ActContractTeleradiology:
			if x.Op == api.TeleOpStart {
				w.ref(RefNewContract, "new:"+fmt.Sprint(i), k)
				w.ref(RefPlan, x.Plan, k)
			} else {
				w.ref(RefContract, x.Contract, k)
			}
		default:
			return nil, nil, vf("action type %q", x.Type)
		}
	}
	v := &View{Schema: ViewSchema, Time: a.DecisionPoint.Time, Refs: w.refs, Operations: ops}
	for _, alt := range a.Alternatives {
		va, err := w.alternative(alt)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %w", alt.ID, err)
		}
		v.Alternatives = append(v.Alternatives, va)
	}
	return v, w.m, nil
}

func (w *viewer) ref(kind, value string, action int) string {
	k := kind + "|" + value
	if id, ok := w.byKey[k]; ok {
		return id
	}
	letter := map[string]string{RefEmployee: "E", RefHire: "H", RefContract: "C", RefNewContract: "C", RefPlan: "P"}[kind]
	n := 1
	for _, r := range w.refs {
		if strings.HasPrefix(r.ID, letter) {
			n++
		}
	}
	id := letter + strconv.Itoa(n)
	w.refs = append(w.refs, Ref{ID: id, Kind: kind, Action: action})
	w.byKey[k] = id
	w.m.Refs[id] = value
	return id
}

func (w *viewer) known(kind, value string) (string, bool) {
	id, ok := w.byKey[kind+"|"+value]
	return id, ok
}

// match is the proposal action an alternative's action is, or adjusts:
// the same type and target, the first not yet used.
func (w *viewer) match(x api.Action, used map[int]bool) (int, bool) {
	for i, p := range w.proposal {
		if used[i] || p.Type != x.Type || p.Employee != x.Employee || p.RequestID != x.RequestID || p.Contract != x.Contract ||
			p.Plan != x.Plan || p.Op != x.Op {
			continue
		}
		return i, true
	}
	return 0, false
}

func enumOK(v string, set []string) bool { return slices.Contains(set, v) }

// action is one action, shown by its typed fields only; every field not
// shown must equal the proposal action's.
func (w *viewer) action(x api.Action, used map[int]bool) (ViewAction, error) {
	i, ok := w.match(x, used)
	if !ok {
		return ViewAction{}, vf("an action that is not one of the proposal's")
	}
	used[i] = true
	p := w.proposal[i]
	va := ViewAction{Action: w.reply[i], Type: x.Type, Fields: []ViewField{}, Changed: []string{}}
	date := func(name, got, was string) error {
		if !isDate(got) {
			return vf("%s is not a date", name)
		}
		va.Fields = append(va.Fields, ViewField{name, got})
		if got != was {
			va.Changed = append(va.Changed, name)
		}
		return nil
	}
	enum := func(name, got string, set []string) error {
		if !enumOK(got, set) {
			return vf("%s is not one of %v", name, set)
		}
		va.Fields = append(va.Fields, ViewField{name, got})
		return nil
	}
	// Fields that may differ from the proposal action, by type (the
	// adjustments of the selection rule); everything else must not.
	q := x
	var err error
	switch x.Type {
	case api.ActTerminate:
		va.Target, _ = w.known(RefEmployee, x.Employee)
		err = date("lastDay", x.LastDay, p.LastDay)
		q.LastDay = p.LastDay
	case api.ActHire:
		va.Target, _ = w.known(RefHire, x.RequestID)
		if err = enum("role", x.Role, viewRoles); err == nil {
			err = date("startDate", x.StartDate, p.StartDate)
		}
		q.StartDate = p.StartDate
	case api.ActSetRoster:
		va.Target, _ = w.known(RefEmployee, x.Employee)
		if err = enum("op", x.Op, viewRosterOps); err == nil {
			err = date("effective", x.Effective, p.Effective)
		}
		q.Effective = p.Effective
	case api.ActSetReadingAllocation:
		va.Target, _ = w.known(RefEmployee, x.Employee)
		if err = enum("op", x.Op, viewRosterOps); err == nil {
			err = date("effective", x.Effective, p.Effective)
		}
		if err == nil && x.SiteAllocationBP != 0 {
			va.Fields = append(va.Fields, ViewField{"siteAllocationBP", strconv.FormatInt(x.SiteAllocationBP, 10)})
		}
		q.Effective = p.Effective
	case api.ActContractTeleradiology:
		if err = enum("op", x.Op, viewTeleOps); err != nil {
			break
		}
		if x.Op == api.TeleOpStart {
			va.Target, _ = w.known(RefNewContract, "new:"+fmt.Sprint(i))
			plan, _ := w.known(RefPlan, x.Plan)
			va.Fields = append(va.Fields, ViewField{"plan", plan})
			for _, wn := range x.Windows {
				if !enumOK(wn, viewWindows) {
					return ViewAction{}, vf("window %q", wn)
				}
			}
			va.Fields = append(va.Fields, ViewField{"windows", strings.Join(x.Windows, "+")})
			if err = enum("scope", x.Scope, viewScopes); err == nil && x.Scope != p.Scope {
				va.Changed = append(va.Changed, "scope")
			}
			if err == nil {
				err = date("effective", x.Effective, p.Effective)
			}
			q.Scope, q.Effective = p.Scope, p.Effective
		} else {
			va.Target, _ = w.known(RefContract, x.Contract)
			err = date("endDate", x.EndDate, p.EndDate)
			q.EndDate = p.EndDate
		}
	default:
		return ViewAction{}, vf("action type %q", x.Type)
	}
	if err != nil {
		return ViewAction{}, err
	}
	if va.Target == "" || !jsonEqual(q, p) {
		return ViewAction{}, vf("action %d differs from the proposal's in a field the view does not show", va.Action)
	}
	return va, nil
}

func (w *viewer) alternative(alt AgentAlternative) (ViewAlt, error) {
	if !altIDForm.MatchString(alt.ID) || !enumOK(alt.Source, viewSources) {
		return ViewAlt{}, vf("alternative %q from %q", alt.ID, alt.Source)
	}
	va := ViewAlt{ID: alt.ID, Source: alt.Source, Actions: []ViewAction{}, Accepted: alt.Validation.OK, Refused: []ViewRefusal{},
		Facts: []ViewItem{}, Estimates: []ViewItem{}, Obligations: []ViewItem{}, Unavailable: []ViewItem{}}
	used := map[int]bool{}
	for _, x := range alt.Actions {
		a, err := w.action(x, used)
		if err != nil {
			return ViewAlt{}, err
		}
		va.Actions = append(va.Actions, a)
	}
	for i, r := range alt.Validation.Results {
		if r.OK {
			continue
		}
		va.Refused = append(va.Refused, ViewRefusal{Action: va.Actions[i].Action})
	}
	kind, ok := reversibilityKind(alt.Reversibility, alt.Actions)
	if !ok || alt.Reversibility.Key != "reversibility" {
		return ViewAlt{}, vf("reversibility")
	}
	va.Reversibility = kind
	items := map[string]string{}
	w.m.Items[alt.ID] = items
	// Money lines need the alternative's hires' offer labels and its
	// targets' engine labels; both are read from the alternative's own
	// values and the observation, never shown.
	offers := map[string]string{} // offer label -> hire ref
	for _, f := range alt.Facts {
		if req, ok := strings.CutPrefix(f.Key, "start_date."); ok {
			h, known := w.known(RefHire, req)
			offer, okl := strings.CutPrefix(f.Label, "Start · ")
			if !known || !okl {
				return ViewAlt{}, vf("hire %q", f.Key)
			}
			if _, dup := offers[offer]; dup {
				offers[offer] = "" // ambiguous: a line naming it fails
			} else {
				offers[offer] = h
			}
		}
	}
	add := func(list *[]ViewItem, v Value, forecast bool) error {
		it, skip, err := w.item(alt, v, forecast, offers)
		if err != nil {
			return fmt.Errorf("%s: %w", v.Key, err)
		}
		if skip {
			return nil
		}
		if _, dup := items[it.Key]; dup {
			return vf("item %s twice", it.Key)
		}
		items[it.Key] = v.Key
		*list = append(*list, it)
		return nil
	}
	for _, v := range alt.Facts {
		if err := add(&va.Facts, v, false); err != nil {
			return ViewAlt{}, err
		}
	}
	for _, v := range alt.Estimates {
		if err := add(&va.Estimates, v, true); err != nil {
			return ViewAlt{}, err
		}
	}
	for _, v := range alt.Obligations {
		if err := add(&va.Obligations, v, false); err != nil {
			return ViewAlt{}, err
		}
	}
	// Repeated lines must equal what they repeat.
	for _, v := range alt.Facts {
		if strings.HasPrefix(v.Key, "guaranteed.") {
			if err := w.checkDrop(alt, v, va.Facts); err != nil {
				return ViewAlt{}, err
			}
		}
	}
	for _, u := range alt.Unavailable {
		it, err := w.unavailable(u)
		if err != nil {
			return ViewAlt{}, err
		}
		va.Unavailable = append(va.Unavailable, it)
	}
	return va, nil
}

// split is a comparison key's kind and its entity ("" for none).
func splitKey(key string) (kind, entity string) {
	if _, ok := viewKinds[key]; ok {
		return key, ""
	}
	if e, ok := strings.CutPrefix(key, "slots_closed."); ok {
		return "slots_closed.", e
	}
	k, e, _ := strings.Cut(key, ".")
	return k, e
}

func (w *viewer) entityRef(kind, entity string) (string, error) {
	switch kind {
	case "":
		if entity != "" {
			return "", vf("unexpected entity")
		}
		return "", nil
	case "modality":
		if !enumOK(entity, viewModalities) {
			return "", vf("modality %q", entity)
		}
		return entity, nil
	}
	id, ok := w.known(kind, entity)
	if !ok {
		return "", vf("an entity the proposal does not name")
	}
	return id, nil
}

// item is one comparison value in the view (skip: a repeated money line).
func (w *viewer) item(alt AgentAlternative, v Value, forecast bool, offers map[string]string) (ViewItem, bool, error) {
	var name string
	var k viewKind
	var ref string
	var err error
	money := strings.HasPrefix(v.Key, "guaranteed.") || strings.HasPrefix(v.Key, "forecast.")
	if money {
		if strings.HasPrefix(v.Key, "forecast.") != forecast {
			return ViewItem{}, false, vf("a money line in the wrong section")
		}
		name, k, ref, err = w.moneyLine(alt, v, offers)
		if err != nil {
			return ViewItem{}, false, err
		}
	} else {
		kind, entity := splitKey(v.Key)
		var ok bool
		if k, ok = viewKinds[kind]; !ok || k.money != "" {
			return ViewItem{}, false, vf("unknown item")
		}
		name = kind
		if ref, err = w.entityRef(k.ref, entity); err != nil {
			return ViewItem{}, false, err
		}
	}
	if k.unit != v.Unit {
		return ViewItem{}, false, vf("unit %q", v.Unit)
	}
	it := ViewItem{Kind: name, Ref: ref, Key: strings.TrimSuffix(name, ".")}
	if ref != "" {
		it.Key = strings.TrimSuffix(name, ".") + "." + ref
	}
	// A field the unit does not use must be empty: nothing unexpected is
	// silently ignored. Label, class, source, basis and derivation are the
	// comparison's own metadata, checked by its contract and never shown.
	if (v.Unit != UnitDate && v.Date != "") || (v.Unit == UnitDate && (v.Number != nil || v.Text != "")) ||
		(v.Unit == UnitText && v.Number != nil) {
		return ViewItem{}, false, vf("a field the unit does not use")
	}
	switch v.Unit {
	case UnitCents, UnitCount, UnitDays:
		if v.Number == nil {
			return ViewItem{}, false, vf("no number")
		}
		it.Number = num(*v.Number)
	case UnitDate:
		if !isDate(v.Date) {
			return ViewItem{}, false, vf("date")
		}
		it.Date = v.Date
	case UnitText:
		if !enumOK(v.Text, k.enum) {
			return ViewItem{}, false, vf("text is not one of the kind's values")
		}
		it.Enum = v.Text
	default:
		return ViewItem{}, false, vf("unit")
	}
	switch k.show {
	case showCountOf:
		vals, ok := displacedFm.parse(v.Basis)
		if !ok {
			return ViewItem{}, false, vf("basis form")
		}
		n, _ := strconv.ParseInt(vals[0], 10, 64)
		kk, _ := strconv.ParseInt(vals[1], 10, 64)
		it.Number, it.Of = num(kk), num(n)
	case showMinutes:
		vals, ok := siteBlocksFm.parse(v.Basis)
		if !ok {
			return ViewItem{}, false, vf("basis form")
		}
		m, _ := strconv.ParseInt(vals[0], 10, 64)
		it.Number = num(m)
	}
	if k.list {
		if v.Text != "" {
			for _, d := range strings.Split(v.Text, ", ") {
				if !isDate(d) {
					return ViewItem{}, false, vf("list entry is not a date")
				}
				it.List = append(it.List, d)
			}
		}
		if int64(len(it.List)) != *it.Number {
			return ViewItem{}, false, vf("list length")
		}
	} else if v.Text != "" && v.Unit != UnitText {
		return ViewItem{}, false, vf("unexpected text")
	}
	if err := whenOf(k, v.When, &it); err != nil {
		return ViewItem{}, false, err
	}
	if k.drop != "" {
		return it, true, nil
	}
	return it, false, nil
}

func whenOf(k viewKind, when string, it *ViewItem) error {
	if when == "" {
		if k.empty || len(k.when) == 0 {
			it.Form = -1
			return nil
		}
		return vf("no when")
	}
	for i, f := range k.when {
		if vals, ok := f.parse(when); ok {
			it.When, it.Form = vals, i
			return nil
		}
	}
	return vf("a when with no form of the kind")
}

// moneyLine resolves a preview money line: its prefix gives the kind,
// the rest must be exactly the engine label of one of the alternative's
// own targets, and its when (and, for two kinds, its basis) must have the
// kind's form.
func (w *viewer) moneyLine(alt AgentAlternative, v Value, offers map[string]string) (string, viewKind, string, error) {
	forecast := strings.HasPrefix(v.Key, "forecast.")
	label := v.Label
	// Remote read fees carry a count in the label.
	var studies string
	if strings.HasPrefix(label, "Remote read fees at trailing demand · ") {
		i := strings.LastIndex(label, " (")
		if i < 0 {
			return "", viewKind{}, "", vf("money line")
		}
		n, ok := mkForm(" ({N} studies)").parse(label[i:])
		if !ok {
			return "", viewKind{}, "", vf("money line")
		}
		studies, label = n[0], label[:i]
	}
	prefix, who, hasWho := strings.Cut(label, " · ")
	type hit struct {
		name string
		k    viewKind
		ref  string
	}
	var hits []hit
	for _, name := range viewKindNames {
		k := viewKinds[name]
		if k.money == "" || k.money != prefix || k.forecast != forecast || hasWho != (k.ref != "") {
			continue
		}
		ref := ""
		if hasWho {
			r, err := w.who(alt, k.ref, who, offers)
			if err != nil {
				continue
			}
			ref = r
		}
		ok := false
		for _, f := range k.when {
			if _, ok = f.parse(v.When); ok {
				break
			}
		}
		if ok {
			hits = append(hits, hit{name, k, ref})
		}
	}
	if len(hits) != 1 {
		return "", viewKind{}, "", vf("a money line of %d known kinds", len(hits))
	}
	name, k, ref := hits[0].name, hits[0].k, hits[0].ref
	switch name {
	case "displaced_replacement":
		if v.Number == nil || *v.Number != 0 {
			return "", viewKind{}, "", vf("displaced bookings estimate is not zero")
		}
		if _, ok := displacedFm.parse(v.Basis); !ok {
			return "", viewKind{}, "", vf("displaced bookings basis")
		}
	case "site_reading_capacity":
		if v.Number == nil || *v.Number != 0 {
			return "", viewKind{}, "", vf("site reading estimate is not zero")
		}
		if _, ok := siteBlocksFm.parse(v.Basis); !ok {
			return "", viewKind{}, "", vf("site reading basis")
		}
	case "remote_read_fees", "remote_read_fees_existing":
		if studies == "" {
			return "", viewKind{}, "", vf("read fees")
		}
	}
	return name, k, ref, nil
}

// who is the reference a money line names: the engine's label of one of
// the alternative's targets, exactly.
func (w *viewer) who(alt AgentAlternative, kind, who string, offers map[string]string) (string, error) {
	var hits []string
	add := func(id string) {
		if !slices.Contains(hits, id) {
			hits = append(hits, id)
		}
	}
	switch kind {
	case RefEmployee:
		for _, x := range alt.Actions {
			if x.Employee == "" {
				continue
			}
			for _, e := range w.o.Staff {
				if e.ID != x.Employee {
					continue
				}
				id, _ := w.known(RefEmployee, e.ID)
				if who == e.Name+" ("+e.ID+")" {
					add(id)
				}
				if e.Hired != nil && who == e.ID+" ("+offerLabelOf(e)+")" {
					add(id)
				}
			}
		}
		// No action: the employees the proposal would terminate.
		if len(alt.Actions) == 0 {
			return "", vf("a money line under no action")
		}
	case RefHire:
		if h, ok := offers[who]; ok && h != "" {
			add(h)
		}
	case RefContract, RefNewContract:
		for i, p := range w.proposal {
			if p.Type != api.ActContractTeleradiology {
				continue
			}
			if kind == RefNewContract && p.Op == api.TeleOpStart && who == "new "+windowLabel(p.Windows)+" contract" && slices.ContainsFunc(alt.Actions, func(x api.Action) bool {
				return x.Type == p.Type && x.Op == p.Op && x.Plan == p.Plan && slices.Equal(x.Windows, p.Windows)
			}) {
				id, _ := w.known(RefNewContract, "new:"+fmt.Sprint(i))
				add(id)
			}
			if kind == RefContract && p.Op == api.TeleOpCancel && who == p.Contract {
				id, _ := w.known(RefContract, p.Contract)
				add(id)
			}
		}
	}
	if len(hits) != 1 || hits[0] == "" {
		return "", vf("a money line naming no single target")
	}
	return hits[0], nil
}

func windowLabel(ws []string) string {
	var out []string
	for _, n := range viewWindows {
		if slices.Contains(ws, n) {
			out = append(out, n)
		}
	}
	return strings.Join(out, ", ")
}

func offerLabelOf(e api.Employee) string {
	if e.Role == "front_desk" {
		return "front desk"
	}
	sk := slices.Clone(e.Skills)
	rank := func(s string) int {
		if i := slices.Index(viewSkills, s); i >= 0 {
			return i
		}
		return len(viewSkills)
	}
	slices.SortStableFunc(sk, func(a, b string) int {
		if ra, rb := rank(a), rank(b); ra != rb {
			return ra - rb
		}
		return strings.Compare(a, b)
	})
	return strings.Join(sk, "+") + " " + e.Role
}

// checkDrop: a money line the view does not show again must equal the
// item it repeats.
func (w *viewer) checkDrop(alt AgentAlternative, v Value, shown []ViewItem) error {
	name, k, ref, err := w.moneyLine(alt, v, nil)
	if err != nil || k.drop == "" {
		return nil
	}
	_ = name
	for _, it := range shown {
		if it.Kind == k.drop && it.Ref == ref {
			if it.Number == nil || v.Number == nil || *it.Number != *v.Number {
				return vf("%s does not equal %s", v.Key, it.Key)
			}
			return nil
		}
	}
	return vf("%s repeats an item that is not shown", v.Key)
}

func (w *viewer) unavailable(u Unavailable) (ViewItem, error) {
	if u.Class != ClassUnavailable {
		return ViewItem{}, vf("unavailable class")
	}
	if _, ok := viewUnavailable[u.Key]; ok {
		return ViewItem{Key: u.Key, Kind: u.Key, Form: -1}, nil
	}
	for _, p := range []string{"employment.", "weekly_guarantee."} {
		if e, ok := strings.CutPrefix(u.Key, p); ok {
			id, known := w.known(RefEmployee, e)
			if !known {
				return ViewItem{}, vf("unavailable %q names an entity the proposal does not", p)
			}
			return ViewItem{Key: p + id, Kind: p, Ref: id, Form: -1}, nil
		}
	}
	return ViewItem{}, vf("unknown unavailable item")
}

// ViewTableText is the closed tables' texts, in a fixed order: every item
// kind's label, forms and basis, every unavailable item's label and
// reason, every reversibility text and every reference description. It is
// part of the presentation and pinned with it (ViewTableSha256).
func ViewTableText() string {
	var b strings.Builder
	b.WriteString(ViewSchema + "\n\nItems:\n")
	for _, name := range viewKindNames {
		k := viewKinds[name]
		fmt.Fprintf(&b, "%s | label %q | ref %q | unit %s | money %q forecast %v | show %q | drop %q | list %v | enum %v | basis %q | basisNoWhen %q | forms",
			name, k.label, k.ref, k.unit, k.money, k.forecast, k.show, k.drop, k.list, k.enum, k.basis, k.basisNoWhen)
		for _, f := range k.when {
			fmt.Fprintf(&b, " %q", f.text)
		}
		fmt.Fprintf(&b, " | empty %v\n", k.empty)
	}
	fmt.Fprintf(&b, "\nBasis forms:\n%q\n%q\n", displacedFm.text, siteBlocksFm.text)
	b.WriteString("\nNot available:\n")
	var keys []string
	for k := range viewUnavailable {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		fmt.Fprintf(&b, "%s | %q | %q\n", k, viewUnavailable[k].label, viewUnavailable[k].reason)
	}
	b.WriteString("\nReversibility:\n")
	keys = keys[:0]
	for k := range viewReversibility {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		fmt.Fprintf(&b, "%s | %q\n", k, viewReversibility[k])
	}
	b.WriteString("\nReferences:\n")
	for _, k := range []string{RefEmployee, RefHire, RefNewContract, RefContract, RefPlan} {
		fmt.Fprintf(&b, "%s | %q\n", k, refText(Ref{Kind: k, Action: 1}))
	}
	fmt.Fprintf(&b, "\nVocabularies:\nsources %v\nmodalities %v\nemployment %v\nroles %v\nwindows %v\nscopes %v\nroster ops %v\ntele ops %v\n",
		viewSources, viewModalities, viewEmployment, viewRoles, viewWindows, viewScopes, viewRosterOps, viewTeleOps)
	return b.String()
}
