package session

import (
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"

	"github.com/georgezero/dreamteam-go/docs/evaluation"
	"github.com/georgezero/dreamteam-go/internal/eval"
	"github.com/georgezero/dreamteam-go/internal/rng"
	"github.com/georgezero/dreamteam-go/internal/scenario"
	"github.com/georgezero/dreamteam-go/internal/sim"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// The T90 evaluation's plan names: the player's plan and the paired
// no-action baseline it is judged against.
const (
	T90Plan     = "your plan"
	T90Baseline = "no action"
)

// T90 errors.
var (
	// ErrT90Scenario: the policy in force is defined for one scenario
	// (its fingerprint); no other scenario is evaluated.
	ErrT90Scenario = errors.New("the T90 evaluation is defined for the sustainable scenario only")
	// ErrT90Started: nothing to evaluate before the takeover.
	ErrT90Started = errors.New("the T90 evaluation needs play to have started")
)

// T90 is the practice's management evaluation under the evaluation policy
// in force (v0.2-eval-4, embedded byte for byte): the player's decisions
// before T90 against the no-action baseline on the same scenario and
// seed, both run with no further decision to the policy's observation
// end, judged by the policy's own rules (eval.Evaluate). It adds no rule:
// the metric vector, the gates and the verdict are the policy's.
type T90 struct {
	PolicyID     string `json:"policyId"`
	PolicyDigest string `json:"policyDigest"`
	Seed         int64  `json:"seed"`
	Scenario     string `json:"scenarioFingerprint"`
	// AsOf: the session's time when evaluated, and its day since the
	// takeover. Projected: the session has not reached T90, so the days
	// from AsOf to T90 are the plan as it stands with no further decision.
	AsOf      string  `json:"asOf"`
	AsOfDay   float64 `json:"asOfDay"`
	Projected bool    `json:"projected"`
	// T90: the primary endpoint; HorizonEnd: the end of the evaluation
	// tail (fixed policy, no decision); ObservationEnd: how far both plans
	// are observed.
	T90            string `json:"t90"`
	HorizonEnd     string `json:"horizonEnd"`
	ObservationEnd string `json:"observationEnd"`
	// Decisions: the batches before T90 (the plan); Later: batches at or
	// after T90, a later period the T90 evaluation does not include.
	Decisions int `json:"decisions"`
	Later     int `json:"later"`
	// Report: eval.Evaluate's report, the plan and the baseline.
	Report *eval.Report `json:"report"`
}

// t90Policy is the embedded policy in force and its digest.
func t90Policy() (*eval.Policy, string, error) {
	p, err := eval.DecodePolicy(evaluation.PolicyV4)
	if err != nil {
		return nil, "", err
	}
	sum := sha256.Sum256(evaluation.PolicyV4)
	return p, hex.EncodeToString(sum[:]), nil
}

// T90 evaluates the session's plan (V0.2_DESIGN §13): T90Job, then Run.
func (s *Session) T90() (*T90, error) {
	j, err := s.T90Job()
	if err != nil {
		return nil, err
	}
	return j.Run()
}

// T90Job is a T90 evaluation's inputs, taken from the session: Run reads
// nothing of the session, so a frontend may run it in the background
// while play goes on.
type T90Job struct {
	cfg        *sim.Config
	fp         string
	seed       int64
	cal        simtime.Calendar
	start, now simtime.Time
	t90        simtime.Time
	plan       []Batch
	later      int
	p          *eval.Policy
	digest     string
}

// T90Job checks that the session can be evaluated and takes its plan: the
// policy's scenario, a seed outside the sealed holdout, play started.
func (s *Session) T90Job() (*T90Job, error) {
	if s.src == nil {
		return nil, ErrUnsavable
	}
	if s.legacy != nil {
		return nil, ErrT90Scenario // v1 sessions are on the V0.1 scenarios
	}
	p, digest, err := t90Policy()
	if err != nil {
		return nil, err
	}
	if s.fp != p.Scenario.Fingerprint {
		return nil, ErrT90Scenario
	}
	// A holdout seed is evaluated only by the canonical runner.
	if err := eval.GuardManifest(evaluation.Holdout, []int64{s.seed}); err != nil {
		return nil, err
	}
	start := s.w.StartTime()
	if s.Now() <= start {
		return nil, ErrT90Started
	}
	cfg, err := scenario.Compile(s.src)
	if err != nil {
		return nil, err
	}
	rec, err := s.Record()
	if err != nil {
		return nil, err
	}
	j := &T90Job{cfg: cfg, fp: s.fp, seed: s.seed, cal: s.Calendar(), start: start, now: s.Now(),
		t90: start.Add(simtime.Duration(p.Horizons.PrimaryDays) * simtime.Day), p: p, digest: digest}
	for _, b := range rec.Batches {
		if simtime.Time(b.AtSec) < j.t90 {
			j.plan = append(j.plan, b)
		} else {
			j.later++
		}
	}
	return j, nil
}

// Run runs the plan and the no-action baseline from the takeover to the
// observation end on fresh worlds (about two 200-day runs) and evaluates
// them.
func (j *T90Job) Run() (*T90, error) {
	p := j.p
	plans := map[string][]Batch{T90Baseline: nil, T90Plan: j.plan}
	names := []string{T90Baseline, T90Plan}
	// The common horizon, as the canonical runner sets it: each plan's own
	// tail end, then confirmed by every record (once rerun if it moves).
	horizon := int64(0)
	for _, n := range names {
		horizon = max(horizon, t90Tail(j.cfg, j.seed, p, plans[n]))
	}
	var records map[string][]eval.Record
	for pass := 0; ; pass++ {
		records = map[string][]eval.Record{}
		need := horizon
		for _, n := range names {
			r, err := t90Run(j.cfg, j.fp, j.seed, p, n, plans[n], horizon)
			if err != nil {
				return nil, err
			}
			records[n] = []eval.Record{r}
			need = max(need, eval.TailEnd(p, r.TakeoverSec, r.LatestEffectSec, r.LatestCommitmentSec))
		}
		if need == horizon {
			break
		}
		if pass > 0 {
			return nil, fmt.Errorf("t90: the common horizon does not settle (%d, then %d)", horizon, need)
		}
		horizon = need
	}
	end := simtime.Time(eval.ObservationEnd(p, int64(j.start), j.cal.Date(j.start).String(), horizon))
	rep := eval.Evaluate(p, j.digest, eval.Input{Baseline: T90Baseline, Plans: records})
	return &T90{PolicyID: p.ID, PolicyDigest: j.digest, Seed: j.seed, Scenario: j.fp,
		AsOf: j.cal.ISO(j.now), AsOfDay: float64(j.now-j.start) / float64(simtime.Day), Projected: j.now < j.t90,
		T90: j.cal.ISO(j.t90), HorizonEnd: j.cal.ISO(simtime.Time(horizon)), ObservationEnd: j.cal.ISO(end),
		Decisions: len(j.plan), Later: j.later, Report: rep}, nil
}

// t90Tail is a plan's own tail end: run to its last batch, apply them all,
// and read the latest effect and commitment (the canonical runner's
// strategyTail).
func t90Tail(cfg *sim.Config, seed int64, p *eval.Policy, plan []Batch) int64 {
	w := sim.New(cfg, rng.SeedFromInt(seed))
	w.Engine().RunUntil(w.StartTime(), nil)
	var last int64
	for _, b := range plan {
		w.Engine().RunUntil(simtime.Time(b.AtSec), nil)
		for _, res := range w.Apply(b.Actions) {
			if res.OK {
				last = b.AtSec
			}
		}
	}
	return eval.TailEnd(p, int64(w.StartTime()), w.LatestEffect(), w.LatestCommitment(last))
}

// t90Run records one plan to the common horizon: daily samples, the
// captures, its batches at their instants (before a sample or capture at
// the same instant), as the canonical runner does. A refused batch is
// refused again, as when it was played.
func t90Run(cfg *sim.Config, fp string, seed int64, p *eval.Policy, name string, plan []Batch, horizon int64) (eval.Record, error) {
	w := sim.New(cfg, rng.SeedFromInt(seed))
	start := w.StartTime()
	w.Engine().RunUntil(start, nil)
	rec := w.NewEvalRecorderUnguarded(name, "", fp, seed, p.Finance.EmployeeCommitmentDays)
	h := p.Horizons
	day := func(n int) simtime.Time { return start.Add(simtime.Duration(n) * simtime.Day) }
	end := simtime.Time(eval.ObservationEnd(p, int64(start), w.Calendar().Date(start).String(), horizon))
	type mark struct {
		at    simtime.Time
		order int // 0 batch, 1 sample, 2 capture
		batch int
		day   int
		name  string
	}
	var marks []mark
	for i, b := range plan {
		marks = append(marks, mark{at: simtime.Time(b.AtSec), batch: i})
	}
	for d := 1; day(d) <= end; d++ {
		marks = append(marks, mark{at: day(d), order: 1, day: d})
	}
	for _, c := range []mark{{at: day(h.FeedbackDays), name: "T30"}, {at: day(h.PrimaryDays), name: "T90"}, {at: simtime.Time(horizon), name: "tailEnd"},
		{at: day(h.ValidationDays), name: "T180"}, {at: end, name: "end"}} {
		c.order = 2
		marks = append(marks, c)
	}
	slices.SortStableFunc(marks, func(x, y mark) int { return cmp.Or(cmp.Compare(x.at, y.at), cmp.Compare(x.order, y.order)) })
	var last int64
	for _, m := range marks {
		w.Engine().RunUntil(m.at, nil)
		switch m.order {
		case 0:
			for _, res := range w.Apply(plan[m.batch].Actions) {
				if res.OK {
					last = int64(m.at)
				}
			}
		case 1:
			rec.Sample(m.day)
		default:
			rec.Capture(m.name)
		}
	}
	if err := w.Check(); err != nil {
		return eval.Record{}, err
	}
	return rec.Finish(last, w.LatestEffect(), horizon, ""), nil
}
