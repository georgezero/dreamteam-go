package bench

import (
	"encoding/json"
	"os"
	"reflect"
	"slices"
	"testing"

	"github.com/georgezero/dreamteam-go/docs/evaluation"
	"github.com/georgezero/dreamteam-go/internal/scenario"
)

// The V0.6a development suite (docs/benchmark/suite/dreamteam-dev-1.json,
// docs/BENCHMARK_SUITE.md): every entry is a bundled scenario that compiles,
// with the recorded fingerprint, built from its documented source and
// nothing else; T90 is claimed exactly where the evaluation policy in force
// is defined; every metric a dimension names is one the harness reports.

type suiteFile struct {
	Schema      string
	ID          string
	Version     int
	Split       string
	HorizonDays int
	Scenarios   []struct {
		ID, Category, Scenario, Fingerprint, Source, Challenge, T90 string
		Version                                                     int
	}
	Dimensions map[string][]string
	Missing    []string
}

const suitePath = "../../docs/benchmark/suite/dreamteam-dev-1.json"

func loadSuite(t *testing.T) suiteFile {
	t.Helper()
	raw, err := os.ReadFile(suitePath)
	if err != nil {
		t.Fatal(err)
	}
	var s suiteFile
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatal(err)
	}
	if s.Schema != "dreamteam.benchmark-suite/v1" || s.ID != "dreamteam-dev" || s.Version != 1 || s.Split != "development" ||
		s.HorizonDays != 90 || len(s.Scenarios) != 5 {
		t.Fatalf("suite header: %+v", s)
	}
	return s
}

// resolved returns a scenario's canonical form as a generic value, with
// its identity (id and name) replaced by the given ones, for comparison
// with a source that carries another identity.
func resolved(t *testing.T, f *scenario.File, id, name string) any {
	t.Helper()
	g := *f
	g.ID, g.Name = id, name
	raw, err := scenario.Canonical(&g)
	if err != nil {
		t.Fatal(err)
	}
	var v any
	json.Unmarshal(raw, &v)
	return v
}

func TestSuiteScenarios(t *testing.T) {
	s := loadSuite(t)
	var pol struct {
		ID       string
		Scenario struct{ Fingerprint string }
	}
	if err := json.Unmarshal(evaluation.PolicyV4, &pol); err != nil {
		t.Fatal(err)
	}
	bundled := scenario.List()
	var ids, fps, names []string
	for _, e := range s.Scenarios {
		if e.Version != 1 || e.Category == "" || e.Challenge == "" || !slices.Contains(bundled, e.Scenario) {
			t.Fatalf("%s: %+v", e.ID, e)
		}
		f, err := scenario.ResolveFile(e.Scenario)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := scenario.Compile(f); err != nil {
			t.Fatalf("%s: %v", e.ID, err)
		}
		fp, _ := scenario.Fingerprint(f)
		if fp != e.Fingerprint {
			t.Errorf("%s: fingerprint %s, recorded %s", e.ID, fp, e.Fingerprint)
		}
		want := "not_applicable"
		if fp == pol.Scenario.Fingerprint {
			want = pol.ID
		}
		if e.T90 != want {
			t.Errorf("%s: t90 %q, the policy in force says %q", e.ID, e.T90, want)
		}
		ids, fps, names = append(ids, e.ID), append(fps, fp), append(names, f.ID)

		// Built from the documented source and nothing else.
		var src any
		switch e.ID {
		case "demand-growth":
			// The sustainable practice; only the November, December and
			// January demand factors differ: October's level ×1.05, ×1.10,
			// ×1.15.
			base, err := scenario.ResolveFile("sustainable")
			if err != nil {
				t.Fatal(err)
			}
			g := *base
			season := base.Demand.SeasonFactorBP
			g.Demand.SeasonFactorBP = append([]int64(nil), season...)
			oct := season[9]
			g.Demand.SeasonFactorBP[10], g.Demand.SeasonFactorBP[11], g.Demand.SeasonFactorBP[0] = oct*10500/10000, oct*11000/10000, oct*11500/10000
			src = resolved(t, &g, f.ID, f.Name)
		default:
			// A calibration manifest: its scenario with its static overlay.
			raw, err := os.ReadFile("../../" + e.Source)
			if err != nil {
				t.Fatal(err)
			}
			var m struct {
				Scenario string
				Overlay  json.RawMessage
			}
			if err := json.Unmarshal(raw, &m); err != nil {
				t.Fatal(err)
			}
			base, err := scenario.ResolveFile(m.Scenario)
			if err != nil {
				t.Fatal(err)
			}
			canon, _ := scenario.Canonical(base)
			var b, o any
			json.Unmarshal(canon, &b)
			json.Unmarshal(m.Overlay, &o)
			merged, _ := json.Marshal(scenario.Merge(b, o))
			g, err := scenario.Decode(merged)
			if err != nil {
				t.Fatal(err)
			}
			src = resolved(t, g, f.ID, f.Name)
		}
		got, _ := json.Marshal(resolved(t, f, f.ID, f.Name))
		exp, _ := json.Marshal(src)
		if string(got) != string(exp) {
			t.Errorf("%s: %s is not its source %s", e.ID, e.Scenario, e.Source)
		}
	}
	for i := range ids {
		if slices.Contains(ids[:i], ids[i]) || slices.Contains(fps[:i], fps[i]) {
			t.Errorf("duplicate scenario %s or fingerprint %s", ids[i], fps[i])
		}
	}
	// Only the reporting-backlog entry is the policy's scenario, whose
	// internal id stays "established"; every other suite scenario has its
	// own.
	if !slices.Equal(names, []string{"bench-financial-pressure", "established", "bench-demand-growth", "bench-staffing-shortage", "bench-stable"}) {
		t.Errorf("internal ids %v", names)
	}
}

func TestSuiteDimensions(t *testing.T) {
	s := loadSuite(t)
	known := map[string]bool{}
	for _, m := range Metrics {
		known[m.Name] = true
	}
	var seen []string
	for _, d := range []string{"operational", "financial", "staffingCapacity", "serviceReliability", "sustainability", "actions"} {
		if len(s.Dimensions[d]) == 0 {
			t.Errorf("dimension %s is empty", d)
		}
		for _, m := range s.Dimensions[d] {
			if !known[m] || slices.Contains(seen, m) {
				t.Errorf("%s: metric %q unknown or listed twice", d, m)
			}
			seen = append(seen, m)
		}
	}
	if len(s.Dimensions) != 6 || len(s.Missing) == 0 {
		t.Errorf("dimensions %v, missing %v", s.Dimensions, s.Missing)
	}
}

// The V0.7a development suite (docs/benchmark/suite/dreamteam-dev-2.json):
// version 1's entries, each at version 2 on its -ot variant, with the
// recorded fingerprint, T90 not applicable (the evaluation policy in force
// names version 1's sustainable scenario only), and the scenario equal to
// version 1's apart from its identity and the radiologist overtime policy.
func TestSuiteV2Scenarios(t *testing.T) {
	v1 := loadSuite(t)
	raw, err := os.ReadFile("../../docs/benchmark/suite/dreamteam-dev-2.json")
	if err != nil {
		t.Fatal(err)
	}
	var v2 suiteFile
	if err := json.Unmarshal(raw, &v2); err != nil {
		t.Fatal(err)
	}
	if v2.Schema != v1.Schema || v2.ID != v1.ID || v2.Version != 2 || v2.Split != "development" || v2.HorizonDays != 90 ||
		len(v2.Scenarios) != len(v1.Scenarios) || !slices.Equal(v2.Missing, v1.Missing) {
		t.Fatalf("suite header: %+v", v2)
	}
	for k, e := range v2.Scenarios {
		o := v1.Scenarios[k]
		if e.ID != o.ID || e.Version != 2 || e.Category != o.Category || e.Scenario != o.Scenario+"-ot" ||
			e.Source != "scenarios/"+e.Scenario+".json" || e.T90 != "not_applicable" {
			t.Fatalf("%s: %+v", e.ID, e)
		}
		f, err := scenario.ResolveFile(e.Scenario)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := scenario.Compile(f); err != nil {
			t.Fatalf("%s: %v", e.ID, err)
		}
		if fp, _ := scenario.Fingerprint(f); fp != e.Fingerprint {
			t.Errorf("%s: fingerprint %s, recorded %s", e.ID, fp, e.Fingerprint)
		}
		b, err := scenario.ResolveFile(o.Scenario)
		if err != nil {
			t.Fatal(err)
		}
		if f.Policies.RadiologistOvertime != "salary_rate" {
			t.Fatalf("%s: policy %q", e.ID, f.Policies.RadiologistOvertime)
		}
		f.Policies.RadiologistOvertime = ""
		if a, b := resolved(t, f, "x", "x"), resolved(t, b, "x", "x"); !reflect.DeepEqual(a, b) {
			t.Errorf("%s: version 2 differs from version 1 beyond the policy", e.ID)
		}
	}
	for dim, ms := range v1.Dimensions {
		if !slices.Equal(v2.Dimensions[dim], ms) {
			t.Errorf("dimension %s differs", dim)
		}
	}
}
