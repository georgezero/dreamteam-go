package sim

import (
	"encoding/hex"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/rng"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// World is one simulation: the engine plus the authoritative state.
type World struct {
	eng *Engine
	st  *state
}

// state is the authoritative simulation state. Everything here is
// unexported: other packages see the world only through api.Observation.
type state struct {
	cfg       *Config
	cal       simtime.Calendar
	startDay  int64 // day index of the start date (day 1 in observations)
	seed      rng.Seed
	log       eventLog
	alertSeen uint64 // log total at the last alert, for auto-pause

	now    simtime.Time // time of the current or last transaction
	day    int64        // current day index
	plan   DayPlan      // today's materialized roster
	open   bool
	opened bool // the practice opened today (rooms may report unstaffed)

	rooms []room
	emps  []employee
	dem   demandState

	// dropOrder, if set (tests only), suppresses exogenous orders that
	// would arrive at or after a time: a change to future hidden demand.
	dropOrder func(at simtime.Time) bool
}

// New builds a world from cfg. The calendar epoch is the start date minus
// the warm-up, so day index WarmupDays is the start date.
func New(cfg *Config, seed rng.Seed) *World {
	st := &state{
		cfg:      cfg,
		cal:      simtime.NewCalendar(cfg.Start.AddDays(-cfg.WarmupDays)),
		startDay: cfg.WarmupDays,
		seed:     seed,
	}
	for i := range cfg.Rooms {
		st.rooms = append(st.rooms, room{cfg: &cfg.Rooms[i]})
	}
	for i := range cfg.Employees {
		st.emps = append(st.emps, employee{cfg: &cfg.Employees[i], shift: -1})
	}
	st.initDemand()
	w := &World{st: st}
	w.eng = NewEngine(st, 0)
	w.eng.Push(Event{At: 0, Class: ClassDayBoundary, Kind: kindDayStart, A: 0})
	st.loadOpening(w.eng)
	return w
}

// Engine exposes the engine for the session layer.
func (w *World) Engine() *Engine { return w.eng }

// Calendar returns the practice calendar.
func (w *World) Calendar() simtime.Calendar { return w.st.cal }

// StartTime returns when play begins: the start date at the start clock.
func (w *World) StartTime() simtime.Time {
	return w.st.cal.At(w.st.startDay, w.st.cfg.StartClock)
}

// LogTotal returns the number of observable log entries emitted so far.
func (w *World) LogTotal() uint64 { return w.st.log.total }

// LastAlert returns the log sequence number of the most recent alert.
func (w *World) LastAlert() uint64 { return w.st.alertSeen }

// LogHash returns the running hash of the observable log.
func (w *World) LogHash() string { return hex.EncodeToString(w.st.log.hash[:]) }

// Event kinds handled by the state.
const (
	kindDayStart Kind = iota + 1
	kindOpen
	kindClose
	kindShiftStart
	kindShiftEnd
	kindBreakStart
	kindBreakEnd
	kindBlockStart
	kindBlockEnd
	kindRosterEdge // a cover window starts or ends; only triggers re-staffing
	kindOrder      // an order is received (disclosed)
	kindCancel     // a patient cancels an appointment
)

func (st *state) emit(at simtime.Time, sev api.Severity, cat, text string) {
	st.log.emit(st.cal, at, sev, cat, text)
	if sev == api.Alert {
		st.alertSeen = st.log.total
	}
}

// IsStale drops employee events from a superseded shift (e.g. after an
// action rewrites today's shift).
func (st *state) IsStale(ev Event) bool {
	switch ev.Kind {
	case kindShiftStart, kindShiftEnd, kindBreakStart, kindBreakEnd, kindBlockStart, kindBlockEnd, kindRosterEdge:
		return ev.Gen != st.emps[ev.A].gen
	case kindCancel:
		return st.dem.appts[ev.A].state != apptBooked
	}
	return false
}

func (st *state) Handle(e *Engine, ev Event) {
	st.now = e.Now()
	switch ev.Kind {
	case kindDayStart:
		st.dayStart(e, ev.A)
	case kindOpen:
		st.open, st.opened = true, true
		st.emit(e.Now(), api.Info, "practice", "Practice opens")
	case kindClose:
		st.open = false
		st.emit(e.Now(), api.Info, "practice", "Practice closes")
	case kindShiftStart, kindShiftEnd, kindBreakStart, kindBreakEnd, kindBlockStart, kindBlockEnd:
		st.staffEvent(e, ev)
	case kindRosterEdge:
	case kindOrder:
		st.receiveOrder(e, unpackOrigin(ev.A, ev.B))
	case kindCancel:
		st.cancel(e, int(ev.A))
	}
}

func (st *state) dayStart(e *Engine, day int64) {
	e.Push(Event{At: st.cal.StartOfDay(day + 1), Class: ClassDayBoundary, Kind: kindDayStart, A: day + 1})
	st.day = day
	st.opened = false
	st.plan = st.cfg.Plan(st.cal.Date(e.Now()))
	if h := st.plan.Hours; h != nil {
		e.Push(Event{At: st.cal.At(day, h.From), Class: ClassCapacityUp, Kind: kindOpen})
		e.Push(Event{At: st.cal.At(day, h.To), Class: ClassCapacityDown, Kind: kindClose})
	}
	for i := range st.emps {
		st.materialize(e, i)
	}

	for _, m := range st.modalities() {
		*st.dem.today[m] = modalityCounts{}
		if day == st.startDay {
			*st.dem.total[m] = modalityCounts{}
		}
	}
	st.ensureDays(day + st.cfg.Policies.BookingHorizonDays)
	st.releaseRecalls(e)
	st.scheduleDemand(e, day)
}

func (st *state) Dispatch(e *Engine) {
	st.now = e.Now()
	st.restaff(e.Now())
}

func (st *state) CheckInvariants() error {
	if err := st.checkStaffing(); err != nil {
		return err
	}
	return st.checkDemand()
}

// Observe projects the observable state. recent bounds the log entries
// included.
func (w *World) Observe(recent int) api.Observation {
	st := w.st
	now := w.eng.Now()
	d := st.cal.Date(now)
	o := api.Observation{
		Schema:  api.ObservationSchema,
		Now:     now,
		Time:    st.cal.ISO(now),
		Day:     st.cal.DayIndex(now) - st.startDay + 1,
		Weekday: d.Weekday().String(),
		Date:    d.String(),
		Clock:   st.cal.Clock(now),
		Practice: api.Practice{Name: st.cfg.Name, Open: st.open, Scenario: st.cfg.ID,
			BookingHorizonDays: st.cfg.Policies.BookingHorizonDays,
			StartOffset:        st.startDay - st.cal.DayIndex(now)},
		LogTotal:  st.log.total,
		RecentLog: st.log.recent(recent),
	}
	st.observeStaff(&o, now)
	st.observeDemand(&o, now)
	return o
}
