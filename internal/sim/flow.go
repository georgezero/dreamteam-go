package sim

import (
	"fmt"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/rng"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Patient flow (SIMULATION_DESIGN §2.9–2.10, §5.6–5.8): scheduled patients
// arrive early, on time or late, or not at all; walk-ins arrive with an
// order but no appointment. An arrived patient is admitted (opens a visit)
// unless nobody can serve them safely today, in which case they are
// rescheduled on the spot. Admitted patients are registered and imaged,
// after scheduled close if need be (§6.9; dispatch in ops.go).
//
// Hidden until they happen: whether a patient will come, when, and how
// long a walk-in will wait. The arrival and grace events are queued at the
// start of the appointment's day (or at booking, for later the same day);
// only their effects are observable.

type visitState uint8

const (
	visitArrived     visitState = iota // admitted, waiting to register
	visitRegistering                   // at the desk
	visitWaiting                       // registered, waiting for the exam
	visitInExam                        // in the room
	visitDone                          // imaged and departed
	visitLeft                          // walk-in ran out of patience
	visitRescheduled                   // no safe coverage today; reason recorded
	numVisitStates
)

// onSite reports whether a visit is still in the building.
func (v *visit) onSite() bool { return v.state <= visitInExam }

type visit struct {
	order   int
	appt    int // -1 for a walk-in
	arrived simtime.Time
	// late is the arrival relative to the appointment start (negative is
	// early); zero for walk-ins.
	late       simtime.Duration
	afterGrace bool // arrived after the appointment was marked missed
	state      visitState
	endedAt    simtime.Time
	attempt    int64 // the order's attempt, for keyed exam and registration draws

	regBy, examBy      int // employees; -1 until then
	room               int // exam room; -1 until the exam starts
	regStart, regEnd   simtime.Time
	examStart, examEnd simtime.Time
	afterHours         bool   // exam started after scheduled close
	reason             string // why it was rescheduled
	supervisor         int    // radiologist supervising a contrast exam; -1 if none
	study              int    // the study acquired, once imaged; -1 before
}

// flowCounts are patient-flow and operations counters.
type flowCounts struct {
	arrived, early, late, afterGrace int // scheduled patients
	// Walk-ins (step 18.1): arrivals; admitted on the visit they came in
	// on; offered an appointment instead — declined by the policy, or
	// unservable today — and whether that offer booked; exams on the
	// walk-in visit itself, and later exams of walk-in orders on scheduled
	// visits (returns).
	walkIns, walkInsAdmitted                int
	walkInsDeclined, walkInsUnservable      int
	walkInsOfferBooked, walkInsOfferFailed  int
	walkInSameVisitExams, walkInReturnExams int
	// Booking events of walk-in-originated orders (any time: offers,
	// rebookings, cancellations), kept apart from the
	// referral demand counters.
	wiBooked, wiLost, wiCancelled    int
	missed, noShows, rescheduled     int
	left                             int
	registered, selfRegistered       int
	examsStarted, examsDone          int
	afterHours                       int // exams started after scheduled close
	noCoverage                       int // admitted patients rescheduled: no safe coverage
	heldOver, skippedBreaks          int
	otSec                            simtime.Duration
	otCost                           int64            // cents
	otApprovedSec                    simtime.Duration // of otSec, approved in advance (extend_shift)
	otApprovedCost                   int64
	acquired, reportsFinal, breaches int // studies; reports; TAT targets missed
}

func visitID(i int) string { return pubID('V', int64(i+1), 5) }

// startAppointments queues arrival and grace events for today's
// appointments, in room and slot order (day start, after booking).
func (st *state) startAppointments(e *Engine) {
	ds := st.dem.days[st.day]
	for r := range ds.rooms {
		for _, s := range ds.rooms[r] {
			if s.appt >= 0 {
				st.scheduleArrival(e, s.appt)
			}
		}
	}
	st.dem.dayStarted = true
}

// scheduleArrival draws, by origin key, whether and when the patient comes
// and queues the hidden events.
func (st *state) scheduleArrival(e *Engine, ai int) {
	a := &st.dem.appts[ai]
	if a.scheduled || a.state != apptBooked {
		return
	}
	a.scheduled = true
	o := &st.dem.orders[a.order]
	s := st.dem.days[a.day].rooms[a.room][a.slot]
	start := st.cal.At(a.day, s.From)
	b := &st.cfg.Behavior

	e.Push(Event{At: start.Add(st.cfg.Policies.Grace), Class: ClassDeadline, Kind: kindGrace, A: int64(ai)})

	att := st.hash("attempt", o.key.keys(a.attempt)...)
	lead := a.day - st.cal.DayIndex(a.bookedAt)
	ppm := b.NoShowPPM[st.cfg.Catalog[o.exam].Modality] +
		min(b.NoShowMaxLeadPPM, rng.PPM(lead/10)*b.NoShowPerTenDaysPPM)
	if rng.Bernoulli(att.Lane(0), ppm) || (st.forceNoShow != nil && st.forceNoShow(start)) {
		return // never comes; revealed at start + grace, final at close
	}
	weights := make([]uint64, len(b.Arrival))
	for i, c := range b.Arrival {
		weights[i] = c.WeightBP
	}
	cls := b.Arrival[rng.Weighted(st.hash("patient", o.key.keys()...).Lane(0), weights)]
	off := cls.From + simtime.Duration(rng.Uniform(att.Lane(1), uint64(cls.To-cls.From)+1))
	at := start.Add(off)
	if at <= e.Now() {
		at = e.Now().Add(1)
	}
	e.Push(Event{At: at, Class: ClassArrival, Kind: kindArrive, A: int64(ai)})
}

// grace marks an appointment missed when the patient has not arrived by
// start + grace. Missed is provisional: the patient may still turn up.
func (st *state) grace(e *Engine, ai int) {
	a := &st.dem.appts[ai]
	a.state, a.endedAt, a.missedAt = apptMissed, e.Now(), e.Now()
	st.dem.flowToday.missed++
	st.dem.flowTotal.missed++
	o := &st.dem.orders[a.order]
	s := st.dem.days[a.day].rooms[a.room][a.slot]
	st.emit(e.Now(), api.Info, "patients", fmt.Sprintf("%s %s %s missed: not here by %s (%s %s)",
		apptID(ai), patientID(o.patient), st.examLabel(o.exam), st.cal.Clock(e.Now()),
		simtime.FormatClock(s.From), "appointment"))
}

// arrive handles a scheduled patient walking in the door.
func (st *state) arrive(e *Engine, ai int) {
	a := &st.dem.appts[ai]
	o := &st.dem.orders[a.order]
	s := st.dem.days[a.day].rooms[a.room][a.slot]
	start := st.cal.At(a.day, s.From)
	late := e.Now().Sub(start)
	who := fmt.Sprintf("%s %s for %s", patientID(o.patient), st.examLabel(o.exam), simtime.FormatClock(s.From))

	if h := st.plan.Hours; h == nil || simtime.ClockOf(e.Now()) >= h.To {
		st.reschedule(e, ai, who+": arrived after closing")
		return
	}
	if ok, why := st.servable(o.exam, st.arrivalEarliest(ai), st.newPatientTask(o.exam, false)); !ok {
		st.reschedule(e, ai, who+": "+why)
		return
	}
	if a.state == apptMissed {
		if st.cfg.Policies.Late == "reschedule" {
			st.reschedule(e, ai, fmt.Sprintf("%s: arrived %s late, past the grace period", who, late))
			return
		}
		// fit_in: registered and waits for a gap, lowest scheduled priority.
		st.dem.flowToday.afterGrace++
		st.dem.flowTotal.afterGrace++
	}
	missed := a.state == apptMissed
	vi := st.openVisit(e, a.order, ai, late, a.attempt)
	st.dem.visits[vi].afterGrace = missed
	a.state, a.visit = apptArrived, vi
	for _, f := range []*flowCounts{&st.dem.flowToday, &st.dem.flowTotal} {
		f.arrived++
		switch {
		case late < -5*simtime.Minute:
			f.early++
		case late > 5*simtime.Minute:
			f.late++
		}
	}
	text := who + ": arrived " + punctuality(late)
	sev := api.Info
	if missed {
		text += ", after being marked missed; fitted in"
		sev = api.Warn
	}
	st.emit(e.Now(), sev, "patients", fmt.Sprintf("%s %s · waiting (%s)", visitID(vi), text, apptID(ai)))
}

// punctuality describes an arrival offset: "12m early", "on time", "9m late".
func punctuality(d simtime.Duration) string {
	switch {
	case d < -5*simtime.Minute:
		return (-d).String() + " early"
	case d > 5*simtime.Minute:
		return d.String() + " late"
	}
	return "on time"
}

// openVisit creates a visit for an arriving patient. Visit IDs are
// allocated here, at disclosure.
func (st *state) openVisit(e *Engine, oi, ai int, late simtime.Duration, attempt int64) int {
	vi := len(st.dem.visits)
	st.dem.visits = append(st.dem.visits, visit{order: oi, appt: ai, arrived: e.Now(), late: late,
		attempt: attempt, regBy: -1, examBy: -1, room: -1, supervisor: -1, study: -1})
	st.dem.waiting = append(st.dem.waiting, vi)
	ds := st.dem.days[st.day]
	ds.visits = append(ds.visits, vi)
	o := &st.dem.orders[oi]
	o.state, o.visit = orderVisit, vi
	return vi
}

// reschedule sends a patient away with a new date (late policy, or the
// practice already closed): the appointment ends and the order is offered
// slots from tomorrow.
func (st *state) reschedule(e *Engine, ai int, why string) {
	a := &st.dem.appts[ai]
	a.state, a.endedAt = apptRescheduled, e.Now()
	st.dem.flowToday.rescheduled++
	st.dem.flowTotal.rescheduled++
	o := &st.dem.orders[a.order]
	o.attempt++
	st.offer(e, a.order, st.day+1, apptID(ai)+" "+why+"; rescheduled", "")
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// accountingClose finalizes today's missed appointments as no-shows
// (SIMULATION_DESIGN §3.4.2, §5.8). A no-show may ask for a new date; that
// request is disclosed the next morning.
func (st *state) accountingClose(e *Engine) {
	if len(st.dem.waiting) > 0 {
		fail("%d visits still on site at accounting close (first %s)", len(st.dem.waiting), visitID(st.dem.waiting[0]))
	}
	// The day's work ends here: shift caps fall by 23:59:59 (validated), so
	// nobody can still be busy. Anyone past shift end goes home now, so
	// their overtime and paid time belong to this day.
	for i := range st.emps {
		emp := &st.emps[i]
		switch {
		case emp.status == busy:
			fail("%s still busy at accounting close", emp.cfg.Name)
		case emp.status != offDuty && emp.over:
			st.goOff(e, i)
		case emp.status != offDuty:
			fail("%s still on regular duty at accounting close", emp.cfg.Name)
		}
	}
	st.guaranteeClose()
	n := 0
	for _, ai := range st.dem.days[st.day].appts {
		a := &st.dem.appts[ai]
		if a.state != apptMissed {
			continue
		}
		n++
		a.state, a.endedAt = apptNoShow, e.Now()
		o := &st.dem.orders[a.order]
		o.state, o.endedAt = orderNoShow, e.Now()
		st.dem.flowToday.noShows++
		st.dem.flowTotal.noShows++
		st.emit(e.Now(), api.Info, "patients", fmt.Sprintf("%s %s %s: no-show", apptID(ai), patientID(o.patient), st.examLabel(o.exam)))
		ex := st.hash("attempt.exam", o.key.keys(a.attempt)...)
		if rng.Bernoulli(ex.Lane(2), st.cfg.Behavior.NoShowRebookPPM) {
			e.Push(Event{At: st.cal.At(st.day+1, 9*simtime.Hour), Class: ClassArrival, Kind: kindNoShowRebook, A: int64(a.order)})
		}
	}
	st.dem.noShowsLastClose = n
	st.recordDay()
}

// noShowRebook: a no-show patient calls for a new date.
func (st *state) noShowRebook(e *Engine, oi int) {
	o := &st.dem.orders[oi]
	o.attempt++
	st.offer(e, oi, st.day, fmt.Sprintf("%s %s %s asks for a new date after a no-show",
		orderID(oi), patientID(o.patient), st.examLabel(o.exam)), "")
}

// scheduleWalkIns queues the day's walk-ins (day boundary). Walk-ins come
// only while the practice is open; their count, time and exam are keyed by
// (day, hour block, k), independent of scheduled demand.
func (st *state) scheduleWalkIns(e *Engine, day int64) {
	h := st.plan.Hours
	w := &st.cfg.WalkIns
	if h == nil || w.DailyMilli == 0 {
		return
	}
	wd := st.plan.Date.Weekday()
	for blk := int64(0); blk < 24; blk++ {
		rate := w.DailyMilli * w.HourBP[blk] / 10_000 * w.WeekdayBP[wd] / 10_000
		if rate <= 0 {
			continue
		}
		n := st.poissonFor(uint32(rate)).Sample(rng.Hash(st.seed, "walkin.count", day, blk))
		for k := 0; k < n; k++ {
			sec := rng.Uniform(st.hash("walkin", day, blk, int64(k)).Lane(0), 3600)
			clock := simtime.Duration(blk)*simtime.Hour + simtime.Duration(sec)
			if !h.Contains(clock) {
				continue
			}
			at := st.cal.At(day, clock)
			if st.dropOrder != nil && st.dropOrder(at) {
				continue
			}
			e.Push(Event{At: at, Class: ClassArrival, Kind: kindWalkIn, A: day, B: blk<<32 | int64(k)})
		}
	}
}

// walkIn discloses a walk-in: a new patient and order (same model as
// referred orders, with no referral source), then either a visit that
// waits or, when the modality cannot be done today or the policy says so,
// an appointment for later.
func (st *state) walkIn(e *Engine, day, blk, k int64) {
	key := originKey{day: day, src: -1, block: blk, k: k}
	w := st.hash("walkin", day, blk, k)
	wc := &st.cfg.WalkIns
	exam := wc.Exams[rng.Weighted(w.Lane(1), wc.ExamWeights)]
	pri := Priority(rng.Weighted(w.Lane(3), wc.PriorityBP[:]))
	tol := st.cfg.Behavior.ToleranceDays[pri].Sample(st.hash("patient", key.keys()...).Lane(1))

	st.dem.patients++
	oi := len(st.dem.orders)
	st.dem.orders = append(st.dem.orders, order{key: key, patient: st.dem.patients, source: -1, exam: exam,
		priority: pri, received: e.Now(), earliest: st.day, tolerance: tol, appt: -1, visit: -1, walkIn: true,
		payer: st.payerOf(w.Lane(2))})
	o := &st.dem.orders[oi]
	st.dem.days[st.day].walkIns = append(st.dem.days[st.day].walkIns, oi)
	st.dem.flowToday.walkIns++
	st.dem.flowTotal.walkIns++
	head := fmt.Sprintf("Walk-in %s %s", patientID(o.patient), st.examLabel(exam))

	ok, why := st.servable(exam, e.Now(), st.newPatientTask(exam, true))
	declined := false
	switch {
	case st.pol.WalkIn == "schedule_later":
		ok, why, declined = false, "walk-in policy: schedule later", true
	case st.pol.WalkIn == "accept_until" && simtime.ClockOf(e.Now()) >= st.pol.WalkInCutoff:
		ok, why, declined = false, "walk-in policy: no walk-ins after "+simtime.FormatClock(st.pol.WalkInCutoff), true
	}
	if !ok {
		booked := st.offer(e, oi, st.day+1, head+" ("+why+")", "")
		for _, f := range []*flowCounts{&st.dem.flowToday, &st.dem.flowTotal} {
			if declined {
				f.walkInsDeclined++
			} else {
				f.walkInsUnservable++
			}
			if booked {
				f.walkInsOfferBooked++
			} else {
				f.walkInsOfferFailed++
			}
		}
		return
	}
	st.dem.flowToday.walkInsAdmitted++
	st.dem.flowTotal.walkInsAdmitted++
	vi := st.openVisit(e, oi, -1, 0, 0)
	patience := simtime.Duration(st.cfg.Behavior.WalkInPatienceMin.Sample(
		st.hash("patient", key.keys()...).Lane(2))) * simtime.Minute
	e.Push(Event{At: e.Now().Add(patience), Class: ClassDeadline, Kind: kindPatience, A: int64(vi)})
	st.emit(e.Now(), api.Info, "patients", fmt.Sprintf("%s %s · waiting (%s)", visitID(vi), head, orderID(oi)))
}

// patience: a walk-in waiting (to register or for the exam) gives up and
// leaves. Patience is checked once, at its deadline: a walk-in who is being
// registered or imaged at that moment stays (the event is stale).
func (st *state) patience(e *Engine, vi int) {
	v := &st.dem.visits[vi]
	v.state, v.endedAt = visitLeft, e.Now()
	st.removeWaiting(vi)
	o := &st.dem.orders[v.order]
	o.state, o.endedAt = orderLeft, e.Now()
	st.dem.flowToday.left++
	st.dem.flowTotal.left++
	st.emit(e.Now(), api.Warn, "patients", fmt.Sprintf("%s Walk-in %s %s left without an exam after waiting %s",
		visitID(vi), patientID(o.patient), st.examLabel(o.exam), e.Now().Sub(v.arrived)))
}

func (st *state) removeWaiting(vi int) {
	w := st.dem.waiting
	for i, x := range w {
		if x == vi {
			st.dem.waiting = append(w[:i], w[i+1:]...)
			return
		}
	}
}

// checkFlow verifies visit invariants: one visit per arrived appointment,
// the on-site list = visits in the building, order and visit states agree,
// timestamps ordered, rescheduled visits carry a reason, and nobody is on
// site at accounting close.
func (st *state) checkFlow() error {
	d := &st.dem
	open := 0
	for vi := range d.visits {
		v := &d.visits[vi]
		o := &d.orders[v.order]
		if v.onSite() {
			open++
			if o.state != orderVisit || o.visit != vi {
				return fmt.Errorf("%s on site but order %s is %d", visitID(vi), orderID(v.order), o.state)
			}
		}
		switch {
		case v.state >= visitRegistering && v.state != visitLeft && v.state != visitRescheduled && v.regStart < v.arrived,
			v.state >= visitWaiting && v.state <= visitDone && v.regEnd < v.regStart,
			v.state >= visitInExam && v.state <= visitDone && v.examStart < v.regEnd,
			v.state == visitDone && v.examEnd < v.examStart:
			return fmt.Errorf("%s timestamps out of order", visitID(vi))
		case v.state == visitRescheduled && v.reason == "":
			return fmt.Errorf("%s rescheduled without a reason", visitID(vi))
		case v.state == visitDone && o.state != orderImaged && o.state != orderReported:
			return fmt.Errorf("%s done but order %s is %d", visitID(vi), orderID(v.order), o.state)
		}
		if v.appt >= 0 {
			a := &d.appts[v.appt]
			if a.visit != vi || a.state != apptArrived {
				return fmt.Errorf("%s: appointment %s is %s", visitID(vi), apptID(v.appt), apptStateNames[a.state])
			}
		} else if !o.walkIn {
			return fmt.Errorf("%s has no appointment and is not a walk-in", visitID(vi))
		}
	}
	if open != len(d.waiting) {
		return fmt.Errorf("%d visits on site, on-site list %d", open, len(d.waiting))
	}
	for _, vi := range d.waiting {
		if !d.visits[vi].onSite() {
			return fmt.Errorf("%s on the on-site list but gone", visitID(vi))
		}
	}
	return nil
}

// newPatientTask is the longest a newly arriving patient can occupy a
// technologist (self-registration plus exam and documentation, p100).
func (st *state) newPatientTask(xi int, walkIn bool) simtime.Duration {
	b := &st.cfg.Behavior
	q := b.RegScheduled
	if walkIn {
		q = b.RegWalkIn
	}
	return st.cfg.Catalog[xi].MaxTask() + simtime.Duration(q.Max()*b.RegSelfBP/10_000)
}

// arrivalEarliest is the earliest an arriving scheduled patient's exam
// could start (due time when early starts are off, not before opening).
func (st *state) arrivalEarliest(ai int) simtime.Time {
	at := st.now
	if !st.cfg.Policies.StartEarly {
		at = max(at, st.apptStart(ai).Add(-st.cfg.Policies.EarlyStart))
	}
	if h := st.plan.Hours; h != nil && !st.open && !st.closing {
		at = max(at, st.cal.At(st.day, h.From))
	}
	return at
}
