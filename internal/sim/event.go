package sim

import "github.com/georgezero/dreamteam-go/internal/simtime"

// Class orders events that share a timestamp (SIMULATION_DESIGN §3.3).
type Class uint8

const (
	ClassDayBoundary  Class = iota // 00:00: shifts, demand scheduling, daily accruals
	ClassCapacityUp                // open, shift start, break end, reading block start, repair
	ClassCompletion                // registration/exam/read/documentation end
	ClassArrival                   // patient arrival, walk-in, order received
	ClassDeadline                  // missed-slot detection, patience, cancellation, wakes
	ClassCapacityDown              // break start, shift end, block end, close, failure
	ClassFinancial                 // claim submit, adjudication, payment
	ClassDayClose                  // 23:59:59 accounting close
	numClasses
)

// Kind identifies an event's handler. Domain kinds are defined alongside
// their handlers.
type Kind uint16

// Event is a plain data record; the queue holds no closures, so pending
// events can be serialized and replayed.
type Event struct {
	At    simtime.Time
	Class Class
	Seq   uint64 // assigned by Engine.Push; unique, gives a total order
	Kind  Kind
	Gen   uint32 // target entity generation at push time; stale if it changed
	A, B  int64  // entity references or small arguments, meaning per Kind
}

// before reports whether e sorts before f by (At, Class, Seq).
func (e Event) before(f Event) bool {
	if e.At != f.At {
		return e.At < f.At
	}
	if e.Class != f.Class {
		return e.Class < f.Class
	}
	return e.Seq < f.Seq
}

// queue is a binary min-heap of events ordered by (At, Class, Seq).
type queue struct {
	h []Event
}

func (q *queue) Len() int { return len(q.h) }

func (q *queue) peek() Event { return q.h[0] }

func (q *queue) push(e Event) {
	q.h = append(q.h, e)
	i := len(q.h) - 1
	for i > 0 {
		p := (i - 1) / 2
		if !q.h[i].before(q.h[p]) {
			break
		}
		q.h[i], q.h[p] = q.h[p], q.h[i]
		i = p
	}
}

func (q *queue) pop() Event {
	top := q.h[0]
	n := len(q.h) - 1
	q.h[0] = q.h[n]
	q.h = q.h[:n]
	i := 0
	for {
		l := 2*i + 1
		if l >= n {
			break
		}
		m := l
		if r := l + 1; r < n && q.h[r].before(q.h[l]) {
			m = r
		}
		if !q.h[m].before(q.h[i]) {
			break
		}
		q.h[i], q.h[m] = q.h[m], q.h[i]
		i = m
	}
	return top
}
