package sim

import (
	"fmt"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Employment (V0.2_DESIGN §5.1–§5.4, Slice 1): every employee has a
// registry record in state — stable identity, lifecycle stage, employed
// dates and contract — apart from the immutable scenario terms
// (EmployeeConfig: name, role, skills, rate, roster) that st.emps[i].cfg
// points to. Record i is employee i; the registry is append-only and IDs
// are never reused. Slice 1 has no lifecycle action: every employee is an
// initial one, active and trained from the epoch with no end.

// stage is an employee's lifecycle stage. Hiring and termination (V0.2
// Slice 2) bring the others; only active exists until then.
type stage uint8

const (
	stageRecruiting stage = iota
	stageOnboarding
	stageActive
	stageNotice
	stageTerminated
)

var stageNames = [...]string{stageRecruiting: "recruiting", stageOnboarding: "onboarding",
	stageActive: "active", stageNotice: "notice", stageTerminated: "terminated"}

// noEnd marks open-ended employment.
const noEnd int64 = 1 << 62

type employment struct {
	ID    string
	Stage stage
	// From is the first employed day index; Until the first day index no
	// longer employed (noEnd: open-ended). The contract accrues on
	// employed days only.
	From, Until int64
	Contract    Contract
	// The current week's guaranteed pay (§5.4), in cents: the guarantee
	// earned on the week's employed days so far (weekG), worked pay at the
	// regular rate (weekB: worked base, premium excluded), and the
	// provision posted for the shortfall max(0, weekG − weekB), as wages
	// and benefits. The provision is accrued payroll that is not yet final
	// (openGuarantee); the week closes on Sunday at accounting close.
	weekG, weekB            int64
	provWages, provBenefits int64
}

func (r *employment) employed(day int64) bool { return r.From <= day && day < r.Until }

// initStaff opens a record for every initial employee, in scenario order.
func (st *state) initStaff() {
	for i := range st.emps {
		c := st.emps[i].cfg
		st.staff = append(st.staff, employment{ID: c.ID, Stage: stageActive, Until: noEnd, Contract: c.Contract})
	}
}

// guaranteeClose runs at accounting close, after everyone has gone home
// and their worked pay is posted: each hourly employee's guarantee earns
// the day's share, the provision moves to the week's shortfall (a signed
// change; none when unchanged), and on Sunday the week closes — the
// provision becomes final ordinary payroll and the counters reset.
//
// A contract at or below the roster never posts: a day's guarantee is at
// most the pay for its original rostered time, which worked time (never
// less than the roster in V0.1) already pays.
func (st *state) guaranteeClose() {
	date := st.cal.Date(st.cal.StartOfDay(st.day))
	wd, week := date.Weekday(), st.cfg.RotaWeek(date)
	for i := range st.staff {
		r := &st.staff[i]
		c := st.emps[i].cfg
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
		if wd != simtime.Sunday {
			continue
		}
		if q > 0 {
			st.emit(st.now, api.Info, "finance", fmt.Sprintf("%s's guaranteed pay this week: %s for contracted time not worked (%s contract)",
				c.Name, money(q+qb), r.Contract.Weekly))
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

// severance is what terminating hourly employee i would owe beyond pay
// earned (§5.4): two contracted weeks at the base rate, plus benefits; no
// assumed overtime, no dependence on the roster. Salaried staff cannot be
// terminated in V0.2 (0).
func (st *state) severance(i int) int64 {
	c := st.emps[i].cfg
	if c.Pay != Hourly {
		return 0
	}
	w := 2 * (c.RateCents * int64(st.staff[i].Contract.Weekly) / 3600)
	return w + roundDiv(w*st.cfg.Finance.BenefitsBP, 10_000)
}

// checkStaff verifies the registry against the employees and the ledger.
func (st *state) checkStaff() error {
	if len(st.staff) != len(st.emps) {
		return fmt.Errorf("%d employment records for %d employees", len(st.staff), len(st.emps))
	}
	for i := range st.staff {
		r := &st.staff[i]
		switch {
		case r.ID != st.emps[i].cfg.ID:
			return fmt.Errorf("employment record %d is %s, employee %s", i, r.ID, st.emps[i].cfg.ID)
		case r.provWages < 0 || r.provBenefits < 0 || r.weekG < 0 || r.weekB < 0:
			return fmt.Errorf("%s: negative week pay counters", r.ID)
		case st.emps[i].cfg.Pay != Hourly && (r.Contract != Contract{} || r.weekG != 0 || r.provWages != 0):
			return fmt.Errorf("%s: salaried with a contract guarantee", r.ID)
		}
	}
	if open := st.openGuarantee(); open > -st.fin.led.bal[acctAccruedPayroll] {
		return fmt.Errorf("open guarantee %d exceeds accrued payroll %d", open, -st.fin.led.bal[acctAccruedPayroll])
	}
	return nil
}

// observeEmployment fills employee i's employment and contract.
func (st *state) observeEmployment(x *api.Employee, i int) {
	r := &st.staff[i]
	x.Employment = stageNames[r.Stage]
	x.ContractMin = int64(r.Contract.Weekly / simtime.Minute)
	x.ContractInferred = r.Contract.Inferred
	x.GuaranteeCents = r.provWages + r.provBenefits
	x.SeveranceCents = st.severance(i)
}
