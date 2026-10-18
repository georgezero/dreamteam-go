package eval

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"slices"
	"testing"
)

// Operating characteristics of the per-seed service rule (policy
// service.operatingCharacteristics, README §2): the probability that a plan
// whose underlying breach rates are the planning rates is rejected by the
// joint seed rule. Synthetic only: no simulation, no candidate result.
//
// The exact part assumes independent studies, Poisson volumes and seeds
// alike in their rates; the Monte Carlo part relaxes the last two (seed-
// to-seed variation and week-level clustering of breaches). Regenerate
// with DT_OC=1 go test -run OperatingCharacteristics ./internal/eval.

const ocPath = "../../docs/evaluation/oc-v0.2-eval-4.json"

// ocV3Path is Slice 5C.2's analysis, kept as written.
const ocV3Path = "../../docs/evaluation/oc-v0.2-eval-3.json"

// Daily volumes per seed, the range of the no-action development results'
// T90 acquisitions (README §2): read once for feasibility, never tuned.
var ocVolume = map[string][2]float64{"urgent": {0.79, 1.19}, "soon": {5.3, 6.5}, "routine": {43, 45}}

// ocDesign is a per-seed rule: the windows (days) in which each priority
// is decided per seed.
type ocDesign struct {
	Name     string               `json:"name"`
	TailDays int                  `json:"tailDays"`
	PerSeed  map[string][]float64 `json:"perSeedWindowDays"`
}

func ocDesigns() []ocDesign {
	sep := func(tail int) map[string][]float64 {
		t := float64(tail)
		return map[string][]float64{"urgent": {90, t}, "soon": {90, t}, "routine": {90, t}}
	}
	whole := func(tail int) map[string][]float64 {
		m := sep(tail)
		m["urgent"] = []float64{90 + float64(tail)}
		return m
	}
	return []ocDesign{
		{"v0.2-eval-1: every priority per seed in T90 and the tail", 60, sep(60)},
		{"v0.2-eval-2: urgent per seed over (t0, tailEnd]", 60, whole(60)},
		{"alternative: separate windows, 90-day tail", 90, sep(90)},
		{"alternative: urgent over (t0, tailEnd], 90-day tail", 90, whole(90)},
		{"alternative: urgent over (t0, tailEnd], 120-day tail", 120, whole(120)},
	}
}

type ocRow struct {
	Design  string  `json:"design"`
	Seeds   int     `json:"seeds"`
	Urgent  float64 `json:"urgentRate"`
	PerSeed float64 `json:"meanPerSeedFailure"`
	Reject  float64 `json:"rejection"`
}

type ocMC struct {
	Design string  `json:"design"`
	Tau    float64 `json:"seedSD"`    // SD of a seed's underlying urgent rate (scaled for the others)
	Rho    float64 `json:"weeklyICC"` // intra-week correlation of breaches
	Trials int     `json:"trials"`    // of 40 seeds each
	Reject float64 `json:"rejection"` // Monte Carlo, urgent at the planning rate
	SE     float64 `json:"standardError"`
}

type ocFile struct {
	Schema   string                `json:"schema"`
	Policy   string                `json:"policy"`
	Model    []string              `json:"model"`
	Gates    map[string]float64    `json:"gates"`
	Planning map[string]float64    `json:"planningRates"`
	Volume   map[string][2]float64 `json:"dailyVolume"`
	MinCount map[string]int64      `json:"minCohort"`
	Limit    float64               `json:"maxFailingSeedFraction"`
	Exact    []ocRow               `json:"exact"`
	MC       []ocMC                `json:"monteCarlo"`
	// Slice 5C.2: the service rule under the run's day and week structure.
	ClusterModel []string       `json:"clusterModel"`
	Models       []ocModel      `json:"clusterModels"`
	Cluster      []ocClusterRow `json:"cluster"`
	// Slice 5C.3: the backlog rule's family on synthetic queues.
	Backlog ocBacklogStudy `json:"backlog"`
}

func lpmf(n, k int, p float64) float64 {
	a, _ := math.Lgamma(float64(n + 1))
	b, _ := math.Lgamma(float64(k + 1))
	c, _ := math.Lgamma(float64(n - k + 1))
	return a - b - c + float64(k)*math.Log(p) + float64(n-k)*math.Log1p(-p)
}

// ocFailN is P(a cohort of n fails): fewer than the minimum, or more than
// floor(g·n) breaches (exactly the evaluator's rule).
func ocFailN(n int, p, g float64, minc int64) float64 {
	if int64(n) < minc {
		return 1
	}
	lim := int(bp(g) * int64(n) / 10_000)
	if lim >= n {
		return 0
	}
	// The upper tail from k = lim + 1, each term from the last; it stops
	// once past the mode the terms no longer matter.
	term := math.Exp(lpmf(n, lim+1, p))
	s, mode := term, float64(n)*p
	for k := lim + 1; k < n; k++ {
		term *= float64(n-k) / float64(k+1) * p / (1 - p)
		s += term
		if float64(k) > mode && term < s*1e-15 {
			break
		}
	}
	return s
}

// ocFailPois is P(fail) for a Poisson(mu) cohort.
func ocFailPois(mu, p, g float64, minc int64, cache map[int]float64) float64 {
	lo, hi := max(0, int(mu-8*math.Sqrt(mu)-5)), int(mu+8*math.Sqrt(mu)+5)
	var tot, mass float64
	for n := lo; n <= hi; n++ {
		lg, _ := math.Lgamma(float64(n + 1))
		pn := math.Exp(-mu + float64(n)*math.Log(mu) - lg)
		q, ok := cache[n]
		if !ok {
			q = ocFailN(n, p, g, minc)
			cache[n] = q
		}
		tot += pn * q
		mass += pn
	}
	return tot + (1 - mass) // outside ±8σ: counted as failing (conservative)
}

// poissonBinomialAbove is P(more than lim of the independent events).
func poissonBinomialAbove(qs []float64, lim int) float64 {
	dp := make([]float64, len(qs)+1)
	dp[0] = 1
	for _, q := range qs {
		for k := len(qs); k >= 1; k-- {
			dp[k] = dp[k]*(1-q) + dp[k-1]*q
		}
		dp[0] *= 1 - q
	}
	var s float64
	for k := lim + 1; k < len(dp); k++ {
		s += dp[k]
	}
	return s
}

// ocExact is the rejection probability of design d over S seeds whose
// volumes spread evenly over the range, every priority at its planning
// rate except urgent at pu.
func ocExact(p *Policy, d ocDesign, S int, pu float64) (perSeed, reject float64) {
	caches := map[string]map[int]float64{}
	qs := make([]float64, S)
	for i := range S {
		surv := 1.0
		for _, q := range Priorities {
			v := ocVolume[q]
			lam := v[0] + (v[1]-v[0])*float64(i)/float64(max(S-1, 1))
			rate := p.Service.PlanningRate[q]
			if q == "urgent" {
				rate = pu
			}
			for _, days := range d.PerSeed[q] {
				key := fmt.Sprintf("%s/%v", q, rate)
				if caches[key] == nil {
					caches[key] = map[int]float64{}
				}
				surv *= 1 - ocFailPois(lam*days, rate, p.Service.BreachGate[q], p.Service.MinCohort[q], caches[key])
			}
		}
		qs[i] = 1 - surv
		perSeed += qs[i] / float64(S)
	}
	return perSeed, poissonBinomialAbove(qs, failLimit(bp(p.Service.MaxFailingSeedFraction), S))
}

type ocRNG struct{ *rand.Rand }

func (r ocRNG) gamma(a float64) float64 {
	if a < 1 {
		return r.gamma(a+1) * math.Pow(r.Float64(), 1/a)
	}
	d := a - 1.0/3
	c := 1 / math.Sqrt(9*d)
	for {
		x := r.NormFloat64()
		v := 1 + c*x
		if v <= 0 {
			continue
		}
		v = v * v * v
		if math.Log(r.Float64()) < 0.5*x*x+d-d*v+d*math.Log(v) {
			return d * v
		}
	}
}

// beta draws from the beta distribution with mean m and SD sd.
func (r ocRNG) beta(m, sd float64) float64 {
	if sd <= 0 {
		return m
	}
	k := max(m*(1-m)/(sd*sd)-1, 1e-3)
	x, y := r.gamma(m*k), r.gamma((1-m)*k)
	return x / (x + y)
}

func (r ocRNG) poisson(mu float64) int {
	if mu > 50 {
		return max(0, int(math.Round(mu+math.Sqrt(mu)*r.NormFloat64())))
	}
	l, k, prod := math.Exp(-mu), 0, 1.0
	for {
		prod *= r.Float64()
		if prod <= l {
			return k
		}
		k++
	}
}

func (r ocRNG) binom(n int, p float64) int {
	if n > 200 {
		return min(max(0, int(math.Round(float64(n)*p+math.Sqrt(float64(n)*p*(1-p))*r.NormFloat64()))), n)
	}
	k := 0
	for range n {
		if r.Float64() < p {
			k++
		}
	}
	return k
}

// ocMonteCarlo simulates design d on 40-seed evaluations: each seed's
// underlying rate drawn around the planning rate (SD tau, scaled to the
// priority's rate), each week's rate around the seed's (intra-week
// correlation rho), Poisson volumes. Deterministic in its seed.
func ocMonteCarlo(p *Policy, d ocDesign, tau, rho float64, trials int) ocMC {
	r := ocRNG{rand.New(rand.NewPCG(20261017, uint64(len(d.Name))*1000+uint64(tau*1000)*10+uint64(rho*100)))}
	S := 40
	lim := failLimit(bp(p.Service.MaxFailingSeedFraction), S)
	rejected := 0
	for range trials {
		fails := 0
		for range S {
			failed := false
			for _, q := range Priorities {
				v := ocVolume[q]
				lam := v[0] + (v[1]-v[0])*r.Float64()
				rate := p.Service.PlanningRate[q]
				ps := r.beta(rate, tau*rate/p.Service.PlanningRate["urgent"])
				for _, days := range d.PerSeed[q] {
					n, b := 0, 0
					for w := 0; float64(w*7) < days; w++ {
						m := r.poisson(lam * math.Min(7, days-float64(w*7)))
						pw := ps
						if rho > 0 {
							pw = r.beta(ps, math.Sqrt(rho*ps*(1-ps)))
						}
						n, b = n+m, b+r.binom(m, pw)
					}
					if int64(n) < p.Service.MinCohort[q] || rateAbove(int64(b), int64(n), bp(p.Service.BreachGate[q])) {
						failed = true
					}
				}
			}
			if failed {
				fails++
			}
		}
		if fails > lim {
			rejected++
		}
	}
	x := float64(rejected) / float64(trials)
	return ocMC{Design: d.Name, Tau: tau, Rho: rho, Trials: trials, Reject: x, SE: math.Sqrt(x * (1 - x) / float64(trials))}
}

func ocExactRows(p *Policy) []ocRow {
	var rows []ocRow
	for _, d := range ocDesigns() {
		for _, S := range []int{10, 20, 40} {
			for _, pu := range []float64{0.06, 0.08, 0.10, 0.12, 0.14} {
				q, rej := ocExact(p, d, S, pu)
				rows = append(rows, ocRow{Design: d.Name, Seeds: S, Urgent: pu, PerSeed: round6(q), Reject: round6(rej)})
			}
		}
	}
	return rows
}

func round6(x float64) float64 { return math.Round(x*1e6) / 1e6 }

// The frozen design meets the owner's target under the planning model
// (urgent 8%, at most 5% false rejection on the 40-seed holdout) where
// v0.2-eval-1's did not, and the committed analysis is what this code
// computes.
func TestOperatingCharacteristics(t *testing.T) {
	p, _ := policyNow(t)
	rows := ocExactRows(p)
	find := func(prefix string, S int, pu float64) ocRow {
		for _, r := range rows {
			if len(r.Design) >= len(prefix) && r.Design[:len(prefix)] == prefix && r.Seeds == S && r.Urgent == pu {
				return r
			}
		}
		t.Fatalf("no row %s %d %v", prefix, S, pu)
		return ocRow{}
	}
	if r := find("v0.2-eval-2", 40, 0.08); r.Reject > 0.05 {
		t.Fatalf("the frozen design rejects an 8%% urgent plan with probability %.4f", r.Reject)
	}
	if r := find("v0.2-eval-1", 40, 0.08); r.Reject < 0.5 {
		t.Fatalf("eval-1 rejection %.4f: the defect it fixes is not shown", r.Reject)
	}
	if os.Getenv("DT_OC") != "" {
		f := ocFile{Schema: "dreamteam.evaloc/v3", Policy: p.ID, Gates: p.Service.BreachGate, Planning: p.Service.PlanningRate,
			Volume: ocVolume, MinCount: p.Service.MinCohort, Limit: p.Service.MaxFailingSeedFraction, Exact: rows,
			Model: []string{
				"A seed fails if any per-seed breach gate of the design fails or lacks evidence (fewer acquisitions than the minimum); a plan is rejected if more than floor(10% × seeds) seeds fail.",
				"Exact: independent studies; each seed's volume is Poisson at a daily rate spread evenly over the range; every seed at the planning rates (urgent as given).",
				"Monte Carlo: 40 seeds; each seed's underlying rate is beta-distributed around the planning rate with SD seedSD (scaled by rate ÷ urgent planning rate for soon and routine); each week's rate is beta-distributed around the seed's with intra-week correlation weeklyICC; volumes Poisson per week at a rate drawn uniformly from the range. Deterministic PCG streams.",
				"Pooled gates are not modelled: with 40 seeds their counts are far larger and pass at the planning rates.",
			}}
		for _, d := range ocDesigns()[1:] {
			if d.TailDays == 90 && len(d.PerSeed["urgent"]) == 2 {
				continue
			}
			for _, tau := range []float64{0, 0.01, 0.02} {
				for _, rho := range []float64{0, 0.02, 0.05} {
					f.MC = append(f.MC, ocMonteCarlo(p, d, tau, rho, 4000))
				}
			}
		}
		f.ClusterModel = []string{
			"Per seed, a 150-day gated period (T90 = days 1-90, tail = days 91-150), each priority's daily volume Poisson at a rate drawn uniformly from dailyVolume, times the weekday profile (Mon 1.1, Tue-Fri 1, Sat 0.25, Sun 0, scaled to a mean of 1) and a gamma day factor (dailyVolumeCV) where the model says so.",
			"Breach probability: each Monday-based week's rate beta-distributed around the setting's rate with intra-week correlation weekICC; each day's around the week's with intra-day correlation dayICC.",
			"The per-seed gates exactly as the policy (urgent over the gated period; soon and routine in T90 and the tail; evidence minimums). Seeds are independent and alike, so an evaluation's rejection is P(Binomial(seeds, perSeedFailure) > floor(10% x seeds)).",
			"The dependence range is bounded by the development diagnostics (docs/calibration/policy-5c2/clustering.md); no threshold, window or allowance was chosen from them. Deterministic PCG streams; regenerate with DT_OC=1.",
		}
		f.Models = ocModels()
		for mi, m := range f.Models {
			for si, s := range ocRateSettings(p) {
				f.Cluster = append(f.Cluster, ocClusterCompute(p, m, s, 4000, uint64(mi*100+si)))
			}
		}
		f.Backlog = ocBacklogCompute(p, ocBacklogSeeds)
		raw, _ := json.MarshalIndent(f, "", " ")
		if err := os.WriteFile(ocPath, append(raw, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	raw, err := os.ReadFile(ocPath)
	if err != nil {
		t.Fatal(err)
	}
	var f ocFile
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&f); err != nil {
		t.Fatal(err)
	}
	got, _ := json.Marshal(rows)
	want, _ := json.Marshal(f.Exact)
	if !bytes.Equal(got, want) || f.Policy != p.ID || len(f.MC) == 0 || len(f.Cluster) != len(ocModels())*len(ocRateSettings(p)) ||
		len(f.Backlog.Candidates) != len(ocFamily()) {
		t.Fatal("the committed operating characteristics are not what the code computes; regenerate with DT_OC=1")
	}
	// What the committed analysis shows, checked so a regeneration that
	// contradicts it fails: at the planning rates, rejection does not fall
	// as the weekly correlation rises; the workload adjustment never makes
	// a seasonal stable queue less stable; capacity loss is more often
	// unstable than stability.
	rej := map[string]float64{}
	for _, r := range f.Cluster {
		if r.Setting == "planning rates (false rejection)" {
			rej[r.Model] = r.Reject["40"]
		}
	}
	if !(rej["week-clustered, ICC 0.02"] <= rej["week-clustered, ICC 0.05"] && rej["week-clustered, ICC 0.05"] <= rej["week-clustered, ICC 0.10"]) {
		t.Fatalf("rejection by weekly correlation %v", rej)
	}
	// DT_OC_BACKLOG=1 recomputes the backlog analysis alone (about 40
	// minutes) and requires the committed one byte for byte.
	if os.Getenv("DT_OC_BACKLOG") != "" {
		got, _ := json.Marshal(ocBacklogCompute(p, ocBacklogSeeds))
		want, _ := json.Marshal(f.Backlog)
		if !bytes.Equal(got, want) {
			t.Fatal("the backlog analysis does not reproduce")
		}
	}
	// The backlog analysis: v0.2-eval-3 reproduces Slice 5C.2's committed
	// stable-queue result on its streams; the frozen policy is the
	// candidate the protocol selects, and it meets both targets.
	raw3, err := os.ReadFile(ocV3Path)
	if err != nil {
		t.Fatal(err)
	}
	var v3 struct {
		Backlog []struct {
			Scenario string             `json:"scenario"`
			Unstable map[string]float64 `json:"unstableSeedFraction"`
		} `json:"backlog"`
	}
	json.Unmarshal(raw3, &v3)
	for i, r := range f.Backlog.Reproduction {
		if old := v3.Backlog[i]; old.Scenario != r.Scenario || old.Unstable["v0.2-eval-3 (workload-adjusted, frozen)"] != r.Unstable {
			t.Fatalf("v0.2-eval-3 on the 5C.2 streams: %s %v, committed %s %v", r.Scenario, r.Unstable, old.Scenario, old.Unstable)
		}
	}
	c, ok := ocCandidateOf(p)
	if !ok || f.Backlog.Selected == "" {
		t.Fatalf("the policy's backlog rule is not a candidate of the family, or none was selected (%q)", f.Backlog.Selected)
	}
	sel := f.Backlog.SelectedRule
	if sel == nil || sel.Name != f.Backlog.Selected || !slices.Contains(f.Backlog.Candidates, sel.row()) || sel.LevelStat != c.LevelStat || sel.LevelGrowth != c.LevelGrowth || sel.LevelFloor != c.LevelFloor ||
		sel.LateStat != c.LateStat || sel.LateGrowth != c.LateGrowth || sel.Cap != c.Cap || !sel.T1 || !sel.T2 {
		t.Fatalf("the frozen backlog rule %+v is not the selected candidate %+v", c, sel)
	}
}
