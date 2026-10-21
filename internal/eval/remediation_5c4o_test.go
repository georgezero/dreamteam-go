package eval

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// The release-gate audit 5C.4o's blockers: measurements that are present
// but partial or contradictory (O1), and data after a line's value (O2).
// Each witness is applied to a line that is otherwise consistent, so that
// the witness alone is what is refused.

// releaseGateLine is scenario id's first line of a consistent fabricated
// run, copied so that editing it leaves the run unchanged.
func releaseGateLine(t *testing.T, ls []groupResult, id string) groupResult {
	t.Helper()
	for _, g := range ls {
		if g.Scenario == id {
			raw, err := json.Marshal(g)
			if err != nil {
				t.Fatal(err)
			}
			var c groupResult
			if err := json.Unmarshal(raw, &c); err != nil {
				t.Fatal(err)
			}
			return c
		}
	}
	t.Fatalf("no line of %s", id)
	return groupResult{}
}

// setUrgent gives every pair of g an urgent difference of k in 30, its
// measurements following, its status kept.
func setUrgent(g *groupResult, k int64) {
	for i := range g.UrgentSeeds {
		g.UrgentSeeds[i].Candidate = g.UrgentSeeds[i].Baseline + k
	}
	gateBUrgent(g)
}

// statedDifferences is g's urgent seed differences as stated.
func statedDifferences(g *groupResult) []float64 {
	var xs []float64
	for _, d := range g.UrgentSeeds {
		xs = append(xs, d.Difference)
	}
	return xs
}

// O1: every partial or contradictory measurement of the audit, and the
// reconciliation's other relations, is a measurement problem.
func TestReleaseGateMeasurements(t *testing.T) {
	m := loadManifest(t)
	ls := fabricatedRun(m)
	for _, id := range []string{"S-A85", "NM0", "NM1", "W-LOST30", "W-ID-REMOVED", "W-ID-CHANGED", "H15"} {
		g := releaseGateLine(t, ls, id)
		if bad := lineProblems(manScenarioOf(m, id), &g); len(bad) > 0 {
			t.Fatalf("%s: the consistent line is refused: %v", id, bad)
		}
	}
	for name, c := range map[string]struct {
		id   string
		edit func(g *groupResult)
	}{
		// The audit's witnesses (O1).
		"partial T5 differences":         {"NM1", func(g *groupResult) { g.UrgentSeeds = g.UrgentSeeds[:1] }},
		"difference contradicts counts":  {"NM1", func(g *groupResult) { g.UrgentSeeds[0].Difference = -0.75 }},
		"PASS despite 90 point increase": {"S-A85", func(g *groupResult) { g.Increase["urgent"] = 0.9 }},
		"impossible injected count":      {"NM1", func(g *groupResult) { g.Pairs[0].Injected = 26 }},
		"hidden pair construction error": {"S-A85", func(g *groupResult) { g.Pairs[0].Error = "construction failed" }},
		// The reconciliation's other relations.
		"injected above unimaged":        {"NM1", func(g *groupResult) { g.Pairs[1].Injected = 4 }},
		"PASS on partial differences":    {"S-A85", func(g *groupResult) { g.UrgentSeeds = g.UrgentSeeds[1:] }},
		"PASS with soon unmatched":       {"S-A85", func(g *groupResult) { g.Matched["soon"]-- }},
		"PASS above the margin":          {"S-A85", func(g *groupResult) { setUrgent(g, 3) }},
		"FAIL within the margin":         {"W-LOST30", func(g *groupResult) { setUrgent(g, 0) }},
		"safeguard flag cleared":         {"H15", func(g *groupResult) { g.UrgentSeeds[0].Deteriorated = false }},
		"deteriorated seed not listed":   {"H15", func(g *groupResult) { g.Deteriorated["urgent"] = g.Deteriorated["urgent"][1:] }},
		"deteriorated seed not observed": {"S-A85", func(g *groupResult) { g.Deteriorated["urgent"] = []int64{int64(g.Pairs[0].Seed)} }},
		"increase without differences":   {"W-ID-REMOVED", func(g *groupResult) { g.UrgentSeeds = nil; g.Increase["urgent"] = 0.01 }},
		// Each check alone: what no other relation catches.
		"matched above pairs, undecided": {"W-ID-REMOVED", func(g *groupResult) { g.Matched["soon"] = 999 }},
		"partial differences, undecided": {"NM1", func(g *groupResult) {
			g.GateB["urgent"], g.Rejected, g.Rules = Insufficient, true, []string{"gateB:urgent"}
			g.UrgentSeeds = g.UrgentSeeds[:1]
			gateBUrgent(g)
		}},
		"difference and increase off the counts": {"S-A85", func(g *groupResult) {
			g.UrgentSeeds[0].Difference = 0.01 // its counts say 0
			g.Increase["urgent"], _ = meanSD(statedDifferences(g))
		}},
	} {
		g := releaseGateLine(t, ls, c.id)
		c.edit(&g)
		if bad := lineProblems(manScenarioOf(m, c.id), &g); len(bad) == 0 {
			t.Errorf("%s: accepted", name)
		}
	}
	// End to end: the audit's partial differences on every NM1 replicate
	// leave T5a and T8 short of PASS (incomplete coverage).
	tol, err := manifestTolerances(m)
	if err != nil {
		t.Fatal(err)
	}
	for i := range ls {
		if ls[i].Scenario == "NM1" {
			ls[i].UrgentSeeds = ls[i].UrgentSeeds[:1]
		}
	}
	full := func(s *manScenario) int { return s.Seeds }
	r := reservedCriteria(m, ls, tol, full)
	if r["T5a"].Status == Pass || r["T8"].Status == Pass || len(coverageProblems(m, ls, full)) == 0 {
		t.Errorf("partial differences: T5a %s, T8 %s", r["T5a"].Status, r["T8"].Status)
	}
}

// O2: anything after a line's single JSON value fails the reader before
// anything is judged; whitespace alone does not.
func TestReleaseGateReader(t *testing.T) {
	m := loadManifest(t)
	for _, k := range []string{"DT_MANIFEST_SEEDS", "DT_MANIFEST_SCENARIOS", "DT_MANIFEST_FAMILY"} {
		t.Setenv(k, "")
	}
	dir := t.TempDir()
	full := filepath.Join(dir, "full.jsonl")
	writeLines(t, full, fabricatedRun(m))
	raw, err := os.ReadFile(full)
	if err != nil {
		t.Fatal(err)
	}
	body := raw[:len(raw)-1] // the last line, without its newline
	for name, suffix := range map[string]string{
		"truncated second value": ` {"scenario":`,
		"complete second value":  ` {"scenario":"S-A85"}`,
		"closing brace":          `}`,
		"text":                   ` x`,
	} {
		p := filepath.Join(dir, "trailing.jsonl")
		if err := os.WriteFile(p, append(append([]byte{}, body...), suffix+"\n"...), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, _, err := analyzeFile(m, p); err == nil {
			t.Errorf("%s after a line: read", name)
		}
	}
	p := filepath.Join(dir, "spaces.jsonl")
	if err := os.WriteFile(p, append(append([]byte{}, body...), " \t\n"...), 0o644); err != nil {
		t.Fatal(err)
	}
	r, bad, err := analyzeFile(m, p)
	if err != nil || len(bad) > 0 || r["certification"].Status != "not certified" {
		t.Errorf("trailing whitespace: %v, %d problems, %s", err, len(bad), r["certification"].Status)
	}
}
