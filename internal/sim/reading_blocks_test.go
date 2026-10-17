package sim_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/rng"
	"github.com/georgezero/dreamteam-go/internal/sim"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Reading-block boundaries and overrun debt (0.2.0-dev.6, V0.2_DESIGN
// §5.11). micro-blocks: Dr. Rae (D1) reads Tue 08:00–20:00 (720 min); 18
// studies enter the worklist from 06:31, one every 20 minutes, so the queue
// is never empty when a block opens. The first is XR_A, the rest XR_B;
// each test fixes the allocation and both read times (p0 = p100). Blocks
// follow the 100-minute cycle from 08:00.

// blocksWorld is micro-blocks.json at allocation bp with reads of exactly
// readA (XR_A) and readB (XR_B) minutes, under dev.5's rules if dev5.
func blocksWorld(t *testing.T, bp, readA, readB int64, dev5 bool) (*sim.World, simtime.Time) {
	t.Helper()
	cfg := microCfg(t, "micro-blocks.json")
	for k := range cfg.Employees {
		if cfg.Employees[k].ID == "D1" {
			cfg.Employees[k].SiteAllocationBP = bp
		}
	}
	for k := range cfg.Catalog {
		m := map[string]int64{"XR_A": readA, "XR_B": readB}[cfg.Catalog[k].Code]
		if m == 0 {
			continue
		}
		v := int64(simtime.Duration(m) * simtime.Minute)
		q, err := rng.NewQuantile([]rng.QuantilePoint{{P: 0, V: v}, {P: rng.OneMillion, V: v}})
		if err != nil {
			t.Fatal(err)
		}
		cfg.Catalog[k].ReadTime = q
	}
	w := sim.New(cfg, rng.SeedFromInt(1))
	w.Engine().CheckInvariants = true
	w.SetDev5Rules(dev5)
	return w, w.Calendar().StartOfDate(cfg.Start)
}

// claimMins runs micro-blocks to the end of the day and returns each
// claim's minute after 08:00, in claim order, checking that every claimed
// study was finalized exactly once and that reads never overlap.
func claimMins(t *testing.T, w *sim.World, day simtime.Time) []int64 {
	t.Helper()
	if err := runTo(w, at(day, "23:00")); err != nil {
		t.Fatal(err)
	}
	var out []int64
	var last simtime.Time
	studies := w.Studies()
	slices.SortFunc(studies, func(a, b sim.StudyInfo) int { return int(a.Claimed - b.Claimed) })
	for _, s := range studies {
		switch s.State {
		case "final":
		case "unread":
			continue
		default:
			t.Fatalf("%s left %s at 23:00", s.ID, s.State)
		}
		if s.Claimed < last {
			t.Fatalf("%s claimed at %s, before the previous read ended", s.ID, w.Calendar().Clock(s.Claimed))
		}
		last = s.Final
		out = append(out, int64(s.Claimed.Sub(at(day, "08:00"))/simtime.Minute))
	}
	return out
}

func every(from, step int64, n int) []int64 {
	out := make([]int64, n)
	for k := range out {
		out[k] = from + int64(k)*step
	}
	return out
}

// Fix 1. At 100% the cycle has no other-duty time: block k ends at 08:00 +
// 100(k+1), exactly where block k+1 starts. Reading is continuous across
// every such boundary, for reads crossing it (30-minute reads cross 09:40,
// 11:20, 14:40 and 16:20) and for reads ending exactly on it (14:00, and
// every boundary with 20-minute reads). Up to dev.5 the earlier block's end
// (class 5) ran after the next block's start (class 1) and ended reading
// for the rest of the shift: 4 and 5 reads instead of 18.
func TestTouchingBlocksReadContinuously(t *testing.T) {
	for _, c := range []struct {
		read      int64
		want, old []int64
	}{
		{30, every(0, 30, 18), every(0, 30, 4)},
		{20, every(0, 20, 18), every(0, 20, 5)},
	} {
		for _, dev5 := range []bool{false, true} {
			w, day := blocksWorld(t, 10_000, c.read, c.read, dev5)
			want := c.want
			if dev5 {
				want = c.old
			}
			if got := claimMins(t, w, day); !slices.Equal(got, want) {
				t.Errorf("%d-minute reads (dev5 %v): claims at %v, want %v", c.read, dev5, got, want)
			}
		}
	}
	// One block a day, whole shift: nothing else changes at 100%.
	w, day := blocksWorld(t, 10_000, 30, 30, false)
	blocks := w.PlanOn(w.Calendar().DayIndex(day)).Reading[2]
	if len(blocks) != 8 || blocks[0].From != 8*simtime.Hour || blocks[7].To != 20*simtime.Hour {
		t.Fatalf("blocks %v", blocks)
	}
}

// Fix 2. At 95% each cycle has 5 minutes of other duties (09:35–09:40,
// 11:15–11:20, ...). A 47-minute read claimed at cycle minute 94 runs
// through that gap into the next block: the 5 minutes are borrowed, and
// the next claim waits 5 minutes after the read ends (09:41 → 09:46).
// Every such read owes exactly its gap, once. Up to dev.5 these reads
// owed nothing, because the next block had opened by the time they ended.
func TestOverrunThroughOtherDutiesIsOwed(t *testing.T) {
	for _, c := range []struct {
		name      string
		read      int64
		want, old []int64
	}{
		{"47-minute reads", 47,
			[]int64{0, 47, 94, 146, 193, 245, 292, 344, 391, 443, 490, 542, 589, 641},
			[]int64{0, 47, 94, 141, 188, 235, 282, 329, 376, 423, 470, 517, 564, 611, 658}},
		// Ending exactly at the next block's start (09:40) is the same: the
		// gap was borrowed, so the next claim is at 09:45, not 09:40.
		{"50-minute reads ending on the block start", 50,
			[]int64{0, 50, 105, 155, 210, 260, 315, 365, 420, 470, 525, 575, 630},
			[]int64{0, 50, 100, 150, 200, 250, 300, 350, 400, 450, 500, 550, 600, 650}},
		// Ending exactly at a block's end owes nothing (the end is class 5,
		// after the read's completion): 19-minute reads, five per block.
		{"19-minute reads ending on the block end", 19,
			[]int64{0, 19, 38, 57, 76, 100, 119, 138, 157, 176, 200, 219, 238, 257, 276, 300, 319, 338},
			[]int64{0, 19, 38, 57, 76, 100, 119, 138, 157, 176, 200, 219, 238, 257, 276, 300, 319, 338}},
	} {
		for _, dev5 := range []bool{false, true} {
			w, day := blocksWorld(t, 9_500, c.read, c.read, dev5)
			want := c.want
			if dev5 {
				want = c.old
			}
			if got := claimMins(t, w, day); !slices.Equal(got, want) {
				t.Errorf("%s (dev5 %v): claims at %v, want %v", c.name, dev5, got, want)
			}
		}
	}
}

// Fix 2, several gaps and later repayment. At 30% the blocks are 08:00–
// 08:30, 09:40–10:10, 11:20–11:50, ... A 205-minute read from 08:00 runs
// through two other-duty intervals (08:30–09:40, 10:10–11:20: 140 minutes
// borrowed) and ends at 11:25 inside the third block. The debt is repaid
// out of block time, in order: 25 minutes of that block, 30 of each of the
// next three, and the last 25 of the block from 18:00, so the next claim
// is 18:25. Then 5-minute reads fill the time left (18:25, 19:40–19:55).
// Up to dev.5 the 140 minutes were never owed: claims from 11:25.
func TestOverrunThroughSeveralGapsIsRepaidLater(t *testing.T) {
	want := []int64{0, 625, 700, 705, 710, 715}
	old := []int64{0, 205, 210, 215, 220, 225, 300, 305, 310, 315, 320, 325, 400, 405, 410, 415, 420, 425}
	for _, dev5 := range []bool{false, true} {
		w, day := blocksWorld(t, 3_000, 205, 5, dev5)
		exp := want
		if dev5 {
			exp = old
		}
		if got := claimMins(t, w, day); !slices.Equal(got, exp) {
			t.Errorf("dev5 %v: claims at %v, want %v", dev5, got, exp)
		}
	}
}

// Conservation of productive reading time, checked against an independent
// account of the rule on every radiologist-day: a read's time outside the
// day's site blocks is borrowed; borrowed time is repaid, in order, out of
// later block time in which the radiologist is not reading; no read is
// claimed while any is owed, or outside a block; what is still owed at the
// end of the shift lapses with the day (as since V0.1). So no claim is made
// while time is owed or outside a block, and reading inside blocks plus
// repayment never exceeds block time: reading time ≤ block time + lapsed.
//
// Cases: the micro-blocks days above, and the established practice for six
// weeks with Dr. Avery at 100% and Dr. Blake at 95% from Oct 27 (catalog
// read times). Under dev.5's rules the same account finds claims made
// while time was still owed (free capacity) wherever blocks have gaps.
func TestReadingTimeConservation(t *testing.T) {
	type tc struct {
		name       string
		w          *sim.World
		end        simtime.Time
		wantClaims bool // the account must find claims while owing (dev.5)
		gaps       bool // blocks with other-duty time between them
	}
	var cases []tc
	for _, dev5 := range []bool{false, true} {
		for _, m := range []struct{ bp, a, b int64 }{{10_000, 30, 30}, {10_000, 20, 20}, {9_500, 47, 47}, {9_500, 50, 50}, {3_000, 205, 5}} {
			w, day := blocksWorld(t, m.bp, m.a, m.b, dev5)
			cases = append(cases, tc{fmt.Sprintf("micro %d %d/%d dev5=%v", m.bp, m.a, m.b, dev5), w, at(day, "23:00"), dev5 && m.bp < 10_000, m.bp < 10_000})
		}
		w := atTakeover(t, `{}`, 41)
		w.SetDev5Rules(dev5)
		mustApply(t, w, setAlloc("E07", "2026-10-27", 10_000), setAlloc("E08", "2026-10-27", 9_500))
		cases = append(cases, tc{fmt.Sprintf("established 100%%/95%% dev5=%v", dev5), w, w.Calendar().StartOfDay(dayOf(w, "2026-11-24")), dev5, true})
	}
	for _, c := range cases {
		if err := runTo(c.w, c.end); err != nil {
			t.Fatal(err)
		}
		acc := readingAccount(t, c.w, c.end)
		t.Logf("%s: %+v", c.name, acc)
		if (acc.owingClaims > 0) != c.wantClaims || acc.outsideClaims > 0 {
			t.Errorf("%s: %d claims while owing, %d outside a block (%+v)", c.name, acc.owingClaims, acc.outsideClaims, acc)
		}
		if !c.wantClaims && (acc.inBlock+acc.repaid > acc.block || acc.reading > acc.block+acc.lapsed) {
			t.Errorf("%s: reading exceeds site time: %+v", c.name, acc)
		}
		if c.gaps && acc.borrowed == 0 {
			t.Errorf("%s: nothing borrowed: the case does not exercise the rule", c.name)
		}
	}
}

// account is the reading-time account of a run, in seconds.
type account struct {
	reading, inBlock, block, borrowed, repaid, lapsed int64
	// Claims made while time was owed (and the debt left then), or
	// outside any block: none under this build's rules.
	owingClaims, owingClaimTime, outsideClaims int64
}

// readingAccount accounts every radiologist-day of w up to end from the
// studies' claim and final times and the day's plan.
func readingAccount(t *testing.T, w *sim.World, end simtime.Time) account {
	t.Helper()
	cal := w.Calendar()
	staff := w.Staff()
	type iv struct{ from, to simtime.Time }
	reads := map[[2]int64][]iv{} // (employee, day) → reads in claim order
	for _, s := range w.Studies() {
		if s.State != "final" || s.Final > end {
			continue
		}
		i := slices.IndexFunc(staff, func(x sim.StaffInfo) bool { return x.Name == s.By })
		key := [2]int64{int64(i), cal.DayIndex(s.Claimed)}
		reads[key] = append(reads[key], iv{s.Claimed, s.Final})
	}
	var a account
	secs := func(d simtime.Duration) int64 { return int64(d / simtime.Second) }
	for key, rs := range reads {
		slices.SortFunc(rs, func(x, y iv) int { return int(x.from - y.from) })
		i, day := int(key[0]), key[1]
		plan := w.PlanOn(day)
		var blocks []iv
		for _, b := range plan.Reading[i] {
			blocks = append(blocks, iv{cal.At(day, b.From), cal.At(day, b.To)})
			a.block += secs(b.To - b.From)
		}
		shiftEnd := cal.At(day, plan.Shifts[plan.ByEmp[i]].To)
		inBlocks := func(from, to simtime.Time) simtime.Duration {
			var n simtime.Duration
			for _, b := range blocks {
				n += max(0, min(to, b.to).Sub(max(from, b.from)))
			}
			return n
		}
		var owed simtime.Duration
		freeFrom := simtime.Time(0) // reads end here; the next can start
		for k, r := range rs {
			if k > 0 {
				// Repayment: block time between the reads, not reading.
				idle := inBlocks(freeFrom, r.from)
				if idle < owed {
					a.owingClaims++
					a.owingClaimTime += secs(owed - idle)
				}
				paid := min(owed, idle)
				a.repaid += secs(paid)
				owed -= paid
			}
			if !slices.ContainsFunc(blocks, func(b iv) bool { return b.from <= r.from && r.from < b.to }) {
				a.outsideClaims++
			}
			a.reading += secs(r.to.Sub(r.from))
			a.inBlock += secs(inBlocks(r.from, r.to))
			out := r.to.Sub(r.from) - inBlocks(r.from, r.to)
			a.borrowed += secs(out)
			owed += out
			freeFrom = r.to
		}
		paid := min(owed, inBlocks(freeFrom, shiftEnd))
		a.repaid += secs(paid)
		a.lapsed += secs(owed - paid)
	}
	return a
}
