package eval

import (
	"encoding/json"
	"math"
	"math/rand/v2"
	"os"
	"runtime"
	"slices"
	"strconv"
	"sync"
	"testing"
)

// The per-seed safeguard's development evidence (Slice 5C.4e,
// docs/calibration/clinical-5c4e/seed-safeguard.md).
//
// 1. The proposed thresholds from the prespecified rule: for each
//    priority, max(5 points, the whole points above z(0.999) × the
//    largest sd of a no-effect seed difference among the development
//    seeds), the sd under independent outcomes sqrt(2u(1 − u)/n), with u
//    and n the no-action plan's Gate B unimaged fraction and eligible
//    orders (sim-access-5c4e.json, seeds 901–910).
// 2. The synthetic fidelity practice with paired demand (development
//    stream PCG(20261020, ·), never a reserved stream): the no-effect seed
//    differences (the same practice, another booking stream), the
//    differences under imaging-capacity cuts, and an injected urgent loss
//    on one seed on top of the no-effect difference.

const safeguardStream = 20261020

// safeguardZ is the one-sided 0.1% normal quantile.
const safeguardZ = 3.0902

// safeguardFloor is the proposal's clinical floor: well above the mean
// rule's 2-point margin (the per-seed rule must not repeat it).
const safeguardFloor = 0.05

// safeguardRule is the prespecified threshold for one priority from the
// development seeds' eligible orders n and unimaged k.
func safeguardRule(n, k []int64) (tau, sdMax float64) {
	for i := range n {
		u := float64(k[i]) / float64(n[i])
		sdMax = max(sdMax, math.Sqrt(2*u*(1-u)/float64(n[i])))
	}
	return max(safeguardFloor, math.Ceil(safeguardZ*sdMax*100-1e-9)/100), sdMax
}

func TestSafeguardRule(t *testing.T) {
	// One seed, u = 0.5, n = 400: sd 0.0354, z × sd 0.109 → 11 points.
	if tau, sd := safeguardRule([]int64{400}, []int64{200}); math.Abs(sd-0.035355) > 1e-5 || tau != 0.11 {
		t.Fatalf("rule: %v %v", tau, sd)
	}
	// Large n: the floor.
	if tau, _ := safeguardRule([]int64{100000}, []int64{20000}); tau != safeguardFloor {
		t.Fatalf("floor: %v", tau)
	}
}

// TestSafeguardThresholdsFromDevelopment: the candidate's maxSeedIncrease
// is the rule's value on the committed development evidence.
func TestSafeguardThresholdsFromDevelopment(t *testing.T) {
	raw, err := os.ReadFile("../../docs/calibration/clinical-5c4e/sim-access-5c4e.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Seeds []struct {
			Seed     int64               `json:"seed"`
			Eligible map[string][2]int64 `json:"gateBEligibleUnimaged"`
		} `json:"seeds"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	cand, _ := candidatePolicy(t)
	if len(doc.Seeds) != 10 {
		t.Fatalf("%d development seeds", len(doc.Seeds))
	}
	for _, q := range Priorities {
		var n, k []int64
		for _, s := range doc.Seeds {
			if s.Seed < 901 || s.Seed > 910 {
				t.Fatalf("seed %d is not a development seed", s.Seed)
			}
			n, k = append(n, s.Eligible[q][0]), append(k, s.Eligible[q][1])
		}
		tau, sd := safeguardRule(n, k)
		t.Logf("%s: largest no-effect sd %.4f, threshold %.2f", q, sd, tau)
		if got := cand.Clinical.LostDemand.MaxSeedIncrease[q]; got != tau {
			t.Errorf("%s: candidate maxSeedIncrease %v, the rule gives %v", q, got, tau)
		}
	}
}

type sgScenario struct {
	Name string     `json:"name"`
	Desc string     `json:"description"`
	Cand ocFidelity `json:"candidate"`
	// Inject: an urgent loss injected on every seed's candidate (the
	// fraction of its eligible urgent orders served by the baseline, made
	// lost), on top of the no-effect difference.
	Inject float64 `json:"injectedUrgentLoss,omitempty"`
}

type sgQuant struct {
	Orders   float64            `json:"ordersMean"`
	Unimaged float64            `json:"baselineUnimagedMean"`
	Mean     float64            `json:"differenceMean"`
	SD       float64            `json:"differenceSD"`
	Q        map[string]float64 `json:"differenceQuantiles"`
	Flag     map[string]float64 `json:"flaggedFraction"` // seeds above each threshold
}

func TestSeedSafeguardEvidence(t *testing.T) {
	out := os.Getenv("DT_SEED_SAFEGUARD")
	if out == "" {
		t.Skip("set DT_SEED_SAFEGUARD=<file> (and DT_SEED_SAFEGUARD_N, default 200) for the per-seed safeguard's evidence")
	}
	n := 200
	if v, err := strconv.Atoi(os.Getenv("DT_SEED_SAFEGUARD_N")); err == nil && v > 0 {
		n = v
	}
	pol := ocClinicalPolicy(t, ocBundles[1])
	cand, _ := candidatePolicy(t)
	p := *cand
	c := *cand.Clinical
	ld := *c.LostDemand
	ld.MinSeeds = 1
	c.LostDemand, p.Clinical = &ld, &c
	taus := []float64{0.02, 0.03, 0.04, 0.05, 0.06, 0.08, 0.10, 0.12, 0.15}
	var doc []map[string]any
	// One reading regime: the fidelity practice's booking, and with it
	// every unimaged outcome, does not depend on reading (fidImgAdjust),
	// so the 0.92 regime gives the same differences.
	for _, u := range []float64{0.85} {
		base := ocFidelity{Util: u, Takeover: fidAlignedTakeover}
		with := func(f func(*ocFidelity)) ocFidelity { q := base; f(&q); return q }
		scen := []sgScenario{
			{Name: "null", Desc: "no effect: the same practice with another booking stream", Cand: base},
			{Name: "IL10", Desc: "imaging capacity 10% lower from day 30", Cand: with(func(q *ocFidelity) { q.ImgCap = []ocCap{{From: 30, Factor: 0.9}} })},
			{Name: "IL20", Desc: "imaging capacity 20% lower from day 30", Cand: with(func(q *ocFidelity) { q.ImgCap = []ocCap{{From: 30, Factor: 0.8}} })},
			{Name: "U05", Desc: "no effect plus 5% of the urgent orders served by the baseline lost", Cand: base, Inject: 0.05},
			{Name: "U10", Desc: "no effect plus 10% injected urgent loss", Cand: base, Inject: 0.10},
			{Name: "U15", Desc: "no effect plus 15% injected urgent loss", Cand: base, Inject: 0.15},
			{Name: "U20", Desc: "no effect plus 20% injected urgent loss", Cand: base, Inject: 0.20},
		}
		// Seed by seed, the baseline's record and each scenario's
		// candidate; nothing is kept but the differences (1 000 records
		// in memory exhausted the development machine).
		type cell struct{ diff, orders, unim float64 }
		cells := make([][]map[string]cell, len(scen)) // scenario → seed → priority
		for k := range scen {
			cells[k] = make([]map[string]cell, n)
		}
		parallelFor(n, func(i int) {
			br, _ := base.recordPaired(ocRNG{rand.New(rand.NewPCG(safeguardStream, uint64(2*i)))}, ocRNG{rand.New(rand.NewPCG(safeguardStream, uint64(2*i+1)))}, pol)
			br.Seed = int64(i)
			for k, sc := range scen {
				// Another booking stream for the candidate: the no-effect
				// path divergence is in every scenario.
				r, _ := sc.Cand.recordPaired(ocRNG{rand.New(rand.NewPCG(safeguardStream, uint64(2*i)))}, ocRNG{rand.New(rand.NewPCG(safeguardStream+1, uint64(i)))}, pol)
				r.Seed = int64(i)
				if sc.Inject > 0 {
					injectUrgentLoss(r, br, &p, sc.Inject)
				}
				cells[k][i] = map[string]cell{}
				for _, g := range lostDemandGates(&p, []Record{*r}, []Record{*br}) {
					if len(g.Seeds) != 1 {
						t.Errorf("%s seed %d %s: not paired (%s)", sc.Name, i, g.Priority, g.Reason)
						continue
					}
					s := g.Seeds[0]
					cells[k][i][g.Priority] = cell{s.Difference, float64(s.Orders), float64(s.Baseline) / float64(s.Orders)}
				}
			}
		})
		for k, sc := range scen {
			diffs := map[string][]float64{}
			orders, unim := map[string]float64{}, map[string]float64{}
			for i := range n {
				for q, c := range cells[k][i] {
					diffs[q] = append(diffs[q], c.diff)
					orders[q] += c.orders / float64(n)
					unim[q] += c.unim / float64(n)
				}
			}
			res := map[string]sgQuant{}
			for _, q := range Priorities {
				xs := slices.Clone(diffs[q])
				slices.Sort(xs)
				m, sd := meanSD(xs)
				at := func(f float64) float64 { return xs[min(len(xs)-1, int(math.Ceil(f*float64(len(xs))))-1)] }
				x := sgQuant{Orders: orders[q], Unimaged: unim[q], Mean: m, SD: sd,
					Q:    map[string]float64{"min": xs[0], "p50": at(0.5), "p90": at(0.9), "p99": at(0.99), "p999": at(0.999), "max": xs[len(xs)-1]},
					Flag: map[string]float64{}}
				for _, tau := range append(slices.Clone(taus), cand.Clinical.LostDemand.MaxSeedIncrease[q]) {
					k := 0
					for _, d := range xs {
						if d > tau+1e-12 {
							k++
						}
					}
					x.Flag[strconv.FormatFloat(tau, 'f', -1, 64)] = float64(k) / float64(len(xs))
				}
				res[q] = x
			}
			t.Logf("%.2f %s: urgent %+v", u, sc.Name, res["urgent"].Q)
			doc = append(doc, map[string]any{"regime": u, "scenario": sc, "byPriority": res})
		}
	}
	raw, err := json.MarshalIndent(map[string]any{
		"schema": "dreamteam.seedsafeguard/v1",
		"source": "synthetic fidelity practice with paired demand (internal/eval/seed_safeguard_test.go), development stream PCG(20261020, ·); no simulator strategy, reference, holdout or reserved stream",
		"seeds":  n, "thresholds": cand.Clinical.LostDemand.MaxSeedIncrease, "scenarios": doc,
	}, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

// injectUrgentLoss makes lost, on the candidate's record, the given
// fraction of the eligible urgent orders the baseline served and the
// candidate served too (by origin; the earliest first).
func injectUrgentLoss(r, base *Record, p *Policy, frac float64) {
	eb, _ := eligibleDemand(p, base, "urgent")
	end := endCapture(r)
	var idx []int
	for i, o := range r.Orders {
		if b, ok := eb[o.Origin]; ok && b.out == OutServed && end.OrderOutcomes[i] == OutServed {
			idx = append(idx, i)
		}
	}
	k := int(math.Round(frac * float64(len(eb))))
	for _, i := range idx[:min(k, len(idx))] {
		dropStudy(r, i)
		r.Orders[i].ResolvedSec, r.Orders[i].FirstApptSec = r.Orders[i].ReceivedSec, 0
		end.OrderOutcomes[i] = OutLost
	}
}

// parallelFor runs f(0) … f(n − 1) on every CPU.
func parallelFor(n int, f func(i int)) {
	var wg sync.WaitGroup
	work := make(chan int)
	for range runtime.NumCPU() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range work {
				f(i)
			}
		}()
	}
	for i := range n {
		work <- i
	}
	close(work)
	wg.Wait()
}
