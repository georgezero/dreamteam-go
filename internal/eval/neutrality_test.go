package eval

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"runtime"
	"testing"
)

// v0.2-eval-4 neutrality (Slice 5C.4b): under the policy in force, the
// evaluation of a fixed, varied synthetic input is byte for byte what it
// was before the v0.2-eval-5 clinical rules were added (digest recorded
// from 31a2573, before any evaluator change). A v4 evaluation never
// reaches the new code.
//
// By processor family (0.3.0-dev.2): the report's exact Clopper–Pearson
// bound goes through the Go math library (Lgamma, Exp, Log), whose last
// digit differs between arm64 and amd64, so one digest cannot hold on
// both. The digest recorded at 31a2573 (eef98fab…) was arm64's and never
// held on amd64, whose output (8c1b0076…) the recorder has kept unchanged
// since. arm64's moved in 0.3.0-dev.2 when the reported statistics began
// rounding every product before a sum (stats.go). Every status and
// verdict is the same on both.
var v4NeutralityDigests = map[string]string{
	"amd64": "8c1b0076b644a98ccd8e031f31e30db2d678a7cd0783d0559979793c4058f036",
	"arm64": "c492dbc858433fce5e958cc65ae31e0ac21d1e5a65951d8f344c13566d0066d1",
}

func neutralityInput() Input {
	mk := func(plan string, f func(s *synth)) []Record {
		return records(10, func(seed int64) synth {
			s := base(plan, seed)
			s.profitNoise, s.followUp = 500, 0
			f(&s)
			return s
		})
	}
	return Input{Baseline: "A", Primary: "B", Pairs: [][2]string{{"B", "C"}}, Plans: map[string][]Record{
		"A": mk("A", func(s *synth) {
			s.breachPM = map[string]int{"urgent": 160, "soon": 380, "routine": 80}
			s.inherited = 40
		}),
		"B": mk("B", func(s *synth) {
			s.profitPD, s.upfront, s.inherited = 9_000, 200_000, 40
			s.bursts = []burst{{q: "soon", from: 100, to: 110, pm: 900}}
			s.late = []lateOrders{{n: 2, recv: 40, want: 41, resolveDay: 60, out: OutServed}, {n: 1, recv: 70, want: 70, out: OutOpen}}
			s.walkIns, s.walkLeft = 1, 100
		}),
		"C": mk("C", func(s *synth) {
			s.profitPD, s.inherited, s.remotePM, s.fee, s.retainerPD = 12_000, 40, 300, 2_500, 1_000
			s.declinePM = map[string]int{"routine": 30}
			s.neverPM = map[string]int{"routine": 200}
			s.rare = 9
		}),
		"D": mk("D", func(s *synth) { s.profitPD, s.inherited = 14_000, 40 }),
	}}
}

func TestV4Neutrality(t *testing.T) {
	p, d := testPolicy(t)
	raw, err := json.Marshal(Evaluate(p, d, neutralityInput()))
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	want, ok := v4NeutralityDigests[runtime.GOARCH]
	if !ok {
		t.Skipf("no digest recorded for %s", runtime.GOARCH)
	}
	if got := hex.EncodeToString(sum[:]); got != want {
		t.Fatalf("the v0.2-eval-4 evaluation changed: digest %s, recorded %s for %s", got, want, runtime.GOARCH)
	}
}
