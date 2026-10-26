package bench

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/georgezero/dreamteam-go/internal/agent"
	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/session"
	"github.com/georgezero/dreamteam-go/internal/simtime"
	"github.com/georgezero/dreamteam-go/internal/version"
)

// RunSchema identifies a run record (run.json).
const RunSchema = "dreamteam.benchmark-run/v1"

// Run statuses. A run that does not complete is classified by what
// stopped it; its files are written as far as it got.
const (
	StatusCompleted  = "completed"
	StatusPolicy     = "policy_error"     // the policy returned an error or panicked, or its decision could not be read
	StatusProtocol   = "protocol_error"   // the harness's own request failed (advance, save, digests)
	StatusEvaluation = "evaluation_error" // T90 failed other than as not applicable
	StatusReplay     = "replay_error"     // the saved session did not replay, or its record differs from the decisions
	StatusHarness    = "harness_error"    // the session could not be opened, or a file could not be written
)

// The files of one run, in its directory.
const (
	FileRun        = "run.json"
	FileDecisions  = "decisions.json"
	FileTranscript = "transcript.jsonl"
	FileSession    = "session.json"
	FileResult     = "result.json"
	FileT90        = "t90.json"
)

// RunRecord is one run: which (scenario, seed, policy), how it went, what
// it decided and what came of it.
type RunRecord struct {
	Schema    string `json:"schema"`
	ID        string `json:"id"`
	Benchmark string `json:"benchmark"`
	Scenario  struct {
		Ref         string `json:"ref"`
		ID          string `json:"id"`
		Fingerprint string `json:"fingerprint"`
	} `json:"scenario"`
	Seed     int64        `json:"seed"`
	Policy   PolicyRef    `json:"policy"`
	Protocol string       `json:"protocol"`
	Engine   string       `json:"engine"`
	Days     int          `json:"days"`
	Cadence  Cadence      `json:"cadence"`
	Status   string       `json:"status"`
	Failure  *Failure     `json:"failure,omitempty"`
	Initial  agent.Clock  `json:"initial"`
	Final    *agent.Clock `json:"final,omitempty"`
	Counts   Counts       `json:"counts"`
	Replay   *Replay      `json:"replay,omitempty"`
	// ActionsMatch: the batches the harness applied are the saved
	// session's batches, in order, at the same times, with the same
	// results.
	ActionsMatch bool    `json:"actionsMatchSession"`
	Metrics      []Value `json:"metrics,omitempty"`
	T90          T90Info `json:"t90"`
}

// PolicyRef names a policy and its version.
type PolicyRef struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

// Failure says what stopped a run: its class (the status), the decision
// point (0-based) and practice time, and the error.
type Failure struct {
	Class   string `json:"class"`
	Point   int    `json:"decisionPoint"`
	Time    string `json:"time"`
	Message string `json:"message"`
}

// Counts are a run's protocol traffic. Refused batches are batches the
// session found invalid (nothing applied); protocol errors are batches the
// protocol could not read (not JSON, an unknown field, a bad size); denied
// are policy requests the gate refused; request errors are the policy's
// other requests answered with an error (a malformed request, bad
// arguments).
type Counts struct {
	DecisionPoints int `json:"decisionPoints"`
	Batches        int `json:"batches"`
	Actions        int `json:"actions"`
	Applied        int `json:"applied"`
	Refused        int `json:"refused"`
	ProtocolErrors int `json:"protocolErrors"`
	PolicyRequests int `json:"policyRequests"`
	Denied         int `json:"denied"`
	RequestErrors  int `json:"requestErrors"`
}

// Replay is the saved session's replay verification (session.Load).
type Replay struct {
	Verified      bool   `json:"verified"`
	Compatibility string `json:"compatibility,omitempty"`
	Batches       int    `json:"batches"`
	RunDigest     string `json:"runDigest,omitempty"`
	Message       string `json:"message,omitempty"`
}

// T90 statuses.
const (
	T90Evaluated     = "evaluated"
	T90NotApplicable = "not_applicable" // the evaluation policy in force does not cover the scenario
	T90NotRequested  = "not_requested"
	T90Skipped       = "skipped" // the run did not complete
	T90Error         = "error"
)

// T90Info is a run's T90 outcome: the status and, when evaluated, the
// policy in force and the verdict, copied from t90.json.
type T90Info struct {
	Status         string          `json:"status"`
	Code           string          `json:"code,omitempty"`
	Message        string          `json:"message,omitempty"`
	PolicyID       string          `json:"policyId,omitempty"`
	PolicyDigest   string          `json:"policyDigest,omitempty"`
	AsOfDay        json.RawMessage `json:"asOfDay,omitempty"`
	Projected      bool            `json:"projected,omitempty"`
	Verdict        string          `json:"verdict,omitempty"`
	Classification string          `json:"classification,omitempty"`
}

// Decision is one decision point: when, and the batches the policy
// returned, each with what became of it.
type Decision struct {
	Point   int         `json:"point"`
	Time    string      `json:"time"`
	Batches []Submitted `json:"batches"`
	Error   string      `json:"error,omitempty"` // the policy's error (the run stops)
}

// Submitted is one batch as the policy returned it and the protocol's
// answer: applied (every action valid), refused (with each action's
// result), or a protocol error (nothing reached the session).
type Submitted struct {
	Actions []json.RawMessage  `json:"actions"`
	Applied bool               `json:"applied"`
	Results []api.ActionResult `json:"results,omitempty"`
	Error   *agent.Error       `json:"error,omitempty"`
}

// runID names a run.
func runID(scen string, seed int64, policy string) string {
	return fmt.Sprintf("%s-s%d-%s", scen, seed, policy)
}

// Matrix runs every scenario × seed × policy of c, in that order, into
// dir, which must not exist or be empty. raw is the configuration file,
// kept verbatim. Progress lines go to progress. It returns the manifest;
// an error means the results could not be written (a failed run is not
// one: it is recorded and classified).
func Matrix(c *Config, raw []byte, dir string, progress io.Writer) (*Manifest, error) {
	if entries, err := os.ReadDir(dir); err == nil && len(entries) > 0 {
		return nil, fmt.Errorf("output %s is not empty: results are written once", dir)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(dir, "runs"), 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(dir, FileConfig), raw, 0o644); err != nil {
		return nil, err
	}
	specs, err := c.prepare()
	if err != nil {
		return nil, err
	}
	var recs []*RunRecord
	timing := Timing{Note: "wall-clock seconds: not deterministic, not in the manifest's hashes"}
	for _, scen := range c.Scenarios {
		for _, seed := range c.Seeds {
			for _, spec := range specs {
				id := runID(scen, seed, spec.ID)
				rd := filepath.Join(dir, "runs", id)
				if err := os.MkdirAll(rd, 0o755); err != nil {
					return nil, err
				}
				t0 := time.Now()
				rec := runOne(c, scen, seed, spec, rd)
				secs := time.Since(t0).Seconds()
				if err := writeJSON(filepath.Join(rd, FileRun), rec); err != nil {
					return nil, err
				}
				timing.Runs = append(timing.Runs, RunTiming{ID: id, Seconds: float64(int64(secs*10)) / 10})
				recs = append(recs, rec)
				if progress != nil {
					fmt.Fprintf(progress, "%-40s %-16s %5.1f s\n", id, rec.Status, secs)
				}
			}
		}
	}
	m, err := finish(c, dir, recs, specs)
	if err != nil {
		return nil, err
	}
	return m, writeJSON(filepath.Join(dir, FileTiming), timing)
}

// Timing is the matrix's wall-clock record (timing.json).
type Timing struct {
	Note string      `json:"note"`
	Runs []RunTiming `json:"runs"`
}

// RunTiming is one run's wall-clock time.
type RunTiming struct {
	ID      string  `json:"id"`
	Seconds float64 `json:"seconds"`
}

// runOne plays one run and measures it. It never panics on a policy's
// behalf: whatever stops the run is classified in the record.
func runOne(c *Config, scen string, seed int64, spec PolicySpec, dir string) *RunRecord {
	rec := &RunRecord{Schema: RunSchema, ID: runID(scen, seed, spec.ID), Benchmark: c.ID, Seed: seed,
		Policy: PolicyRef{spec.ID, spec.Version}, Protocol: agent.Protocol, Engine: version.Number,
		Days: c.Days, Cadence: c.Cadence, Status: StatusCompleted, T90: T90Info{Status: T90NotRequested}}
	rec.Scenario.Ref = scen
	if c.Evaluation.T90 {
		rec.T90.Status = T90Skipped
	}
	fail := func(class string, point int, at string, err error) {
		if rec.Failure == nil {
			rec.Status, rec.Failure = class, &Failure{Class: class, Point: point, Time: at, Message: err.Error()}
		}
	}
	s, err := session.Open(scen, seed)
	if err != nil {
		fail(StatusHarness, 0, "", err)
		return rec
	}
	h := agent.New(s)
	h.Ref, h.Dir = scen, dir
	h.Benchmark = &agent.Benchmark{SeedWithheld: true, PermittedOps: PolicyOps, Decision: PolicyProtocol}
	tf, err := os.Create(filepath.Join(dir, FileTranscript))
	if err != nil {
		fail(StatusHarness, 0, "", err)
		return rec
	}
	tw := bufio.NewWriter(tf)
	hc := &agent.Client{H: h, Transcript: tw, IDPrefix: "h"}
	g := &gate{h: h}
	var pl player
	if spec.External != nil {
		pl = newExternal(spec, g, tw)
	} else {
		pl = &builtin{p: spec.New(), c: &agent.Client{H: g, Transcript: tw, IDPrefix: "p"}}
	}

	decisions := []Decision{}
	play(c, pl, hc, rec, &decisions, fail)
	if err := pl.finish(rec.Status == StatusCompleted, dir); err != nil {
		var pf *policyFailure
		if errors.As(err, &pf) {
			fail(pf.class, rec.Counts.DecisionPoints, "", err)
		} else {
			fail(StatusHarness, 0, "", err)
		}
	}
	rec.Counts.PolicyRequests, rec.Counts.Denied, rec.Counts.RequestErrors = pl.requests(), g.denied, g.errors

	var final agent.Clock
	if err := hc.Call("digests", nil, &final); err != nil {
		fail(StatusProtocol, rec.Counts.DecisionPoints, "", fmt.Errorf("digests: %w", err))
	} else {
		rec.Final = &final
	}
	saved := hc.Call("save", map[string]any{"path": FileSession}, nil)
	if saved != nil {
		fail(StatusProtocol, rec.Counts.DecisionPoints, "", fmt.Errorf("save: %w", saved))
	}
	if c.Evaluation.T90 && rec.Status == StatusCompleted {
		t90(hc, dir, rec, fail)
	}
	if err := errors.Join(tw.Flush(), tf.Close()); err != nil {
		fail(StatusHarness, 0, "", err)
	}
	if err := writeJSON(filepath.Join(dir, FileDecisions), decisions); err != nil {
		fail(StatusHarness, 0, "", err)
	}
	if saved == nil {
		measure(dir, rec, decisions, fail)
	}
	return rec
}

// play runs the decision points: the first at the cadence time on or
// after the takeover, then every day the clock moves to the next cadence
// time, the policy deciding every EveryDays days.
func play(c *Config, p player, hc *agent.Client, rec *RunRecord, decisions *[]Decision, fail func(string, int, string, error)) {
	var hello agent.Hello
	if err := hc.Call("hello", nil, &hello); err != nil {
		fail(StatusProtocol, 0, "", fmt.Errorf("hello: %w", err))
		return
	}
	rec.Initial = hello.Clock
	rec.Scenario.ID, rec.Scenario.Fingerprint = hello.Scenario.ID, hello.Scenario.Fingerprint
	at := hello.Clock.Time[:10] + "T" + c.Cadence.At + ":00"
	if at < hello.Clock.Time {
		var err error
		if at, err = nextDay(at); err != nil {
			fail(StatusHarness, 0, "", err)
			return
		}
	}
	if at > hello.Clock.Time && !advance(hc, at, 0, fail) {
		return
	}
	for d := 0; d < c.Days; d++ {
		if d%c.Cadence.EveryDays == 0 {
			dec := Decision{Point: rec.Counts.DecisionPoints, Time: at, Batches: []Submitted{}}
			rec.Counts.DecisionPoints++
			bs, err := p.decide(dec.Point)
			if err != nil {
				dec.Error = err.Error()
				*decisions = append(*decisions, dec)
				class := StatusPolicy
				var pf *policyFailure
				if errors.As(err, &pf) {
					class = pf.class
				}
				fail(class, dec.Point, at, err)
				return
			}
			for _, b := range bs {
				dec.Batches = append(dec.Batches, submit(hc, b, &rec.Counts))
			}
			*decisions = append(*decisions, dec)
		}
		next, err := nextDay(at)
		if err != nil {
			fail(StatusHarness, rec.Counts.DecisionPoints, at, err)
			return
		}
		if !advance(hc, next, rec.Counts.DecisionPoints, fail) {
			return
		}
		at = next
	}
}

// player is a run's policy as the harness plays it: a built-in policy
// with its gated client, or an external policy process.
type player interface {
	decide(point int) ([][]json.RawMessage, error)
	// finish ends the policy after the run (completed: it ran to the
	// end), keeping what it leaves in the run's directory.
	finish(completed bool, dir string) error
	requests() int
}

// builtin is a built-in policy and its client, which reaches the session
// through the gate.
type builtin struct {
	p agent.Policy
	c *agent.Client
}

func (b *builtin) decide(int) ([][]json.RawMessage, error) { return decide(b.p, b.c) }
func (b *builtin) finish(bool, string) error               { return nil }
func (b *builtin) requests() int                           { return b.c.Requests() }

// decide asks the policy, turning a panic into an error.
func decide(p agent.Policy, c *agent.Client) (bs [][]json.RawMessage, err error) {
	defer func() {
		if r := recover(); r != nil {
			bs, err = nil, fmt.Errorf("the policy panicked: %v", r)
		}
	}()
	return p.Decide(c)
}

// submit applies one batch through the protocol (apply validates first;
// an invalid batch changes nothing) and records the answer.
func submit(hc *agent.Client, b []json.RawMessage, n *Counts) Submitted {
	sub := Submitted{Actions: b}
	if sub.Actions == nil {
		sub.Actions = []json.RawMessage{}
	}
	n.Batches++
	n.Actions += len(b)
	for _, a := range b {
		if !json.Valid(a) {
			n.ProtocolErrors++
			sub.Actions = nil
			sub.Error = &agent.Error{Code: agent.ErrInvalidJSON, Message: "an action is not JSON; the batch was not sent"}
			return sub
		}
	}
	var r agent.Applied
	if err := hc.Call("apply", map[string]any{"actions": b}, &r); err != nil {
		n.ProtocolErrors++
		var pe *agent.Error
		if !errors.As(err, &pe) {
			pe = &agent.Error{Code: "client_error", Message: err.Error()}
		}
		sub.Error = pe
		return sub
	}
	sub.Applied, sub.Results = r.Applied, r.Results
	if r.Applied {
		n.Applied++
	} else {
		n.Refused++
	}
	return sub
}

func advance(hc *agent.Client, to string, point int, fail func(string, int, string, error)) bool {
	if err := hc.Call("advance", map[string]any{"to": to}, nil); err != nil {
		fail(StatusProtocol, point, to, fmt.Errorf("advance: %w", err))
		return false
	}
	return true
}

// nextDay is the same practice-local time on the next calendar day.
func nextDay(iso string) (string, error) {
	d, err := simtime.ParseDate(iso[:10])
	if err != nil {
		return "", err
	}
	return d.AddDays(1).String() + iso[10:], nil
}

// t90 asks the protocol for the T90 evaluation and keeps its result.
func t90(hc *agent.Client, dir string, rec *RunRecord, fail func(string, int, string, error)) {
	var raw json.RawMessage
	err := hc.Call("t90", nil, &raw)
	var pe *agent.Error
	switch {
	case errors.As(err, &pe) && pe.Code == agent.ErrT90Scenario:
		rec.T90 = T90Info{Status: T90NotApplicable, Code: pe.Code, Message: pe.Message}
		return
	case err != nil:
		rec.T90 = T90Info{Status: T90Error, Message: err.Error()}
		if pe != nil {
			rec.T90.Code = pe.Code
		}
		fail(StatusEvaluation, rec.Counts.DecisionPoints, "", fmt.Errorf("t90: %w", err))
		return
	}
	var b bytes.Buffer
	if err := json.Indent(&b, raw, "", " "); err != nil {
		fail(StatusHarness, 0, "", err)
		return
	}
	b.WriteByte('\n')
	if err := os.WriteFile(filepath.Join(dir, FileT90), b.Bytes(), 0o644); err != nil {
		fail(StatusHarness, 0, "", err)
		return
	}
	info, err := t90Info(raw)
	if err != nil {
		rec.T90 = T90Info{Status: T90Error, Message: err.Error()}
		fail(StatusEvaluation, rec.Counts.DecisionPoints, "", err)
		return
	}
	rec.T90 = info
}

// t90Info copies the evaluation's identity and verdict from the
// protocol's t90 result.
func t90Info(raw []byte) (T90Info, error) {
	var e struct {
		Evaluation struct {
			PolicyID     string          `json:"policyId"`
			PolicyDigest string          `json:"policyDigest"`
			AsOfDay      json.RawMessage `json:"asOfDay"`
			Projected    bool            `json:"projected"`
			Report       struct {
				Comparisons []struct {
					Verdict        string `json:"verdict"`
					Classification string `json:"classification"`
				} `json:"comparisons"`
			} `json:"report"`
		} `json:"evaluation"`
	}
	if err := json.Unmarshal(raw, &e); err != nil {
		return T90Info{}, err
	}
	ev := e.Evaluation
	if len(ev.Report.Comparisons) != 1 {
		return T90Info{}, fmt.Errorf("t90: %d comparisons, want 1", len(ev.Report.Comparisons))
	}
	return T90Info{Status: T90Evaluated, PolicyID: ev.PolicyID, PolicyDigest: ev.PolicyDigest, AsOfDay: ev.AsOfDay,
		Projected: ev.Projected, Verdict: ev.Report.Comparisons[0].Verdict, Classification: ev.Report.Comparisons[0].Classification}, nil
}

// measure replays the saved session (the existing replay verification),
// writes its canonical result (as dreamteam replay -format json does),
// reads the metrics from it, and checks that the session's batches are
// the decisions the harness applied.
func measure(dir string, rec *RunRecord, decisions []Decision, fail func(string, int, string, error)) {
	res, rep, err := Measure(filepath.Join(dir, FileSession))
	rec.Replay = &Replay{Verified: err == nil}
	if rep != nil {
		rec.Replay.Compatibility, rec.Replay.Batches, rec.Replay.RunDigest = rep.Compatibility(), rep.Batches, rep.Replayed.Run
	}
	if err != nil {
		rec.Replay.Message = err.Error()
		fail(StatusReplay, rec.Counts.DecisionPoints, "", fmt.Errorf("replay: %w", err))
		return
	}
	if rec.Final != nil && rep.Replayed != rec.Final.Digests {
		rec.Replay.Verified = false
		rec.Replay.Message = "the replayed digests differ from the session's at the end of the run"
		fail(StatusReplay, rec.Counts.DecisionPoints, "", errors.New(rec.Replay.Message))
		return
	}
	if err := os.WriteFile(filepath.Join(dir, FileResult), res, 0o644); err != nil {
		fail(StatusHarness, 0, "", err)
		return
	}
	if rec.Metrics, err = Extract(res); err != nil {
		fail(StatusHarness, 0, "", err)
		return
	}
	raw, err := os.ReadFile(filepath.Join(dir, FileSession))
	if err == nil {
		err = MatchSession(raw, decisions)
	}
	if err != nil {
		fail(StatusReplay, rec.Counts.DecisionPoints, "", err)
		return
	}
	rec.ActionsMatch = true
}

// Measure loads a saved session (replaying and verifying it) and returns
// its canonical result's bytes: what dreamteam replay FILE -format json
// writes.
func Measure(path string) ([]byte, *session.Report, error) {
	s, rep, err := session.Load(path, session.ReplayOptions{})
	if err != nil {
		return nil, rep, err
	}
	res, err := s.Result(false)
	if err != nil {
		return nil, rep, err
	}
	res.Replay = &session.ResultReplay{Verified: true, Batches: rep.Batches, Run: rep.Replayed.Run}
	raw, err := res.Encode()
	return raw, rep, err
}

// MatchSession checks that a saved session's batches are the applied
// batches of the decisions, in order: the same time, actions and results.
func MatchSession(raw []byte, decisions []Decision) error {
	r, err := session.DecodeRecord(raw)
	if err != nil {
		return err
	}
	var applied []Batchlike
	for _, d := range decisions {
		for _, b := range d.Batches {
			if b.Applied {
				applied = append(applied, Batchlike{d.Time, b.Actions, b.Results})
			}
		}
	}
	if len(applied) != len(r.Batches) {
		return fmt.Errorf("the session has %d batches, the decisions applied %d", len(r.Batches), len(applied))
	}
	for i, b := range r.Batches {
		a := applied[i]
		var acts []api.Action
		for _, x := range a.Actions {
			var act api.Action
			if err := json.Unmarshal(x, &act); err != nil {
				return err
			}
			acts = append(acts, act)
		}
		got, _ := json.Marshal([]any{b.Time, b.Actions, b.Results})
		want, _ := json.Marshal([]any{a.Time, acts, a.Results})
		if !bytes.Equal(got, want) {
			return fmt.Errorf("batch %d: the session has %s, the decisions %s", i+1, got, want)
		}
	}
	return nil
}

// Batchlike is an applied batch as the decisions record it.
type Batchlike struct {
	Time    string
	Actions []json.RawMessage
	Results []api.ActionResult
}

// writeJSON writes v indented (one space), with a trailing newline.
func writeJSON(path string, v any) error {
	raw, err := json.MarshalIndent(v, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}
