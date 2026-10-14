package sim

import (
	"encoding/hex"
	"errors"

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
	cal       simtime.Calendar
	startDay  int64 // day index of the start date (day 1 in observations)
	seed      rng.Seed
	name      string
	scenario  string
	log       eventLog
	open      bool
	demo      demoState
	alertSeen uint64 // log total at the last alert, for auto-pause
}

// NewDemo builds the step-4 demo world: a practice that opens and closes on
// schedule and a scripted stream of synthetic visits. Steps 5–8 replace the
// demo source with the real domain (ROADMAP §2).
func NewDemo(seed int64, start simtime.Date) *World {
	st := &state{
		cal:      simtime.NewCalendar(start),
		seed:     rng.SeedFromInt(seed),
		name:     "DreamTeam Imaging Center",
		scenario: "demo",
	}
	w := &World{st: st}
	w.eng = NewEngine(st, 0)
	w.eng.Push(Event{At: 0, Class: ClassDayBoundary, Kind: kindDayStart, A: 0})
	return w
}

// Engine exposes the engine for the session layer.
func (w *World) Engine() *Engine { return w.eng }

// Calendar returns the practice calendar.
func (w *World) Calendar() simtime.Calendar { return w.st.cal }

// LogTotal returns the number of observable log entries emitted so far.
func (w *World) LogTotal() uint64 { return w.st.log.total }

// LastAlert returns the log sequence number of the most recent alert.
func (w *World) LastAlert() uint64 { return w.st.alertSeen }

// LogHash returns the running hash of the observable log.
func (w *World) LogHash() string { return hex.EncodeToString(w.st.log.hash[:]) }

// Observe projects the observable state. recent bounds the log entries
// included.
func (w *World) Observe(recent int) api.Observation {
	st := w.st
	now := w.eng.Now()
	d := st.cal.Date(now)
	return api.Observation{
		Schema:    api.ObservationSchema,
		Now:       now,
		Time:      st.cal.ISO(now),
		Day:       st.cal.DayIndex(now) - st.startDay + 1,
		Weekday:   d.Weekday().String(),
		Date:      d.String(),
		Clock:     st.cal.Clock(now),
		Practice:  api.Practice{Name: st.name, Open: st.open, Scenario: st.scenario},
		LogTotal:  st.log.total,
		RecentLog: st.log.recent(recent),
	}
}

// Event kinds handled by the state.
const (
	kindDayStart Kind = iota + 1
	kindOpen
	kindClose
	kindDemoArrival
	kindDemoDone
)

func (st *state) emit(at simtime.Time, sev api.Severity, cat, text string) {
	st.log.emit(st.cal, at, sev, cat, text)
	if sev == api.Alert {
		st.alertSeen = st.log.total
	}
}

func (st *state) IsStale(Event) bool { return false }

func (st *state) Handle(e *Engine, ev Event) {
	switch ev.Kind {
	case kindDayStart:
		st.dayStart(e, ev.A)
	case kindOpen:
		st.open = true
		st.emit(e.Now(), api.Info, "practice", "Practice opens")
	case kindClose:
		st.open = false
		st.emit(e.Now(), api.Info, "practice", "Practice closes")
	case kindDemoArrival, kindDemoDone:
		st.demoHandle(e, ev)
	}
}

func (st *state) dayStart(e *Engine, day int64) {
	e.Push(Event{At: st.cal.StartOfDay(day + 1), Class: ClassDayBoundary, Kind: kindDayStart, A: day + 1})
	wd := st.cal.Weekday(e.Now())
	hours, ok := demoHours[wd]
	if !ok {
		return
	}
	open := st.cal.At(day, hours[0])
	closeAt := st.cal.At(day, hours[1])
	e.Push(Event{At: open, Class: ClassCapacityUp, Kind: kindOpen})
	e.Push(Event{At: closeAt, Class: ClassCapacityDown, Kind: kindClose})
	st.demoSchedule(e, day, open, closeAt)
}

func (st *state) Dispatch(*Engine) {}

func (st *state) CheckInvariants() error {
	if st.demo.inside < 0 {
		return errors.New("demo: negative patients inside")
	}
	return nil
}
