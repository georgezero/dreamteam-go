package session

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// V0.13b E1/E2: branches of one decision point (mini, seed 52, Tue
// 2026-10-13 07:00, the V0.13a mechanics check's point).

var (
	termE04 = []api.Action{{Type: api.ActTerminate, Employee: "E04", LastDay: "2026-10-27"}}
	keep    = []api.Action(nil)
)

func decisionPoint(t testing.TB) *Session {
	t.Helper()
	s, err := Open("mini", 52)
	if err != nil {
		t.Fatal(err)
	}
	at, err := s.Calendar().ParseISO("2026-10-13T07:00:00")
	if err != nil {
		t.Fatal(err)
	}
	s.AdvanceTo(at)
	return s
}

func decisionRecord(t testing.TB) *Record {
	t.Helper()
	r, err := decisionPoint(t).Record()
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func defaultOpts() BranchOptions {
	c := NoFurtherAction()
	return BranchOptions{Continuation: c}
}

func TestBranchTerminateVersusKeep(t *testing.T) {
	s := decisionPoint(t)
	before, saved := s.Digests(), s.saved
	set, err := s.Branch([]Alternative{{ID: "proposal", Actions: termE04}, {ID: "no_action", Actions: keep}},
		BranchOptions{Continuation: NoFurtherAction(), Verify: true})
	if err != nil {
		t.Fatal(err)
	}
	// The source session is only read.
	if s.Digests() != before || s.saved != saved || len(s.ActionLog()) != 0 {
		t.Fatal("branching changed the source session")
	}
	a, b := set.Branches[0], set.Branches[1]
	if !a.Verified || !b.Verified || a.Results[0].Code != api.ActionAccepted || b.Results != nil {
		t.Fatalf("branches: %+v %+v", a.Results, b.Results)
	}
	// Horizon: the end date (Oct 28) plus 30 days.
	if set.Horizon != "2026-11-27T00:00:00" {
		t.Fatalf("horizon %s", set.Horizon)
	}
	// The same start; the same outside demand; different outcomes.
	if a.Start.Digests != b.Start.Digests || a.Start.Digests != set.Source.Digests {
		t.Fatal("branches start from different states")
	}
	if a.End.Window.Ops.Demand.Received != b.End.Window.Ops.Demand.Received || a.End.Window.Ops.WalkIns.Arrived != b.End.Window.Ops.WalkIns.Arrived {
		t.Fatalf("outside demand differs: %d/%d orders, %d/%d walk-ins", a.End.Window.Ops.Demand.Received, b.End.Window.Ops.Demand.Received,
			a.End.Window.Ops.WalkIns.Arrived, b.End.Window.Ops.WalkIns.Arrived)
	}
	for i := range a.End.Calendar {
		x, y := a.End.Calendar[i], b.End.Calendar[i]
		if x.Date != y.Date || x.Orders != y.Orders || x.WalkIns != y.WalkIns {
			t.Fatalf("day %s: orders %d/%d, walk-ins %d/%d", x.Date, x.Orders, y.Orders, x.WalkIns, y.WalkIns)
		}
	}
	if a.AtHorizon == b.AtHorizon || a.End.Window.Ops.Exams == b.End.Window.Ops.Exams {
		t.Fatal("terminating changed nothing")
	}
	if a.Identity.Digest == b.Identity.Digest {
		t.Fatal("different actions, same identity")
	}
	t.Logf("horizon %s: exams %d vs %d, lost %d vs %d, orders %d", set.Horizon, a.End.Window.Ops.Exams, b.End.Window.Ops.Exams,
		a.End.Window.Ops.Demand.Lost, b.End.Window.Ops.Demand.Lost, a.End.Window.Ops.Demand.Received)
}

// Identical inputs give identical branches and identities; a different
// horizon or batch gives another identity; the identity names no seed.
func TestBranchIdentity(t *testing.T) {
	r := decisionRecord(t)
	alts := []Alternative{{ID: "proposal", Actions: termE04}}
	x, err := Branch(r, alts, defaultOpts())
	if err != nil {
		t.Fatal(err)
	}
	y, err := Branch(r, alts, defaultOpts())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(x.Branches[0].Identity, y.Branches[0].Identity) || x.Branches[0].AtHorizon != y.Branches[0].AtHorizon {
		t.Fatal("identical inputs, different branches")
	}
	// The ID labels the branch; it does not identify it.
	z, _ := Branch(r, []Alternative{{ID: "other", Actions: termE04}}, defaultOpts())
	if z.Branches[0].Identity.Digest != x.Branches[0].Identity.Digest {
		t.Fatal("the label changed the identity")
	}
	longer := defaultOpts()
	longer.Continuation.AfterLatestEffectDays = 31
	h, err := Branch(r, alts, longer)
	if err != nil {
		t.Fatal(err)
	}
	if h.HorizonSec != x.HorizonSec+86400 || h.Branches[0].Identity.Digest == x.Branches[0].Identity.Digest {
		t.Fatal("another horizon, same identity")
	}
	// The same batch and continuation with another sibling: the common
	// horizon moves, and so does the identity.
	sib, err := Branch(r, append(alts, Alternative{ID: "adjusted", Actions: []api.Action{{Type: api.ActTerminate, Employee: "E04", LastDay: "2026-11-10"}}}), defaultOpts())
	if err != nil {
		t.Fatal(err)
	}
	if sib.HorizonSec == x.HorizonSec || sib.Branches[0].Identity.Digest == x.Branches[0].Identity.Digest ||
		sib.Branches[0].Identity.ActionsSha256 != x.Branches[0].Identity.ActionsSha256 {
		t.Fatal("another common horizon, same identity")
	}
	later := []Alternative{{ID: "proposal", Actions: []api.Action{{Type: api.ActTerminate, Employee: "E04", LastDay: "2026-11-10"}}}}
	l, err := Branch(r, later, defaultOpts())
	if err != nil {
		t.Fatal(err)
	}
	if l.Branches[0].Identity.ActionsSha256 == x.Branches[0].Identity.ActionsSha256 || l.Branches[0].Identity.Digest == x.Branches[0].Identity.Digest {
		t.Fatal("another batch, same identity")
	}
	raw, _ := json.Marshal(x.Branches[0].Identity)
	var m map[string]any
	json.Unmarshal(raw, &m)
	if _, ok := m["seed"]; ok {
		t.Fatal("the identity names the seed")
	}
}

// A refused batch and an accepted batch that changes nothing material: the
// accrual point. Two worklist changes that undo each other are both
// accepted and accrue the interval metrics at the decision instant; every
// reported metric must equal the no-action branch's.
func TestBranchAccrualPoint(t *testing.T) {
	r := decisionRecord(t)
	o := decisionPoint(t).Observe(0)
	cur := o.Policies.Worklist
	other := "fifo"
	if cur == other {
		other = "priority"
	}
	set, err := Branch(r, []Alternative{
		{ID: "no_action"},
		{ID: "flip", Actions: []api.Action{{Type: api.ActSetWorklistPolicy, Policy: other}, {Type: api.ActSetWorklistPolicy, Policy: cur}}},
		{ID: "refused", Actions: []api.Action{{Type: api.ActTerminate, Employee: "E04", LastDay: "2026-10-14"}}},
	}, defaultOpts())
	if err != nil {
		t.Fatal(err)
	}
	n, f, x := set.Branches[0], set.Branches[1], set.Branches[2]
	if !f.Results[0].OK || !f.Results[1].OK || x.Results[0].OK {
		t.Fatalf("results %+v %+v", f.Results, x.Results)
	}
	// What differs is the record of the batch: the action counts and the
	// log, state (which holds the action log) and run digests. The ledger
	// and every reported metric are those of no action.
	noActions := func(w ResultWindow) ResultWindow { w.Ops.Actions = api.ActionCounts{}; return w }
	for _, b := range []BranchResult{f, x} {
		if b.AtHorizon.Ledger != n.AtHorizon.Ledger {
			t.Errorf("%s: ledger differs from no action", b.ID)
		}
		if !reflect.DeepEqual(noActions(b.End.Window), noActions(n.End.Window)) || !reflect.DeepEqual(b.End.Calendar, n.End.Calendar) ||
			!reflect.DeepEqual(b.End.Balances, n.End.Balances) || !reflect.DeepEqual(b.End.Obligations, n.End.Obligations) ||
			!reflect.DeepEqual(b.End.Alerts, n.End.Alerts) {
			t.Errorf("%s: a reported metric differs from no action", b.ID)
		}
		if b.AtHorizon.Actions == n.AtHorizon.Actions || b.End.Window.Ops.Actions == n.End.Window.Ops.Actions {
			t.Errorf("%s: the batch is not recorded", b.ID)
		}
	}
}

func TestBranchRefusals(t *testing.T) {
	r := decisionRecord(t)
	alt := []Alternative{{ID: "proposal", Actions: termE04}}
	for _, c := range []struct {
		name string
		alts []Alternative
		opts BranchOptions
	}{
		{"no alternative", nil, defaultOpts()},
		{"too many", make([]Alternative, MaxBranches+1), defaultOpts()},
		{"empty ID", []Alternative{{Actions: termE04}}, defaultOpts()},
		{"repeated ID", []Alternative{{ID: "a"}, {ID: "a", Actions: termE04}}, defaultOpts()},
		{"same batch twice", []Alternative{{ID: "a", Actions: termE04}, {ID: "b", Actions: termE04}}, defaultOpts()},
		{"unknown action", []Alternative{{ID: "a", Actions: []api.Action{{Type: "rehire", Employee: "E04"}}}}, defaultOpts()},
		{"unknown continuation", alt, BranchOptions{Continuation: Continuation{ID: "greedy", AfterDecisionDays: 30}}},
		{"zero horizon", alt, BranchOptions{Continuation: Continuation{ID: ContinuationNoFurtherAction}}},
	} {
		if _, err := Branch(r, c.alts, c.opts); !errors.Is(err, ErrBranchInput) {
			t.Errorf("%s: %v", c.name, err)
		}
	}
	// A damaged record.
	bad := *r
	bad.Seed = 53
	if _, err := Branch(&bad, alt, defaultOpts()); !errors.Is(err, ErrCorrupt) {
		t.Errorf("edited seed: %v", err)
	}
	bad = *r
	bad.Digests.Run = "00"
	bad.Checksum = bad.checksum()
	if _, err := Branch(&bad, alt, defaultOpts()); !errors.Is(err, ErrMismatch) {
		t.Errorf("wrong digest with a matching checksum: %v", err)
	}
}

// A record under earlier action rules, or continuing one, is refused; one
// under this build's rules from the 0.12.0 release is branched exactly.
func TestBranchHistoricalRules(t *testing.T) {
	r, err := ReadRecord(slice2Fixture)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Branch(r, []Alternative{{ID: "no_action"}}, defaultOpts()); !errors.Is(err, ErrBranchRules) {
		t.Fatalf("slice 2 record: %v", err)
	}
	s, _, err := Load(slice2Fixture, ReplayOptions{})
	if err != nil {
		t.Fatal(err)
	}
	s.Advance(simtime.Hour, Options{})
	if _, err := s.Branch([]Alternative{{ID: "no_action"}}, defaultOpts()); !errors.Is(err, ErrBranchRules) {
		t.Fatalf("continuing a slice 2 record: %v", err)
	}
	// A deeper prefix under earlier rules: a 0.2.0-dev.6 record continued
	// by a 0.13.0-dev.1 session (same rules as this build), continued
	// again here. The immediate prefix uses this build's rules; the one
	// below it does not.
	d6, _, err := Load("../../testdata/sessions/mini-seed26-dev6.session.json", ReplayOptions{})
	if err != nil {
		t.Fatal(err)
	}
	d6.Advance(simtime.Hour, Options{})
	r1, err := d6.Record()
	if err != nil {
		t.Fatal(err)
	}
	r1.Engine = "0.13.0-dev.1" // as that build would have saved it
	r1.Checksum = r1.checksum()
	c1, rep1, err := Replay(r1, ReplayOptions{})
	if err != nil {
		t.Fatalf("%v %v", err, rep1)
	}
	c1.Advance(simtime.Hour, Options{})
	r2, err := c1.Record()
	if err != nil {
		t.Fatal(err)
	}
	if r2.Legacy == nil || r2.Legacy.BatchMode != batchModeV0131 {
		t.Fatalf("prefix %+v", r2.Legacy)
	}
	if _, err := Branch(r2, []Alternative{{ID: "no_action"}}, defaultOpts()); !errors.Is(err, ErrBranchRules) {
		t.Fatalf("a dev.6 record two prefixes down: %v", err)
	}
	r, err = ReadRecord(v0120Fixture)
	if err != nil {
		t.Fatal(err)
	}
	set, err := Branch(r, []Alternative{{ID: "no_action"}}, BranchOptions{Continuation: NoFurtherAction(), Verify: true})
	if err != nil || set.Source.Compatibility != "earlier_build_exact" || !set.Branches[0].Verified {
		t.Fatalf("0.12.0 record: %v %+v", err, set)
	}
}

// A branch whose record, results, digests or identity were edited fails
// verification.
func TestBranchTamper(t *testing.T) {
	r := decisionRecord(t)
	set, err := Branch(r, []Alternative{{ID: "proposal", Actions: termE04}}, defaultOpts())
	if err != nil {
		t.Fatal(err)
	}
	good := set.Branches[0]
	if err := VerifyBranch(r, &good); err != nil {
		t.Fatal(err)
	}
	clone := func() BranchResult {
		b := good
		b.Verified = false
		raw, _ := json.Marshal(good.Record)
		b.Record, _ = DecodeRecord(raw)
		b.Results = append([]api.ActionResult(nil), good.Results...)
		return b
	}
	for name, edit := range map[string]func(b *BranchResult){
		"identity horizon":   func(b *BranchResult) { b.Identity.HorizonSec += 86400 },
		"identity actions":   func(b *BranchResult) { b.Identity.Actions[0].LastDay = "2026-11-10" },
		"identity re-digest": func(b *BranchResult) { b.Identity.Engine = "0.12.0"; b.Identity.Digest = b.Identity.digest() },
		"forged short horizon": func(b *BranchResult) {
			b.Identity.Continuation.AfterLatestEffectDays = 31
			b.Identity.ContinuationSha256 = b.Identity.Continuation.Sha256()
			b.Identity.Digest = b.Identity.digest()
		},
		"horizon digests": func(b *BranchResult) { b.AtHorizon.Run = good.AtDecision.Run },
		"results":         func(b *BranchResult) { b.Results[0].Code = "rejected" },
		"record batch": func(b *BranchResult) {
			b.Record.Batches[len(b.Record.Batches)-1].Actions[0].LastDay = "2026-11-10"
			b.Record.Checksum = b.Record.checksum()
		},
		"record checksum": func(b *BranchResult) { b.Record.Summary.Actions++ },
		"no record":       func(b *BranchResult) { b.Record = nil },
	} {
		b := clone()
		b.Identity.Actions = canonicalActions(good.Identity.Actions)
		edit(&b)
		if err := VerifyBranch(r, &b); err == nil || b.Verified {
			t.Errorf("%s: accepted", name)
		} else {
			t.Logf("%s: %v", name, err)
		}
	}
	// A set whose continuation was forged consistently with one branch
	// (horizon still long enough for it) fails the set's horizon rule.
	if err := VerifyBranchSet(r, set); err != nil {
		t.Fatal(err)
	}
	forged := *set
	forged.Branches = []BranchResult{clone()}
	fb := &forged.Branches[0]
	fb.Identity.Actions = canonicalActions(good.Identity.Actions)
	fb.Identity.Continuation.AfterLatestEffectDays = 29
	fb.Identity.ContinuationSha256 = fb.Identity.Continuation.Sha256()
	fb.Identity.Digest = fb.Identity.digest()
	forged.Continuation = fb.Identity.Continuation
	if err := VerifyBranchSet(r, &forged); err == nil {
		t.Error("a forged continuation verified")
	}
	// A branch of another source fails against this one.
	other := decisionPoint(t)
	other.Advance(simtime.Hour, Options{})
	r2, _ := other.Record()
	b := clone()
	if err := VerifyBranch(r2, &b); err == nil {
		t.Error("a branch verified against another source")
	}
}

// Saved branch records replay from the file, independently.
func TestBranchRecordSavesAndReplays(t *testing.T) {
	set, err := Branch(decisionRecord(t), []Alternative{{ID: "proposal", Actions: termE04}}, defaultOpts())
	if err != nil {
		t.Fatal(err)
	}
	raw, err := set.Branches[0].Record.Encode()
	if err != nil {
		t.Fatal(err)
	}
	p := t.TempDir() + "/branch.session.json"
	if err := os.WriteFile(p, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	s, rep, err := Load(p, ReplayOptions{Step: 5 * simtime.Hour})
	if err != nil || !rep.OK || s.Digests() != set.Branches[0].AtHorizon {
		t.Fatalf("%v %v", err, rep)
	}
}
