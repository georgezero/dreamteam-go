package bench

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/agent"
	"github.com/georgezero/dreamteam-go/internal/scenario"
)

// testSuite writes a suite like dreamteam-dev 1 whose entries are the
// small mini and established scenarios, and a suite run of it, and
// returns the run configuration's path.
func testSuite(t *testing.T, policies []string, seeds []int64, days int) string {
	t.Helper()
	raw, err := os.ReadFile(suitePath)
	if err != nil {
		t.Fatal(err)
	}
	var s Suite
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatal(err)
	}
	s.ID, s.Split = "test-suite", "development"
	s.Scenarios = nil
	for _, id := range []string{"mini", "established"} {
		f, err := scenario.ResolveFile(id)
		if err != nil {
			t.Fatal(err)
		}
		fp, _ := scenario.Fingerprint(f)
		s.Scenarios = append(s.Scenarios, SuiteEntry{ID: "t-" + id, Version: 1, Category: "test", Scenario: id, Fingerprint: fp,
			Source: "test", Challenge: "test", T90: T90NotApplicable})
	}
	dir := t.TempDir()
	sraw, _ := json.MarshalIndent(s, "", " ")
	if err := os.WriteFile(filepath.Join(dir, "suite-file.json"), sraw, 0o644); err != nil {
		t.Fatal(err)
	}
	r := SuiteRunConfig{Schema: SuiteRunSchema, ID: "test-run", Description: "test", Suite: "suite-file.json", Seeds: seeds,
		SeedPlan: "test seeds", Days: days, Policies: policies, Baseline: policies[0], Output: "out"}
	rraw, _ := json.MarshalIndent(r, "", " ")
	p := filepath.Join(dir, "run.json")
	if err := os.WriteFile(p, rraw, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// A suite run is the expanded configuration's ordinary matrix: every run
// file and both harness reports are byte for byte the standalone run's.
// It verifies, reproduces, and its contract report rebuilds.
func TestSuiteRunIsTheMatrix(t *testing.T) {
	sr, err := LoadSuiteRun(testSuite(t, []string{"noop", "greedy"}, []int64{5, 6}, 3))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(sr.Config.Scenarios, ",") != "mini,established" || sr.Config.Days != 3 || sr.Config.Cadence != sr.Suite.Cadence {
		t.Fatalf("expansion: %+v", sr.Config)
	}
	dir := filepath.Join(t.TempDir(), "suite")
	m, err := RunSuite(sr, dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Suite) != 4 {
		t.Fatalf("suite files %+v", m.Suite)
	}
	// The standalone run of the expanded configuration.
	c, raw, err := LoadConfig(filepath.Join(dir, FileConfig))
	if err != nil {
		t.Fatal(err)
	}
	alone := filepath.Join(t.TempDir(), "alone")
	n, err := Matrix(c, raw, alone, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(n.Suite) != 0 {
		t.Fatal("a standalone run lists suite files")
	}
	for _, r := range m.Runs {
		for _, f := range r.Files {
			a, _ := os.ReadFile(filepath.Join(dir, f.Path))
			b, err := os.ReadFile(filepath.Join(alone, f.Path))
			if err != nil || !bytes.Equal(a, b) {
				t.Errorf("%s differs from the standalone run's (%v)", f.Path, err)
			}
		}
	}
	for _, f := range []string{FileConfig, FileReportJSON, FileReportText} {
		a, _ := os.ReadFile(filepath.Join(dir, f))
		b, _ := os.ReadFile(filepath.Join(alone, f))
		if !bytes.Equal(a, b) {
			t.Errorf("%s differs from the standalone run's", f)
		}
	}
	m.Suite = nil
	x, _ := json.Marshal(m)
	y, _ := json.Marshal(n)
	if !bytes.Equal(x, y) {
		t.Error("the manifests differ beyond the suite files")
	}

	checks, err := Verify(dir, VerifyOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, ck := range checks {
		if !ck.OK {
			t.Errorf("verify: %s: %s", ck.What, ck.Note)
		}
	}
	checks, err = Reproduce(dir, filepath.Join(t.TempDir(), "again"), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, ck := range checks {
		if !ck.OK {
			t.Errorf("reproduce: %s: %s", ck.What, ck.Note)
		}
	}

	var k Contract
	readJSON(filepath.Join(dir, FileContractJSON), &k)
	if k.Status != "complete" || k.Expected != 8 || k.Completed != 8 || len(k.Runs) != 8 || len(k.Pairs) != 4 || len(k.Entries) != 2 {
		t.Fatalf("contract: %s %d/%d runs %d pairs %d", k.Status, k.Completed, k.Expected, len(k.Runs), len(k.Pairs))
	}
	for _, e := range k.Entries {
		for _, st := range e.Starts {
			if !st.Identical || st.Runs != 2 {
				t.Errorf("%s seed %d: starting state %+v", e.ID, st.Seed, st)
			}
		}
	}
	for _, r := range k.Runs {
		if len(r.Dimensions) != 6 || r.Fingerprint == "" || r.Start == "" || !r.Replay || r.T90.Applicability != T90NotApplicable {
			t.Errorf("%s: %+v", r.ID, r)
		}
		for _, d := range r.Dimensions {
			for _, v := range d.Metrics {
				if v.Value == nil && v.Absent == "" {
					t.Errorf("%s %s: absent without a reason", r.ID, v.Name)
				}
			}
		}
	}
	text, _ := os.ReadFile(filepath.Join(dir, FileContractText))
	for _, want := range []string{"complete: 8 of 8 runs", "Operational performance", "Long-term sustainability", "Paired with noop", "10. Missing"} {
		if !strings.Contains(string(text), want) {
			t.Errorf("contract.txt lacks %q", want)
		}
	}

	// Tampering is caught: the contract report, and a suite fingerprint.
	os.WriteFile(filepath.Join(dir, FileContractText), append(text, '!'), 0o644)
	checks, _ = Verify(dir, VerifyOptions{})
	if failed(checks) == 0 {
		t.Error("an edited contract.txt passed")
	}
	os.WriteFile(filepath.Join(dir, FileContractText), text, 0o644)
	sraw, _ := os.ReadFile(filepath.Join(dir, FileSuite))
	bad := bytes.Replace(sraw, []byte(sr.Suite.Scenarios[0].Fingerprint), []byte(strings.Repeat("0", 64)), 1)
	if _, err := DecodeSuiteRun(sr.RunRaw, bad); err == nil || !strings.Contains(err.Error(), "fingerprint") {
		t.Errorf("a wrong fingerprint decoded: %v", err)
	}
}

func failed(cs []Check) int {
	n := 0
	for _, c := range cs {
		if !c.OK {
			n++
		}
	}
	return n
}

// A failed run is recorded, and the suite is reported as partial, never as
// complete.
func TestSuiteRunPartial(t *testing.T) {
	withPolicies(t, spec("broken", func(c *agent.Client, point int) ([][]json.RawMessage, error) {
		if point == 1 {
			return nil, errors.New("broken on purpose")
		}
		return nil, nil
	}))
	sr, err := LoadSuiteRun(testSuite(t, []string{"noop", "broken"}, []int64{5}, 3))
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(t.TempDir(), "suite")
	if _, err := RunSuite(sr, dir, nil); err != nil {
		t.Fatal(err)
	}
	var k Contract
	readJSON(filepath.Join(dir, FileContractJSON), &k)
	if k.Status != "partial" || k.Completed != 2 || k.Expected != 4 || len(k.Failed) != 2 || len(k.Pairs) != 0 {
		t.Fatalf("contract: %s %d/%d failed %v pairs %d", k.Status, k.Completed, k.Expected, k.Failed, len(k.Pairs))
	}
	text, _ := os.ReadFile(filepath.Join(dir, FileContractText))
	if !strings.Contains(string(text), "PARTIAL SUITE: 2 of 4 runs completed") {
		t.Errorf("contract.txt:\n%s", text)
	}
	checks, err := Verify(dir, VerifyOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, ck := range checks {
		if !ck.OK && !strings.Contains(ck.What, "saved session") && !strings.Contains(ck.What, "session replays") {
			t.Errorf("verify: %s: %s", ck.What, ck.Note)
		}
	}
}

// Suite-run configurations are refused before anything runs: an unknown
// entry, a repeated one, an unknown field, a reference seed.
func TestSuiteRunInvalid(t *testing.T) {
	p := testSuite(t, []string{"noop"}, []int64{5}, 3)
	raw, _ := os.ReadFile(p)
	suite, _ := os.ReadFile(filepath.Join(filepath.Dir(p), "suite-file.json"))
	for name, edit := range map[string]func(map[string]any){
		"unknown entry":  func(m map[string]any) { m["entries"] = []string{"nope"} },
		"repeated entry": func(m map[string]any) { m["entries"] = []string{"t-mini", "t-mini"} },
		"unknown field":  func(m map[string]any) { m["extra"] = 1 },
		"reference seed": func(m map[string]any) { m["seeds"] = []int64{1005} },
		"wrong schema":   func(m map[string]any) { m["schema"] = "x" },
	} {
		var m map[string]any
		json.Unmarshal(raw, &m)
		edit(m)
		b, _ := json.Marshal(m)
		if _, err := DecodeSuiteRun(b, suite); !errors.Is(err, ErrConfig) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// Every committed suite-run configuration loads: its suite checks against
// the bundled scenarios, and its expansion passes the configuration's
// checks, the sealed holdout guard included.
func TestCommittedSuiteRuns(t *testing.T) {
	paths, _ := filepath.Glob("../../docs/benchmark/*/*.json")
	n := 0
	for _, p := range paths {
		raw, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		var head struct{ Schema string }
		if json.Unmarshal(raw, &head) != nil || head.Schema != SuiteRunSchema {
			continue
		}
		n++
		if _, err := LoadSuiteRun(p); err != nil {
			t.Error(err)
		}
	}
	if n < 2 {
		t.Fatalf("%d suite-run configurations", n)
	}
}
