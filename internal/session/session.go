// Package session wraps one simulation with the pacing primitives every
// frontend uses: advance by a duration (optionally within an event budget),
// step to the next observable log entry, and observe. Frontends never touch
// the engine directly.
package session

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/rng"
	"github.com/georgezero/dreamteam-go/internal/scenario"
	"github.com/georgezero/dreamteam-go/internal/sim"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Session is one running simulation.
type Session struct {
	w *sim.World
}

// New returns a session on cfg, advanced to the scenario's start time.
func New(cfg *sim.Config, seed int64) *Session {
	s := &Session{w: sim.New(cfg, rng.SeedFromInt(seed))}
	s.w.Engine().RunUntil(s.w.StartTime(), nil)
	return s
}

// Open resolves a scenario (bundled ID or .json path) and starts a session.
func Open(ref string, seed int64) (*Session, error) {
	cfg, err := scenario.Resolve(ref)
	if err != nil {
		return nil, err
	}
	return New(cfg, seed), nil
}

// Now returns the simulated time.
func (s *Session) Now() simtime.Time { return s.w.Engine().Now() }

// Calendar returns the practice calendar.
func (s *Session) Calendar() simtime.Calendar { return s.w.Calendar() }

// Reason says why an advance ended.
type Reason int

const (
	Reached Reason = iota // advanced the full requested amount
	Budget                // event budget used up; clock is at a transaction boundary
	Alert                 // an alert was logged (auto-pause)
	Visible               // StepVisible reached a new log entry
	Horizon               // StepVisible found nothing within its horizon
)

// Options tune an advance.
type Options struct {
	// EventBudget, if positive, stops after about this many handled events
	// (always at a transaction boundary). Used for Max speed.
	EventBudget uint64
	// StopOnAlert stops right after a transaction that logged an alert.
	StopOnAlert bool
}

// Advance moves the clock forward by d, subject to opts.
func (s *Session) Advance(d simtime.Duration, opts Options) Reason {
	eng := s.w.Engine()
	startHandled := eng.Handled()
	startAlert := s.w.LastAlert()
	var why Reason
	stop := func() bool {
		if opts.StopOnAlert && s.w.LastAlert() != startAlert {
			why = Alert
			return true
		}
		if opts.EventBudget > 0 && eng.Handled()-startHandled >= opts.EventBudget {
			why = Budget
			return true
		}
		return false
	}
	if eng.RunUntil(eng.Now().Add(d), stop) == sim.Stopped {
		return why
	}
	return Reached
}

// StepHorizon bounds how far StepVisible searches.
const StepHorizon = 7 * simtime.Day

// StepVisible advances to the transaction that emits the next observable log
// entry. It never stops on hidden events (SIMULATION_DESIGN §9.3).
func (s *Session) StepVisible() Reason {
	eng := s.w.Engine()
	before := s.w.LogTotal()
	if eng.RunUntil(eng.Now().Add(StepHorizon), func() bool { return s.w.LogTotal() != before }) == sim.Stopped {
		return Visible
	}
	return Horizon
}

// Observe returns the current observation with up to recent log entries.
func (s *Session) Observe(recent int) api.Observation { return s.w.Observe(recent) }

// Digest identifies the run's observable history and clock. It grows to
// cover the full state digest in step 14.
func (s *Session) Digest() string {
	var b []byte
	b = append(b, s.w.LogHash()...)
	b = binary.BigEndian.AppendUint64(b, uint64(s.Now()))
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
