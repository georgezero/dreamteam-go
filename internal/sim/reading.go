package sim

import (
	"fmt"
	"slices"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Reading: studies, the radiologist worklist and report turnaround
// (SIMULATION_DESIGN §2.11, §6.7, §6.8, §11).
//
// A study is acquired when its exam ends (TAT starts), enters the worklist
// as unread when the technologist's documentation is done, is claimed by a
// radiologist (reading) and becomes final when the read ends. TAT =
// final − acquired.
//
// A radiologist claims a read only while on regular duty (not held over),
// inside a site reading block (after any overrun owed from the last block)
// and only if the read's p100 time ends by the scheduled shift end, so
// reading never extends a shift and never crosses midnight. One read at a
// time. Contrast supervision is presence (on duty, on site) and is
// compatible with reading; a remote radiologist reads but never supervises.
// The read time of a case is drawn by its origin key (domain "read", lane
// 0), so worklist order, staffing and speed never change it.

type studyState uint8

const (
	studyAcquired studyState = iota // imaging done; documentation in progress
	studyUnread                     // on the worklist, awaiting interpretation
	studyReading                    // claimed by a radiologist
	studyFinal                      // report finalized
)

type study struct {
	order, visit, exam int
	priority           Priority
	acquiredAt         simtime.Time // exam end: imaging complete (TAT start)
	enteredAt          simtime.Time // documentation done: on the worklist
	claimedAt          simtime.Time
	finalAt            simtime.Time
	by                 int // radiologist; -1 until claimed
	state              studyState
	breached           bool // its TAT target passed before it was final
	claim              int  // claim opened at final; -1 before
}

func studyID(i int) string { return pubID('S', int64(i+1), 5) }

// acquireStudy records imaging completion for visit vi (exam end). Study
// IDs are allocated here, at acquisition.
func (st *state) acquireStudy(e *Engine, vi int) {
	v := &st.dem.visits[vi]
	o := &st.dem.orders[v.order]
	si := len(st.dem.studies)
	st.dem.studies = append(st.dem.studies, study{order: v.order, visit: vi, exam: o.exam, priority: o.priority,
		acquiredAt: e.Now(), by: -1, claim: -1})
	v.study = si
	ds := st.dem.days[st.day]
	ds.studies = append(ds.studies, si)
	st.dem.flowToday.acquired++
	st.dem.flowTotal.acquired++
}

// enterWorklist puts a study on the worklist once the technologist's
// documentation is done, and queues its TAT-target check.
func (st *state) enterWorklist(e *Engine, si int) {
	s := &st.dem.studies[si]
	s.state, s.enteredAt = studyUnread, e.Now()
	st.dem.worklist = append(st.dem.worklist, si)
	x := &st.cfg.Catalog[s.exam]
	st.emit(e.Now(), api.Info, "reading", fmt.Sprintf("%s %s %s on the worklist (%s)", studyID(si), x.Modality, x.Name, s.priority))
	due := s.acquiredAt.Add(st.cfg.Policies.TATTarget[s.priority])
	if s.priority != Routine {
		e.Push(Event{At: max(due, e.Now().Add(1)), Class: ClassDeadline, Kind: kindTATBreach, A: int64(si)})
	}
}

// tatBreach: an urgent or soon study is not final by its TAT target.
// (Routine breaches are counted at finalization or in the censored metrics,
// not logged one by one.)
func (st *state) tatBreach(e *Engine, si int) {
	s := &st.dem.studies[si]
	s.breached = true
	st.dem.flowToday.breaches++
	st.dem.flowTotal.breaches++
	x := &st.cfg.Catalog[s.exam]
	what := "unread"
	if s.state == studyReading {
		what = "being read by " + st.emps[s.by].cfg.Name
	}
	st.emit(e.Now(), api.Warn, "reading", fmt.Sprintf("%s %s report overdue: %s %s not final after %s (%s)",
		studyID(si), s.priority, x.Modality, x.Name, st.cfg.Policies.TATTarget[s.priority], what))
}

// worklistOrder returns the unread studies in the order the worklist
// policy (§6.8) offers them. UrgentThenFIFO, the default, puts urgent
// studies and studies already past their TAT target first (the latter so a
// steady urgent stream can't starve routine work forever), then the rest by
// time on the worklist. Every key ends in the study ID, so ties are
// deterministic.
func (st *state) worklistOrder() []int {
	pol := &st.cfg.Policies
	var out []int
	for _, si := range st.dem.worklist {
		if st.dem.studies[si].state == studyUnread {
			out = append(out, si)
		}
	}
	rank := func(si int) int {
		s := &st.dem.studies[si]
		if s.priority == Urgent || st.now >= s.acquiredAt.Add(pol.TATTarget[s.priority]) {
			return 0
		}
		return 1
	}
	key := func(si int) simtime.Time {
		s := &st.dem.studies[si]
		switch pol.Worklist {
		case "oldest_first":
			return s.acquiredAt
		case "shortest_first":
			// Published expected read time (p50), never the case's own.
			return simtime.Time(st.cfg.Catalog[s.exam].ReadTime.At(500_000))
		}
		return s.enteredAt
	}
	slices.SortStableFunc(out, func(a, b int) int {
		if pol.Worklist == "urgent_then_fifo" {
			if ra, rb := rank(a), rank(b); ra != rb {
				return ra - rb
			}
		}
		if ka, kb := key(a), key(b); ka != kb {
			return int(ka - kb)
		}
		return a - b
	})
	return out
}

// canRead reports whether radiologist i may claim a read now.
func (st *state) canRead(i int) bool {
	emp := &st.emps[i]
	return emp.cfg.Role == Radiologist && emp.status == available && !emp.over && emp.reading && st.now >= emp.readFrom
}

// matchReading lets each free radiologist in a site block, in employee
// order, claim the first study in worklist order whose longest read (p100)
// ends by their scheduled shift end.
func (st *state) matchReading(e *Engine) {
	if len(st.dem.worklist) == 0 {
		return
	}
	var order []int
	for i := range st.emps {
		if !st.canRead(i) {
			continue
		}
		if order == nil {
			order = st.worklistOrder()
		}
		emp := &st.emps[i]
		for k, si := range order {
			s := &st.dem.studies[si]
			if s.state != studyUnread {
				continue
			}
			x := &st.cfg.Catalog[s.exam]
			if st.now.Add(simtime.Duration(x.ReadTime.Max())) > min(emp.endAt, emp.capAt) {
				continue
			}
			st.claim(e, i, si)
			order = slices.Delete(order, k, k+1)
			break
		}
	}
}

func (st *state) claim(e *Engine, i, si int) {
	s := &st.dem.studies[si]
	emp := &st.emps[i]
	o := &st.dem.orders[s.order]
	x := &st.cfg.Catalog[s.exam]
	s.state, s.claimedAt, s.by = studyReading, e.Now(), i
	emp.status, emp.task, emp.taskVisit, emp.taskStudy = busy, taskRead, s.visit, si
	dur := simtime.Duration(x.ReadTime.Sample(st.hash("read", o.key.keys()...).Lane(0)))
	if s.priority == Urgent {
		st.emit(e.Now(), api.Info, "reading", fmt.Sprintf("%s reading urgent %s %s %s", emp.cfg.Name, studyID(si), x.Modality, x.Name))
	}
	e.Push(Event{At: e.Now().Add(max(1, dur)), Class: ClassCompletion, Kind: kindReadEnd, A: int64(si)})
}

// readEnd finalizes the report. A read that ran past its block's end
// (overrun rule, §6.7) delays the radiologist's next block by the overrun.
func (st *state) readEnd(e *Engine, si int) {
	s := &st.dem.studies[si]
	i := s.by
	emp := &st.emps[i]
	s.state, s.finalAt = studyFinal, e.Now()
	st.dem.worklist = slices.DeleteFunc(st.dem.worklist, func(x int) bool { return x == si })
	st.dem.finals = append(st.dem.finals, si)
	o := &st.dem.orders[s.order]
	o.state = orderReported
	if !emp.reading {
		emp.readDebt += e.Now().Sub(emp.blockEndAt)
	}
	emp.status, emp.task, emp.taskVisit, emp.taskStudy = available, taskNone, -1, -1
	tat := s.finalAt.Sub(s.acquiredAt)
	late := tat > st.cfg.Policies.TATTarget[s.priority]
	for _, f := range []*flowCounts{&st.dem.flowToday, &st.dem.flowTotal} {
		f.reportsFinal++
		if late && !s.breached && s.priority == Routine {
			f.breaches++
		}
	}
	if late {
		s.breached = true
	}
	x := &st.cfg.Catalog[s.exam]
	rev := st.recognize(si)
	st.emit(e.Now(), api.Info, "reading", fmt.Sprintf("%s %s %s report final (%s) · TAT %s · revenue %s",
		studyID(si), x.Modality, x.Name, emp.cfg.Name, tat, money(rev)))
}

// blockStart opens a site reading block, after any overrun owed.
func (st *state) blockStart(i, k int) {
	emp := &st.emps[i]
	b := st.plan.Reading[i][k]
	emp.reading = true
	emp.readFrom = st.now.Add(emp.readDebt)
	emp.readDebt = max(0, emp.readDebt-(b.To-b.From))
}

func (st *state) blockEnd(i int) {
	emp := &st.emps[i]
	emp.reading = false
	emp.blockEndAt = st.now
}

// checkReading verifies study invariants: one study per imaged visit and
// none otherwise; the worklist holds exactly the unread and reading
// studies; a reading study's radiologist is on duty, busy with it; and
// timestamps run acquired ≤ entered ≤ claimed ≤ final.
func (st *state) checkReading() error {
	d := &st.dem
	for vi := range d.visits {
		v := &d.visits[vi]
		if (v.state == visitDone) != (v.study >= 0) {
			return fmt.Errorf("%s state %d with study %d", visitID(vi), v.state, v.study)
		}
		if v.study >= 0 && d.studies[v.study].visit != vi {
			return fmt.Errorf("%s and its study disagree", visitID(vi))
		}
	}
	listed := 0
	for si := range d.studies {
		s := &d.studies[si]
		if s.state == studyUnread || s.state == studyReading {
			listed++
		}
		switch {
		case s.state >= studyUnread && s.enteredAt < s.acquiredAt,
			s.state >= studyReading && s.claimedAt < s.enteredAt,
			s.state == studyFinal && s.finalAt < s.claimedAt:
			return fmt.Errorf("%s timestamps out of order", studyID(si))
		case s.state == studyReading:
			r := &st.emps[s.by]
			if r.status != busy || r.task != taskRead || r.taskStudy != si {
				return fmt.Errorf("%s reading but %s is not reading it", studyID(si), r.cfg.Name)
			}
		}
	}
	if listed != len(d.worklist) {
		return fmt.Errorf("worklist has %d entries, %d studies unread or reading", len(d.worklist), listed)
	}
	for i := range st.emps {
		emp := &st.emps[i]
		if emp.task == taskRead && (emp.cfg.Role != Radiologist || d.studies[emp.taskStudy].state != studyReading ||
			d.studies[emp.taskStudy].by != i) {
			return fmt.Errorf("%s reading task inconsistent", emp.cfg.Name)
		}
	}
	return nil
}

// --- observable projections -------------------------------------------------

const (
	worklistItemsCap = 300
	recentReportsCap = 50
)

func (st *state) studyItem(si int, now simtime.Time) api.WorklistItem {
	s := &st.dem.studies[si]
	o := &st.dem.orders[s.order]
	x := &st.cfg.Catalog[s.exam]
	it := api.WorklistItem{Study: studyID(si), Patient: patientID(o.patient), Modality: x.Modality,
		ExamName: x.Name, Priority: s.priority.String(), Acquired: st.cal.ISO(s.acquiredAt)}
	end := now
	switch s.state {
	case studyAcquired:
		it.State = api.StudyAcquired
	case studyUnread:
		it.State = api.StudyUnread
	case studyReading:
		it.State, it.Radiologist = api.StudyReading, st.emps[s.by].cfg.Name
		it.ReadingMin = int64(now.Sub(s.claimedAt) / simtime.Minute)
	case studyFinal:
		it.State, it.Radiologist = api.StudyFinal, st.emps[s.by].cfg.Name
		end = s.finalAt
		it.TATMin = int64(end.Sub(s.acquiredAt) / simtime.Minute)
	}
	it.AgeMin = int64(end.Sub(s.acquiredAt) / simtime.Minute)
	it.Overdue = end.Sub(s.acquiredAt) > st.cfg.Policies.TATTarget[s.priority]
	return it
}

func median(xs []int64) int64 {
	if len(xs) == 0 {
		return 0
	}
	slices.Sort(xs)
	return xs[len(xs)/2]
}

func pct90(xs []int64) int64 {
	if len(xs) == 0 {
		return 0
	}
	slices.Sort(xs)
	return xs[len(xs)*9/10]
}

// observeWorklist fills the worklist: unread and reading studies in policy
// order, recent reports, backlog by modality and priority, and turnaround.
func (st *state) observeWorklist(o *api.Observation, now simtime.Time) {
	d := &st.dem
	w := &o.Worklist
	w.Policy = st.cfg.Policies.Worklist
	for _, m := range st.modalities() {
		w.ByModality = append(w.ByModality, api.Backlog{Key: m})
	}
	for p := Routine; p < numPriorities; p++ {
		w.ByPriority = append(w.ByPriority, api.Backlog{Key: p.String()})
		w.TargetMin = append(w.TargetMin, api.PriorityTAT{Priority: p.String(),
			Minutes: int64(st.cfg.Policies.TATTarget[p] / simtime.Minute)})
	}
	modIdx := func(m string) int {
		for k := range w.ByModality {
			if w.ByModality[k].Key == m {
				return k
			}
		}
		return 0
	}
	var reading []int
	for _, si := range d.worklist {
		if d.studies[si].state == studyReading {
			reading = append(reading, si)
		}
	}
	for _, si := range append(reading, st.worklistOrder()...) {
		s := &d.studies[si]
		it := st.studyItem(si, now)
		bm, bp := &w.ByModality[modIdx(it.Modality)], &w.ByPriority[s.priority]
		if s.state == studyReading {
			w.Reading, bm.Reading, bp.Reading = w.Reading+1, bm.Reading+1, bp.Reading+1
		} else {
			w.Unread, bm.Unread, bp.Unread = w.Unread+1, bm.Unread+1, bp.Unread+1
		}
		if s.priority == Urgent {
			w.Urgent++
		}
		w.OldestMin = max(w.OldestMin, it.AgeMin)
		if len(w.Items) < worklistItemsCap {
			w.Items = append(w.Items, it)
		}
	}
	w.FinalToday, w.FinalTotal = d.flowToday.reportsFinal, d.flowTotal.reportsFinal
	w.BreachToday = d.flowToday.breaches

	// Reports finalized today, newest first (finals is in finalization
	// order): the recent list and the wait and read-time medians.
	today := st.cal.DayIndex(now)
	var waits, reads []int64
	for k := len(d.finals) - 1; k >= 0; k-- {
		si := d.finals[k]
		s := &d.studies[si]
		if len(w.Recent) < recentReportsCap {
			w.Recent = append(w.Recent, st.studyItem(si, now))
		}
		if st.cal.DayIndex(s.finalAt) != today {
			if len(w.Recent) >= recentReportsCap {
				break
			}
			continue
		}
		waits = append(waits, int64(s.claimedAt.Sub(s.enteredAt)/simtime.Minute))
		reads = append(reads, int64(s.finalAt.Sub(s.claimedAt)/simtime.Minute))
	}
	w.WaitMedianMin, w.ReadMedianMin = median(waits), median(reads)

	// TAT over studies acquired today, unfinished ones at their age (censored).
	if today < 0 || today >= int64(len(d.days)) {
		return
	}
	var tats []int64
	byPri := make([][]int64, numPriorities)
	for _, si := range d.days[today].studies {
		s := &d.studies[si]
		end := now
		if s.state == studyFinal {
			end = s.finalAt
		}
		m := int64(end.Sub(s.acquiredAt) / simtime.Minute)
		tats = append(tats, m)
		byPri[s.priority] = append(byPri[s.priority], m)
	}
	w.AcquiredToday = len(tats)
	w.TATMedianMin, w.TATP90Min = median(tats), pct90(tats)
	for p := Routine; p < numPriorities; p++ {
		w.TATByPriority = append(w.TATByPriority, api.PriorityTAT{Priority: p.String(), Count: len(byPri[p]), Minutes: median(byPri[p])})
	}
}
