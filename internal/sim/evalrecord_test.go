package sim_test

import (
	"bytes"
	"cmp"
	"encoding/json"
	"slices"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/eval"
	"github.com/georgezero/dreamteam-go/internal/sim"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// The evaluation recorder on real runs (V0.2_DESIGN §13.4): the mini
// practice with and without a teleradiology contract, under the frozen
// policy with its horizons shortened to fit a short run. The records
// check out (samples reconcile with the studies, remote fees with the
// ledger, the cash bridge), remote reading changes no booking, and
// recording changes nothing in the run.

// shortPolicy is the frozen policy over days 5 / 20 / 40 with a 10-day
// tail, 14-day backlog blocks (one rota cycle) and payback to days 20
// and 40.
func shortPolicy(t *testing.T, cfg *sim.Config) (*eval.Policy, string) {
	t.Helper()
	p, d, err := eval.LoadPolicy("../../" + eval.PolicyPath)
	if err != nil {
		t.Fatal(err)
	}
	h := &p.Horizons
	h.FeedbackDays, h.PrimaryDays, h.OperatingFrom, h.ValidationDays = 5, 20, 5, 40
	h.Tail.MinDaysAfterPrimary, h.Tail.DaysAfterEffect = 10, 5
	p.Backlog.BlockDays = 14
	p.Payback.Horizons = []int{20, 40}
	p.Scenario.Fingerprint = ""
	for q := sim.Routine; q <= sim.Urgent; q++ {
		p.Scenario.TATTargetMin[q.String()] = int64(cfg.Policies.TATTarget[q] / simtime.Minute)
	}
	return p, d
}

// miniWorld is the mini practice at the takeover on seed. When tele is
// set, a teleradiology contract is applied at the takeover, after before
// (if any) has run: its last action and latest effect (0 without).
func miniWorld(t *testing.T, seed int64, tele bool, before func(*sim.World)) (w *sim.World, last, effect int64) {
	t.Helper()
	cfg := *miniConfig(t)
	cfg.EvaluationDays = 0
	w = sim.NewEvalWorld(&cfg, seed)
	start := w.StartTime()
	w.Engine().RunUntil(start, nil)
	if before != nil {
		before(w)
	}
	if tele {
		eff := w.Calendar().Date(start.Add(14 * simtime.Day))
		res := w.Apply([]api.Action{{Type: api.ActContractTeleradiology, Op: api.TeleOpStart, RequestID: "eval", Plan: "standard-v1",
			Windows: []string{"weeknights", "saturday", "sunday"}, Scope: "all", Effective: eff.String()}})
		if !res[0].OK {
			t.Fatalf("%+v", res[0])
		}
		last, effect = int64(start), int64(w.Calendar().StartOfDate(eff))
	}
	return w, last, effect
}

// commonHorizon is the evaluation's horizon over both plans, computed
// before either runs: the latest tail end (the contract's, here).
func commonHorizon(t *testing.T, p *eval.Policy, seed int64) int64 {
	t.Helper()
	w, last, effect := miniWorld(t, seed, true, nil)
	t0 := int64(w.StartTime())
	return max(eval.TailEnd(p, t0, 0, 0), eval.TailEnd(p, t0, effect, w.LatestCommitment(last)))
}

// recordMini runs the mini practice on seed for the short policy's
// horizons, both plans to the common horizon and its observation end,
// applying a teleradiology contract at the takeover when tele is set;
// record false runs the same without a recorder.
func recordMini(t *testing.T, p *eval.Policy, seed int64, tele, record bool) (eval.Record, string) {
	t.Helper()
	horizon := commonHorizon(t, p, seed)
	var rec *sim.EvalRecorder
	w, last, effect := miniWorld(t, seed, tele, func(w *sim.World) {
		if record {
			rec = w.NewEvalRecorder(map[bool]string{false: "A", true: "T"}[tele], "", "", seed, p.Finance.EmployeeCommitmentDays)
		}
	})
	start := w.StartTime()
	h := p.Horizons
	day := func(n int) simtime.Time { return start.Add(simtime.Duration(n) * simtime.Day) }
	tail := simtime.Time(horizon)
	end := simtime.Time(eval.ObservationEnd(p, int64(start), w.Calendar().Date(start).String(), horizon))
	type mark struct {
		at   simtime.Time
		day  int    // a daily sample, or
		name string // a capture (after the sample at the same instant)
	}
	var marks []mark
	for d := 1; day(d) <= end; d++ {
		marks = append(marks, mark{at: day(d), day: d})
	}
	for _, c := range []mark{{at: day(h.FeedbackDays), name: "T30"}, {at: day(h.PrimaryDays), name: "T90"}, {at: tail, name: "tailEnd"},
		{at: day(h.ValidationDays), name: "T180"}, {at: end, name: "end"}} {
		marks = append(marks, c)
	}
	slices.SortStableFunc(marks, func(x, y mark) int { return cmp.Compare(x.at, y.at) })
	for _, m := range marks {
		w.Engine().RunUntil(m.at, nil)
		switch {
		case rec == nil:
		case m.name == "":
			rec.Sample(m.day)
		default:
			rec.Capture(m.name)
		}
	}
	if err := w.Check(); err != nil {
		t.Fatal(err)
	}
	if rec == nil {
		return eval.Record{}, w.StateDigest()
	}
	return rec.Finish(last, effect, horizon, map[bool]string{true: "A"}[tele]), w.StateDigest()
}

func TestEvalRecordOnRealRuns(t *testing.T) {
	p, d := shortPolicy(t, miniConfig(t))
	plans := map[string][]eval.Record{}
	for _, seed := range []int64{3, 4} {
		a, da := recordMini(t, p, seed, false, true)
		tr, dt := recordMini(t, p, seed, true, true)
		if _, db := recordMini(t, p, seed, true, false); db != dt {
			t.Fatal("recording changed the run")
		}
		if da == dt {
			t.Fatal("the contract changed nothing")
		}
		plans["A"], plans["T"] = append(plans["A"], a), append(plans["T"], tr)
		for _, r := range []eval.Record{a, tr} {
			m := eval.Measure(p, &r)
			if len(m.Integrity) != 0 {
				t.Fatalf("%s seed %d: %v", r.Plan, seed, m.Integrity)
			}
		}
		m := eval.Measure(p, &tr)
		rm := m.Windows[eval.WinT180].Remote
		if rm.RemoteReports == 0 || rm.ReadFeeCents != rm.ClaimFeeCents || rm.RetainerCents == 0 || rm.HandoffWaitMin.P50 != 10 {
			t.Fatalf("seed %d: %+v", seed, rm)
		}
		var finals int64
		for _, s := range tr.Studies {
			if s.FinalSec > tr.TakeoverSec && s.FinalSec <= tr.TakeoverSec+40*86400 {
				finals++
			}
		}
		if rm.LocalReports+rm.RemoteReports != finals {
			t.Fatalf("seed %d: %d local + %d remote, %d finals", seed, rm.LocalReports, rm.RemoteReports, finals)
		}
	}
	rep := eval.Evaluate(p, d, eval.Input{Baseline: "A", Plans: plans})
	for _, pr := range rep.Plans {
		if !pr.Valid {
			t.Fatalf("%s: %v", pr.Plan, pr.Integrity)
		}
		if pr.Plan == "T" && (!pr.BookingInvariant || pr.BookingInvariantWith != "A") {
			t.Fatal("remote reading changed the bookings")
		}
	}
	if c := rep.Comparisons[0]; !c.Valid || c.RemoteCost[eval.WinT90].Mean <= 0 || c.Verdict == "" {
		t.Fatalf("%+v", c)
	}
	// Both plans share the horizon, the contract's minimum term sets it,
	// and the bridge carries the forward employee pay at every capture.
	for _, r := range append(plans["A"], plans["T"]...) {
		if r.HorizonEndSec != rep.HorizonEndSec || r.HorizonEndSec <= eval.TailEnd(p, r.TakeoverSec, 0, 0) {
			t.Fatalf("%s: horizon %d, evaluation %d", r.Plan, r.HorizonEndSec, rep.HorizonEndSec)
		}
		for _, c := range r.Captures {
			if c.Money.EmployeeCommitmentCents <= 0 || c.Money.MinCashSec < r.TakeoverSec {
				t.Fatalf("%s %s: %+v", r.Plan, c.Name, c.Money)
			}
		}
		if r.Plan == "T" && r.LatestCommitmentSec == 0 {
			t.Fatal("no commitment recorded for the contract")
		}
	}
	// The same runs record the same bytes.
	again, _ := recordMini(t, p, 3, true, true)
	x, _ := json.Marshal(plans["T"][0])
	y, _ := json.Marshal(again)
	if !bytes.Equal(x, y) {
		t.Fatal("the record differs between identical runs")
	}
}
