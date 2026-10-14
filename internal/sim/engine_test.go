package sim

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/rng"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// toyWorld is a small synthetic processor: jobs arrive, wait for one of a
// few servers, are served, and sometimes cancel each other. It exercises
// same-time pushes, stale events and dispatch.
const (
	kArrive Kind = iota + 1
	kDone
	kCancel
	kEcho  // pushes a same-time follow-up at a higher class
	kNoop  // records only
	kCycle // re-pushes itself at the same time forever
)

type toyWorld struct {
	r        *rng.Stream
	gen      map[int64]uint32 // job generations (lookups only, never ranged)
	waiting  []int64
	free     int
	h        []byte // running digest input
	nextJob  int64
	dispatch int
	busy     int
}

func newToyWorld(seed int64, servers int) *toyWorld {
	return &toyWorld{
		r:    rng.New(rng.SeedFromInt(seed), "toy"),
		gen:  map[int64]uint32{},
		free: servers,
	}
}

func (w *toyWorld) record(now simtime.Time, ev Event) {
	w.h = binary.BigEndian.AppendUint64(w.h, uint64(now))
	w.h = binary.BigEndian.AppendUint16(w.h, uint16(ev.Kind))
	w.h = binary.BigEndian.AppendUint64(w.h, uint64(ev.A))
}

func (w *toyWorld) IsStale(ev Event) bool {
	if ev.Kind == kDone || ev.Kind == kCancel {
		return w.gen[ev.A] != ev.Gen
	}
	return false
}

func (w *toyWorld) Handle(e *Engine, ev Event) {
	w.record(e.Now(), ev)
	switch ev.Kind {
	case kArrive:
		w.nextJob++
		id := w.nextJob
		w.waiting = append(w.waiting, id)
		// Next arrival 0–5 minutes later; 0 means a same-time arrival.
		e.Push(Event{At: e.Now().Add(simtime.Duration(w.r.Uniform(300))), Class: ClassArrival, Kind: kArrive})
		if w.r.Bernoulli(150_000) {
			e.Push(Event{At: e.Now(), Class: ClassDeadline, Kind: kEcho, A: id})
		}
		if w.r.Bernoulli(100_000) && id > 1 {
			// Cancel an earlier job's pending completion, maybe later.
			victim := id - 1 - int64(w.r.Uniform(uint64(id-1)))
			e.Push(Event{At: e.Now().Add(simtime.Duration(w.r.Uniform(120))), Class: ClassDeadline, Kind: kCancel, A: victim, Gen: w.gen[victim]})
		}
	case kDone:
		w.free++
		w.busy--
		w.gen[ev.A]++
	case kCancel:
		// Bumping the generation makes the job's pending kDone stale; the
		// server it held is released here instead.
		if w.gen[ev.A]%2 == 1 { // odd generation = in service
			w.gen[ev.A]++
			w.free++
			w.busy--
		}
	case kEcho:
		e.Push(Event{At: e.Now(), Class: ClassCapacityDown, Kind: kNoop, A: ev.A})
	case kNoop:
	case kCycle:
		e.Push(Event{At: e.Now(), Class: ev.Class, Kind: kCycle})
	}
}

func (w *toyWorld) Dispatch(e *Engine) {
	w.dispatch++
	for w.free > 0 && len(w.waiting) > 0 {
		id := w.waiting[0]
		w.waiting = w.waiting[1:]
		w.free--
		w.busy++
		w.gen[id]++ // odd: in service
		d := simtime.Duration(1 + w.r.Uniform(600))
		e.Push(Event{At: e.Now().Add(d), Class: ClassCompletion, Kind: kDone, A: id, Gen: w.gen[id]})
	}
}

func (w *toyWorld) CheckInvariants() error {
	if w.free < 0 || w.busy < 0 {
		return errors.New("negative servers")
	}
	return nil
}

func (w *toyWorld) digest(e *Engine) string {
	h := sha256.New()
	h.Write(w.h)
	var b []byte
	b = binary.BigEndian.AppendUint64(b, uint64(e.Now()))
	b = binary.BigEndian.AppendUint64(b, uint64(w.free))
	b = binary.BigEndian.AppendUint64(b, uint64(len(w.waiting)))
	b = binary.BigEndian.AppendUint64(b, uint64(w.dispatch))
	for _, ev := range e.PendingEvents() {
		b = binary.BigEndian.AppendUint64(b, uint64(ev.At))
		b = append(b, byte(ev.Class))
		b = binary.BigEndian.AppendUint64(b, ev.Seq)
	}
	h.Write(b)
	return hex.EncodeToString(h.Sum(nil))
}

func newToyRun(seed int64) (*toyWorld, *Engine) {
	w := newToyWorld(seed, 3)
	e := NewEngine(w, 0)
	e.CheckInvariants = true
	e.Push(Event{At: 0, Class: ClassArrival, Kind: kArrive})
	return w, e
}

const toyHorizon = simtime.Time(2 * simtime.Day)

func TestChunkingInvariance(t *testing.T) {
	for seed := int64(1); seed <= 10; seed++ {
		w, e := newToyRun(seed)
		e.RunUntil(toyHorizon, nil)
		want := w.digest(e)
		chunker := rng.New(rng.SeedFromInt(seed), "chunks")
		for trial := 0; trial < 100; trial++ {
			w2, e2 := newToyRun(seed)
			for e2.Now() < toyHorizon {
				var step simtime.Duration
				switch chunker.Uniform(4) {
				case 0:
					step = 0 // zero-length chunks must be harmless
				case 1:
					step = simtime.Duration(chunker.Uniform(5))
				case 2:
					step = simtime.Duration(chunker.Uniform(3600))
				default:
					step = simtime.Duration(chunker.Uniform(uint64(simtime.Day)))
				}
				target := min(e2.Now().Add(step), toyHorizon)
				e2.RunUntil(target, nil)
			}
			if got := w2.digest(e2); got != want {
				t.Fatalf("seed %d trial %d: chunked digest differs", seed, trial)
			}
		}
	}
}

func TestStopConditionResumes(t *testing.T) {
	w, e := newToyRun(5)
	e.RunUntil(toyHorizon, nil)
	want := w.digest(e)

	w2, e2 := newToyRun(5)
	stops := 0
	for {
		n := e2.Transactions()
		r := e2.RunUntil(toyHorizon, func() bool { return e2.Transactions()%97 == 0 && e2.Transactions() != n })
		if r == ReachedTarget {
			break
		}
		stops++
		if next, ok := e2.NextAt(); ok && next < e2.Now() {
			t.Fatal("stopped with an earlier event pending")
		}
	}
	if stops == 0 {
		t.Fatal("stop condition never fired")
	}
	if got := w2.digest(e2); got != want {
		t.Fatal("stop/resume changed the outcome")
	}
}

func TestStaleEventsDoNotInterfere(t *testing.T) {
	w, e := newToyRun(7)
	e.RunUntil(toyHorizon, nil)

	w2, e2 := newToyRun(7)
	inj := rng.New(rng.SeedFromInt(7), "inject")
	for e2.Now() < toyHorizon {
		target := min(e2.Now().Add(simtime.Duration(inj.Uniform(1800))), toyHorizon)
		// Job -1 never has generation 99, so this event is always stale: it
		// must be discarded without handling or triggering dispatch.
		e2.Push(Event{At: target, Class: ClassCompletion, Kind: kDone, A: -1, Gen: 99})
		e2.RunUntil(target, nil)
	}
	// Injected events consume sequence numbers, so compare the event log and
	// world state rather than the full digest (which includes pending Seqs).
	if string(w2.h) != string(w.h) || w2.dispatch != w.dispatch ||
		w2.free != w.free || len(w2.waiting) != len(w.waiting) || e2.Handled() != e.Handled() {
		t.Fatal("stale events changed the outcome")
	}
}

func TestOrderingByTimeClassSeq(t *testing.T) {
	var order []int64
	p := &funcProc{handle: func(e *Engine, ev Event) { order = append(order, ev.A) }}
	e := NewEngine(p, 0)
	e.Push(Event{At: 10, Class: ClassCapacityDown, A: 4})
	e.Push(Event{At: 10, Class: ClassCapacityUp, A: 1})
	e.Push(Event{At: 5, Class: ClassDayClose, A: 0})
	e.Push(Event{At: 10, Class: ClassArrival, A: 2})
	e.Push(Event{At: 10, Class: ClassArrival, A: 3}) // same class: by Seq
	e.Push(Event{At: 11, Class: ClassDayBoundary, A: 5})
	e.RunUntil(100, nil)
	want := []int64{0, 1, 2, 3, 4, 5}
	if len(order) != len(want) {
		t.Fatalf("order %v", order)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order %v, want %v", order, want)
		}
	}
	if e.Now() != 100 {
		t.Fatalf("clock %d, want 100", e.Now())
	}
}

func TestSameTimeHigherClassJoinsTransaction(t *testing.T) {
	var seen []int64
	dispatches := 0
	p := &funcProc{
		handle: func(e *Engine, ev Event) {
			seen = append(seen, ev.A)
			if ev.A == 1 {
				e.Push(Event{At: e.Now(), Class: ClassDeadline, A: 3})
				e.Push(Event{At: e.Now(), Class: ClassArrival, A: 2}) // same class
			}
		},
		dispatch: func(*Engine) { dispatches++ },
	}
	e := NewEngine(p, 0)
	e.Push(Event{At: 7, Class: ClassArrival, A: 1})
	e.RunUntil(7, nil)
	if len(seen) != 3 || seen[0] != 1 || seen[1] != 2 || seen[2] != 3 {
		t.Fatalf("seen %v", seen)
	}
	if dispatches != 1 || e.Transactions() != 1 {
		t.Fatalf("dispatches %d transactions %d", dispatches, e.Transactions())
	}
}

func TestEngineRejectsBadPushes(t *testing.T) {
	cases := map[string]func(e *Engine){
		"backward class": func(e *Engine) {
			e.Push(Event{At: 1, Class: ClassDeadline, Kind: 1})
		},
		"past": func(e *Engine) {
			e.Push(Event{At: 1, Class: ClassArrival, Kind: 2})
		},
		"dispatch at now": func(e *Engine) {
			e.Push(Event{At: 1, Class: ClassArrival, Kind: 3})
		},
	}
	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			p := &funcProc{}
			switch name {
			case "backward class":
				p.handle = func(e *Engine, ev Event) {
					if ev.Kind == 1 {
						e.Push(Event{At: e.Now(), Class: ClassArrival})
					}
				}
			case "past":
				p.handle = func(e *Engine, ev Event) { e.Push(Event{At: e.Now() - 1, Class: ClassArrival}) }
			case "dispatch at now":
				p.dispatch = func(e *Engine) { e.Push(Event{At: e.Now(), Class: ClassDayClose}) }
			}
			e := NewEngine(p, 0)
			setup(e)
			expectInvariantPanic(t, func() { e.RunUntil(10, nil) })
		})
	}
}

func TestAdministrativeCycleBounded(t *testing.T) {
	w := newToyWorld(1, 1)
	e := NewEngine(w, 0)
	e.MaxEventsPerTimestamp = 1000
	e.Push(Event{At: 3, Class: ClassArrival, Kind: kCycle})
	expectInvariantPanic(t, func() { e.RunUntil(10, nil) })
}

func TestStaleOnlyTimestampSkipsDispatch(t *testing.T) {
	dispatches := 0
	p := &funcProc{
		stale:    func(ev Event) bool { return ev.Gen != 0 },
		dispatch: func(*Engine) { dispatches++ },
	}
	e := NewEngine(p, 0)
	e.Push(Event{At: 5, Class: ClassCompletion, Gen: 1})
	e.Push(Event{At: 6, Class: ClassCompletion})
	e.RunUntil(10, nil)
	if dispatches != 1 || e.Handled() != 1 {
		t.Fatalf("dispatches %d handled %d", dispatches, e.Handled())
	}
}

func TestRunUntilWithoutEventsOnlyMovesClock(t *testing.T) {
	w, e := newToyRun(3)
	e.RunUntil(1000, nil)
	before := w.digest(e)
	e.RunUntil(1000, nil)
	if w.digest(e) != before {
		t.Fatal("RunUntil(now) changed state")
	}
	expectInvariantPanic(t, func() { e.RunUntil(999, nil) })
}

// funcProc adapts closures to Processor for focused tests.
type funcProc struct {
	stale    func(Event) bool
	handle   func(*Engine, Event)
	dispatch func(*Engine)
}

func (p *funcProc) IsStale(ev Event) bool {
	return p.stale != nil && p.stale(ev)
}
func (p *funcProc) Handle(e *Engine, ev Event) {
	if p.handle != nil {
		p.handle(e, ev)
	}
}
func (p *funcProc) Dispatch(e *Engine) {
	if p.dispatch != nil {
		p.dispatch(e)
	}
}
func (p *funcProc) CheckInvariants() error { return nil }

func expectInvariantPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		r := recover()
		if _, ok := r.(*InvariantError); !ok {
			t.Fatalf("expected InvariantError panic, got %v", r)
		}
	}()
	f()
}

// BenchmarkSyntheticWarmupLoad pushes roughly the event volume of a 180-day
// warm-up (~270k events) through the engine with a light handler, as early
// evidence for the < 1.5 s build target (ROADMAP §2 step 3).
func BenchmarkSyntheticWarmupLoad(b *testing.B) {
	for i := 0; i < b.N; i++ {
		w := newToyWorld(int64(i+1), 4)
		e := NewEngine(w, 0)
		e.Push(Event{At: 0, Class: ClassArrival, Kind: kArrive})
		for e.Handled() < 270_000 {
			e.RunUntil(e.Now().Add(simtime.Hour), nil)
		}
	}
}
