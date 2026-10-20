package sim_test

import (
	"testing"

	"github.com/georgezero/dreamteam-go/internal/eval"
)

// Slice 5C.4j (audit 5C.4i I4): the candidate's 45-day clinical follow-up
// only observes longer. A development seed recorded to the v4 policy's
// end of observation (31 days past the horizon) and to the candidate's
// (45 days) agrees on everything up to the earlier end: every order's
// identity, request, receipt, wanted time and, when resolved by then, its
// resolution and outcome at every capture before it; every study acquired
// by then, with its documentation, claim, read and final times when they
// fell by then. The simulator's decisions and random streams do not
// depend on how long the recorder watches.
func TestFollowUpPrefixUnchanged(t *testing.T) {
	if testing.Short() {
		t.Skip("short")
	}
	pol4, _, err := eval.LoadPolicy("../../" + eval.PolicyPath)
	if err != nil {
		t.Fatal(err)
	}
	cand, _, err := eval.LoadPolicy("../../" + candidatePolicy)
	if err != nil {
		t.Fatal(err)
	}
	cfg, fp := profileConfig(t)
	a, b := recordBaseline(pol4, cfg, fp, 901), recordBaseline(cand, cfg, fp, 901)
	E := a.ObservedUntilSec
	if b.ObservedUntilSec-b.HorizonEndSec != 45*86400 || E-a.HorizonEndSec != 31*86400 || a.HorizonEndSec != b.HorizonEndSec {
		t.Fatalf("observation: %d and %d days past the horizon", (E-a.HorizonEndSec)/86400, (b.ObservedUntilSec-b.HorizonEndSec)/86400)
	}
	// The 45-day record keeps the order–study contract: every study after
	// the takeover names an order of the record, including an inherited
	// no-show whose new date was already asked for (seed 901 has three).
	if m := eval.Measure(cand, &b); m.Clinical == nil {
		t.Fatalf("no clinical measures: %v", m.Integrity)
	}
	orders := map[string]eval.Order{}
	for _, o := range b.Orders {
		orders[o.ID] = o
	}
	endA, endB := a.Captures[len(a.Captures)-1].OrderOutcomes, map[string]string{}
	for i, o := range b.Orders {
		endB[o.ID] = b.Captures[len(b.Captures)-1].OrderOutcomes[i]
	}
	resolved := 0
	for i, o := range a.Orders {
		x, ok := orders[o.ID]
		if !ok || x.Origin != o.Origin || x.Priority != o.Priority || x.Modality != o.Modality || x.ReceivedSec != o.ReceivedSec || x.WantedSec != o.WantedSec {
			t.Fatalf("order %s differs: %+v / %+v", o.ID, o, x)
		}
		if o.ResolvedSec != 0 {
			resolved++
			if x.ResolvedSec != o.ResolvedSec || endB[o.ID] != endA[i] {
				t.Fatalf("order %s resolved differently: %d %s / %d %s", o.ID, o.ResolvedSec, endA[i], x.ResolvedSec, endB[o.ID])
			}
		}
	}
	for k, c := range a.Captures[:len(a.Captures)-1] {
		if c.AtSec != b.Captures[k].AtSec {
			t.Fatalf("capture %s at %d / %d", c.Name, c.AtSec, b.Captures[k].AtSec)
		}
		for i := range c.OrderOutcomes {
			if c.OrderOutcomes[i] != b.Captures[k].OrderOutcomes[i] {
				t.Fatalf("capture %s: order %d %s / %s", c.Name, i, c.OrderOutcomes[i], b.Captures[k].OrderOutcomes[i])
			}
		}
	}
	studies := map[string]eval.Study{}
	for _, s := range b.Studies {
		studies[s.ID] = s
	}
	by := func(t0 int64) int64 {
		if t0 > E {
			return 0
		}
		return t0
	}
	for _, s := range a.Studies {
		x, ok := studies[s.ID]
		if !ok || x.Order != s.Order || x.AcquiredSec != s.AcquiredSec || by(x.EnteredSec) != by(s.EnteredSec) || by(x.FinalSec) != s.FinalSec {
			t.Fatalf("study %s differs: %+v / %+v", s.ID, s, x)
		}
	}
	if len(b.Studies) <= len(a.Studies) || resolved < 1000 {
		t.Fatalf("%d / %d studies, %d resolved orders", len(a.Studies), len(b.Studies), resolved)
	}
	t.Logf("%d orders (%d resolved), %d studies identical through day %d; %d more studies in the follow-up", len(a.Orders), resolved, len(a.Studies),
		(E-a.TakeoverSec)/86400, len(b.Studies)-len(a.Studies))
}
