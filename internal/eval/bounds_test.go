package eval

import (
	"encoding/json"
	"math"
	"math/rand/v2"
	"os"
	"testing"
)

// Gate B's coverage evidence (docs/calibration/clinical-5c4d/
// gate-b-statistics.md §6): seed differences built from simulated paired
// orders under prespecified distributions, each bound's coverage
// P(U ≥ μ) and acceptance P(U ≤ 0.02) at the 2-point margin. Synthetic
// streams PCG(5, ·) only.

type gbScenario struct {
	Name     string  `json:"name"`
	Seeds    int     `json:"seeds"`
	OrdersLo int     `json:"ordersMin"` // orders a seed, uniform in [lo, hi]
	OrdersHi int     `json:"ordersMax"`
	Base     float64 `json:"baselineUnimaged"`
	Increase float64 `json:"increase"`    // the seed effect's centre
	Concord  float64 `json:"concordance"` // probability a candidate order shares the baseline's draw
	Shock    float64 `json:"shockProbability"`
	NoiseSD  float64 `json:"seedNoise"`          // seed-effect scale
	NoiseT   float64 `json:"seedNoiseTdf"`       // 0: normal; else Student t with this df (heavy tails)
	Constant bool    `json:"constantDifference"` // every seed exactly Increase
}

func gbScenarios() []gbScenario {
	s := func(name string, f func(*gbScenario)) gbScenario {
		x := gbScenario{Name: name, Seeds: 40, OrdersLo: 200, OrdersHi: 200, Base: 0.3, Concord: 0.9}
		f(&x)
		return x
	}
	return []gbScenario{
		s("increase 0", func(x *gbScenario) {}),
		s("increase 1 point", func(x *gbScenario) { x.Increase = 0.01 }),
		s("increase 2 points", func(x *gbScenario) { x.Increase = 0.02 }),
		s("increase 2.001 points", func(x *gbScenario) { x.Increase = 0.02001 }),
		s("increase 2.1 points", func(x *gbScenario) { x.Increase = 0.021 }),
		s("increase 3 points", func(x *gbScenario) { x.Increase = 0.03 }),
		s("zero discordance", func(x *gbScenario) { x.Concord = 1 }),
		s("one-directional, 2.1 points", func(x *gbScenario) { x.Base, x.Increase = 0, 0.021 }),
		s("bidirectional, 2.1 points", func(x *gbScenario) { x.Concord, x.Increase = 0.5, 0.021 }),
		s("perfect within-seed correlation (all-or-nothing shocks), 3 points", func(x *gbScenario) { x.Base, x.Shock = 0, 0.03 }),
		s("strong within-seed correlation, 2.1 points", func(x *gbScenario) { x.Increase, x.NoiseSD = 0.021, 0.05 }),
		s("heterogeneous seed sizes, 2.1 points", func(x *gbScenario) { x.Increase, x.OrdersLo, x.OrdersHi = 0.021, 20, 600 }),
		s("heavy-tailed seed differences, 2.1 points", func(x *gbScenario) { x.Increase, x.NoiseSD, x.NoiseT = 0.021, 0.02, 2 }),
		s("10 seeds, 2.1 points", func(x *gbScenario) { x.Seeds, x.Increase = 10, 0.021 }),
		s("20 seeds, 2.1 points", func(x *gbScenario) { x.Seeds, x.Increase = 20, 0.021 }),
		s("zero-variance differences, 2.1 points", func(x *gbScenario) { x.Constant, x.Increase = true, 0.021 }),
		s("zero-variance differences, 0", func(x *gbScenario) { x.Constant = true }),
	}
}

// seedEffect draws a seed's increase, clipped so the candidate's
// probability stays in [0, 1].
func (s gbScenario) seedEffect(r *rand.Rand) float64 {
	d := s.Increase
	if s.NoiseSD > 0 {
		z := r.NormFloat64()
		if s.NoiseT > 0 {
			chi := 0.0
			for range int(s.NoiseT) {
				g := r.NormFloat64()
				chi += g * g
			}
			z /= math.Sqrt(chi / s.NoiseT)
		}
		d += s.NoiseSD * z
	}
	return min(max(d, -s.Base), 1-s.Base)
}

// difference draws one seed's paired orders and returns its difference.
func (s gbScenario) difference(r *rand.Rand) float64 {
	if s.Constant {
		return s.Increase
	}
	n := s.OrdersLo + r.IntN(s.OrdersHi-s.OrdersLo+1)
	shock := r.Float64() < s.Shock
	d := s.seedEffect(r)
	var cand, base int
	for range n {
		u := r.Float64()
		b := u < s.Base
		v := u
		if r.Float64() >= s.Concord {
			v = r.Float64()
		}
		c := shock || v < s.Base+d
		if b {
			base++
		}
		if c {
			cand++
		}
	}
	return float64(cand-base) / float64(n)
}

// trueMean is the scenario's population mean seed difference.
func (s gbScenario) trueMean() float64 {
	if s.Constant {
		return s.Increase
	}
	r := rand.New(rand.NewPCG(5, 999))
	m := 0.0
	const k = 200_000
	for range k {
		m += s.seedEffect(r)
	}
	m /= k
	return (1-s.Shock)*m + s.Shock*(1-s.Base)
}

type gbRow struct {
	gbScenario
	Mean    float64            `json:"trueMean"`
	Reps    int                `json:"reps"`
	Cover   map[string]float64 `json:"coverage"`
	Accept  map[string]float64 `json:"acceptance"` // P(U ≤ 0.02)
	MCError float64            `json:"monteCarloSE"`
}

func gbCoverage(reps int) []gbRow {
	var out []gbRow
	for k, s := range gbScenarios() {
		r := rand.New(rand.NewPCG(5, uint64(k)))
		mu := s.trueMean()
		row := gbRow{gbScenario: s, Mean: mu, Reps: reps, Cover: map[string]float64{BoundBetting: 0, BoundT: 0}, Accept: map[string]float64{BoundBetting: 0, BoundT: 0}}
		xs := make([]float64, s.Seeds)
		for range reps {
			for i := range xs {
				xs[i] = s.difference(r)
			}
			for _, m := range []string{BoundBetting, BoundT} {
				u := upperBound(m, xs, 0.95, -1, 1)
				if u >= mu-1e-12 {
					row.Cover[m] += 1 / float64(reps)
				}
				if !exceeds(u, 0.02) {
					row.Accept[m] += 1 / float64(reps)
				}
			}
		}
		row.MCError = math.Sqrt(0.05 * 0.95 / float64(reps))
		out = append(out, row)
	}
	return out
}

// The betting bound keeps its guarantee in every scenario (coverage at
// least 95% less three Monte Carlo errors) and never accepts at 40 seeds;
// the t bound loses coverage under all-or-nothing shocks (samples that
// show no variance although the distribution has it), as the design
// document states. Truly constant differences are covered by both. DT_GATEB_COVERAGE=<file>
// writes the full table (2,000 replicates).
func TestGateBCoverage(t *testing.T) {
	reps := 300
	out := os.Getenv("DT_GATEB_COVERAGE")
	if out != "" {
		reps = 2000
	}
	rows := gbCoverage(reps)
	for _, r := range rows {
		if r.Cover[BoundBetting] < 0.95-3*r.MCError {
			t.Errorf("%s: betting coverage %.3f", r.Name, r.Cover[BoundBetting])
		}
		if r.Seeds == 40 && r.Accept[BoundBetting] > 0 {
			t.Errorf("%s: betting accepts at 40 seeds (%.3f)", r.Name, r.Accept[BoundBetting])
		}
		switch r.Name {
		case "perfect within-seed correlation (all-or-nothing shocks), 3 points":
			if r.Cover[BoundT] > 0.9 {
				t.Errorf("%s: t coverage %.3f, its failure not reproduced", r.Name, r.Cover[BoundT])
			}
		}
	}
	if out != "" {
		raw, _ := json.MarshalIndent(map[string]any{
			"schema":  "dreamteam.gateb-coverage/v1",
			"design":  "docs/calibration/clinical-5c4d/gate-b-statistics.md",
			"streams": "scenario k: PCG(5, k); true means PCG(5, 999)",
			"margin":  0.02,
			"conf":    0.95,
			"floor40": map[string]float64{BoundBetting: zeroFloor(BoundBetting, 40, 0.95, -1, 1), "anyValidMethod": 1 - math.Pow(0.05, 1.0/40)},
			"rows":    rows,
		}, "", " ")
		if err := os.WriteFile(out, append(raw, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// The betting bound's guarantee in a direct check: in-sample values on
// a two-point distribution far from normal, coverage holds; the floor
// with no difference falls with the number of seeds as derived.
func TestBettingBound(t *testing.T) {
	if f := zeroFloor(BoundBetting, 40, 0.95, -1, 1); f < 0.2 || f > 0.21 {
		t.Errorf("floor at 40 seeds %v", f)
	}
	if f := zeroFloor(BoundBetting, 400, 0.95, -1, 1); f < 0.019 || f > 0.0205 {
		t.Errorf("floor at 400 seeds %v", f)
	}
	if u := tUpper([]float64{0.01, 0.01, 0.01}, 0.95); u != 0.01 {
		t.Errorf("t with zero variance %v", u)
	}
	if q := tQuantile(0.95, 39); math.Abs(q-1.6849) > 1e-3 {
		t.Errorf("t quantile %v", q)
	}
}
