package scenario

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/georgezero/dreamteam-go/internal/sim"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Issue is one validation problem, located by JSON path.
type Issue struct {
	Path string
	Msg  string
}

func (i Issue) String() string {
	if i.Path == "" {
		return i.Msg
	}
	return i.Path + ": " + i.Msg
}

// ValidationError lists every problem found in a scenario.
type ValidationError struct{ Issues []Issue }

func (e *ValidationError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "scenario invalid (%d problem", len(e.Issues))
	if len(e.Issues) != 1 {
		b.WriteString("s")
	}
	b.WriteString("):")
	for _, i := range e.Issues {
		b.WriteString("\n  " + i.String())
	}
	return b.String()
}

var (
	modalities  = []string{"XR", "US", "CT", "MR"}
	roles       = []string{string(sim.FrontDesk), string(sim.Technologist), string(sim.Radiologist), string(sim.OpsManager)}
	walkIn      = []string{"accept_always", "accept_if_gap", "accept_until", "schedule_later"}
	worklist    = []string{"urgent_then_fifo", "fifo", "oldest_first", "shortest_first"}
	late        = []string{"fit_in", "reschedule"}
	supervision = []string{"on_site"} // "virtual" arrives in V0.2
)

const registration = "registration"

type checker struct {
	issues []Issue
}

func (c *checker) fail(p, format string, args ...any) {
	c.issues = append(c.issues, Issue{p, fmt.Sprintf(format, args...)})
}

func (c *checker) oneOf(p, v string, allowed []string) {
	if !slices.Contains(allowed, v) {
		c.fail(p, "%q is not one of %s", v, strings.Join(allowed, ", "))
	}
}

func (c *checker) clock(p, s string) simtime.Duration {
	d, err := simtime.ParseClock(s)
	if err != nil {
		c.fail(p, "%q is not a time of day HH:MM", s)
	}
	return d
}

func (c *checker) date(p, s string) simtime.Date {
	d, err := simtime.ParseDate(s)
	if err != nil {
		c.fail(p, "%q is not a date YYYY-MM-DD", s)
	}
	return d
}

func (c *checker) span(p string, pair [2]string) sim.Span {
	s := sim.Span{From: c.clock(p+"[0]", pair[0]), To: c.clock(p+"[1]", pair[1])}
	if s.From >= s.To {
		c.fail(p, "start %s is not before end %s", pair[0], pair[1])
	}
	return s
}

// weekdays returns the map's keys in Sun..Sat order, reporting bad keys.
func weekdays[V any](c *checker, p string, m map[string]V) []simtime.Weekday {
	var out []simtime.Weekday
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		wd, err := simtime.ParseWeekday(k)
		if err != nil || k != wd.String() {
			c.fail(p+"."+k, "not a weekday (use Mon, Tue, …)")
			continue
		}
		out = append(out, wd)
	}
	slices.Sort(out)
	return out
}

// Compile validates f and converts it to an engine configuration. All
// problems are reported together, each with its JSON path.
func Compile(f *File) (*sim.Config, error) {
	c := &checker{}
	cfg := &sim.Config{ID: f.ID, Name: f.Name, WarmupDays: f.WarmupDays, OpeningCashCents: f.OpeningCashCents}
	if f.Schema != Schema {
		c.fail("schema", "%q is not %q", f.Schema, Schema)
	}
	if f.ID == "" {
		c.fail("id", "required")
	}
	if f.Name == "" {
		c.fail("name", "required")
	}
	cfg.Start = c.date("startDate", f.StartDate)
	cfg.StartClock = c.clock("startTime", f.StartTime)
	if f.WarmupDays < 0 || f.WarmupDays > 3650 {
		c.fail("warmupDays", "%d is outside 0–3650", f.WarmupDays)
	}
	cfg.RotaWeekA = cfg.Start
	if f.RotaWeekA != "" {
		cfg.RotaWeekA = c.date("rotaWeekA", f.RotaWeekA)
	}
	if f.EvaluationDays < 0 || f.EvaluationDays > 3650 {
		c.fail("evaluationDays", "%d is outside 0–3650", f.EvaluationDays)
	}
	cfg.EvaluationDays = f.EvaluationDays
	cfg.Alerts = sim.DefaultAlerts()
	if a := f.Alerts; a != nil {
		minutes := func(field string, v, lo, hi int64) simtime.Duration {
			if v < lo || v > hi {
				c.fail("alerts."+field, "%d is outside %d–%d", v, lo, hi)
			}
			return simtime.Duration(v) * simtime.Minute
		}
		cfg.Alerts = sim.AlertConfig{
			WaitWarn: minutes("waitWarnMin", a.WaitWarnMin, 1, 480), WaitAlert: minutes("waitAlertMin", a.WaitAlertMin, 1, 480),
			CapWarn:     minutes("capWarnMin", a.CapWarnMin, 1, 240),
			ClosingWarn: minutes("closingWarnMin", a.ClosingWarnMin, 1, 480), ClosingAlert: minutes("closingAlertMin", a.ClosingAlertMin, 1, 480),
			ClearAfter:  minutes("clearAfterMin", a.ClearAfterMin, 0, 240),
			BacklogWarn: a.BacklogWarn, LostWarn: a.LostWarn, LowCashCents: a.LowCashCents}
		if a.WaitAlertMin < a.WaitWarnMin {
			c.fail("alerts.waitAlertMin", "below waitWarnMin")
		}
		if a.ClosingAlertMin < a.ClosingWarnMin {
			c.fail("alerts.closingAlertMin", "below closingWarnMin")
		}
		if a.BacklogWarn < 0 || a.BacklogWarn > 10_000 {
			c.fail("alerts.backlogWarn", "%d is outside 0–10000", a.BacklogWarn)
		}
		if a.LostWarn < 0 || a.LostWarn > 10_000 {
			c.fail("alerts.lostWarn", "%d is outside 0–10000", a.LostWarn)
		}
		if a.LowCashCents < 0 {
			c.fail("alerts.lowCashCents", "negative")
		}
	}

	// Policies come first: shift checks need MaxShiftHours.
	p := f.Policies
	c.oneOf("policies.walkIn", p.WalkIn, walkIn)
	c.oneOf("policies.worklist", p.Worklist, worklist)
	c.oneOf("policies.late", p.Late, late)
	c.oneOf("policies.contrastSupervision", p.ContrastSupervision, supervision)
	if p.GraceMin < 0 || p.GraceMin > 120 {
		c.fail("policies.graceMin", "%d is outside 0–120", p.GraceMin)
	}
	if p.OvertimeAllowanceMin < 0 || p.OvertimeAllowanceMin > 240 {
		c.fail("policies.overtimeAllowanceMin", "%d is outside 0–240", p.OvertimeAllowanceMin)
	}
	if p.MaxShiftHours < 1 || p.MaxShiftHours > 16 {
		c.fail("policies.maxShiftHours", "%d is outside 1–16", p.MaxShiftHours)
	}
	cfg.Policies = sim.Policies{
		WalkIn: p.WalkIn, Worklist: p.Worklist, Late: p.Late,
		Grace:               simtime.Duration(p.GraceMin) * simtime.Minute,
		OvertimeAllowance:   simtime.Duration(p.OvertimeAllowanceMin) * simtime.Minute,
		MaxShift:            simtime.Duration(p.MaxShiftHours) * simtime.Hour,
		ContrastSupervision: p.ContrastSupervision,
	}
	switch {
	case p.WalkIn == "accept_until":
		cfg.Policies.WalkInCutoff = c.clock("policies.walkInCutoff", p.WalkInCutoff)
		if cfg.Policies.WalkInCutoff <= 0 || cfg.Policies.WalkInCutoff >= simtime.Day {
			c.fail("policies.walkInCutoff", "must be between 00:00 and 24:00")
		}
	case p.WalkInCutoff != "":
		c.fail("policies.walkInCutoff", "only for walkIn accept_until")
	}

	if len(f.Hours) == 0 {
		c.fail("hours", "required")
	}
	for _, wd := range weekdays(c, "hours", f.Hours) {
		if h := f.Hours[wd.String()]; h != nil {
			s := c.span("hours."+wd.String(), *h)
			cfg.Hours[wd] = &s
		}
	}

	if len(f.Rooms) == 0 {
		c.fail("rooms", "at least one room is required")
	}
	rooms := map[string]sim.RoomConfig{}
	for i, r := range f.Rooms {
		rp := fmt.Sprintf("rooms[%d]", i)
		switch {
		case r.ID == "":
			c.fail(rp+".id", "required")
		case r.ID == sim.AssignDesk || r.ID == sim.AssignReading || r.ID == sim.AssignAdmin:
			c.fail(rp+".id", "%q is reserved", r.ID)
		case rooms[r.ID].ID != "":
			c.fail(rp+".id", "duplicate %q", r.ID)
		}
		c.oneOf(rp+".modality", r.Modality, modalities)
		if r.Name == "" {
			c.fail(rp+".name", "required")
		}
		rc := sim.RoomConfig{ID: r.ID, Name: r.Name, Modality: r.Modality}
		rooms[r.ID] = rc
		cfg.Rooms = append(cfg.Rooms, rc)
	}

	// V0.2 Slice 2: staffing sets grow with the registry (no 64-employee
	// mask); the bound only keeps a scenario file sane.
	if len(f.Employees) > 1000 {
		c.fail("employees", "at most 1000 employees")
	}
	ids := map[string]bool{}
	for i, e := range f.Employees {
		ep := fmt.Sprintf("employees[%d]", i)
		if e.ID == "" {
			c.fail(ep+".id", "required")
		} else if ids[e.ID] {
			c.fail(ep+".id", "duplicate %q", e.ID)
		}
		ids[e.ID] = true
		if e.Name == "" {
			c.fail(ep+".name", "required")
		}
		c.oneOf(ep+".role", e.Role, roles)
		role := sim.Role(e.Role)
		ec := sim.EmployeeConfig{ID: e.ID, Name: e.Name, Role: role, Skills: e.Skills,
			Pay: sim.PayType(e.Pay.Type), RateCents: e.Pay.RateCents, SiteAllocationBP: e.SiteAllocationBP}
		c.employeeSkills(ep, &ec)
		c.oneOf(ep+".pay.type", e.Pay.Type, []string{string(sim.Hourly), string(sim.Salaried)})
		if e.Pay.RateCents <= 0 {
			c.fail(ep+".pay.rateCents", "must be positive")
		}
		if role == sim.Radiologist {
			if e.SiteAllocationBP < 1 || e.SiteAllocationBP > 10_000 {
				c.fail(ep+".siteAllocationBP", "%d is outside 1–10000", e.SiteAllocationBP)
			}
		} else if e.SiteAllocationBP != 0 {
			c.fail(ep+".siteAllocationBP", "only radiologists have a site allocation")
		}
		for _, wd := range weekdays(c, ep+".roster", e.Roster) {
			sh := e.Roster[wd.String()]
			if sh == nil {
				continue
			}
			ec.Roster[wd] = c.shift(fmt.Sprintf("%s.roster.%s", ep, wd), sh, &ec, rooms, cfg.Policies.MaxShift)
		}
		c.contract(ep, e.ContractedWeeklyMinutes, &ec)
		cfg.Employees = append(cfg.Employees, ec)
	}

	c.demand(f, cfg)
	c.finance(f, cfg)
	if len(c.issues) == 0 {
		c.openingSlots(cfg)
	}

	if len(c.issues) > 0 {
		return nil, &ValidationError{c.issues}
	}
	return cfg, nil
}

// contract sets an hourly employee's contract (V0.2_DESIGN §5.2): the
// stated contracted minutes, 1–2400 a week (the V0.2 hire band), or else
// inferred from the roster. Its accrual weights are always the roster's.
func (c *checker) contract(ep string, minutes *int64, e *sim.EmployeeConfig) {
	e.Contract = sim.InferContract(e)
	if minutes == nil {
		return
	}
	switch {
	case e.Pay != sim.Hourly:
		c.fail(ep+".contractedWeeklyMinutes", "only hourly employees have contracted hours")
	case *minutes < 1 || *minutes > 2400:
		c.fail(ep+".contractedWeeklyMinutes", "%d is outside 1–2400", *minutes)
	default:
		e.Contract.Weekly, e.Contract.Inferred = simtime.Duration(*minutes)*simtime.Minute, false
	}
}

func (c *checker) employeeSkills(ep string, e *sim.EmployeeConfig) {
	seen := map[string]bool{}
	for k, s := range e.Skills {
		sp := fmt.Sprintf("%s.skills[%d]", ep, k)
		if seen[s] {
			c.fail(sp, "duplicate %q", s)
		}
		seen[s] = true
		switch e.Role {
		case sim.Technologist:
			c.oneOf(sp, s, modalities)
		case sim.FrontDesk, sim.OpsManager:
			c.oneOf(sp, s, []string{registration})
		case sim.Radiologist:
			c.fail(sp, "radiologists have no listed skills (reading and supervision are implied)")
		}
	}
	switch {
	case e.Role == sim.Technologist && len(e.Skills) == 0:
		c.fail(ep+".skills", "a technologist needs at least one modality")
	case e.Role == sim.FrontDesk && !seen[registration]:
		c.fail(ep+".skills", "front desk staff need %q", registration)
	}
}

func (c *checker) shift(sp string, s *Shift, e *sim.EmployeeConfig, rooms map[string]sim.RoomConfig, maxShift simtime.Duration) *sim.ShiftTemplate {
	t := &sim.ShiftTemplate{Span: c.span(sp, [2]string{s.Start, s.End}), Assignment: s.Assignment}
	if t.To-t.From > maxShift {
		c.fail(sp, "shift %s–%s is longer than maxShiftHours", s.Start, s.End)
	}
	// The shift cap (sim.Policies.ShiftCap) never passes accounting close,
	// so the rostered end must not either.
	if t.To > simtime.Day-1 {
		c.fail(sp, "shift end %s is after accounting close (23:59:59): work may not cross midnight", s.End)
	}
	switch s.Weeks {
	case "":
		t.Weeks = sim.WeekBoth
	case "A":
		t.Weeks = sim.WeekA
	case "B":
		t.Weeks = sim.WeekB
	default:
		c.fail(sp+".weeks", "%q is not A or B", s.Weeks)
	}
	for k, b := range s.Breaks {
		bp := fmt.Sprintf("%s.breaks[%d]", sp, k)
		br := c.span(bp, b)
		if br.From <= t.From || br.To >= t.To {
			c.fail(bp, "break must lie strictly inside the shift")
		}
		if k > 0 && br.From < t.Breaks[k-1].To {
			c.fail(bp, "breaks must be in order and not overlap")
		}
		t.Breaks = append(t.Breaks, br)
	}

	ap := sp + ".assignment"
	switch e.Role {
	case sim.Technologist:
		if r, ok := rooms[s.Assignment]; !ok {
			c.fail(ap, "%q is not a room", s.Assignment)
		} else if !e.HasSkill(r.Modality) {
			c.fail(ap, "%s lacks the %s skill for room %s", e.Name, r.Modality, r.ID)
		}
	case sim.FrontDesk:
		c.oneOf(ap, s.Assignment, []string{sim.AssignDesk})
	case sim.OpsManager:
		c.oneOf(ap, s.Assignment, []string{sim.AssignAdmin, sim.AssignDesk})
	case sim.Radiologist:
		c.oneOf(ap, s.Assignment, []string{sim.AssignReading})
	}

	for k, cv := range s.Covers {
		cp := fmt.Sprintf("%s.covers[%d]", sp, k)
		if e.Role != sim.Technologist {
			c.fail(cp, "only technologists cover rooms")
			continue
		}
		r, ok := rooms[cv.Room]
		switch {
		case !ok:
			c.fail(cp+".room", "%q is not a room", cv.Room)
		case cv.Room == s.Assignment:
			c.fail(cp+".room", "%q is already the assignment", cv.Room)
		case !e.HasSkill(r.Modality):
			c.fail(cp+".room", "%s lacks the %s skill for room %s", e.Name, r.Modality, r.ID)
		}
		w := sim.Cover{Room: cv.Room, Span: t.Span}
		if cv.From != "" {
			w.From = c.clock(cp+".from", cv.From)
		}
		if cv.Until != "" {
			w.To = c.clock(cp+".until", cv.Until)
		}
		if w.From >= w.To || w.From < t.From || w.To > t.To {
			c.fail(cp, "cover window must be non-empty and inside the shift")
		}
		t.Covers = append(t.Covers, w)
	}

	switch s.Location {
	case "", "onsite":
	case "remote":
		if e.Role != sim.Radiologist {
			c.fail(sp+".location", "only radiologists work remotely")
		}
		t.Remote = true
	default:
		c.fail(sp+".location", "%q is not onsite or remote", s.Location)
	}
	return t
}
