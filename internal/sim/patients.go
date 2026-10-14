package sim

import (
	"fmt"
	"slices"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Observable projections of patients. Everything here comes from events
// that already happened; latent arrival times, no-show draws and patience
// are never read.

func flowView(c flowCounts) api.FlowCounts {
	return api.FlowCounts{Arrived: c.arrived, Early: c.early, Late: c.late, AfterGrace: c.afterGrace,
		WalkIns: c.walkIns, WalkInsBooked: c.walkInsBooked, Missed: c.missed, NoShows: c.noShows,
		Rescheduled: c.rescheduled, Left: c.left, Registered: c.registered, SelfRegistered: c.selfRegistered,
		ExamsStarted: c.examsStarted, ExamsDone: c.examsDone, AfterHours: c.afterHours, NoCoverage: c.noCoverage,
		HeldOver: c.heldOver, SkippedBreaks: c.skippedBreaks, OvertimeMin: int64(c.otSec / simtime.Minute),
		OvertimeCostCents: c.otCost}
}

var visitStateKeys = [...]string{visitArrived: api.VisitArrived, visitRegistering: api.VisitRegistering,
	visitWaiting: api.VisitWaiting, visitInExam: api.VisitInExam}

// examView describes the exam in progress for visit vi.
func (st *state) examView(vi int, now simtime.Time) *api.RoomExam {
	v := &st.dem.visits[vi]
	o := &st.dem.orders[v.order]
	x := &st.cfg.Catalog[o.exam]
	return &api.RoomExam{Visit: visitID(vi), Patient: patientID(o.patient), ExamName: x.Name,
		Technologist: st.emps[v.examBy].cfg.Name, StartedAt: st.cal.Clock(v.examStart),
		ElapsedMin: int64(now.Sub(v.examStart) / simtime.Minute), ExpectedMin: int64(x.Expected() / simtime.Minute)}
}

// observePractice fills the closing phase and today's overtime.
func (st *state) observePractice(o *api.Observation, now simtime.Time) {
	p := &o.Practice
	switch {
	case st.open:
		p.Phase = api.PhaseOpen
	case st.closing:
		p.Phase = api.PhaseClosing
	default:
		p.Phase = api.PhaseClosed
	}
	if h := st.plan.Hours; h != nil {
		p.ScheduledClose = simtime.FormatClock(h.To)
	}
	if st.opClosedAt > 0 {
		p.OperationalClose = st.cal.Clock(st.opClosedAt)
	}
	p.OnSite = len(st.dem.waiting)
	p.OvertimeCostCents = st.dem.flowToday.otCost
	for i := range st.emps {
		p.OvertimeMin += int64(st.overtimeNow(i, now) / simtime.Minute)
		if st.emps[i].held {
			p.HeldOver = append(p.HeldOver, st.emps[i].cfg.Name)
		}
	}
}

func (st *state) observeFlow(o *api.Observation, now simtime.Time) {
	d := &st.dem
	f := &o.Flow
	f.Today, f.SinceStart = flowView(d.flowToday), flowView(d.flowTotal)
	f.NoShowsLastClose = d.noShowsLastClose
	if today := st.cal.DayIndex(now); today < int64(len(d.days)) {
		for _, ai := range d.days[today].appts {
			if d.appts[ai].state == apptMissed {
				f.MissedNow++
			}
		}
	}
	for _, vi := range d.waiting {
		v := &d.visits[vi]
		ord := &d.orders[v.order]
		x := &st.cfg.Catalog[ord.exam]
		w := api.WaitingPatient{State: visitStateKeys[v.state], Visit: visitID(vi), Patient: patientID(ord.patient), Order: orderID(v.order),
			Modality: x.Modality, Exam: x.Code, ExamName: x.Name, Priority: ord.priority.String(),
			WalkIn: ord.walkIn, Arrived: st.cal.Clock(v.arrived), WaitedMin: int64(now.Sub(v.arrived) / simtime.Minute),
			LateMin: int64(v.late / simtime.Minute), AfterGrace: v.afterGrace}
		if v.appt >= 0 {
			a := &d.appts[v.appt]
			w.Appointment = apptID(v.appt)
			w.ApptTime = simtime.FormatClock(d.days[a.day].rooms[a.room][a.slot].From)
		}
		f.Waiting = append(f.Waiting, w)
	}
}

// Patients lists the patients of day index day: every appointment booked
// for that day (in time order, cancelled ones included) and every walk-in.
func (w *World) Patients(day int64) []api.PatientRow {
	st := w.st
	d := &st.dem
	if day < 0 || day >= int64(len(d.days)) {
		return nil
	}
	ds := d.days[day]
	now := w.eng.Now()
	appts := slices.Clone(ds.appts)
	slices.SortStableFunc(appts, func(x, y int) int {
		a, b := &d.appts[x], &d.appts[y]
		sa, sb := ds.rooms[a.room][a.slot].From, ds.rooms[b.room][b.slot].From
		if sa != sb {
			return int(sa - sb)
		}
		return a.room - b.room
	})
	var rows []api.PatientRow
	for _, ai := range appts {
		a := &d.appts[ai]
		r := st.patientRow(a.order, now)
		r.Appointment = apptID(ai)
		r.ApptTime = simtime.FormatClock(ds.rooms[a.room][a.slot].From)
		r.Status, r.StatusText = st.apptStatus(ai)
		if a.visit >= 0 {
			st.fillVisit(&r, a.visit, now)
		}
		rows = append(rows, r)
	}
	for _, oi := range ds.walkIns {
		r := st.patientRow(oi, now)
		o := &d.orders[oi]
		switch {
		case o.visit >= 0:
			st.fillVisit(&r, o.visit, now)
		case len(o.appts) > 0:
			r.Status, r.StatusText = api.PatientSentToBook, "booked for later"
		default:
			r.Status, r.StatusText = api.PatientLost, "lost (no slot)"
		}
		rows = append(rows, r)
	}
	return rows
}

func (st *state) patientRow(oi int, now simtime.Time) api.PatientRow {
	o := &st.dem.orders[oi]
	x := &st.cfg.Catalog[o.exam]
	return api.PatientRow{Patient: patientID(o.patient), Order: orderID(oi), Modality: x.Modality,
		ExamName: x.Name, Priority: o.priority.String(), WalkIn: o.walkIn}
}

func (st *state) fillVisit(r *api.PatientRow, vi int, now simtime.Time) {
	v := &st.dem.visits[vi]
	r.Visit = visitID(vi)
	r.Arrived = st.cal.Clock(v.arrived)
	// Waited: until now while still waiting (to register, at the desk, or
	// for the exam); until the exam started once imaged; until they left or
	// were rescheduled otherwise.
	end := now
	switch v.state {
	case visitInExam, visitDone:
		end = v.examStart
	case visitLeft, visitRescheduled:
		end = v.endedAt
	}
	r.WaitedMin = int64(end.Sub(v.arrived) / simtime.Minute)
	if v.appt >= 0 {
		r.Punctuality = punctuality(v.late)
	}
	switch v.state {
	case visitArrived:
		r.Status, r.StatusText = api.PatientArrived, "waiting to register"
	case visitRegistering:
		r.Status, r.StatusText = api.PatientRegistering, "registering"
	case visitWaiting:
		r.Status, r.StatusText = api.PatientWaiting, "waiting for exam"
		if v.afterGrace {
			r.StatusText = "waiting (late, fitted in)"
		}
	case visitInExam:
		r.Status, r.StatusText = api.PatientInExam, "in exam"
	case visitDone:
		r.Status, r.StatusText = api.PatientDone, "imaged"
		if v.study >= 0 {
			switch s := &st.dem.studies[v.study]; s.state {
			case studyAcquired, studyUnread:
				r.StatusText = "imaged · report pending"
			case studyReading:
				r.StatusText = "imaged · being read"
			case studyFinal:
				r.StatusText = "report final"
			}
		}
	case visitLeft:
		r.Status, r.StatusText = api.PatientLeft, "left without an exam"
	case visitRescheduled:
		r.Status, r.StatusText = api.PatientRescheduled, "rescheduled: "+v.reason
	}
}

func (st *state) apptStatus(ai int) (string, string) {
	switch st.dem.appts[ai].state {
	case apptBooked:
		return api.PatientBooked, "booked"
	case apptCancelled:
		return api.PatientCancelled, "cancelled"
	case apptMissed:
		return api.PatientMissed, "missed (provisional)"
	case apptNoShow:
		return api.PatientNoShow, "no-show"
	case apptRescheduled:
		return api.PatientRescheduled, "rescheduled"
	}
	return api.PatientWaiting, "arrived" // overwritten by fillVisit
}

// Journey returns the observable history of patient id ("P00042").
func (w *World) Journey(id string) (api.Journey, bool) {
	st := w.st
	d := &st.dem
	var n int64
	if _, err := fmt.Sscanf(id, "P%d", &n); err != nil {
		return api.Journey{}, false
	}
	oi := -1
	for i := range d.orders {
		if d.orders[i].patient == n {
			oi = i
			break
		}
	}
	if oi < 0 {
		return api.Journey{}, false
	}
	o := &d.orders[oi]
	x := &st.cfg.Catalog[o.exam]
	j := api.Journey{Patient: id, Order: orderID(oi), Exam: x.Code, ExamName: x.Name, Modality: x.Modality,
		Priority: o.priority.String(), Source: "—", WalkIn: o.walkIn}
	if o.source >= 0 {
		j.Source = st.cfg.Sources[o.source].Name
	}
	iso := func(t simtime.Time) string { return st.cal.ISO(t) }
	switch {
	case o.walkIn:
		j.Source = "walk-in"
		j.Steps = append(j.Steps, api.JourneyStep{Time: iso(o.received), Label: "Walked in", Detail: "order in hand, no appointment", Done: true})
	case o.key.day < 0:
		j.Steps = append(j.Steps, api.JourneyStep{Label: "Ordered", Detail: "before the simulation started", Done: true})
	default:
		j.Steps = append(j.Steps, api.JourneyStep{Time: iso(o.received), Label: "Ordered", Detail: "referred by " + j.Source, Done: true})
	}
	for _, ai := range o.appts {
		a := &d.appts[ai]
		s := d.days[a.day].rooms[a.room][a.slot]
		start := st.cal.At(a.day, s.From)
		booked := api.JourneyStep{Time: iso(a.bookedAt), Label: "Booked",
			Detail: fmt.Sprintf("%s for %s (%s)", apptID(ai), st.cal.Short(start), st.cfg.Rooms[a.room].Name), Done: true}
		if o.key.day < 0 {
			booked.Time = "" // on the opening schedule
		}
		j.Steps = append(j.Steps, booked)
		switch a.state {
		case apptBooked:
			j.Steps = append(j.Steps, api.JourneyStep{Time: iso(start), Label: "Appointment", Detail: "upcoming"})
		case apptCancelled:
			j.Steps = append(j.Steps, api.JourneyStep{Time: iso(a.endedAt), Label: "Cancelled", Detail: "by the patient", Done: true})
		case apptMissed, apptNoShow:
			j.Steps = append(j.Steps, api.JourneyStep{Time: iso(a.missedAt), Label: "Missed",
				Detail: "not arrived by the end of the grace period", Done: true})
			if a.state == apptNoShow {
				j.Steps = append(j.Steps, api.JourneyStep{Time: iso(a.endedAt), Label: "No-show", Detail: "finalized at accounting close", Done: true})
			}
		case apptArrived:
			if a.missedAt != 0 {
				j.Steps = append(j.Steps, api.JourneyStep{Time: iso(a.missedAt), Label: "Missed (provisional)",
					Detail: "not arrived by the end of the grace period", Done: true})
			}
		case apptRescheduled:
			j.Steps = append(j.Steps, api.JourneyStep{Time: iso(a.endedAt), Label: "Arrived, rescheduled",
				Detail: punctuality(a.endedAt.Sub(start)) + "; too late to be seen", Done: true})
		}
	}
	if o.visit >= 0 {
		v := &d.visits[o.visit]
		detail := visitID(o.visit)
		if v.appt >= 0 {
			detail += " · " + punctuality(v.late)
			if v.afterGrace {
				detail += ", after being marked missed (fitted in)"
			}
		}
		j.Steps = append(j.Steps, api.JourneyStep{Time: iso(v.arrived), Label: "Arrived", Detail: detail, Done: true})
		now := w.eng.Now()
		if v.state == visitArrived {
			j.Steps = append(j.Steps, api.JourneyStep{Label: "Waiting to register",
				Detail: fmt.Sprintf("%s so far", now.Sub(v.arrived)), Done: true})
		}
		if v.regBy >= 0 {
			step := api.JourneyStep{Time: iso(v.regStart), Label: "Registered", Done: true,
				Detail: st.emps[v.regBy].cfg.Name}
			if v.state == visitRegistering {
				step.Label, step.Detail = "Registering", step.Detail+fmt.Sprintf(" · %s so far", now.Sub(v.regStart))
			} else {
				step.Detail += fmt.Sprintf(" · %s–%s", st.cal.Clock(v.regStart), st.cal.Clock(v.regEnd))
			}
			j.Steps = append(j.Steps, step)
		}
		if v.state == visitWaiting {
			j.Steps = append(j.Steps, api.JourneyStep{Label: "Waiting for exam",
				Detail: fmt.Sprintf("%s since registration", now.Sub(v.regEnd)), Done: true})
		}
		if v.examBy >= 0 {
			x := &st.cfg.Catalog[o.exam]
			step := api.JourneyStep{Time: iso(v.examStart), Label: "Exam", Done: true,
				Detail: fmt.Sprintf("%s · %s · waited %s from arrival", st.emps[v.examBy].cfg.Name, st.cfg.Rooms[v.room].Name, v.examStart.Sub(v.arrived))}
			if v.state == visitInExam {
				step.Label = "In exam"
				step.Detail += fmt.Sprintf(" · %s so far (expected %s)", now.Sub(v.examStart), x.Expected())
			} else {
				step.Detail += fmt.Sprintf(" · took %s (expected %s)", v.examEnd.Sub(v.examStart), x.Expected())
			}
			if v.afterHours {
				step.Detail += " · after scheduled close"
			}
			j.Steps = append(j.Steps, step)
		}
		switch v.state {
		case visitDone:
			j.Steps = append(j.Steps, api.JourneyStep{Time: iso(v.examEnd), Label: "Imaging complete", Detail: "departed", Done: true})
			j.Steps = append(j.Steps, st.studySteps(v.study, now)...)
		case visitLeft:
			j.Steps = append(j.Steps, api.JourneyStep{Time: iso(v.endedAt), Label: "Left",
				Detail: fmt.Sprintf("without an exam after waiting %s", v.endedAt.Sub(v.arrived)), Done: true})
		case visitRescheduled:
			j.Steps = append(j.Steps, api.JourneyStep{Time: iso(v.endedAt), Label: "Rescheduled",
				Detail: v.reason, Done: true})
		}
	}
	switch o.state {
	case orderLost:
		j.Steps = append(j.Steps, api.JourneyStep{Time: iso(o.endedAt), Label: "Lost", Detail: o.lostWhy, Done: true})
	case orderRecall:
		j.Steps = append(j.Steps, api.JourneyStep{Label: "Recall list",
			Detail: "due " + st.cal.Date(st.cal.StartOfDay(o.earliest)).String() + "; booked when it enters the horizon"})
	}
	j.Status = orderStatusText[o.state]
	return j, true
}

var orderStatusText = [...]string{orderBooked: "booked", orderRecall: "on the recall list", orderLost: "lost",
	orderCancelled: "cancelled", orderVisit: "on site", orderNoShow: "no-show", orderLeft: "left without an exam",
	orderImaged: "imaged, report pending", orderReported: "report final"}

// studySteps describes a study's reporting: documentation, the worklist,
// the read and the final report. Read times appear only once the read
// has ended.
func (st *state) studySteps(si int, now simtime.Time) []api.JourneyStep {
	if si < 0 {
		return nil
	}
	s := &st.dem.studies[si]
	iso := st.cal.ISO
	target := st.cfg.Policies.TATTarget[s.priority]
	var out []api.JourneyStep
	if s.state == studyAcquired {
		return append(out, api.JourneyStep{Label: "Study " + studyID(si), Detail: "technologist documenting", Done: true})
	}
	out = append(out, api.JourneyStep{Time: iso(s.enteredAt), Label: "On worklist", Done: true,
		Detail: fmt.Sprintf("%s · %s priority · TAT target %s", studyID(si), s.priority, target)})
	switch s.state {
	case studyUnread:
		out = append(out, api.JourneyStep{Label: "Awaiting read", Done: true,
			Detail: fmt.Sprintf("%s since imaging", now.Sub(s.acquiredAt))})
	case studyReading:
		out = append(out, api.JourneyStep{Time: iso(s.claimedAt), Label: "Being read", Done: true,
			Detail: fmt.Sprintf("%s · %s so far", st.emps[s.by].cfg.Name, now.Sub(s.claimedAt))})
	case studyFinal:
		out = append(out, api.JourneyStep{Time: iso(s.claimedAt), Label: "Read", Done: true,
			Detail: fmt.Sprintf("%s · waited %s on the worklist · read %s", st.emps[s.by].cfg.Name,
				s.claimedAt.Sub(s.enteredAt), s.finalAt.Sub(s.claimedAt))})
		tat := s.finalAt.Sub(s.acquiredAt)
		d := fmt.Sprintf("TAT %s (imaging complete → final)", tat)
		if tat > target {
			d += fmt.Sprintf(" · over the %s target", target)
		}
		out = append(out, api.JourneyStep{Time: iso(s.finalAt), Label: "Report final", Detail: d, Done: true})
	}
	return out
}
