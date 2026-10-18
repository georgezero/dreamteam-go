package eval

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"sort"
	"strings"
	"sync"
)

// Slice 5C.3's backlog operating characteristics (oc-v0.2-eval-4.json,
// backlog; docs/calibration/backlog-5c3/protocol.md): every candidate rule
// of the prespecified family, and v0.2-eval-3, judged by the production
// code (judgeBacklog) on synthetic queues that are stable or deteriorate
// in a stated way; the selection the protocol fixes; and the attribution
// of v0.2-eval-3's false failures by check. Synthetic only.

// ocSeason is the scenario's monthly season factor (scenario demand
// seasonFactorBP, January first).
var ocSeason = [12]float64{1.05, 1.00, 1.00, 1.00, 0.98, 0.95, 0.90, 0.92, 1.00, 1.02, 1.00, 0.93}

type ocBacklogScenario struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Role  string  `json:"role"` // stable (T1), deteriorating (T2), ladder or sensitivity (reported)
	Queue ocQueue `json:"queue"`
}

func ocBacklogScenarios() []ocBacklogScenario {
	seasonal := func(q ocQueue) ocQueue { q.Seasonal = true; return q }
	lower := func(from, to int, f float64) []ocCap { return []ocCap{{From: from, To: to, Factor: f}} }
	return []ocBacklogScenario{
		{"A", "stable stationary queue (utilization 0.85)", "stable", ocQueue{}},
		{"B", "stable seasonal queue", "stable", seasonal(ocQueue{})},
		{"C", "stable queue with bursty days (day CV 0.15)", "stable", ocQueue{DayCV: 0.15}},
		{"Cs", "stable seasonal queue with bursty days (day CV 0.15)", "stable", seasonal(ocQueue{DayCV: 0.15})},
		{"D", "sustained capacity deterioration (20% lower from day 100)", "deteriorating", seasonal(ocQueue{Capacity: lower(100, 0, 0.80)})},
		{"E", "persistent inherited overdue work (10 studies unread to day 120)", "deteriorating", seasonal(ocQueue{Inherit: 10, InheritTo: 120})},
		{"F", "rising age without count growth (one study a weekday held 12 days from day 100)", "deteriorating", seasonal(ocQueue{ParkFrom: 100, ParkTo: 181, ParkDays: 12})},
		{"G", "intermediate deterioration then recovery (25% lower, days 90-117)", "deteriorating", seasonal(ocQueue{Capacity: lower(90, 118, 0.75)})},
		{"H", "late-tail deterioration (20% lower from day 140)", "deteriorating", seasonal(ocQueue{Capacity: lower(140, 0, 0.80)})},
		{"I", "reference inflation (30% lower in the reference block, then 20% lower from day 111)", "deteriorating",
			seasonal(ocQueue{Capacity: []ocCap{{From: 55, To: 83, Factor: 0.70}, {From: 111, Factor: 0.80}}})},
		{"L5", "capacity 5% lower from day 100 (still stable)", "ladder", seasonal(ocQueue{Capacity: lower(100, 0, 0.95)})},
		{"L10", "capacity 10% lower from day 100 (still stable)", "ladder", seasonal(ocQueue{Capacity: lower(100, 0, 0.90)})},
		{"L15", "capacity 15% lower from day 100 (utilization 1.0)", "sensitivity", seasonal(ocQueue{Capacity: lower(100, 0, 0.85)})},
		{"A75", "stable stationary queue, utilization 0.75", "sensitivity", ocQueue{Util: 0.75}},
		{"A92", "stable stationary queue, utilization 0.92", "sensitivity", ocQueue{Util: 0.92}},
		{"B92", "stable seasonal queue, utilization 0.92", "sensitivity", seasonal(ocQueue{Util: 0.92})},
		{"Cx", "stable seasonal queue with very bursty days (day CV 0.30)", "sensitivity", seasonal(ocQueue{DayCV: 0.30})},
		{"K", "slow erosion: capacity 0.2% lower each day from day 83 (utilization 1.0 by day 158)", "sensitivity", seasonal(ocQueue{ErodeFrom: 83, ErodePerDay: 0.002})},
	}
}

// ocCandidate is one rule of the family: the protocol's parameters on the
// policy's fixed structure. Unstable counts each scenario's unstable
// seeds, in scenario order.
type ocCandidate struct {
	Name        string  `json:"name"`
	LevelStat   string  `json:"levelStatistic"`
	LevelGrowth float64 `json:"levelGrowthFraction"`
	LevelFloor  int64   `json:"levelGrowthStudies"`
	LateStat    string  `json:"lateWorkStatistic"`
	LateGrowth  float64 `json:"growthFraction"`
	Cap         int64   `json:"staleCapStudies"`
	Unstable    []int   `json:"unstableSeeds"`
	T1          bool    `json:"meetsT1"`
	T2          bool    `json:"meetsT2"`
	Gray        float64 `json:"ladderDetection"`
}

// ocFamily is the protocol's family, in its tie-break order.
func ocFamily() []ocCandidate {
	var out []ocCandidate
	for _, ls := range []string{"mean", "median"} {
		for _, ws := range []string{"mean", "median"} {
			for _, g := range []float64{0.10, 0.25, 0.50, 1.00, 1.25, 1.50, 1.75, 2.00, 2.50, 3.00} {
				for _, f := range []int64{2, 5} {
					for _, l := range []float64{0.10, 0.25, 0.50} {
						for _, c := range []int64{0, 3, 2} {
							out = append(out, ocCandidate{Name: fmt.Sprintf("level %s +%.0f%%/%d, late %s +%.0f%%, cap %d", ls, 100*g, f, ws, 100*l, c),
								LevelStat: ls, LevelGrowth: g, LevelFloor: f, LateStat: ws, LateGrowth: l, Cap: c})
						}
					}
				}
			}
		}
	}
	return out
}

// ocWithBacklog is p with a backlog rule's own settings: the structure
// every candidate shares (level, overdue and exposure workload-adjusted;
// stale, the cap and persistence absolute; the mean and peak reported
// only).
func ocWithBacklog(p *Policy, c ocCandidate) *Policy {
	q := *p
	q.Backlog.LevelStatistic, q.Backlog.LateStatistic = c.LevelStat, c.LateStat
	q.Backlog.LevelGrowth, q.Backlog.LevelGrowthStudies = c.LevelGrowth, c.LevelFloor
	q.Backlog.Growth, q.Backlog.GrowthStudies, q.Backlog.StaleCapStudies = c.LateGrowth, 2, c.Cap
	q.Backlog.WorkloadAdjustedChecks = []string{"exposure", "level", "max", "mean", "overdue"}
	q.Backlog.AbsoluteChecks = []string{"persistent", "stale", "staleCap"}
	q.Backlog.InformationalChecks = []string{"max", "mean"}
	return &q
}

// ocEval3 is v0.2-eval-3's rule in the same code: the mean (10% / 2
// studies) and, unless dropped, the peak hard; late work by means; no cap;
// the level check reported only.
func ocEval3(p *Policy, peak bool) *Policy {
	q := ocWithBacklog(p, ocCandidate{LevelStat: "mean", LateStat: "mean", LevelGrowth: 0.10, LevelFloor: 2, LateGrowth: 0.10})
	q.Backlog.InformationalChecks = []string{"level"}
	if !peak {
		q.Backlog.InformationalChecks = []string{"level", "max"}
	}
	return q
}

// ocFailed is the set of hard checks a judged seed fails.
func ocFailed(bm *BacklogMeasures) map[string]bool {
	f := map[string]bool{}
	for _, c := range bm.Checks {
		if !c.Pass && !c.Informational {
			f[c.Check] = true
		}
	}
	if bm.Persistent > 0 {
		f["persistent"] = true
	}
	return f
}

// ocAttribution is v0.2-eval-3's unstable seeds of one scenario by the
// checks that failed: in any later block, an intermediate one, or the
// block covering the horizon (the residual period); and each exact set of
// failed checks (the overlaps).
type ocAttribution struct {
	Scenario string                        `json:"scenario"`
	Streams  string                        `json:"streams"`
	Seeds    int                           `json:"seeds"`
	Unstable float64                       `json:"unstableSeedFraction"`
	ByCheck  map[string]map[string]float64 `json:"failingSeedFractionByCheck"` // check → any, intermediate, covering
	Overlaps map[string]float64            `json:"failingSeedFractionByCheckSet"`
}

type ocAttributor struct {
	a        ocAttribution
	n        map[string]map[string]int
	overlaps map[string]int
}

func newAttributor(scenario, streams string) *ocAttributor {
	return &ocAttributor{a: ocAttribution{Scenario: scenario, Streams: streams}, n: map[string]map[string]int{}, overlaps: map[string]int{}}
}

func (t *ocAttributor) add(bm *BacklogMeasures) {
	t.a.Seeds++
	if bm.Status != "unstable" {
		return
	}
	t.a.Unstable++
	cover := bm.Later[len(bm.Later)-1].FirstDay
	seen := map[string]bool{}
	mark := func(check, where string) {
		if seen[check+where] {
			return
		}
		seen[check+where] = true
		if t.n[check] == nil {
			t.n[check] = map[string]int{}
		}
		t.n[check][where]++
	}
	for _, c := range bm.Checks {
		if c.Pass || c.Informational {
			continue
		}
		mark(c.Check, "any")
		if c.FirstDay == cover {
			mark(c.Check, "covering")
		} else if c.FirstDay != bm.Reference.FirstDay {
			mark(c.Check, "intermediate")
		}
	}
	if bm.Persistent > 0 {
		mark("persistent", "any")
	}
	var set []string
	for c := range ocFailed(bm) {
		set = append(set, c)
	}
	sort.Strings(set)
	t.overlaps[strings.Join(set, "+")]++
}

func (t *ocAttributor) done() ocAttribution {
	n := float64(t.a.Seeds)
	t.a.Unstable = round6(t.a.Unstable / n)
	t.a.ByCheck, t.a.Overlaps = map[string]map[string]float64{}, map[string]float64{}
	for c, m := range t.n {
		t.a.ByCheck[c] = map[string]float64{}
		for w, k := range m {
			t.a.ByCheck[c][w] = round6(float64(k) / n)
		}
	}
	for s, k := range t.overlaps {
		t.a.Overlaps[s] = round6(float64(k) / n)
	}
	return t.a
}

// ocRuleRow is one rule over every scenario: the unstable fraction with
// its exact 95% interval, and the rejection probability of an
// evaluation of 40 and of 10 seeds.
type ocRuleRow struct {
	Rule     string                        `json:"rule"`
	Unstable map[string]float64            `json:"unstableSeedFraction"`
	Lower    map[string]float64            `json:"unstableLower95"`
	Upper    map[string]float64            `json:"unstableUpper95"`
	Reject40 map[string]float64            `json:"rejectionAt40Seeds"`
	Reject10 map[string]float64            `json:"rejectionAt10Seeds"`
	ByCheck  map[string]map[string]float64 `json:"failingSeedFractionByCheck,omitempty"` // scenario → check → fraction
}

type ocBacklogStudy struct {
	Protocol  string              `json:"protocol"`
	Model     []string            `json:"model"`
	Seeds     int                 `json:"seedsPerScenario"`
	Streams   string              `json:"streams"`
	Scenarios []ocBacklogScenario `json:"scenarios"`
	Targets   struct {
		MaxRejection40 float64 `json:"t1MaxRejectionAt40Seeds"`
		PerSeedLimit   float64 `json:"t1PerSeedUnstableLimit"`
		MinDetection   float64 `json:"t2MinPerSeedDetection"`
	} `json:"targets"`
	// Reproduction: v0.2-eval-3 on the Slice 5C.2 streams (oc-v0.2-eval-3.json).
	Reproduction []ocAttribution `json:"reproduction5C2"`
	// Attribution: v0.2-eval-3 on this run's streams, every scenario.
	Attribution []ocAttribution `json:"attributionEval3"`
	Rules       []ocRuleRow     `json:"rules"` // v0.2-eval-3, without the peak, and the selected candidate
	// Candidates: every candidate of the family as one row (ocCandidate.row).
	CandidateFormat string       `json:"candidateFormat"`
	Candidates      []string     `json:"candidates"`
	Eligible        int          `json:"eligibleCandidates"`
	Selected        string       `json:"selected"`
	SelectedRule    *ocCandidate `json:"selectedCandidate"`
}

const ocCandidateFormat = "levelStatistic,levelGrowthFraction,levelGrowthStudies,lateWorkStatistic,growthFraction,staleCapStudies | unstable seeds by scenario (scenario order) | meets T1 | meets T2 | ladder detection"

func (c ocCandidate) row() string {
	n := make([]string, len(c.Unstable))
	for i, x := range c.Unstable {
		n[i] = fmt.Sprint(x)
	}
	return fmt.Sprintf("%s,%.2f,%d,%s,%.2f,%d | %s | %t | %t | %.6f", c.LevelStat, c.LevelGrowth, c.LevelFloor, c.LateStat, c.LateGrowth, c.Cap,
		strings.Join(n, " "), c.T1, c.T2, c.Gray)
}

const (
	ocBacklogSeeds = 1000
	ocT1Reject     = 0.05
	ocT2Detect     = 0.90
)

// ocPerSeedLimit is the largest per-seed unstable fraction (to 1e-6) at
// which P(more than 4 of 40 unstable) is at most ocT1Reject.
func ocPerSeedLimit(p *Policy) float64 {
	lim := failLimit(bp(p.Backlog.MaxUnstableSeedFrac), 40)
	lo, hi := 0.0, 1.0
	for range 60 {
		if mid := (lo + hi) / 2; binomAbove(40, mid, lim) <= ocT1Reject {
			lo = mid
		} else {
			hi = mid
		}
	}
	return round6(lo)
}

func ocBacklogCompute(p *Policy, seeds int) ocBacklogStudy {
	var st ocBacklogStudy
	st.Protocol, st.Seeds, st.Scenarios = "docs/calibration/backlog-5c3/protocol.md", seeds, ocBacklogScenarios()
	st.Streams = "PCG(20261019, scenario index)"
	st.Targets.MaxRejection40, st.Targets.PerSeedLimit, st.Targets.MinDetection = ocT1Reject, ocPerSeedLimit(p), ocT2Detect
	st.Model = []string{
		"A reading queue simulated hour by hour from the takeover (Tue 2026-10-13 06:00) for 181 days (internal/eval/oc_queue_test.go): 52 acquisitions a day (Poisson; the weekday profile; 08:00-20:00), 2% urgent, 12% soon, 86% routine; reading Mon-Fri 07:30-20:00 and Sat 10:00-14:00 at a utilization of 0.85 of the season-1 demand, urgent first, then soon, then routine, first in first out. Daily samples at 06:00; the record's horizon is day 150, so the reference block is days 55-82 and the later blocks 83-110, 111-138 and 139-166 (covering).",
		"Seasonal scenarios multiply demand by the scenario's monthly season factor; bursty ones by a gamma day factor of the stated CV; capacity scenarios multiply reading capacity on the stated days; inherited work is acquired before the takeover and not read until the stated day; held-back work joins the queue the stated days after its acquisition.",
		"Each seed is judged by judgeBacklog under each rule. A rule's rejection at S seeds is P(Binomial(S, unstable fraction) > floor(10% x S)).",
		"T1: every stable scenario's rejection at 40 seeds at most 0.05. T2: every deteriorating scenario's unstable fraction at least 0.90. Selection among the candidates meeting both: the highest mean unstable fraction on the ladder (L5, L10), then the protocol's tie-break order (the family's order). Ladder and sensitivity scenarios are reported, not targets.",
	}
	scen := st.Scenarios
	fam := ocFamily()
	pols := make([]*Policy, len(fam))
	for i, c := range fam {
		pols[i] = ocWithBacklog(p, c)
	}
	e3, e3np := ocEval3(p, true), ocEval3(p, false)
	counts := make([][]int, len(scen)) // scenario → candidate → unstable seeds
	refs := make([][2]int, len(scen))  // v0.2-eval-3, without the peak
	attrs := make([]ocAttribution, len(scen))
	var wg sync.WaitGroup
	for k, s := range scen {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := ocRNG{rand.New(rand.NewPCG(20261019, uint64(k)))}
			counts[k] = make([]int, len(fam))
			at := newAttributor(s.ID, st.Streams)
			for range seeds {
				bm, ok := backlogBlocks(p, s.Queue.record(r))
				if !ok {
					panic("synthetic record without whole blocks")
				}
				judgeBacklog(e3, &bm)
				at.add(&bm)
				if bm.Status == "unstable" {
					refs[k][0]++
				}
				judgeBacklog(e3np, &bm)
				if bm.Status == "unstable" {
					refs[k][1]++
				}
				for i, q := range pols {
					judgeBacklog(q, &bm)
					if bm.Status == "unstable" {
						counts[k][i]++
					}
				}
			}
			attrs[k] = at.done()
		}()
	}
	wg.Wait()
	st.Attribution = attrs
	// The Slice 5C.2 reproduction: its streams, 300 seeds.
	for k, q := range []ocQueue{{}, {Seasonal: true}} {
		r := ocRNG{rand.New(rand.NewPCG(20261018, uint64(1000+k)))}
		at := newAttributor([]string{"stationary demand, stable capacity (utilization 0.85)", "scenario seasonality, stable capacity"}[k], "PCG(20261018, 1000+k) (Slice 5C.2)")
		for range 300 {
			bm, _ := backlogBlocks(p, q.record(r))
			judgeBacklog(e3, &bm)
			at.add(&bm)
		}
		st.Reproduction = append(st.Reproduction, at.done())
	}
	frac := func(n int) float64 { return float64(n) / float64(seeds) }
	for i := range fam {
		c := &fam[i]
		c.T1, c.T2 = true, true
		for k, s := range scen {
			c.Unstable = append(c.Unstable, counts[k][i])
			q := frac(counts[k][i])
			switch s.Role {
			case "stable":
				c.T1 = c.T1 && binomAbove(40, q, failLimit(bp(p.Backlog.MaxUnstableSeedFrac), 40)) <= ocT1Reject
			case "deteriorating":
				c.T2 = c.T2 && q >= ocT2Detect
			case "ladder":
				c.Gray += q / 2
			}
		}
		c.Gray = round6(c.Gray)
	}
	best := -1
	for i, c := range fam {
		if c.T1 && c.T2 {
			st.Eligible++
			if best < 0 || c.Gray > fam[best].Gray {
				best = i
			}
		}
	}
	st.CandidateFormat = ocCandidateFormat
	for _, c := range fam {
		st.Candidates = append(st.Candidates, c.row())
	}
	row := func(name string, n func(k int) int) ocRuleRow {
		r := ocRuleRow{Rule: name, Unstable: map[string]float64{}, Lower: map[string]float64{}, Upper: map[string]float64{},
			Reject40: map[string]float64{}, Reject10: map[string]float64{}}
		for k, s := range scen {
			x := n(k)
			q := frac(x)
			r.Unstable[s.ID] = round6(q)
			r.Lower[s.ID] = round6(1 - binomUpper(seeds-x, seeds, 0.975))
			r.Upper[s.ID] = round6(binomUpper(x, seeds, 0.975))
			r.Reject40[s.ID] = round6(binomAbove(40, q, failLimit(bp(p.Backlog.MaxUnstableSeedFrac), 40)))
			r.Reject10[s.ID] = round6(binomAbove(10, q, failLimit(bp(p.Backlog.MaxUnstableSeedFrac), 10)))
		}
		return r
	}
	st.Rules = append(st.Rules, row("v0.2-eval-3", func(k int) int { return refs[k][0] }), row("v0.2-eval-3 without the peak check", func(k int) int { return refs[k][1] }))
	if best >= 0 {
		st.Selected, st.SelectedRule = fam[best].Name, &fam[best]
		st.Rules = append(st.Rules, row("selected: "+fam[best].Name, func(k int) int { return counts[k][best] }))
	}
	return st
}

// ocCandidateOf is the candidate a policy's backlog settings are, if it
// has the family's structure.
func ocCandidateOf(p *Policy) (ocCandidate, bool) {
	c := ocCandidate{LevelStat: p.Backlog.LevelStatistic, LevelGrowth: p.Backlog.LevelGrowth, LevelFloor: p.Backlog.LevelGrowthStudies,
		LateStat: p.Backlog.LateStatistic, LateGrowth: p.Backlog.Growth, Cap: p.Backlog.StaleCapStudies}
	q := ocWithBacklog(p, c)
	ok := p.Backlog.GrowthStudies == 2 && slices.Equal(sorted(p.Backlog.WorkloadAdjustedChecks), q.Backlog.WorkloadAdjustedChecks) &&
		slices.Equal(sorted(p.Backlog.AbsoluteChecks), q.Backlog.AbsoluteChecks) && slices.Equal(sorted(p.Backlog.InformationalChecks), q.Backlog.InformationalChecks)
	return c, ok
}

func sorted(xs []string) []string {
	ys := slices.Clone(xs)
	slices.Sort(ys)
	return ys
}
