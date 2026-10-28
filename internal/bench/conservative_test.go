package bench

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// The conservative baseline's rules, one at a time.
func TestConservativeRules(t *testing.T) {
	// Sustained pressure: three decision points in a row over 24 hours;
	// a clear streak of seven at most 8 hours; anything between resets
	// both.
	var p Conservative
	for i, oldest := range []int64{25 * 60, 30 * 60, 9 * 60, 25 * 60, 26 * 60, 27 * 60} {
		p.observe(oldest)
		if want := map[int]int{0: 1, 1: 2, 2: 0, 3: 1, 4: 2, 5: 3}[i]; p.pressure != want || p.clear != 0 {
			t.Fatalf("point %d: pressure %d clear %d", i, p.pressure, p.clear)
		}
	}
	// A weekend gap (two points over a day) is not sustained.
	q := Conservative{}
	for _, oldest := range []int64{2 * 60, 30 * 60, 40 * 60, 3 * 60} {
		q.observe(oldest)
		if _, raise := q.wanted("2026-10-20", 1_000_000, 10_000_000); raise {
			t.Fatal("raised on a two-day gap")
		}
	}

	ok := 10_000_000 // cash
	for _, c := range []struct {
		name            string
		pressure, clear int
		wait, date      string
		profit, cash    int64
		retreat, raise  bool
	}{
		{"sustained, affordable", 3, 0, "", "2026-10-20", 500_000, int64(ok), false, true},
		{"not yet sustained", 2, 0, "", "2026-10-20", 500_000, int64(ok), false, false},
		{"below the cash floor", 3, 0, "", "2026-10-20", 500_000, 4_999_999, false, false},
		{"cash at the floor", 3, 0, "", "2026-10-20", 500_000, 5_000_000, false, true},
		{"losing money: step back, never raise", 3, 0, "", "2026-10-20", -1, int64(ok), true, false},
		{"clear for a week", 0, 7, "", "2026-10-20", 500_000, int64(ok), true, false},
		{"settling after a change", 3, 0, "2026-10-21", "2026-10-20", 500_000, int64(ok), false, false},
		{"settled", 3, 0, "2026-10-21", "2026-10-21", 500_000, int64(ok), false, true},
	} {
		p := Conservative{pressure: c.pressure, clear: c.clear, waitUntil: c.wait}
		if r, s := p.wanted(c.date, c.profit, c.cash); r != c.retreat || s != c.raise {
			t.Errorf("%s: retreat %v raise %v", c.name, r, s)
		}
	}

	at := func(inEffect, scenario int64) consTarget {
		return consTarget{Employee: "E", Editable: true, InEffect: inEffect, Scenario: scenario, Max: 10_000, Earliest: "2026-11-01"}
	}
	for _, c := range []struct {
		name           string
		retreat, raise bool
		ts             []consTarget
		best           int
		bp             int64
	}{
		{"raise the lowest", false, true, []consTarget{at(6000, 6000), at(5000, 5000)}, 1, 5500},
		{"the cap: 5 points above in all", false, true, []consTarget{at(6500, 6000), at(5000, 5000)}, -1, 0},
		{"pending: wait", false, true, []consTarget{at(6000, 6000), {Employee: "E", Editable: true, InEffect: 5000, Scenario: 5000, Max: 10_000, Pending: true}}, -1, 0},
		{"not editable", false, true, []consTarget{{Employee: "E", InEffect: 5000, Scenario: 5000, Max: 10_000}}, -1, 0},
		{"at the maximum", false, true, []consTarget{{Employee: "E", Editable: true, InEffect: 10_000, Scenario: 10_000, Max: 10_000}}, -1, 0},
		{"step back the furthest above", true, false, []consTarget{at(6000, 6000), at(5500, 5000)}, 1, 5000},
		{"never below the scenario level", true, false, []consTarget{at(6200, 6000)}, 0, 6000},
		{"nothing to step back", true, false, []consTarget{at(6000, 6000), at(5000, 5000)}, -1, 0},
	} {
		if best, bp := choose(c.retreat, c.raise, c.ts); best != c.best || bp != c.bp {
			t.Errorf("%s: %d %d", c.name, best, bp)
		}
	}

	if !affordable(250_000, 250_000) || affordable(249_999, 250_000) || affordable(-1, 0) {
		t.Error("affordable")
	}
}

// overloaded is a test-only practice (not a suite scenario): the stable
// practice with its radiologists' site allocations cut to 40% and 30%,
// so that reading falls behind for weeks, and every payer's rates
// doubled, so that it stays profitable regardless (revenue is billed on
// final reports, so the backlog alone would make it lose money, and the
// policy never raises then).
const overloaded = `{"schema": "dreamteam.scenario/v1", "extends": "bench-stable.json", "id": "test-overloaded",
 "employees": [{"id": "E07", "siteAllocationBP": 4000}, {"id": "E08", "siteAllocationBP": 3000}],
 "finance": {"payers": [{"id": "commercial", "multiplierBP": 32000}, {"id": "medicare", "multiplierBP": 20000},
  {"id": "medicaid", "multiplierBP": 15000}, {"id": "self_pay", "multiplierBP": 18000}]}}`

// Over a run: valid allocation changes only, at most one per decision
// point, each at least 14 days after the previous one took effect, never
// more than 5 points above the scenario levels in all; and no raise
// while the practice loses money (the financial-pressure scenario at the
// takeover).
func TestConservativeRun(t *testing.T) {
	if testing.Short() {
		t.Skip("plays 60 days of three practices")
	}
	scen := filepath.Join(t.TempDir(), "overloaded.json")
	if err := os.WriteFile(scen, []byte(overloaded), 0o644); err != nil {
		t.Fatal(err)
	}
	c := testConfig("noop", "conservative")
	c.Scenarios, c.Seeds, c.Days = []string{scen, "bench-financial-pressure", "sustainable"}, []int64{7}, 60
	c.Evaluation.T90 = false
	raw, _ := json.Marshal(c)
	dir := filepath.Join(t.TempDir(), "results")
	if _, err := Matrix(c, raw, dir, nil); err != nil {
		t.Fatal(err)
	}
	levels := map[string]map[string]int64{scen: {"E07": 4000, "E08": 3000},
		"bench-financial-pressure": {"E07": 6000, "E08": 5000}, "sustainable": {"E07": 6000, "E08": 5000}}
	for _, scen := range c.Scenarios {
		var ds []Decision
		raw, err := os.ReadFile(filepath.Join(dir, "runs", runID(scen, 7, "conservative"), FileDecisions))
		if err != nil {
			t.Fatal(err)
		}
		json.Unmarshal(raw, &ds)
		var next time.Time
		above := map[string]int64{}
		total := 0
		for _, d := range ds {
			if len(d.Batches) > 1 || (len(d.Batches) == 1 && (len(d.Batches[0].Actions) != 1 || !d.Batches[0].Applied)) {
				t.Fatalf("%s point %d: %+v", scen, d.Point, d.Batches)
			}
			if len(d.Batches) == 0 {
				continue
			}
			var a struct {
				Employee  string
				Effective string
				SiteBP    int64 `json:"siteAllocationBP"`
			}
			json.Unmarshal(d.Batches[0].Actions[0], &a)
			now, _ := time.Parse("2006-01-02T15:04:05", d.Time)
			if !next.IsZero() && now.Before(next) {
				t.Fatalf("%s: change at %s before %s", scen, d.Time, next)
			}
			eff, _ := time.Parse("2006-01-02", a.Effective)
			next = eff.AddDate(0, 0, 14)
			total++
			above[a.Employee] = a.SiteBP
			sum := int64(0)
			for e, bp := range above {
				sum += max(0, bp-levels[scen][e])
			}
			if sum > 500 {
				t.Fatalf("%s: %d points above the scenario levels", scen, sum)
			}
		}
		if scen == "bench-financial-pressure" && total > 0 {
			t.Errorf("%s: %d changes while losing money", scen, total)
		}
		if scen != "bench-financial-pressure" && scen != "sustainable" && total == 0 {
			t.Errorf("%s: no change under a sustained backlog", scen)
		}
		t.Logf("%s: %d changes, last allocations %v", scen, total, above)
	}
}
