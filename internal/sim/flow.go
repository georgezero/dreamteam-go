package sim

import (
	"fmt"
	"strings"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/rng"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Patient flow (SIMULATION_DESIGN §2.9–2.10, §5.6–5.8): scheduled patients
// arrive early, on time or late, or not at all; walk-ins arrive with an
// order but no appointment. An arrived patient opens a visit and waits.
// Registration and imaging come in step 8, so in V0.1 step 7 every visit
// waits until it ends: a walk-in may run out of patience, and patients
// still waiting when the practice closes are sent home unseen (no exam
// starts after closing, §6.5).
//
// Hidden until they happen: whether a patient will come, when, and how
// long a walk-in will wait. The arrival and grace events are queued at the
// start of the appointment's day (or at booking, for later the same day);
// only their effects are observable.

type visitState uint8

const (
	visitWaiting visitState = iota // arrived; not yet registered or imaged
	visitLeft                      // walk-in ran out of patience
	visitUnseen                    // still waiting when the practice closed
	numVisitStates
)

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
}

// flowCounts are patient-flow event counters.
type flowCounts struct {
	arrived, early, late, afterGrace int // scheduled patients
	walkIns, walkInsBooked           int // walk-ins; those sent away with an appointment
	missed, noShows, rescheduled     int
	left, unseen                     int
}

func visitID(i int) string { return fmt.Sprintf("V%05d", i+1) }

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

	att := rng.Hash(st.seed, "attempt", o.key.keys(a.attempt)...)
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
	cls := b.Arrival[rng.Weighted(rng.Hash(st.seed, "patient", o.key.keys()...).Lane(0), weights)]
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
	vi := st.openVisit(e, a.order, ai, late)
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
func (st *state) openVisit(e *Engine, oi, ai int, late simtime.Duration) int {
	vi := len(st.dem.visits)
	st.dem.visits = append(st.dem.visits, visit{order: oi, appt: ai, arrived: e.Now(), late: late})
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

// closeVisits sends home everyone still waiting at closing time: no exam
// starts after the practice closes.
func (st *state) closeVisits(e *Engine) {
	if len(st.dem.waiting) == 0 {
		return
	}
	var ids []string
	for _, vi := range st.dem.waiting {
		v := &st.dem.visits[vi]
		v.state, v.endedAt = visitUnseen, e.Now()
		o := &st.dem.orders[v.order]
		o.state, o.endedAt = orderUnseen, e.Now()
		st.dem.flowToday.unseen++
		st.dem.flowTotal.unseen++
		ids = append(ids, patientID(o.patient))
	}
	st.dem.waiting = st.dem.waiting[:0]
	st.emit(e.Now(), api.Warn, "patients", fmt.Sprintf("%d patient%s still waiting sent home unseen: %s",
		len(ids), plural(len(ids)), strings.Join(ids, " ")))
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
		ex := rng.Hash(st.seed, "attempt.exam", o.key.keys(a.attempt)...)
		if rng.Bernoulli(ex.Lane(2), st.cfg.Behavior.NoShowRebookPPM) {
			e.Push(Event{At: st.cal.At(st.day+1, 9*simtime.Hour), Class: ClassArrival, Kind: kindNoShowRebook, A: int64(a.order)})
		}
	}
	st.dem.noShowsLastClose = n
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
			sec := rng.Uniform(rng.Hash(st.seed, "walkin", day, blk, int64(k)).Lane(0), 3600)
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
	w := rng.Hash(st.seed, "walkin", day, blk, k)
	wc := &st.cfg.WalkIns
	exam := wc.Exams[rng.Weighted(w.Lane(1), wc.ExamWeights)]
	// lane 2 is reserved for the payer (step 10).
	pri := Priority(rng.Weighted(w.Lane(3), wc.PriorityBP[:]))
	tol := st.cfg.Behavior.ToleranceDays[pri].Sample(rng.Hash(st.seed, "patient", key.keys()...).Lane(1))

	st.dem.patients++
	oi := len(st.dem.orders)
	st.dem.orders = append(st.dem.orders, order{key: key, patient: st.dem.patients, source: -1, exam: exam,
		priority: pri, received: e.Now(), earliest: st.day, tolerance: tol, appt: -1, visit: -1, walkIn: true})
	o := &st.dem.orders[oi]
	st.dem.days[st.day].walkIns = append(st.dem.days[st.day].walkIns, oi)
	st.dem.flowToday.walkIns++
	st.dem.flowTotal.walkIns++
	head := fmt.Sprintf("Walk-in %s %s", patientID(o.patient), st.examLabel(exam))

	if st.cfg.Policies.WalkIn == "schedule_later" || !st.staffedLaterToday(st.cfg.Catalog[exam].Modality) {
		st.dem.flowToday.walkInsBooked++
		st.dem.flowTotal.walkInsBooked++
		why := "no " + st.cfg.Catalog[exam].Modality + " technologist for the rest of today"
		if st.cfg.Policies.WalkIn == "schedule_later" {
			why = "walk-in policy: schedule later"
		}
		st.offer(e, oi, st.day+1, head+" ("+why+")", "")
		return
	}
	vi := st.openVisit(e, oi, -1, 0)
	patience := simtime.Duration(st.cfg.Behavior.WalkInPatienceMin.Sample(
		rng.Hash(st.seed, "patient", key.keys()...).Lane(2))) * simtime.Minute
	e.Push(Event{At: e.Now().Add(patience), Class: ClassDeadline, Kind: kindPatience, A: int64(vi)})
	st.emit(e.Now(), api.Info, "patients", fmt.Sprintf("%s %s · waiting (%s)", visitID(vi), head, orderID(oi)))
}

// staffedLaterToday reports whether some room of modality m has a
// technologist rostered between now and closing.
func (st *state) staffedLaterToday(m string) bool {
	now := simtime.ClockOf(st.now)
	for _, seg := range st.cov.Segments {
		if !seg.Open || seg.To <= now {
			continue
		}
		for r, room := range st.cfg.Rooms {
			if room.Modality == m && len(seg.Rooms[r]) > 0 {
				return true
			}
		}
	}
	return false
}

// patience: a walk-in still waiting gives up and leaves. The event is stale
// once the visit has moved on (step 8: registration or imaging began).
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
// waiting list = open visits, order and visit states agree, no appointment
// both arrived and a no-show.
func (st *state) checkFlow() error {
	d := &st.dem
	open := 0
	for vi := range d.visits {
		v := &d.visits[vi]
		o := &d.orders[v.order]
		if v.state == visitWaiting {
			open++
			if o.state != orderVisit || o.visit != vi {
				return fmt.Errorf("%s waiting but order %s is %d", visitID(vi), orderID(v.order), o.state)
			}
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
		return fmt.Errorf("%d open visits, waiting list %d", open, len(d.waiting))
	}
	for _, vi := range d.waiting {
		if d.visits[vi].state != visitWaiting {
			return fmt.Errorf("%s on the waiting list but not waiting", visitID(vi))
		}
	}
	return nil
}
