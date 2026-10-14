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
	cov    Coverage     // today's planned coverage
	open   bool         // between scheduled open and scheduled close
	opened bool         // the practice opened today (rooms may report unstaffed)
	// closing: after scheduled close while admitted patients or on-site
	// tasks remain; opClosedAt: today's operational close (0 if not yet).
	closing    bool
	opClosedAt simtime.Time

	rooms []room
	emps  []employee
	dem   demandState

	// dropOrder, if set (tests only), suppresses exogenous orders that
	// would arrive at or after a time: a change to future hidden demand.
	dropOrder func(at simtime.Time) bool
	// forceNoShow, if set (tests only), makes the patients of matching
	// appointments never come: a change to hidden latent behavior.
	forceNoShow func(start simtime.Time) bool
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
		st.rooms = append(st.rooms, room{cfg: &cfg.Rooms[i], visit: -1})
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
	kindArrive     // a scheduled patient arrives
	kindGrace      // start + grace: not arrived means missed
	kindWalkIn     // a walk-in arrives
	kindPatience   // a waiting walk-in gives up
	kindDayClose   // 23:59:59 accounting close
	kindNoShowRebook
	kindRegEnd  // registration finished
	kindExamEnd // exam finished; the patient departs
	kindDocEnd  // technologist documentation finished
	kindWake    // eligibility wake for a room (§3.4 rule 4)
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
	case kindShiftStart, kindShiftEnd, kindBreakStart, kindBlockStart, kindBlockEnd, kindRosterEdge, kindDocEnd:
		return ev.Gen != st.emps[ev.A].gen
	case kindBreakEnd:
		emp := &st.emps[ev.A]
		return ev.Gen != emp.gen || emp.status != onBreak || emp.breakUntil != ev.At
	case kindWake:
		return ev.Gen != st.rooms[ev.A].wakeGen
	case kindCancel, kindGrace:
		return st.dem.appts[ev.A].state != apptBooked
	case kindArrive:
		s := st.dem.appts[ev.A].state
		return s != apptBooked && s != apptMissed
	case kindPatience:
		s := st.dem.visits[ev.A].state
		return s != visitArrived && s != visitWaiting
	case kindNoShowRebook:
		return st.dem.orders[ev.A].state != orderNoShow
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
		st.scheduledClose(e)
	case kindShiftStart, kindShiftEnd, kindBreakStart, kindBreakEnd, kindBlockStart, kindBlockEnd:
		st.staffEvent(e, ev)
	case kindRosterEdge:
	case kindOrder:
		st.receiveOrder(e, unpackOrigin(ev.A, ev.B))
	case kindCancel:
		st.cancel(e, int(ev.A))
	case kindArrive:
		st.arrive(e, int(ev.A))
	case kindGrace:
		st.grace(e, int(ev.A))
	case kindWalkIn:
		st.walkIn(e, ev.A, ev.B>>32, ev.B&0xffffffff)
	case kindPatience:
		st.patience(e, int(ev.A))
	case kindDayClose:
		st.accountingClose(e)
	case kindNoShowRebook:
		st.noShowRebook(e, int(ev.A))
	case kindRegEnd:
		st.regEnd(e, int(ev.A))
	case kindExamEnd:
		st.examEnd(e, int(ev.A))
	case kindDocEnd:
		st.taskDone(e, int(ev.A))
	case kindWake:
	}
}

func (st *state) dayStart(e *Engine, day int64) {
	e.Push(Event{At: st.cal.StartOfDay(day + 1), Class: ClassDayBoundary, Kind: kindDayStart, A: day + 1})
	st.day = day
	st.opened, st.closing, st.opClosedAt = false, false, 0
	if st.cal.Weekday(e.Now()) == simtime.Monday {
		for i := range st.emps {
			st.emps[i].weekPaid = 0
		}
	}
	st.plan = st.cfg.Plan(st.cal.Date(e.Now()))
	st.cov = st.plan.Coverage()
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
	st.dem.flowToday = flowCounts{}
	if day == st.startDay {
		st.dem.flowTotal = flowCounts{}
	}
	st.dem.dayStarted = false
	e.Push(Event{At: st.cal.At(day, simtime.Day-1), Class: ClassDayClose, Kind: kindDayClose})
	st.ensureDays(day + st.cfg.Policies.BookingHorizonDays)
	st.releaseRecalls(e)
	st.scheduleDemand(e, day)
	st.startAppointments(e)
	st.scheduleWalkIns(e, day)
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
	st.observeFlow(&o, now)
	st.observePractice(&o, now)
	return o
}
