package sim

import (
	"slices"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Metrics (SIMULATION_DESIGN §11): measured from observed facts only.
//
// One cumulative tally runs from the epoch. Its interval part integrates
// the state between transactions (state only changes inside them): room
// open, staffed and busy time, employee duty, task, break, overtime and
// reading-block time, and patient-time waiting to register with no desk
// staff. Counters and sample lists only grow. A period's metrics are the
// difference of two snapshots of the tally plus the samples between them,
// so a calendar day (midnight snapshots), today so far (a snapshot taken
// now, without changing anything) and the evaluation window (a snapshot
// at the takeover instant) share one definition.

type roomTally struct {
	open, staffed, busy, afterHours simtime.Duration
	exams                           int
}

type empTally struct {
	duty, busy, brk, over  simtime.Duration
	approved               simtime.Duration // of over, approved in advance
	block, readIn, readOut simtime.Duration
}

// tally is the cumulative measurement at time at. flow and dem hold the
// operational counters (closed days folded in, plus today's in a
// snapshot); the list lengths bound the samples, studies, reports and
// alerts that existed then.
type tally struct {
	at       simtime.Time
	rooms    []roomTally
	emps     []empTally
	deskSelf simtime.Duration // patient-time waiting to register, no desk staff on duty
	flow     flowCounts
	dem      [maxMods]modalityCounts

	waits, studies, finals, raised int
	actions                        int // action records
	orders                         int // orders received
	backlog                        int // unread + reading
}

func (t *tally) clone() tally {
	c := *t
	c.rooms, c.emps = slices.Clone(t.rooms), slices.Clone(t.emps)
	return c
}

// waitSample is one exam start: seconds from arrival to registration
// start and to exam start, and past the appointment time (−1 for
// walk-ins and patients fitted in after being marked missed).
type waitSample struct {
	mod                   int8
	reg, toExam, pastAppt int32
}

type metricsState struct {
	cum   tally   // intervals accrued to cum.at; flow and dem hold closed days only
	marks []tally // marks[d]: the tally at the start of day d
	waits []waitSample
	// unreadAtClose[d]: the worklist at day d's accounting close.
	unreadAtClose []int
	// done caches the summaries of completed days (they never change).
	done map[int64]api.DaySummary
}

func (st *state) initMetrics() {
	st.met.cum.rooms = make([]roomTally, len(st.rooms))
	st.met.cum.emps = make([]empTally, len(st.emps))
	st.met.done = map[int64]api.DaySummary{}
}

// accrue integrates the state since the last transaction up to now. It
// runs before the first event of each transaction changes anything.
func (st *state) accrue(now simtime.Time) {
	if dt := now.Sub(st.met.cum.at); dt > 0 {
		st.addIntervals(&st.met.cum, dt)
	}
	st.met.cum.at = now
}

// addIntervals adds dt of the current state to t.
func (st *state) addIntervals(t *tally, dt simtime.Duration) {
	for r := range st.rooms {
		rm, x := &st.rooms[r], &t.rooms[r]
		switch {
		case st.open:
			x.open += dt
			if rm.state == roomIdle || rm.state == roomOccupied {
				x.staffed += dt
			}
			if rm.state == roomOccupied {
				x.busy += dt
			}
		case rm.state == roomOccupied:
			x.afterHours += dt
		}
	}
	from, to := t.at, t.at.Add(dt)
	for i := range st.emps {
		emp, x := &st.emps[i], &t.emps[i]
		switch emp.status {
		case available, busy:
			x.duty += dt
			// Overtime: on duty past the rostered end; approved up to the
			// extended end, required beyond it.
			x.over += max(0, to.Sub(max(from, emp.rosterEnd)))
			x.approved += max(0, min(to, emp.endAt).Sub(max(from, emp.rosterEnd)))
			if emp.reading {
				x.block += dt
			}
			if emp.status == busy {
				x.busy += dt
				if emp.task == taskRead {
					if emp.reading {
						x.readIn += dt
					} else {
						x.readOut += dt
					}
				}
			}
		case onBreak:
			x.brk += dt
		}
	}
	if st.open || st.closing {
		if lvl, _ := st.desk(); lvl == DeskSelf {
			n := 0
			for _, vi := range st.dem.waiting {
				if st.dem.visits[vi].state == visitArrived {
					n++
				}
			}
			t.deskSelf += simtime.Duration(n) * dt
		}
	}
}

// snapshot returns the tally as of now without changing the state.
func (st *state) snapshot(now simtime.Time) tally {
	t := st.met.cum.clone()
	if dt := now.Sub(t.at); dt > 0 {
		st.addIntervals(&t, dt)
	}
	t.at = now
	t.flow.add(&st.dem.flowToday)
	for k, m := range st.mods {
		t.dem[k].add(st.dem.today[m])
	}
	t.waits, t.studies, t.finals = len(st.met.waits), len(st.dem.studies), len(st.dem.finals)
	t.raised, t.backlog = len(st.alerts.raised), len(st.dem.worklist)
	t.actions, t.orders = len(st.actions.log), len(st.dem.orders)
	return t
}

// metricsCloseDay runs at midnight, after accrual: yesterday's counters
// fold into the cumulative tally before dayStart resets them; metricsMark
// then takes the new day's opening mark.
func (st *state) metricsCloseDay() {
	st.met.cum.flow.add(&st.dem.flowToday)
	for k, m := range st.mods {
		st.met.cum.dem[k].add(st.dem.today[m])
	}
}

func (st *state) metricsMark(day int64) {
	if int64(len(st.met.marks)) != day {
		fail("metrics mark for day %d with %d marks", day, len(st.met.marks))
	}
	st.met.marks = append(st.met.marks, st.snapshot(st.now))
}

func (c *modalityCounts) add(d *modalityCounts) {
	c.received += d.received
	c.booked += d.booked
	c.recall += d.recall
	c.lost += d.lost
	c.cancelled += d.cancelled
}

func (f *flowCounts) add(g *flowCounts) {
	f.arrived += g.arrived
	f.early += g.early
	f.late += g.late
	f.afterGrace += g.afterGrace
	f.walkIns += g.walkIns
	f.walkInsAdmitted += g.walkInsAdmitted
	f.walkInsDeclined += g.walkInsDeclined
	f.walkInsUnservable += g.walkInsUnservable
	f.walkInsOfferBooked += g.walkInsOfferBooked
	f.walkInsOfferFailed += g.walkInsOfferFailed
	f.walkInSameVisitExams += g.walkInSameVisitExams
	f.walkInReturnExams += g.walkInReturnExams
	f.wiBooked += g.wiBooked
	f.wiLost += g.wiLost
	f.wiCancelled += g.wiCancelled
	f.otApprovedSec += g.otApprovedSec
	f.otApprovedCost += g.otApprovedCost
	f.missed += g.missed
	f.noShows += g.noShows
	f.rescheduled += g.rescheduled
	f.left += g.left
	f.registered += g.registered
	f.selfRegistered += g.selfRegistered
	f.examsStarted += g.examsStarted
	f.examsDone += g.examsDone
	f.afterHours += g.afterHours
	f.noCoverage += g.noCoverage
	f.heldOver += g.heldOver
	f.skippedBreaks += g.skippedBreaks
	f.otSec += g.otSec
	f.otCost += g.otCost
	f.acquired += g.acquired
	f.reportsFinal += g.reportsFinal
	f.breaches += g.breaches
	for k := range f.disp {
		f.disp[k] += g.disp[k]
	}
}

// walkInCohort follows the walk-ins who arrived with orders [from, to)
// to their outcome as of time at, from timestamps only (later events do
// not leak in): each originating walk-in is counted once, in exactly one
// outcome, unresolved ones included.
func (st *state) walkInCohort(from, to int, at simtime.Time) api.WalkInCohort {
	var c api.WalkInCohort
	for oi := from; oi < to; oi++ {
		o := &st.dem.orders[oi]
		if !o.walkIn {
			continue
		}
		c.Arrived++
		examined := o.examAt != 0 && o.examAt <= at
		switch {
		case examined && o.examSameVisit:
			c.ImagedSameVisit++
		case len(o.appts) > 0 && st.dem.appts[o.appts[0]].bookedAt <= at:
			c.Converted++
			if examined {
				c.ConvertedImaged++
			}
		case o.state == orderLeft && o.endedAt <= at:
			c.Left++
		case o.state == orderLost && o.endedAt <= at:
			c.Lost++
		default:
			c.OnSite++
		}
	}
	return c
}

// noteExamStart records the wait sample for visit vi.
func (st *state) noteExamStart(vi int) {
	v := &st.dem.visits[vi]
	o := &st.dem.orders[v.order]
	s := waitSample{mod: st.modIndex(st.cfg.Catalog[o.exam].Modality), pastAppt: -1,
		toExam: int32(v.examStart.Sub(v.arrived)), reg: int32(v.regStart.Sub(v.arrived))}
	if v.appt >= 0 && !v.afterGrace {
		s.pastAppt = int32(max(0, v.examStart.Sub(max(v.arrived, st.apptStart(v.appt)))))
	}
	if o.examAt == 0 {
		o.examAt, o.examSameVisit = v.examStart, v.appt < 0
		if o.walkIn {
			for _, f := range []*flowCounts{&st.dem.flowToday, &st.dem.flowTotal} {
				if o.examSameVisit {
					f.walkInSameVisitExams++
				} else {
					f.walkInReturnExams++
				}
			}
		}
	}
	st.met.waits = append(st.met.waits, s)
}

// --- periods -------------------------------------------------------------

// dist summarizes durations with nearest-rank percentiles. It sorts xs.
func dist(xs []int64) api.Dist {
	n := len(xs)
	if n == 0 {
		return api.Dist{}
	}
	slices.Sort(xs)
	return api.Dist{Count: n, P50Sec: xs[rank(n, 50)], P90Sec: xs[rank(n, 90)], MaxSec: xs[n-1]}
}

// rank is the nearest-rank index of the p-th percentile of n values:
// ceil(p·n/100) − 1.
func rank(n, p int) int { return (n*p+99)/100 - 1 }

func ratio(num, den int64) api.Ratio { return api.Ratio{Num: num, Den: den} }

func secs(d simtime.Duration) int64 { return int64(d) }

// periodMetrics measures the period between snapshots a and b. complete
// says the period has ended (b is its end, not just now).
func (st *state) periodMetrics(a, b *tally, complete bool) api.PeriodMetrics {
	p := api.PeriodMetrics{From: st.cal.ISO(a.at), To: st.cal.ISO(b.at), Complete: complete}
	f := b.flow
	f.sub(&a.flow)

	// Demand.
	dm := &p.Demand
	for k, m := range st.mods {
		c := b.dem[k]
		c.sub(&a.dem[k])
		dm.Received += c.received
		dm.Booked += c.booked
		dm.Recall += c.recall
		dm.Lost += c.lost
		dm.Cancelled += c.cancelled
		dm.ByModality = append(dm.ByModality, api.ModalityDemandMetrics{Modality: m, Received: c.received, Lost: c.lost})
	}
	dm.LostRate = ratio(int64(dm.Lost), int64(dm.Received))
	dm.WalkInOrigin = api.WalkInOriginDemand{Booked: f.wiBooked, Lost: f.wiLost, Cancelled: f.wiCancelled}

	// Appointments whose slot starts in [a.at, b.at).
	p.Appointments = st.apptMetrics(a.at, b.at)
	p.Displacement = dispMetrics(&f)

	// Walk-ins.
	wi := &p.WalkIns
	wi.Arrived, wi.Admitted, wi.Left = f.walkIns, f.walkInsAdmitted, f.left
	wi.Declined, wi.Unservable = f.walkInsDeclined, f.walkInsUnservable
	wi.Offered = f.walkInsDeclined + f.walkInsUnservable
	wi.OfferBooked, wi.OfferFailed = f.walkInsOfferBooked, f.walkInsOfferFailed
	wi.SameVisitExams, wi.ReturnExams = f.walkInSameVisitExams, f.walkInReturnExams
	wi.Cohort = st.walkInCohort(a.orders, b.orders, b.at)
	wi.Acceptance = ratio(int64(wi.Cohort.ImagedSameVisit), int64(wi.Cohort.Arrived))
	wi.Abandonment = ratio(int64(wi.Cohort.Left), int64(wi.Cohort.Arrived))

	// Waits over exams started in the period.
	var reg, toExam, past []int64
	for _, s := range st.met.waits[a.waits:b.waits] {
		reg = append(reg, int64(s.reg))
		toExam = append(toExam, int64(s.toExam))
		if s.toExam > 30*60 {
			p.Waits.Over30++
		}
		if s.pastAppt >= 0 {
			past = append(past, int64(s.pastAppt))
		}
	}
	p.Waits.Registration, p.Waits.ToExam, p.Waits.PastAppointment = dist(reg), dist(toExam), dist(past)
	p.Exams, p.AfterHoursExams = f.examsDone, f.afterHours

	// Rooms.
	for r, rc := range st.cfg.Rooms {
		x, y := &a.rooms[r], &b.rooms[r]
		rm := api.RoomMetrics{Room: rc.ID, Modality: rc.Modality,
			OpenSec: secs(y.open - x.open), StaffedSec: secs(y.staffed - x.staffed), BusySec: secs(y.busy - x.busy),
			AfterHoursBusySec: secs(y.afterHours - x.afterHours), Exams: y.exams - x.exams}
		rm.UtilOpen = ratio(rm.BusySec, rm.OpenSec)
		rm.UtilStaffed = ratio(rm.BusySec, rm.StaffedSec)
		rm.Coverage = ratio(rm.StaffedSec, rm.OpenSec)
		p.Rooms = append(p.Rooms, rm)
		if k := st.modIndex(rc.Modality); k >= 0 {
			dm.ByModality[k].Exams += rm.Exams
		}
	}

	// Staff.
	// Employees disclosed by the period's end (a hire from its acceptance);
	// one hired during the period has no tally at its start.
	for i := range b.emps {
		ec := st.emps[i].terms
		var x0 empTally
		x, y := &x0, &b.emps[i]
		if i < len(a.emps) {
			x = &a.emps[i]
		}
		sm := api.StaffMetrics{Name: ec.Name, Role: string(ec.Role), OnDutySec: secs(y.duty - x.duty), BusySec: secs(y.busy - x.busy),
			BreakSec: secs(y.brk - x.brk), OvertimeSec: secs(y.over - x.over), ApprovedOvertimeSec: secs(y.approved - x.approved),
			BlockSec:       secs(y.block - x.block),
			ReadInBlockSec: secs(y.readIn - x.readIn), ReadOutsideSec: secs(y.readOut - x.readOut)}
		sm.Util = ratio(sm.BusySec, sm.OnDutySec)
		sm.ReadUtil = ratio(sm.ReadInBlockSec, sm.BlockSec)
		p.Staff = append(p.Staff, sm)
	}

	p.Reporting = st.reportingMetrics(a, b)
	p.Reporting.Breaches = f.breaches

	w := &p.Workforce
	w.OvertimeSec, w.OvertimeCostCents = secs(f.otSec), f.otCost
	w.ApprovedOvertimeSec, w.ApprovedOvertimeCostCents = secs(f.otApprovedSec), f.otApprovedCost
	w.HeldOver, w.SkippedBreaks, w.NoCoverage = f.heldOver, f.skippedBreaks, f.noCoverage
	w.DeskSelfPatientSec = secs(b.deskSelf - a.deskSelf)

	p.Alerts = st.alertCounts(a.raised, b.raised)
	p.Actions = st.actionCounts(a.actions, b.actions)
	return p
}

// apptMetrics covers appointments in slots starting in [from, to).
func (st *state) apptMetrics(from, to simtime.Time) api.ApptMetrics {
	var m api.ApptMetrics
	d := &st.dem
	for day := max(0, st.cal.DayIndex(from)); day <= st.cal.DayIndex(to) && day < int64(len(d.days)); day++ {
		ds := d.days[day]
		for r := range ds.rooms {
			for _, s := range ds.rooms[r] {
				if at := st.cal.At(day, s.From); at < from || at >= to {
					continue
				}
				if s.why == slotOK {
					m.BookableSec += secs(s.To - s.From)
				}
				if s.appt >= 0 {
					m.BookedSec += secs(s.To - s.From)
				}
			}
		}
		for _, ai := range ds.appts {
			a := &d.appts[ai]
			if at := st.apptStart(ai); at < from || at >= to {
				continue
			}
			m.Scheduled++
			switch a.state {
			case apptCancelled:
				m.Cancelled++
			case apptMissed:
				m.NoShows++
				m.Missed++
			case apptNoShow:
				m.NoShows++
			case apptRescheduled:
				m.Rescheduled++
			case apptDisplaced:
				m.Displaced++
			}
			if a.visit >= 0 {
				m.Arrived++
			}
		}
	}
	m.NoShowRate = ratio(int64(m.NoShows), int64(m.Scheduled-m.Cancelled-m.Displaced))
	m.CancelRate = ratio(int64(m.Cancelled), int64(m.Scheduled-m.Displaced))
	m.Fill = ratio(m.BookedSec, m.BookableSec)
	return m
}

// reportingMetrics: censored TAT over the studies acquired between a and
// b (unfinished ones at their age at b), TAT of the reports finalized
// between them, and the worklist at both ends.
func (st *state) reportingMetrics(a, b *tally) api.ReportingMetrics {
	d := &st.dem
	pol := &st.cfg.Policies
	rm := api.ReportingMetrics{BacklogStart: a.backlog, BacklogEnd: b.backlog}
	var all []int64
	byPri := make([][]int64, numPriorities)
	over := make([]int, numPriorities)
	for si := a.studies; si < b.studies; si++ {
		s := &d.studies[si]
		end := b.at
		if s.state == studyFinal && s.finalAt <= b.at {
			end = s.finalAt
		} else {
			rm.Unfinished++
		}
		tat := secs(end.Sub(s.acquiredAt))
		all = append(all, tat)
		byPri[s.priority] = append(byPri[s.priority], tat)
		if tat > secs(pol.TATTarget[s.priority]) {
			rm.OverTarget++
			over[s.priority]++
		}
	}
	rm.Acquired = len(all)
	rm.TAT = dist(all)
	for p := Routine; p < numPriorities; p++ {
		rm.ByPriority = append(rm.ByPriority, api.PriorityMetrics{Priority: p.String(), Acquired: len(byPri[p]),
			TAT: dist(byPri[p]), OverTarget: over[p], TargetSec: secs(pol.TATTarget[p])})
	}
	var fin []int64
	for _, si := range d.finals[a.finals:b.finals] {
		s := &d.studies[si]
		fin = append(fin, secs(s.finalAt.Sub(s.acquiredAt)))
		if si < a.studies {
			rm.Inherited++
		}
	}
	rm.Final = len(fin)
	rm.FinalTAT = dist(fin)
	return rm
}

func (c *modalityCounts) sub(d *modalityCounts) {
	c.received -= d.received
	c.booked -= d.booked
	c.recall -= d.recall
	c.lost -= d.lost
	c.cancelled -= d.cancelled
}

func (f *flowCounts) sub(g *flowCounts) {
	neg := *g
	neg.scale(-1)
	f.add(&neg)
}

func (f *flowCounts) scale(k int) {
	f.arrived *= k
	f.early *= k
	f.late *= k
	f.afterGrace *= k
	f.walkIns *= k
	f.walkInsAdmitted *= k
	f.walkInsDeclined *= k
	f.walkInsUnservable *= k
	f.walkInsOfferBooked *= k
	f.walkInsOfferFailed *= k
	f.walkInSameVisitExams *= k
	f.walkInReturnExams *= k
	f.wiBooked *= k
	f.wiLost *= k
	f.wiCancelled *= k
	f.otApprovedSec *= simtime.Duration(k)
	f.otApprovedCost *= int64(k)
	f.missed *= k
	f.noShows *= k
	f.rescheduled *= k
	f.left *= k
	f.registered *= k
	f.selfRegistered *= k
	f.examsStarted *= k
	f.examsDone *= k
	f.afterHours *= k
	f.noCoverage *= k
	f.heldOver *= k
	f.skippedBreaks *= k
	f.otSec *= simtime.Duration(k)
	f.otCost *= int64(k)
	f.acquired *= k
	f.reportsFinal *= k
	f.breaches *= k
	for j := range f.disp {
		f.disp[j] *= k
	}
}

// dayBounds returns the snapshots bounding day d: its midnight mark and
// the next (complete) or now (today, partial).
func (st *state) dayBounds(d int64, now simtime.Time) (a, b tally, complete bool) {
	a = st.met.marks[d]
	if d+1 < int64(len(st.met.marks)) {
		return a, st.met.marks[d+1], true
	}
	return a, st.snapshot(now), false
}

// TodayMetrics are today's metrics so far.
func (st *state) todayMetrics(now simtime.Time) api.PeriodMetrics {
	d := st.cal.DayIndex(now)
	if d < 0 || d >= int64(len(st.met.marks)) {
		return api.PeriodMetrics{}
	}
	a, b, complete := st.dayBounds(d, now)
	return st.periodMetrics(&a, &b, complete)
}
