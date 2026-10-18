package eval

import (
	"strconv"
	"strings"
	"testing"
)

// Slice 5C.3 (v0.2-eval-4): the backlog gate distinguishes a stable queue
// from clinically meaningful deterioration. Temporary spikes, bursts of
// late work that clear, a seasonal rise and a stable queue at a higher
// utilization pass; growth without bound, persistent and aging old work,
// late-tail deterioration and old work in an inflated reference fail; a
// record missing its workload or its overdue-age data is never evidence.
// Each case beside v0.2-eval-3's rule (ocEval3) where they differ.

// heldStudies are perDay routine studies acquired at 03:00 on each day of
// [from, to), each finished wait days later.
func heldStudies(from, to, perDay int, wait float64) []Study {
	var out []Study
	for d := from; d < to; d++ {
		for k := range perDay {
			acq := t0 + int64(d)*daySec - 3*hour + int64(k)*60
			fin := acq + int64(wait*float64(daySec))
			out = append(out, Study{ID: "H", Priority: "routine", AcquiredSec: acq, EnteredSec: acq, ClaimedSec: fin - hour, ReadStartSec: fin - hour, FinalSec: fin})
		}
	}
	return out
}

func failedBlocks(bm BacklogMeasures, check string) []int {
	var out []int
	for _, c := range bm.Checks {
		if c.Check == check && !c.Pass {
			out = append(out, c.FirstDay)
		}
	}
	return out
}

func TestBacklogStability(t *testing.T) {
	p, _ := testPolicy(t)
	v3 := ocEval3(p, true)
	t90 := t0 + 90*daySec
	run := func(pol *Policy, level, acq func(int) int, studies []Study) BacklogMeasures {
		return measureBacklog(pol, workloadRecord(level, acq, studies), t90)
	}
	flat := func(int) int { return 20 }
	steady := func(int) int { return 40 }
	rising := func(d int) int {
		if d <= 82 {
			return 40
		}
		return 50
	}

	// Stable bursty workload: every ninth day after the reference, 30
	// more acquisitions, 40 more studies waiting the next morning and 12
	// read three days late (late work on 2 of every 9 mornings).
	burst := func(d int) bool { return d > 82 && d%9 == 0 }
	var late []Study
	for d := 83; d < 167; d++ {
		if burst(d) {
			late = append(late, heldStudies(d, d+1, 12, 3)...)
		}
	}
	bLevel := func(d int) int {
		if burst(d - 1) {
			return 60
		}
		return 20
	}
	bAcq := func(d int) int {
		if burst(d) {
			return 70
		}
		return 40
	}
	if bm := run(p, bLevel, bAcq, late); bm.Status != "stable" || len(failedBlocks(bm, "max")) == 0 {
		t.Fatalf("bursty: %v", bm.Reasons)
	}
	if bm := run(v3, bLevel, bAcq, late); bm.Status != "unstable" || len(failedBlocks(bm, "overdue")) == 0 || len(failedBlocks(bm, "max")) == 0 {
		t.Fatalf("bursty under v0.2-eval-3: %v", bm.Reasons)
	}

	// Seasonal increase: demand +25% after the reference. In proportion
	// (20 → 25) and as a queue near capacity rises (20 → 45, 1.8 times the
	// workload-scaled reference): both stable; v0.2-eval-3 fails the second.
	follows := func(lo, hi int) func(int) int {
		return func(d int) int {
			if d <= 82 {
				return lo
			}
			return hi
		}
	}
	for _, hi := range []int{25, 45} {
		if bm := run(p, follows(20, hi), rising, nil); bm.Status != "stable" {
			t.Fatalf("seasonal 20 → %d: %v", hi, bm.Reasons)
		}
	}
	if bm := run(v3, follows(20, 45), rising, nil); bm.Status != "unstable" || len(failedBlocks(bm, "mean")) == 0 {
		t.Fatalf("seasonal under v0.2-eval-3: %v", bm.Reasons)
	}
	// The level scales with the workload: demand doubling (40 → 80) and the
	// level 20 → 62 is within the tolerance of the scaled reference (40);
	// unscaled it would not be.
	double := func(d int) int {
		if d <= 82 {
			return 40
		}
		return 80
	}
	if bm := run(p, follows(20, 62), double, nil); bm.Status != "stable" || bm.Checks[0].WorkloadRatio != 2 {
		t.Fatalf("doubled demand: %v", bm.Reasons)
	}
	if bm := run(p, follows(20, 62), steady, nil); bm.Status != "unstable" || len(failedBlocks(bm, "level")) != 3 {
		t.Fatalf("level 20 → 62 at flat demand: %v", bm.Reasons)
	}

	// Block medians are exact: days 83–110 at 0, 1, …, 27 have the median
	// (13 + 14) / 2.
	ramp := func(d int) int {
		if d >= 83 && d <= 110 {
			return d - 83
		}
		return 20
	}
	if bm := run(p, ramp, steady, nil); bm.Later[0].MedianBacklog != 13.5 || bm.Later[1].MedianBacklog != 20 {
		t.Fatalf("medians %v %v", bm.Later[0].MedianBacklog, bm.Later[1].MedianBacklog)
	}

	// Sustained deterioration: the backlog grows 2 a day from day 83, and
	// from day 100 three routine studies a day wait 9 days (late work on
	// every morning). The level fails once it passes three times the
	// reference (blocks 111 and 139), and the late work fails from block 83.
	grow := func(d int) int { return 20 + 2*max(0, d-82) }
	bm := run(p, grow, steady, heldStudies(100, 167, 3, 9))
	if bm.Status != "unstable" || strings.Join(strs(failedBlocks(bm, "level")), ",") != "111,139" ||
		len(failedBlocks(bm, "overdue")) == 0 || len(failedBlocks(bm, "staleCap")) == 0 || bm.Persistent != 0 {
		t.Fatalf("deterioration: %v", bm.Reasons)
	}

	// Persistent old studies: three inherited studies never reported,
	// under proportional seasonal growth.
	var old []Study
	for k := range 3 {
		acq := t0 - 2*hour + int64(k)*60
		old = append(old, Study{ID: "I", Priority: "routine", AcquiredSec: acq, EnteredSec: acq})
	}
	if bm := run(p, follows(20, 25), rising, old); bm.Status != "unstable" || bm.Persistent != 3 {
		t.Fatalf("persistent: %v", bm.Reasons)
	}

	// Aging without count growth: the backlog stays at 20, but from day
	// 100 one routine study a day waits 10 days (never 14 days past due).
	// The level passes; the stale count and the cap fail.
	bm = run(p, flat, steady, heldStudies(100, 167, 1, 10))
	if bm.Status != "unstable" || len(failedBlocks(bm, "level")) != 0 || len(failedBlocks(bm, "stale")) == 0 ||
		len(failedBlocks(bm, "staleCap")) == 0 || bm.Persistent != 0 {
		t.Fatalf("aging: %v", bm.Reasons)
	}

	// Temporary spikes alone are not clinical failure: three days at 80
	// (four times the reference) with 30 more acquisitions a day and 12
	// studies a day read three days late, back at the scaled reference
	// three days after the peak. The peak and recovery are reported.
	spike := func(d int) int {
		if d >= 97 && d <= 99 {
			return 80
		}
		return 20
	}
	spikeAcq := func(d int) int {
		if d >= 96 && d <= 98 {
			return 70
		}
		return 40
	}
	bm = run(p, spike, spikeAcq, heldStudies(96, 99, 12, 3))
	if bm.Status != "stable" || len(failedBlocks(bm, "max")) != 1 || bm.Later[0].RecoveryDays != 3 {
		t.Fatalf("spike: %v %+v", bm.Reasons, bm.Later[0])
	}
	if bm := run(v3, spike, spikeAcq, heldStudies(96, 99, 12, 3)); bm.Status != "unstable" {
		t.Fatalf("spike under v0.2-eval-3: %v", bm.Reasons)
	}

	// The level is the block mean, not its median: a surge on 12 of the
	// 28 days of block 111 at five times the reference (mean 54.3 against
	// a tolerance of 50) fails, although the block's median is unmoved.
	surge := func(d int) int {
		if d >= 115 && d < 127 {
			return 100
		}
		return 20
	}
	if bm := run(p, surge, steady, nil); bm.Status != "unstable" || strings.Join(strs(failedBlocks(bm, "level")), ",") != "111" || bm.Later[1].MedianBacklog != 20 {
		t.Fatalf("surge: %v", bm.Reasons)
	}

	// Late-tail deterioration: growth of 5 a day from day 139, in the
	// covering block alone (139–166, the horizon day 150 and the residual
	// days after it).
	tail := func(d int) int { return 20 + 5*max(0, d-138) }
	bm = run(p, tail, steady, nil)
	if bm.Status != "unstable" || strings.Join(strs(failedBlocks(bm, "level")), ",") != "139" {
		t.Fatalf("late tail: %v", bm.Reasons)
	}
	for _, c := range bm.Checks {
		if !c.Pass && !c.Informational && c.FirstDay != 139 {
			t.Fatalf("late tail: block %d failed %s", c.FirstDay, c.Check)
		}
	}

	// Reference-period inflation: five studies a day wait 8 days in the
	// reference block, three a day later. Every relative check falls
	// against the inflated reference, and v0.2-eval-3 passes; the cap
	// holds in every block, the reference included.
	inflated := append(heldStudies(47, 83, 5, 8), heldStudies(83, 167, 3, 8)...)
	bm = run(p, flat, steady, inflated)
	if bm.Status != "unstable" || strings.Join(strs(failedBlocks(bm, "staleCap")), ",") != "55,83,111,139" || len(failedBlocks(bm, "stale")) != 0 {
		t.Fatalf("inflated reference: %v", bm.Reasons)
	}
	if bm := run(v3, flat, steady, inflated); bm.Status != "stable" {
		t.Fatalf("inflated reference under v0.2-eval-3: %v", bm.Reasons)
	}

	// Missing workload samples: no recorded acquisitions compares the level
	// unscaled, never waved through (20 → 70 fails); a record missing a
	// daily sample, or whose samples omit its acquisitions, is not
	// evidence.
	if bm := run(p, follows(20, 70), func(int) int { return 0 }, nil); bm.Status != "unstable" || len(failedBlocks(bm, "level")) == 0 {
		t.Fatalf("no workload: %v", bm.Reasons)
	}
	good := base("B", 901).record()
	if m := Measure(p, &good); len(m.Integrity) != 0 {
		t.Fatalf("fixture: %v", m.Integrity)
	}
	missing := map[string]func(r *Record){
		"a missing daily sample": func(r *Record) { r.Samples = append(r.Samples[:119:119], r.Samples[120:]...) },
		"samples without acquisitions": func(r *Record) {
			for i := range r.Samples {
				r.Samples[i].AcquiredCum = 0
			}
		},
		"no samples after day 150": func(r *Record) { r.Samples = r.Samples[:150] },
		// Missing overdue-age data: a priority's TAT target, a study's
		// priority or its acquisition time.
		"no routine target":        func(r *Record) { delete(r.TATTargetMin, "routine") },
		"a study without priority": func(r *Record) { r.Studies[5].Priority = "" },
		"a study without acquisition time": func(r *Record) {
			r.Studies[5].AcquiredSec = 0
		},
	}
	for name, mut := range missing {
		var recs []Record
		for i, r := range records(10, func(s int64) synth { return base("B", s) }) {
			r.TATTargetMin = map[string]int64{"urgent": 60, "soon": 240, "routine": 1440}
			if i == 3 {
				mut(&r)
			}
			recs = append(recs, r)
		}
		m := Measure(p, &recs[3])
		if len(m.Integrity) == 0 {
			t.Fatalf("%s: no integrity failure", name)
		}
		rep := Evaluate(p, "d", Input{Baseline: "A", Plans: map[string][]Record{"A": records(10, func(s int64) synth { return base("A", s) }), "B": recs}})
		if c := comparison(t, rep, "B"); c.Verdict != INVALID {
			t.Fatalf("%s: %s %v", name, c.Verdict, c.Reasons)
		}
		if name == "no samples after day 150" && m.Backlog.Status != "insufficient" {
			t.Fatalf("%s: backlog %s", name, m.Backlog.Status)
		}
	}
}

func strs(xs []int) []string {
	var out []string
	for _, x := range xs {
		out = append(out, strconv.Itoa(x))
	}
	return out
}
