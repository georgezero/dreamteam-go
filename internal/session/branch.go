package session

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/georgezero/dreamteam-go/docs/evaluation"
	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/eval"
	"github.com/georgezero/dreamteam-go/internal/sim"
	"github.com/georgezero/dreamteam-go/internal/simtime"
	"github.com/georgezero/dreamteam-go/internal/version"
)

// Branches (V0.13b, E1 and E2): paired continuations of one decision
// point, each rebuilt from the same saved record. A branch is never a copy
// of a running world: each is a fresh replay of the record, verified digest
// for digest, then given its alternative's batch at the decision instant
// and advanced, with no further decision, to a common horizon. The source
// record and any session it came from are only read.

// Schemas and identities.
const (
	BranchIdentitySchema        = "dreamteam.branch-identity/v1"
	ContinuationNoFurtherAction = "dreamteam.continuation/no-further-action-v1"
	// MaxBranches bounds one call (four candidates and slack for tests).
	MaxBranches = 8
)

// Branch errors.
var (
	// ErrBranchRules: the record, or a prefix it continues, replays under
	// earlier action rules, or with another state encoding; it is never
	// approximated.
	ErrBranchRules = errors.New("branching needs a record under this build's action rules and state encoding")
	// ErrBranchInput: the alternatives or the continuation are malformed
	// or contradictory.
	ErrBranchInput = errors.New("invalid branch request")
	// ErrBranchIdentity: a branch does not match its identity.
	ErrBranchIdentity = errors.New("branch does not match its identity")
)

// Continuation is a declared continuation: what happens after the decision
// instant. no-further-action-v1: no management action of any kind; the
// policies in force continue; the horizon is the later of the decision
// instant plus AfterDecisionDays and the latest effect of any branch plus
// AfterLatestEffectDays.
type Continuation struct {
	ID                    string `json:"id"`
	AfterDecisionDays     int64  `json:"afterDecisionDays"`
	AfterLatestEffectDays int64  `json:"afterLatestEffectDays"`
}

// NoFurtherAction is the declared continuation with its default horizon
// (30 days after the decision and after the latest effect, the evaluation
// policy's daysAfterLatestEffect).
func NoFurtherAction() Continuation {
	return Continuation{ID: ContinuationNoFurtherAction, AfterDecisionDays: 30, AfterLatestEffectDays: 30}
}

func (c Continuation) check() error {
	if c.ID != ContinuationNoFurtherAction {
		return fmt.Errorf("%w: continuation %q is not %s", ErrBranchInput, c.ID, ContinuationNoFurtherAction)
	}
	if c.AfterDecisionDays < 1 || c.AfterDecisionDays > 3650 || c.AfterLatestEffectDays < 0 || c.AfterLatestEffectDays > 3650 {
		return fmt.Errorf("%w: continuation days %d and %d out of range", ErrBranchInput, c.AfterDecisionDays, c.AfterLatestEffectDays)
	}
	return nil
}

// Sha256 is the continuation's identity: the SHA-256 of its JSON.
func (c Continuation) Sha256() string { return sha256JSON(c) }

// Alternative is one branch to build: an ID and the batch applied at the
// decision instant (none: no batch).
type Alternative struct {
	ID      string       `json:"id"`
	Actions []api.Action `json:"actions"`
}

// BranchIdentity identifies a branch: everything that determines it. The
// seed enters only as a digest bound to the source record; it is not
// written in clear (a digest of a small integer binds it but cannot hide
// it from someone who enumerates seeds). Digest is the SHA-256 of the
// identity's JSON with Digest empty.
type BranchIdentity struct {
	Schema             string       `json:"schema"`
	SourceChecksum     string       `json:"sourceChecksum"`
	Scenario           string       `json:"scenarioFingerprint"`
	SeedDigest         string       `json:"seedDigest"`
	DecisionSec        int64        `json:"decisionSec"`
	DecisionTime       string       `json:"decisionTime"`
	DecisionRun        string       `json:"decisionRunDigest"`
	Engine             string       `json:"engine"`
	Actions            []api.Action `json:"actions"`
	ActionsSha256      string       `json:"actionsSha256"`
	Continuation       Continuation `json:"continuation"`
	ContinuationSha256 string       `json:"continuationSha256"`
	HorizonSec         int64        `json:"horizonSec"`
	Horizon            string       `json:"horizon"`
	Digest             string       `json:"digest"`
}

// BranchSource is the decision point, as rebuilt from the record.
type BranchSource struct {
	Checksum      string  `json:"checksum"`
	Engine        string  `json:"recordedBy"`
	Compatibility string  `json:"compatibility"`
	Scenario      string  `json:"scenarioFingerprint"`
	DecisionSec   int64   `json:"decisionSec"`
	DecisionTime  string  `json:"decisionTime"`
	Digests       Digests `json:"digests"`
}

// BranchResult is one branch: its identity, the engine's answer to its
// batch, its digests at the decision instant (after the batch) and at the
// horizon, its results at both ends, and its own record (replayable).
type BranchResult struct {
	ID         string             `json:"id"`
	Identity   BranchIdentity     `json:"identity"`
	Results    []api.ActionResult `json:"results"`
	AtDecision Digests            `json:"atDecision"`
	AtHorizon  Digests            `json:"atHorizon"`
	Start      *Result            `json:"-"`
	End        *Result            `json:"-"`
	Record     *Record            `json:"-"`
	Verified   bool               `json:"replayVerified"`
}

// BranchSet is every branch of one decision point under one continuation.
type BranchSet struct {
	Source       BranchSource   `json:"source"`
	Continuation Continuation   `json:"continuation"`
	HorizonSec   int64          `json:"horizonSec"`
	Horizon      string         `json:"horizon"`
	Branches     []BranchResult `json:"branches"`
}

// BranchOptions tune Branch.
type BranchOptions struct {
	Continuation Continuation
	// Verify replays each branch's own record from scratch and requires
	// every digest (the record's independent verification).
	Verify bool
}

// Branch builds alts from the decision point at the record's saved time
// (see the package comment above). It refuses: a damaged record; a holdout
// seed; a record (or prefix) under earlier rules or another state
// encoding; malformed, duplicate or too many alternatives; an unknown
// continuation. Nothing is approximated.
func Branch(r *Record, alts []Alternative, opts BranchOptions) (*BranchSet, error) {
	c := opts.Continuation
	if err := c.check(); err != nil {
		return nil, err
	}
	if err := checkAlternatives(alts); err != nil {
		return nil, err
	}
	src, base, err := branchSource(r)
	if err != nil {
		return nil, err
	}
	at := simtime.Time(src.DecisionSec)

	// Pass 1: the common horizon, from every branch's latest effect.
	latest := int64(0)
	for _, a := range alts {
		s, err := branchAt(r, src)
		if err != nil {
			return nil, err
		}
		if len(a.Actions) > 0 {
			s.Apply(a.Actions...)
		}
		latest = max(latest, s.w.LatestEffect())
	}
	horizon := commonHorizon(at, latest, c)
	cal := base.Calendar()
	set := &BranchSet{Source: src, Continuation: c, HorizonSec: int64(horizon), Horizon: cal.ISO(horizon)}

	// Pass 2: each branch to the horizon.
	for _, a := range alts {
		s, err := branchAt(r, src)
		if err != nil {
			return nil, err
		}
		start, err := s.Result(false)
		if err != nil {
			return nil, err
		}
		b := BranchResult{ID: a.ID, Start: start}
		if len(a.Actions) > 0 {
			b.Results = s.Apply(a.Actions...)
		}
		b.AtDecision = s.Digests()
		s.AdvanceTo(horizon)
		if err := s.w.Check(); err != nil {
			return nil, fmt.Errorf("branch %s: %w", a.ID, err)
		}
		b.AtHorizon = s.Digests()
		if b.End, err = s.Result(false); err != nil {
			return nil, err
		}
		if b.Record, err = s.Record(); err != nil {
			return nil, err
		}
		b.Identity = NewBranchIdentity(src, a.Actions, c, int64(horizon), cal.ISO(horizon), r.Seed)
		set.Branches = append(set.Branches, b)
	}
	if opts.Verify {
		if err := VerifyBranchSet(r, set); err != nil {
			return nil, err
		}
	}
	return set, nil
}

// Branch builds alts from this session's current state (its own record at
// the current time). The session is only read.
func (s *Session) Branch(alts []Alternative, opts BranchOptions) (*BranchSet, error) {
	r, err := s.Record()
	if err != nil {
		return nil, err
	}
	return Branch(r, alts, opts)
}

func checkAlternatives(alts []Alternative) error {
	if len(alts) == 0 || len(alts) > MaxBranches {
		return fmt.Errorf("%w: %d alternatives (1 to %d)", ErrBranchInput, len(alts), MaxBranches)
	}
	ids, batches := map[string]bool{}, map[string]string{}
	for _, a := range alts {
		if a.ID == "" || ids[a.ID] {
			return fmt.Errorf("%w: alternative ID %q empty or repeated", ErrBranchInput, a.ID)
		}
		ids[a.ID] = true
		k := sha256JSON(canonicalActions(a.Actions))
		if other, ok := batches[k]; ok {
			return fmt.Errorf("%w: alternatives %s and %s have the same batch", ErrBranchInput, other, a.ID)
		}
		batches[k] = a.ID
		for _, act := range a.Actions {
			if !knownAction(act.Type) {
				return fmt.Errorf("%w: alternative %s: action type %q is not an existing action", ErrBranchInput, a.ID, act.Type)
			}
		}
	}
	return nil
}

// knownAction reports an existing action type (ACTION_INVENTORY §1).
func knownAction(t string) bool {
	switch t {
	case api.ActExtendShift, api.ActSetWalkInPolicy, api.ActSetWorklistPolicy, api.ActHire, api.ActTerminate,
		api.ActSetRoster, api.ActSetReadingAllocation, api.ActContractTeleradiology:
		return true
	}
	return false
}

// branchSource checks the record and rebuilds its decision point once.
func branchSource(r *Record) (BranchSource, *Session, error) {
	if r == nil {
		return BranchSource{}, nil, fmt.Errorf("%w: no record", ErrBranchInput)
	}
	if r.Checksum != r.checksum() {
		return BranchSource{}, nil, ErrCorrupt
	}
	if err := eval.GuardManifest(evaluation.Holdout, []int64{r.Seed}); err != nil {
		return BranchSource{}, nil, err
	}
	// Every record in the chain (the record, its prefix, the prefix's
	// prefix, …) must use this build's action set and rules.
	for p := r; ; {
		if !currentRules(batchMode(p)) || (p.Legacy != nil && !currentRules(p.Legacy.BatchMode)) {
			return BranchSource{}, nil, ErrBranchRules
		}
		if p.Legacy == nil {
			break
		}
		next, err := p.prefix()
		if err != nil {
			return BranchSource{}, nil, err
		}
		p = next
	}
	s, rep, err := Replay(r, ReplayOptions{})
	if err != nil {
		return BranchSource{}, nil, err
	}
	for p := rep; p != nil; p = p.Prefix {
		if p.StateChanged {
			return BranchSource{}, nil, ErrBranchRules
		}
	}
	if s.Fingerprint() != r.Scenario.Fingerprint || int64(s.Now()) != r.UntilSec {
		return BranchSource{}, nil, ErrMismatch
	}
	return BranchSource{Checksum: r.Checksum, Engine: r.Engine, Compatibility: rep.Compatibility(), Scenario: r.Scenario.Fingerprint,
		DecisionSec: r.UntilSec, DecisionTime: r.Until, Digests: s.Digests()}, s, nil
}

// branchAt is a fresh replay of the record, at the decision point, with
// the source's digests.
func branchAt(r *Record, src BranchSource) (*Session, error) {
	s, _, err := Replay(r, ReplayOptions{})
	if err != nil {
		return nil, err
	}
	if s.Digests() != src.Digests {
		return nil, fmt.Errorf("%w: a branch's starting state differs from the decision point", ErrMismatch)
	}
	return s, nil
}

// NewBranchIdentity computes a branch's identity.
func NewBranchIdentity(src BranchSource, actions []api.Action, c Continuation, horizonSec int64, horizon string, seed int64) BranchIdentity {
	acts := canonicalActions(actions)
	id := BranchIdentity{Schema: BranchIdentitySchema, SourceChecksum: src.Checksum, Scenario: src.Scenario,
		SeedDigest: seedDigest(src.Checksum, seed), DecisionSec: src.DecisionSec, DecisionTime: src.DecisionTime,
		DecisionRun: src.Digests.Run, Engine: version.Number, Actions: acts, ActionsSha256: sha256JSON(acts),
		Continuation: c, ContinuationSha256: c.Sha256(), HorizonSec: horizonSec, Horizon: horizon}
	id.Digest = id.digest()
	return id
}

func (id BranchIdentity) digest() string {
	id.Digest = ""
	return sha256JSON(id)
}

// VerifyBranch checks a branch against its source record: its identity
// (recomputed, every field), its record (a continuation of the source:
// the same scenario, seed, takeover and batches, then at most the
// alternative's batch at the decision instant, saved at the horizon), and
// a replay of that record from scratch, which must reproduce every digest
// at the horizon. On success it sets Verified.
func VerifyBranch(src *Record, b *BranchResult) error {
	_, err := verifyBranch(src, b)
	return err
}

// VerifyBranchSet verifies every branch of a set and the set itself: one
// source, one continuation, and the common horizon its rule gives from
// every branch's latest effect (as replayed).
func VerifyBranchSet(src *Record, set *BranchSet) error {
	if set == nil || len(set.Branches) == 0 {
		return fmt.Errorf("%w: empty set", ErrBranchIdentity)
	}
	if set.Source.Checksum != src.Checksum || set.Source.DecisionSec != src.UntilSec || set.Source.Digests != src.Digests {
		return fmt.Errorf("%w: the set's source is not the record", ErrBranchIdentity)
	}
	c := set.Continuation
	if err := c.check(); err != nil {
		return err
	}
	latest := int64(0)
	ids := map[string]bool{}
	for i := range set.Branches {
		b := &set.Branches[i]
		if ids[b.ID] || b.Identity.Continuation != c || b.Identity.HorizonSec != set.HorizonSec || b.Identity.Horizon != set.Horizon {
			return fmt.Errorf("%w: %s: continuation, horizon or ID differs from the set's", ErrBranchIdentity, b.ID)
		}
		ids[b.ID] = true
		l, err := verifyBranch(src, b)
		if err != nil {
			return err
		}
		latest = max(latest, l)
	}
	if want := commonHorizon(simtime.Time(src.UntilSec), latest, c); int64(want) != set.HorizonSec {
		return fmt.Errorf("%w: horizon %s, the continuation gives %d", ErrBranchIdentity, set.Horizon, want)
	}
	return nil
}

// commonHorizon is the continuation's horizon rule.
func commonHorizon(at simtime.Time, latest int64, c Continuation) simtime.Time {
	h := at.Add(simtime.Duration(c.AfterDecisionDays) * simtime.Day)
	if latest > 0 {
		h = max(h, simtime.Time(latest).Add(simtime.Duration(c.AfterLatestEffectDays)*simtime.Day))
	}
	return h
}

// verifyBranch is VerifyBranch, returning the branch's latest effect as
// replayed; its horizon must be at least what its own effects require.
func verifyBranch(src *Record, b *BranchResult) (int64, error) {
	b.Verified = false
	l, err := verifyBranchOnly(src, b)
	if err != nil {
		return 0, err
	}
	if int64(commonHorizon(simtime.Time(src.UntilSec), l, b.Identity.Continuation)) > b.Identity.HorizonSec {
		return 0, fmt.Errorf("%w: %s: horizon before what its continuation requires", ErrBranchIdentity, b.ID)
	}
	b.Verified = true
	return l, nil
}

func verifyBranchOnly(src *Record, b *BranchResult) (int64, error) {
	id := b.Identity
	rec := b.Record
	if rec == nil {
		return 0, fmt.Errorf("%w: %s has no record", ErrBranchIdentity, b.ID)
	}
	if id.Digest != id.digest() {
		return 0, fmt.Errorf("%w: %s: identity digest", ErrBranchIdentity, b.ID)
	}
	if rec.Checksum != rec.checksum() || src.Checksum != src.checksum() {
		return 0, ErrCorrupt
	}
	want := BranchSource{Checksum: src.Checksum, Scenario: src.Scenario.Fingerprint, DecisionSec: src.UntilSec, DecisionTime: src.Until,
		Digests: src.Digests}
	again := NewBranchIdentity(want, id.Actions, id.Continuation, id.HorizonSec, id.Horizon, src.Seed)
	if again.Digest != id.Digest || id.Engine != version.Number || rec.Engine != version.Number {
		return 0, fmt.Errorf("%w: %s: identity does not follow from the source record", ErrBranchIdentity, b.ID)
	}
	if err := id.Continuation.check(); err != nil {
		return 0, err
	}
	switch {
	case rec.Seed != src.Seed || rec.Scenario.Fingerprint != src.Scenario.Fingerprint || rec.StartSec != src.StartSec:
		return 0, fmt.Errorf("%w: %s: record is not of the source's run", ErrBranchIdentity, b.ID)
	case rec.UntilSec != id.HorizonSec || rec.Until != id.Horizon:
		return 0, fmt.Errorf("%w: %s: record saved at %s, identity horizon %s", ErrBranchIdentity, b.ID, rec.Until, id.Horizon)
	case rec.Digests != b.AtHorizon:
		return 0, fmt.Errorf("%w: %s: horizon digests", ErrBranchIdentity, b.ID)
	}
	// The branch's batches are the source's, then the alternative's.
	srcBatches, err := allBatches(src)
	if err != nil {
		return 0, err
	}
	got, err := allBatches(rec)
	if err != nil {
		return 0, err
	}
	n := len(srcBatches)
	extra := 0
	if len(id.Actions) > 0 {
		extra = 1
	}
	if len(got) != n+extra || (n > 0 && !reflect.DeepEqual(got[:n], srcBatches)) {
		return 0, fmt.Errorf("%w: %s: record batches do not continue the source's", ErrBranchIdentity, b.ID)
	}
	if extra == 1 {
		x := got[n]
		if x.AtSec != src.UntilSec || sha256JSON(canonicalActions(x.Actions)) != id.ActionsSha256 || !reflect.DeepEqual(x.Results, b.Results) {
			return 0, fmt.Errorf("%w: %s: the decision batch differs from the identity's", ErrBranchIdentity, b.ID)
		}
	}
	if len(b.Results) != len(id.Actions) {
		return 0, fmt.Errorf("%w: %s: %d results for %d actions", ErrBranchIdentity, b.ID, len(b.Results), len(id.Actions))
	}
	s, _, err := Replay(rec, ReplayOptions{})
	if err != nil {
		return 0, fmt.Errorf("%s: %w", b.ID, err)
	}
	if s.Digests() != b.AtHorizon {
		return 0, fmt.Errorf("%w: %s: replay digests", ErrMismatch, b.ID)
	}
	return s.w.LatestEffect(), nil
}

// allBatches is every batch of a record, its prefixes' included, oldest
// first, compared by what was played (number, time, actions, results).
func allBatches(r *Record) ([]Batch, error) {
	var out []Batch
	if r.Legacy != nil {
		p, err := r.prefix()
		if err != nil {
			return nil, err
		}
		if out, err = allBatches(p); err != nil {
			return nil, err
		}
	}
	return append(out, r.Batches...), nil
}

// canonicalActions is a batch as JSON-decoded actions (nil: none), so
// that equal batches compare and hash equal however they were built.
func canonicalActions(actions []api.Action) []api.Action {
	if len(actions) == 0 {
		return []api.Action{}
	}
	raw, err := json.Marshal(actions)
	if err != nil {
		panic(err)
	}
	var out []api.Action
	if err := json.Unmarshal(raw, &out); err != nil {
		panic(err)
	}
	return out
}

func seedDigest(checksum string, seed int64) string {
	h := sha256.New()
	h.Write([]byte("dreamteam.seed/v1"))
	h.Write([]byte{0})
	h.Write([]byte(checksum))
	h.Write([]byte{0})
	h.Write(binary.BigEndian.AppendUint64(nil, uint64(seed)))
	return hex.EncodeToString(h.Sum(nil))
}

func sha256JSON(v any) string {
	raw, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// PreviewSlotsClosed reports the slots a termination-only batch closes
// (V0.13b, E4; sim.SlotsClosedReport). It is a separate, opt-in request:
// PreviewStaffing is unchanged. It changes nothing.
func (s *Session) PreviewSlotsClosed(actions ...api.Action) sim.SlotsClosedReport {
	return s.w.PreviewSlotsClosed(actions)
}

// BookingHorizonDays is how far ahead appointments can be booked.
func (s *Session) BookingHorizonDays() int64 { return s.w.Observe(0).Practice.BookingHorizonDays }
