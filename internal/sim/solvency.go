package sim

import (
	"fmt"
	"strings"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Solvency (V0.7b, docs/V0.7_SOLVENCY_DESIGN.md): a credit line with a
// fixed limit and simple daily interest, and a funding rule for
// discretionary actions. Only a scenario with finance.solvency has it,
// and only from the takeover on; without it nothing here runs, so a run
// encodes exactly as before.
//
// Cash leaves only at 00:00 (payroll, severance, payables). A disbursement
// that would leave cash below zero draws on the line, up to the limit;
// what the line cannot cover is the unfunded shortfall (financial
// distress), recorded, never hidden: the obligation is still paid. At
// 00:00, interest on yesterday's balance accrues to payables, and after
// the day's disbursements cash above zero repays the line.
//
// A discretionary commitment (hire, salary increase, teleradiology
// contract, paid extension) is accepted only if the
// batch's commitments fit in the funding headroom: cash plus undrawn
// credit, less current liabilities and the commitments already reserved.
// Preview and application call the same check on the same state.

// creditState is the solvency state: totals for the reconciliation and
// the reserved commitments. Nil until the first draw, charge or
// reservation (the state digest encodes it only when non-nil).
type creditState struct {
	Draws, Repaid, Interest int64
	Reserves                []reserve
}

// reserve is an accepted commitment held against the funding headroom,
// released evenly over [From, From+Days) as its cost accrues.
type reserve struct {
	Amount, From, Days int64
	Action             int // the action record that made it
}

// remaining is what the reservation still holds on day d.
func (r *reserve) remaining(d int64) int64 {
	el := min(r.Days, max(0, d-r.From+1))
	return r.Amount - r.Amount*el/r.Days
}

// funding is a V0.2 item's commitment: check must be fundable now; reserve
// is held from From for Days (the recruiting fee is checked, not reserved:
// it is a payable at acceptance).
type funding struct {
	check, reserve, from, days int64
	what                       string
}

// fundCheck is a batch's funding test, for the preview.
type fundCheck struct {
	commit, headroom int64
}

// solvency is the credit line in force: the scenario's, from the takeover
// (nil: none).
func (st *state) solvency() *SolvencyConfig {
	if !st.fin.opened {
		return nil
	}
	return st.cfg.Finance.Solvency
}

func (st *state) credit() *creditState {
	if st.fin.credit == nil {
		st.fin.credit = &creditState{}
	}
	return st.fin.credit
}

func (st *state) creditDrawn() int64 { return -st.fin.led.bal[acctCreditLine] }

// currentLiabilities is what the practice owes now: accrued payroll,
// payables (interest included) and severance payable.
func (st *state) currentLiabilities() int64 {
	b := &st.fin.led.bal
	return -b[acctAccruedPayroll] - b[acctAP] - b[acctSeverancePayable]
}

// reserved is the commitments held on day d: accepted commitments not yet
// accrued, and approved paid extensions still to be worked.
func (st *state) reserved(d int64) int64 {
	var r int64
	if st.fin.credit != nil {
		for k := range st.fin.credit.Reserves {
			r += st.fin.credit.Reserves[k].remaining(d)
		}
	}
	for day, m := range st.ext {
		if day < st.day {
			continue
		}
		plan := st.plan
		if day != st.day {
			plan = st.planFor(day)
		}
		for i, x := range m {
			if x <= 0 || plan.ByEmp[i] < 0 || day == st.day && st.emps[i].done {
				continue
			}
			r += st.extensionCommitment(st.emps[i].terms, x)
		}
	}
	return r
}

// headroom is the funding headroom now: cash plus undrawn credit, less
// current liabilities and reserved commitments.
func (st *state) headroom(sv *SolvencyConfig) int64 {
	return st.fin.led.bal[acctCash] + sv.CreditLimitCents - st.creditDrawn() - st.currentLiabilities() - st.reserved(st.day)
}

// headroomText explains the headroom.
func (st *state) headroomText(sv *SolvencyConfig) string {
	return fmt.Sprintf("funding headroom %s = cash %s + undrawn credit %s − current liabilities %s − reserved commitments %s",
		money(st.headroom(sv)), money(st.fin.led.bal[acctCash]), money(sv.CreditLimitCents-st.creditDrawn()),
		money(st.currentLiabilities()), money(st.reserved(st.day)))
}

// extensionCommitment is what an approved extension of d commits: the
// time at the overtime rate with benefits for hourly staff, at the
// salary's hourly rate with benefits for a radiologist under salary_rate,
// otherwise nothing.
func (st *state) extensionCommitment(t *EmployeeConfig, d simtime.Duration) int64 {
	p := &st.cfg.Policies
	var w int64
	switch {
	case t.Pay == Hourly:
		w = t.RateCents * int64(d) * max(10_000, p.OvertimeBP) / 10_000 / 3600
	case st.cfg.Finance.RadOvertimePaid && t.Role == Radiologist && t.Pay == Salaried && p.WeeklyHours > 0:
		w = t.RateCents * int64(d) / (52 * int64(p.WeeklyHours))
	}
	return w + roundDiv(w*st.cfg.Finance.BenefitsBP, 10_000)
}

// checkExtensionFunds refuses a paid extension the headroom cannot fund.
func (st *state) checkExtensionFunds(t *EmployeeConfig, d simtime.Duration) (bool, string) {
	sv := st.solvency()
	if sv == nil {
		return true, ""
	}
	c := st.extensionCommitment(t, d)
	if c == 0 || c <= st.headroom(sv) {
		return true, ""
	}
	return false, fmt.Sprintf("Not funded: extending %s by %s commits %s (paid time with benefits), more than the %s.",
		t.Name, d, money(c), st.headroomText(sv))
}

// itemFunding is a valid V0.2 item's commitment (nil: none).
func (st *state) itemFunding(o *staffOutcome) *funding {
	switch {
	case o.hire != nil:
		h := o.hire
		exit := st.hireExit(h)
		return &funding{check: h.fee + exit, reserve: exit, from: h.from, days: NoticeDays + 1,
			what: "hire " + offerLabel(&h.terms) + ": recruiting fee and earliest lawful exit"}
	case o.alloc != nil:
		as := o.alloc
		c := st.salary30(as.emp, as.then) - st.salary30(as.emp, as.was)
		if c <= 0 {
			return nil
		}
		return &funding{check: c, reserve: c, from: as.from, days: 30,
			what: st.staffLabel(as.emp) + "'s salary increase for 30 days"}
	case o.tele != nil && o.tele.op == api.TeleOpStart:
		k := &o.tele.k
		end := max(st.day+teleCancelDays, k.from+teleMinTermDays)
		c := st.retainerBetween(k, max(st.day+1, k.from), end)
		if c <= 0 {
			return nil
		}
		return &funding{check: c, reserve: c, from: k.from, days: end - k.from,
			what: "teleradiology retainer to the earliest end"}
	}
	return nil
}

// checkFunding is the funding rule for a V0.2 batch whose items are all
// valid: their commitments, in input order, must fit in the headroom. It
// records each item's commitment, and the batch's test in p, and returns
// the first item that does not fit (-1: none).
func (st *state) checkFunding(p *proposal, out []staffOutcome) (int, staffOutcome) {
	sv := st.solvency()
	h := st.headroom(sv)
	p.fund = &fundCheck{headroom: h}
	fail := -1
	for k := range out {
		if out[k].code != api.ActionAccepted {
			continue
		}
		f := st.itemFunding(&out[k])
		if f == nil || f.check <= 0 {
			continue
		}
		out[k].fund = f
		p.fund.commit += f.check
		if fail < 0 && p.fund.commit > h {
			fail = k
		}
	}
	if fail < 0 {
		return -1, staffOutcome{}
	}
	var parts []string
	for k := 0; k <= fail; k++ {
		if f := out[k].fund; f != nil {
			parts = append(parts, fmt.Sprintf("%s %s", f.what, money(f.check)))
		}
	}
	return fail, staffOutcome{code: "insufficient_funds", msg: fmt.Sprintf("Not funded: this batch commits %s through action %d (%s), more than the %s. Cost-cutting actions and actions without a commitment remain available.",
		money(p.fund.commit), fail+1, strings.Join(parts, "; "), st.headroomText(sv))}
}

// reserveFunding holds an accepted item's commitment.
func (st *state) reserveFunding(f *funding, action int) {
	if f == nil || f.reserve <= 0 || f.days <= 0 {
		return
	}
	cs := st.credit()
	cs.Reserves = append(cs.Reserves, reserve{Amount: f.reserve, From: f.from, Days: f.days, Action: action})
}

// chargeInterest accrues, at 00:00, a day's interest on the balance drawn
// through the day before, to payables.
func (st *state) chargeInterest(now simtime.Time, day int64) {
	sv := st.solvency()
	if sv == nil || sv.CreditRateBP == 0 {
		return
	}
	d := st.creditDrawn()
	if d <= 0 {
		return
	}
	amt := roundDiv(d*sv.CreditRateBP, 10_000*365)
	if amt <= 0 {
		return
	}
	st.fin.led.mustPost(now, entryKey{kind: entInterest, a: day}, line{acctInterest, -1, amt, -1}, line{acctAP, -1, -amt, -1})
	st.fin.apAccrued += amt
	st.credit().Interest += amt
}

// drawCredit runs after a disbursement of amt (what): if cash is below
// zero, it draws on the line up to the limit (key b distinguishes the
// day's disbursements); what the line cannot cover is recorded as
// financial distress.
func (st *state) drawCredit(now simtime.Time, b int64, what string, amt int64) {
	sv := st.solvency()
	led := &st.fin.led
	if sv == nil || led.bal[acctCash] >= 0 {
		return
	}
	if d := min(-led.bal[acctCash], sv.CreditLimitCents-st.creditDrawn()); d > 0 {
		led.mustPost(now, entryKey{kind: entCreditDraw, a: st.day, b: b}, line{acctCash, -1, d, -1}, line{acctCreditLine, -1, -d, -1})
		st.credit().Draws += d
		st.emit(now, api.Info, "finance", fmt.Sprintf("Credit line drawn %s for %s (drawn %s of %s)", money(d), what,
			money(st.creditDrawn()), money(sv.CreditLimitCents)))
	}
	if c := led.bal[acctCash]; c < 0 {
		line := fmt.Sprintf("credit line fully drawn (%s)", money(sv.CreditLimitCents))
		if sv.CreditLimitCents == 0 {
			line = "no credit line"
		}
		st.emit(now, api.Warn, "finance", fmt.Sprintf("Financial distress: %s %s not fully funded: %s; unfunded %s",
			what, money(amt), line, money(-c)))
	}
}

// repayCredit runs at 00:00 after the day's disbursements: cash above zero
// repays the line, up to the balance drawn.
func (st *state) repayCredit(now simtime.Time, day int64) {
	sv := st.solvency()
	if sv == nil {
		return
	}
	r := min(st.fin.led.bal[acctCash], st.creditDrawn())
	if r <= 0 {
		return
	}
	st.fin.led.mustPost(now, entryKey{kind: entCreditRepay, a: day}, line{acctCreditLine, -1, r, -1}, line{acctCash, -1, -r, -1})
	st.credit().Repaid += r
	st.noteCash(now)
	st.emit(now, api.Info, "finance", fmt.Sprintf("Credit line repaid %s (drawn %s of %s)", money(r), money(st.creditDrawn()),
		money(sv.CreditLimitCents)))
}

// checkCredit verifies the solvency invariants.
func (st *state) checkCredit() error {
	led := &st.fin.led
	cs := st.fin.credit
	if cs == nil {
		if led.bal[acctCreditLine] != 0 || led.bal[acctInterest] != 0 {
			return fmt.Errorf("credit line %d, interest %d, without solvency state", led.bal[acctCreditLine], led.bal[acctInterest])
		}
		return nil
	}
	sv := st.cfg.Finance.Solvency
	switch d := st.creditDrawn(); {
	case sv == nil:
		return fmt.Errorf("solvency state without a credit line")
	case d != cs.Draws-cs.Repaid:
		return fmt.Errorf("credit drawn %d, draws − repayments %d", d, cs.Draws-cs.Repaid)
	case d < 0 || d > sv.CreditLimitCents:
		return fmt.Errorf("credit drawn %d outside 0–%d", d, sv.CreditLimitCents)
	case led.bal[acctInterest] != cs.Interest:
		return fmt.Errorf("interest expense %d, charged %d", led.bal[acctInterest], cs.Interest)
	}
	return nil
}

// solvencyView is the observation's solvency status (nil: none).
func (st *state) solvencyView() *api.Solvency {
	sv := st.solvency()
	if sv == nil {
		return nil
	}
	var interest int64
	if st.fin.credit != nil {
		interest = st.fin.credit.Interest
	}
	return &api.Solvency{CreditLimitCents: sv.CreditLimitCents, CreditRateBP: sv.CreditRateBP, CreditDrawnCents: st.creditDrawn(),
		CreditAvailableCents: sv.CreditLimitCents - st.creditDrawn(), CurrentLiabilitiesCents: st.currentLiabilities(),
		ReservedCents: st.reserved(st.day), FundingHeadroomCents: st.headroom(sv), InterestChargedCents: interest}
}
