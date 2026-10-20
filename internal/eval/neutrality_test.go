package eval

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
)

// v0.2-eval-4 neutrality (Slice 5C.4b): under the policy in force, the
// evaluation of a fixed, varied synthetic input is byte for byte what it
// was before the v0.2-eval-5 clinical rules were added (digest recorded
// from 31a2573, before any evaluator change). A v4 evaluation never
// reaches the new code.
const v4NeutralityDigest = "eef98fab6cd6355eb854b86675f914ecc8be02ba6cfcd5bd83f607dd98f0b070"

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
	if got := hex.EncodeToString(sum[:]); got != v4NeutralityDigest {
		t.Fatalf("the v0.2-eval-4 evaluation changed: digest %s, recorded %s", got, v4NeutralityDigest)
	}
}
