package sim

import (
	"fmt"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Employment (V0.2_DESIGN §5.1–§5.4): every employee has a registry record
// in state — stable identity, lifecycle, employed dates, contract and the
// week's pay counters — apart from the immutable terms (name, role, skills,
// rate, roster). An initial employee's terms are the scenario's
// (cfg.Employees[i]); a hired employee's are frozen in its hire record.
// Record i is employee i: the registry is append-only and indices and IDs
// are never reused, departed employees included.
//
// Lifecycle (§5.3): recruiting (hire accepted, before the start date) →
// onboarding (employed and paid, training: no patient-service capacity) →
// active (independent work from the midnight after training completes).
// Notice is an overlay: an accepted termination fixes the last day, and
// the service stage carries on until then; the employee is terminated
// from the following midnight. Initial employees are active and trained
// from the epoch.

// stage is an employee's lifecycle stage (its display stage: notice
// overlays the service stage).
type stage uint8

const (
	stageRecruiting stage = iota
	stageOnboarding
	stageActive
	stageNotice
	stageTerminated
)

var stageNames = [...]string{stageRecruiting: api.EmploymentRecruiting, stageOnboarding: api.EmploymentOnboarding,
	stageActive: api.EmploymentActive, stageNotice: api.EmploymentNotice, stageTerminated: api.EmploymentTerminated}

// noEnd marks open-ended employment.
const noEnd int64 = 1 << 62

type employment struct {
	ID    string
	Stage stage
	// From is the first employed day index; Until the first day index no
	// longer employed (noEnd: open-ended). The contract accrues on
	// employed days only. Productive is the first day of independent work:
	// From for an initial employee, the day after training completes for a
	// hire (capacity exists only on days in [Productive, Until)).
	From, Until, Productive int64
	Contract                Contract
	// The current week's guaranteed pay (§5.4), in cents: the guarantee
	// earned on the week's employed days so far (weekG), worked pay at the
	// regular rate (weekB: worked base, premium excluded), and the
	// provision posted for the shortfall max(0, weekG − weekB), as wages
	// and benefits. The provision is accrued payroll that is not yet final
	// (openGuarantee); the week closes on Sunday, or on the last employed
	// day, at accounting close.
	weekG, weekB            int64
	provWages, provBenefits int64
	// Hire is a hired employee's terms and acceptance (nil: initial).
	Hire *hireRecord
	// Exit is the accepted termination (nil: none).
	Exit *exitRecord
	// Trained is the onboarding time attended (paid, net of breaks).
	Trained simtime.Duration
	// trainView is the training-day view of the hire's terms (role
	// Trainee): derived from Hire.Terms, not state.
	trainView *EmployeeConfig
	// Rosters are the dated roster versions in date order (roster.go);
	// RosterLog every set, removal and voiding, append-only. Both are
	// empty without set_roster (and then not encoded in the state digest).
	Rosters   []rosterVersion
	RosterLog []rosterEntry
}

// hireRecord is an accepted hire (§5.1, §6.3).
type hireRecord struct {
	Terms     *EmployeeConfig // frozen at acceptance; Name is the public ID
	RequestID string
	Canon     string // the request's canonical content, for idempotency
	Action    int    // the accepting action's index in the action record
	Batch     int64
	At        simtime.Time
	Offer     string // "standard" or "premium" pay tier
	FeeCents  int64  // recruiting fee
	Training  simtime.Duration
	LeadDays  int64
}

// exitRecord is an accepted termination (§5.3, §5.4).
type exitRecord struct {
	LastDay        int64 // inclusive; employment ends at the next midnight
	Action         int
	Batch          int64
	At             simtime.Time
	SeveranceCents int64 // recognized at acceptance (wages and benefits)
	PayDay         int64 // the first payday after the last day
	Paid           bool
}

func (r *employment) employed(day int64) bool { return r.From <= day && day < r.Until }

// stageOn is the display stage on day.
func (r *employment) stageOn(day int64) stage {
	switch {
	case day >= r.Until:
		return stageTerminated
	case r.Exit != nil:
		return stageNotice
	case day < r.From:
		return stageRecruiting
	case day < r.Productive:
		return stageOnboarding
	}
	return stageActive
}

// serviceOn is the service stage on day, under any notice.
func (r *employment) serviceOn(day int64) stage {
	switch {
	case day >= r.Until:
		return stageTerminated
	case day < r.From:
		return stageRecruiting
	case day < r.Productive:
		return stageOnboarding
	}
	return stageActive
}

// initStaff opens a record for every initial employee, in scenario order.
func (st *state) initStaff() {
	for i := range st.emps {
		c := st.emps[i].terms
		st.staff = append(st.staff, employment{ID: c.ID, Stage: stageActive, Until: noEnd, Contract: c.Contract})
	}
}

// workingTerms is employee i's terms as working on day, with the roster
// version in effect then: nil when not employed, the training view in
// onboarding, the terms otherwise.
func (st *state) workingTerms(i int, day int64) *EmployeeConfig {
	r := &st.staff[i]
	return st.workingView(i, day, r.Rosters, r.Productive, r.Until)
}

// dayTerms is every employee's working terms on day (planFor's input).
func (st *state) dayTerms(day int64) []*EmployeeConfig {
	out := make([]*EmployeeConfig, len(st.staff))
	for i := range st.staff {
		out[i] = st.workingTerms(i, day)
	}
	return out
}

// lifecycleDayStart moves every employee to the day's stage at midnight,
// in registry order, before finance and the day's plan (§10.1): starts,
// training completions, endings. Training must be complete when
// independent work begins (the projected plan is the attended one).
func (st *state) lifecycleDayStart(day int64) {
	for i := range st.staff {
		r := &st.staff[i]
		if r.Hire == nil && r.Exit == nil {
			continue // an initial employee: active from the epoch, no end
		}
		next := r.stageOn(day)
		svcWas, svc := r.serviceOn(day-1), r.serviceOn(day)
		if next == r.Stage && svc == svcWas {
			continue // (notice overlays the stage: a service change still logs)
		}
		name := st.emps[i].terms.Name
		at := st.cal.StartOfDay(day)
		switch {
		case svc == stageTerminated:
			st.emit(at, api.Info, "staff", fmt.Sprintf("%s has left the practice (last day %s)", st.staffLabel(i), st.dayLabel(r.Until-1)))
		case svc == stageOnboarding && svcWas == stageRecruiting:
			st.emit(at, api.Info, "staff", fmt.Sprintf("%s starts: onboarding, %s of paid training (no patient capacity) · independent from %s",
				st.staffLabel(i), hours(r.Hire.Training), st.dayLabel(r.Productive)))
		case svc == stageActive && svcWas != stageActive:
			if r.Hire != nil && r.Trained < r.Hire.Training {
				fail("%s independent after %s of %s training", name, r.Trained, r.Hire.Training)
			}
			st.emit(at, api.Info, "staff", fmt.Sprintf("%s completed training (%s attended): works independently from today", st.staffLabel(i), hours(r.Trained)))
		}
		r.Stage = next
	}
}

// staffLabel names employee i for the log: "Sam (E04)", a hire by its ID
// and offer ("H00001 (XR+CT technologist)").
func (st *state) staffLabel(i int) string {
	t := st.emps[i].terms
	if st.staff[i].Hire != nil {
		return fmt.Sprintf("%s (%s)", t.ID, offerLabel(t))
	}
	return fmt.Sprintf("%s (%s)", t.Name, t.ID)
}

// guaranteeClose runs at accounting close, after everyone has gone home
// and their worked pay is posted: each hourly employee's guarantee earns
// the day's share, the provision moves to the week's shortfall (a signed
// change; none when unchanged), and on Sunday — or the last employed day —
// the week closes: the provision becomes final ordinary payroll and the
// counters reset.
//
// A contract at or below the roster never posts: a day's guarantee is at
// most the pay for its original rostered time, which worked time (never
// less than the roster in V0.1) already pays.
func (st *state) guaranteeClose() {
	date := st.cal.Date(st.cal.StartOfDay(st.day))
	wd, week := date.Weekday(), st.cfg.RotaWeek(date)
	for i := range st.staff {
		r := &st.staff[i]
		c := st.emps[i].terms
		if c.Pay != Hourly {
			continue
		}
		if r.employed(st.day) {
			r.weekG += r.Contract.DayCents(c.RateCents, week, wd)
		}
		q := max(0, r.weekG-r.weekB)
		qb := roundDiv(q*st.cfg.Finance.BenefitsBP, 10_000)
		dw, db := q-r.provWages, qb-r.provBenefits
		if dw != 0 || db != 0 {
			var lines []line
			if dw != 0 {
				lines = append(lines, line{acctWages, -1, dw, -1})
			}
			if db != 0 {
				lines = append(lines, line{acctBenefits, -1, db, -1})
			}
			lines = append(lines, line{acctAccruedPayroll, -1, -(dw + db), -1})
			st.fin.led.mustPost(st.now, entryKey{kind: entGuarantee, a: int64(i), b: st.day}, lines...)
			st.fin.payrollAccrued += dw + db
			r.provWages, r.provBenefits = q, qb
		}
		if wd != simtime.Sunday && st.day != r.Until-1 {
			continue
		}
		if q > 0 {
			when := "this week"
			if wd != simtime.Sunday {
				when = "in the final week"
			}
			st.emit(st.now, api.Info, "finance", fmt.Sprintf("%s's guaranteed pay %s: %s for contracted time not worked (%s contract)",
				c.Name, when, money(q+qb), hours(r.Contract.Weekly)))
		}
		r.weekG, r.weekB, r.provWages, r.provBenefits = 0, 0, 0, 0
	}
}

// openGuarantee is the guaranteed pay provisioned this week and not yet
// final (wages and benefits): it may still be reversed as worked pay
// replaces it, so no payday pays it before the week closes.
func (st *state) openGuarantee() int64 {
	var t int64
	for i := range st.staff {
		t += st.staff[i].provWages + st.staff[i].provBenefits
	}
	return t
}

// severance is what terminating hourly employee i owes beyond pay earned
// (§5.4): two contracted weeks at the base rate, plus benefits; no assumed
// overtime, no dependence on the roster. Salaried staff cannot be
// terminated in V0.2 (0).
func (st *state) severance(i int) int64 {
	c := st.emps[i].terms
	if c.Pay != Hourly {
		return 0
	}
	w := 2 * (c.RateCents * int64(st.staff[i].Contract.Weekly) / 3600)
	return w + roundDiv(w*st.cfg.Finance.BenefitsBP, 10_000)
}

// weeklyGuarantee is employee i's full-week contract cost: r × C plus
// benefits.
func (st *state) weeklyGuarantee(i int) int64 {
	c := st.emps[i].terms
	if c.Pay != Hourly {
		return 0
	}
	w := c.RateCents * int64(st.staff[i].Contract.Weekly) / 3600
	return w + roundDiv(w*st.cfg.Finance.BenefitsBP, 10_000)
}

// paySeverance pays, on a payday, every severance due: recognized at
// termination and payable on the first payday after the last day.
func (st *state) paySeverance(now simtime.Time, day int64) {
	led := &st.fin.led
	for i := range st.staff {
		x := st.staff[i].Exit
		if x == nil || x.Paid || x.PayDay != day {
			continue
		}
		led.mustPost(now, entryKey{kind: entSeverancePay, a: int64(i)},
			line{acctSeverancePayable, -1, x.SeveranceCents, -1}, line{acctCash, -1, -x.SeveranceCents, -1})
		x.Paid = true
		st.fin.severancePaid += x.SeveranceCents
		st.noteCash(now)
		st.emit(now, api.Info, "finance", fmt.Sprintf("Severance paid %s to %s", money(x.SeveranceCents), st.staffLabel(i)))
	}
}

// checkStaff verifies the registry against the employees and the ledger.
func (st *state) checkStaff() error {
	if len(st.staff) != len(st.emps) || len(st.met.cum.emps) != len(st.emps) {
		return fmt.Errorf("%d employment records, %d employees, %d metric rows", len(st.staff), len(st.emps), len(st.met.cum.emps))
	}
	var owed int64
	for i := range st.staff {
		r := &st.staff[i]
		t := st.emps[i].terms
		switch {
		case r.ID != t.ID:
			return fmt.Errorf("employment record %d is %s, employee %s", i, r.ID, t.ID)
		case r.Hire != nil && r.Hire.Terms != t, r.Hire == nil && t != &st.cfg.Employees[i]:
			return fmt.Errorf("%s: terms are not the record's", r.ID)
		case r.provWages < 0 || r.provBenefits < 0 || r.weekG < 0 || r.weekB < 0:
			return fmt.Errorf("%s: negative week pay counters", r.ID)
		case t.Pay != Hourly && (r.Contract != Contract{} || r.weekG != 0 || r.provWages != 0):
			return fmt.Errorf("%s: salaried with a contract guarantee", r.ID)
		case r.From > r.Productive || r.From >= r.Until:
			return fmt.Errorf("%s: employment dates out of order (%d, %d, %d)", r.ID, r.From, r.Productive, r.Until)
		case r.Stage != r.stageOn(st.day):
			return fmt.Errorf("%s: stage %s on day %d, dates say %s", r.ID, stageNames[r.Stage], st.day, stageNames[r.stageOn(st.day)])
		case (r.Exit != nil) != (r.Until != noEnd):
			return fmt.Errorf("%s: an end date without a termination, or the reverse", r.ID)
		}
		if x := r.Exit; x != nil {
			if x.LastDay+1 != r.Until || x.PayDay <= x.LastDay {
				return fmt.Errorf("%s: termination dates inconsistent", r.ID)
			}
			if !x.Paid {
				owed += x.SeveranceCents
			}
		}
		if r.Hire != nil && st.day >= r.Productive && r.Productive < r.Until && r.Trained < r.Hire.Training {
			return fmt.Errorf("%s: independent with %s of %s training", r.ID, r.Trained, r.Hire.Training)
		}
	}
	if open := st.openGuarantee(); open > -st.fin.led.bal[acctAccruedPayroll] {
		return fmt.Errorf("open guarantee %d exceeds accrued payroll %d", open, -st.fin.led.bal[acctAccruedPayroll])
	}
	if owed != -st.fin.led.bal[acctSeverancePayable] {
		return fmt.Errorf("severance payable %d, unpaid terminations owe %d", -st.fin.led.bal[acctSeverancePayable], owed)
	}
	return st.checkRosters()
}

// observeEmployment fills employee i's employment, contract and lifecycle.
func (st *state) observeEmployment(x *api.Employee, i int) {
	r := &st.staff[i]
	x.Employment = stageNames[r.Stage]
	x.ContractMin = int64(r.Contract.Weekly / simtime.Minute)
	x.ContractInferred = r.Contract.Inferred
	x.GuaranteeCents = r.provWages + r.provBenefits
	x.SeveranceCents = st.severance(i)
	date := func(d int64) string { return st.cal.Date(st.cal.StartOfDay(d)).String() }
	if h := r.Hire; h != nil {
		x.Hired = &api.HireInfo{StartDate: date(r.From), ProductiveDate: date(r.Productive), Offer: h.Offer,
			RecruitingFeeCents: h.FeeCents, TrainingMin: int64(h.Training / simtime.Minute),
			TrainedMin: int64(r.Trained / simtime.Minute), RequestID: h.RequestID, ServiceStage: stageNames[r.serviceOn(st.day)]}
	}
	st.observeRosters(x, i, st.day)
	if e := r.Exit; e != nil {
		x.Leaving = &api.ExitInfo{LastDay: date(e.LastDay), SeveranceCents: e.SeveranceCents,
			SeverancePayday: date(e.PayDay), SeverancePaid: e.Paid}
		x.SeveranceCents = 0 // owed already: in Leaving
	}
}
