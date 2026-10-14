package session

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"syscall"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/scenario"
	"github.com/georgezero/dreamteam-go/internal/simtime"
	"github.com/georgezero/dreamteam-go/internal/version"
)

// Session records (SIMULATION_DESIGN §4.5, as built in step 14).
//
// A saved session is a run record, not a snapshot: the fully resolved
// scenario, the seed and every action batch with its results. Loading
// rebuilds the run from them — the warm-up, then each batch applied at its
// time through Apply — and checks every action result and the final
// digests. No serialized world is trusted.

// RecordSchema identifies the session file format. A different schema is
// refused; fields are never reinterpreted.
const RecordSchema = "dreamteam.session/v1"

// Record is one saved session.
type Record struct {
	Schema string `json:"schema"`
	// Engine is the build that recorded it. Replay needs the same
	// simulation rules: a different engine is refused unless the caller
	// allows it (and then the digests say whether outcomes still agree).
	Engine   string         `json:"engine"`
	Scenario RecordScenario `json:"scenario"`
	Seed     int64          `json:"seed"`
	// Times: simulation seconds since the scenario epoch, with the
	// practice-local ISO time alongside for reading.
	EpochDate string  `json:"epochDate"` // the calendar's day 0
	StartSec  int64   `json:"startSec"`  // the takeover
	Start     string  `json:"start"`
	UntilSec  int64   `json:"untilSec"` // where the run was saved
	Until     string  `json:"until"`
	Batches   []Batch `json:"batches"`
	Digests   Digests `json:"digests"`
	Summary   Summary `json:"summary"`
	// Checksum is the SHA-256 of the record's canonical JSON with this
	// field empty: any change or damage to the file is detected.
	Checksum string `json:"checksum"`
}

// RecordScenario is the scenario a session ran: its identity and the
// resolved file itself (after extends and presets), fingerprinted.
type RecordScenario struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Fingerprint string          `json:"fingerprint"` // SHA-256 of Resolved
	Resolved    json.RawMessage `json:"resolved"`
}

// Batch is one decision: the actions applied at one time and their
// results, rejections and no-ops included.
type Batch struct {
	Batch   int64              `json:"batch"`
	AtSec   int64              `json:"atSec"`
	Time    string             `json:"time"`
	Actions []api.Action       `json:"actions"`
	Results []api.ActionResult `json:"results"`
}

// Digests verify a run (sim/digest.go): the observable log, the ledger,
// the action record, the full hidden state with pending events, and Run
// over all of them and the clock.
type Digests struct {
	Log     string `json:"log"`
	Ledger  string `json:"ledger"`
	Actions string `json:"actions"`
	State   string `json:"state"`
	Run     string `json:"run"`
}

// Summary is a readable account of where the run stood, for diagnosing a
// mismatch.
type Summary struct {
	LogEntries    uint64 `json:"logEntries"`
	LedgerEntries int    `json:"ledgerEntries"`
	Actions       int    `json:"actions"`
	Day           int64  `json:"day"`
	Time          string `json:"time"`
	CashCents     int64  `json:"cashCents"`
	ExamsSince    int    `json:"examsSinceDay1"`
}

// Error kinds.
var (
	ErrCorrupt      = errors.New("session file is damaged or was edited")
	ErrIncompatible = errors.New("session file is not compatible with this build")
	ErrMismatch     = errors.New("replay did not reproduce the recorded run")
	ErrUnsavable    = errors.New("this session has no scenario source to save")
)

// Digests computes the verification digests now.
func (s *Session) Digests() Digests {
	d := Digests{Log: s.w.LogHash(), Ledger: s.w.LedgerHash(), Actions: s.w.ActionDigest(), State: s.w.StateDigest()}
	// Run binds the components to the scenario (its fingerprint: the rules
	// still to come, which the state digest does not see) and the seed.
	h := sha256.New()
	for _, x := range []string{RecordSchema, s.fp, d.Log, d.Ledger, d.Actions, d.State} {
		h.Write([]byte(x))
		h.Write([]byte{0})
	}
	h.Write(binary.BigEndian.AppendUint64(nil, uint64(s.seed)))
	h.Write(binary.BigEndian.AppendUint64(nil, uint64(s.Now())))
	d.Run = hex.EncodeToString(h.Sum(nil))
	return d
}

func (s *Session) summary() Summary {
	o := s.w.Observe(0)
	return Summary{LogEntries: o.LogTotal, LedgerEntries: o.Finance.JournalTotal, Actions: o.Actions.Attempted,
		Day: o.Day, Time: o.Time, CashCents: o.Finance.CashCents, ExamsSince: o.Flow.SinceStart.ExamsDone}
}

// Record builds the session's record at the current time.
func (s *Session) Record() (*Record, error) {
	if s.src == nil {
		return nil, ErrUnsavable
	}
	raw, err := scenario.Canonical(s.src)
	if err != nil {
		return nil, err
	}
	cal := s.Calendar()
	r := &Record{Schema: RecordSchema, Engine: version.Number, Seed: s.seed,
		Scenario:  RecordScenario{ID: s.src.ID, Name: s.src.Name, Fingerprint: s.fp, Resolved: raw},
		EpochDate: cal.Date(0).String(),
		StartSec:  int64(s.w.StartTime()), Start: cal.ISO(s.w.StartTime()),
		UntilSec: int64(s.Now()), Until: cal.ISO(s.Now()),
		Digests: s.Digests(), Summary: s.summary()}
	for _, res := range s.w.ActionLog() {
		if n := len(r.Batches); n == 0 || r.Batches[n-1].Batch != res.Batch {
			at, err := cal.ParseISO(res.Time)
			if err != nil {
				return nil, err
			}
			r.Batches = append(r.Batches, Batch{Batch: res.Batch, AtSec: int64(at), Time: res.Time})
		}
		b := &r.Batches[len(r.Batches)-1]
		b.Actions = append(b.Actions, res.Action)
		b.Results = append(b.Results, res)
	}
	r.Checksum = r.checksum()
	return r, nil
}

// checksum is the SHA-256 of the canonical JSON with Checksum empty.
func (r *Record) checksum() string {
	c := *r
	c.Checksum = ""
	raw, err := json.Marshal(&c)
	if err != nil {
		panic(err)
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// Encode is the record's file content.
func (r *Record) Encode() ([]byte, error) {
	raw, err := json.MarshalIndent(r, "", " ")
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

// Save writes the session's record to path atomically: a temporary file
// in the same directory, synced, then renamed over the target. A failure
// leaves any existing file untouched and no temporary behind.
func (s *Session) Save(path string) error {
	r, err := s.Record()
	if err != nil {
		return err
	}
	raw, err := r.Encode()
	if err != nil {
		return err
	}
	if err := writeAtomic(path, raw); err != nil {
		return err
	}
	s.path = path
	s.saved = savePoint{at: s.Now(), actions: len(s.w.ActionLog()), valid: true}
	return nil
}

// failBeforeRename, if set (tests), fails a save after the temporary file
// is written and synced: the moment a crash would hurt most.
var failBeforeRename func() error

func writeAtomic(path string, data []byte) (err error) {
	dir, base := filepath.Split(path)
	if dir == "" {
		dir = "."
	}
	tmp, err := os.CreateTemp(dir, "."+base+".tmp-*")
	if err != nil {
		return fmt.Errorf("save %s: %w", path, err)
	}
	defer func() {
		if err != nil {
			tmp.Close()
			os.Remove(tmp.Name())
		}
	}()
	if err = tmp.Chmod(0o644); err != nil {
		return fmt.Errorf("save %s: %w", path, err)
	}
	if _, err = tmp.Write(data); err != nil {
		return fmt.Errorf("save %s: %w", path, err)
	}
	if err = tmp.Sync(); err != nil {
		return fmt.Errorf("save %s: %w", path, err)
	}
	if failBeforeRename != nil {
		if err = failBeforeRename(); err != nil {
			return fmt.Errorf("save %s: %w", path, err)
		}
	}
	if err = tmp.Close(); err != nil {
		return fmt.Errorf("save %s: %w", path, err)
	}
	if err = os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("save %s: %w", path, err)
	}
	// Make the rename durable. The new file is in place either way; if the
	// directory cannot be synced, say so rather than claim a durable save.
	if err := syncDir(dir); err != nil {
		return fmt.Errorf("save %s: written, but its durability is not confirmed (directory sync: %w)", path, err)
	}
	return nil
}

// failDirSync, if set (tests), fails the directory sync after the rename.
var failDirSync func() error

func syncDir(dir string) error {
	if failDirSync != nil {
		return failDirSync()
	}
	if runtime.GOOS == "windows" {
		return nil // directories cannot be synced there; the rename is the guarantee
	}
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	if err := d.Sync(); err != nil && !errors.Is(err, syscall.EINVAL) && !errors.Is(err, syscall.ENOTSUP) {
		return err // EINVAL/ENOTSUP: the file system does not sync directories
	}
	return nil
}

// ReadRecord reads and checks a session file: it must parse completely
// (strictly: no unknown fields, nothing after the record), match its
// checksum, use this schema and carry an intact scenario. It does not
// replay; see Replay.
func ReadRecord(path string) (*Record, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return DecodeRecord(raw)
}

// DecodeRecord parses and checks a session record (see ReadRecord).
func DecodeRecord(raw []byte) (*Record, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var r Record
	if err := dec.Decode(&r); err != nil {
		return nil, fmt.Errorf("%w: cannot parse (%v; truncated?)", ErrCorrupt, err)
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return nil, fmt.Errorf("%w: data after the record", ErrCorrupt)
	}
	if r.Schema != RecordSchema {
		return nil, fmt.Errorf("%w: schema %q, this build reads %q", ErrIncompatible, r.Schema, RecordSchema)
	}
	if want := r.checksum(); r.Checksum != want {
		return nil, fmt.Errorf("%w: checksum %.12s… does not match the content (%.12s…)", ErrCorrupt, r.Checksum, want)
	}
	f, err := scenario.Decode(r.Scenario.Resolved)
	if err != nil {
		return nil, fmt.Errorf("%w: embedded scenario: %v", ErrCorrupt, err)
	}
	fp, err := scenario.Fingerprint(f)
	if err != nil {
		return nil, err
	}
	if fp != r.Scenario.Fingerprint {
		return nil, fmt.Errorf("%w: scenario fingerprint %.12s… does not match its content (%.12s…)", ErrCorrupt, r.Scenario.Fingerprint, fp)
	}
	return &r, nil
}

// ReplayOptions tune a replay.
type ReplayOptions struct {
	// AllowEngineMismatch replays a record from another build; the digests
	// then report whether outcomes still agree.
	AllowEngineMismatch bool
	// Step, if positive, advances the clock in chunks of at most this much
	// (to show chunking cannot matter).
	Step simtime.Duration
	// Noise, if set, is called before each chunk (tests: unrelated events).
	Noise func(s *Session, at simtime.Time)
}

// Report is the outcome of a replay.
type Report struct {
	OK         bool
	Engine     string // the build that recorded it
	Batches    int
	Actions    int
	Mismatches []string // what differed, in order
	Recorded   Digests
	Replayed   Digests
	RecSummary Summary
	RepSummary Summary
}

func (rep *Report) fail(format string, args ...any) {
	rep.OK = false
	rep.Mismatches = append(rep.Mismatches, fmt.Sprintf(format, args...))
}

// String is a readable report.
func (rep *Report) String() string {
	var b strings.Builder
	if rep.OK {
		fmt.Fprintf(&b, "OK: %d batches, %d actions replayed; digests match (run %.16s…)\n", rep.Batches, rep.Actions, rep.Replayed.Run)
		return b.String()
	}
	fmt.Fprintf(&b, "MISMATCH (recorded by %s, replayed by %s):\n", rep.Engine, version.Number)
	for _, m := range rep.Mismatches {
		fmt.Fprintf(&b, "  - %s\n", m)
	}
	fmt.Fprintf(&b, "  recorded: %+v\n  replayed: %+v\n", rep.RecSummary, rep.RepSummary)
	return b.String()
}

// Replay rebuilds a recorded run: the scenario and seed, every batch
// applied at its time through Apply (rejections and no-ops too), then the
// clock to the saved time. It checks each action result and the final
// digests and returns the session at the saved time. err is ErrMismatch
// (with the report) when anything differs.
func Replay(r *Record, opts ReplayOptions) (*Session, *Report, error) {
	// OK only once everything has been checked (fail sets it false; an
	// early refusal leaves it false too).
	rep := &Report{OK: true, Engine: r.Engine, Batches: len(r.Batches), Recorded: r.Digests, RecSummary: r.Summary}
	if r.Engine != version.Number && !opts.AllowEngineMismatch {
		rep.OK = false
		return nil, rep, fmt.Errorf("%w: recorded by %s, this is %s (replay with the engine mismatch allowed to compare outcomes)",
			ErrIncompatible, r.Engine, version.Number)
	}
	// Structure first: batches in time and number order, a result per
	// action, the end after the last batch.
	for i, b := range r.Batches {
		switch {
		case i > 0 && (b.AtSec < r.Batches[i-1].AtSec || b.Batch <= r.Batches[i-1].Batch):
			rep.fail("batch %d at %s follows batch %d at %s: batches out of order", b.Batch, b.Time, r.Batches[i-1].Batch, r.Batches[i-1].Time)
		case len(b.Results) != len(b.Actions) || len(b.Actions) == 0:
			rep.fail("batch %d: %d actions, %d results", b.Batch, len(b.Actions), len(b.Results))
		case b.AtSec < r.StartSec || b.AtSec > r.UntilSec:
			rep.fail("batch %d at %s is outside the run (%s to %s)", b.Batch, b.Time, r.Start, r.Until)
		}
	}
	if !rep.OK {
		return nil, rep, ErrMismatch
	}
	f, err := scenario.Decode(r.Scenario.Resolved)
	if err != nil {
		rep.OK = false
		return nil, rep, fmt.Errorf("%w: embedded scenario: %v", ErrCorrupt, err)
	}
	if r.Scenario.ID != f.ID || r.Scenario.Name != f.Name {
		rep.fail("scenario identity %q (%s) does not match its content %q (%s)", r.Scenario.ID, r.Scenario.Name, f.ID, f.Name)
		return nil, rep, ErrMismatch
	}
	s, err := FromFile(f, r.Seed)
	if err != nil {
		rep.OK = false
		return nil, rep, err
	}
	cal := s.Calendar()
	// Every readable copy of a time agrees with the seconds it describes,
	// and every result belongs to its batch.
	if cal.ISO(simtime.Time(r.StartSec)) != r.Start || cal.ISO(simtime.Time(r.UntilSec)) != r.Until {
		rep.fail("times: start %q, until %q do not match %d, %d", r.Start, r.Until, r.StartSec, r.UntilSec)
	}
	for _, b := range r.Batches {
		if cal.ISO(simtime.Time(b.AtSec)) != b.Time {
			rep.fail("batch %d: time %q does not match %d", b.Batch, b.Time, b.AtSec)
		}
		for _, res := range b.Results {
			if res.Batch != b.Batch || res.Time != b.Time {
				rep.fail("batch %d at %s holds a result of batch %d at %s", b.Batch, b.Time, res.Batch, res.Time)
				break
			}
		}
	}
	if !rep.OK {
		return s, rep, ErrMismatch
	}
	if got := cal.Date(0).String(); got != r.EpochDate || int64(s.w.StartTime()) != r.StartSec {
		rep.fail("calendar: epoch %s, takeover %s; recorded %s, %s", got, cal.ISO(s.w.StartTime()), r.EpochDate, r.Start)
		return s, rep, ErrMismatch
	}
	advance := func(to simtime.Time) {
		for now := s.Now(); now < to; {
			next := to
			if opts.Step > 0 {
				next = min(to, now.Add(opts.Step))
			}
			if opts.Noise != nil {
				opts.Noise(s, next)
			}
			s.w.Engine().RunUntil(next, nil)
			now = next
		}
	}
	for _, b := range r.Batches {
		at := simtime.Time(b.AtSec)
		if at < s.Now() {
			rep.fail("batch %d at %s is before the clock (%s): batches out of order", b.Batch, b.Time, cal.ISO(s.Now()))
			return s, rep, ErrMismatch
		}
		advance(at)
		got := s.Apply(b.Actions...)
		rep.Actions += len(got)
		if !reflect.DeepEqual(got, b.Results) {
			for i := range b.Results {
				if i >= len(got) || !reflect.DeepEqual(got[i], b.Results[i]) {
					var g any = "nothing"
					if i < len(got) {
						g = got[i]
					}
					rep.fail("batch %d at %s, action %d: recorded %+v, replay gave %+v", b.Batch, b.Time, i+1, b.Results[i], g)
					break
				}
			}
			rep.RepSummary = s.summary() // where the replay stood
			return s, rep, ErrMismatch
		}
	}
	if simtime.Time(r.UntilSec) < s.Now() {
		rep.fail("saved time %s is before the last batch", r.Until)
		return s, rep, ErrMismatch
	}
	advance(simtime.Time(r.UntilSec))
	rep.Replayed, rep.RepSummary = s.Digests(), s.summary()
	if rep.RepSummary != r.Summary {
		rep.fail("summary: recorded %+v, replayed %+v", r.Summary, rep.RepSummary)
	}
	for _, c := range []struct{ name, rec, got string }{
		{"log", r.Digests.Log, rep.Replayed.Log}, {"ledger", r.Digests.Ledger, rep.Replayed.Ledger},
		{"actions", r.Digests.Actions, rep.Replayed.Actions}, {"state", r.Digests.State, rep.Replayed.State},
		{"run", r.Digests.Run, rep.Replayed.Run},
	} {
		if c.rec != c.got {
			rep.fail("%s digest: recorded %.16s…, replayed %.16s…", c.name, c.rec, c.got)
		}
	}
	if !rep.OK {
		return s, rep, ErrMismatch
	}
	s.saved = savePoint{at: s.Now(), actions: len(s.w.ActionLog()), valid: true}
	return s, rep, nil
}

// Load reads, checks and replays a session file, returning the session at
// its saved time, ready to continue.
func Load(path string, opts ReplayOptions) (*Session, *Report, error) {
	r, err := ReadRecord(path)
	if err != nil {
		return nil, nil, err
	}
	s, rep, err := Replay(r, opts)
	if s != nil {
		s.path = path
	}
	return s, rep, err
}

// Verify replays this session's own record from scratch and compares: the
// run as played must be reproducible from its scenario, seed and actions.
func (s *Session) Verify() (*Report, error) {
	r, err := s.Record()
	if err != nil {
		return nil, err
	}
	_, rep, err := Replay(r, ReplayOptions{})
	return rep, err
}

// Path is the file this session was last saved to or loaded from.
func (s *Session) Path() string { return s.path }

// Seed is the session's seed.
func (s *Session) Seed() int64 { return s.seed }

// ScenarioID is the scenario's ID ("" without a source).
func (s *Session) ScenarioID() string {
	if s.src == nil {
		return ""
	}
	return s.src.ID
}

// Fingerprint is the scenario fingerprint ("" without a source).
func (s *Session) Fingerprint() string { return s.fp }

// Unsaved reports whether the run moved on (clock or actions) since it was
// last saved or loaded; a session never saved is unsaved once it moves
// past its start.
func (s *Session) Unsaved() bool {
	if !s.saved.valid {
		return !s.AtStart() || len(s.w.ActionLog()) > 0
	}
	return s.Now() != s.saved.at || len(s.w.ActionLog()) != s.saved.actions
}

// DefaultPath is where a session is saved when no file is named.
func (s *Session) DefaultPath() string {
	return fmt.Sprintf("%s-seed%d.session.json", s.ScenarioID(), s.seed)
}
