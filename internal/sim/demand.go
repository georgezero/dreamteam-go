package sim

import (
	"fmt"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/rng"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Demand and booking (SIMULATION_DESIGN §5.1–5.3).
//
// Orders are exogenous: at each day boundary the day's order counts and
// arrival seconds are drawn from origin-keyed hashes and queued as pending
// events that carry only the origin key. Nothing about a pending order is
// observable, and it has no public ID. When the event fires the order is
// disclosed: it gets its IDs, its attributes are drawn (again keyed by
// origin, so booking cannot change them) and booking runs.

// originKey identifies an exogenous order independently of public IDs and
// player actions: (day, source, hour block, k). Opening-schedule orders use
// day = -1, source = -1, block = -1.
type originKey struct{ day, src, block, k int64 }

func (o originKey) keys(extra ...int64) []int64 {
	return append([]int64{o.day, o.src, o.block, o.k}, extra...)
}

// Event argument packing for kindOrder: A = day, B = src<<40 | block<<32 | k.
func packOrigin(o originKey) (a, b int64) { return o.day, o.src<<40 | o.block<<32 | o.k }

func unpackOrigin(a, b int64) originKey {
	return originKey{day: a, src: b >> 40, block: (b >> 32) & 0xff, k: b & 0xffffffff}
}

type orderState uint8

const (
	orderBooked    orderState = iota // holds an appointment
	orderRecall                      // wanted date beyond the booking horizon
	orderLost                        // declined or no capacity: demand lost
	orderCancelled                   // cancelled, no new date requested
	numOrderStates
)

type order struct {
	key       originKey
	patient   int64 // public patient number
	source    int   // -1 for the opening schedule
	exam      int
	priority  Priority
	window    Window
	received  simtime.Time
	earliest  int64 // day index of the earliest wanted date
	tolerance int64 // days past earliest the patient accepts
	attempt   int64
	state     orderState
	appt      int // current or last appointment, -1 if none
}

type apptState uint8

const (
	apptBooked apptState = iota
	apptCancelled
)

type appt struct {
	order    int
	room     int
	day      int64
	slot     int
	attempt  int64
	state    apptState
	bookedAt simtime.Time
}

// Why a template slot cannot be booked.
const (
	slotOK = iota
	slotUnstaffed
	slotNoSupervision
)

type slot struct {
	SlotTemplate
	why  uint8 // slotOK when bookable
	appt int   // -1 when free
}

type daySlots struct {
	rooms  [][]slot
	booked int
}

// modalityCounts are demand event counters for one modality: orders
// received, appointments booked (new, recall and rebookings), orders put on
// the recall list, orders lost and appointments cancelled.
type modalityCounts struct {
	received, booked, recall, lost, cancelled int
}

type demandState struct {
	orders  []order
	appts   []appt
	days    []*daySlots // by day index, materialized up to today + horizon
	recall  []int       // orders waiting for their date to enter the horizon
	poisson map[uint32]*rng.Poisson
	catIdx  map[string]int

	patients int64
	today    map[string]*modalityCounts // reset each day; read by modality key only
	total    map[string]*modalityCounts // since the start date
}

func (st *state) initDemand() {
	d := &st.dem
	d.poisson = map[uint32]*rng.Poisson{}
	d.today = map[string]*modalityCounts{}
	d.total = map[string]*modalityCounts{}
	for _, m := range st.modalities() {
		d.today[m], d.total[m] = &modalityCounts{}, &modalityCounts{}
	}
}

// modalities returns the room modalities in room order, without repeats.
func (st *state) modalities() []string {
	var out []string
	for _, r := range st.cfg.Rooms {
		seen := false
		for _, m := range out {
			seen = seen || m == r.Modality
		}
		if !seen {
			out = append(out, r.Modality)
		}
	}
	return out
}

func (st *state) count(exam int, f func(*modalityCounts)) {
	m := st.cfg.Catalog[exam].Modality
	f(st.dem.today[m])
	f(st.dem.total[m])
}

// ensureDays materializes slot days up to and including day.
func (st *state) ensureDays(day int64) {
	for int64(len(st.dem.days)) <= day {
		st.dem.days = append(st.dem.days, st.buildDay(int64(len(st.dem.days))))
	}
}

// buildDay lays out a day's template slots and decides which are bookable
// from the roster as known now.
func (st *state) buildDay(day int64) *daySlots {
	date := st.cal.Date(st.cal.StartOfDay(day))
	plan := st.cfg.Plan(date)
	ds := &daySlots{rooms: make([][]slot, len(st.cfg.Rooms))}
	if plan.Hours == nil {
		return ds
	}
	cov := plan.Coverage()
	for r := range st.cfg.Rooms {
		for _, t := range st.cfg.Templates[r][date.Weekday()] {
			s := slot{SlotTemplate: t, appt: -1}
			if ok, why := cov.SlotBookable(r, t.Span, st.cfg.Categories[t.Category].Contrast); !ok {
				s.why = slotUnstaffed
				if why == "no_supervision" {
					s.why = slotNoSupervision
				}
			}
			ds.rooms[r] = append(ds.rooms[r], s)
		}
	}
	return ds
}

// scheduleDemand queues the day's exogenous orders (class DayBoundary).
func (st *state) scheduleDemand(e *Engine, day int64) {
	date := st.cal.Date(st.cal.StartOfDay(day))
	dm := &st.cfg.Demand
	for si := range st.cfg.Sources {
		src := &st.cfg.Sources[si]
		for h := int64(0); h < 24; h++ {
			rate := src.BaseDailyMilli * dm.HourBP[h] / 10_000
			rate = rate * dm.WeekdayBP[date.Weekday()] / 10_000
			rate = rate * dm.SeasonBP[date.Month-1] / 10_000
			if rate <= 0 {
				continue
			}
			n := st.poissonFor(uint32(rate)).Sample(rng.Hash(st.seed, "demand.count", day, int64(si), h))
			for k := 0; k < n; k++ {
				key := originKey{day, int64(si), h, int64(k)}
				sec := rng.Uniform(rng.Hash(st.seed, "order.core", key.keys()...).Lane(0), 3600)
				at := st.cal.At(day, simtime.Duration(h)*simtime.Hour+simtime.Duration(sec))
				if st.dropOrder != nil && st.dropOrder(at) {
					continue
				}
				a, b := packOrigin(key)
				e.Push(Event{At: at, Class: ClassArrival, Kind: kindOrder, A: a, B: b})
			}
		}
	}
}

func (st *state) poissonFor(milli uint32) *rng.Poisson {
	if p, ok := st.dem.poisson[milli]; ok {
		return p
	}
	p, err := rng.NewPoisson(milli)
	if err != nil {
		panic(err)
	}
	st.dem.poisson[milli] = p
	return p
}

// receiveOrder discloses an order: public IDs are allocated here and only
// here, then attributes are drawn by origin key and booking runs.
func (st *state) receiveOrder(e *Engine, key originKey) {
	core := rng.Hash(st.seed, "order.core", key.keys()...)
	ext := rng.Hash(st.seed, "order.ext", key.keys()...)
	pat := rng.Hash(st.seed, "patient", key.keys()...)
	src := &st.cfg.Sources[key.src]

	exam := src.Exams[rng.Weighted(core.Lane(1), src.ExamWeights)]
	pri := Priority(rng.Weighted(core.Lane(2), src.PriorityBP[:]))
	lead := st.cfg.Demand.LeadDays.Sample(core.Lane(3))
	switch pri {
	case Urgent:
		lead = 0
	case Soon:
		lead = min(lead, 3)
	}
	win := Window(rng.Weighted(ext.Lane(0), st.cfg.Demand.WindowBP[:]))
	// ext lane 1 is reserved for the payer (step 10); patient lanes 0 and 2
	// for punctuality and walk-in patience (step 7).
	tol := st.cfg.Behavior.ToleranceDays[pri].Sample(pat.Lane(1))

	st.dem.patients++
	oi := len(st.dem.orders)
	st.dem.orders = append(st.dem.orders, order{
		key: key, patient: st.dem.patients, source: int(key.src), exam: exam, priority: pri,
		window: win, received: e.Now(), earliest: st.day + lead, tolerance: tol, appt: -1,
	})
	st.count(exam, func(c *modalityCounts) { c.received++ })
	o := &st.dem.orders[oi]
	head := fmt.Sprintf("%s %s", orderID(oi), st.examLabel(exam))
	if o.priority != Routine {
		head += " (" + o.priority.String() + ")"
	}
	tail := " · " + src.Name
	if o.earliest > st.day+st.cfg.Policies.BookingHorizonDays {
		o.state = orderRecall
		st.dem.recall = append(st.dem.recall, oi)
		st.count(exam, func(c *modalityCounts) { c.recall++ })
		st.emit(e.Now(), api.Info, "orders", fmt.Sprintf("%s → recall list, due %s%s", head, st.dayLabel(o.earliest), tail))
		return
	}
	st.offer(e, oi, o.earliest, head, tail)
}

// offer runs FirstFit for order oi from day earliest and books, or marks
// the order lost.
func (st *state) offer(e *Engine, oi int, earliest int64, head, tail string) bool {
	o := &st.dem.orders[oi]
	pref, any, hasPref, hasAny := st.firstFit(o, earliest, e.Now())
	var pick slotRef
	switch {
	case hasPref && pref.day-earliest <= o.tolerance:
		pick = pref
	case hasAny && any.day-earliest <= o.tolerance:
		pick = any
	default:
		o.state = orderLost
		st.count(o.exam, func(c *modalityCounts) { c.lost++ })
		why := fmt.Sprintf("no %s slot within %d days", st.cfg.Catalog[o.exam].Modality, st.cfg.Policies.BookingHorizonDays)
		if hasAny {
			why = "declined, earliest slot " + st.dayLabel(any.day)
			if n := any.day - st.day; n > 1 {
				why += fmt.Sprintf(" (%d days)", n)
			}
		}
		st.emit(e.Now(), api.Warn, "orders", head+" → lost: "+why+tail)
		return false
	}
	ai := st.book(e, oi, pick, false)
	st.count(o.exam, func(c *modalityCounts) { c.booked++ })
	st.emit(e.Now(), api.Info, "orders", fmt.Sprintf("%s → booked %s %s (%s)%s",
		head, st.dayLabel(pick.day), simtime.FormatClock(st.slotAt(pick).From), apptID(ai), tail))
	return true
}

type slotRef struct {
	day  int64
	room int
	idx  int
}

func (st *state) slotAt(r slotRef) *slot { return &st.dem.days[r.day].rooms[r.room][r.idx] }

// firstFit finds the earliest free, bookable slot for the order's exam on or
// after day earliest: the first in the patient's preferred window (pref)
// and the first at any time (any). Same-day slots need the booking notice.
func (st *state) firstFit(o *order, earliest int64, now simtime.Time) (pref, any slotRef, hasPref, hasAny bool) {
	cat := st.cfg.Catalog[o.exam].Category
	last := st.day + st.cfg.Policies.BookingHorizonDays
	notBefore := now.Add(st.cfg.Policies.BookingNotice)
	for d := max(earliest, st.day); d <= last; d++ {
		if hasAny && d-earliest > o.tolerance {
			break // a later preferred slot could not be accepted
		}
		ds := st.dem.days[d]
		dayPref, dayAny := slotRef{idx: -1}, slotRef{idx: -1}
		var prefAt, anyAt simtime.Duration
		for r := range ds.rooms {
			for i := range ds.rooms[r] {
				s := &ds.rooms[r][i]
				if s.Category != cat || s.appt >= 0 || s.why != slotOK || st.cal.At(d, s.From) < notBefore {
					continue
				}
				if dayAny.idx < 0 || s.From < anyAt {
					dayAny, anyAt = slotRef{d, r, i}, s.From
				}
				if inWindow(o.window, s.From) && (dayPref.idx < 0 || s.From < prefAt) {
					dayPref, prefAt = slotRef{d, r, i}, s.From
				}
			}
		}
		if !hasAny && dayAny.idx >= 0 {
			any, hasAny = dayAny, true
		}
		if dayPref.idx >= 0 {
			return dayPref, any, true, hasAny
		}
	}
	return pref, any, false, hasAny
}

func inWindow(w Window, at simtime.Duration) bool {
	switch w {
	case Morning:
		return at < 12*simtime.Hour
	case Afternoon:
		return at >= 12*simtime.Hour && at < 16*simtime.Hour
	case Evening:
		return at >= 16*simtime.Hour
	}
	return true
}

// book places order oi in slot r and draws whether this attempt will be
// cancelled (hidden until the cancellation fires).
func (st *state) book(e *Engine, oi int, r slotRef, confirmed bool) int {
	o := &st.dem.orders[oi]
	s := st.slotAt(r)
	if s.appt >= 0 || s.why != slotOK {
		panic("sim: booking an unavailable slot")
	}
	ai := len(st.dem.appts)
	st.dem.appts = append(st.dem.appts, appt{order: oi, room: r.room, day: r.day, slot: r.idx,
		attempt: o.attempt, bookedAt: e.Now()})
	s.appt = ai
	st.dem.days[r.day].booked++
	o.state, o.appt = orderBooked, ai
	if confirmed {
		return ai
	}
	att := rng.Hash(st.seed, "attempt", o.key.keys(o.attempt)...)
	if !rng.Bernoulli(att.Lane(2), st.cfg.Behavior.CancelPPM) {
		return ai
	}
	start := st.cal.At(r.day, s.From)
	lead := simtime.Duration(st.cfg.Behavior.CancelLeadHours.Sample(att.Lane(3))) * simtime.Hour
	at := start.Add(-lead)
	if at <= e.Now() {
		at = e.Now().Add(max(1, start.Sub(e.Now())/2))
	}
	if at < start {
		e.Push(Event{At: at, Class: ClassDeadline, Kind: kindCancel, A: int64(ai)})
	}
	return ai
}

// cancel handles a patient cancellation: the slot is freed at once (later
// orders can take it) and, by the rebook draw, the patient asks for a new
// date.
func (st *state) cancel(e *Engine, ai int) {
	a := &st.dem.appts[ai]
	o := &st.dem.orders[a.order]
	s := &st.dem.days[a.day].rooms[a.room][a.slot]
	a.state = apptCancelled
	s.appt = -1
	st.dem.days[a.day].booked--
	st.count(o.exam, func(c *modalityCounts) { c.cancelled++ })
	head := fmt.Sprintf("%s cancelled (%s %s, %s %s)", apptID(ai), orderID(a.order),
		st.examLabel(o.exam), st.dayLabel(a.day), simtime.FormatClock(s.From))

	ex := rng.Hash(st.seed, "attempt.exam", o.key.keys(a.attempt)...)
	if !rng.Bernoulli(ex.Lane(3), st.cfg.Behavior.RebookPPM) {
		o.state = orderCancelled
		st.emit(e.Now(), api.Info, "orders", head)
		return
	}
	// The patient could not make that day: look from the day after.
	o.attempt++
	st.offer(e, a.order, max(o.earliest, a.day+1, st.day+1), head+"; new date requested", "")
}

// releaseRecalls books recall-list orders whose wanted date has entered the
// booking horizon, in (date, order) order.
func (st *state) releaseRecalls(e *Engine) {
	last := st.day + st.cfg.Policies.BookingHorizonDays
	var due, keep []int
	for _, oi := range st.dem.recall {
		if st.dem.orders[oi].earliest <= last {
			due = append(due, oi)
		} else {
			keep = append(keep, oi)
		}
	}
	st.dem.recall = keep
	// Stable insertion sort by earliest date; order index breaks ties.
	for i := 1; i < len(due); i++ {
		for j := i; j > 0 && st.dem.orders[due[j]].earliest < st.dem.orders[due[j-1]].earliest; j-- {
			due[j], due[j-1] = due[j-1], due[j]
		}
	}
	for _, oi := range due {
		o := &st.dem.orders[oi]
		st.offer(e, oi, o.earliest, fmt.Sprintf("%s %s from recall list", orderID(oi), st.examLabel(o.exam)), "")
	}
}

// loadOpening books the scenario's hand-built opening schedule. Those
// appointments are confirmed (no cancellation draw) and disclosed at start.
func (st *state) loadOpening(e *Engine) {
	for i, op := range st.cfg.Opening {
		day := st.startDay + op.Day
		st.ensureDays(day)
		idx := -1
		for k, s := range st.dem.days[day].rooms[op.Room] {
			if s.From == op.Start {
				idx = k
			}
		}
		st.dem.patients++
		oi := len(st.dem.orders)
		st.dem.orders = append(st.dem.orders, order{
			key: originKey{-1, -1, -1, int64(i)}, patient: st.dem.patients, source: op.Source,
			exam: op.Exam, priority: op.Priority, earliest: day, appt: -1,
		})
		st.book(e, oi, slotRef{day, op.Room, idx}, true)
	}
}

func orderID(i int) string     { return fmt.Sprintf("O%05d", i+1) }
func apptID(i int) string      { return fmt.Sprintf("A%05d", i+1) }
func patientID(n int64) string { return fmt.Sprintf("P%05d", n) }

func (st *state) examLabel(exam int) string {
	x := &st.cfg.Catalog[exam]
	return x.Modality + " " + x.Name
}

// dayLabel formats a day index: "today", "tomorrow" or "Thu Oct 15".
func (st *state) dayLabel(day int64) string {
	switch day - st.day {
	case 0:
		return "today"
	case 1:
		return "tomorrow"
	}
	d := st.cal.Date(st.cal.StartOfDay(day))
	return fmt.Sprintf("%s %s %d", d.Weekday(), d.MonthName(), d.Day)
}

// checkDemand verifies booking invariants: each active appointment holds
// exactly one bookable slot of its exam's category and the slot points
// back; per-day booked counters match; every order is in one state.
func (st *state) checkDemand() error {
	d := &st.dem
	active := 0
	for ai := range d.appts {
		a := &d.appts[ai]
		if a.state != apptBooked {
			continue
		}
		active++
		s := &d.days[a.day].rooms[a.room][a.slot]
		o := &d.orders[a.order]
		switch {
		case s.appt != ai:
			return fmt.Errorf("%s: slot does not point back", apptID(ai))
		case s.why != slotOK:
			return fmt.Errorf("%s: booked in an unbookable slot", apptID(ai))
		case s.Category != st.cfg.Catalog[o.exam].Category:
			return fmt.Errorf("%s: slot category does not admit %s", apptID(ai), st.cfg.Catalog[o.exam].Code)
		case o.state != orderBooked || o.appt != ai:
			return fmt.Errorf("%s: order %s not holding it", apptID(ai), orderID(a.order))
		}
	}
	booked := 0
	for _, ds := range d.days {
		booked += ds.booked
	}
	if booked != active {
		return fmt.Errorf("day counters say %d booked, %d active appointments", booked, active)
	}
	var states [numOrderStates]int
	for _, o := range d.orders {
		states[o.state]++
	}
	if states[orderBooked] != active || states[orderRecall] != len(d.recall) {
		return fmt.Errorf("order states %v vs %d active, %d recall", states, active, len(d.recall))
	}
	return nil
}
