package sim

import (
	"fmt"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/rng"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Finance (SIMULATION_DESIGN §8): revenue at report final, claims, payroll
// and operating costs, all posted to the double-entry ledger.
//
// Accrual for profit, cash for liquidity, both from the one ledger:
// revenue is earned (Dr A/R / Cr revenue at expected net) when the report
// is finalized, and cash moves only when a payment arrives or a bill or
// payroll is paid. Payer behavior is drawn by the order's origin key
// (domains "claim" and "claim.lag"), so outcomes and their delays never
// depend on processing order, speed, staffing or worklist order; only the
// date the clock starts (report final) depends on operations.

// FinanceConfig holds the financial assumptions (§8.2–8.3, §6.1).
type FinanceConfig struct {
	Payers     []PayerConfig
	PayerMixBP []uint64 // parallel to Payers; sums to 10,000
	// ChargeLagDays: a claim is submitted this many business days
	// (Mon–Fri) after the report is final, inclusive range.
	ChargeLagDays [2]int64
	// ReworkDays: a denied claim's final outcome (recovered or written
	// off) comes this many days after the denial, inclusive range.
	ReworkDays    [2]int64
	ShortSpreadBP int64 // short-paid fraction is uniform within ±spread of the payer's mean
	// SubmitClock is when the day's claims go out; PaymentClock when
	// remittances (payments, denials, write-offs) post. Business days only.
	SubmitClock, PaymentClock simtime.Duration
	BenefitsBP                int64 // benefits and payroll tax, share of wages
	BillingFeeBP              int64 // billing service, share of cash collected
	PerStudyITCents           int64 // PACS/RIS charge per study, at acquisition
	// Payroll is paid every PayrollEveryDays from PayrollAnchor (a payday).
	PayrollAnchor    simtime.Date
	PayrollEveryDays int64
	Fixed            []FixedCost
}

// PayerConfig is one payer class and its outcome model (§8.2).
// Outcomes are mutually exclusive: denied (DeniedPPM, of which
// RecoveredPPM are paid in full on rework), short-paid (ShortPPM, paying
// ShortMeanBP of allowed on average) or paid in full.
type PayerConfig struct {
	ID, Name     string
	SelfPay      bool // the patient pays: no rework; non-payment is written off, not a denial
	MultiplierBP int64
	DelayDays    [2]int64 // submission to remittance, calendar days
	DeniedPPM    rng.PPM
	RecoveredPPM rng.PPM
	ShortPPM     rng.PPM
	ShortMeanBP  int64
}

// NCRPPM is the payer's expected lifetime collected fraction of the
// allowed amount, (1−d−s) + s·m + d·r, in parts per million.
func (p *PayerConfig) NCRPPM() int64 {
	d, s, r := int64(p.DeniedPPM), int64(p.ShortPPM), int64(p.RecoveredPPM)
	return (1_000_000 - d - s) + s*p.ShortMeanBP/10_000 + d*r/1_000_000
}

// FixedCost is a monthly cost accrued daily and paid on the 1st (§8.3).
type FixedCost struct {
	ID, Name string
	Account  string // "facility", "equipment" or "it"
	Modality string // optional: the modality the cost belongs to
	Monthly  int64  // cents per month
}

// FixedAccounts are the accounts a fixed cost may post to.
var FixedAccounts = []string{"facility", "equipment", "it"}

func fixedAccount(id string) account {
	switch id {
	case "equipment":
		return acctEquipment
	case "it":
		return acctIT
	}
	return acctFacility
}

// roundDiv divides a non-negative numerator, rounding half up.
func roundDiv(n, d int64) int64 { return (n + d/2) / d }

// AllowedCents is the allowed amount for an exam: base rate × payer multiplier.
func (f *FinanceConfig) AllowedCents(x *ExamConfig, payer int) int64 {
	return roundDiv(x.BaseRateCents*f.Payers[payer].MultiplierBP, 10_000)
}

// ExpectedNetCents is the revenue recognized at report final: allowed × NCR.
func (f *FinanceConfig) ExpectedNetCents(x *ExamConfig, payer int) int64 {
	return roundDiv(f.AllowedCents(x, payer)*f.Payers[payer].NCRPPM(), 1_000_000)
}

// DailyAccrual is day k's (1-based) share of a monthly amount in a month of
// n days: cumulative rounding, so a month accrues exactly the amount.
func DailyAccrual(monthly int64, k, n int) int64 {
	return monthly*int64(k)/int64(n) - monthly*int64(k-1)/int64(n)
}

// SalaryAccrual is day d's share of an annual amount: cumulative rounding
// over a 365-day year.
func SalaryAccrual(annual, d int64) int64 {
	return annual*(d+1)/365 - annual*d/365
}

// --- claims -------------------------------------------------------------

type claimState uint8

const (
	claimPending     claimState = iota // awaiting submission (charge lag)
	claimSubmitted                     // with the payer (or a statement with the patient)
	claimDenied                        // denied; awaiting resubmission
	claimResubmitted                   // reworked and resubmitted
	claimPaid                          // closed: paid in full (incl. recovered on rework)
	claimShortPaid                     // closed: paid less than allowed
	claimWrittenOff                    // closed: written off (denial lost, or self-pay unpaid)
)

var claimStateNames = [...]string{claimPending: api.ClaimPending, claimSubmitted: api.ClaimSubmitted,
	claimDenied: api.ClaimDenied, claimResubmitted: api.ClaimResubmitted, claimPaid: api.ClaimPaid,
	claimShortPaid: api.ClaimShortPaid, claimWrittenOff: api.ClaimWrittenOff}

func (s claimState) closed() bool { return s >= claimPaid }

type claim struct {
	study, order, payer int
	mod                 int8
	allowed, expected   int64
	paid                int64 // cash received so far
	state               claimState
	finalAt             simtime.Time
	submitDue           int64 // day index the (re)submission is due
	submittedAt         simtime.Time
	deniedAt            simtime.Time
	resubmittedAt       simtime.Time
	closedAt            simtime.Time
	denied              bool // denied at first adjudication (observable once it happens)
	payments            int64
	// draw is the latent payer behavior, drawn at submission and kept to
	// remittance (never observed; claimDraw recomputes the same value).
	draw claimDraw
}

func claimID(i int) string { return pubID('C', int64(i+1), 5) }

// Claim outcomes (latent until adjudication).
const (
	outPaid = iota
	outShort
	outRecovered
	outWrittenOff
)

// claimDraw is a claim's latent payer behavior, a pure function of the
// order's origin key: lane 0 outcome, lane 1 adjudication delay, lane 2
// short-paid fraction, lane 3 rework delay (§4.3).
type claimDraw struct {
	outcome    int
	delayDays  int64
	fracBP     int64
	reworkDays int64
}

func uniformIn(src rng.Source, r [2]int64) int64 {
	return r[0] + int64(rng.Uniform(src, uint64(r[1]-r[0]+1)))
}

func (st *state) claimDraw(key originKey, payer int) claimDraw {
	f := &st.cfg.Finance
	p := &f.Payers[payer]
	h := rng.Hash(st.seed, "claim", key.keys()...)
	d, s := uint64(p.DeniedPPM), uint64(p.ShortPPM)
	rec := d * uint64(p.RecoveredPPM) / 1_000_000
	out := rng.Weighted(h.Lane(0), []uint64{1_000_000 - d - s, s, rec, d - rec})
	frac := p.ShortMeanBP - f.ShortSpreadBP + int64(rng.Uniform(h.Lane(2), uint64(2*f.ShortSpreadBP+1)))
	return claimDraw{outcome: out, delayDays: uniformIn(h.Lane(1), p.DelayDays),
		fracBP: max(0, min(10_000, frac)), reworkDays: uniformIn(h.Lane(3), f.ReworkDays)}
}

func businessDay(d simtime.Date) bool {
	return d.Weekday() != simtime.Saturday && d.Weekday() != simtime.Sunday
}

// addBusinessDays returns the n-th business day after day index day.
func (st *state) addBusinessDays(day, n int64) int64 {
	for n > 0 {
		day++
		if businessDay(st.cal.Date(st.cal.StartOfDay(day))) {
			n--
		}
	}
	return day
}

// onOrAfterBusiness returns day, or the next business day.
func (st *state) onOrAfterBusiness(day int64) int64 {
	for !businessDay(st.cal.Date(st.cal.StartOfDay(day))) {
		day++
	}
	return day
}

// modIndex maps a modality to the ledger's modality dimension.
func (st *state) modIndex(m string) int8 {
	for i, x := range st.mods {
		if x == m {
			return int8(i)
		}
	}
	return -1
}

// finDay counts one day's finance activity that is not money.
type finDay struct {
	finals      [maxMods]int // reports finalized (revenue recognized)
	adjudicated int          // first adjudications of insurance claims
	denied      int          // of which denied
}

type finState struct {
	led    ledger
	claims []claim
	days   []finDay
	// pending holds claims awaiting (re)submission, in claim order.
	pending []int
	// Independent totals for the reconciliation invariants.
	receipts, payrollPaid, apPaid, payrollAccrued, apAccrued int64
	// forceDenied, if set (tests only), makes matching claims denied and
	// written off: a change to a hidden latent outcome.
	forceDenied func(ci int) bool

	// The opening adjustment at the start time (opening.go): the amount
	// posted to cash against opening equity, and the state it was made in.
	opened  bool
	opening openingSnap
	// Financing shortfall since the start: the lowest cash balance and when,
	// and whether cash is below zero now (for the one-time alert per dip).
	minCash   int64
	minCashAt simtime.Time
	belowZero bool
}

func (st *state) finDay(day int64) *finDay {
	for int64(len(st.fin.days)) <= day {
		st.fin.days = append(st.fin.days, finDay{})
	}
	return &st.fin.days[day]
}

// initFinance sets up the ledger. Cash starts at zero at the epoch; the
// opening adjustment at the start time (opening.go) sets it to the
// scenario's opening cash (§8.4).
func (st *state) initFinance() {
	st.fin.led = newLedger(st.cal)
	st.mods = st.modalities()
	if len(st.mods) > maxMods {
		panic("sim: more modalities than the ledger dimension")
	}
}

// payerOf draws an order's payer class from its origin lane.
func (st *state) payerOf(src rng.Source) int {
	return rng.Weighted(src, st.cfg.Finance.PayerMixBP)
}

// financeDayStart runs at 00:00: payroll and payables due today are paid
// first (they cover what was accrued before today), then today's salaried
// and fixed accruals post, and the day's claim submission is queued.
func (st *state) financeDayStart(e *Engine, day int64) {
	f := &st.cfg.Finance
	now := e.Now()
	date := st.cal.Date(now)
	led := &st.fin.led
	st.finDay(day)
	if st.isPayday(date) {
		if amt := led.bal[acctAccruedPayroll]; amt < 0 {
			led.mustPost(now, entryKey{kind: entPayroll, a: day}, line{acctAccruedPayroll, -1, -amt, -1}, line{acctCash, -1, amt, -1})
			st.fin.payrollPaid += -amt
			st.noteCash(now)
			st.emit(now, api.Info, "finance", fmt.Sprintf("Payroll paid %s (wages and benefits accrued through %s)",
				money(-amt), st.cal.Date(now-1).String()))
		}
	}
	if date.Day == 1 {
		if amt := led.bal[acctAP]; amt < 0 {
			led.mustPost(now, entryKey{kind: entPayAP, a: day}, line{acctAP, -1, -amt, -1}, line{acctCash, -1, amt, -1})
			st.fin.apPaid += -amt
			st.noteCash(now)
			st.emit(now, api.Info, "finance", fmt.Sprintf("Bills paid %s (rent, leases, service, supplies, fees accrued through %s)",
				money(-amt), st.cal.Date(now-1).String()))
		}
	}
	for i := range st.emps {
		c := st.emps[i].cfg
		if c.Pay != Salaried {
			continue
		}
		annual := c.RateCents
		if c.Role == Radiologist {
			annual = annual * c.SiteAllocationBP / 10_000
		}
		if w := SalaryAccrual(annual, day); w > 0 {
			st.accruePay(now, entryKey{kind: entSalary, a: int64(i), b: day}, w)
		}
	}
	var lines []line
	var total int64
	for k, fc := range f.Fixed {
		amt := DailyAccrual(fc.Monthly, date.Day, date.DaysInMonth())
		if amt == 0 {
			continue
		}
		lines = append(lines, line{acct: fixedAccount(fc.Account), mod: st.modIndex(fc.Modality), amt: amt, item: int16(k)})
		total += amt
	}
	if total > 0 {
		lines = append(lines, line{acctAP, -1, -total, -1})
		led.mustPost(now, entryKey{kind: entFixed, a: day}, lines...)
		st.fin.apAccrued += total
	}
	if businessDay(date) {
		for _, ci := range st.fin.pending {
			if st.fin.claims[ci].submitDue <= day {
				e.Push(Event{At: st.cal.At(day, f.SubmitClock), Class: ClassFinancial, Kind: kindSubmitClaims})
				break
			}
		}
	}
}

func (st *state) isPayday(d simtime.Date) bool {
	f := &st.cfg.Finance
	n := d.DaysSince(f.PayrollAnchor)
	return ((n%f.PayrollEveryDays)+f.PayrollEveryDays)%f.PayrollEveryDays == 0
}

// nextPayday is the first payday after day index day.
func (st *state) nextPayday(day int64) int64 {
	for d := day + 1; ; d++ {
		if st.isPayday(st.cal.Date(st.cal.StartOfDay(d))) {
			return d
		}
	}
}

// accruePay posts wages w and benefits on them to accrued payroll.
func (st *state) accruePay(at simtime.Time, key entryKey, w int64) {
	b := roundDiv(w*st.cfg.Finance.BenefitsBP, 10_000)
	st.fin.led.mustPost(at, key, line{acctWages, -1, w, -1}, line{acctBenefits, -1, b, -1}, line{acctAccruedPayroll, -1, -(w + b), -1})
	st.fin.payrollAccrued += w + b
}

// HourlyPay prices a day's paid time for an hourly employee: regular rate
// until the week's paid time reaches weekly, then ×factorBP (§6.4).
// weekBefore is the paid time earlier in the week. It returns the wages
// and the premium (beyond-threshold) seconds.
func HourlyPay(rate int64, weekBefore, paid, weekly simtime.Duration, factorBP int64) (int64, simtime.Duration) {
	premium := min(paid, max(0, weekBefore+paid-weekly))
	regular := paid - premium
	return (rate*int64(regular) + rate*int64(premium)*factorBP/10_000) / 3600, premium
}

// payWages accrues an hourly employee's pay for today when they go home.
// It runs before weekPaid is updated, so weekPaid is the week before today.
func (st *state) payWages(i int, paid simtime.Duration) {
	emp := &st.emps[i]
	if emp.cfg.Pay != Hourly || paid <= 0 {
		return
	}
	p := &st.cfg.Policies
	w, _ := HourlyPay(emp.cfg.RateCents, emp.weekPaid, paid, p.WeeklyHours, p.OvertimeBP)
	if w > 0 {
		st.accruePay(st.now, entryKey{kind: entWages, a: int64(i), b: st.day}, w)
	}
}

// studyCost posts supplies (contrast included) and the per-study IT charge
// when imaging completes.
func (st *state) studyCost(si int) {
	s := &st.dem.studies[si]
	x := &st.cfg.Catalog[s.exam]
	m := st.modIndex(x.Modality)
	it := st.cfg.Finance.PerStudyITCents
	if x.SupplyCostCents+it == 0 {
		return
	}
	st.fin.led.mustPost(st.now, entryKey{kind: entStudyCost, a: int64(si)},
		line{acctSupplies, m, x.SupplyCostCents, -1}, line{acctIT, m, it, -1}, line{acctAP, -1, -(x.SupplyCostCents + it), -1})
	st.fin.apAccrued += x.SupplyCostCents + it
}

// recognize earns a finalized study's revenue at expected net and opens
// its claim. It is the only place revenue is posted.
func (st *state) recognize(si int) int64 {
	s := &st.dem.studies[si]
	o := &st.dem.orders[s.order]
	x := &st.cfg.Catalog[s.exam]
	f := &st.cfg.Finance
	ci := len(st.fin.claims)
	c := claim{study: si, order: s.order, payer: o.payer, mod: st.modIndex(x.Modality),
		allowed: f.AllowedCents(x, o.payer), expected: f.ExpectedNetCents(x, o.payer), finalAt: st.now}
	lag := uniformIn(st.hash("claim.lag", o.key.keys()...).Lane(0), f.ChargeLagDays)
	c.submitDue = st.addBusinessDays(st.day, lag)
	st.fin.claims = append(st.fin.claims, c)
	st.fin.pending = append(st.fin.pending, ci)
	s.claim = ci
	st.fin.led.mustPost(st.now, entryKey{kind: entRevenue, a: int64(si)},
		line{acctAR, c.mod, c.expected, -1}, line{acctRevenue, c.mod, -c.expected, -1})
	if c.mod >= 0 {
		st.finDay(st.day).finals[c.mod]++
	}
	return c.expected
}

// submitClaims sends every claim whose (re)submission is due.
func (st *state) submitClaims(e *Engine) {
	f := &st.cfg.Finance
	now := e.Now()
	var n, re int
	var sum int64
	keep := st.fin.pending[:0]
	for _, ci := range st.fin.pending {
		c := &st.fin.claims[ci]
		if c.submitDue > st.day {
			keep = append(keep, ci)
			continue
		}
		n++
		sum += c.expected - c.paid
		if c.state == claimDenied {
			re++
			c.state, c.resubmittedAt = claimResubmitted, now
			continue // its outcome event was queued at the denial
		}
		c.state, c.submittedAt = claimSubmitted, now
		c.draw = st.claimDraw(st.dem.orders[c.order].key, c.payer)
		day := st.onOrAfterBusiness(st.day + c.draw.delayDays)
		e.Push(Event{At: st.cal.At(day, f.PaymentClock), Class: ClassFinancial, Kind: kindAdjudicate, A: int64(ci)})
	}
	st.fin.pending = keep
	if n > 0 {
		text := fmt.Sprintf("Claims submitted: %d · %s expected", n, money(sum))
		if re > 0 {
			text += fmt.Sprintf(" (%d resubmitted after denial)", re)
		}
		st.emit(now, api.Info, "finance", text)
	}
}

// adjudicate applies the payer's remittance for claim ci.
func (st *state) adjudicate(e *Engine, ci int) {
	f := &st.cfg.Finance
	c := &st.fin.claims[ci]
	p := &f.Payers[c.payer]
	dr := c.draw
	if st.fin.forceDenied != nil && st.fin.forceDenied(ci) {
		dr.outcome = outWrittenOff
	}
	if !p.SelfPay {
		fd := st.finDay(st.day)
		fd.adjudicated++
		if dr.outcome == outRecovered || dr.outcome == outWrittenOff {
			fd.denied++
		}
	}
	switch dr.outcome {
	case outPaid:
		st.pay(ci, c.allowed)
		st.settle(ci, claimPaid)
		st.emit(e.Now(), api.Info, "finance", fmt.Sprintf("Payment received: %s (%s) %s", claimID(ci), p.Name, money(c.allowed)))
	case outShort:
		amt := roundDiv(c.allowed*dr.fracBP, 10_000)
		st.pay(ci, amt)
		st.settle(ci, claimShortPaid)
		st.emit(e.Now(), api.Info, "finance", fmt.Sprintf("Short payment: %s (%s) %s of %s allowed",
			claimID(ci), p.Name, money(amt), money(c.allowed)))
	default:
		if p.SelfPay {
			st.settle(ci, claimWrittenOff)
			st.emit(e.Now(), api.Warn, "finance", fmt.Sprintf("Self-pay balance %s unpaid · written off (%s expected)",
				claimID(ci), money(c.expected)))
			return
		}
		c.state, c.deniedAt, c.denied = claimDenied, e.Now(), true
		c.submitDue = st.addBusinessDays(st.day, 1)
		st.fin.pending = append(st.fin.pending, ci)
		day := st.onOrAfterBusiness(st.day + dr.reworkDays)
		e.Push(Event{At: st.cal.At(day, f.PaymentClock), Class: ClassFinancial, Kind: kindReworkOutcome, A: int64(ci)})
		st.emit(e.Now(), api.Warn, "finance", fmt.Sprintf("Claim %s denied (%s) · reworking and resubmitting", claimID(ci), p.Name))
	}
}

// reworkOutcome resolves a denied, resubmitted claim.
func (st *state) reworkOutcome(e *Engine, ci int) {
	c := &st.fin.claims[ci]
	p := &st.cfg.Finance.Payers[c.payer]
	dr := c.draw
	if (st.fin.forceDenied == nil || !st.fin.forceDenied(ci)) && dr.outcome == outRecovered {
		st.pay(ci, c.allowed)
		st.settle(ci, claimPaid)
		st.emit(e.Now(), api.Info, "finance", fmt.Sprintf("Denied claim %s recovered on rework (%s) %s", claimID(ci), p.Name, money(c.allowed)))
		return
	}
	st.settle(ci, claimWrittenOff)
	st.emit(e.Now(), api.Warn, "finance", fmt.Sprintf("Claim %s written off after rework (%s) · %s expected", claimID(ci), p.Name, money(c.expected)))
}

// pay posts a payment received and the billing service fee on it.
func (st *state) pay(ci int, amt int64) {
	if amt <= 0 {
		return
	}
	c := &st.fin.claims[ci]
	c.payments++
	c.paid += amt
	st.fin.receipts += amt
	fee := roundDiv(amt*st.cfg.Finance.BillingFeeBP, 10_000)
	lines := []line{{acctCash, -1, amt, -1}, {acctAR, c.mod, -amt, -1}}
	if fee > 0 {
		lines = append(lines, line{acctBilling, c.mod, fee, -1}, line{acctAP, -1, -fee, -1})
		st.fin.apAccrued += fee
	}
	st.fin.led.mustPost(st.now, entryKey{kind: entPayment, a: int64(ci), b: c.payments}, lines...)
	st.noteCash(st.now)
}

// settle closes claim ci: the residual between expected net and cash
// received goes to collection variance, leaving the claim's A/R at zero.
func (st *state) settle(ci int, final claimState) {
	c := &st.fin.claims[ci]
	c.state, c.closedAt = final, st.now
	if r := c.expected - c.paid; r != 0 {
		st.fin.led.mustPost(st.now, entryKey{kind: entSettle, a: int64(ci)}, line{acctVariance, c.mod, r, -1}, line{acctAR, c.mod, -r, -1})
	}
}

// checkFinance verifies the accounting invariants (§10, 13–16).
func (st *state) checkFinance() error {
	led := &st.fin.led
	if err := led.check(); err != nil {
		return err
	}
	var ar, revenue int64
	for ci := range st.fin.claims {
		c := &st.fin.claims[ci]
		revenue += c.expected
		if !c.state.closed() {
			ar += c.expected - c.paid
		}
		if st.dem.studies[c.study].state != studyFinal || st.dem.studies[c.study].claim != ci {
			return fmt.Errorf("%s exists for %s, which is not final", claimID(ci), studyID(c.study))
		}
	}
	if len(st.fin.claims) != len(st.dem.finals) {
		return fmt.Errorf("%d claims for %d final reports", len(st.fin.claims), len(st.dem.finals))
	}
	switch {
	case led.bal[acctAR] != ar:
		return fmt.Errorf("A/R %d, open claims owe %d", led.bal[acctAR], ar)
	case -led.bal[acctRevenue] != revenue:
		return fmt.Errorf("revenue %d, claims expected %d", -led.bal[acctRevenue], revenue)
	case led.bal[acctCash] != st.fin.opening.adjustment+st.fin.receipts-st.fin.payrollPaid-st.fin.apPaid:
		return fmt.Errorf("cash %d, opening adjustment + receipts − disbursements %d", led.bal[acctCash],
			st.fin.opening.adjustment+st.fin.receipts-st.fin.payrollPaid-st.fin.apPaid)
	case -led.bal[acctEquity] != st.fin.opening.adjustment:
		return fmt.Errorf("opening equity %d, adjustment %d", -led.bal[acctEquity], st.fin.opening.adjustment)
	case -led.bal[acctAccruedPayroll] != st.fin.payrollAccrued-st.fin.payrollPaid:
		return fmt.Errorf("accrued payroll %d, accrued − paid %d", -led.bal[acctAccruedPayroll], st.fin.payrollAccrued-st.fin.payrollPaid)
	case -led.bal[acctAP] != st.fin.apAccrued-st.fin.apPaid:
		return fmt.Errorf("payables %d, accrued − paid %d", -led.bal[acctAP], st.fin.apAccrued-st.fin.apPaid)
	}
	return nil
}

// money formats cents as whole dollars with thousands separators.
func money(c int64) string {
	neg := c < 0
	if neg {
		c = -c
	}
	d := (c + 50) / 100
	s := fmt.Sprint(d)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	if neg {
		return "−$" + s
	}
	return "$" + s
}
