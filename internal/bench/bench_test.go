package bench

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/agent"
	"github.com/georgezero/dreamteam-go/internal/session"
)

// testConfig is a small matrix on the mini scenario (no T90: it does not
// apply there).
func testConfig(policies ...string) *Config {
	return &Config{Schema: ConfigSchema, ID: "test", Description: "test", Scenarios: []string{"mini"},
		Seeds: []int64{5, 6}, SeedPlan: "test seeds", Days: 3, Cadence: Cadence{EveryDays: 1, At: "07:00"},
		Policies: policies, Baseline: policies[0], Output: "out", Evaluation: Evaluation{T90: true}}
}

func runMatrix(t *testing.T, c *Config) string {
	t.Helper()
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(c)
	dir := filepath.Join(t.TempDir(), "results")
	if _, err := Matrix(c, raw, dir, nil); err != nil {
		t.Fatal(err)
	}
	return dir
}

// testPolicy is a policy defined by a function, for tests only.
type testPolicy struct {
	name   string
	decide func(c *agent.Client, point int) ([][]json.RawMessage, error)
	point  int
}

func (p *testPolicy) Name() string { return p.name }
func (p *testPolicy) Decide(c *agent.Client) ([][]json.RawMessage, error) {
	defer func() { p.point++ }()
	return p.decide(c, p.point)
}

// withPolicies registers test policies for one test.
func withPolicies(t *testing.T, specs ...PolicySpec) {
	saved := policies
	policies = append(slices.Clone(policies), specs...)
	t.Cleanup(func() { policies = saved })
}

func spec(id string, decide func(c *agent.Client, point int) ([][]json.RawMessage, error)) PolicySpec {
	return PolicySpec{ID: id, Version: "test", Description: "test policy",
		New: func() agent.Policy { return &testPolicy{name: id, decide: decide} }}
}

// active, at its first decision point: tries the harness's ops (denied),
// submits an applied batch, a refused one and one the protocol cannot
// read; later, an applied walk-in policy change.
func active(c *agent.Client, point int) ([][]json.RawMessage, error) {
	if point == 0 {
		for _, op := range []string{"apply", "advance", "save", "t90", "no_such_op"} {
			var e *agent.Error
			if err := c.Call(op, map[string]any{}, nil); !errors.As(err, &e) || e.Code != ErrNotPermitted {
				return nil, errors.New("the gate let " + op + " through")
			}
		}
		return [][]json.RawMessage{
			{json.RawMessage(`{"type":"set_worklist_policy","policy":"fifo"}`)},
			{json.RawMessage(`{"type":"set_worklist_policy","policy":"no_such_policy"}`)},
			{json.RawMessage(`{"type":"set_worklist_policy","policy":"fifo","bogus":1}`)},
		}, nil
	}
	if point == 1 {
		return [][]json.RawMessage{{json.RawMessage(`{"type":"set_walkin_policy","policy":"accept_if_gap"}`)}}, nil
	}
	return nil, nil
}

func TestConfigValidation(t *testing.T) {
	good := testConfig("noop", "greedy")
	if err := good.Validate(); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		edit func(c *Config)
		want string
	}{
		{"schema", func(c *Config) { c.Schema = "x" }, "schema"},
		{"id", func(c *Config) { c.ID = "Has Space" }, "id"},
		{"no scenario", func(c *Config) { c.Scenarios = nil }, "scenarios"},
		{"unknown scenario", func(c *Config) { c.Scenarios = []string{"nowhere"} }, "not a bundled scenario"},
		{"no seeds", func(c *Config) { c.Seeds = nil }, "explicit list"},
		{"seed plan", func(c *Config) { c.SeedPlan = "" }, "seedPlan"},
		{"duplicate seed", func(c *Config) { c.Seeds = []int64{5, 5} }, "listed twice"},
		{"reference seed", func(c *Config) { c.Seeds = []int64{1007} }, "reference evaluation seed"},
		{"negative seed", func(c *Config) { c.Seeds = []int64{-3} }, "positive"},
		{"days", func(c *Config) { c.Days = 0 }, "days"},
		{"cadence days", func(c *Config) { c.Cadence.EveryDays = 4 }, "everyDays"},
		{"cadence time", func(c *Config) { c.Cadence.At = "7:00" }, "cadence.at"},
		{"no policies", func(c *Config) { c.Policies = nil }, "policies"},
		{"unknown policy", func(c *Config) { c.Policies = []string{"noop", "oracle"} }, "not a policy"},
		{"duplicate policy", func(c *Config) { c.Policies = []string{"noop", "noop"} }, "listed twice"},
		{"baseline", func(c *Config) { c.Baseline = "rules" }, "baseline"},
		{"absolute output", func(c *Config) { c.Output = "/tmp/x" }, "relative"},
	}
	for _, tc := range cases {
		c := *testConfig("noop", "greedy")
		tc.edit(&c)
		err := c.Validate()
		if !errors.Is(err, ErrConfig) || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: %v", tc.name, err)
		}
	}
	for _, raw := range []string{`{"schema":"dreamteam.benchmark/v1","unknown":1}`, `{}{}`, `[`} {
		if _, err := DecodeConfig([]byte(raw)); !errors.Is(err, ErrConfig) {
			t.Errorf("%s: %v", raw, err)
		}
	}
}

// The gate passes the read and check operations and answers the others
// itself: the session is untouched.
func TestGate(t *testing.T) {
	s, err := session.Open("mini", 5)
	if err != nil {
		t.Fatal(err)
	}
	h := agent.New(s)
	h.Dir = t.TempDir() // a save that got through would land here
	g := &gate{h: h}
	before := s.Digests()
	for _, line := range []string{
		`{"id":1,"op":"apply","args":{"actions":[{"type":"set_worklist_policy","policy":"fifo"}]}}`,
		`{"id":"x","op":"advance","args":{"minutes":60}}`,
		`{"op":"save","args":{"path":"leak.json"}}`,
		`{"id":2,"op":"t90"}`,
		`{"id":3,"OP":"advance","args":{"minutes":60}}`,
	} {
		var r agent.Response
		var e agent.Error
		r.Error = &e
		if err := json.Unmarshal(g.Handle([]byte(line)), &r); err != nil || r.OK || e.Code != ErrNotPermitted {
			t.Errorf("%s: %+v %+v", line, r, e)
		}
	}
	if g.denied != 5 || s.Digests() != before {
		t.Fatalf("denied %d; digests changed: %v", g.denied, s.Digests() != before)
	}
	for _, op := range PolicyOps {
		var r agent.Response
		args := ""
		if op == "validate" || op == "preview" {
			args = `,"args":{"actions":[{"type":"set_reading_allocation","op":"remove","employee":"x","effective":"2027-01-01"}]}`
		}
		json.Unmarshal(g.Handle([]byte(`{"id":1,"op":"`+op+`"`+args+`}`)), &r)
		if r.Error != nil && r.Error.Code == ErrNotPermitted {
			t.Errorf("%s denied", op)
		}
	}
	// What the gate cannot read goes to the handler, which refuses it.
	var r agent.Response
	json.Unmarshal(g.Handle([]byte(`{"id":1,"op":`)), &r)
	if r.Error == nil || r.Error.Code != agent.ErrInvalidJSON || s.Digests() != before {
		t.Fatalf("malformed: %+v", r.Error)
	}
}

// Acceptance 1, 2, 4, 6, 7: identical configurations give byte-identical
// results (timing.json aside); every run completes, starts from the same
// state on its seed, replays, and its session holds exactly its applied
// decisions; the results verify.
func TestMatrixDeterministicAndVerified(t *testing.T) {
	withPolicies(t, spec("test-active", active))
	c := testConfig("noop", "test-active", "greedy")
	a, b := runMatrix(t, c), runMatrix(t, c)
	files := 0
	filepath.WalkDir(a, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() == FileTiming {
			return err
		}
		rel, _ := filepath.Rel(a, p)
		x, _ := os.ReadFile(p)
		y, err := os.ReadFile(filepath.Join(b, rel))
		if err != nil || !bytes.Equal(x, y) {
			t.Errorf("%s differs between identical runs", rel)
		}
		files++
		return nil
	})
	if files < 6*5 {
		t.Fatalf("%d files", files)
	}
	_, _, recs, err := LoadResults(a)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range recs {
		if r.Status != StatusCompleted || !r.Replay.Verified || !r.ActionsMatch || r.T90.Status != T90NotApplicable {
			t.Errorf("%s: %s %+v match %v t90 %+v", r.ID, r.Status, r.Replay, r.ActionsMatch, r.T90)
		}
	}
	act := recs[1]
	if act.Policy.ID != "test-active" || act.Counts != (Counts{DecisionPoints: 3, Batches: 4, Actions: 4, Applied: 2, Refused: 1, ProtocolErrors: 1, PolicyRequests: 5, Denied: 5}) {
		t.Fatalf("%s counts %+v", act.ID, act.Counts)
	}
	rep := BuildReport(c, recs)
	for _, in := range rep.Initial {
		if !in.Identical || in.Runs != 3 {
			t.Errorf("initial conditions %+v", in)
		}
	}
	for _, p := range rep.Pairs {
		want := map[string]string{"test-active": "different", "greedy": "same"}[p.Policy]
		if p.Decisions != want || p.Flag != "" || (want == "same") != (p.Outcome == "identical") {
			t.Errorf("pair %+v", p)
		}
	}
	checks, err := Verify(a, VerifyOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, ck := range checks {
		if !ck.OK {
			t.Errorf("verify: %s: %s", ck.What, ck.Note)
		}
	}
	// A changed file fails its hash; a changed decision fails the match.
	p := filepath.Join(a, "runs", act.ID, FileDecisions)
	raw, _ := os.ReadFile(p)
	os.WriteFile(p, bytes.Replace(raw, []byte(`accept_if_gap`), []byte(`accept_always`), 1), 0o644)
	checks, _ = Verify(a, VerifyOptions{})
	failed := map[string]bool{}
	for _, ck := range checks {
		if !ck.OK {
			failed[ck.What] = true
		}
	}
	if !failed["hash runs/"+act.ID+"/"+FileDecisions] || !failed[act.ID+": session batches are the applied decisions"] || len(failed) != 2 {
		t.Fatalf("tampered decisions: %v", failed)
	}
}

// Acceptance 5: a run's outcome does not depend on the other policies in
// the matrix or their order: each run has its own session and policy.
func TestRunsAreIsolated(t *testing.T) {
	withPolicies(t, spec("test-active", active))
	alone := runMatrix(t, testConfig("noop"))
	mixed := runMatrix(t, testConfig("test-active", "noop", "greedy"))
	reversed := runMatrix(t, testConfig("greedy", "noop", "test-active"))
	for _, f := range runFiles {
		x, err := os.ReadFile(filepath.Join(alone, "runs", "mini-s6-noop", f))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		for _, d := range []string{mixed, reversed} {
			y, _ := os.ReadFile(filepath.Join(d, "runs", "mini-s6-noop", f))
			if !bytes.Equal(x, y) {
				t.Errorf("%s of the noop run differs with other policies in the matrix", f)
			}
		}
	}
	a, _ := os.ReadFile(filepath.Join(mixed, "runs", "mini-s5-test-active", FileRun))
	b, _ := os.ReadFile(filepath.Join(reversed, "runs", "mini-s5-test-active", FileRun))
	if !bytes.Equal(a, b) {
		t.Error("the active run differs with the policy order")
	}
}

// Acceptance 3: a policy's requests are only the read and check
// operations; everything else it tried was denied by the gate and never
// reached the session.
func TestPolicySeesOnlyThePublicProtocol(t *testing.T) {
	withPolicies(t, spec("test-active", active))
	dir := runMatrix(t, testConfig("test-active", "rules"))
	for _, id := range []string{"mini-s5-test-active", "mini-s5-rules"} {
		f, err := os.Open(filepath.Join(dir, "runs", id, FileTranscript))
		if err != nil {
			t.Fatal(err)
		}
		sc := bufio.NewScanner(f)
		sc.Buffer(nil, 64<<20)
		var lines [][]byte
		for sc.Scan() {
			lines = append(lines, slices.Clone(sc.Bytes()))
		}
		f.Close()
		policyOps := map[string]int{}
		for i := 0; i+1 < len(lines); i += 2 {
			var req agent.Request
			var resp agent.Response
			resp.Error = &agent.Error{}
			json.Unmarshal(lines[i], &req)
			json.Unmarshal(lines[i+1], &resp)
			var rid string
			json.Unmarshal(req.ID, &rid)
			if !strings.HasPrefix(rid, "p") {
				if !strings.HasPrefix(rid, "h") {
					t.Fatalf("%s: request %s from neither the policy nor the harness", id, lines[i])
				}
				continue
			}
			policyOps[req.Op]++
			if !slices.Contains(PolicyOps, req.Op) && (resp.OK || resp.Error.Code != ErrNotPermitted) {
				t.Errorf("%s: the policy's %s was answered %s", id, req.Op, lines[i+1])
			}
		}
		if id == "mini-s5-test-active" && policyOps["apply"] != 1 {
			t.Errorf("%s: policy ops %v", id, policyOps)
		}
		if id == "mini-s5-rules" && (policyOps["observe"] != 3 || len(policyOps) != 1) {
			t.Errorf("%s: policy ops %v", id, policyOps)
		}
	}
}

// Acceptance 10: a policy that fails is classified, its run's files are
// written as far as it got (and verify), and the other runs go on.
func TestPolicyFailures(t *testing.T) {
	withPolicies(t,
		spec("test-error", func(c *agent.Client, point int) ([][]json.RawMessage, error) {
			if point == 1 {
				return nil, errors.New("the model is unavailable")
			}
			return [][]json.RawMessage{{json.RawMessage(`{"type":"set_worklist_policy","policy":"fifo"}`)}}, nil
		}),
		spec("test-panic", func(c *agent.Client, point int) ([][]json.RawMessage, error) {
			var m map[string]int
			m["x"]++ // a nil map: the policy panics
			return nil, nil
		}))
	dir := runMatrix(t, testConfig("noop", "test-error", "test-panic"))
	_, _, recs, err := LoadResults(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range recs {
		switch r.Policy.ID {
		case "noop":
			if r.Status != StatusCompleted {
				t.Errorf("%s: %s", r.ID, r.Status)
			}
		case "test-error":
			if r.Status != StatusPolicy || r.Failure.Point != 1 || r.Failure.Time != "2026-10-14T07:00:00" ||
				!strings.Contains(r.Failure.Message, "the model is unavailable") || !r.Replay.Verified || !r.ActionsMatch || r.T90.Status != T90Skipped {
				t.Errorf("%s: %+v %+v", r.ID, r.Failure, r.Replay)
			}
		case "test-panic":
			if r.Status != StatusPolicy || r.Failure.Point != 0 || !strings.Contains(r.Failure.Message, "panicked") {
				t.Errorf("%s: %+v", r.ID, r.Failure)
			}
		}
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
	txt, _ := os.ReadFile(filepath.Join(dir, FileReportText))
	if !strings.Contains(string(txt), "policy_error  4") {
		t.Errorf("report:\n%s", txt)
	}
}

// Acceptance 8: the metrics are the replayed session's result values.
func TestMetricsAreTheResults(t *testing.T) {
	dir := runMatrix(t, testConfig("noop"))
	rd := filepath.Join(dir, "runs", "mini-s5-noop")
	res, _, err := Measure(filepath.Join(rd, FileSession))
	if err != nil {
		t.Fatal(err)
	}
	stored, _ := os.ReadFile(filepath.Join(rd, FileResult))
	if !bytes.Equal(res, stored) {
		t.Fatal("result.json is not the replayed session's result")
	}
	var rec RunRecord
	readJSON(filepath.Join(rd, FileRun), &rec)
	var root map[string]any
	json.Unmarshal(stored, &root)
	for i, m := range Metrics {
		v, ok := lookup(root, m.Path)
		raw, _ := json.Marshal(v)
		if !ok {
			raw = []byte("null")
		}
		var a, b any
		json.Unmarshal(raw, &a)
		json.Unmarshal(rec.Metrics[i].Value, &b)
		x, _ := json.Marshal(a)
		y, _ := json.Marshal(b)
		if rec.Metrics[i].Name != m.Name || !bytes.Equal(x, y) {
			t.Errorf("%s: run.json %s, result %s", m.Name, rec.Metrics[i].Value, raw)
		}
	}
}

// The rule-based baseline acts only after validate and preview, and only
// on the reading allocation; on a quiet backlog it does nothing.
func TestRulesPolicy(t *testing.T) {
	c := testConfig("rules")
	c.Scenarios, c.Seeds, c.Days, c.Evaluation.T90 = []string{"sustainable"}, []int64{77}, 8, false
	dir := runMatrix(t, c)
	var decisions []Decision
	if err := readJSON(filepath.Join(dir, "runs", "sustainable-s77-rules", FileDecisions), &decisions); err != nil {
		t.Fatal(err)
	}
	var acted []Submitted
	for _, d := range decisions {
		acted = append(acted, d.Batches...)
	}
	if len(acted) != 1 || !acted[0].Applied || !strings.Contains(string(acted[0].Actions[0]), `"set_reading_allocation"`) {
		t.Fatalf("decisions %+v", acted)
	}
	raw, _ := os.ReadFile(filepath.Join(dir, "runs", "sustainable-s77-rules", FileTranscript))
	v, p, a := bytes.Index(raw, []byte(`"op":"validate"`)), bytes.Index(raw, []byte(`"op":"preview"`)), bytes.Index(raw, []byte(`"op":"apply"`))
	if v < 0 || p < v || a < p {
		t.Fatalf("validate %d, preview %d, apply %d", v, p, a)
	}
}

// Every benchmark configuration committed under docs/benchmark validates,
// sealed holdout guard included, without running anything.
func TestCommittedConfigs(t *testing.T) {
	paths, _ := filepath.Glob("../../docs/benchmark/*/*.json")
	n := 0
	for _, p := range paths {
		raw, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		var head struct{ Schema string }
		if json.Unmarshal(raw, &head) != nil || head.Schema != ConfigSchema {
			continue
		}
		n++
		if _, _, err := LoadConfig(p); err != nil {
			t.Error(err)
		}
	}
	if n < 10 {
		t.Fatalf("%d configurations", n)
	}
}
