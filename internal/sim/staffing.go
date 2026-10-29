package sim

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// V0.2 staffing actions: hire and terminate (Slice 2), set_roster (Slice
// 3, roster.go), set_reading_allocation (Slice 4, allocation.go)
// (V0.2_DESIGN §5.3, §5.8, §5.10, §6.1–§6.4, §7).
//
// A batch holding a V0.2 action holds only V0.2 actions (else every item
// is refused, the first legacy one as mixed_batch) and is atomic: every
// item is validated, in input order, against the state now plus the
// earlier items of the batch, and then the composed roster timelines are
// checked as a whole (checkComposed); one invalid item refuses the whole
// batch (it gets its own code, every other item batch_rejected) and
// nothing is committed. A valid batch commits in input order — hires get
// their IDs, recruiting fees and severance post, roster and allocation
// versions enter their timelines — then trainees' productive dates move, and the
// bookings of every day the batch changes are reconciled once, against
// the final timeline (displace.go), and one dispatch runs.
//
// Identical repeats are no-ops (ok, unchanged): a hire whose requestId was
// already accepted with the same content, even after the employee left;
// a termination with the same last day. A reused requestId with other
// content is request_conflict; a different last day already_terminating.

// V0.2 action sets: the types that make a V0.2 batch. This build knows
// set_roster, set_reading_allocation and contract_teleradiology too; a
// 0.2.0-dev.2 record (Slice 2) is replayed with its own set, where
// set_roster was an unknown V0.1-path action, a 0.2.0-dev.3 or dev.4
// record (Slice 3) with Slice 3's, where set_reading_allocation was, and
// a 0.2.0-dev.5 to dev.7 record (Slice 4) with Slice 4's, where
// contract_teleradiology was (session package, V0.2_DESIGN §12).
var (
	v2Actions     = []string{api.ActHire, api.ActTerminate, api.ActSetRoster, api.ActSetReadingAllocation, api.ActContractTeleradiology}
	slice2Actions = []string{api.ActHire, api.ActTerminate}
	slice3Actions = []string{api.ActHire, api.ActTerminate, api.ActSetRoster}
	slice4Actions = []string{api.ActHire, api.ActTerminate, api.ActSetRoster, api.ActSetReadingAllocation}
)

func isV2Action(t string) bool { return slices.Contains(v2Actions, t) }

// staffOutcome is one V0.2 item's validation or commit result.
type staffOutcome struct {
	ok        bool
	code, msg string
	employee  string // hire: the ID given (commit only)
	contract  string // contract_teleradiology: the contract (start: at commit)
	effective string // the start date, or the first day after the last day
	hire      *hireSpec
	term      *termSpec
	roster    *rosterSpec
	alloc     *allocSpec
	tele      *teleSpec
	fund      *funding // V0.7b: its commitment, under a credit line
}

// hireSpec is a validated hire, not yet committed.
type hireSpec struct {
	req, canon       string
	terms            EmployeeConfig // ID and Name set at commit
	offer            *Offer
	tier             string
	lead, fee        int64
	from, productive int64
	training         simtime.Duration
}

// termSpec is a validated termination, not yet committed.
type termSpec struct {
	emp     int
	lastDay int64
}

// proposal is what a batch's earlier items have proposed.
type proposal struct {
	hires   []*hireSpec
	terms   []*termSpec
	rosters []*rosterSpec
	reqs    map[string]bool
	emps    map[int]bool
	rkeys   map[rosterKey]bool
	// versions are the composed roster timelines of the employees the
	// batch reschedules; productive their recomputed productive dates
	// (trainees only, set by checkComposed).
	versions   map[int][]rosterVersion
	productive map[int]int64
	// allocs are the batch's allocation changes; aversions the composed
	// allocation timelines of the radiologists they change; akeys the
	// (radiologist, date) pairs written.
	allocs    []*allocSpec
	aversions map[int][]allocVersion
	akeys     map[rosterKey]bool
	// teles are the batch's contract items; tcons the contract timeline
	// they compose (nil until the first); treqs the start requestIds and
	// tcancel the contracts cancelled in the batch.
	teles   []*teleSpec
	tcons   []teleContract
	treqs   map[string]bool
	tcancel map[int]bool
	// fund is the batch's funding test (V0.7b), set once every item is
	// valid under a credit line.
	fund *fundCheck
}

// staffBatch validates a V0.2 batch and, if do, commits it. known is the
// action set the batch is applied with (v2Actions, or slice2Actions for a
// Slice 2 record).
func (st *state) staffBatch(e *Engine, actions []api.Action, do bool, known []string) ([]staffOutcome, *proposal) {
	out := make([]staffOutcome, len(actions))
	fail := -1
	// The refusal names the staffing actions as earlier builds did, and
	// set_reading_allocation or contract_teleradiology only when the
	// batch holds one: a batch without them keeps its recorded message.
	names := slices.DeleteFunc(slices.Clone(known), func(t string) bool {
		return t == api.ActSetReadingAllocation || t == api.ActContractTeleradiology
	})
	for _, t := range []string{api.ActSetReadingAllocation, api.ActContractTeleradiology} {
		if slices.ContainsFunc(actions, func(a api.Action) bool { return a.Type == t }) && slices.Contains(known, t) {
			names = append(names, t)
		}
	}
	for k, a := range actions {
		if !slices.Contains(known, a.Type) {
			out[k] = staffOutcome{code: "mixed_batch", msg: fmt.Sprintf("%q cannot share a batch with V0.2 staffing actions (%s): submit it in its own batch at the same time.",
				a.Type, strings.Join(names, ", "))}
			fail = k
			break
		}
	}
	p := &proposal{reqs: map[string]bool{}, emps: map[int]bool{}, rkeys: map[rosterKey]bool{},
		versions: map[int][]rosterVersion{}, productive: map[int]int64{}, aversions: map[int][]allocVersion{}, akeys: map[rosterKey]bool{},
		treqs: map[string]bool{}, tcancel: map[int]bool{}}
	if fail < 0 {
		for k, a := range actions {
			var o staffOutcome
			switch a.Type {
			case api.ActHire:
				o = st.checkHire(a, p)
			case api.ActTerminate:
				o = st.checkTerminate(a, p)
			case api.ActSetRoster:
				o = st.checkSetRoster(a, p, k)
			case api.ActContractTeleradiology:
				o = st.checkTeleAction(a, p)
			default:
				o = st.checkSetAllocation(a, p)
			}
			out[k] = o
			if !o.ok {
				fail = k
				break
			}
		}
	}
	if fail < 0 && len(p.rosters) > 0 {
		if k, o := st.checkComposed(p); k >= 0 {
			out[k], fail = o, k
		}
	}
	if fail < 0 && len(p.allocs) > 0 && !st.dev5 {
		if k, o := st.checkAllocCommit(p, out); k >= 0 {
			out[k], fail = o, k
		}
	}
	if fail < 0 && len(p.teles) > 0 {
		if k, o := st.checkTeleTimeline(p, out); k >= 0 {
			out[k], fail = o, k
		}
	}
	if fail < 0 && st.solvency() != nil {
		if k, o := st.checkFunding(p, out); k >= 0 {
			out[k], fail = o, k
		}
	}
	if fail >= 0 {
		for k := range out {
			if k != fail {
				out[k] = staffOutcome{code: "batch_rejected", msg: fmt.Sprintf("Batch refused: action %d of %d (%s) was refused, so nothing in this batch was applied.",
					fail+1, len(actions), out[fail].code)}
			}
		}
		return out, p
	}
	if !do {
		return out, p
	}
	changed := false
	for k := range out {
		changed = changed || out[k].code == api.ActionAccepted
	}
	if !changed {
		return out, p
	}
	st.beforeChange()
	from := noEnd
	tele := false
	var causes []int
	var why []string
	add := func(c string) {
		if !slices.Contains(why, c) {
			why = append(why, c)
		}
	}
	for k := range out {
		o := &out[k]
		if o.code == api.ActionAccepted {
			st.reserveFunding(o.fund, len(st.actions.log)+k)
		}
		switch {
		case o.code != api.ActionAccepted:
		case o.hire != nil:
			i := st.commitHire(o.hire, len(st.actions.log)+k)
			o.employee = st.staff[i].ID
			o.msg = st.hireText(o.hire, st.staff[i].ID)
		case o.term != nil:
			st.commitTerminate(o.term, len(st.actions.log)+k)
			from = min(from, o.term.lastDay+1)
			causes = append(causes, len(st.actions.log)+k)
			add(causeEmploymentEnded)
		case o.roster != nil:
			st.commitRoster(o.roster, len(st.actions.log)+k)
			from = min(from, o.roster.from)
			causes = append(causes, len(st.actions.log)+k)
			add(causeRosterChanged)
		case o.alloc != nil:
			st.commitAlloc(o.alloc, len(st.actions.log)+k)
		case o.tele != nil:
			o.contract = st.commitTele(o.tele, len(st.actions.log)+k)
			o.msg = st.teleText(o.tele, true)
			tele = true
		}
	}
	if tele {
		st.scheduleTeleEdge(e)
	}
	if moved, delayed := st.trainingMoves(p); moved != noEnd {
		from = min(from, moved)
		if delayed {
			add(causeTrainingDelayed)
		}
	}
	if from != noEnd {
		st.reconcile(e, from, strings.Join(why, "+"), causes)
	}
	return out, p
}

// --- hire ----------------------------------------------------------------

func (st *state) checkHire(a api.Action, p *proposal) staffOutcome {
	bad := func(code, format string, args ...any) staffOutcome {
		return staffOutcome{code: code, msg: fmt.Sprintf(format, args...)}
	}
	switch {
	case a.Employee != "" || a.Minutes != 0 || a.Date != "" || a.Policy != "" || a.Cutoff != "" || a.LastDay != "" || a.Op != "" || a.Effective != "" ||
		a.SiteAllocationBP != 0 || teleFields(a):
		return bad("invalid_schema", "hire takes requestId, role, skills, rateCents, contractedWeeklyMinutes, roster and startDate only.")
	case a.RequestID == "" || len(a.RequestID) > 128:
		return bad("invalid_schema", "hire needs a requestId of 1–128 characters.")
	case a.Role == "" || len(a.Skills) == 0 || a.RateCents == 0 || a.ContractedWeeklyMinutes == 0 || a.Roster == nil || a.StartDate == "":
		return bad("invalid_schema", "hire needs role, skills, rateCents, contractedWeeklyMinutes, roster and startDate.")
	case p.reqs[a.RequestID]:
		return bad("invalid_parameters", "requestId %q appears twice in this batch.", a.RequestID)
	}
	canon := hireCanon(a)
	for i := range st.staff {
		if h := st.staff[i].Hire; h != nil && h.RequestID == a.RequestID {
			if h.Canon == canon {
				return staffOutcome{ok: true, code: api.ActionUnchanged, employee: st.staff[i].ID,
					msg: fmt.Sprintf("Hire %q was already accepted as %s: nothing changed.", a.RequestID, st.staff[i].ID)}
			}
			return bad("request_conflict", "requestId %q was already used for %s with different terms.", a.RequestID, st.staff[i].ID)
		}
	}
	role := Role(a.Role)
	if role != Technologist && role != FrontDesk {
		return bad("wrong_role", "Only technologists and front desk staff can be hired (not %q).", a.Role)
	}
	offer := findOffer(role, a.Skills)
	if offer == nil || hasDup(a.Skills) {
		var sets []string
		for _, o := range Offers {
			if o.Role == role {
				sets = append(sets, strings.Join(o.Skills, "+"))
			}
		}
		return bad("invalid_skills", "No %s offer for skills %v: recruitable sets are %s.", role, a.Skills, strings.Join(sets, ", "))
	}
	date, err := simtime.ParseDate(a.StartDate)
	if err != nil {
		return bad("invalid_date", "startDate %q is not YYYY-MM-DD.", a.StartDate)
	}
	from := st.cal.DayIndex(st.cal.StartOfDate(date))
	switch {
	case from <= st.day:
		return bad("past_date", "startDate %s is not after today.", a.StartDate)
	case from > st.day+st.cfg.Policies.BookingHorizonDays:
		return bad("beyond_horizon", "startDate %s is beyond the %d-day booking horizon.", a.StartDate, st.cfg.Policies.BookingHorizonDays)
	}
	var tier string
	var lead int64
	switch a.RateCents {
	case offer.StandardCents:
		tier, lead = "standard", offer.StandardLead
	case offer.PremiumCents:
		tier, lead = "premium", offer.PremiumLead
	default:
		return bad("rate_out_of_band", "A %s is hired at %s/h (standard, %d-day lead) or %s/h (premium, %d-day lead), not %s.",
			offerLabel(&EmployeeConfig{Role: role, Skills: offer.Skills}), rate(offer.StandardCents), offer.StandardLead,
			rate(offer.PremiumCents), offer.PremiumLead, rate(a.RateCents))
	}
	if a.ContractedWeeklyMinutes < MinContractMin || a.ContractedWeeklyMinutes > MaxContractMin {
		return bad("invalid_contract_hours", "contractedWeeklyMinutes must be %d–%d (40 hours).", MinContractMin, MaxContractMin)
	}
	if from < st.day+lead {
		return bad("lead_too_short", "At the %s tier recruiting takes %d days: the earliest start is %s.", tier, lead,
			st.cal.Date(st.cal.StartOfDay(st.day+lead)))
	}
	t := EmployeeConfig{Role: role, Skills: slices.Clone(offer.Skills), Pay: Hourly, RateCents: a.RateCents,
		Contract: Contract{Weekly: simtime.Duration(a.ContractedWeeklyMinutes) * simtime.Minute}}
	if err := st.hireRoster(&t, a.Roster); err != nil {
		return bad("invalid_roster", "%v", err)
	}
	need := trainingFor(role)
	done, ok := st.trainingEnd(&t, from, need)
	if !ok {
		return bad("training_incomplete", "The roster gives fewer than %s of training within %d days of the start: no independent work would ever begin.",
			hours(need), TrainingWindowDays)
	}
	limit, n := MaxTechnologists, 0
	if role == FrontDesk {
		limit = MaxDesk
	}
	for i := range st.staff {
		if st.emps[i].terms.Role == role && st.staff[i].Until > st.day {
			n++
		}
	}
	for _, h := range p.hires {
		if h.terms.Role == role {
			n++
		}
	}
	if n >= limit {
		return bad("headcount_limit", "The practice already has %d %s staff (recruiting, training, active or on notice): the limit is %d.",
			n, strings.ReplaceAll(string(role), "_", " "), limit)
	}
	h := &hireSpec{req: a.RequestID, canon: canon, terms: t, offer: offer, tier: tier, lead: lead, fee: offer.FeeCents,
		from: from, productive: done + 1, training: need}
	p.reqs[a.RequestID] = true
	p.hires = append(p.hires, h)
	return staffOutcome{ok: true, code: api.ActionAccepted, hire: h,
		effective: st.cal.Date(st.cal.StartOfDay(from)).String(), msg: st.hireText(h, "")}
}

func hasDup(s []string) bool {
	c := slices.Clone(s)
	slices.Sort(c)
	return len(slices.Compact(c)) != len(s)
}

// hireCanon is a hire request's canonical content: skills in table order,
// covers by room and start, the rest as given.
func hireCanon(a api.Action) string {
	c := api.Action{Type: a.Type, RequestID: a.RequestID, Role: a.Role, Skills: canonicalSkills(a.Skills),
		RateCents: a.RateCents, ContractedWeeklyMinutes: a.ContractedWeeklyMinutes, StartDate: a.StartDate}
	if a.Roster != nil {
		r := *a.Roster
		for _, w := range []*api.Week{&r.A, &r.B} {
			for _, d := range []**api.Shift{&w.Mon, &w.Tue, &w.Wed, &w.Thu, &w.Fri, &w.Sat, &w.Sun} {
				if *d == nil {
					continue
				}
				s := **d
				s.Covers = slices.Clone(s.Covers)
				slices.SortFunc(s.Covers, func(x, y api.Cover) int {
					if x.Room != y.Room {
						return strings.Compare(x.Room, y.Room)
					}
					return strings.Compare(x.From, y.From)
				})
				*d = &s
			}
		}
		c.Roster = &r
	}
	raw, err := json.Marshal(c)
	if err != nil {
		panic(err)
	}
	return string(raw)
}

// clockHM parses a strict "HH:MM" (00:00–23:59).
func clockHM(s string) (simtime.Duration, bool) {
	if len(s) != 5 || s[2] != ':' {
		return 0, false
	}
	c, err := simtime.ParseClock(s)
	return c, err == nil && c < simtime.Day
}

// hireRoster converts and validates a hire's A/B roster into t (§6.2,
// §6.4): one shift or none per weekday and rota week; times inside the day
// and the shift cap; breaks strictly inside, ordered, at least BreakMin in
// a shift longer than LongShift and no stretch longer than LongShift
// without one; a technologist assigned to a room of a skill they hold
// (covers likewise, inside the shift, no room twice at once), front desk
// to the desk; at most MaxWorkDays a week; net rostered time in either
// rota week within the contract; MinRest between one shift's cap and the
// next start, across weeks.
func (st *state) hireRoster(t *EmployeeConfig, r *api.Roster) error {
	if err := st.parseRoster(t, r); err != nil {
		return err
	}
	return st.rosterRules(t)
}

// parseRoster converts r into t's Roster and Alt, checking each shift on
// its own (hireShift).
func (st *state) parseRoster(t *EmployeeConfig, r *api.Roster) error {
	var weeks [2][7]*ShiftTemplate
	for k, w := range []*api.Week{&r.A, &r.B} {
		days := w.Days()
		for d, sh := range days {
			if sh == nil {
				continue
			}
			wd := simtime.Weekday((d + 1) % 7) // Days is Monday first
			path := fmt.Sprintf("roster.%s.%s", "AB"[k:k+1], wd)
			tpl, err := st.hireShift(path, t, sh)
			if err != nil {
				return err
			}
			weeks[k][wd] = tpl
		}
	}
	t.Roster, t.Alt = [7]*ShiftTemplate{}, [7]*ShiftTemplate{}
	for wd := range 7 {
		a, b := weeks[0][wd], weeks[1][wd]
		switch {
		case a == nil && b == nil:
		case a != nil && b != nil && sameTemplate(a, b):
			t.Roster[wd] = a
		case b == nil:
			a.Weeks, t.Roster[wd] = WeekA, a
		case a == nil:
			b.Weeks, t.Roster[wd] = WeekB, b
		default:
			a.Weeks, b.Weeks = WeekA, WeekB
			t.Roster[wd], t.Alt[wd] = a, b
		}
	}
	return nil
}

// rosterRules checks t's whole A/B roster: at most MaxWorkDays a rota
// week, net rostered time in each within the contract, and MinRest
// between one shift's cap and the next start across the 14-day cycle.
func (st *state) rosterRules(t *EmployeeConfig) error {
	at := func(n int) *ShiftTemplate { // n: 0 = week A Monday … 13 = week B Sunday
		return t.ShiftOn(simtime.Weekday((n%7+1)%7), WeekA+(n/7)%2)
	}
	for k := range 2 {
		n := 0
		var net simtime.Duration
		for d := range 7 {
			if tpl := at(7*k + d); tpl != nil {
				n++
				net += netTime(tpl)
			}
		}
		week := "AB"[k : k+1]
		switch {
		case n > MaxWorkDays:
			return fmt.Errorf("roster.%s: %d working days, at most %d", week, n, MaxWorkDays)
		case net > t.Contract.Weekly:
			return fmt.Errorf("roster.%s: %s rostered net of breaks, more than the %s contract (a part-time contract cannot carry a longer regular roster)",
				week, hours(net), hours(t.Contract.Weekly))
		}
	}
	// Rest between consecutive days over the 14-day cycle (A Mon … B Sun).
	for n := 0; n < 14; n++ {
		a, b := at(n), at((n+1)%14)
		if a == nil || b == nil {
			continue
		}
		if rest := simtime.Day + b.From - st.cfg.Policies.ShiftCap(a.From); rest < MinRest {
			return fmt.Errorf("roster: only %s between the latest permitted end of week %s %s's shift (%s) and the next start (%s); at least %s",
				rest, "AB"[(n/7)%2:(n/7)%2+1], simtime.Weekday((n%7+1)%7), simtime.FormatClock(st.cfg.Policies.ShiftCap(a.From)),
				simtime.FormatClock(b.From), MinRest)
		}
	}
	return nil
}

func (st *state) hireShift(path string, t *EmployeeConfig, sh *api.Shift) (*ShiftTemplate, error) {
	from, ok1 := clockHM(sh.Start)
	to, ok2 := clockHM(sh.End)
	switch {
	case !ok1 || !ok2:
		return nil, fmt.Errorf("%s: start and end must be HH:MM", path)
	case from >= to:
		return nil, fmt.Errorf("%s: shift %s–%s ends before it starts", path, sh.Start, sh.End)
	case to > st.cfg.Policies.ShiftCap(from):
		return nil, fmt.Errorf("%s: shift %s–%s runs past its cap %s (the %d-hour limit or accounting close)", path, sh.Start, sh.End,
			simtime.FormatClock(st.cfg.Policies.ShiftCap(from)), st.cfg.Policies.MaxShift/simtime.Hour)
	}
	tpl := &ShiftTemplate{Span: Span{from, to}, Assignment: sh.Assignment}
	last := from
	var brk simtime.Duration
	for k, b := range sh.Breaks {
		bf, ok1 := clockHM(b[0])
		bt, ok2 := clockHM(b[1])
		switch {
		case !ok1 || !ok2:
			return nil, fmt.Errorf("%s.breaks[%d]: times must be HH:MM", path, k)
		case bf <= last || bt <= bf || bt >= to:
			return nil, fmt.Errorf("%s.breaks[%d]: %s–%s must lie strictly inside the shift, after the previous break", path, k, b[0], b[1])
		case bf-last > LongShift:
			return nil, fmt.Errorf("%s: %s worked without a break before %s (at most %s)", path, bf-last, b[0], LongShift)
		}
		tpl.Breaks = append(tpl.Breaks, Span{bf, bt})
		brk += bt - bf
		last = bt
	}
	switch {
	case len(sh.Breaks) > 4:
		return nil, fmt.Errorf("%s: at most 4 breaks", path)
	case to-last > LongShift:
		return nil, fmt.Errorf("%s: %s worked without a break before the end (at most %s)", path, to-last, LongShift)
	case to-from > LongShift && brk < BreakMin:
		return nil, fmt.Errorf("%s: a shift longer than %s needs at least %s of unpaid break", path, LongShift, BreakMin)
	}
	if t.Role == FrontDesk {
		if sh.Assignment != AssignDesk || len(sh.Covers) > 0 {
			return nil, fmt.Errorf("%s: front desk staff are assigned to %q with no covers", path, AssignDesk)
		}
		return tpl, nil
	}
	qualified := func(id string) error {
		r := slices.IndexFunc(st.cfg.Rooms, func(rc RoomConfig) bool { return rc.ID == id })
		switch {
		case r < 0:
			return fmt.Errorf("%s: unknown room %q", path, id)
		case !t.HasSkill(st.cfg.Rooms[r].Modality):
			return fmt.Errorf("%s: room %s is %s, not one of %s", path, id, st.cfg.Rooms[r].Modality, strings.Join(t.Skills, "+"))
		}
		return nil
	}
	if err := qualified(sh.Assignment); err != nil {
		return nil, err
	}
	if len(sh.Covers) > 4 {
		return nil, fmt.Errorf("%s: at most 4 covers", path)
	}
	for k, cv := range sh.Covers {
		cf, ok1 := clockHM(cv.From)
		ct, ok2 := clockHM(cv.Until)
		where := fmt.Sprintf("%s.covers[%d]", path, k)
		switch {
		case !ok1 || !ok2:
			return nil, fmt.Errorf("%s: times must be HH:MM", where)
		case cf < from || ct > to || cf >= ct:
			return nil, fmt.Errorf("%s: %s–%s must lie inside the shift", where, cv.From, cv.Until)
		case cv.Room == sh.Assignment:
			return nil, fmt.Errorf("%s: %s is already the assignment", where, cv.Room)
		}
		if err := qualified(cv.Room); err != nil {
			return nil, fmt.Errorf("%s: %v", where, err)
		}
		for _, o := range tpl.Covers {
			if o.Room == cv.Room && o.From < ct && cf < o.To {
				return nil, fmt.Errorf("%s: overlaps another cover of %s", where, cv.Room)
			}
		}
		tpl.Covers = append(tpl.Covers, Cover{Room: cv.Room, Span: Span{cf, ct}})
	}
	return tpl, nil
}

func netTime(t *ShiftTemplate) simtime.Duration {
	n := t.To - t.From
	for _, b := range t.Breaks {
		n -= b.To - b.From
	}
	return n
}

func sameTemplate(a, b *ShiftTemplate) bool {
	return a.Span == b.Span && a.Assignment == b.Assignment && a.Remote == b.Remote &&
		slices.Equal(a.Breaks, b.Breaks) && slices.Equal(a.Covers, b.Covers)
}

// trainingEnd is the last training day of a hire starting on day from: the
// first day by which its rostered time, net of breaks, reaches need
// (training is attended exactly as rostered: a trainee is never busy or
// held over, and no extension applies to a training day). ok is false if
// that is not within TrainingWindowDays of the start.
func (st *state) trainingEnd(t *EmployeeConfig, from int64, need simtime.Duration) (int64, bool) {
	var got simtime.Duration
	for d := from; d < from+TrainingWindowDays; d++ {
		date := st.cal.Date(st.cal.StartOfDay(d))
		if tpl := t.ShiftOn(date.Weekday(), st.cfg.RotaWeek(date)); tpl != nil {
			if got += netTime(tpl); got >= need {
				return d, true
			}
		}
	}
	return 0, false
}

// nextHireID is the first unused H-number ID (scenario IDs included).
func (st *state) nextHireID() string {
	for n := int64(1); ; n++ {
		id := pubID('H', n, 5)
		if !slices.ContainsFunc(st.staff, func(r employment) bool { return r.ID == id }) {
			return id
		}
	}
}

// commitHire adds the hire to the registry: its ID, record, recruiting fee
// and its future capacity (from the productive date, on days already in
// the booking horizon).
func (st *state) commitHire(h *hireSpec, action int) int {
	id := st.nextHireID()
	t := h.terms
	t.ID, t.Name = id, id
	terms := &t
	i := len(st.staff)
	st.emps = append(st.emps, employee{cfg: terms, terms: terms, shift: -1})
	st.staff = append(st.staff, employment{ID: id, From: h.from, Until: noEnd, Productive: h.productive, Contract: t.Contract,
		Hire: &hireRecord{Terms: terms, RequestID: h.req, Canon: h.canon, Action: action, Batch: st.actions.batches, At: st.now,
			Offer: h.tier, FeeCents: h.fee, Training: h.training, LeadDays: h.lead}})
	st.staff[i].Stage = st.staff[i].stageOn(st.day)
	st.met.cum.emps = append(st.met.cum.emps, empTally{})
	// Today's plan does not change (the start is in the future); its
	// per-employee arrays grow with the registry.
	st.plan.ByEmp = append(st.plan.ByEmp, -1)
	st.plan.emps = append(st.plan.emps, nil)
	st.plan.Reading = append(st.plan.Reading, nil)
	st.fin.led.mustPost(st.now, entryKey{kind: entRecruiting, a: int64(i)},
		line{acctRecruiting, -1, h.fee, -1}, line{acctAP, -1, -h.fee, -1})
	st.fin.apAccrued += h.fee
	st.recoverDays(h.productive)
	st.emit(st.now, api.Info, "actions", st.hireText(h, id))
	return i
}

// recoverDays recomputes the planned coverage and slot staffing of the
// materialized days from day on, for a change that only adds capacity.
func (st *state) recoverDays(from int64) {
	for d := max(from, st.day+1); d < int64(len(st.dem.days)); d++ {
		st.applyCoverage(d, st.planFor(d))
	}
}

// applyCoverage sets day's planned coverage and every slot's staffing.
func (st *state) applyCoverage(day int64, plan DayPlan) {
	ds := st.dem.days[day]
	if plan.Hours == nil {
		return
	}
	cov := plan.Coverage()
	ds.segs = cov.Segments
	for r := range ds.rooms {
		for k := range ds.rooms[r] {
			st.coverSlot(&cov, r, &ds.rooms[r][k])
		}
	}
}

func (st *state) hireText(h *hireSpec, id string) string {
	t := &h.terms
	who := "Hire: "
	if id != "" {
		who = "Hired " + id + ": "
	}
	date := func(d int64) string { return st.dayLabel(d) }
	var a, b simtime.Duration
	for wd := range 7 {
		if x := t.ShiftOn(simtime.Weekday(wd), WeekA); x != nil {
			a += netTime(x)
		}
		if x := t.ShiftOn(simtime.Weekday(wd), WeekB); x != nil {
			b += netTime(x)
		}
	}
	rostered := "rostered " + hours(a)
	if a != b {
		rostered = fmt.Sprintf("rostered %s (week A), %s (week B)", hours(a), hours(b))
	}
	w := t.RateCents * int64(t.Contract.Weekly) / 3600
	g := w + roundDiv(w*st.cfg.Finance.BenefitsBP, 10_000)
	return fmt.Sprintf("%s%s at %s/h (%s, %d-day lead), %s/week contract, %s · starts %s: paid onboarding, %s of training, no patient capacity · independent from %s · recruiting fee %s (payable on the 1st) · guaranteed %s/week with benefits.",
		who, offerLabel(t), rate(t.RateCents), h.tier, h.lead, hours(t.Contract.Weekly), rostered, date(h.from), hours(h.training), date(h.productive),
		money(h.fee), money(g))
}

// rate formats an hourly rate: "$38.00".
func rate(c int64) string { return fmt.Sprintf("$%d.%02d", c/100, c%100) }

// --- terminate ----------------------------------------------------------

func (st *state) findEmployee(id string) int {
	for i := range st.staff {
		if st.staff[i].ID == id {
			return i
		}
	}
	return -1
}

func (st *state) checkTerminate(a api.Action, p *proposal) staffOutcome {
	bad := func(code, format string, args ...any) staffOutcome {
		return staffOutcome{code: code, msg: fmt.Sprintf(format, args...)}
	}
	switch {
	case a.Minutes != 0 || a.Date != "" || a.Policy != "" || a.Cutoff != "" || a.RequestID != "" || a.Role != "" ||
		a.Skills != nil || a.RateCents != 0 || a.ContractedWeeklyMinutes != 0 || a.Roster != nil || a.StartDate != "" || a.Op != "" || a.Effective != "" ||
		a.SiteAllocationBP != 0 || teleFields(a):
		return bad("invalid_schema", "terminate takes employee and lastDay only.")
	case a.Employee == "" || a.LastDay == "":
		return bad("invalid_schema", "terminate needs employee and lastDay.")
	}
	i := st.findEmployee(a.Employee)
	if i < 0 {
		return bad("unknown_employee", "No employee %q.", a.Employee)
	}
	if p.emps[i] {
		return bad("invalid_parameters", "%s is terminated twice in this batch.", a.Employee)
	}
	r := &st.staff[i]
	t := st.emps[i].terms
	label := st.staffLabel(i)
	if t.Pay != Hourly || (t.Role != Technologist && t.Role != FrontDesk) {
		return bad("wrong_role", "Only hourly technologists and front desk staff can be terminated (%s is %s).", label, strings.ReplaceAll(string(t.Role), "_", " "))
	}
	if r.Until <= st.day {
		return bad("not_employed", "%s left the practice on %s.", label, st.cal.Date(st.cal.StartOfDay(r.Until-1)))
	}
	date, err := simtime.ParseDate(a.LastDay)
	if err != nil {
		return bad("invalid_date", "lastDay %q is not YYYY-MM-DD.", a.LastDay)
	}
	last := st.cal.DayIndex(st.cal.StartOfDate(date))
	if x := r.Exit; x != nil {
		if x.LastDay == last {
			return staffOutcome{ok: true, code: api.ActionUnchanged,
				msg: fmt.Sprintf("%s is already leaving with last day %s: nothing changed.", label, a.LastDay)}
		}
		return bad("already_terminating", "%s is already leaving with last day %s: notice cannot be moved or withdrawn.", label,
			st.cal.Date(st.cal.StartOfDay(x.LastDay)))
	}
	earliest := max(st.day, r.From) + NoticeDays
	switch {
	case last < earliest:
		return bad("notice_too_short", "The last day must be at least %d days after today and the start date: %s at the earliest.",
			NoticeDays, st.cal.Date(st.cal.StartOfDay(earliest)))
	case last > st.day+st.cfg.Policies.BookingHorizonDays:
		return bad("beyond_horizon", "lastDay %s is beyond the %d-day booking horizon.", a.LastDay, st.cfg.Policies.BookingHorizonDays)
	}
	ts := &termSpec{emp: i, lastDay: last}
	p.emps[i] = true
	p.terms = append(p.terms, ts)
	return staffOutcome{ok: true, code: api.ActionAccepted, term: ts,
		effective: st.cal.Date(st.cal.StartOfDay(last + 1)).String(), msg: st.termText(ts)}
}

func (st *state) termText(ts *termSpec) string {
	i := ts.emp
	r := &st.staff[i]
	notice := ts.lastDay - st.day
	pay := st.nextPayday(ts.lastDay)
	svc := ""
	switch r.serviceOn(st.day) {
	case stageRecruiting:
		svc = " (before the start date: the fee is not refunded and the contract runs from the start)"
	case stageOnboarding:
		svc = " (in training)"
	}
	return fmt.Sprintf("%s terminated%s: last day %s, %d days' notice with contract pay continuing (%s/week guaranteed) · severance %s recognized now, paid %s.",
		st.staffLabel(i), svc, st.dayLabel(ts.lastDay), notice, money(st.weeklyGuarantee(i)), money(st.severance(i)), st.dayLabel(pay))
}

// commitTerminate fixes the end, recognizes severance and voids approved
// extensions on and after the end. Booking reconciliation follows once
// for the batch.
func (st *state) commitTerminate(ts *termSpec, action int) {
	i := ts.emp
	r := &st.staff[i]
	sev := st.severance(i)
	r.Exit = &exitRecord{LastDay: ts.lastDay, Action: action, Batch: st.actions.batches, At: st.now,
		SeveranceCents: sev, PayDay: st.nextPayday(ts.lastDay)}
	r.Until = ts.lastDay + 1
	r.Stage = r.stageOn(st.day)
	st.fin.led.mustPost(st.now, entryKey{kind: entSeverance, a: int64(i)},
		line{acctSeverance, -1, sev, -1}, line{acctSeverancePayable, -1, -sev, -1})
	st.emit(st.now, api.Info, "actions", st.termText(ts))
	st.voidRosters(i, action)
	var days []int64
	for d := range st.ext {
		if d >= r.Until && st.ext[d][i] > 0 {
			days = append(days, d)
		}
	}
	slices.Sort(days)
	for _, d := range days {
		st.emit(st.now, api.Info, "actions", fmt.Sprintf("Approved extension voided: %s on %s (+%s): after the last day",
			st.staffLabel(i), st.dayLabel(d), st.ext[d][i]))
		delete(st.ext[d], i)
		if len(st.ext[d]) == 0 {
			delete(st.ext, d)
		}
	}
}
