package sim

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"

	"github.com/georgezero/dreamteam-go/internal/eval"
	"github.com/georgezero/dreamteam-go/internal/simtime"
	"github.com/georgezero/dreamteam-go/internal/version"
)

// EvalRecorder observes a run for the frozen V0.2 evaluation
// (internal/eval, docs/evaluation): daily samples and horizon captures,
// then the record. It only reads state, so a recorded run is the same run.
type EvalRecorder struct {
	w         *World
	rec       eval.Record
	studies   []int // inherited (unfinished at the takeover) and later studies, by index
	inhOrders []int // orders open at the takeover
	nOrders0  int   // orders existing at the takeover
	retainer0 int64
	reads0    int64
	paid0     int64
	outcomes  [][]string
}

// NewEvalRecorder starts recording at the takeover instant; the world must
// be exactly there, with an open-ended evaluation window.
func (w *World) NewEvalRecorder(plan, planSHA256, fingerprint string, seed int64) *EvalRecorder {
	st := w.st
	if !st.eval.on || st.eval.endAt != 0 || w.eng.Now() != w.StartTime() {
		panic("sim: the evaluation recorder starts at the takeover of an open-ended window")
	}
	r := &EvalRecorder{w: w, nOrders0: len(st.dem.orders)}
	r.rec = eval.Record{Schema: eval.RecordSchema, Plan: plan, PlanSHA256: planSHA256, Seed: seed, Engine: version.Number,
		ScenarioFingerprint: fingerprint, TATTargetMin: map[string]int64{}, TakeoverSec: int64(w.eng.Now()),
		TakeoverDate: st.cal.Date(w.eng.Now()).String()}
	for p := Routine; p <= Urgent; p++ {
		r.rec.TATTargetMin[p.String()] = int64(st.cfg.Policies.TATTarget[p] / 60)
	}
	for si := range st.dem.studies {
		if st.dem.studies[si].state != studyFinal {
			r.studies = append(r.studies, si)
		}
	}
	for oi := range st.dem.orders {
		switch st.dem.orders[oi].state {
		case orderBooked, orderRecall, orderVisit:
			r.inhOrders = append(r.inhOrders, oi)
		}
	}
	r.retainer0 = st.fin.led.bal[acctRemoteRetainer]
	r.reads0 = st.fin.led.bal[acctRemoteReads]
	if st.tele != nil {
		r.paid0 = st.tele.paid
	}
	return r
}

func (r *EvalRecorder) orderIndices() []int {
	out := slices.Clone(r.inhOrders)
	for oi := r.nOrders0; oi < len(r.w.st.dem.orders); oi++ {
		out = append(out, oi)
	}
	return out
}

// Sample records day d's backlog and money; call it at the takeover's
// clock time d days on.
func (r *EvalRecorder) Sample(d int) {
	st := r.w.st
	now := r.w.eng.Now()
	s := eval.Sample{Day: d, AtSec: int64(now), Date: st.cal.Date(now).String()}
	var ages []int64
	t0 := r.rec.TakeoverSec
	for si := range st.dem.studies {
		x := &st.dem.studies[si]
		if int64(x.acquiredAt) > t0 {
			s.AcquiredCum++
		}
		if x.state == studyFinal {
			if int64(x.finalAt) > t0 {
				s.FinalCum++
			}
			continue
		}
		switch x.state {
		case studyAcquired:
			s.AwaitingDoc++
		case studyUnread:
			s.Unread++
		case studyReading:
			if c, ok := x.by.remote(); ok {
				if st.tele.claims[c].readAt == 0 {
					s.RemoteHandoff++
				} else {
					s.RemoteReading++
				}
			} else {
				s.LocalReading++
			}
		}
		age := now.Sub(x.acquiredAt)
		ages = append(ages, int64(age)/60)
		if age > st.cfg.Policies.TATTarget[x.priority] {
			s.Overdue++
		}
	}
	if len(ages) > 0 {
		slices.Sort(ages)
		at := func(p int) int64 { return ages[max((p*len(ages)+99)/100, 1)-1] }
		s.AgeP50Min, s.AgeP90Min, s.AgeMaxMin = at(50), at(90), ages[len(ages)-1]
	}
	ev := r.w.Evaluation()
	s.ProfitCents, s.CashCents = ev.Money.OperatingProfitCents, st.fin.led.bal[acctCash]
	r.rec.Samples = append(r.rec.Samples, s)
}

// Capture records the horizon instant name (T30, T90, T180, tailEnd, end).
func (r *EvalRecorder) Capture(name string) {
	st := r.w.st
	now := r.w.eng.Now()
	var outs []string
	for _, oi := range r.orderIndices() {
		o := &st.dem.orders[oi]
		var out string
		switch o.state {
		case orderImaged, orderReported:
			out = eval.OutServed
		case orderLost, orderDisplacedLost:
			out = eval.OutLost
		case orderLeft:
			out = eval.OutLost
			if o.walkIn {
				out = eval.OutLeft
			}
		case orderCancelled, orderNoShow:
			out = eval.OutPatient
		default:
			out = eval.OutOpen
		}
		outs = append(outs, out)
	}
	r.outcomes = append(r.outcomes, outs)
	ev := r.w.Evaluation()
	led := &st.fin.led
	m := eval.Money{NetRevenueCents: ev.Money.NetRevenueCents, OperatingExpenseCents: ev.Money.OperatingExpenseCents,
		OperatingProfitCents: ev.Money.OperatingProfitCents, CashCents: led.bal[acctCash], MinCashCents: ev.Money.MinCashCents,
		ReceivablesCents: led.bal[acctAR], PayablesCents: -led.bal[acctAP], AccruedPayrollCents: -led.bal[acctAccruedPayroll],
		SeverancePayableCents: -led.bal[acctSeverancePayable], CashBridgeReconciles: ev.Money.Reconciled,
		RemoteRetainerCents: led.bal[acctRemoteRetainer] - r.retainer0, RemoteReadsCents: led.bal[acctRemoteReads] - r.reads0}
	if st.fin.minCash == ev.Money.MinCashCents {
		m.MinCashSec = int64(st.fin.minCashAt)
	}
	m.EarnedUnpostedWagesCents = st.earnedUnposted(now)
	if t := st.tele; t != nil {
		m.RemoteInFlightCents, m.VendorPaidCents = st.inFlight(), t.paid-r.paid0
		today := st.cal.DayIndex(now)
		for c := range t.contracts {
			k := &t.contracts[c]
			end := k.until
			if end < 0 {
				end = max(today+teleCancelDays, k.from+teleMinTermDays)
			}
			m.ContractMinimumCents += st.retainerBetween(k, max(today+1, k.from), end)
		}
	}
	for si := range st.dem.studies {
		s := &st.dem.studies[si]
		if s.state == studyUnread || s.state == studyReading || s.state == studyAcquired {
			m.UnbilledCompletedCents += st.cfg.Finance.ExpectedNetCents(&st.cfg.Catalog[s.exam], st.dem.orders[s.order].payer)
		}
	}
	// V0.2 has no way to defer a due payment: payroll, payables, severance
	// and vendor bills are paid when due, into negative cash if need be
	// (the shortfall), so nothing is ever past due.
	m.PastDueCents = 0
	r.rec.Captures = append(r.rec.Captures, eval.Capture{Name: name, AtSec: int64(now), Money: m})
}

// earnedUnposted is the hourly pay (with benefits) of shifts in progress,
// earned but posted only when the employee goes home.
func (st *state) earnedUnposted(now simtime.Time) int64 {
	var total int64
	p := &st.cfg.Policies
	for i := range st.emps {
		emp := &st.emps[i]
		if emp.cfg.Pay != Hourly || emp.shift < 0 || emp.done || emp.startAt == 0 || emp.startAt > now {
			continue
		}
		if s := emp.status; s != available && s != busy {
			continue
		}
		paid := now.Sub(emp.startAt) - emp.breakTaken
		if paid <= 0 {
			continue
		}
		w, _ := HourlyPay(emp.cfg.RateCents, emp.weekPaid, paid, p.WeeklyHours, p.OvertimeBP)
		total += w + roundDiv(w*st.cfg.Finance.BenefitsBP, 10_000)
	}
	return total
}

// Finish completes the record at the end of observation (after the
// "end" capture).
func (r *EvalRecorder) Finish(lastActionSec, latestEffectSec int64, teleOnlyOf string) eval.Record {
	st := r.w.st
	rec := r.rec
	rec.LastActionSec, rec.LatestEffectSec, rec.TeleOnlyOf = lastActionSec, latestEffectSec, teleOnlyOf
	rec.ObservedUntilSec = int64(r.w.eng.Now())
	idx := r.orderIndices()
	firstAppt := map[int]int64{}
	for ai := range st.dem.appts {
		o := st.dem.appts[ai].order
		if _, ok := firstAppt[o]; !ok {
			firstAppt[o] = int64(st.apptStart(ai))
		}
	}
	disp := map[int]int{}
	for _, x := range st.dem.displaced {
		disp[x.Order]++
	}
	for _, oi := range idx {
		o := &st.dem.orders[oi]
		wanted := int64(st.cal.StartOfDay(o.earliest))
		if o.walkIn || wanted < int64(o.received) {
			wanted = int64(o.received)
		}
		rec.Orders = append(rec.Orders, eval.Order{ID: orderID(oi), Priority: o.priority.String(), Modality: st.cfg.Catalog[o.exam].Modality,
			WalkIn: o.walkIn, ReceivedSec: int64(o.received), WantedSec: wanted, FirstApptSec: firstAppt[oi], Displacements: disp[oi]})
	}
	for k, outs := range r.outcomes {
		for len(outs) < len(idx) {
			outs = append(outs, eval.OutNotDisclosed)
		}
		rec.Captures[k].OrderOutcomes = outs
	}
	studies := slices.Clone(r.studies)
	for si := range st.dem.studies {
		if int64(st.dem.studies[si].acquiredAt) > rec.TakeoverSec {
			studies = append(studies, si)
		}
	}
	for _, si := range studies {
		s := &st.dem.studies[si]
		x := eval.Study{ID: studyID(si), Priority: s.priority.String(), Modality: st.cfg.Catalog[s.exam].Modality,
			AcquiredSec: int64(s.acquiredAt), EnteredSec: int64(s.enteredAt), ClaimedSec: int64(s.claimedAt)}
		if s.state == studyFinal {
			x.FinalSec = int64(s.finalAt)
		}
		if c, ok := s.by.remote(); ok {
			cl := &st.tele.claims[c]
			x.Remote, x.ReadStartSec, x.FeeCents = true, int64(cl.readAt), cl.fee+cl.surcharge
			x.ClaimedSec = int64(cl.claimedAt)
		} else if s.state == studyReading || s.state == studyFinal {
			x.ReadStartSec = int64(s.claimedAt)
		}
		rec.Studies = append(rec.Studies, x)
	}
	h := sha256.New()
	for _, a := range r.w.Appointments() {
		fmt.Fprintf(h, "%s %s %d %d %d %d %s %d\n", a.ID, a.Order, a.Room, a.Day, a.Span.From, a.Span.To, a.State, a.BookedAt)
	}
	rec.AppointmentsSHA256 = hex.EncodeToString(h.Sum(nil))
	return rec
}
