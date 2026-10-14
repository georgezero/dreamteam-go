package sim

import (
	"fmt"
	"slices"

	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Processor supplies the domain behavior the engine drives. The simulation
// state implements it; tests use small synthetic worlds.
type Processor interface {
	// IsStale reports whether ev targets an entity whose generation changed.
	IsStale(ev Event) bool
	// Handle applies one event. It may push events at the current time only
	// with a class ≥ ev.Class, or at any later time.
	Handle(e *Engine, ev Event)
	// Dispatch matches waiting work to free resources once per transaction.
	// It may push events only strictly after the current time.
	Dispatch(e *Engine)
	// CheckInvariants returns an error describing any violated invariant.
	CheckInvariants() error
}

// Observer is optionally implemented by a Processor whose bookkeeping
// watches the simulation without changing it (metrics, alerts).
//
// Passive events are handled like any other but do not count as activity:
// a transaction whose only non-stale events are passive runs no dispatch,
// so pushing or handling one cannot change an outcome. AfterTransaction
// runs once after every transaction that handled a non-stale event, after
// dispatch; like dispatch it may push events only strictly after now.
type Observer interface {
	Passive(ev Event) bool
	AfterTransaction(e *Engine)
}

// StopReason says why RunUntil returned.
type StopReason int

const (
	ReachedTarget StopReason = iota // clock advanced to the target
	Stopped                         // stop condition held after a transaction
)

// InvariantError reports a broken engine or domain invariant. The engine
// panics with it: invariant failures are bugs, not gameplay.
type InvariantError struct{ Msg string }

func (e *InvariantError) Error() string { return "sim: invariant violated: " + e.Msg }

func fail(format string, args ...any) {
	panic(&InvariantError{Msg: fmt.Sprintf(format, args...)})
}

type phase uint8

const (
	phaseIdle phase = iota
	phaseHandling
	phaseDispatch
)

// DefaultMaxEventsPerTimestamp bounds one transaction (SIMULATION_DESIGN §3.4).
const DefaultMaxEventsPerTimestamp = 100_000

// Engine is the discrete-event core: a clock, a priority queue and the
// timestamp-transaction loop. It is single-threaded and never reads the
// wall clock.
type Engine struct {
	now   simtime.Time
	seq   uint64
	q     queue
	proc  Processor
	obs   Observer // proc as an Observer, or nil
	phase phase
	class Class // class being processed during phaseHandling

	// MaxEventsPerTimestamp bounds one transaction; exceeding it is an
	// invariant failure.
	MaxEventsPerTimestamp int
	// CheckInvariants runs Processor.CheckInvariants after every transaction.
	CheckInvariants bool

	handled      uint64
	transactions uint64
}

// NewEngine returns an engine whose clock starts at start.
func NewEngine(p Processor, start simtime.Time) *Engine {
	e := &Engine{now: start, proc: p, MaxEventsPerTimestamp: DefaultMaxEventsPerTimestamp}
	e.obs, _ = p.(Observer)
	return e
}

// Now returns the simulated clock.
func (e *Engine) Now() simtime.Time { return e.now }

// Pending returns the number of queued events, including stale ones.
func (e *Engine) Pending() int { return e.q.Len() }

// Handled returns the number of non-stale events processed so far.
func (e *Engine) Handled() uint64 { return e.handled }

// Transactions returns the number of completed timestamp transactions.
func (e *Engine) Transactions() uint64 { return e.transactions }

// NextAt returns the time of the earliest pending event.
func (e *Engine) NextAt() (simtime.Time, bool) {
	if e.q.Len() == 0 {
		return 0, false
	}
	return e.q.peek().At, true
}

// Push schedules ev, assigning its sequence number, and returns the stored
// event. Pushing into the past, behind the class being processed, or at the
// current time during dispatch is an invariant failure.
func (e *Engine) Push(ev Event) Event {
	if ev.Class >= numClasses {
		fail("event kind %d has invalid class %d", ev.Kind, ev.Class)
	}
	if ev.At < e.now {
		fail("event kind %d scheduled at %d, before now %d", ev.Kind, ev.At, e.now)
	}
	if ev.At == e.now {
		switch e.phase {
		case phaseHandling:
			if ev.Class < e.class {
				fail("event kind %d pushed at now with class %d while processing class %d", ev.Kind, ev.Class, e.class)
			}
		case phaseDispatch:
			fail("dispatch pushed event kind %d at the current time", ev.Kind)
		}
	}
	e.seq++
	ev.Seq = e.seq
	e.q.push(ev)
	return ev
}

// RunUntil processes every transaction with time ≤ target, then sets the
// clock to target. After each transaction that handled at least one
// non-stale event, stop (if non-nil) is consulted; if it returns true the
// clock stays at that transaction's time and RunUntil returns Stopped.
//
// Calling RunUntil(a) then RunUntil(b) is equivalent to RunUntil(b) for any
// a ≤ b: advancing the clock without events changes nothing.
func (e *Engine) RunUntil(target simtime.Time, stop func() bool) StopReason {
	if e.phase != phaseIdle {
		fail("RunUntil called re-entrantly")
	}
	if target < e.now {
		fail("RunUntil target %d before now %d", target, e.now)
	}
	for e.q.Len() > 0 && e.q.peek().At <= target {
		if e.transaction() && stop != nil && stop() {
			return Stopped
		}
	}
	e.now = target
	return ReachedTarget
}

// transaction processes all events at the earliest pending time, class by
// class, then dispatches once if anything non-stale and not passive
// happened. It reports whether any non-stale event was handled.
func (e *Engine) transaction() bool {
	t := e.q.peek().At
	e.now = t
	n := 0
	handledAny, active := false, false
	e.phase = phaseHandling
	for e.q.Len() > 0 && e.q.peek().At == t {
		ev := e.q.pop()
		n++
		if n > e.MaxEventsPerTimestamp {
			e.phase = phaseIdle
			fail("more than %d events at time %d (administrative cycle?)", e.MaxEventsPerTimestamp, t)
		}
		if e.proc.IsStale(ev) {
			continue
		}
		e.class = ev.Class
		e.proc.Handle(e, ev)
		e.handled++
		handledAny = true
		active = active || e.obs == nil || !e.obs.Passive(ev)
	}
	if active {
		e.phase = phaseDispatch
		e.proc.Dispatch(e)
	}
	if handledAny && e.obs != nil {
		e.phase = phaseDispatch
		e.obs.AfterTransaction(e)
	}
	e.phase = phaseIdle
	e.transactions++
	if e.CheckInvariants {
		if err := e.proc.CheckInvariants(); err != nil {
			fail("after transaction at %d: %v", t, err)
		}
	}
	return handledAny
}

// PendingEvents returns a copy of the queue in processing order, for
// snapshots and debugging. It is never part of an observation.
func (e *Engine) PendingEvents() []Event {
	out := slices.Clone(e.q.h)
	slices.SortFunc(out, func(a, b Event) int {
		if a.before(b) {
			return -1
		}
		if b.before(a) {
			return 1
		}
		return 0
	})
	return out
}
